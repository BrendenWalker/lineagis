package signing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	cosignoptions "github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/verify"
)

// VerifyOptions tunes Sigstore verification (FR-SIGN-006, NFR-SIGN-001).
type VerifyOptions struct {
	// PublicKeyPEM verifies key-signed bundles offline (tests). When empty, keyless
	// bundles use Sigstore trusted root (TUF) and Rekor/Fulcio from Config.
	PublicKeyPEM []byte
	IgnoreTlog   bool
	IgnoreSCT    bool
}

// VerifyManifestBundle checks that bundle cryptographically covers manifestJSON (FR-SIGN-003, FR-SIGN-006).
func VerifyManifestBundle(ctx context.Context, cfg Config, manifestJSON, bundleJSON []byte, opts VerifyOptions) error {
	if len(manifestJSON) == 0 {
		return fmt.Errorf("signing: manifest is empty")
	}
	if len(bundleJSON) == 0 {
		return fmt.Errorf("signing: bundle is empty")
	}

	tmpDir, err := os.MkdirTemp("", "verity-verify-*")
	if err != nil {
		return fmt.Errorf("signing: temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	manifestPath := filepath.Join(tmpDir, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestJSON, 0600); err != nil {
		return fmt.Errorf("signing: write manifest: %w", err)
	}

	bundlePath := filepath.Join(tmpDir, "bundle.json")
	if err := os.WriteFile(bundlePath, bundleJSON, 0600); err != nil {
		return fmt.Errorf("signing: write bundle: %w", err)
	}

	newBundle := isSigstoreBundleV3(bundleJSON)

	ko := cosignoptions.KeyOpts{
		FulcioURL:       cfg.FulcioURL,
		RekorURL:        cfg.RekorURL,
		BundlePath:      bundlePath,
		NewBundleFormat: newBundle,
	}
	if len(opts.PublicKeyPEM) > 0 {
		pubPath := filepath.Join(tmpDir, "pub.pem")
		if err := os.WriteFile(pubPath, opts.PublicKeyPEM, 0600); err != nil {
			return fmt.Errorf("signing: write public key: %w", err)
		}
		ko.KeyRef = pubPath
	}

	cmd := verify.VerifyBlobCmd{
		KeyOpts:             ko,
		IgnoreTlog:          opts.IgnoreTlog || !newBundle,
		IgnoreSCT:           opts.IgnoreSCT || len(opts.PublicKeyPEM) > 0 || !newBundle,
		UseSignedTimestamps: false,
	}
	if err := cmd.Exec(ctx, manifestPath); err != nil {
		return fmt.Errorf("signing: verify bundle: %w", err)
	}
	return nil
}

// SignManifestForTest signs manifest bytes with an ephemeral cosign key and returns a v0.3 bundle.
// Used by unit tests; production publish uses keyless SignManifest.
func SignManifestForTest(manifestJSON []byte) (bundle json.RawMessage, publicKeyPEM []byte, err error) {
	if len(manifestJSON) == 0 {
		return nil, nil, fmt.Errorf("signing: manifest is empty")
	}
	bundle, _, pub, err := signManifestWithKey(nil, manifestJSON)
	return bundle, pub, err
}
