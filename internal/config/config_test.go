package config

import (
	"log/slog"
	"testing"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("VERITY_DATABASE_URL", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("VERITY_DATABASE_URL", "postgres://verity:verity@localhost:5432/verity?sslmode=disable")
	t.Setenv("VERITY_API_ADDR", "")
	t.Setenv("VERITY_REGISTRY_URL", "")
	t.Setenv("VERITY_LOG_LEVEL", "")
	t.Setenv("VERITY_LOG_FORMAT", "")
	t.Setenv("VERITY_MIGRATE_ON_STARTUP", "")
	t.Setenv("VERITY_API_TLS_CERT", "")
	t.Setenv("VERITY_API_TLS_KEY", "")

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
	t.Setenv("VERITY_DATABASE_URL", "postgres://localhost/verity")
	t.Setenv("VERITY_API_TLS_CERT", "/tmp/cert.pem")
	t.Setenv("VERITY_API_TLS_KEY", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
}
