package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/BrendenWalker/verity/internal/metadata"
)

type attachSignatureRequest struct {
	Digest    string          `json:"digest"`
	BundleRef *string         `json:"bundle_ref,omitempty"`
	Bundle    json.RawMessage `json:"bundle,omitempty"`
	Issuer    *string         `json:"issuer,omitempty"`
	Subject   *string         `json:"subject,omitempty"`
}

type signatureResponse struct {
	ID        int64           `json:"id"`
	Digest    string          `json:"digest"`
	BundleRef *string         `json:"bundle_ref,omitempty"`
	Bundle    json.RawMessage `json:"bundle,omitempty"`
	Issuer    *string         `json:"issuer,omitempty"`
	Subject   *string         `json:"subject,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type listSignaturesResponse struct {
	Namespace  string              `json:"namespace"`
	Artifact   string              `json:"artifact"`
	Digest     string              `json:"digest"`
	Signatures []signatureResponse `json:"signatures"`
}

func signatureFromRow(digest string, sig metadata.Signature) signatureResponse {
	return signatureResponse{
		ID:        sig.ID,
		Digest:    digest,
		BundleRef: sig.BundleRef,
		Bundle:    sig.BundleJSON,
		Issuer:    sig.Issuer,
		Subject:   sig.Subject,
		CreatedAt: sig.CreatedAt,
	}
}

func (h *Handler) postAttachSignature(w http.ResponseWriter, r *http.Request, ns, artifact string) {
	if ns == "" || artifact == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace and artifact are required", nil)
		return
	}

	var req attachSignatureRequest
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
		return
	}
	req.Digest = strings.TrimSpace(req.Digest)
	if req.Digest == "" || !strings.HasPrefix(req.Digest, "sha256:") {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "digest must be a sha256:… reference", nil)
		return
	}
	if req.BundleRef == nil && len(req.Bundle) == 0 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "bundle_ref or bundle is required", nil)
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
	if d.ArtifactID != art.ID {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "digest not found for artifact", nil)
		return
	}

	sig, err := h.Store.AttachSignature(ctx, d.ID, req.BundleRef, req.Bundle, req.Issuer, req.Subject)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(signatureFromRow(d.Digest, *sig))
}

func (h *Handler) getListSignatures(w http.ResponseWriter, r *http.Request, ns, artifact string) {
	if ns == "" || artifact == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace and artifact are required", nil)
		return
	}
	digestRef := strings.TrimSpace(r.URL.Query().Get("digest"))
	if digestRef == "" || !strings.HasPrefix(digestRef, "sha256:") {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "digest query parameter must be a sha256:… reference", nil)
		return
	}

	ctx := r.Context()
	namespace, err := h.Store.GetNamespaceByName(ctx, ns)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	if err := authorizeRead(ctx, ns, namespace.Config); err != nil {
		writeAuthError(w, err)
		return
	}
	art, err := h.Store.GetArtifact(ctx, namespace.ID, artifact)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	d, err := h.Store.GetDigestByString(ctx, digestRef)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	if d.ArtifactID != art.ID {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "digest not found for artifact", nil)
		return
	}

	sigs, err := h.Store.ListSignatures(ctx, d.ID)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	resp := listSignaturesResponse{
		Namespace:  namespace.Name,
		Artifact:   art.Name,
		Digest:     d.Digest,
		Signatures: make([]signatureResponse, 0, len(sigs)),
	}
	for _, sig := range sigs {
		resp.Signatures = append(resp.Signatures, signatureFromRow(d.Digest, sig))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}
