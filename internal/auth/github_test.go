package auth_test

import (
	"encoding/json"
	"testing"

	"github.com/BrendenWalker/verity/internal/auth"
)

func TestExpectedRepository(t *testing.T) {
	t.Parallel()
	tests := []struct {
		ns     string
		config string
		want   string
		ok     bool
	}{
		{"gh/acme/widget", "{}", "acme/widget", true},
		{"local/dev", "{}", "", false},
		{"gh/acme/widget", `{"repository":"other/repo"}`, "other/repo", true},
	}
	for _, tc := range tests {
		got, ok := auth.ExpectedRepository(tc.ns, json.RawMessage(tc.config))
		if got != tc.want || ok != tc.ok {
			t.Fatalf("ExpectedRepository(%q) = (%q, %v), want (%q, %v)", tc.ns, got, ok, tc.want, tc.ok)
		}
	}
}

func TestAuthorizeNamespace_github(t *testing.T) {
	t.Parallel()
	actor := auth.Actor{
		Subject: "repo:acme/widget:ref:refs/heads/main",
		GitHub:  &auth.GitHubClaims{Repository: "acme/widget", Ref: "refs/heads/main"},
	}
	if err := auth.AuthorizeNamespace(actor, "gh/acme/widget", nil); err != nil {
		t.Fatalf("expected allow: %v", err)
	}

	actor.GitHub.Repository = "other/repo"
	if err := auth.AuthorizeNamespace(actor, "gh/acme/widget", nil); err == nil {
		t.Fatal("expected forbidden for mismatched repository")
	}
}

func TestAuthorizeNamespace_allowedRefs(t *testing.T) {
	t.Parallel()
	cfg := json.RawMessage(`{"allowed_refs":["refs/heads/main"]}`)
	actor := auth.Actor{
		GitHub: &auth.GitHubClaims{Repository: "acme/widget", Ref: "refs/heads/release"},
	}
	if err := auth.AuthorizeNamespace(actor, "gh/acme/widget", cfg); err == nil {
		t.Fatal("expected ref rejection")
	}
	actor.GitHub.Ref = "refs/heads/main"
	if err := auth.AuthorizeNamespace(actor, "gh/acme/widget", cfg); err != nil {
		t.Fatalf("expected allow: %v", err)
	}
}

func TestAuthorizeNamespace_devSkips(t *testing.T) {
	t.Parallel()
	if err := auth.AuthorizeNamespace(auth.Actor{Dev: true}, "gh/acme/widget", nil); err != nil {
		t.Fatalf("dev actor should skip: %v", err)
	}
}
