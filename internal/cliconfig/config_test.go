package cliconfig

import "testing"

func TestLoad_requiresToken(t *testing.T) {
	t.Setenv("VERITY_TOKEN", "")
	t.Setenv("VERITY_DEV_TOKEN", "")
	_, err := Load()
	if err == nil {
		t.Fatal("expected error without token")
	}
}

func TestLoad_defaults(t *testing.T) {
	t.Setenv("VERITY_TOKEN", "tok")
	t.Setenv("VERITY_API_URL", "")
	t.Setenv("VERITY_REGISTRY_URL", "")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.APIURL != "http://localhost:8080" {
		t.Fatalf("APIURL = %q", cfg.APIURL)
	}
	if cfg.RegistryURL != "http://localhost:5000" {
		t.Fatalf("RegistryURL = %q", cfg.RegistryURL)
	}
}

func TestLoad_devTokenFallback(t *testing.T) {
	t.Setenv("VERITY_TOKEN", "")
	t.Setenv("VERITY_DEV_TOKEN", "dev-secret")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Token != "dev-secret" {
		t.Fatalf("Token = %q", cfg.Token)
	}
}
