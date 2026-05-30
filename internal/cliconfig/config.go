package cliconfig

import (
	"fmt"
	"os"
	"strings"

	"github.com/BrendenWalker/verity/internal/cliauth"
)

// Config holds CLI defaults for API and registry endpoints (FR-DX-004).
type Config struct {
	APIURL      string
	RegistryURL string
	Token       string
}

// Load reads configuration from environment variables and optional ~/.verity/config (FR-DX-004, FR-DX-011).
func Load() (Config, error) {
	cfg := Config{
		APIURL:      strings.TrimSpace(os.Getenv("VERITY_API_URL")),
		RegistryURL: strings.TrimSpace(os.Getenv("VERITY_REGISTRY_URL")),
		Token:       strings.TrimSpace(os.Getenv("VERITY_TOKEN")),
	}
	if cfg.Token == "" {
		if dev := strings.TrimSpace(os.Getenv("VERITY_DEV_TOKEN")); dev != "" {
			cfg.Token = dev
		}
	}
	file, _ := cliauth.LoadFile()
	if cfg.APIURL == "" && file.APIURL != "" {
		cfg.APIURL = file.APIURL
	}
	if cfg.RegistryURL == "" && file.RegistryURL != "" {
		cfg.RegistryURL = file.RegistryURL
	}
	if cfg.Token == "" && file.Token != "" {
		cfg.Token = file.Token
	}
	if cfg.APIURL == "" {
		cfg.APIURL = "http://localhost:8080"
	}
	if cfg.RegistryURL == "" {
		cfg.RegistryURL = "http://localhost:5000"
	}
	if cfg.Token == "" {
		return Config{}, fmt.Errorf("VERITY_TOKEN (or VERITY_DEV_TOKEN) is required; run verity login")
	}
	return cfg, nil
}
