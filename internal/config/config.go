package config

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// Config holds Lineagis API runtime configuration from environment variables.
type Config struct {
	APIAddr          string
	DatabaseURL      string
	RegistryURL      string
	DevToken         string // LINEAGIS_DEV_TOKEN — bearer stub for local dev (OQ-API-002)
	OIDCIssuer       string // LINEAGIS_OIDC_ISSUER — e.g. https://token.actions.githubusercontent.com
	OIDCAudience     string // LINEAGIS_OIDC_AUDIENCE — expected JWT aud claim
	GitHubToken      string // LINEAGIS_GITHUB_TOKEN — optional PAT for repository-ownership API checks
	TLSCertFile      string
	TLSKeyFile       string
	LogLevel         slog.Level
	LogFormat        string // "json" or "text"
	MigrateOnStartup bool
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		APIAddr:          envOr("LINEAGIS_API_ADDR", ":8080"),
		DatabaseURL:      os.Getenv("LINEAGIS_DATABASE_URL"),
		RegistryURL:      envOr("LINEAGIS_REGISTRY_URL", "http://registry:5000"),
		DevToken:         os.Getenv("LINEAGIS_DEV_TOKEN"),
		OIDCIssuer:       strings.TrimSpace(os.Getenv("LINEAGIS_OIDC_ISSUER")),
		OIDCAudience:     strings.TrimSpace(os.Getenv("LINEAGIS_OIDC_AUDIENCE")),
		GitHubToken:      strings.TrimSpace(os.Getenv("LINEAGIS_GITHUB_TOKEN")),
		TLSCertFile:      os.Getenv("LINEAGIS_API_TLS_CERT"),
		TLSKeyFile:       os.Getenv("LINEAGIS_API_TLS_KEY"),
		LogFormat:        envOr("LINEAGIS_LOG_FORMAT", "json"),
		MigrateOnStartup: envBool("LINEAGIS_MIGRATE_ON_STARTUP", true),
	}

	level, err := parseLogLevel(envOr("LINEAGIS_LOG_LEVEL", "info"))
	if err != nil {
		return Config{}, err
	}
	cfg.LogLevel = level

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("LINEAGIS_DATABASE_URL is required")
	}

	if cfg.OIDCIssuer != "" && cfg.OIDCAudience == "" {
		return Config{}, fmt.Errorf("LINEAGIS_OIDC_AUDIENCE is required when LINEAGIS_OIDC_ISSUER is set")
	}
	if cfg.DevToken == "" && cfg.OIDCIssuer == "" {
		return Config{}, fmt.Errorf("LINEAGIS_DEV_TOKEN or LINEAGIS_OIDC_ISSUER is required for API authentication")
	}

	if (cfg.TLSCertFile == "") != (cfg.TLSKeyFile == "") {
		return Config{}, fmt.Errorf("LINEAGIS_API_TLS_CERT and LINEAGIS_API_TLS_KEY must both be set or both be empty")
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
		return slog.LevelInfo, fmt.Errorf("invalid LINEAGIS_LOG_LEVEL %q", s)
	}
}
