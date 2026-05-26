package signing_test

import (
	"os"
	"testing"

	cosignoptions "github.com/sigstore/cosign/v2/cmd/cosign/cli/options"

	"github.com/BrendenWalker/verity/internal/signing"
)

func TestSkipSignFromEnv(t *testing.T) {
	t.Setenv("VERITY_SKIP_SIGN", "1")
	if !signing.SkipSignFromEnv() {
		t.Fatal("expected skip sign true")
	}
}

func TestLoadConfig_defaultsPublicGood(t *testing.T) {
	for _, k := range []string{
		"VERITY_SIGSTORE_FULCIO_URL",
		"VERITY_SIGSTORE_REKOR_URL",
		"SIGSTORE_FULCIO_URL",
		"SIGSTORE_REKOR_URL",
	} {
		t.Setenv(k, "")
	}
	cfg := signing.LoadConfig()
	if cfg.FulcioURL != cosignoptions.DefaultFulcioURL {
		t.Fatalf("fulcio: got %q want %q", cfg.FulcioURL, cosignoptions.DefaultFulcioURL)
	}
	if cfg.RekorURL != cosignoptions.DefaultRekorURL {
		t.Fatalf("rekor: got %q want %q", cfg.RekorURL, cosignoptions.DefaultRekorURL)
	}
}

func TestLoadConfig_verityOverridesSigstore(t *testing.T) {
	t.Setenv("VERITY_SIGSTORE_FULCIO_URL", "https://fulcio.example")
	t.Setenv("SIGSTORE_FULCIO_URL", "https://fulcio.other")
	t.Setenv("VERITY_SIGSTORE_REKOR_URL", "https://rekor.example")
	t.Setenv("SIGSTORE_REKOR_URL", "https://rekor.other")

	cfg := signing.LoadConfig()
	if cfg.FulcioURL != "https://fulcio.example" {
		t.Fatalf("fulcio: %q", cfg.FulcioURL)
	}
	if cfg.RekorURL != "https://rekor.example" {
		t.Fatalf("rekor: %q", cfg.RekorURL)
	}
}

func TestLoadConfig_sigstoreFallback(t *testing.T) {
	t.Setenv("VERITY_SIGSTORE_FULCIO_URL", "")
	t.Setenv("VERITY_SIGSTORE_REKOR_URL", "")
	t.Setenv("SIGSTORE_FULCIO_URL", "https://fulcio.fallback")
	t.Setenv("SIGSTORE_REKOR_URL", "https://rekor.fallback")

	cfg := signing.LoadConfig()
	if cfg.FulcioURL != "https://fulcio.fallback" {
		t.Fatalf("fulcio: %q", cfg.FulcioURL)
	}
	if cfg.RekorURL != "https://rekor.fallback" {
		t.Fatalf("rekor: %q", cfg.RekorURL)
	}
}

func TestLoadConfig_trustMaterial(t *testing.T) {
	t.Setenv("VERITY_SIGSTORE_TRUSTED_ROOT", "/etc/verity/trustedroot.json")
	t.Setenv("VERITY_SIGSTORE_CA_ROOTS", "/etc/verity/ca-roots.pem")
	t.Setenv("VERITY_SIGSTORE_ROOT_FILE", "/etc/verity/fulcio-root.pem")
	t.Setenv("SIGSTORE_ROOT_FILE", "")

	cfg := signing.LoadConfig()
	if cfg.TrustedRootPath != "/etc/verity/trustedroot.json" {
		t.Fatalf("trusted root: %q", cfg.TrustedRootPath)
	}
	if cfg.CARoots != "/etc/verity/ca-roots.pem" {
		t.Fatalf("ca roots: %q", cfg.CARoots)
	}
	if cfg.RootCAFile != "/etc/verity/fulcio-root.pem" {
		t.Fatalf("root file: %q", cfg.RootCAFile)
	}
}

func TestApplyTrustEnv_setsSigstoreWhenUnset(t *testing.T) {
	t.Setenv("SIGSTORE_ROOT_FILE", "")
	t.Setenv("SIGSTORE_REKOR_PUBLIC_KEY", "")
	t.Setenv("SIGSTORE_CT_LOG_PUBLIC_KEY_FILE", "")

	cfg := signing.Config{
		RootCAFile:     "/roots.pem",
		RekorPublicKey: "/rekor.pub",
		CTLogPublicKey: "/ct.pem",
	}
	cfg.ApplyTrustEnv()

	if got := os.Getenv("SIGSTORE_ROOT_FILE"); got != "/roots.pem" {
		t.Fatalf("SIGSTORE_ROOT_FILE: %q", got)
	}
	if got := os.Getenv("SIGSTORE_REKOR_PUBLIC_KEY"); got != "/rekor.pub" {
		t.Fatalf("SIGSTORE_REKOR_PUBLIC_KEY: %q", got)
	}
	if got := os.Getenv("SIGSTORE_CT_LOG_PUBLIC_KEY_FILE"); got != "/ct.pem" {
		t.Fatalf("SIGSTORE_CT_LOG_PUBLIC_KEY_FILE: %q", got)
	}
}

func TestApplyTrustEnv_doesNotOverrideExisting(t *testing.T) {
	t.Setenv("SIGSTORE_ROOT_FILE", "/existing.pem")

	cfg := signing.Config{RootCAFile: "/verity.pem"}
	cfg.ApplyTrustEnv()

	if got := os.Getenv("SIGSTORE_ROOT_FILE"); got != "/existing.pem" {
		t.Fatalf("SIGSTORE_ROOT_FILE: %q", got)
	}
}
