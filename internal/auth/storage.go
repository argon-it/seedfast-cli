// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package auth implements persistence for authentication state.
//
// This file stores the serialized state in the OS keychain via internal/keychain.
package auth

import (
	"encoding/json"
	"fmt"
	"os"

	"seedfast/cli/internal/keychain"
)

// isVerbose checks if verbose mode is enabled dynamically
func isVerbose() bool {
	return os.Getenv("SEEDFAST_VERBOSE") == "1"
}

// State represents persisted authentication state for the current user.
type State struct {
	LoggedIn bool   `json:"logged_in"`
	Account  string `json:"account"`
}

// Load reads the auth state from the keychain. Missing state yields zero value.
func Load() (State, error) {
	verbose := isVerbose()
	if verbose {
		fmt.Printf("[DEBUG] auth.Load: Loading auth state from keychain\n")
	}

	var s State
	km, err := keychain.GetManager()
	if err != nil {
		if verbose {
			fmt.Printf("[DEBUG] auth.Load: GetManager failed: %v\n", err)
		}
		return s, err
	}

	data, err := km.LoadAuthState()
	if err != nil {
		if verbose {
			fmt.Printf("[DEBUG] auth.Load: LoadAuthState failed: %v\n", err)
		}
		return s, err
	}

	if len(data) == 0 {
		if verbose {
			fmt.Printf("[DEBUG] auth.Load: No auth state found (empty data)\n")
		}
		return s, nil
	}

	if verbose {
		fmt.Printf("[DEBUG] auth.Load: Got data, length: %d\n", len(data))
		fmt.Printf("[DEBUG] auth.Load: Raw data: %q\n", string(data))
	}

	if err := json.Unmarshal(data, &s); err != nil {
		if verbose {
			fmt.Printf("[DEBUG] auth.Load: Unmarshal failed: %v\n", err)
			fmt.Printf("[DEBUG] auth.Load: Failed data (hex): %x\n", data)
		}
		return s, err
	}

	if verbose {
		fmt.Printf("[DEBUG] auth.Load: Success - LoggedIn: %v, Account: %s\n", s.LoggedIn, s.Account)
	}

	return s, nil
}

// Save writes the auth state to the keychain.
func Save(s State) error {
	verbose := isVerbose()
	if verbose {
		fmt.Printf("[DEBUG] auth.Save: Saving auth state - LoggedIn: %v, Account: %s\n", s.LoggedIn, s.Account)
	}

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		if verbose {
			fmt.Printf("[DEBUG] auth.Save: MarshalIndent failed: %v\n", err)
		}
		return err
	}

	if verbose {
		fmt.Printf("[DEBUG] auth.Save: Marshaled to JSON, length: %d\n", len(b))
		fmt.Printf("[DEBUG] auth.Save: JSON content: %q\n", string(b))
	}

	km, err := keychain.GetManager()
	if err != nil {
		if verbose {
			fmt.Printf("[DEBUG] auth.Save: GetManager failed: %v\n", err)
		}
		return err
	}

	if verbose {
		fmt.Printf("[DEBUG] auth.Save: Calling SaveAuthState...\n")
	}

	err = km.SaveAuthState(b)
	if err != nil {
		if verbose {
			fmt.Printf("[DEBUG] auth.Save: SaveAuthState failed: %v\n", err)
		}
		return err
	}

	if verbose {
		fmt.Printf("[DEBUG] auth.Save: Success!\n")
	}

	return nil
}

// Clear removes the auth state from the keychain.
func Clear() error {
	km, err := keychain.GetManager()
	if err != nil {
		return err
	}
	return km.ClearAuthState()
}
