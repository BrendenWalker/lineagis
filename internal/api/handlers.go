package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/BrendenWalker/verity/internal/metadata"
	"github.com/BrendenWalker/verity/internal/semver"
	"github.com/BrendenWalker/verity/internal/signing"
)

// Handler serves Verity control-plane HTTP routes (OQ-API-001 layout).
type Handler struct {
	Store        *metadata.Store
	Manifests    ManifestSource
	Policy       PushPolicy
	VerifyPolicy VerifyPolicy
	Auth         func(http.Handler) http.Handler
}

type registerDigestRequest struct {
	Digest    string          `json:"digest"`
	MediaType *string         `json:"media_type,omitempty"`
	SizeBytes *int64          `json:"size_bytes,omitempty"`
	Bundle    json.RawMessage `json:"bundle,omitempty"`
	Issuer    *string         `json:"issuer,omitempty"`
	Subject   *string         `json:"subject,omitempty"`
}

type registerDigestResponse struct {
	Digest    string `json:"digest"`
	Artifact  string `json:"artifact"`
	Namespace string `json:"namespace"`
}

type setTagRequest struct {
	Digest string `json:"digest"`
}

type setTagResponse struct {
	Tag       string `json:"tag"`
	Digest    string `json:"digest"`
	Artifact  string `json:"artifact"`
	Namespace string `json:"namespace"`
}

type artifactResponse struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func (h *Handler) putArtifact(w http.ResponseWriter, r *http.Request, ns, artifact string) {
	if ns == "" || artifact == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace and artifact are required", nil)
		return
	}

	ctx := r.Context()
	namespace, err := h.Store.CreateNamespace(ctx, ns, nil)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	if err := authorizeNamespace(ctx, ns, namespace.Config); err != nil {
		writeAuthError(w, err)
		return
	}
	art, err := h.Store.RegisterArtifact(ctx, namespace.ID, artifact)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(artifactResponse{Name: art.Name, Namespace: namespace.Name})
}

func (h *Handler) postRegisterDigest(w http.ResponseWriter, r *http.Request, ns, artifact string) {
	if ns == "" || artifact == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace and artifact are required", nil)
		return
	}

	var req registerDigestRequest
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
		return
	}
	req.Digest = strings.TrimSpace(req.Digest)
	if req.Digest == "" || !strings.HasPrefix(req.Digest, "sha256:") {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "digest must be a sha256:… reference", nil)
		return
	}

	ctx := r.Context()
	namespace, err := h.Store.GetNamespaceByName(ctx, ns)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	if err := authorizeNamespace(ctx, ns, namespace.Config); err != nil {
		writeAuthError(w, err)
		return
	}
	art, err := h.Store.GetArtifact(ctx, namespace.ID, artifact)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	requireSig := false
	if policy, polErr := h.Store.GetActivePolicy(ctx, namespace.ID); polErr == nil {
		requireSig = policyRequiresSignatures(policy.Document)
	} else if !errors.Is(polErr, metadata.ErrNotFound) {
		if mapStoreError(w, polErr) {
			return
		}
	}

	if requireSig && len(req.Bundle) == 0 {
		if existing, getErr := h.Store.GetDigestByString(ctx, req.Digest); getErr == nil {
			if existing.ArtifactID == art.ID {
				sigs, listErr := h.Store.ListSignatures(ctx, existing.ID)
				if listErr == nil && len(sigs) > 0 {
					writeRegisterDigest(w, existing.Digest, art.Name, namespace.Name)
					return
				}
			}
		}
		writePolicyFailed(w, PolicyFailure{
			Rule: "require-signatures",
			Hint: "digest has no signature; attach a Sigstore bundle in RegisterDigest when require-signatures is enabled",
		})
		return
	}

	var d *metadata.Digest
	if len(req.Bundle) > 0 {
		if h.Manifests == nil {
			WriteError(w, http.StatusInternalServerError, "INTERNAL", "manifest source not configured", nil)
			return
		}
		manifestJSON, err := h.Manifests.FetchManifest(ctx, ns, artifact, req.Digest)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "cannot verify signature without registry manifest: "+err.Error(), nil)
			return
		}
		if err := manifestDigestMatches(manifestJSON, req.Digest); err != nil {
			WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
			return
		}
		var policyDoc json.RawMessage
		if policy, polErr := h.Store.GetActivePolicy(ctx, namespace.ID); polErr == nil {
			policyDoc = policy.Document
		}
		opts := signing.KeylessVerifyOptions(policyDoc)
		if pub := signing.VerificationKeyPEM(req.Bundle); len(pub) > 0 {
			opts.PublicKeyPEM = pub
			opts.IgnoreTlog = true
			opts.IgnoreSCT = true
		}
		cfg := signing.LoadConfig()
		if err := signing.VerifyManifestBundle(ctx, cfg, manifestJSON, req.Bundle, opts); err != nil {
			fail := PolicyFailure{
				Rule: "require-signatures",
				Hint: "signature bundle does not verify for manifest digest: " + err.Error(),
			}
			if policyHasTrustedPublishers(policyDoc) && isTrustedPublisherIdentityFailure(err) {
				fail.Rule = "trusted-publishers"
			}
			writePolicyFailed(w, fail)
			return
		}
		reg, _, err := h.Store.RegisterDigestWithSignature(ctx, art.ID, req.Digest, req.MediaType, req.SizeBytes, req.Bundle, req.Issuer, req.Subject)
		if err != nil {
			if mapStoreError(w, err) {
				return
			}
		}
		d = reg
	} else {
		reg, err := h.Store.RegisterDigest(ctx, art.ID, req.Digest, req.MediaType, req.SizeBytes)
		if err != nil {
			if mapStoreError(w, err) {
				return
			}
		}
		d = reg
	}

	writeRegisterDigest(w, d.Digest, art.Name, namespace.Name)
}

func writeRegisterDigest(w http.ResponseWriter, digest, artifact, namespace string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(registerDigestResponse{
		Digest:    digest,
		Artifact:  artifact,
		Namespace: namespace,
	})
}

func (h *Handler) putSetTag(w http.ResponseWriter, r *http.Request, ns, artifact, tag string) {
	if ns == "" || artifact == "" || tag == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace, artifact, and tag are required", nil)
		return
	}
	if err := semver.ValidateTag(tag); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
		return
	}

	var req setTagRequest
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
		return
	}
	req.Digest = strings.TrimSpace(req.Digest)
	if req.Digest == "" || !strings.HasPrefix(req.Digest, "sha256:") {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "digest must be a sha256:… reference", nil)
		return
	}

	ctx := r.Context()
	namespace, err := h.Store.GetNamespaceByName(ctx, ns)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	if err := authorizeNamespace(ctx, ns, namespace.Config); err != nil {
		writeAuthError(w, err)
		return
	}
	art, err := h.Store.GetArtifact(ctx, namespace.ID, artifact)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	d, err := h.Store.GetDigestByString(ctx, req.Digest)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	if h.Policy == nil {
		h.Policy = AllowAllPolicy{}
	}
	if err := h.Policy.AllowSetTag(ctx, namespace.ID, art.ID, d.ID); err != nil {
		writePolicyFailed(w, err)
		return
	}

	actor := ActorFromContext(ctx)
	var actorPtr *string
	if actor != "" {
		actorPtr = &actor
	}

	tagRow, err := h.Store.SetTag(ctx, art.ID, tag, d.ID, actorPtr)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	resID := fmt.Sprintf("%d", tagRow.ID)
	h.recordAudit(ctx, namespace.ID, "tag.set", actorPtr, strPtr("tag"), &resID, map[string]any{
		"tag":    tagRow.Name,
		"digest": d.Digest,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(setTagResponse{
		Tag:       tagRow.Name,
		Digest:    d.Digest,
		Artifact:  art.Name,
		Namespace: namespace.Name,
	})
}

func decodeJSON(r *http.Request, v any) error {
	defer func() { _ = r.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return err
	}
	if len(body) == 0 {
		return nil
	}
	return json.Unmarshal(body, v)
}
