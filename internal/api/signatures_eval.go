package api

import (
	"bytes"
	"context"
	"fmt"

	"github.com/google/go-containerregistry/pkg/v1"

	"github.com/BrendenWalker/verity/internal/metadata"
	"github.com/BrendenWalker/verity/internal/signing"
)

// evaluateSignatures returns valid, invalid, or missing (FR-SIGN-006, AC-SIGN-004).
func (h *Handler) evaluateSignatures(ctx context.Context, ns, artifact string, d *metadata.Digest, sigs []metadata.Signature) (string, error) {
	if len(sigs) == 0 {
		return "missing", nil
	}
	if h.Manifests == nil {
		return "invalid", nil
	}

	manifestJSON, err := h.Manifests.FetchManifest(ctx, ns, artifact, d.Digest)
	if err != nil {
		return "invalid", fmt.Errorf("fetch manifest: %w", err)
	}
	if err := manifestDigestMatches(manifestJSON, d.Digest); err != nil {
		return "invalid", err
	}

	cfg := signing.LoadConfig()
	keylessOpts := signing.KeylessVerifyOptions(nil)
	if h.Store != nil {
		if namespace, err := h.Store.GetNamespaceByName(ctx, ns); err == nil {
			if policy, err := h.Store.GetActivePolicy(ctx, namespace.ID); err == nil {
				keylessOpts = signing.KeylessVerifyOptions(policy.Document)
			}
		}
	}

	for _, sig := range sigs {
		bundle := signatureBundleBytes(sig)
		if len(bundle) == 0 {
			continue
		}
		pub := signing.PublicKeyPEMFromBundle(bundle)
		if len(pub) == 0 {
			pub = signing.LegacyBundleCertPEM(bundle)
		}
		opts := keylessOpts
		switch {
		case len(pub) > 0:
			opts.PublicKeyPEM = pub
			opts.IgnoreTlog = true
			opts.IgnoreSCT = true
		default:
			opts.PublicKeyPEM = nil
			opts.IgnoreTlog = false
			opts.IgnoreSCT = false
		}
		if err := signing.VerifyManifestBundle(ctx, cfg, manifestJSON, bundle, opts); err == nil {
			return "valid", nil
		}
	}
	return "invalid", nil
}

func signatureBundleBytes(sig metadata.Signature) []byte {
	if len(sig.BundleJSON) > 0 {
		return sig.BundleJSON
	}
	return nil
}

func manifestDigestMatches(manifestJSON []byte, want string) error {
	h, _, err := v1.SHA256(bytes.NewReader(manifestJSON))
	if err != nil {
		return fmt.Errorf("compute manifest digest: %w", err)
	}
	if h.String() != want {
		return fmt.Errorf("manifest digest mismatch: got %s, want %s", h, want)
	}
	return nil
}
