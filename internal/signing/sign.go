package signing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	cosignoptions "github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/sign"
)

// SignManifest performs keyless Sigstore signing over the canonical manifest bytes (FR-SIGN-003).
// The returned bundle uses application/vnd.dev.sigstore.bundle.v0.3+json (FR-SIGN-009).
func SignManifest(ctx context.Context, cfg Config, manifestJSON []byte) (bundle json.RawMessage, identity Identity, err error) {
	if len(manifestJSON) == 0 {
		return nil, Identity{}, fmt.Errorf("signing: manifest is empty")
	}

	tmpDir, err := os.MkdirTemp("", "verity-sign-*")
	if err != nil {
		return nil, Identity{}, fmt.Errorf("signing: temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	manifestPath := filepath.Join(tmpDir, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestJSON, 0600); err != nil {
		return nil, Identity{}, fmt.Errorf("signing: write manifest: %w", err)
	}

	bundlePath := filepath.Join(tmpDir, "bundle.json")
	certPath := filepath.Join(tmpDir, "cert.pem")

	ro := &cosignoptions.RootOptions{Timeout: cfg.Timeout}
	ko := cosignoptions.KeyOpts{
		FulcioURL:            cfg.FulcioURL,
		RekorURL:             cfg.RekorURL,
		IDToken:              cfg.IDToken,
		BundlePath:           bundlePath,
		NewBundleFormat:      true,
		SkipConfirmation:     true,
		OIDCDisableProviders: cfg.IDToken != "",
	}

	if _, err := sign.SignBlobCmd(ro, ko, manifestPath, false, "", certPath, true); err != nil {
		return nil, Identity{}, fmt.Errorf("signing: cosign sign-blob: %w", err)
	}

	bundleBytes, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, Identity{}, fmt.Errorf("signing: read bundle: %w", err)
	}
	if len(bundleBytes) == 0 {
		return nil, Identity{}, fmt.Errorf("signing: empty bundle output")
	}

	identity = Identity{}
	if certBytes, err := os.ReadFile(certPath); err == nil && len(certBytes) > 0 {
		identity, _ = IdentityFromCertificatePEM(certBytes)
	}

	return json.RawMessage(bundleBytes), identity, nil
}
