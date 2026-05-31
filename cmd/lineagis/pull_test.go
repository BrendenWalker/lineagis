package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/go-containerregistry/pkg/v1"

	"github.com/BrendenWalker/lineagis/internal/registry"
)

func TestRunPull_verifyFailsWithoutSignatures(t *testing.T) {
	layers := []registry.FileLayer{{Path: "out.txt", Data: []byte("data")}}
	manifestJSON, manifestHash, err := registry.BuildArtifactManifest(layers, registry.ManifestOptions{})
	if err != nil {
		t.Fatal(err)
	}

	repo := "gh/acme/widget/widget"
	regSrv := startMiniRegistry(t, repo, manifestHash.String(), manifestJSON, layers)
	defer regSrv.Close()

	apiSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/trust"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"digest":     manifestHash.String(),
				"signatures": map[string]string{"status": "missing"},
			})
		case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/signatures"):
			_ = json.NewEncoder(w).Encode(map[string]any{"signatures": []any{}})
		default:
			http.NotFound(w, r)
		}
	}))
	defer apiSrv.Close()

	t.Setenv("LINEAGIS_TOKEN", "tok")
	t.Setenv("LINEAGIS_API_URL", apiSrv.URL)
	t.Setenv("LINEAGIS_REGISTRY_URL", regSrv.URL)

	if got := run([]string{"pull", "gh/acme/widget/widget@" + manifestHash.String(), "--verify"}); got != 1 {
		t.Fatalf("exit = %d, want 1", got)
	}
}

func startMiniRegistry(t *testing.T, repo, digest string, manifestJSON []byte, layers []registry.FileLayer) *httptest.Server {
	t.Helper()
	blobs := make(map[string][]byte)
	for _, layer := range layers {
		h, _, err := v1.SHA256(bytes.NewReader(layer.Data))
		if err != nil {
			t.Fatal(err)
		}
		blobs[h.String()] = layer.Data
	}
	prefix := "/v2/" + repo + "/"
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, prefix+"manifests/") && r.Method == http.MethodGet {
			_, _ = w.Write(manifestJSON)
			return
		}
		if strings.HasPrefix(r.URL.Path, prefix+"blobs/") && r.Method == http.MethodGet {
			d := strings.TrimPrefix(r.URL.Path, prefix+"blobs/")
			if data, ok := blobs[d]; ok {
				_, _ = w.Write(data)
				return
			}
		}
		http.NotFound(w, r)
	}))
}
