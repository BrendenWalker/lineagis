package signing

import (
	"os"
	"strings"
	"time"

	cosignoptions "github.com/sigstore/cosign/v2/cmd/cosign/cli/options"
)

// Config holds Sigstore/cosign settings for keyless manifest signing (FR-SIGN-001, NFR-SIGN-001).
type Config struct {
	FulcioURL string
	RekorURL  string
	IDToken   string
	Timeout   time.Duration
}

// LoadConfig reads signing configuration from the environment.
//
// Local dev without Fulcio: set VERITY_SKIP_SIGN=1 or pass --skip-sign on publish.
// CI / keyless: set SIGSTORE_ID_TOKEN, or rely on GitHub Actions ambient OIDC
// (ACTIONS_ID_TOKEN_REQUEST_URL / ACTIONS_ID_TOKEN_REQUEST_TOKEN).
// Optional overrides: SIGSTORE_FULCIO_URL, SIGSTORE_REKOR_URL (cosign defaults apply when unset).
func LoadConfig() Config {
	cfg := Config{
		FulcioURL: strings.TrimSpace(os.Getenv("SIGSTORE_FULCIO_URL")),
		RekorURL:  strings.TrimSpace(os.Getenv("SIGSTORE_REKOR_URL")),
		IDToken:   strings.TrimSpace(os.Getenv("SIGSTORE_ID_TOKEN")),
		Timeout:   cosignoptions.DefaultTimeout,
	}
	if cfg.FulcioURL == "" {
		cfg.FulcioURL = cosignoptions.DefaultFulcioURL
	}
	if cfg.RekorURL == "" {
		cfg.RekorURL = cosignoptions.DefaultRekorURL
	}
	return cfg
}

// SkipSignFromEnv reports whether publish should skip signing (local dev).
func SkipSignFromEnv() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("VERITY_SKIP_SIGN")))
	return v == "1" || v == "true" || v == "yes"
}
