package cliauth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// ResolveToken returns an API bearer token from env or GitHub Actions OIDC (FR-DX-011).
func ResolveToken(ctx context.Context) (string, error) {
	if t := strings.TrimSpace(os.Getenv("LINEAGIS_TOKEN")); t != "" {
		return t, nil
	}
	if t := strings.TrimSpace(os.Getenv("LINEAGIS_DEV_TOKEN")); t != "" {
		return t, nil
	}
	if t, err := fetchGitHubActionsIDToken(ctx, apiAudience()); err == nil && t != "" {
		return t, nil
	} else if err != nil && !isActionsNotConfigured(err) {
		return "", err
	}
	f, err := LoadFile()
	if err != nil {
		return "", err
	}
	if t := strings.TrimSpace(f.Token); t != "" {
		return t, nil
	}
	return "", fmt.Errorf("no API token: set LINEAGIS_TOKEN, run in GitHub Actions with id-token: write, or run lineagis login after setting LINEAGIS_TOKEN")
}

func apiAudience() string {
	if a := strings.TrimSpace(os.Getenv("LINEAGIS_OIDC_AUDIENCE")); a != "" {
		return a
	}
	return "lineagis-api"
}

func isActionsNotConfigured(err error) bool {
	return strings.Contains(err.Error(), "not configured")
}

// fetchGitHubActionsIDToken exchanges ACTIONS_ID_TOKEN_REQUEST_* for a JWT.
func fetchGitHubActionsIDToken(ctx context.Context, audience string) (string, error) {
	url := strings.TrimSpace(os.Getenv("ACTIONS_ID_TOKEN_REQUEST_URL"))
	tok := strings.TrimSpace(os.Getenv("ACTIONS_ID_TOKEN_REQUEST_TOKEN"))
	if url == "" || tok == "" {
		return "", fmt.Errorf("GitHub Actions OIDC not configured")
	}
	body, err := json.Marshal(map[string]string{"audience": audience})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request GitHub OIDC token: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub OIDC token request: %s", strings.TrimSpace(string(data)))
	}
	var out struct {
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("decode GitHub OIDC response: %w", err)
	}
	if strings.TrimSpace(out.Value) == "" {
		return "", fmt.Errorf("empty GitHub OIDC token in response")
	}
	return out.Value, nil
}

// Login persists env or Actions token to the config file.
func Login(ctx context.Context) (File, error) {
	token, err := ResolveToken(ctx)
	if err != nil {
		return File{}, err
	}
	f := File{
		APIURL:      strings.TrimSpace(os.Getenv("LINEAGIS_API_URL")),
		RegistryURL: strings.TrimSpace(os.Getenv("LINEAGIS_REGISTRY_URL")),
		Token:       token,
	}
	if f.APIURL == "" {
		f.APIURL = "http://localhost:8080"
	}
	if f.RegistryURL == "" {
		f.RegistryURL = "http://localhost:5000"
	}
	if err := SaveFile(f); err != nil {
		return File{}, err
	}
	return f, nil
}
