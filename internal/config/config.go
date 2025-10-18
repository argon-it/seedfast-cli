// Package config loads and stores CLI configuration in the XDG config dir.
// Only non-secret settings are kept here; secrets go to OS keychain.
package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"seedfast/cli/internal/xdg"
)

// Config holds non-sensitive CLI settings.
type Config struct {
	LogLevel    string   `json:"log_level"`
	DB          DBConfig `json:"db"`
	Concurrency int      `json:"concurrency"`
}

// DBConfig holds database connection settings.
type DBConfig struct {
	DSN      string `json:"dsn"`
	Provided bool   `json:"provided"`
}

// path returns the path to the config file.
func path() (string, error) {
	dir, err := xdg.ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads configuration; missing file returns defaults.
func Load() (Config, error) {
	var c Config
	p, err := path()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Defaults (DB credentials loaded from env/keychain, not config)
			c.LogLevel = "info"
			c.DB = DBConfig{} // No default DSN - fail-fast if not provided via env/keychain
			c.Concurrency = 4
			return c, nil
		}
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return c, err
	}
	return c, nil
}

// Save writes configuration with 0600 permissions.
func Save(c Config) error {
	p, err := path()
	if err != nil {
		return err
	}
	b, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, b, 0o600)
}
