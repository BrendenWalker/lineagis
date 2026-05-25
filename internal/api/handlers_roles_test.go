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
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	ns, _ := store.CreateNamespace(ctx, "gh/acme/widget", nil)
	art, _ := store.RegisterArtifact(ctx, ns.ID, "widget")
	d, _ := store.RegisterDigest(ctx, art.ID, "sha256:abc", nil, nil)
	_, _ = store.AttachSignature(ctx, d.ID, nil, json.RawMessage(`{"bundle":"stub"}`), nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/v1/namespaces/gh/acme/widget/artifacts/widget/trust?digest=sha256:abc", nil)
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
