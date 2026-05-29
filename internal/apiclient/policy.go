package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// PolicyDocument is the active namespace policy JSON.
type PolicyDocument struct {
	Namespace string          `json:"namespace"`
	Version   int             `json:"version"`
	Document  json.RawMessage `json:"document"`
	IsActive  bool            `json:"is_active"`
}

// GetPolicy returns the active policy for a namespace.
func (c *Client) GetPolicy(ctx context.Context, namespace string) (*PolicyDocument, error) {
	path, err := joinURL(c.baseURL, "v1", "namespaces", namespace, "policy")
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var out PolicyDocument
	if err := c.do(req, http.StatusOK, &out); err != nil {
		return nil, err
	}
	if !out.IsActive || len(out.Document) == 0 {
		return nil, fmt.Errorf("no active policy for namespace %q", namespace)
	}
	return &out, nil
}
