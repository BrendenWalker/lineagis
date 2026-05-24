package registry

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// MaxBlobSize is the maximum layer blob size per ADR-0001 (512 MiB).
const MaxBlobSize = 512 << 20

var repoNamePattern = regexp.MustCompile(`^[a-z0-9]+([._-][a-z0-9]+)*(/[a-z0-9]+([._-][a-z0-9]+)*)*$`)

// Client performs OCI Distribution blob operations against a registry v2 endpoint.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient sets the HTTP client used for registry requests.
func WithHTTPClient(c *http.Client) Option {
	return func(cl *Client) {
		cl.httpClient = c
	}
}

// New creates a registry client for the given base URL (e.g. http://localhost:5000).
func New(registryURL string, opts ...Option) (*Client, error) {
	registryURL = strings.TrimSpace(registryURL)
	if registryURL == "" {
		return nil, errors.New("registry: URL is required")
	}

	u, err := url.Parse(registryURL)
	if err != nil {
		return nil, fmt.Errorf("registry: parse URL: %w", err)
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, fmt.Errorf("registry: URL must include scheme and host")
	}
	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawQuery = ""
	u.Fragment = ""

	cl := &Client{
		baseURL: u.String(),
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
	}
	for _, opt := range opts {
		opt(cl)
	}
	return cl, nil
}

func (c *Client) blobURL(repo, digest string) (string, error) {
	if err := validateRepo(repo); err != nil {
		return "", err
	}
	if digest == "" {
		return "", errors.New("registry: digest is required")
	}

	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("registry: parse base URL: %w", err)
	}

	pathPrefix := strings.TrimSuffix(base.Path, "/")
	base.Path = pathPrefix + "/v2/" + repo + "/blobs/" + digest
	return base.String(), nil
}

func (c *Client) uploadsURL(repo string) (string, error) {
	if err := validateRepo(repo); err != nil {
		return "", err
	}

	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("registry: parse base URL: %w", err)
	}

	pathPrefix := strings.TrimSuffix(base.Path, "/")
	base.Path = pathPrefix + "/v2/" + repo + "/blobs/uploads/"
	return base.String(), nil
}

func (c *Client) resolveReference(ref string) (string, error) {
	u, err := url.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("registry: parse reference: %w", err)
	}
	if u.IsAbs() {
		return u.String(), nil
	}

	base, err := url.Parse(c.baseURL)
	if err != nil {
		return "", fmt.Errorf("registry: parse base URL: %w", err)
	}
	return base.ResolveReference(u).String(), nil
}

func appendDigestParam(rawURL, digest string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("registry: parse upload location: %w", err)
	}
	q := u.Query()
	q.Set("digest", digest)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func validateRepo(repo string) error {
	repo = strings.Trim(repo, "/")
	if repo == "" {
		return errors.New("registry: repository name is required")
	}
	if !repoNamePattern.MatchString(repo) {
		return fmt.Errorf("registry: invalid repository name %q", repo)
	}
	return nil
}
