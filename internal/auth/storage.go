// Package auth implements persistence for authentication state.
//
// This file stores the serialized state in the OS keychain via internal/keychain.
package auth

import (
	"encoding/json"

	"seedfast/cli/internal/keychain"
)

// State represents persisted authentication state for the current user.
type State struct {
	LoggedIn bool   `json:"logged_in"`
	Account  string `json:"account"`
}

// Load reads the auth state from the keychain. Missing state yields zero value.
func Load() (State, error) {
	var s State
	data, err := keychain.MustGetManager().LoadAuthState()
	if err != nil {
		return s, err
	}
	if len(data) == 0 {
		return s, nil
	}
	if err := json.Unmarshal(data, &s); err != nil {
		return s, err
	}
	return s, nil
}

// Save writes the auth state to the keychain.
func Save(s State) error {
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return keychain.MustGetManager().SaveAuthState(b)
}

// Clear removes the auth state from the keychain.
func Clear() error { return keychain.MustGetManager().ClearAuthState() }
