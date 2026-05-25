package api

import (
	"context"
	"testing"

	"github.com/BrendenWalker/verity/internal/metadata"
	"github.com/BrendenWalker/verity/internal/registry"
	"github.com/BrendenWalker/verity/internal/signing"
)

func TestEvaluateSignatures_validInvalidMissing(t *testing.T) {
	layers := []registry.FileLayer{{Path: "bin/app", Data: []byte("payload")}}
	manifestJSON, digest, err := registry.BuildArtifactManifest(layers, registry.ManifestOptions{})
	if err != nil {
		t.Fatal(err)
	}
	digestStr := digest.String()

	bundle, _, err := signing.SignManifestForTest(manifestJSON)
	if err != nil {
		t.Fatal(err)
	}

	d := &metadata.Digest{Digest: digestStr}
	h := &Handler{Manifests: NewStaticManifestSource(map[string][]byte{digestStr: manifestJSON})}

	status, err := h.evaluateSignatures(context.Background(), "gh/acme/widget", "widget", d, nil)
	if err != nil || status != "missing" {
		t.Fatalf("missing: status=%q err=%v", status, err)
	}

	pubPEM := signing.LegacyBundleCertPEM(bundle)
	if len(pubPEM) == 0 {
		t.Fatalf("expected cert in bundle, got %s", bundle)
	}
	cfg := signing.LoadConfig()
	if err := signing.VerifyManifestBundle(context.Background(), cfg, manifestJSON, bundle, signing.VerifyOptions{
		PublicKeyPEM: pubPEM,
		IgnoreTlog:   true,
		IgnoreSCT:    true,
	}); err != nil {
		t.Fatalf("verify: %v", err)
	}

	status, err = h.evaluateSignatures(context.Background(), "gh/acme/widget", "widget", d, []metadata.Signature{
		{BundleJSON: bundle},
	})
	if err != nil || status != "valid" {
		t.Fatalf("valid: status=%q err=%v", status, err)
	}

	status, err = h.evaluateSignatures(context.Background(), "gh/acme/widget", "widget", d, []metadata.Signature{
		{BundleJSON: []byte(`{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json"}`)},
	})
	if err != nil || status != "invalid" {
		t.Fatalf("invalid stub: status=%q err=%v", status, err)
	}

	h.Manifests = nil
	status, err = h.evaluateSignatures(context.Background(), "gh/acme/widget", "widget", d, []metadata.Signature{
		{BundleJSON: bundle},
	})
	if err != nil || status != "invalid" {
		t.Fatalf("no manifest source: status=%q err=%v", status, err)
	}
}
