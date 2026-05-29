package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/BrendenWalker/verity/internal/metadata"
)

type auditEventResponse struct {
	ID           int64           `json:"id"`
	EventType    string          `json:"event_type"`
	Actor        *string         `json:"actor,omitempty"`
	ResourceType *string         `json:"resource_type,omitempty"`
	ResourceID   *string         `json:"resource_id,omitempty"`
	Payload      json.RawMessage `json:"payload"`
	CreatedAt    string          `json:"created_at"`
}

type listAuditResponse struct {
	Namespace string               `json:"namespace"`
	Events    []auditEventResponse `json:"events"`
}

func (h *Handler) getAuditEvents(w http.ResponseWriter, r *http.Request, ns string) {
	if ns == "" {
		WriteError(w, http.StatusBadRequest, "VALIDATION_FAILED", "namespace is required", nil)
		return
	}
	limit := 100
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 500 {
		limit = 500
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

	events, err := h.Store.ListAuditEvents(ctx, namespace.ID, limit)
	if err != nil {
		if mapStoreError(w, err) {
			return
		}
	}

	resp := listAuditResponse{Namespace: namespace.Name, Events: make([]auditEventResponse, 0, len(events))}
	for _, e := range events {
		resp.Events = append(resp.Events, auditEventFromRow(e))
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func auditEventFromRow(e metadata.AuditEvent) auditEventResponse {
	return auditEventResponse{
		ID:           e.ID,
		EventType:    e.EventType,
		Actor:        e.Actor,
		ResourceType: e.ResourceType,
		ResourceID:   e.ResourceID,
		Payload:      e.Payload,
		CreatedAt:    e.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

func (h *Handler) recordAudit(ctx context.Context, namespaceID int64, eventType string, actor *string, resourceType, resourceID *string, payload map[string]any) {
	if h == nil || h.Store == nil {
		return
	}
	var raw json.RawMessage
	if payload != nil {
		raw, _ = json.Marshal(payload)
	}
	nsID := namespaceID
	_, _ = h.Store.RecordAuditEvent(ctx, &nsID, eventType, actor, resourceType, resourceID, raw)
}
