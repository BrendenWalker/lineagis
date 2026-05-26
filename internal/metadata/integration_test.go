//go:build integration

package metadata_test

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

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

func testStore(t *testing.T) (*metadata.Store, *pgxpool.Pool) {
	t.Helper()

	url := os.Getenv("VERITY_TEST_DATABASE_URL")
	if url == "" {
		t.Skip("VERITY_TEST_DATABASE_URL not set")
	}
	if testDBErr != nil {
		t.Fatalf("test database migrate: %v", testDBErr)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	t.Cleanup(cancel)

	pool, err := db.OpenPool(ctx, url)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	if err := truncateMetadata(ctx, pool); err != nil {
		t.Fatalf("truncate metadata: %v", err)
	}

	return metadata.NewStore(pool), pool
}

func truncateMetadata(ctx context.Context, pool *pgxpool.Pool) error {
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
			namespaces,
			verity_meta
		RESTART IDENTITY CASCADE
	`)
	return err
}

func setupFixture(t *testing.T) (context.Context, *metadata.Store, *metadata.Namespace, *metadata.Artifact) {
	t.Helper()
	store, _ := testStore(t)
	ctx := context.Background()

	ns, err := store.CreateNamespace(ctx, "gh/acme/widget", nil)
	if err != nil {
		t.Fatalf("create namespace: %v", err)
	}
	art, err := store.RegisterArtifact(ctx, ns.ID, "widget")
	if err != nil {
		t.Fatalf("register artifact: %v", err)
	}
	return ctx, store, ns, art
}

// AC-META-001: identical bytes produce the same digest row.
func TestRegisterDigest_idempotent(t *testing.T) {
	ctx, store, _, art := setupFixture(t)

	d1, err := store.RegisterDigest(ctx, art.ID, "sha256:abc111", nil, nil)
	if err != nil {
		t.Fatalf("first register: %v", err)
	}
	d2, err := store.RegisterDigest(ctx, art.ID, "sha256:abc111", nil, nil)
	if err != nil {
		t.Fatalf("second register: %v", err)
	}
	if d1.ID != d2.ID {
		t.Fatalf("expected same row id %d, got %d", d1.ID, d2.ID)
	}
	if d1.Digest != d2.Digest {
		t.Fatalf("expected same digest string")
	}
}

// AC-META-002: tag moves preserve prior digest; tag_events records history.
func TestSetTag_movePreservesDigest(t *testing.T) {
	ctx, store, _, art := setupFixture(t)

	d1, err := store.RegisterDigest(ctx, art.ID, "sha256:digest1", nil, nil)
	if err != nil {
		t.Fatalf("register d1: %v", err)
	}
	d2, err := store.RegisterDigest(ctx, art.ID, "sha256:digest2", nil, nil)
	if err != nil {
		t.Fatalf("register d2: %v", err)
	}

	tag, err := store.SetTag(ctx, art.ID, "v1.0.0", d1.ID, nil)
	if err != nil {
		t.Fatalf("set tag d1: %v", err)
	}
	tag, err = store.SetTag(ctx, art.ID, "v1.0.0", d2.ID, nil)
	if err != nil {
		t.Fatalf("set tag d2: %v", err)
	}
	if tag.DigestID != d2.ID {
		t.Fatalf("tag should point to d2, got digest_id %d", tag.DigestID)
	}

	stillD1, err := store.GetDigestByString(ctx, "sha256:digest1")
	if err != nil {
		t.Fatalf("get d1 by string: %v", err)
	}
	if stillD1.Digest != "sha256:digest1" {
		t.Fatalf("d1 content changed unexpectedly")
	}

	events, err := store.ListTagEvents(ctx, tag.ID)
	if err != nil {
		t.Fatalf("list tag events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 tag events, got %d", len(events))
	}
	if events[0].FromDigestID != nil {
		t.Fatalf("first event should have nil from_digest_id")
	}
	if events[0].ToDigestID != d1.ID {
		t.Fatalf("first event to_digest_id want %d", d1.ID)
	}
	if events[1].FromDigestID == nil || *events[1].FromDigestID != d1.ID {
		t.Fatalf("second event from_digest_id want %d", d1.ID)
	}
	if events[1].ToDigestID != d2.ID {
		t.Fatalf("second event to_digest_id want %d", d2.ID)
	}
}

// AC-META-003: signatures are scoped to the digest they cover.
func TestListSignatures_scopedToDigest(t *testing.T) {
	ctx, store, _, art := setupFixture(t)

	d1, err := store.RegisterDigest(ctx, art.ID, "sha256:sig1", nil, nil)
	if err != nil {
		t.Fatalf("register d1: %v", err)
	}
	d2, err := store.RegisterDigest(ctx, art.ID, "sha256:sig2", nil, nil)
	if err != nil {
		t.Fatalf("register d2: %v", err)
	}

	ref := "bundles://example"
	_, err = store.AttachSignature(ctx, d1.ID, &ref, nil, nil, nil)
	if err != nil {
		t.Fatalf("attach signature: %v", err)
	}

	sigsD1, err := store.ListSignatures(ctx, d1.ID)
	if err != nil {
		t.Fatalf("list d1 signatures: %v", err)
	}
	if len(sigsD1) != 1 {
		t.Fatalf("expected 1 signature on d1, got %d", len(sigsD1))
	}

	sigsD2, err := store.ListSignatures(ctx, d2.ID)
	if err != nil {
		t.Fatalf("list d2 signatures: %v", err)
	}
	if len(sigsD2) != 0 {
		t.Fatalf("expected 0 signatures on d2, got %d", len(sigsD2))
	}
}

func TestSetTag_rejectsWrongArtifact(t *testing.T) {
	ctx, store, ns, art1 := setupFixture(t)

	art2, err := store.RegisterArtifact(ctx, ns.ID, "other")
	if err != nil {
		t.Fatalf("register art2: %v", err)
	}
	d, err := store.RegisterDigest(ctx, art2.ID, "sha256:other", nil, nil)
	if err != nil {
		t.Fatalf("register digest: %v", err)
	}

	_, err = store.SetTag(ctx, art1.ID, "v1.0.0", d.ID, nil)
	if err != metadata.ErrDigestWrongArtifact {
		t.Fatalf("expected ErrDigestWrongArtifact, got %v", err)
	}
}

func TestPutPolicy_auditLogged(t *testing.T) {
	ctx, store, ns, _ := setupFixture(t)

	actor := "operator-1"
	doc := json.RawMessage(`{"rules":[{"id":"require-signatures"}]}`)
	p, err := store.PutPolicy(ctx, ns.ID, doc, &actor)
	if err != nil {
		t.Fatalf("put policy: %v", err)
	}
	if !p.IsActive || p.Version != 1 {
		t.Fatalf("unexpected policy: version=%d active=%v", p.Version, p.IsActive)
	}

	events, err := store.ListAuditEvents(ctx, ns.ID, 10)
	if err != nil {
		t.Fatalf("list audit events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(events))
	}
	if events[0].EventType != "policy.updated" {
		t.Fatalf("event type: %s", events[0].EventType)
	}
	if events[0].Actor == nil || *events[0].Actor != actor {
		t.Fatalf("unexpected actor: %v", events[0].Actor)
	}
	if events[0].CreatedAt.IsZero() {
		t.Fatal("expected audit event timestamp")
	}
	var payload struct {
		PolicyID int64 `json:"policy_id"`
		Version  int   `json:"version"`
	}
	if err := json.Unmarshal(events[0].Payload, &payload); err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	if payload.PolicyID != p.ID || payload.Version != p.Version {
		t.Fatalf("payload = %+v, want policy_id=%d version=%d", payload, p.ID, p.Version)
	}
}
