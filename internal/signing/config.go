package signing

import (
	"os"
	"strings"
	"time"

	cosignoptions "github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
)

// Config holds Sigstore/cosign settings for keyless manifest signing and verification
// (FR-SIGN-001, NFR-SIGN-001). Unset endpoint fields default to Sigstore public-good
// (cosignoptions.DefaultFulcioURL / DefaultRekorURL). Unset trust file fields use
// cosign TUF / SIGSTORE_* when those are set in the environment.
type Config struct {
	FulcioURL       string
	RekorURL        string
	IDToken         string
	TrustedRootPath string
	CARoots         string
	CAIntermediates string
	RootCAFile      string
	RekorPublicKey  string
	CTLogPublicKey  string
	Timeout         time.Duration
}

// LoadConfig reads signing configuration from the environment.
//
// Verity-prefixed variables take precedence over cosign-standard SIGSTORE_* names.
// Local dev without Fulcio: VERITY_SKIP_SIGN=1 or --skip-sign on publish.
// CI / keyless: VERITY_SIGSTORE_ID_TOKEN or SIGSTORE_ID_TOKEN, or GitHub Actions ambient OIDC.
//
// Endpoints (default public-good when unset):
//
//	VERITY_SIGSTORE_FULCIO_URL, VERITY_SIGSTORE_REKOR_URL
//
// Trust material for verification (optional; see docs/signing-local.md):
//
//	VERITY_SIGSTORE_TRUSTED_ROOT — Sigstore trusted root JSON (v0.3 bundles)
//	VERITY_SIGSTORE_CA_ROOTS, VERITY_SIGSTORE_CA_INTERMEDIATES — PEM paths (legacy bundles)
//	VERITY_SIGSTORE_ROOT_FILE, VERITY_SIGSTORE_REKOR_PUBLIC_KEY, VERITY_SIGSTORE_CT_LOG_PUBLIC_KEY_FILE
//	(each falls back to the matching SIGSTORE_* variable)
func LoadConfig() Config {
	cfg := Config{
		FulcioURL:       envFirst("VERITY_SIGSTORE_FULCIO_URL", "SIGSTORE_FULCIO_URL"),
		RekorURL:        envFirst("VERITY_SIGSTORE_REKOR_URL", "SIGSTORE_REKOR_URL"),
		IDToken:         envFirst("VERITY_SIGSTORE_ID_TOKEN", "SIGSTORE_ID_TOKEN"),
		TrustedRootPath: envFirst("VERITY_SIGSTORE_TRUSTED_ROOT", "SIGSTORE_TRUSTED_ROOT"),
		CARoots:         envFirst("VERITY_SIGSTORE_CA_ROOTS", "SIGSTORE_CA_ROOTS"),
		CAIntermediates: envFirst("VERITY_SIGSTORE_CA_INTERMEDIATES", "SIGSTORE_CA_INTERMEDIATES"),
		RootCAFile:      envFirst("VERITY_SIGSTORE_ROOT_FILE", "SIGSTORE_ROOT_FILE"),
		RekorPublicKey:  envFirst("VERITY_SIGSTORE_REKOR_PUBLIC_KEY", "SIGSTORE_REKOR_PUBLIC_KEY"),
		CTLogPublicKey:  envFirst("VERITY_SIGSTORE_CT_LOG_PUBLIC_KEY_FILE", "SIGSTORE_CT_LOG_PUBLIC_KEY_FILE"),
		Timeout:         cosignoptions.DefaultTimeout,
	}
	if cfg.FulcioURL == "" {
		cfg.FulcioURL = cosignoptions.DefaultFulcioURL
	}
	if cfg.RekorURL == "" {
		cfg.RekorURL = cosignoptions.DefaultRekorURL
	}
	return cfg
}

// ApplyTrustEnv exports trust-material paths into cosign-standard SIGSTORE_* variables
// when those variables are not already set. Call before cosign sign/verify operations.
func (c Config) ApplyTrustEnv() {
	setIfEmpty("SIGSTORE_ROOT_FILE", c.RootCAFile)
	setIfEmpty("SIGSTORE_REKOR_PUBLIC_KEY", c.RekorPublicKey)
	setIfEmpty("SIGSTORE_CT_LOG_PUBLIC_KEY_FILE", c.CTLogPublicKey)
}

func envFirst(keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return ""
}

func setIfEmpty(key, value string) {
	if value == "" {
		return
	}
	if strings.TrimSpace(os.Getenv(key)) == "" {
		_ = os.Setenv(key, value)
	}
}

// SkipSignFromEnv reports whether publish should skip signing (local dev).
func SkipSignFromEnv() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("VERITY_SKIP_SIGN")))
	return v == "1" || v == "true" || v == "yes"
}
