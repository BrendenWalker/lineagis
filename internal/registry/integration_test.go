//go:build integration

package registry_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/BrendenWalker/lineagis/internal/registry"
)

func testRegistryClient(t *testing.T) *registry.Client {
	t.Helper()

	url := os.Getenv("LINEAGIS_TEST_REGISTRY_URL")
	if url == "" {
		t.Skip("LINEAGIS_TEST_REGISTRY_URL not set")
	}

	client, err := registry.New(url)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return client
}

func testRepo(t *testing.T) string {
	t.Helper()
	return fmt.Sprintf("lineagis-test/blob-%d", time.Now().UnixNano())
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

func TestIntegrationPushPullManifest(t *testing.T) {
	client := testRegistryClient(t)
	repo := testRepo(t)

	layers := []registry.FileLayer{
		{Path: "SHA256SUMS", Data: []byte("deadbeef")},
		{Path: "release.tar.gz", Data: []byte("gzip-bytes-here")},
		{Path: "package.whl", Data: []byte("zip-bytes-here")},
	}
	manifestData, wantDigest, err := registry.BuildArtifactManifest(layers, registry.ManifestOptions{
		PublishRoot: "dist/",
	})
	if err != nil {
		t.Fatalf("BuildArtifactManifest: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, layer := range layers {
		if _, err := client.PushBlob(ctx, repo, layer.Data); err != nil {
			t.Fatalf("PushBlob %q: %v", layer.Path, err)
		}
	}

	first, err := client.PushManifest(ctx, repo, manifestData)
	if err != nil {
		t.Fatalf("first PushManifest: %v", err)
	}
	second, err := client.PushManifest(ctx, repo, manifestData)
	if err != nil {
		t.Fatalf("second PushManifest: %v", err)
	}
	if first != second || first != wantDigest {
		t.Fatalf("manifest digests differ: %s vs %s (want %s)", first, second, wantDigest)
	}

	pulled, gotDigest, err := client.PullManifest(ctx, repo, wantDigest.String())
	if err != nil {
		t.Fatalf("PullManifest: %v", err)
	}
	if gotDigest != wantDigest {
		t.Fatalf("pulled digest = %s, want %s", gotDigest, wantDigest)
	}
	if string(pulled) != string(manifestData) {
		t.Fatal("pulled manifest bytes differ from pushed manifest")
	}
}
