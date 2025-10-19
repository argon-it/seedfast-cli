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

var verboseAuth = os.Getenv("SEEDFAST_VERBOSE") == "1"

// State represents persisted authentication state for the current user.
type State struct {
	LoggedIn bool   `json:"logged_in"`
	Account  string `json:"account"`
}

// Load reads the auth state from the keychain. Missing state yields zero value.
func Load() (State, error) {
	if verboseAuth {
		fmt.Printf("[DEBUG] auth.Load: Loading auth state from keychain\n")
	}

	var s State
	km, err := keychain.GetManager()
	if err != nil {
		if verboseAuth {
			fmt.Printf("[DEBUG] auth.Load: GetManager failed: %v\n", err)
		}
		return s, err
	}

	data, err := km.LoadAuthState()
	if err != nil {
		if verboseAuth {
			fmt.Printf("[DEBUG] auth.Load: LoadAuthState failed: %v\n", err)
		}
		return s, err
	}

	if len(data) == 0 {
		if verboseAuth {
			fmt.Printf("[DEBUG] auth.Load: No auth state found (empty data)\n")
		}
		return s, nil
	}

	if verboseAuth {
		fmt.Printf("[DEBUG] auth.Load: Got data, length: %d, unmarshaling...\n", len(data))
	}

	if err := json.Unmarshal(data, &s); err != nil {
		if verboseAuth {
			fmt.Printf("[DEBUG] auth.Load: Unmarshal failed: %v\n", err)
		}
		return s, err
	}

	if verboseAuth {
		fmt.Printf("[DEBUG] auth.Load: Success - LoggedIn: %v, Account: %s\n", s.LoggedIn, s.Account)
	}

	return s, nil
}

// Save writes the auth state to the keychain.
func Save(s State) error {
	if verboseAuth {
		fmt.Printf("[DEBUG] auth.Save: Saving auth state - LoggedIn: %v, Account: %s\n", s.LoggedIn, s.Account)
	}

	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		if verboseAuth {
			fmt.Printf("[DEBUG] auth.Save: MarshalIndent failed: %v\n", err)
		}
		return err
	}

	if verboseAuth {
		fmt.Printf("[DEBUG] auth.Save: Marshaled to JSON, length: %d\n", len(b))
	}

	km, err := keychain.GetManager()
	if err != nil {
		if verboseAuth {
			fmt.Printf("[DEBUG] auth.Save: GetManager failed: %v\n", err)
		}
		return err
	}

	if verboseAuth {
		fmt.Printf("[DEBUG] auth.Save: Calling SaveAuthState...\n")
	}

	err = km.SaveAuthState(b)
	if err != nil {
		if verboseAuth {
			fmt.Printf("[DEBUG] auth.Save: SaveAuthState failed: %v\n", err)
		}
		return err
	}

	if verboseAuth {
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
