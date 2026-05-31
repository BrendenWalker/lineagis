package signing

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/BrendenWalker/lineagis/internal/registry"
)

func TestVerifyManifestBundle_validAndInvalid(t *testing.T) {
	layers := []registry.FileLayer{{Path: "bin/app", Data: []byte("hello")}}
	manifestJSON, _, err := registry.BuildArtifactManifest(layers, registry.ManifestOptions{})
	if err != nil {
		t.Fatal(err)
	}

	bundle, pub, err := SignManifestForTest(manifestJSON)
	if err != nil {
		t.Fatal(err)
	}

	cfg := LoadConfig()
	opts := VerifyOptions{PublicKeyPEM: pub, IgnoreTlog: true, IgnoreSCT: true}
	if err := VerifyManifestBundle(context.Background(), cfg, manifestJSON, bundle, opts); err != nil {
		t.Fatalf("valid bundle: %v", err)
	}

	tampered := bytes.Clone(manifestJSON)
	tampered[len(tampered)-1] ^= 0xff
	if err := VerifyManifestBundle(context.Background(), cfg, tampered, bundle, opts); err == nil {
		t.Fatal("expected verify failure for tampered manifest")
	}

	stub := json.RawMessage(`{"mediaType":"application/vnd.dev.sigstore.bundle.v0.3+json"}`)
	if err := VerifyManifestBundle(context.Background(), cfg, manifestJSON, stub, opts); err == nil {
		t.Fatal("expected verify failure for stub bundle")
	}
}

func TestSignManifestForTest_roundTrip(t *testing.T) {
	manifestJSON := []byte(`{"hello":"world"}`)
	bundle, pub, err := SignManifestForTest(manifestJSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(bundle) == 0 || len(pub) == 0 {
		t.Fatal("expected non-empty bundle and public key")
	}
	if !BundleHasEmbeddedCert(bundle) {
		t.Fatalf("expected embedded cert in legacy test bundle; prefix: %.200s", bundle)
	}
}
