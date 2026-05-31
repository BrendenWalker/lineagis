package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/BrendenWalker/lineagis/internal/provenance"
)

type attachAttestationRequest struct {
	Digest        string          `json:"digest"`
	PredicateType string          `json:"predicate_type"`
	Statement     json.RawMessage `json:"statement"`
	Bundle        json.RawMessage `json:"bundle"`
}

type attestationResponse struct {
	ID            int64     `json:"id"`
	Digest        string    `json:"digest"`
	PredicateType string    `json:"predicate_type"`
	CreatedAt     time.Time `json:"created_at"`
}

func (h *Handler) postAttachAttestation(w http.ResponseWriter, r *http.Request, ns, artifact, digestRef string) {
	if ns == "" || artifact == "" || digestRef == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace, artifact, and digest are required", nil)
		return
	}
	digestRef = strings.TrimSpace(digestRef)
	if !strings.HasPrefix(digestRef, "sha256:") {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "digest must be a sha256:… reference", nil)
		return
	}

	var req attachAttestationRequest
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
		return
	}
	if strings.TrimSpace(req.PredicateType) == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "predicate_type is required", nil)
		return
	}
	if len(req.Statement) == 0 || len(req.Bundle) == 0 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "statement and bundle are required", nil)
		return
	}

	envelope, err := json.Marshal(attestationEnvelope{
		Statement: req.Statement,
		Bundle:    req.Bundle,
	})
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INTERNAL", "encode envelope", nil)
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

	stmt, verified, err := verifyAttestationEnvelope(ctx, envelope)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
		return
	}

	att, err := h.Store.AttachAttestation(ctx, d.ID, req.PredicateType, nil, nil, envelope)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	if isProvenancePredicate(req.PredicateType) {
		fields := provenance.ParseFields(stmt)
		_, err = h.Store.InsertProvenanceRecord(ctx, att.ID, d.ID, fields.RepositoryURI,
			strPtr(fields.CommitSHA), strPtr(fields.WorkflowName), strPtr(fields.WorkflowRef), strPtr(fields.RunID), verified)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "INTERNAL", err.Error(), nil)
			return
		}
	}

	actor := ActorFromContext(ctx)
	var actorPtr *string
	if actor != "" {
		actorPtr = &actor
	}
	resID := fmt.Sprintf("%d", att.ID)
	h.recordAudit(ctx, namespace.ID, "attestation.attached", actorPtr, strPtr("attestation"), &resID, map[string]any{
		"digest":         d.Digest,
		"predicate_type": req.PredicateType,
	})

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(attestationResponse{
		ID:            att.ID,
		Digest:        d.Digest,
		PredicateType: att.PredicateType,
		CreatedAt:     att.CreatedAt,
	})
}
