package apiclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BrendenWalker/verity/internal/apiclient"
)

func TestGetTrustStatus_byDigest(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/v1/namespaces/gh/acme/app/artifacts/widget/trust" {
			http.NotFound(w, r)
			return
		}
		if r.URL.Query().Get("digest") != "sha256:abc" {
			http.Error(w, "digest", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"namespace":  "gh/acme/app",
			"artifact":   "widget",
			"digest":     "sha256:abc",
			"overall":    "pass",
			"signatures": map[string]string{"status": "valid"},
		})
	}))
	defer srv.Close()

	out, err := apiclient.New(srv.URL, "tok").GetTrustStatus(context.Background(), "gh/acme/app", "widget", "sha256:abc", "")
	if err != nil {
		t.Fatal(err)
	}
	if out.Signatures.Status != "valid" || out.Overall != "pass" {
		t.Fatalf("got %+v", out)
	}
}
