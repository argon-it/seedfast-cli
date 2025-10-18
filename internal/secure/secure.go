// Package secure provides a legacy compatibility layer for keychain operations.
// This package now uses the centralized keychain manager from internal/keychain
// and exists primarily for backward compatibility. For new code, use internal/keychain directly.
//
// The package provides wrapper functions that delegate to the centralized keychain manager,
// ensuring existing code continues to work while encouraging migration to the new interface.
package secure

import (
	"seedfast/cli/internal/keychain"
)

// SaveAuthTokens stores access and refresh tokens in the OS keychain.
func SaveAuthTokens(accessToken string, refreshToken string) error {
	manager, err := keychain.GetManager()
	if err != nil {
		return err
	}
	return manager.SaveAuthTokens(accessToken, refreshToken)
}

// LoadAccessToken retrieves the access token from the keychain.
func LoadAccessToken() (string, error) {
	manager, err := keychain.GetManager()
	if err != nil {
		return "", err
	}
	return manager.LoadAccessToken()
}

// ClearAuth removes all auth-related secrets from the keychain.
func ClearAuth() error {
	manager, err := keychain.GetManager()
	if err != nil {
		return err
	}
	return manager.ClearAuth()
}

// SaveAuthStateBytes stores serialized auth state in the keychain.
func SaveAuthStateBytes(data []byte) error {
	manager, err := keychain.GetManager()
	if err != nil {
		return err
	}
	return manager.SaveAuthState(data)
}

// LoadAuthStateBytes retrieves serialized auth state from the keychain.
func LoadAuthStateBytes() ([]byte, error) {
	manager, err := keychain.GetManager()
	if err != nil {
		return nil, err
	}
	return manager.LoadAuthState()
}

// ClearAuthState removes the stored auth state from the keychain.
func ClearAuthState() error {
	manager, err := keychain.GetManager()
	if err != nil {
		return err
	}
	return manager.ClearAuthState()
}

// SaveDBDSN stores the database DSN in the keychain.
func SaveDBDSN(dsn string) error {
	manager, err := keychain.GetManager()
	if err != nil {
		return err
	}
	return manager.SaveDBDSN(dsn)
}

// LoadDBDSN retrieves the database DSN from the keychain.
func LoadDBDSN() (string, error) {
	manager, err := keychain.GetManager()
	if err != nil {
		return "", err
	}
	return manager.LoadDBDSN()
}

// ClearDB removes DB-related secrets from the keychain.
func ClearDB() error {
	manager, err := keychain.GetManager()
	if err != nil {
		return err
	}
	return manager.ClearDB()
}
