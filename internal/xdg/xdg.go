// Package xdg provides helpers to resolve XDG Base Directory paths for seedfast.
// It implements the XDG Base Directory specification for determining appropriate
// locations for configuration files, state data, and other application-specific
// directories on Unix-like systems.
//
// The package handles fallback to traditional locations when XDG environment
// variables are not set and ensures proper permissions for security-sensitive
// directories like configuration storage.
package xdg

import (
	"os"
	"path/filepath"
)

// ConfigDir returns the XDG config directory for seedfast.
// The directory is created with private permissions (0700) if missing.
// It falls back to ~/.config/seedfast when XDG_CONFIG_HOME is unset.
func ConfigDir() (string, error) {
	base := os.Getenv("XDG_CONFIG_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	dir := filepath.Join(base, "seedfast")
	if err := os.MkdirAll(dir, 0o700); err != nil { // private dir
		return "", err
	}
	return dir, nil
}

// StateDir returns the XDG state directory for seedfast.
// The directory is created with private permissions (0700) if missing.
// It falls back to ~/.local/state/seedfast when XDG_STATE_HOME is unset.
func StateDir() (string, error) {
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "state")
	}
	dir := filepath.Join(base, "seedfast")
	if err := os.MkdirAll(dir, 0o700); err != nil { // private dir
		return "", err
	}
	return dir, nil
}
