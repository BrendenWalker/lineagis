package inspect

import (
	"context"
	"fmt"

	"github.com/BrendenWalker/verity/internal/apiclient"
	"github.com/BrendenWalker/verity/internal/registry"
	"github.com/BrendenWalker/verity/internal/signing"
)

// LocalVerifyResult holds client-side Sigstore verification outcome (FR-SIGN-005).
type LocalVerifyResult struct {
	Status   string // valid, invalid, missing
	Signer   signing.GitHubPublisher
	Identity signing.Identity
}

// VerifyLocally checks signatures against registry manifest bytes without trusting API crypto.
func VerifyLocally(ctx context.Context, reg *registry.Client, api *apiclient.Client, namespace, artifact, digest string) (*LocalVerifyResult, error) {
	sigs, err := api.ListSignatures(ctx, namespace, artifact, digest)
	if err != nil {
		return nil, fmt.Errorf("list signatures: %w", err)
	}
	if len(sigs) == 0 {
		return &LocalVerifyResult{Status: "missing"}, nil
	}

	repo := namespace + "/" + artifact
	manifestJSON, pulled, err := reg.PullManifest(ctx, repo, digest)
	if err != nil {
		return nil, fmt.Errorf("pull manifest: %w", err)
	}
	if pulled.String() != digest {
		return nil, fmt.Errorf("manifest digest mismatch: got %s, want %s", pulled, digest)
	}

	cfg := signing.LoadConfig()
	policyDoc := policyDocument(ctx, api, namespace)
	keylessOpts := signing.KeylessVerifyOptions(policyDoc)
	for _, sig := range sigs {
		if len(sig.Bundle) == 0 {
			continue
		}
		opts := keylessOpts
		if pub := signing.PublicKeyPEMFromBundle(sig.Bundle); len(pub) > 0 {
			opts.PublicKeyPEM = pub
			opts.IgnoreTlog = true
			opts.IgnoreSCT = true
		}
		if err := signing.VerifyManifestBundle(ctx, cfg, manifestJSON, sig.Bundle, opts); err != nil {
			continue
		}
		out := &LocalVerifyResult{Status: "valid"}
		if pub, ok := signing.GitHubPublisherFromBundle(sig.Bundle); ok {
			out.Signer = pub
		}
		if pem := signing.LegacyBundleCertPEM(sig.Bundle); len(pem) > 0 {
			out.Identity, _ = signing.IdentityFromCertificatePEM(pem)
		}
		return out, nil
	}
	return &LocalVerifyResult{Status: "invalid"}, nil
}
