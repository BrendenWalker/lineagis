package api

import (
	"testing"

	"github.com/BrendenWalker/verity/internal/signing"
)

func TestMatchPublisher(t *testing.T) {
	t.Parallel()
	pub := signing.GitHubPublisher{Repository: "acme/widget", Workflow: "release.yml"}
	allowed := trustedPublisher{Repository: "acme/widget", Workflow: "release.yml"}
	if !matchPublisher(pub, allowed) {
		t.Fatal("expected match")
	}
	if matchPublisher(pub, trustedPublisher{Repository: "other/repo", Workflow: "release.yml"}) {
		t.Fatal("expected repo mismatch")
	}
}

func TestRepositoryFromURI(t *testing.T) {
	t.Parallel()
	if got := repositoryFromURI("https://github.com/acme/widget"); got != "acme/widget" {
		t.Fatalf("got %q", got)
	}
}

func TestRuleMatches_shouldPolicies(t *testing.T) {
	t.Parallel()
	if !ruleTrustedPublishers(policyRule{Type: "trusted-publishers"}) {
		t.Fatal("trusted-publishers")
	}
	if !ruleRepositoryOwnership(policyRule{ID: "repository-ownership"}) {
		t.Fatal("repository-ownership")
	}
}
