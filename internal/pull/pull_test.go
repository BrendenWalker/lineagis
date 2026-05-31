package pull_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1"

	"github.com/BrendenWalker/lineagis/internal/pull"
	"github.com/BrendenWalker/lineagis/internal/registry"
)

func TestPull_byDigest_writesLayers(t *testing.T) {
	t.Parallel()

	const wantContent = "hello-pull"
	layers := []registry.FileLayer{{Path: "bin/app.txt", Data: []byte(wantContent)}}
	manifestJSON, manifestHash, err := registry.BuildArtifactManifest(layers, registry.ManifestOptions{})
	if err != nil {
		t.Fatal(err)
	}

	regSrv, regURL := newTestRegistry(t, "gh/acme/widget/widget", manifestHash.String(), manifestJSON, layers)
	defer regSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer apiSrv.Close()

	dir := t.TempDir()
	digest, err := pull.Pull(context.Background(), pull.Options{
		Ref:         "gh/acme/widget/widget@" + manifestHash.String(),
		OutputDir:   dir,
		APIURL:      apiSrv.URL,
		RegistryURL: regURL,
		Token:       "tok",
	})
	if err != nil {
		t.Fatal(err)
	}
	if digest != manifestHash.String() {
		t.Fatalf("digest = %q, want %s", digest, manifestHash.String())
	}
	data, err := os.ReadFile(filepath.Join(dir, "bin", "app.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != wantContent {
		t.Fatalf("content = %q", data)
	}
}

func TestPull_resolveTag_thenPull(t *testing.T) {
	t.Parallel()

	const wantContent = "tagged"
	layers := []registry.FileLayer{{Path: "release.txt", Data: []byte(wantContent)}}
	manifestJSON, manifestHash, err := registry.BuildArtifactManifest(layers, registry.ManifestOptions{})
	if err != nil {
		t.Fatal(err)
	}

	regSrv, regURL := newTestRegistry(t, "gh/acme/widget/widget", manifestHash.String(), manifestJSON, layers)
	defer regSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/v1/namespaces/gh/acme/widget/artifacts/widget/tags/v1.0.0" {
			_ = json.NewEncoder(w).Encode(map[string]string{
				"namespace": "gh/acme/widget",
				"artifact":  "widget",
				"tag":       "v1.0.0",
				"digest":    manifestHash.String(),
			})
			return
		}
		http.NotFound(w, r)
	}))
	defer apiSrv.Close()

	dir := t.TempDir()
	digest, err := pull.Pull(context.Background(), pull.Options{
		Ref:         "gh/acme/widget/widget:v1.0.0",
		OutputDir:   dir,
		APIURL:      apiSrv.URL,
		RegistryURL: regURL,
		Token:       "tok",
	})
	if err != nil {
		t.Fatal(err)
	}
	if digest != manifestHash.String() {
		t.Fatalf("digest = %q", digest)
	}
	data, err := os.ReadFile(filepath.Join(dir, "release.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != wantContent {
		t.Fatalf("content = %q", data)
	}
}

func TestPull_withVerify_failsWithoutSignatures(t *testing.T) {
	t.Parallel()

	layers := []registry.FileLayer{{Path: "a.txt", Data: []byte("x")}}
	manifestJSON, manifestHash, err := registry.BuildArtifactManifest(layers, registry.ManifestOptions{})
	if err != nil {
		t.Fatal(err)
	}

	regSrv, regURL := newTestRegistry(t, "gh/acme/widget/widget", manifestHash.String(), manifestJSON, layers)
	defer regSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/trust"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"namespace":  "gh/acme/widget",
				"artifact":   "widget",
				"digest":     manifestHash.String(),
				"signatures": map[string]string{"status": "missing"},
				"overall":    "fail",
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/signatures"):
			_ = json.NewEncoder(w).Encode(map[string]any{"signatures": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer apiSrv.Close()

	_, err = pull.Pull(context.Background(), pull.Options{
		Ref:         "gh/acme/widget/widget@" + manifestHash.String(),
		Verify:      true,
		APIURL:      apiSrv.URL,
		RegistryURL: regURL,
		Token:       "tok",
	})
	if err == nil {
		t.Fatal("expected verify failure")
	}
	if !strings.Contains(err.Error(), "trust verification failed") {
		t.Fatalf("err = %v", err)
	}
}

type testRegistryStore struct {
	mu        sync.Mutex
	blobs     map[string][]byte
	manifests map[string][]byte
}

func newTestRegistry(t *testing.T, repo, digest string, manifestJSON []byte, layers []registry.FileLayer) (*httptest.Server, string) {
	t.Helper()
	store := &testRegistryStore{
		blobs:     make(map[string][]byte),
		manifests: make(map[string][]byte),
	}
	store.manifests[digest] = append([]byte(nil), manifestJSON...)
	for _, layer := range layers {
		h, _, err := v1.SHA256(bytes.NewReader(layer.Data))
		if err != nil {
			t.Fatal(err)
		}
		store.blobs[h.String()] = append([]byte(nil), layer.Data...)
	}

	prefix := "/v2/" + repo + "/"
	mux := http.NewServeMux()
	mux.HandleFunc("/v2/", func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, prefix) {
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.HasPrefix(r.URL.Path, prefix+"manifests/") {
			ref := strings.TrimPrefix(r.URL.Path, prefix+"manifests/")
			if r.Method != http.MethodGet {
				http.Error(w, "method", http.StatusMethodNotAllowed)
				return
			}
			store.mu.Lock()
			data, ok := store.manifests[ref]
			store.mu.Unlock()
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", registry.ArtifactManifestMediaType)
			_, _ = w.Write(data)
			return
		}
		if strings.HasPrefix(r.URL.Path, prefix+"blobs/") {
			d := strings.TrimPrefix(r.URL.Path, prefix+"blobs/")
			if r.Method != http.MethodGet {
				http.Error(w, "method", http.StatusMethodNotAllowed)
				return
			}
			store.mu.Lock()
			data, ok := store.blobs[d]
			store.mu.Unlock()
			if !ok {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(data)
			return
		}
		http.NotFound(w, r)
	})
	srv := httptest.NewServer(mux)
	return srv, srv.URL
}
