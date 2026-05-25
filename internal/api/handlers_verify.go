package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/BrendenWalker/verity/internal/metadata"
)

type verifyRequest struct {
	Digest string `json:"digest,omitempty"`
	Tag    string `json:"tag,omitempty"`
}

type verifyPolicyResult struct {
	Status  string         `json:"status"`
	Reasons []PolicyReason `json:"reasons,omitempty"`
}

type verifyResponse struct {
	Namespace  string             `json:"namespace"`
	Artifact   string             `json:"artifact"`
	Digest     string             `json:"digest"`
	Outcome    string             `json:"outcome"`
	Signatures trustSignatures    `json:"signatures"`
	Policy     verifyPolicyResult `json:"policy"`
}

func (h *Handler) verifyPolicy() VerifyPolicy {
	if h.VerifyPolicy != nil {
		return h.VerifyPolicy
	}
	return NewStoreVerifyPolicy(h.Store)
}

func (h *Handler) postVerify(w http.ResponseWriter, r *http.Request, ns, artifact string) {
	if ns == "" || artifact == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace and artifact are required", nil)
		return
	}

	var req verifyRequest
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
		return
	}
	req.Digest = strings.TrimSpace(req.Digest)
	req.Tag = strings.TrimSpace(req.Tag)
	if req.Digest == "" && req.Tag == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "digest or tag is required", nil)
		return
	}
	if req.Digest != "" && req.Tag != "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "provide digest or tag, not both", nil)
		return
	}
	if req.Digest != "" && !strings.HasPrefix(req.Digest, "sha256:") {
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

	d, err := h.resolveTrustDigest(ctx, art.ID, req.Tag, req.Digest)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	if d.ArtifactID != art.ID {
		WriteError(w, http.StatusNotFound, "NOT_FOUND", "digest not found for artifact", nil)
		return
	}

	resp, err := h.runVerify(ctx, namespace.ID, ns, artifact, d)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (h *Handler) runVerify(ctx context.Context, namespaceID int64, ns, artifact string, d *metadata.Digest) (*verifyResponse, error) {
	sigs, err := h.Store.ListSignatures(ctx, d.ID)
	if err != nil {
		return nil, err
	}
	sigStatus := "missing"
	if len(sigs) > 0 {
		sigStatus = "valid"
	}

	result, err := h.verifyPolicy().Evaluate(ctx, namespaceID, d.ID)
	if err != nil {
		return nil, err
	}

	if result.PolicyID != nil {
		reasonsJSON, err := json.Marshal(result.Reasons)
		if err != nil {
			return nil, err
		}
		if _, err := h.Store.RecordPolicyDecision(ctx, d.ID, *result.PolicyID, result.Outcome, reasonsJSON); err != nil {
			return nil, err
		}
	}

	outcome := "pass"
	if sigStatus != "valid" || result.Outcome == "fail" {
		outcome = "fail"
	} else if result.Outcome == "warn" {
		outcome = "warn"
	}

	return &verifyResponse{
		Namespace: ns,
		Artifact:  artifact,
		Digest:    d.Digest,
		Outcome:   outcome,
		Signatures: trustSignatures{
			Status: sigStatus,
		},
		Policy: verifyPolicyResult{
			Status:  result.Outcome,
			Reasons: result.Reasons,
		},
	}, nil
}
