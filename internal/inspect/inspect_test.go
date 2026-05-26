package inspect_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/BrendenWalker/verity/internal/apiclient"
	"github.com/BrendenWalker/verity/internal/inspect"
)

func TestRun_digestValid(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/namespaces/ns/artifacts/app/trust" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("digest") != "sha256:abc" {
			http.Error(w, "bad digest", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"signatures": map[string]string{"status": "valid"},
			"overall":    "pass",
		})
	}))
	defer srv.Close()

	result, err := inspect.Run(context.Background(), apiclient.New(srv.URL, "tok"), inspect.Options{
		Namespace: "ns",
		Artifact:  "app",
		Ref:       "sha256:abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !result.SignatureOK || result.SignatureLine != "✓ Signed by GitHub Actions" {
		t.Fatalf("got line=%q ok=%v", result.SignatureLine, result.SignatureOK)
	}
}

func TestRun_digestInvalid(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"signatures": map[string]string{"status": "invalid"},
		})
	}))
	defer srv.Close()

	result, err := inspect.Run(context.Background(), apiclient.New(srv.URL, "tok"), inspect.Options{
		Namespace: "ns",
		Artifact:  "app",
		Ref:       "sha256:dead",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SignatureOK || result.SignatureLine != "✗ Signature invalid" {
		t.Fatalf("got line=%q ok=%v", result.SignatureLine, result.SignatureOK)
	}
}

func TestRun_tagMissing(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("tag") != "1.0.0" {
			http.Error(w, "bad tag", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"signatures": map[string]string{"status": "missing"},
		})
	}))
	defer srv.Close()

	result, err := inspect.Run(context.Background(), apiclient.New(srv.URL, "tok"), inspect.Options{
		Namespace: "ns",
		Artifact:  "app",
		Ref:       "1.0.0",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.SignatureOK || result.SignatureLine != "✗ Signature missing" {
		t.Fatalf("got line=%q ok=%v", result.SignatureLine, result.SignatureOK)
	}
}

func TestRun_localPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/f.txt", []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	var gotDigest string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotDigest = r.URL.Query().Get("digest")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"signatures": map[string]string{"status": "valid"},
			"digest":     gotDigest,
		})
	}))
	defer srv.Close()

	result, err := inspect.Run(context.Background(), apiclient.New(srv.URL, "tok"), inspect.Options{
		Namespace: "ns",
		Artifact:  "app",
		Ref:       dir,
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotDigest == "" || !strings.HasPrefix(gotDigest, "sha256:") {
		t.Fatalf("digest = %q", gotDigest)
	}
	if !result.SignatureOK {
		t.Fatalf("line=%q", result.SignatureLine)
	}
}
