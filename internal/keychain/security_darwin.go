// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

//go:build darwin

package keychain

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// securityBackend implements keychain operations using macOS security command.
type securityBackend struct{}

// newSecurityBackend creates a new macOS security command backend.
func newSecurityBackend() (*securityBackend, error) {
	// Verify security command is available
	if _, err := exec.LookPath("security"); err != nil {
		return nil, fmt.Errorf("security command not found: %w", err)
	}
	return &securityBackend{}, nil
}

// Set stores a key-value pair in macOS keychain.
func (s *securityBackend) Set(key, value string) error {
	// Delete existing entry first (ignore errors if it doesn't exist)
	_ = s.Delete(key)

	// Add new entry
	// Use -U flag to update if exists
	cmd := exec.Command("security", "add-generic-password",
		"-a", ServiceName,        // account name
		"-s", key,                 // service name
		"-w", value,               // password
		"-U",                      // update if exists
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Include both stderr and the key name in error for debugging
		return fmt.Errorf("failed to store '%s' in keychain: %s: %w", key, stderr.String(), err)
	}

	return nil
}

// Get retrieves a value from macOS keychain.
func (s *securityBackend) Get(key string) (string, error) {
	cmd := exec.Command("security", "find-generic-password",
		"-a", ServiceName,        // account name
		"-s", key,                 // service name
		"-w",                      // output password only
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if strings.Contains(stderr.String(), "could not be found") {
			return "", fmt.Errorf("key not found")
		}
		return "", fmt.Errorf("failed to retrieve from keychain: %s: %w", stderr.String(), err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// Delete removes a key from macOS keychain.
func (s *securityBackend) Delete(key string) error {
	cmd := exec.Command("security", "delete-generic-password",
		"-a", ServiceName,        // account name
		"-s", key,                 // service name
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Ignore "not found" errors
		if strings.Contains(stderr.String(), "could not be found") {
			return nil
		}
		return fmt.Errorf("failed to delete from keychain: %s: %w", stderr.String(), err)
	}

	return nil
}
