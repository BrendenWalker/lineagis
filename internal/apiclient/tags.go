package apiclient

import (
	"context"
	"fmt"
	"net/http"
)

// TagResolution is the response from GetTag (FR-PUB-006).
type TagResolution struct {
	Namespace string `json:"namespace"`
	Artifact  string `json:"artifact"`
	Tag       string `json:"tag"`
	Digest    string `json:"digest"`
}

// GetTag resolves a semver tag to a digest.
func (c *Client) GetTag(ctx context.Context, namespace, artifact, tag string) (*TagResolution, error) {
	path, err := joinURL(c.baseURL, "v1", "namespaces", namespace, "artifacts", artifact, "tags", tag)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var out TagResolution
	if err := c.do(req, http.StatusOK, &out); err != nil {
		return nil, fmt.Errorf("get tag: %w", err)
	}
	return &out, nil
}
