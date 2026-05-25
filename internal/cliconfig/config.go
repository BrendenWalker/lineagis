package cliconfig

import (
	"fmt"
	"os"
	"strings"
)

// Config holds CLI defaults for API and registry endpoints (FR-DX-004).
type Config struct {
	APIURL      string
	RegistryURL string
	Token       string
}

// Load reads configuration from environment variables.
func Load() (Config, error) {
	cfg := Config{
		APIURL:      strings.TrimSpace(os.Getenv("VERITY_API_URL")),
		RegistryURL: strings.TrimSpace(os.Getenv("VERITY_REGISTRY_URL")),
		Token:       strings.TrimSpace(os.Getenv("VERITY_TOKEN")),
	}
	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:8080"
	}
	if cfg.RegistryURL == "" {
		cfg.RegistryURL = "http://localhost:5000"
	}
	if cfg.Token == "" {
		if dev := strings.TrimSpace(os.Getenv("VERITY_DEV_TOKEN")); dev != "" {
			cfg.Token = dev
		}
	}
	if cfg.Token == "" {
		return Config{}, fmt.Errorf("VERITY_TOKEN (or VERITY_DEV_TOKEN) is required")
	}
	return cfg, nil
}
