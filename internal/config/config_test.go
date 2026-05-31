package config

import (
	"log/slog"
	"testing"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("LINEAGIS_DATABASE_URL", "")
	t.Setenv("LINEAGIS_DEV_TOKEN", "x")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("LINEAGIS_DATABASE_URL", "postgres://lineagis:lineagis@localhost:5432/lineagis?sslmode=disable")
	t.Setenv("LINEAGIS_API_ADDR", "")
	t.Setenv("LINEAGIS_REGISTRY_URL", "")
	t.Setenv("LINEAGIS_LOG_LEVEL", "")
	t.Setenv("LINEAGIS_LOG_FORMAT", "")
	t.Setenv("LINEAGIS_MIGRATE_ON_STARTUP", "")
	t.Setenv("LINEAGIS_API_TLS_CERT", "")
	t.Setenv("LINEAGIS_API_TLS_KEY", "")
	t.Setenv("LINEAGIS_DEV_TOKEN", "test-dev-token")
	t.Setenv("LINEAGIS_OIDC_ISSUER", "")
	t.Setenv("LINEAGIS_OIDC_AUDIENCE", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.APIAddr != ":8080" {
		t.Fatalf("APIAddr = %q, want :8080", cfg.APIAddr)
	}
	if cfg.RegistryURL != "http://registry:5000" {
		t.Fatalf("RegistryURL = %q, want http://registry:5000", cfg.RegistryURL)
	}
	if cfg.LogLevel != slog.LevelInfo {
		t.Fatalf("LogLevel = %v, want info", cfg.LogLevel)
	}
	if cfg.LogFormat != "json" {
		t.Fatalf("LogFormat = %q, want json", cfg.LogFormat)
	}
	if !cfg.MigrateOnStartup {
		t.Fatal("MigrateOnStartup = false, want true")
	}
}

func TestLoadTLSRequiresBothCertAndKey(t *testing.T) {
	t.Setenv("LINEAGIS_DATABASE_URL", "postgres://localhost/lineagis")
	t.Setenv("LINEAGIS_DEV_TOKEN", "x")
	t.Setenv("LINEAGIS_API_TLS_CERT", "/tmp/cert.pem")
	t.Setenv("LINEAGIS_API_TLS_KEY", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}
