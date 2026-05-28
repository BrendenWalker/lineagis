package apiclient

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// PolicyReason is a single verify-time policy rule outcome (FR-POL-009).
type PolicyReason struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// TrustStatus is the aggregated trust report from GetTrustStatus (FR-API-008).
type TrustStatus struct {
	Namespace  string `json:"namespace"`
	Artifact   string `json:"artifact"`
	Digest     string `json:"digest"`
	Overall    string `json:"overall"`
	Signatures struct {
		Status string `json:"status"`
	} `json:"signatures"`
	Policy struct {
		Status  string         `json:"status"`
		Reasons []PolicyReason `json:"reasons,omitempty"`
	} `json:"policy"`
	Attestations struct {
		Provenance         bool   `json:"provenance"`
		SBOM               bool   `json:"sbom"`
		ProvenanceVerified bool   `json:"provenance_verified"`
		Repository         string `json:"repository,omitempty"`
		Commit             string `json:"commit,omitempty"`
		Workflow           string `json:"workflow,omitempty"`
		WorkflowRef        string `json:"workflow_ref,omitempty"`
		RunID              string `json:"run_id,omitempty"`
	} `json:"attestations"`
}

// GetTrustStatus returns trust for a digest or tag (FR-SIGN-006).
func (c *Client) GetTrustStatus(ctx context.Context, namespace, artifact, digest, tag string) (*TrustStatus, error) {
	digest = strings.TrimSpace(digest)
	tag = strings.TrimSpace(tag)
	if digest == "" && tag == "" {
		return nil, fmt.Errorf("digest or tag is required")
	}
	if digest != "" && tag != "" {
		return nil, fmt.Errorf("provide digest or tag, not both")
	}

	path, err := joinURL(c.baseURL, "v1", "namespaces", namespace, "artifacts", artifact, "trust")
	if err != nil {
		return nil, err
	}
	u, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	q := u.Query()
	if digest != "" {
		q.Set("digest", digest)
	} else {
		q.Set("tag", tag)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	var out TrustStatus
	if err := c.do(req, http.StatusOK, &out); err != nil {
		return nil, err
	}
	return &out, nil
}
