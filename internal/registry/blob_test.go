package registry_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1"

	"github.com/BrendenWalker/verity/internal/registry"
)

func TestPushBlobUploadsAndReturnsDigest(t *testing.T) {
	t.Parallel()

	data := []byte("hello blob")
	wantDigest := sha256Hex(data)
	repo := "test/push"

	srv, store := newBlobRegistry(t, repo)
	defer srv.Close()

	client, err := registry.New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got, err := client.PushBlob(context.Background(), repo, data)
	if err != nil {
		t.Fatalf("PushBlob: %v", err)
	}
	if got.String() != wantDigest {
		t.Fatalf("digest = %q, want %q", got, wantDigest)
	}
	if store.blobCount() != 1 {
		t.Fatalf("blob count = %d, want 1", store.blobCount())
	}
}

func TestPushBlobIdempotentWhenBlobExists(t *testing.T) {
	t.Parallel()

	data := []byte("repeat me")
	repo := "test/idempotent"

	srv, store := newBlobRegistry(t, repo)
	defer srv.Close()

	client, err := registry.New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	first, err := client.PushBlob(context.Background(), repo, data)
	if err != nil {
		t.Fatalf("first PushBlob: %v", err)
	}
	second, err := client.PushBlob(context.Background(), repo, data)
	if err != nil {
		t.Fatalf("second PushBlob: %v", err)
	}
	if first != second {
		t.Fatalf("digests differ: %s vs %s", first, second)
	}
	if store.blobCount() != 1 {
		t.Fatalf("blob count = %d, want 1", store.blobCount())
	}
}

func TestPushBlobRejectsOversizedContent(t *testing.T) {
	t.Parallel()

	srv, _ := newBlobRegistry(t, "test/large")
	defer srv.Close()

	client, err := registry.New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	data := make([]byte, registry.MaxBlobSize+1)
	_, err = client.PushBlob(context.Background(), "test/large", data)
	if err == nil {
		t.Fatal("expected error for oversized blob")
	}
	if !strings.Contains(err.Error(), registry.ErrBlobTooLarge.Error()) {
		t.Fatalf("error = %v, want ErrBlobTooLarge", err)
	}
}

func TestPullBlobReturnsUploadedBytes(t *testing.T) {
	t.Parallel()

	data := []byte("pull me back")
	repo := "test/pull"

	srv, _ := newBlobRegistry(t, repo)
	defer srv.Close()

	client, err := registry.New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	h, err := client.PushBlob(context.Background(), repo, data)
	if err != nil {
		t.Fatalf("PushBlob: %v", err)
	}

	got, err := client.PullBlob(context.Background(), repo, h)
	if err != nil {
		t.Fatalf("PullBlob: %v", err)
	}
	if string(got) != string(data) {
		t.Fatalf("PullBlob = %q, want %q", got, data)
	}
}

func TestPullBlobNotFound(t *testing.T) {
	t.Parallel()

	repo := "test/missing"
	srv, _ := newBlobRegistry(t, repo)
	defer srv.Close()

	client, err := registry.New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	h, _, err := v1.SHA256(strings.NewReader("never uploaded"))
	if err != nil {
		t.Fatalf("SHA256: %v", err)
	}

	_, err = client.PullBlob(context.Background(), repo, h)
	if err != registry.ErrNotFound {
		t.Fatalf("PullBlob error = %v, want ErrNotFound", err)
	}
}

func TestBlobExists(t *testing.T) {
	t.Parallel()

	data := []byte("exists check")
	repo := "test/exists"

	srv, _ := newBlobRegistry(t, repo)
	defer srv.Close()

	client, err := registry.New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	h, err := client.PushBlob(context.Background(), repo, data)
	if err != nil {
		t.Fatalf("PushBlob: %v", err)
	}

	ok, err := client.BlobExists(context.Background(), repo, h)
	if err != nil {
		t.Fatalf("BlobExists: %v", err)
	}
	if !ok {
		t.Fatal("BlobExists = false, want true")
	}

	missing, _, err := v1.SHA256(strings.NewReader("missing"))
	if err != nil {
		t.Fatalf("SHA256: %v", err)
	}
	ok, err = client.BlobExists(context.Background(), repo, missing)
	if err != nil {
		t.Fatalf("BlobExists missing: %v", err)
	}
	if ok {
		t.Fatal("BlobExists = true, want false")
	}
}

type blobStore struct {
	mu    sync.Mutex
	blobs map[string][]byte
}

func (s *blobStore) blobCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.blobs)
}

func (s *blobStore) put(digest string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blobs[digest] = append([]byte(nil), data...)
}

func (s *blobStore) get(digest string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.blobs[digest]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), data...), true
}

func newBlobRegistry(t *testing.T, repo string) (*httptest.Server, *blobStore) {
	t.Helper()

	store := &blobStore{blobs: make(map[string][]byte)}
	prefix := "/v2/" + repo + "/blobs/"
	uploadPrefix := prefix + "uploads/"
	var uploadSeq int

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, prefix) {
			w.WriteHeader(http.StatusOK)
			return
		}

		if r.URL.Path == uploadPrefix && r.Method == http.MethodPost {
			uploadSeq++
			w.Header().Set("Location", fmt.Sprintf("%s%d", uploadPrefix, uploadSeq))
			w.WriteHeader(http.StatusAccepted)
			return
		}

		if strings.HasPrefix(r.URL.Path, uploadPrefix) && r.Method == http.MethodPut {
			digest := r.URL.Query().Get("digest")
			if digest == "" {
				http.Error(w, "missing digest", http.StatusBadRequest)
				return
			}
			data, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			store.put(digest, data)
			w.WriteHeader(http.StatusCreated)
			return
		}

		if !strings.HasPrefix(r.URL.Path, prefix) || strings.HasPrefix(r.URL.Path, uploadPrefix) {
			http.NotFound(w, r)
			return
		}

		digest := strings.TrimPrefix(r.URL.Path, prefix)
		switch r.Method {
		case http.MethodHead:
			if _, ok := store.get(digest); ok {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.NotFound(w, r)
		case http.MethodGet:
			data, ok := store.get(digest)
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(data)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	return httptest.NewServer(mux), store
}

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
