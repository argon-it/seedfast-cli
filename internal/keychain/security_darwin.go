// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

//go:build darwin

package keychain

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// isVerbose checks if verbose mode is enabled dynamically
func isVerbose() bool {
	return os.Getenv("SEEDFAST_VERBOSE") == "1"
}

// truncate returns first n characters of s, or entire s if shorter
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// isHexString checks if a string contains only hex characters (0-9, a-f, A-F)
func isHexString(s string) bool {
	if len(s) == 0 || len(s)%2 != 0 {
		return false
	}
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

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
	verbose := isVerbose()
	if verbose {
		fmt.Printf("[DEBUG] security_darwin: Set() called for key '%s', value length: %d\n", key, len(value))
		fmt.Printf("[DEBUG] security_darwin: Set() value content (first 100 chars): %q\n", truncate(value, 100))
	}

	// Delete existing entry first (ignore errors if it doesn't exist)
	deleteErr := s.Delete(key)
	if verbose && deleteErr != nil {
		fmt.Printf("[DEBUG] security_darwin: Delete() returned: %v\n", deleteErr)
	}

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
		errMsg := fmt.Errorf("failed to store '%s' in keychain: %s: %w", key, stderr.String(), err)
		if verbose {
			fmt.Printf("[DEBUG] security_darwin: Set() failed: %v\n", errMsg)
		}
		return errMsg
	}

	if verbose {
		fmt.Printf("[DEBUG] security_darwin: Set() succeeded for key '%s'\n", key)
	}

	return nil
}

// Get retrieves a value from macOS keychain.
func (s *securityBackend) Get(key string) (string, error) {
	verbose := isVerbose()
	if verbose {
		fmt.Printf("[DEBUG] security_darwin: Get() called for key '%s'\n", key)
	}

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
			if verbose {
				fmt.Printf("[DEBUG] security_darwin: Get() key not found: '%s'\n", key)
			}
			return "", fmt.Errorf("key not found")
		}
		errMsg := fmt.Errorf("failed to retrieve from keychain: %s: %w", stderr.String(), err)
		if verbose {
			fmt.Printf("[DEBUG] security_darwin: Get() failed: %v\n", errMsg)
		}
		return "", errMsg
	}

	rawOutput := stdout.String()
	result := strings.TrimSpace(rawOutput)

	if verbose {
		fmt.Printf("[DEBUG] security_darwin: Get() raw output length: %d\n", len(rawOutput))
		fmt.Printf("[DEBUG] security_darwin: Get() raw output (first 100 chars): %q\n", truncate(rawOutput, 100))
		fmt.Printf("[DEBUG] security_darwin: Get() trimmed length: %d\n", len(result))
		fmt.Printf("[DEBUG] security_darwin: Get() trimmed (first 100 chars): %q\n", truncate(result, 100))
	}

	// macOS 26.0 returns hex-encoded data, decode if it looks like hex
	if isHexString(result) {
		decoded, err := hex.DecodeString(result)
		if err != nil {
			if verbose {
				fmt.Printf("[DEBUG] security_darwin: Get() hex decode failed: %v\n", err)
			}
			return "", fmt.Errorf("failed to decode hex output: %w", err)
		}
		result = string(decoded)
		if verbose {
			fmt.Printf("[DEBUG] security_darwin: Get() decoded from hex, new length: %d\n", len(result))
			fmt.Printf("[DEBUG] security_darwin: Get() decoded (first 100 chars): %q\n", truncate(result, 100))
		}
	}

	return result, nil
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
