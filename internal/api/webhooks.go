package api

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/BrendenWalker/verity/internal/metadata"
)

const webhookMaxAttempts = 5

// WebhookDispatcher delivers namespace webhook events (FR-API-012).
type WebhookDispatcher struct {
	Store      *metadata.Store
	HTTPClient *http.Client
}

// Emit schedules delivery of an event to all enabled endpoints in the namespace.
func (d *WebhookDispatcher) Emit(ctx context.Context, namespaceID int64, namespaceName, eventType string, body map[string]any, correlationID string) {
	if d == nil || d.Store == nil {
		return
	}
	endpoints, err := d.Store.ListEnabledWebhookEndpoints(ctx, namespaceID)
	if err != nil || len(endpoints) == 0 {
		return
	}
	payload := map[string]any{
		"event_type":     eventType,
		"namespace":      namespaceName,
		"correlation_id": correlationID,
	}
	for k, v := range body {
		payload[k] = v
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	client := d.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	for _, ep := range endpoints {
		ep := ep
		go d.deliverWithRetry(ep, raw, client)
	}
}

func (d *WebhookDispatcher) deliverWithRetry(ep metadata.WebhookEndpoint, body []byte, client *http.Client) {
	var lastErr error
	for attempt := 1; attempt <= webhookMaxAttempts; attempt++ {
		if err := deliverWebhook(client, ep, body); err == nil {
			return
		} else {
			lastErr = err
		}
		time.Sleep(time.Duration(attempt) * 200 * time.Millisecond)
	}
	_ = lastErr
}

func deliverWebhook(client *http.Client, ep metadata.WebhookEndpoint, body []byte) error {
	req, err := http.NewRequest(http.MethodPost, ep.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "verity-webhooks/1.0")
	if ep.Secret != nil && strings.TrimSpace(*ep.Secret) != "" {
		mac := hmac.New(sha256.New, []byte(*ep.Secret))
		_, _ = mac.Write(body)
		req.Header.Set("X-Verity-Signature", hex.EncodeToString(mac.Sum(nil)))
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook %s returned %s", ep.Name, resp.Status)
	}
	return nil
}
