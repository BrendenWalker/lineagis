package apiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client calls the Verity control-plane API.
type Client struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// New creates an API client. baseURL is e.g. http://localhost:8080.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(strings.TrimSpace(baseURL), "/"),
		token:   strings.TrimSpace(token),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

type apiError struct {
	Code    string         `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

func (e apiError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("%s: %s", e.Code, e.Message)
	}
	return e.Message
}

// EnsureArtifact registers namespace and artifact (idempotent).
func (c *Client) EnsureArtifact(ctx context.Context, namespace, artifact string) error {
	path, err := joinURL(c.baseURL, "v1", "namespaces", namespace, "artifacts", artifact)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, path, nil)
	if err != nil {
		return err
	}
	return c.do(req, http.StatusOK, nil)
}

// RegisterDigest records a manifest digest after OCI push.
func (c *Client) RegisterDigest(ctx context.Context, namespace, artifact, digest string, mediaType *string, sizeBytes *int64) error {
	body, err := json.Marshal(map[string]any{
		"digest":     digest,
		"media_type": mediaType,
		"size_bytes": sizeBytes,
	})
	if err != nil {
		return err
	}
	path, err := joinURL(c.baseURL, "v1", "namespaces", namespace, "artifacts", artifact, "digests")
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, http.StatusCreated, nil)
}

// AttachSignature stores a Sigstore bundle for a registered digest (FR-SIGN-009).
func (c *Client) AttachSignature(ctx context.Context, namespace, artifact, digest string, bundle json.RawMessage, issuer, subject *string) error {
	body := map[string]any{
		"digest": digest,
		"bundle": bundle,
	}
	if issuer != nil {
		body["issuer"] = *issuer
	}
	if subject != nil {
		body["subject"] = *subject
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return err
	}
	path, err := joinURL(c.baseURL, "v1", "namespaces", namespace, "artifacts", artifact, "signatures")
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, path, bytes.NewReader(encoded))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, http.StatusCreated, nil)
}

// SetTag maps a semver tag to a digest.
func (c *Client) SetTag(ctx context.Context, namespace, artifact, tag, digest string) error {
	body, err := json.Marshal(map[string]string{"digest": digest})
	if err != nil {
		return err
	}
	path, err := joinURL(c.baseURL, "v1", "namespaces", namespace, "artifacts", artifact, "tags", tag)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	return c.do(req, http.StatusOK, nil)
}

func (c *Client) do(req *http.Request, wantStatus int, out any) error {
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("api request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("api read body: %w", err)
	}
	if resp.StatusCode != wantStatus {
		var apiErr apiError
		if json.Unmarshal(data, &apiErr) == nil && apiErr.Message != "" {
			return apiErr
		}
		return fmt.Errorf("api: unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	if out != nil && len(data) > 0 {
		if err := json.Unmarshal(data, out); err != nil {
			return fmt.Errorf("api decode response: %w", err)
		}
	}
	return nil
}

func joinURL(base string, elems ...string) (string, error) {
	u, err := url.Parse(strings.TrimSuffix(base, "/"))
	if err != nil {
		return "", err
	}
	path := "/" + strings.Join(elems, "/")
	u.Path = strings.TrimSuffix(u.Path, "/") + path
	return u.String(), nil
}
