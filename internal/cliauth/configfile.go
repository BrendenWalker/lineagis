package cliauth

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// File holds persisted CLI credentials (FR-DX-011).
type File struct {
	APIURL      string `json:"api_url,omitempty"`
	RegistryURL string `json:"registry_url,omitempty"`
	Token       string `json:"token,omitempty"`
}

// ConfigPath returns the default config file path (~/.lineagis/config).
func ConfigPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		dir = filepath.Join(home, ".lineagis")
	} else {
		dir = filepath.Join(dir, "lineagis")
	}
	return filepath.Join(dir, "config"), nil
}

// LoadFile reads the config file if present.
func LoadFile() (File, error) {
	path, err := ConfigPath()
	if err != nil {
		return File{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return File{}, nil
		}
		return File{}, err
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return File{}, fmt.Errorf("parse %s: %w", path, err)
	}
	return f, nil
}

// SaveFile writes credentials with mode 0600.
func SaveFile(f File) error {
	path, err := ConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, err := json.Marshal(f)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
