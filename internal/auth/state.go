// Package auth provides authentication state management and service integration for the CLI.
// It handles user authentication flows, token management, and session validation
// through integration with the backend service and secure credential storage.
//
// The package provides both high-level authentication state management and low-level
// persistence operations, supporting device-based authentication flows with
// automatic token refresh and secure storage of credentials.
package auth

import (
	"context"
)

// IsLoggedIn reports whether the user is considered logged in.
func IsLoggedIn(ctx context.Context) (bool, error) {
	st, err := Load()
	if err != nil {
		return false, err
	}
	return st.LoggedIn, nil
}

// SetLoggedIn marks the user as logged in by writing state to disk.
func SetLoggedIn(ctx context.Context, account string) error {
	return Save(State{LoggedIn: true, Account: account})
}

// SetLoggedOut clears login state.
func SetLoggedOut(ctx context.Context) error {
	return Clear()
}
