package api_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/BrendenWalker/verity/internal/api"
	"github.com/BrendenWalker/verity/internal/db"
	"github.com/BrendenWalker/verity/internal/metadata"
	"github.com/BrendenWalker/verity/internal/registry"
	"github.com/BrendenWalker/verity/internal/signing"
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
		TRUNCATE TABLE
			audit_events,
			policy_decisions,
			policies,
			attestations,
			signatures,
			tag_events,
			tags,
			digests,
			artifacts,
			namespaces
		RESTART IDENTITY CASCADE
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

func TestRegisterDigest_requireSignatures_requiresBundle(t *testing.T) {
	h, store, _ := testHandler(t, "test-token")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	layers := []registry.FileLayer{{Path: "bin/app", Data: []byte("register-policy-test")}}
	manifestJSON, digest, err := registry.BuildArtifactManifest(layers, registry.ManifestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	digestStr := digest.String()
	h.Manifests = api.NewStaticManifestSource(map[string][]byte{digestStr: manifestJSON})

	ctx := context.Background()
	ns, err := store.CreateNamespace(ctx, "gh/acme/widget", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.RegisterArtifact(ctx, ns.ID, "widget"); err != nil {
		t.Fatal(err)
	}
	doc := []byte(`{"rules":[{"id":"require-signatures"}]}`)
	if _, err := store.PutPolicy(ctx, ns.ID, doc, nil); err != nil {
		t.Fatal(err)
	}

	unsignedBody, _ := json.Marshal(map[string]string{"digest": digestStr})
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/gh/acme/widget/artifacts/widget/digests", bytes.NewReader(unsignedBody))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("unsigned register status = %d body = %s", rec.Code, rec.Body.String())
	}

	bundle, _, err := signing.SignManifestForTest(manifestJSON)
	if err != nil {
		t.Fatal(err)
	}
	signedBody, _ := json.Marshal(map[string]any{"digest": digestStr, "bundle": bundle})
	req = httptest.NewRequest(http.MethodPost, "/v1/namespaces/gh/acme/widget/artifacts/widget/digests", bytes.NewReader(signedBody))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("signed register status = %d body = %s", rec.Code, rec.Body.String())
	}
}

// AC-API-002 / AC-POL-001: require-signatures blocks unsigned SetTag.
func TestSetTag_requireSignatures_policyFailed(t *testing.T) {
	h, store, _ := testHandler(t, "test-token")
	h.Policy = api.NewStorePushPolicy(store)
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
	d, err := store.RegisterDigest(ctx, art.ID, "sha256:unsigned", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := []byte(`{"rules":[{"id":"require-signatures"}]}`)
	if _, err := store.PutPolicy(ctx, ns.ID, doc, nil); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(map[string]string{"digest": d.Digest})
	req := httptest.NewRequest(http.MethodPut, "/v1/namespaces/gh/acme/widget/artifacts/widget/tags/v1.0.0", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d body = %s", rec.Code, http.StatusForbidden, rec.Body.String())
	}
	var body struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Details struct {
			Rule string `json:"rule"`
			Hint string `json:"hint"`
		} `json:"details"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error: %v", err)
	}
	if body.Code != "POLICY_FAILED" {
		t.Fatalf("code = %q, want POLICY_FAILED", body.Code)
	}
	if body.Details.Rule != "require-signatures" {
		t.Fatalf("details.rule = %q, want require-signatures", body.Details.Rule)
	}
	if body.Details.Hint == "" || !strings.Contains(body.Details.Hint, "signature") {
		t.Fatalf("details.hint = %q, want remediation hint", body.Details.Hint)
	}
	if !strings.Contains(body.Message, "require-signatures") {
		t.Fatalf("message = %q, want require-signatures hint", body.Message)
	}

	_, err = store.GetTag(ctx, art.ID, "v1.0.0")
	if !errors.Is(err, metadata.ErrNotFound) {
		t.Fatalf("tag should not exist: err=%v", err)
	}
}

func TestSetTag_requireSignatures_passesWithSignature(t *testing.T) {
	h, store, _ := testHandler(t, "test-token")
	h.Policy = api.NewStorePushPolicy(store)
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
	d, err := store.RegisterDigest(ctx, art.ID, "sha256:signed", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := []byte(`{"rules":[{"id":"require-signatures"}]}`)
	if _, err := store.PutPolicy(ctx, ns.ID, doc, nil); err != nil {
		t.Fatal(err)
	}
	bundle := json.RawMessage(`{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json"}`)
	if _, err := store.AttachSignature(ctx, d.ID, nil, bundle, nil, nil); err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(map[string]string{"digest": d.Digest})
	req := httptest.NewRequest(http.MethodPut, "/v1/namespaces/gh/acme/widget/artifacts/widget/tags/v1.0.0", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	tag, err := store.GetTag(ctx, art.ID, "v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if tag.DigestID != d.ID {
		t.Fatalf("tag digest_id = %d, want %d", tag.DigestID, d.ID)
	}
}

func TestAttachSignature_authRequired(t *testing.T) {
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

	body := []byte(`{"digest":"sha256:abc","bundle":{"stub":true}}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/gh/acme/widget/artifacts/widget/signatures", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}
}

// AC-META-003 / FR-SIGN-009: attach via API and list signatures indexed by digest.
func TestAttachSignature_storeAndList(t *testing.T) {
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
	d, err := store.RegisterDigest(ctx, art.ID, "sha256:signeddigest", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	attachBody, _ := json.Marshal(map[string]any{
		"digest":  d.Digest,
		"bundle":  map[string]string{"mediaType": "application/vnd.dev.sigstore.bundle.v0.3+json"},
		"issuer":  "https://token.actions.githubusercontent.com",
		"subject": "repo:acme/widget:ref:refs/heads/main",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/gh/acme/widget/artifacts/widget/signatures", bytes.NewReader(attachBody))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("AttachSignature status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID     int64  `json:"id"`
		Digest string `json:"digest"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode attach response: %v", err)
	}
	if created.ID == 0 || created.Digest != d.Digest {
		t.Fatalf("attach response = %+v", created)
	}

	listReq := httptest.NewRequest(http.MethodGet, "/v1/namespaces/gh/acme/widget/artifacts/widget/signatures?digest="+d.Digest, nil)
	listReq.Header.Set("Authorization", "Bearer test-token")
	listRec := httptest.NewRecorder()
	mux.ServeHTTP(listRec, listReq)
	if listRec.Code != http.StatusOK {
		t.Fatalf("ListSignatures status = %d body = %s", listRec.Code, listRec.Body.String())
	}
	var listed struct {
		Digest     string `json:"digest"`
		Signatures []struct {
			ID int64 `json:"id"`
		} `json:"signatures"`
	}
	if err := json.Unmarshal(listRec.Body.Bytes(), &listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if listed.Digest != d.Digest {
		t.Fatalf("listed digest = %q, want %q", listed.Digest, d.Digest)
	}
	if len(listed.Signatures) != 1 || listed.Signatures[0].ID != created.ID {
		t.Fatalf("listed signatures = %+v, want id %d", listed.Signatures, created.ID)
	}
}

func TestListSignatures_scopedToDigest(t *testing.T) {
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
	d1, err := store.RegisterDigest(ctx, art.ID, "sha256:sigscope1", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	d2, err := store.RegisterDigest(ctx, art.ID, "sha256:sigscope2", nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	attachBody, _ := json.Marshal(map[string]any{
		"digest":     d1.Digest,
		"bundle_ref": "oci://example/sig.bundle",
	})
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/gh/acme/widget/artifacts/widget/signatures", bytes.NewReader(attachBody))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("AttachSignature status = %d body = %s", rec.Code, rec.Body.String())
	}

	listD2 := httptest.NewRequest(http.MethodGet, "/v1/namespaces/gh/acme/widget/artifacts/widget/signatures?digest="+d2.Digest, nil)
	listD2.Header.Set("Authorization", "Bearer test-token")
	recD2 := httptest.NewRecorder()
	mux.ServeHTTP(recD2, listD2)
	if recD2.Code != http.StatusOK {
		t.Fatalf("ListSignatures d2 status = %d body = %s", recD2.Code, recD2.Body.String())
	}
	var listed struct {
		Signatures []any `json:"signatures"`
	}
	if err := json.Unmarshal(recD2.Body.Bytes(), &listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Signatures) != 0 {
		t.Fatalf("expected 0 signatures on d2, got %d", len(listed.Signatures))
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

func TestVerify_authRequired(t *testing.T) {
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

	body := []byte(`{"digest":"sha256:abc"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/gh/acme/widget/artifacts/widget/verify", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d body = %s", rec.Code, http.StatusUnauthorized, rec.Body.String())
	}
	var errBody struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody.Code != "AUTH_REQUIRED" {
		t.Fatalf("code = %q, want AUTH_REQUIRED", errBody.Code)
	}
}

func TestVerify_requireSignatures_fail(t *testing.T) {
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
	d, err := store.RegisterDigest(ctx, art.ID, "sha256:unsigned", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := []byte(`{"rules":[{"id":"require-signatures"}]}`)
	if _, err := store.PutPolicy(ctx, ns.ID, doc, nil); err != nil {
		t.Fatal(err)
	}

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
				Rule string `json:"rule"`
			} `json:"reasons"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Outcome != "fail" || resp.Policy.Status != "fail" {
		t.Fatalf("got %+v", resp)
	}
	if len(resp.Policy.Reasons) == 0 || resp.Policy.Reasons[0].Rule != "require-signatures" {
		t.Fatalf("reasons = %+v", resp.Policy.Reasons)
	}

	decision, err := store.LatestPolicyDecision(ctx, d.ID)
	if err != nil {
		t.Fatal(err)
	}
	if decision.Outcome != "fail" {
		t.Fatalf("decision outcome = %q, want fail", decision.Outcome)
	}
}

func TestVerify_signedPass(t *testing.T) {
	h, store, _ := testHandler(t, "test-token")
	manifestJSON, digest, err := registry.BuildArtifactManifest(
		[]registry.FileLayer{{Path: "bin/app", Data: []byte("signed-payload")}},
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
	ns, err := store.CreateNamespace(ctx, "gh/acme/widget", nil)
	if err != nil {
		t.Fatal(err)
	}
	art, err := store.RegisterArtifact(ctx, ns.ID, "widget")
	if err != nil {
		t.Fatal(err)
	}
	d, err := store.RegisterDigest(ctx, art.ID, digest.String(), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := []byte(`{"rules":[{"id":"require-signatures"}]}`)
	if _, err := store.PutPolicy(ctx, ns.ID, doc, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := store.AttachSignature(ctx, d.ID, nil, bundle, nil, nil); err != nil {
		t.Fatal(err)
	}

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
		Outcome    string `json:"outcome"`
		Signatures struct {
			Status string `json:"status"`
		} `json:"signatures"`
		Policy struct {
			Status string `json:"status"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Outcome != "pass" || resp.Signatures.Status != "valid" || resp.Policy.Status != "pass" {
		t.Fatalf("got %+v", resp)
	}
}

func TestGetTrustStatus_requireSignatures_fail(t *testing.T) {
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
	d, err := store.RegisterDigest(ctx, art.ID, "sha256:unsigned", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := []byte(`{"rules":[{"id":"require-signatures"}]}`)
	if _, err := store.PutPolicy(ctx, ns.ID, doc, nil); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/namespaces/gh/acme/widget/artifacts/widget/trust?digest="+d.Digest, nil)
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
		Policy struct {
			Status string `json:"status"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Overall != "fail" || resp.Signatures.Status != "missing" || resp.Policy.Status != "fail" {
		t.Fatalf("got %+v", resp)
	}
	var withReasons struct {
		Policy struct {
			Reasons []struct {
				Rule string `json:"rule"`
			} `json:"reasons"`
		} `json:"policy"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &withReasons); err != nil {
		t.Fatal(err)
	}
	if len(withReasons.Policy.Reasons) == 0 || withReasons.Policy.Reasons[0].Rule != "require-signatures" {
		t.Fatalf("policy reasons = %+v", withReasons.Policy.Reasons)
	}
}

func TestEvaluatePolicy_deterministic(t *testing.T) {
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
	d, err := store.RegisterDigest(ctx, art.ID, "sha256:unsigned", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := []byte(`{"rules":[{"id":"require-signatures"}]}`)
	if _, err := store.PutPolicy(ctx, ns.ID, doc, nil); err != nil {
		t.Fatal(err)
	}

	evaluate := func(phase string) []byte {
		t.Helper()
		payload, _ := json.Marshal(map[string]string{"digest": d.Digest, "phase": phase})
		req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/gh/acme/widget/artifacts/widget/policy/evaluate", bytes.NewReader(payload))
		req.Header.Set("Authorization", "Bearer test-token")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("phase %s: status = %d body = %s", phase, rec.Code, rec.Body.String())
		}
		return rec.Body.Bytes()
	}

	first := evaluate("verify")
	second := evaluate("verify")
	if string(first) != string(second) {
		t.Fatalf("verify evaluations differ:\nfirst=%s\nsecond=%s", first, second)
	}

	var resp struct {
		Outcome       string `json:"outcome"`
		PolicyVersion int    `json:"policy_version"`
		Reasons       []struct {
			Rule    string `json:"rule"`
			Message string `json:"message"`
		} `json:"reasons"`
	}
	if err := json.Unmarshal(first, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Outcome != "fail" || resp.PolicyVersion != 1 {
		t.Fatalf("got %+v", resp)
	}
	if len(resp.Reasons) != 1 || resp.Reasons[0].Rule != "require-signatures" {
		t.Fatalf("reasons = %+v", resp.Reasons)
	}

	pushBody := evaluate("push")
	if err := json.Unmarshal(pushBody, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Outcome != "fail" {
		t.Fatalf("push phase: got %+v", resp)
	}
}

func TestEvaluatePolicy_pushMatchesSetTag(t *testing.T) {
	h, store, _ := testHandler(t, "test-token")
	h.Policy = api.NewStorePushPolicy(store)
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
	d, err := store.RegisterDigest(ctx, art.ID, "sha256:unsigned", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	doc := []byte(`{"rules":[{"id":"require-signatures"}]}`)
	if _, err := store.PutPolicy(ctx, ns.ID, doc, nil); err != nil {
		t.Fatal(err)
	}

	evalPayload, _ := json.Marshal(map[string]string{"digest": d.Digest, "phase": "push"})
	evalReq := httptest.NewRequest(http.MethodPost, "/v1/namespaces/gh/acme/widget/artifacts/widget/policy/evaluate", bytes.NewReader(evalPayload))
	evalReq.Header.Set("Authorization", "Bearer test-token")
	evalReq.Header.Set("Content-Type", "application/json")
	evalRec := httptest.NewRecorder()
	mux.ServeHTTP(evalRec, evalReq)
	if evalRec.Code != http.StatusOK {
		t.Fatalf("evaluate: status = %d body = %s", evalRec.Code, evalRec.Body.String())
	}
	var evalResp struct {
		Outcome string `json:"outcome"`
	}
	if err := json.Unmarshal(evalRec.Body.Bytes(), &evalResp); err != nil {
		t.Fatal(err)
	}
	if evalResp.Outcome != "fail" {
		t.Fatalf("evaluate outcome = %q, want fail", evalResp.Outcome)
	}

	tagPayload, _ := json.Marshal(map[string]string{"digest": d.Digest})
	tagReq := httptest.NewRequest(http.MethodPut, "/v1/namespaces/gh/acme/widget/artifacts/widget/tags/v1.0.0", bytes.NewReader(tagPayload))
	tagReq.Header.Set("Authorization", "Bearer test-token")
	tagReq.Header.Set("Content-Type", "application/json")
	tagRec := httptest.NewRecorder()
	mux.ServeHTTP(tagRec, tagReq)
	if tagRec.Code != http.StatusForbidden {
		t.Fatalf("set tag: status = %d body = %s", tagRec.Code, tagRec.Body.String())
	}
	var errBody struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(tagRec.Body.Bytes(), &errBody); err != nil {
		t.Fatal(err)
	}
	if errBody.Code != "POLICY_FAILED" {
		t.Fatalf("code = %q, want POLICY_FAILED", errBody.Code)
	}
}

func TestEvaluatePolicy_validation(t *testing.T) {
	h, _, _ := testHandler(t, "test-token")
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	payload, _ := json.Marshal(map[string]string{"digest": "sha256:abc", "phase": "push"})
	req := httptest.NewRequest(http.MethodPost, "/v1/namespaces/gh/acme/widget/artifacts/widget/policy/evaluate", bytes.NewReader(payload))
	req.Header.Set("Authorization", "Bearer test-token")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestListArtifacts_byCommit(t *testing.T) {
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
	d, err := store.RegisterDigest(ctx, art.ID, "sha256:commitquery", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	commit := "c0ffee"
	att, err := store.AttachAttestation(ctx, d.ID, "https://slsa.dev/provenance/v1", nil, nil, json.RawMessage(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.InsertProvenanceRecord(ctx, att.ID, d.ID, "https://github.com/acme/widget", &commit, nil, nil, nil, true); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/v1/namespaces/gh/acme/widget/artifacts?commit="+commit, nil)
	req.Header.Set("Authorization", "Bearer test-token")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		Commit  string `json:"commit"`
		Results []struct {
			Name   string `json:"name"`
			Digest string `json:"digest"`
		} `json:"results"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if resp.Commit != commit || len(resp.Results) != 1 {
		t.Fatalf("got %+v", resp)
	}
	if resp.Results[0].Name != "widget" || resp.Results[0].Digest != d.Digest {
		t.Fatalf("results = %+v", resp.Results)
	}
}
