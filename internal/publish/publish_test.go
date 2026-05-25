package publish

import (
	"testing"

	"github.com/BrendenWalker/verity/internal/registry"
)

func TestManifestDigest_stableOrder(t *testing.T) {
	t.Parallel()
	layers := []registry.FileLayer{
		{Path: "b.txt", Data: []byte("b")},
		{Path: "a.txt", Data: []byte("a")},
	}
	h1, err := ManifestDigest(layers, "dist/")
	if err != nil {
		t.Fatal(err)
	}
	reordered := []registry.FileLayer{
		{Path: "a.txt", Data: []byte("a")},
		{Path: "b.txt", Data: []byte("b")},
	}
	h2, err := ManifestDigest(reordered, "dist/")
	if err != nil {
		t.Fatal(err)
	}
	if h1 != h2 {
		t.Fatalf("digests differ: %s vs %s", h1, h2)
	}
}

func TestRegistryRepo(t *testing.T) {
	t.Parallel()
	if got := RegistryRepo("gh/acme/widget", "widget"); got != "gh/acme/widget/widget" {
		t.Fatalf("RegistryRepo = %q", got)
	}
}
