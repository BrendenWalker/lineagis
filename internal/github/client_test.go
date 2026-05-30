package github_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/BrendenWalker/verity/internal/github"
)

func TestRepositoryExists(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/acme/widget" {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	c := github.NewClient("tok")
	c.SetBaseURL(srv.URL) // need to add SetBaseURL for tests
	exists, err := c.RepositoryExists(context.Background(), "acme/widget")
	if err != nil || !exists {
		t.Fatalf("exists=%v err=%v", exists, err)
	}
}
