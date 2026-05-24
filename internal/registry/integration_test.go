//go:build integration

package registry_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/BrendenWalker/verity/internal/registry"
)

func testRegistryClient(t *testing.T) *registry.Client {
	t.Helper()

	url := os.Getenv("VERITY_TEST_REGISTRY_URL")
	if url == "" {
		t.Skip("VERITY_TEST_REGISTRY_URL not set")
	}

	client, err := registry.New(url)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func testRepo(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("verity-test/blob-%d", time.Now().UnixNano())
}

func TestIntegrationPushPullBlob(t *testing.T) {
	client := testRegistryClient(t)
	repo := testRepo(t)

	data := make([]byte, 4096)
	if _, err := rand.Read(data); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	h, err := client.PushBlob(ctx, repo, data)
	if err != nil {
		t.Fatalf("PushBlob: %v", err)
	}

	got, err := client.PullBlob(ctx, repo, h)
	if err != nil {
		t.Fatalf("PullBlob: %v", err)
	}
	if string(got) != string(data) {
		t.Fatal("pulled bytes differ from pushed bytes")
	}
}

func TestIntegrationPushBlobIdempotent(t *testing.T) {
	client := testRegistryClient(t)
	repo := testRepo(t)

	data := []byte("integration idempotent blob")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	first, err := client.PushBlob(ctx, repo, data)
	if err != nil {
		t.Fatalf("first PushBlob: %v", err)
	}
	second, err := client.PushBlob(ctx, repo, data)
	if err != nil {
		t.Fatalf("second PushBlob: %v", err)
	}
	if first != second {
		t.Fatalf("digests differ: %s vs %s", first, second)
	}

	exists, err := client.BlobExists(ctx, repo, first)
	if err != nil {
		t.Fatalf("BlobExists: %v", err)
	}
	if !exists {
		t.Fatal("BlobExists = false, want true")
	}
}
