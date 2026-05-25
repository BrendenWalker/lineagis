package signing_test

import (
	"testing"

	"github.com/BrendenWalker/verity/internal/signing"
)

func TestSkipSignFromEnv(t *testing.T) {
	t.Setenv("VERITY_SKIP_SIGN", "1")
	if !signing.SkipSignFromEnv() {
		t.Fatal("expected skip sign true")
	}
}

func TestLoadConfig_defaults(t *testing.T) {
	t.Setenv("SIGSTORE_FULCIO_URL", "")
	t.Setenv("SIGSTORE_REKOR_URL", "")
	cfg := signing.LoadConfig()
	if cfg.FulcioURL == "" || cfg.RekorURL == "" {
		t.Fatalf("fulcio=%q rekor=%q", cfg.FulcioURL, cfg.RekorURL)
	}
}
