package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BrendenWalker/verity/internal/api"
	"github.com/BrendenWalker/verity/internal/auth"
	"github.com/BrendenWalker/verity/internal/metadata"
	"github.com/BrendenWalker/verity/internal/registry"
	"github.com/BrendenWalker/verity/internal/signing"
)

func testHandlerWithActor(t *testing.T, actor auth.Actor) (*api.Handler, *metadata.Store) {
	h, store, _ := testHandler(t, "unused")
	h.Auth = func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			next.ServeHTTP(w, r.WithContext(auth.ContextWithActor(r.Context(), actor)))
		})
	}
	return h, store
}

func TestPutPolicy_requiresOperator(t *testing.T) {
	h, store := testHandlerWithActor(t, auth.Actor{Subject: "bob"})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	cfg := json.RawMessage(`{"operators":["alice"]}`)
	ns, err := store.CreateNamespace(ctx, "gh/acme/widget", cfg)
	if err != nil {
		t.Fatal(err)
	}
	_ = ns

	body := []byte(`{"document":{"rules":[]}}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/namespaces/gh/acme/widget/policy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
}

func TestPutPolicy_operatorAllowed(t *testing.T) {
	h, store := testHandlerWithActor(t, auth.Actor{Subject: "alice"})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	cfg := json.RawMessage(`{"operators":["alice"]}`)
	ns, err := store.CreateNamespace(ctx, "gh/acme/widget", cfg)
	if err != nil {
		t.Fatal(err)
	}
	_ = ns

	body := []byte(`{"document":{"rules":[{"type":"require-signature"}]}}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/namespaces/gh/acme/widget/policy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestPutPolicy_invalidDocumentSchema(t *testing.T) {
	h, store := testHandlerWithActor(t, auth.Actor{Subject: "alice"})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	cfg := json.RawMessage(`{"operators":["alice"]}`)
	_, err := store.CreateNamespace(ctx, "gh/acme/widget", cfg)
	if err != nil {
		t.Fatal(err)
	}

	body := []byte(`{"document":{"rules":[{}]}}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/namespaces/gh/acme/widget/policy", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d body = %s", rec.Code, http.StatusBadRequest, rec.Body.String())
	}
	var errBody struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody.Code != "VALIDATION_FAILED" {
		t.Fatalf("code = %q, want VALIDATION_FAILED", errBody.Code)
	}
}

func TestGetPolicy_returnsActiveVersion(t *testing.T) {
	h, store := testHandlerWithActor(t, auth.Actor{Subject: "alice"})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	_, err := store.CreateNamespace(ctx, "gh/acme/widget", nil)
	if err != nil {
		t.Fatal(err)
	}

	// Use API writes to ensure active version tracking matches endpoint behavior.
	for _, body := range [][]byte{
		[]byte(`{"document":{"rules":[{"id":"require-signatures"}]}}`),
		[]byte(`{"document":{"rules":[{"type":"trusted-publishers"}]}}`),
	} {
		req := httptest.NewRequest(http.MethodPut, "/v1/namespaces/gh/acme/widget/policy", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("put status = %d body = %s", rec.Code, rec.Body.String())
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/namespaces/gh/acme/widget/policy", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d body = %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var resp struct {
		Namespace string `json:"namespace"`
		Version   int    `json:"version"`
		Document  struct {
			Rules []struct {
				Type string `json:"type"`
			} `json:"rules"`
		} `json:"document"`
		IsActive bool `json:"is_active"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Namespace != "gh/acme/widget" || resp.Version != 2 || !resp.IsActive {
		t.Fatalf("unexpected response: %+v", resp)
	}
	if len(resp.Document.Rules) == 0 || resp.Document.Rules[0].Type != "trusted-publishers" {
		t.Fatalf("unexpected document: %+v", resp.Document)
	}
}

func TestGetArtifact_authRequired(t *testing.T) {
	h, store, _ := testHandler(t, "test-token")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	ns, _ := store.CreateNamespace(ctx, "gh/acme/widget", nil)
	_, _ = store.RegisterArtifact(ctx, ns.ID, "widget")

	req := httptest.NewRequest(http.MethodGet, "/v1/namespaces/gh/acme/widget/artifacts/widget", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestGetTrustStatus_signedPass(t *testing.T) {
	h, store, _ := testHandler(t, "test-token")
	manifestJSON, digest, err := registry.BuildArtifactManifest(
		[]registry.FileLayer{{Path: "bin/app", Data: []byte("x")}},
		registry.ManifestOptions{},
	)
	if err != nil {
		t.Fatal(err)
	}
	bundle, _, err := signing.SignManifestForTest(manifestJSON)
	if err != nil {
		t.Fatal(err)
	}
	h.Manifests = api.NewStaticManifestSource(map[string][]byte{digest.String(): manifestJSON})

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	ns, _ := store.CreateNamespace(ctx, "gh/acme/widget", nil)
	art, _ := store.RegisterArtifact(ctx, ns.ID, "widget")
	d, _ := store.RegisterDigest(ctx, art.ID, digest.String(), nil, nil)
	_, _ = store.AttachSignature(ctx, d.ID, nil, bundle, nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/namespaces/gh/acme/widget/artifacts/widget/trust?digest="+digest.String(), nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Overall    string `json:"overall"`
		Signatures struct {
			Status string `json:"status"`
		} `json:"signatures"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Signatures.Status != "valid" || resp.Overall != "pass" {
		t.Fatalf("got %+v", resp)
	}
}
