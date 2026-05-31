package signing

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	cosignoptions "github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
	"github.com/sigstore/cosign/v2/cmd/cosign/cli/sign"
	"github.com/sigstore/cosign/v2/pkg/cosign"
)

func signManifestWithKey(pass cosign.PassFunc, manifestJSON []byte) (bundle json.RawMessage, _, publicKeyPEM []byte, err error) {
	keys, err := cosign.GenerateKeyPair(pass)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("signing: generate key: %w", err)
	}

	tmpDir, err := os.MkdirTemp("", "lineagis-test-sign-*")
	if err != nil {
		return nil, nil, nil, fmt.Errorf("signing: temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	manifestPath := filepath.Join(tmpDir, "manifest.json")
	if err := os.WriteFile(manifestPath, manifestJSON, 0600); err != nil {
		return nil, nil, nil, fmt.Errorf("signing: write manifest: %w", err)
	}

	keyPath := filepath.Join(tmpDir, "cosign.key")
	if err := os.WriteFile(keyPath, keys.PrivateBytes, 0600); err != nil {
		return nil, nil, nil, fmt.Errorf("signing: write key: %w", err)
	}

	bundlePath := filepath.Join(tmpDir, "bundle.json")
	ro := &cosignoptions.RootOptions{Timeout: DefaultTestTimeout()}
	ko := cosignoptions.KeyOpts{
		KeyRef:           keyPath,
		BundlePath:       bundlePath,
		NewBundleFormat:  false,
		SkipConfirmation: true,
	}
	if _, err := sign.SignBlobCmd(ro, ko, manifestPath, true, "", "", false); err != nil {
		return nil, nil, nil, fmt.Errorf("signing: cosign sign-blob: %w", err)
	}

	bundleBytes, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("signing: read bundle: %w", err)
	}
	bundleBytes, err = augmentBundleWithPublicKey(bundleBytes, keys.PublicBytes)
	if err != nil {
		return nil, nil, nil, err
	}
	return json.RawMessage(bundleBytes), keys.PrivateBytes, keys.PublicBytes, nil
}

func augmentBundleWithPublicKey(bundleJSON, pubPEM []byte) ([]byte, error) {
	var sp struct {
		Base64Signature string          `json:"base64Signature"`
		Cert            string          `json:"cert,omitempty"`
		RekorBundle     json.RawMessage `json:"rekorBundle,omitempty"`
	}
	if err := json.Unmarshal(bundleJSON, &sp); err != nil {
		return nil, fmt.Errorf("signing: parse bundle: %w", err)
	}
	if sp.Cert == "" && len(pubPEM) > 0 {
		sp.Cert = base64.StdEncoding.EncodeToString(pubPEM)
	}
	return json.Marshal(sp)
}
