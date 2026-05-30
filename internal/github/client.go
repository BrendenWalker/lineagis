package github

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Client calls the GitHub REST API (FR-POL-013).
type Client struct {
	token      string
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a GitHub API client using VERITY_GITHUB_TOKEN.
func NewClient(token string) *Client {
	return &Client{
		token:   strings.TrimSpace(token),
		baseURL: "https://api.github.com",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// SetBaseURL overrides the API base URL (tests only).
func (c *Client) SetBaseURL(base string) {
	if c != nil {
		c.baseURL = strings.TrimSuffix(strings.TrimSpace(base), "/")
	}
}

// RepositoryExists reports whether owner/repo exists (GET /repos/{owner}/{repo}).
func (c *Client) RepositoryExists(ctx context.Context, ownerRepo string) (bool, error) {
	if c == nil || c.token == "" {
		return false, fmt.Errorf("GitHub token is not configured")
	}
	ownerRepo = strings.Trim(ownerRepo, "/")
	parts := strings.SplitN(ownerRepo, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false, fmt.Errorf("invalid repository %q", ownerRepo)
	}
	url := fmt.Sprintf("%s/repos/%s/%s", c.baseURL, parts[0], parts[1])
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusOK:
		return true, nil
	case http.StatusNotFound:
		return false, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return false, fmt.Errorf("GitHub API auth failed: %s", resp.Status)
	default:
		return false, fmt.Errorf("GitHub API: %s", resp.Status)
	}
}
