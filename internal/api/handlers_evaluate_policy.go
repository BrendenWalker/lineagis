package api

import (
	"encoding/json"
	"net/http"
	"strings"
)

type evaluatePolicyRequest struct {
	Digest string `json:"digest"`
	Phase  string `json:"phase"`
}

type evaluatePolicyResponse struct {
	Namespace     string         `json:"namespace"`
	Artifact      string         `json:"artifact"`
	Digest        string         `json:"digest"`
	Phase         string         `json:"phase"`
	PolicyVersion *int           `json:"policy_version,omitempty"`
	Outcome       string         `json:"outcome"`
	Reasons       []PolicyReason `json:"reasons,omitempty"`
}

func (h *Handler) postEvaluatePolicy(w http.ResponseWriter, r *http.Request, ns, artifact string) {
	if ns == "" || artifact == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace and artifact are required", nil)
		return
	}

	var req evaluatePolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
		return
	}
	req.Digest = strings.TrimSpace(req.Digest)
	req.Phase = strings.TrimSpace(strings.ToLower(req.Phase))
	if req.Digest == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "digest is required", nil)
		return
	}
	if !strings.HasPrefix(req.Digest, "sha256:") {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "digest must be a sha256:… reference", nil)
		return
	}
	phase, err := parseEvalPhase(req.Phase)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
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

	result, err := evaluateActivePolicy(ctx, h.Store, namespace.ID, d.ID, phase, VerifyEvalOpts{})
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal server error", nil)
		return
	}

	if result.PolicyID != nil {
		reasonsJSON, err := json.Marshal(result.Reasons)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal server error", nil)
			return
		}
		if _, err := h.Store.RecordPolicyDecision(ctx, d.ID, *result.PolicyID, result.Outcome, reasonsJSON); err != nil {
			if mapStoreError(w, err) {
				return
			}
			WriteError(w, http.StatusInternalServerError, "INTERNAL", "internal server error", nil)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(evaluatePolicyResponse{
		Namespace:     ns,
		Artifact:      artifact,
		Digest:        d.Digest,
		Phase:         string(phase),
		PolicyVersion: result.PolicyVersion,
		Outcome:       result.Outcome,
		Reasons:       result.Reasons,
	})
}
