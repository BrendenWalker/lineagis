package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Config holds Verity API runtime configuration from environment variables.
type Config struct {
	APIAddr          string
	DatabaseURL      string
	RegistryURL      string
	DevToken         string // VERITY_DEV_TOKEN — bearer stub for local dev (OQ-API-002)
	OIDCIssuer       string // VERITY_OIDC_ISSUER — e.g. https://token.actions.githubusercontent.com
	OIDCAudience     string // VERITY_OIDC_AUDIENCE — expected JWT aud claim
	TLSCertFile      string
	TLSKeyFile       string
	LogLevel         slog.Level
	LogFormat        string // "json" or "text"
	MigrateOnStartup bool
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		APIAddr:          envOr("VERITY_API_ADDR", ":8080"),
		DatabaseURL:      os.Getenv("VERITY_DATABASE_URL"),
		RegistryURL:      envOr("VERITY_REGISTRY_URL", "http://registry:5000"),
		DevToken:         os.Getenv("VERITY_DEV_TOKEN"),
		OIDCIssuer:       strings.TrimSpace(os.Getenv("VERITY_OIDC_ISSUER")),
		OIDCAudience:     strings.TrimSpace(os.Getenv("VERITY_OIDC_AUDIENCE")),
		TLSCertFile:      os.Getenv("VERITY_API_TLS_CERT"),
		TLSKeyFile:       os.Getenv("VERITY_API_TLS_KEY"),
		LogFormat:        envOr("VERITY_LOG_FORMAT", "json"),
		MigrateOnStartup: envBool("VERITY_MIGRATE_ON_STARTUP", true),
	}

	level, err := parseLogLevel(envOr("VERITY_LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}
	cfg.LogLevel = level

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("VERITY_DATABASE_URL is required")
	}

	if cfg.OIDCIssuer != "" && cfg.OIDCAudience == "" {
		return Config{}, fmt.Errorf("VERITY_OIDC_AUDIENCE is required when VERITY_OIDC_ISSUER is set")
	}
	if cfg.DevToken == "" && cfg.OIDCIssuer == "" {
		return Config{}, fmt.Errorf("VERITY_DEV_TOKEN or VERITY_OIDC_ISSUER is required for API authentication")
	}

	if (cfg.TLSCertFile == "") != (cfg.TLSKeyFile == "") {
		return Config{}, fmt.Errorf("VERITY_API_TLS_CERT and VERITY_API_TLS_KEY must both be set or both be empty")
	}

	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func parseLogLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("invalid VERITY_LOG_LEVEL %q", s)
	}
}
