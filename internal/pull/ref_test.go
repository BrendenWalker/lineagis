package pull_test

import (
	"testing"

	"github.com/BrendenWalker/verity/internal/pull"
)

func TestParseRef_digest(t *testing.T) {
	r, err := pull.ParseRef("gh/acme/widget/app@sha256:abc")
	if err != nil {
		t.Fatal(err)
	}
	if r.Namespace != "gh/acme/widget" || r.Artifact != "app" || r.Digest != "sha256:abc" {
		t.Fatalf("%+v", r)
	}
}

func TestParseRef_tag(t *testing.T) {
	r, err := pull.ParseRef("gh/acme/widget/app:v1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if r.Tag != "v1.0.0" {
		t.Fatalf("%+v", r)
	}
}
