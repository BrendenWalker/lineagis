package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/BrendenWalker/lineagis/internal/auth"
)

type putPolicyRequest struct {
	Document json.RawMessage `json:"document"`
}

type policyResponse struct {
	Namespace string          `json:"namespace"`
	Version   int             `json:"version"`
	Document  json.RawMessage `json:"document"`
	IsActive  bool            `json:"is_active"`
}

func (h *Handler) putPolicy(w http.ResponseWriter, r *http.Request, ns string) {
	if ns == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace is required", nil)
		return
	}

	var req putPolicyRequest
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
		return
	}
	if len(req.Document) == 0 {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "policy document is required", nil)
		return
	}
	if err := validatePolicyDocument(req.Document); err != nil {
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
	if err := authorizeOperator(ctx, ns, namespace.Config); err != nil {
		writeAuthError(w, err)
		return
	}

	actor, _ := auth.ActorFromContext(ctx)
	actorStr := actor.Subject
	p, err := h.Store.PutPolicy(ctx, namespace.ID, req.Document, &actorStr)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	h.emitWebhook(ctx, namespace.ID, ns, "policy.updated", map[string]any{
		"policy_version": p.Version,
	}, fmt.Sprintf("%d", p.ID))

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(policyResponse{
		Namespace: namespace.Name,
		Version:   p.Version,
		Document:  p.Document,
		IsActive:  p.IsActive,
	})
}

func (h *Handler) getPolicy(w http.ResponseWriter, r *http.Request, ns string) {
	if ns == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace is required", nil)
		return
	}

	ctx := r.Context()
	namespace, err := h.Store.GetNamespaceByName(ctx, ns)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	p, err := h.Store.GetActivePolicy(ctx, namespace.ID)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(policyResponse{
		Namespace: namespace.Name,
		Version:   p.Version,
		Document:  p.Document,
		IsActive:  p.IsActive,
	})
}
