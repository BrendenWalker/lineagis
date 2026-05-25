package apiclient_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BrendenWalker/verity/internal/apiclient"
)

func TestAttachSignature_postsBundle(t *testing.T) {
	t.Parallel()
	var got struct {
		Digest  string          `json:"digest"`
		Bundle  json.RawMessage `json:"bundle"`
		Issuer  string          `json:"issuer"`
		Subject string          `json:"subject"`
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/namespaces/ns/artifacts/app/signatures" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer tok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	bundle := json.RawMessage(`{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json"}`)
	issuer := "https://token.actions.githubusercontent.com"
	subject := "repo:acme/app:ref:refs/heads/main"
	c := apiclient.New(srv.URL, "tok")
	if err := c.AttachSignature(context.Background(), "ns", "app", "sha256:abc", bundle, &issuer, &subject); err != nil {
		t.Fatal(err)
	}
	if got.Digest != "sha256:abc" {
		t.Fatalf("digest = %q", got.Digest)
	}
	if len(got.Bundle) == 0 {
		t.Fatal("expected bundle in request")
	}
	if got.Issuer != issuer || got.Subject != subject {
		t.Fatalf("identity: issuer=%q subject=%q", got.Issuer, got.Subject)
	}
}
