package signing

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestCertIdentityRegexp_publisher(t *testing.T) {
	t.Parallel()
	got := certIdentityRegexp(PublisherMatcher{
		Repository: "acme/widget",
		Workflow:   "release.yml",
		Ref:        "refs/heads/main",
	})
	want := `https://github\.com/acme/widget/\.github/workflows/release\.yml@refs/heads/main`
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestCertIdentityRegexp_refWildcard(t *testing.T) {
	t.Parallel()
	got := certIdentityRegexp(PublisherMatcher{
		Repository: "acme/widget",
		Workflow:   "release.yml",
		Ref:        "refs/tags/*",
	})
	if got != `https://github\.com/acme/widget/\.github/workflows/release\.yml@refs/tags/.*` {
		t.Fatalf("got %q", got)
	}
}

func TestKeylessVerifyOptions_trustedPublishers(t *testing.T) {
	t.Setenv("VERITY_PERMISSIVE_KEYLESS_IDENTITY", "")
	doc := json.RawMessage(`{"rules":[{"id":"trusted-publishers","config":{"publishers":[{"repository":"acme/widget","workflow":"release.yml"}]}}]}`)
	opts := KeylessVerifyOptions(doc)
	if opts.CertOidcIssuer != githubActionsIssuer {
		t.Fatalf("issuer = %q", opts.CertOidcIssuer)
	}
	if opts.CertIdentity == "" || opts.CertIdentity == ".*" {
		t.Fatalf("identity = %q", opts.CertIdentity)
	}
	if !strings.Contains(opts.CertIdentity, `acme/widget`) {
		t.Fatalf("expected repo in identity %q", opts.CertIdentity)
	}
}
