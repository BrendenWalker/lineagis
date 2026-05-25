package registry_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/BrendenWalker/verity/internal/registry"
)

func TestBuildArtifactManifestDigestStability(t *testing.T) {
	t.Parallel()

	layers := []registry.FileLayer{
		{Path: "widget-1.2.0.tar.gz", Data: []byte("tarball bytes")},
		{Path: "SHA256SUMS", Data: []byte("checksums")},
		{Path: "widget-1.2.0-py3-none-any.whl", Data: []byte("wheel bytes")},
	}
	opts := registry.ManifestOptions{PublishRoot: "dist/"}

	first, firstDigest, err := registry.BuildArtifactManifest(layers, opts)
	if err != nil {
		t.Fatalf("first BuildArtifactManifest: %v", err)
	}

	shuffled := []registry.FileLayer{layers[2], layers[0], layers[1]}
	second, secondDigest, err := registry.BuildArtifactManifest(shuffled, opts)
	if err != nil {
		t.Fatalf("second BuildArtifactManifest: %v", err)
	}

	if firstDigest != secondDigest {
		t.Fatalf("digests differ: %s vs %s", firstDigest, secondDigest)
	}
	if string(first) != string(second) {
		t.Fatal("manifest bytes differ for identical content in different order")
	}

	var parsed struct {
		Layers []struct {
			Annotations map[string]string `json:"annotations"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(first, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(parsed.Layers) != 3 {
		t.Fatalf("layer count = %d, want 3", len(parsed.Layers))
	}
	wantOrder := []string{"SHA256SUMS", "widget-1.2.0-py3-none-any.whl", "widget-1.2.0.tar.gz"}
	for i, want := range wantOrder {
		got := parsed.Layers[i].Annotations["dev.verity.path"]
		if got != want {
			t.Fatalf("layer[%d].dev.verity.path = %q, want %q", i, got, want)
		}
	}
}

func TestBuildArtifactManifestLayerMediaTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		path string
		want string
	}{
		{"pkg.whl", "application/zip"},
		{"pkg.tar.gz", "application/vnd.oci.image.layer.v1.tar+gzip"},
		{"pkg.tgz", "application/vnd.oci.image.layer.v1.tar+gzip"},
		{"pkg.tar", "application/vnd.oci.image.layer.v1.tar"},
		{"pkg.zip", "application/zip"},
		{"meta.json", "application/json"},
		{"SHA256SUMS", "application/octet-stream"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			t.Parallel()
			data, _, err := registry.BuildArtifactManifest([]registry.FileLayer{
				{Path: tc.path, Data: []byte("x")},
			}, registry.ManifestOptions{})
			if err != nil {
				t.Fatalf("BuildArtifactManifest: %v", err)
			}

			var parsed struct {
				MediaType string `json:"mediaType"`
				Layers    []struct {
					MediaType string `json:"mediaType"`
				} `json:"layers"`
			}
			if err := json.Unmarshal(data, &parsed); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if parsed.MediaType != registry.ArtifactManifestMediaType {
				t.Fatalf("manifest mediaType = %q, want %q", parsed.MediaType, registry.ArtifactManifestMediaType)
			}
			if parsed.Layers[0].MediaType != tc.want {
				t.Fatalf("layer mediaType = %q, want %q", parsed.Layers[0].MediaType, tc.want)
			}
		})
	}
}

func TestBuildArtifactManifestEmptyConfigDescriptor(t *testing.T) {
	t.Parallel()

	data, _, err := registry.BuildArtifactManifest([]registry.FileLayer{
		{Path: "file.bin", Data: []byte("data")},
	}, registry.ManifestOptions{})
	if err != nil {
		t.Fatalf("BuildArtifactManifest: %v", err)
	}

	var parsed struct {
		ArtifactType string `json:"artifactType"`
		Config       struct {
			MediaType string `json:"mediaType"`
			Digest    string `json:"digest"`
			Size      int64  `json:"size"`
		} `json:"config"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if parsed.ArtifactType != registry.VerityReleaseArtifactType {
		t.Fatalf("artifactType = %q, want %q", parsed.ArtifactType, registry.VerityReleaseArtifactType)
	}
	if parsed.Config.MediaType != "application/vnd.oci.empty.v1+json" {
		t.Fatalf("config mediaType = %q, want empty config", parsed.Config.MediaType)
	}
	if parsed.Config.Digest != "sha256:44136fa355b3678a1146ad16f7e8649e94fb4fc21fe77e8310c060f61caaff8a" {
		t.Fatalf("config digest = %q, want OCI empty JSON digest", parsed.Config.Digest)
	}
	if parsed.Config.Size != 2 {
		t.Fatalf("config size = %d, want 2", parsed.Config.Size)
	}
}

func TestPushManifestUploadsAndReturnsDigest(t *testing.T) {
	t.Parallel()

	repo := "test/manifest-push"
	data, wantDigest, err := registry.BuildArtifactManifest([]registry.FileLayer{
		{Path: "a.txt", Data: []byte("alpha")},
	}, registry.ManifestOptions{})
	if err != nil {
		t.Fatalf("BuildArtifactManifest: %v", err)
	}

	srv, store := newRegistry(t, repo)
	defer srv.Close()

	client, err := registry.New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	got, err := client.PushManifest(context.Background(), repo, data)
	if err != nil {
		t.Fatalf("PushManifest: %v", err)
	}
	if got != wantDigest {
		t.Fatalf("digest = %s, want %s", got, wantDigest)
	}
	if store.manifestCount() != 1 {
		t.Fatalf("manifest count = %d, want 1", store.manifestCount())
	}
}

func TestPushManifestIdempotentWhenManifestExists(t *testing.T) {
	t.Parallel()

	repo := "test/manifest-idempotent"
	data, wantDigest, err := registry.BuildArtifactManifest([]registry.FileLayer{
		{Path: "repeat.txt", Data: []byte("same content")},
	}, registry.ManifestOptions{PublishRoot: "dist/"})
	if err != nil {
		t.Fatalf("BuildArtifactManifest: %v", err)
	}

	srv, store := newRegistry(t, repo)
	defer srv.Close()

	client, err := registry.New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	first, err := client.PushManifest(context.Background(), repo, data)
	if err != nil {
		t.Fatalf("first PushManifest: %v", err)
	}
	second, err := client.PushManifest(context.Background(), repo, data)
	if err != nil {
		t.Fatalf("second PushManifest: %v", err)
	}
	if first != second || first != wantDigest {
		t.Fatalf("digests differ: first=%s second=%s want=%s", first, second, wantDigest)
	}
	if store.manifestCount() != 1 {
		t.Fatalf("manifest count = %d, want 1", store.manifestCount())
	}
}

func TestPullManifestReturnsUploadedBytes(t *testing.T) {
	t.Parallel()

	repo := "test/manifest-pull"
	data, digest, err := registry.BuildArtifactManifest([]registry.FileLayer{
		{Path: "pull.txt", Data: []byte("pull me")},
	}, registry.ManifestOptions{})
	if err != nil {
		t.Fatalf("BuildArtifactManifest: %v", err)
	}

	srv, _ := newRegistry(t, repo)
	defer srv.Close()

	client, err := registry.New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := client.PushManifest(context.Background(), repo, data); err != nil {
		t.Fatalf("PushManifest: %v", err)
	}

	got, gotDigest, err := client.PullManifest(context.Background(), repo, digest.String())
	if err != nil {
		t.Fatalf("PullManifest: %v", err)
	}
	if gotDigest != digest {
		t.Fatalf("digest = %s, want %s", gotDigest, digest)
	}
	if string(got) != string(data) {
		t.Fatalf("PullManifest = %s, want %s", got, data)
	}
}

func TestPullManifestNotFound(t *testing.T) {
	t.Parallel()

	repo := "test/manifest-missing"
	srv, _ := newRegistry(t, repo)
	defer srv.Close()

	client, err := registry.New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, _, err = client.PullManifest(context.Background(), repo, "sha256:0000000000000000000000000000000000000000000000000000000000000000")
	if err != registry.ErrNotFound {
		t.Fatalf("PullManifest error = %v, want ErrNotFound", err)
	}
}

func TestPushPullMultiLayerManifest(t *testing.T) {
	t.Parallel()

	created := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	layers := []registry.FileLayer{
		{Path: "SHA256SUMS", Data: []byte("abc123"), Created: &created},
		{Path: "widget-1.0.tar.gz", Data: []byte("gzip-data")},
		{Path: "widget-1.0.whl", Data: []byte("zip-data")},
	}
	opts := registry.ManifestOptions{PublishRoot: "dist/*"}

	manifestData, manifestDigest, err := registry.BuildArtifactManifest(layers, opts)
	if err != nil {
		t.Fatalf("BuildArtifactManifest: %v", err)
	}

	repo := "test/multi-layer"
	srv, _ := newRegistry(t, repo)
	defer srv.Close()

	client, err := registry.New(srv.URL)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	for _, layer := range layers {
		if _, err := client.PushBlob(ctx, repo, layer.Data); err != nil {
			t.Fatalf("PushBlob %q: %v", layer.Path, err)
		}
	}

	firstPush, err := client.PushManifest(ctx, repo, manifestData)
	if err != nil {
		t.Fatalf("first PushManifest: %v", err)
	}
	secondPush, err := client.PushManifest(ctx, repo, manifestData)
	if err != nil {
		t.Fatalf("second PushManifest: %v", err)
	}
	if firstPush != secondPush || firstPush != manifestDigest {
		t.Fatalf("manifest digests differ: %s vs %s (want %s)", firstPush, secondPush, manifestDigest)
	}

	pulled, pulledDigest, err := client.PullManifest(ctx, repo, manifestDigest.String())
	if err != nil {
		t.Fatalf("PullManifest: %v", err)
	}
	if pulledDigest != manifestDigest {
		t.Fatalf("pulled digest = %s, want %s", pulledDigest, manifestDigest)
	}
	if string(pulled) != string(manifestData) {
		t.Fatal("pulled manifest bytes differ")
	}
}

type registryStore struct {
	mu        sync.Mutex
	blobs     map[string][]byte
	manifests map[string][]byte
}

func (s *registryStore) manifestCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.manifests)
}

func (s *registryStore) putBlob(digest string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.blobs[digest] = append([]byte(nil), data...)
}

func (s *registryStore) getBlob(digest string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.blobs[digest]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), data...), true
}

func (s *registryStore) putManifest(reference string, data []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.manifests[reference] = append([]byte(nil), data...)
}

func (s *registryStore) getManifest(reference string) ([]byte, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, ok := s.manifests[reference]
	if !ok {
		return nil, false
	}
	return append([]byte(nil), data...), true
}

func newRegistry(t *testing.T, repo string) (*httptest.Server, *registryStore) {
	t.Helper()

	store := &registryStore{
		blobs:     make(map[string][]byte),
		manifests: make(map[string][]byte),
	}
	prefix := "/v2/" + repo + "/"
	blobPrefix := prefix + "blobs/"
	uploadPrefix := blobPrefix + "uploads/"
	manifestPrefix := prefix + "manifests/"
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
			store.putBlob(digest, data)
			w.WriteHeader(http.StatusCreated)
			return
		}

		if strings.HasPrefix(r.URL.Path, manifestPrefix) {
			reference := strings.TrimPrefix(r.URL.Path, manifestPrefix)
			switch r.Method {
			case http.MethodPut:
				data, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				store.putManifest(reference, data)
				w.Header().Set("Docker-Content-Digest", reference)
				w.WriteHeader(http.StatusCreated)
			case http.MethodGet:
				data, ok := store.getManifest(reference)
				if !ok {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", registry.ArtifactManifestMediaType)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
			case http.MethodHead:
				if _, ok := store.getManifest(reference); ok {
					w.WriteHeader(http.StatusOK)
					return
				}
				http.NotFound(w, r)
			default:
				http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			}
			return
		}

		if !strings.HasPrefix(r.URL.Path, blobPrefix) || strings.HasPrefix(r.URL.Path, uploadPrefix) {
			http.NotFound(w, r)
			return
		}

		digest := strings.TrimPrefix(r.URL.Path, blobPrefix)
		switch r.Method {
		case http.MethodHead:
			if _, ok := store.getBlob(digest); ok {
				w.WriteHeader(http.StatusOK)
				return
			}
			http.NotFound(w, r)
		case http.MethodGet:
			data, ok := store.getBlob(digest)
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
