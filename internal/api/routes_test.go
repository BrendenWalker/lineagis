package api

import "testing"

func TestParseNamespacesPath(t *testing.T) {
	t.Parallel()
	ns, art, suffix, ok := parseNamespacesPath("namespaces/gh/acme/widget/artifacts/widget/digests")
	if !ok || ns != "gh/acme/widget" || art != "widget" || suffix != "digests" {
		t.Fatalf("got %q %q %q ok=%v", ns, art, suffix, ok)
	}

	ns, art, suffix, ok = parseNamespacesPath("namespaces/gh/acme/widget/artifacts/widget/tags/v1.0.0")
	if !ok || suffix != "tags/v1.0.0" {
		t.Fatalf("tags: got %q %q %q ok=%v", ns, art, suffix, ok)
	}

	ns, art, suffix, ok = parseNamespacesPath("namespaces/gh/acme/widget/artifacts/widget")
	if !ok || suffix != "" || art != "widget" {
		t.Fatalf("artifact only: got %q %q %q ok=%v", ns, art, suffix, ok)
	}

	ns, art, suffix, ok = parseNamespacesPath("namespaces/gh/acme/widget/artifacts/widget/signatures")
	if !ok || suffix != "signatures" || art != "widget" {
		t.Fatalf("signatures: got %q %q %q ok=%v", ns, art, suffix, ok)
	}

	if _, _, _, ok = parseNamespacesPath("bad"); ok {
		t.Fatal("expected false for bad path")
	}

	ns, ok = parseNamespaceArtifactsListPath("namespaces/gh/acme/widget/artifacts")
	if !ok || ns != "gh/acme/widget" {
		t.Fatalf("list: got %q ok=%v", ns, ok)
	}

	ns, ok = parseNamespacePolicyPath("namespaces/gh/acme/widget/policy")
	if !ok || ns != "gh/acme/widget" {
		t.Fatalf("policy: got %q ok=%v", ns, ok)
	}
}
