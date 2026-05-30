package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/BrendenWalker/verity/internal/api"
	"github.com/BrendenWalker/verity/internal/metadata"
)

type stubGitHubChecker struct {
	exists bool
	err    error
}

func (s *stubGitHubChecker) RepositoryExists(context.Context, string) (bool, error) {
	return s.exists, s.err
}

func setupRepositoryOwnershipPolicy(t *testing.T, h *api.Handler, store *metadata.Store) *metadata.Digest {
	t.Helper()
	ctx := context.Background()
	ns, err := store.CreateNamespace(ctx, "gh/acme/widget", nil)
	if err != nil {
		t.Fatal(err)
	}
	art, err := store.RegisterArtifact(ctx, ns.ID, "widget")
	if err != nil {
		t.Fatal(err)
	}
	d, err := store.RegisterDigest(ctx, art.ID, "sha256:ownership", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	att, err := store.AttachAttestation(ctx, d.ID, "https://slsa.dev/provenance/v1", nil, nil, json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	commit := "abc"
	if _, err := store.InsertProvenanceRecord(ctx, att.ID, d.ID, "https://github.com/acme/widget", &commit, nil, nil, nil, true); err != nil {
		t.Fatal(err)
	}
	doc := []byte(`{"rules":[{"id":"repository-ownership","config":{"verify_with_github_api":true}}]}`)
	if _, err := store.PutPolicy(ctx, ns.ID, doc, nil); err != nil {
		t.Fatal(err)
	}
	return d
}

func TestVerify_repositoryOwnership_githubNotConfigured(t *testing.T) {
	h, store, _ := testHandler(t, "test-token")
	d := setupRepositoryOwnershipPolicy(t, h, store)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	payload, _ := json.Marshal(map[string]string{"digest": d.Digest})
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/gh/acme/widget/artifacts/widget/verify", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Outcome string `json:"outcome"`
		Policy  struct {
			Status  string `json:"status"`
			Reasons []struct {
				Rule    string `json:"rule"`
				Message string `json:"message"`
			} `json:"reasons"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Outcome != "fail" || resp.Policy.Status != "fail" {
		t.Fatalf("got %+v", resp)
	}
	if len(resp.Policy.Reasons) == 0 || resp.Policy.Reasons[0].Rule != "repository-ownership" {
		t.Fatalf("reasons = %+v", resp.Policy.Reasons)
	}
	if !strings.Contains(resp.Policy.Reasons[0].Message, "VERITY_GITHUB_TOKEN") {
		t.Fatalf("message = %q", resp.Policy.Reasons[0].Message)
	}
}

func TestVerify_repositoryOwnership_githubAPIError(t *testing.T) {
	h, store, _ := testHandler(t, "test-token")
	h.GitHub = &stubGitHubChecker{err: errors.New("connection refused")}
	d := setupRepositoryOwnershipPolicy(t, h, store)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	payload, _ := json.Marshal(map[string]string{"digest": d.Digest})
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/gh/acme/widget/artifacts/widget/verify", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Outcome string `json:"outcome"`
		Policy  struct {
			Reasons []struct {
				Rule    string `json:"rule"`
				Message string `json:"message"`
			} `json:"reasons"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Outcome != "fail" {
		t.Fatalf("outcome = %q", resp.Outcome)
	}
	if len(resp.Policy.Reasons) == 0 || !strings.Contains(resp.Policy.Reasons[0].Message, "GitHub API verification failed") {
		t.Fatalf("reasons = %+v", resp.Policy.Reasons)
	}
}

func TestVerify_requireDigestOnVerify_rejectsTag(t *testing.T) {
	h, store, _ := testHandler(t, "test-token")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	ns, err := store.CreateNamespace(ctx, "gh/acme/widget", nil)
	if err != nil {
		t.Fatal(err)
	}
	art, err := store.RegisterArtifact(ctx, ns.ID, "widget")
	if err != nil {
		t.Fatal(err)
	}
	d, err := store.RegisterDigest(ctx, art.ID, "sha256:tagged", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetTag(ctx, art.ID, "v1.0.0", d.ID, nil); err != nil {
		t.Fatal(err)
	}
	doc := []byte(`{"rules":[{"id":"require-digest-on-verify"}]}`)
	if _, err := store.PutPolicy(ctx, ns.ID, doc, nil); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(map[string]string{"tag": "v1.0.0"})
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/gh/acme/widget/artifacts/widget/verify", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Outcome string `json:"outcome"`
		Policy  struct {
			Reasons []struct {
				Rule string `json:"rule"`
			} `json:"reasons"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Outcome != "fail" {
		t.Fatalf("outcome = %q, want fail", resp.Outcome)
	}
	if len(resp.Policy.Reasons) != 1 || resp.Policy.Reasons[0].Rule != "require-digest-on-verify" {
		t.Fatalf("reasons = %+v", resp.Policy.Reasons)
	}
}
