package signing

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"
)

func isSigstoreBundleV3(bundleJSON []byte) bool {
	return bytes.Contains(bundleJSON, []byte("application/vnd.dev.sigstore.bundle.v0.3"))
}

// PublicKeyPEMFromBundle returns PEM public key bytes when explicitly embedded in a v0.3 bundle.
func PublicKeyPEMFromBundle(bundleJSON []byte) []byte {
	if !isSigstoreBundleV3(bundleJSON) {
		return nil
	}
	var doc struct {
		VerificationMaterial struct {
			PublicKey *struct {
				RawBytes string `json:"rawBytes"`
			} `json:"publicKey"`
		} `json:"verificationMaterial"`
	}
	if err := json.Unmarshal(bundleJSON, &doc); err != nil {
		return nil
	}
	if doc.VerificationMaterial.PublicKey == nil {
		return nil
	}
	raw := doc.VerificationMaterial.PublicKey.RawBytes
	if raw == "" {
		return nil
	}
	return []byte(raw)
}

// BundleHasEmbeddedCert reports whether a legacy cosign bundle JSON includes a certificate.
func BundleHasEmbeddedCert(bundleJSON []byte) bool {
	return len(LegacyBundleCertPEM(bundleJSON)) > 0
}

// LegacyBundleCertPEM returns PEM bytes from a legacy bundle cert field (certificate or public key).
func LegacyBundleCertPEM(bundleJSON []byte) []byte {
	var doc struct {
		Cert string `json:"cert"`
	}
	if err := json.Unmarshal(bundleJSON, &doc); err != nil {
		return nil
	}
	raw := strings.TrimSpace(doc.Cert)
	if raw == "" {
		return nil
	}
	if pem, err := base64.StdEncoding.DecodeString(raw); err == nil && len(pem) > 0 {
		return pem
	}
	return []byte(raw)
}
