package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

type putWebhookRequest struct {
	URL     string  `json:"url"`
	Secret  *string `json:"secret,omitempty"`
	Enabled *bool   `json:"enabled,omitempty"`
}

type webhookResponse struct {
	Name    string `json:"name"`
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

type listWebhooksResponse struct {
	Namespace string            `json:"namespace"`
	Webhooks  []webhookResponse `json:"webhooks"`
}

func (h *Handler) listWebhooks(w http.ResponseWriter, r *http.Request, ns string) {
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
	eps, err := h.Store.ListWebhookEndpoints(ctx, namespace.ID)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	out := make([]webhookResponse, 0, len(eps))
	for _, ep := range eps {
		out = append(out, webhookResponse{Name: ep.Name, URL: ep.URL, Enabled: ep.Enabled})
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(listWebhooksResponse{Namespace: ns, Webhooks: out})
}

func (h *Handler) putWebhook(w http.ResponseWriter, r *http.Request, ns, name string) {
	if name == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "webhook name is required", nil)
		return
	}
	var req putWebhookRequest
	if err := decodeJSON(r, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", err.Error(), nil)
		return
	}
	req.URL = strings.TrimSpace(req.URL)
	if req.URL == "" || !strings.HasPrefix(req.URL, "https://") {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "url must be an https:// endpoint", nil)
		return
	}
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
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
	ep, err := h.Store.PutWebhookEndpoint(ctx, namespace.ID, name, req.URL, req.Secret, enabled)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(webhookResponse{Name: ep.Name, URL: ep.URL, Enabled: ep.Enabled})
}

func (h *Handler) deleteWebhook(w http.ResponseWriter, r *http.Request, ns, name string) {
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
	if err := h.Store.DeleteWebhookEndpoint(ctx, namespace.ID, name); err != nil {
		if mapStoreError(w, err) {
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) emitWebhook(ctx context.Context, namespaceID int64, namespaceName, eventType string, body map[string]any, correlationID string) {
	if h.Webhooks != nil {
		h.Webhooks.Emit(ctx, namespaceID, namespaceName, eventType, body, correlationID)
	}
}
