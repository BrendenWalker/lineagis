package api

import (
	"testing"

	"github.com/BrendenWalker/verity/internal/signing"
)

func TestMatchRefPattern(t *testing.T) {
	t.Parallel()
	cases := []struct {
		actual, pattern string
		want            bool
	}{
		{"refs/tags/v1.0.0", "refs/tags/*", true},
		{"refs/heads/main", "refs/tags/*", false},
		{"refs/heads/main", "refs/heads/main", true},
		{"refs/heads/feature/x", "refs/heads/feature/*", true},
	}
	for _, tc := range cases {
		if got := matchRefPattern(tc.actual, tc.pattern); got != tc.want {
			t.Fatalf("matchRefPattern(%q, %q) = %v, want %v", tc.actual, tc.pattern, got, tc.want)
		}
	}
}

func TestMatchPublisher_granular(t *testing.T) {
	t.Parallel()
	pub := signing.GitHubPublisher{
		Repository:  "acme/widget",
		Workflow:    "release.yml",
		Ref:         "refs/tags/v1.0.0",
		Environment: "push",
	}
	id := signing.Identity{Issuer: "https://token.actions.githubusercontent.com"}
	allowed := trustedPublisher{
		Repository: "acme/widget",
		Workflow:   "release.yml",
		Ref:        "refs/tags/*",
		Issuer:     "https://token.actions.githubusercontent.com",
	}
	if !matchPublisher(pub, allowed, id) {
		t.Fatal("expected match")
	}
	allowed.Ref = "refs/heads/*"
	if matchPublisher(pub, allowed, id) {
		t.Fatal("expected ref mismatch")
	}
}
