package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/BrendenWalker/verity/internal/api"
	"github.com/BrendenWalker/verity/internal/db"
	"github.com/BrendenWalker/verity/internal/metadata"
)

var (
	testDBOnce sync.Once
	testDBErr  error
)

func TestMain(m *testing.M) {
	url := os.Getenv("VERITY_TEST_DATABASE_URL")
	if url != "" {
		testDBOnce.Do(func() {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			pool, err := db.OpenPool(ctx, url)
			if err != nil {
				testDBErr = err
				return
			}
			defer pool.Close()
			testDBErr = db.MigrateUp(ctx, pool)
		})
	}
	os.Exit(m.Run())
}

func testHandler(t *testing.T, token string) (*api.Handler, *metadata.Store, *pgxpool.Pool) {
	t.Helper()
	url := os.Getenv("VERITY_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("VERITY_TEST_DATABASE_URL not set")
	}
	if testDBErr != nil {
		t.Fatalf("migrate: %v", testDBErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	pool, err := db.OpenPool(ctx, url)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	if err := truncate(ctx, pool); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	store := metadata.NewStore(pool)
	h := &api.Handler{
		Store:  store,
		Policy: api.AllowAllPolicy{},
		Auth: func(next http.Handler) http.Handler {
			return api.RequireBearer(token, next)
		},
	}
	return h, store, pool
}

func truncate(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		TRUNCATE TABLE tag_events, tags, digests, artifacts, namespaces RESTART IDENTITY CASCADE
	`)
	return err
}

func TestSetTag_authRequired(t *testing.T) {
	t.Parallel()
	h := &api.Handler{
		Store:  nil,
		Policy: api.AllowAllPolicy{},
		Auth: func(next http.Handler) http.Handler {
			return api.RequireBearer("secret", next)
		},
	}
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPut, "/v1/namespaces/gh/acme/widget/artifacts/widget/tags/v1.0.0", bytes.NewReader([]byte(`{"digest":"sha256:abc"}`)))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

func TestRegisterDigest_idempotent(t *testing.T) {
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
	_ = art

	body := []byte(`{"digest":"sha256:deadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeefdeadbeef"}`)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/gh/acme/widget/artifacts/widget/digests", bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusCreated {
			t.Fatalf("attempt %d: status = %d body = %s", i+1, rec.Code, rec.Body.String())
		}
	}
}

func TestSetTag_movePreservesDigest(t *testing.T) {
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
	d1, err := store.RegisterDigest(ctx, art.ID, "sha256:d111", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	d2, err := store.RegisterDigest(ctx, art.ID, "sha256:d222", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	_ = d1

	set := func(digest string) {
		t.Helper()
		payload, _ := json.Marshal(map[string]string{"digest": digest})
		req := httptest.NewRequest(http.MethodPut, "/v1/namespaces/gh/acme/widget/artifacts/widget/tags/v1.0.0", bytes.NewReader(payload))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("SetTag status = %d body = %s", rec.Code, rec.Body.String())
		}
	}

	set("sha256:d111")
	set("sha256:d222")

	got, err := store.GetDigestByID(ctx, d1.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Digest != "sha256:d111" {
		t.Fatalf("digest d1 changed: %s", got.Digest)
	}
	got2, err := store.GetDigestByID(ctx, d2.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got2.Digest != "sha256:d222" {
		t.Fatalf("digest d2 = %s", got2.Digest)
	}
}

func TestSetTag_invalidSemver(t *testing.T) {
	h, store, _ := testHandler(t, "test-token")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	ctx := context.Background()
	ns, _ := store.CreateNamespace(ctx, "gh/acme/widget", nil)
	art, _ := store.RegisterArtifact(ctx, ns.ID, "widget")
	_, _ = store.RegisterDigest(ctx, art.ID, "sha256:abc", nil, nil)

	payload := []byte(`{"digest":"sha256:abc"}`)
	req := httptest.NewRequest(http.MethodPut, "/v1/namespaces/gh/acme/widget/artifacts/widget/tags/latest", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
