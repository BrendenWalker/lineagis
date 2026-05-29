package apiclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SignatureRecord is a stored Sigstore bundle for a digest.
type SignatureRecord struct {
	ID        int64           `json:"id"`
	Digest    string          `json:"digest"`
	Bundle    json.RawMessage `json:"bundle,omitempty"`
	Issuer    *string         `json:"issuer,omitempty"`
	Subject   *string         `json:"subject,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

// ListSignatures returns signature bundles for a digest (FR-SIGN-005 local verify).
func (c *Client) ListSignatures(ctx context.Context, namespace, artifact, digest string) ([]SignatureRecord, error) {
	digest = strings.TrimSpace(digest)
	if digest == "" || !strings.HasPrefix(digest, "sha256:") {
		return nil, fmt.Errorf("digest must be a sha256:… reference")
	}
	path, err := joinURL(c.baseURL, "v1", "namespaces", namespace, "artifacts", artifact, "signatures")
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("digest", digest)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Signatures []SignatureRecord `json:"signatures"`
	}
	if err := c.do(req, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return out.Signatures, nil
}
