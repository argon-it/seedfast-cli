// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package keychain provides centralized, thread-safe keychain operations for seedfast.
// This module manages all interactions with the OS keychain/credential store,
// providing a unified interface for storing and retrieving sensitive data such as
// authentication tokens, database credentials, and other secrets.
//
// The package supports multiple operating systems including macOS Keychain and
// Windows Credential Manager, with thread-safe operations and proper error handling.
// It provides both individual credential management and bulk operations for
// authentication state and database connections.
package keychain

import (
	"errors"
	"runtime"
	"sync"

	"github.com/99designs/keyring"
)

// Global keychain manager instance
var (
	globalManager *Manager
	globalError   error
	mu            sync.Mutex
)

// Manager provides centralized, thread-safe operations for the OS keychain.
type Manager struct {
	mu      sync.RWMutex
	ring    keyring.Keyring
	backend keychainBackend
}

// keychainBackend defines the interface for keychain operations.
type keychainBackend interface {
	Set(key, value string) error
	Get(key string) (string, error)
	Delete(key string) error
}

// ServiceName identifies our keychain/credential store namespace.
const ServiceName = "seedfast"

// Keys used for storing secrets in the OS keychain.
const (
	KeyAccessToken  = "auth_access_token"
	KeyRefreshToken = "auth_refresh_token"
	KeyAuthState    = "auth_state"
	KeyDBDSN        = "db_dsn"
)

// NewManager creates a new keychain manager with the OS keyring initialized.
func NewManager() (*Manager, error) {
	// Try native security backend first on macOS
	if runtime.GOOS == "darwin" {
		backend, err := newSecurityBackend()
		if err == nil {
			return &Manager{backend: backend}, nil
		}
		// Fall through to keyring library if security command fails
	}

	ring, err := openRing()
	if err != nil {
		return nil, err
	}

	return &Manager{
		ring: ring,
	}, nil
}

// GetManager returns the global keychain manager instance.
// If not initialized, it will be created on first call.
// If initialization fails, it will retry on subsequent calls.
func GetManager() (*Manager, error) {
	mu.Lock()
	defer mu.Unlock()

	// If already initialized successfully, return it
	if globalManager != nil {
		return globalManager, nil
	}

	// If previous initialization failed, retry
	globalManager, globalError = NewManager()
	if globalError != nil {
		return nil, globalError
	}

	return globalManager, nil
}

// MustGetManager returns the global keychain manager instance.
// Panics if initialization fails. Use only when you're sure initialization will succeed.
func MustGetManager() *Manager {
	manager, err := GetManager()
	if err != nil {
		panic(err)
	}
	return manager
}

// openRing opens the OS keyring using native platform backends only.
// Forces use of macOS Keychain or Windows Credential Manager - no file fallback.
func openRing() (keyring.Keyring, error) {
	// Only support darwin/windows platforms
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return nil, errors.New("secure storage not supported on this OS (macOS/Windows only)")
	}

	// Use platform-specific native backends only
	var allowedBackends []keyring.BackendType
	if runtime.GOOS == "darwin" {
		// Try macOS Keychain first, then pass (password store) as fallback
		// Pass requires 'pass' utility installed: brew install pass
		allowedBackends = []keyring.BackendType{
			keyring.KeychainBackend,
			keyring.PassBackend,
		}
	} else if runtime.GOOS == "windows" {
		allowedBackends = []keyring.BackendType{keyring.WinCredBackend}
	}

	cfg := keyring.Config{
		ServiceName:     ServiceName,
		AllowedBackends: allowedBackends,
		PassPrefix:      ServiceName,
	}

	// Hint prefixes where supported to minimize namespace collisions
	if runtime.GOOS == "windows" {
		cfg.WinCredPrefix = ServiceName
	}

	ring, err := keyring.Open(cfg)
	if err != nil {
		if runtime.GOOS == "darwin" {
			return nil, errors.New("macOS Keychain unavailable. On macOS 26.0+, install 'pass': brew install pass gnupg && gpg --generate-key && pass init <gpg-key-id>")
		}
		return nil, err
	}

	return ring, nil
}

// SaveAuthTokens stores access and refresh tokens in the OS keychain.
// This method is thread-safe.
func (m *Manager) SaveAuthTokens(accessToken, refreshToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Use native backend if available
	if m.backend != nil {
		if accessToken != "" {
			if err := m.backend.Set(KeyAccessToken, accessToken); err != nil {
				return err
			}
		}
		if refreshToken != "" {
			if err := m.backend.Set(KeyRefreshToken, refreshToken); err != nil {
				return err
			}
		}
		return nil
	}

	// Fallback to keyring library
	if accessToken != "" {
		if err := m.ring.Set(keyring.Item{Key: KeyAccessToken, Data: []byte(accessToken)}); err != nil {
			return err
		}
	}
	if refreshToken != "" {
		if err := m.ring.Set(keyring.Item{Key: KeyRefreshToken, Data: []byte(refreshToken)}); err != nil {
			return err
		}
	}
	return nil
}

// LoadAccessToken retrieves the access token from the keychain.
// This method is thread-safe.
func (m *Manager) LoadAccessToken() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Use native backend if available
	if m.backend != nil {
		token, err := m.backend.Get(KeyAccessToken)
		if err != nil {
			return "", err
		}
		if token == "" {
			return "", errors.New("empty access token")
		}
		return token, nil
	}

	// Fallback to keyring library
	it, err := m.ring.Get(KeyAccessToken)
	if err != nil {
		return "", err
	}
	if len(it.Data) == 0 {
		return "", errors.New("empty access token")
	}
	return string(it.Data), nil
}

// LoadRefreshToken retrieves the refresh token from the keychain.
// This method is thread-safe.
func (m *Manager) LoadRefreshToken() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.backend != nil {
		token, err := m.backend.Get(KeyRefreshToken)
		if err != nil {
			return "", err
		}
		if token == "" {
			return "", errors.New("empty refresh token")
		}
		return token, nil
	}

	it, err := m.ring.Get(KeyRefreshToken)
	if err != nil {
		return "", err
	}
	if len(it.Data) == 0 {
		return "", errors.New("empty refresh token")
	}
	return string(it.Data), nil
}

// ClearAuth removes all auth-related secrets from the keychain.
// This method is thread-safe.
func (m *Manager) ClearAuth() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.backend != nil {
		_ = m.backend.Delete(KeyAccessToken)
		_ = m.backend.Delete(KeyRefreshToken)
		_ = m.backend.Delete(KeyAuthState)
		return nil
	}

	_ = m.ring.Remove(KeyAccessToken)
	_ = m.ring.Remove(KeyRefreshToken)
	_ = m.ring.Remove(KeyAuthState)
	return nil
}

// SaveAuthState stores serialized auth state in the keychain.
// This method is thread-safe.
func (m *Manager) SaveAuthState(data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.backend != nil {
		return m.backend.Set(KeyAuthState, string(data))
	}

	return m.ring.Set(keyring.Item{Key: KeyAuthState, Data: data})
}

// LoadAuthState retrieves serialized auth state from the keychain.
// This method is thread-safe.
func (m *Manager) LoadAuthState() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.backend != nil {
		data, err := m.backend.Get(KeyAuthState)
		if err != nil {
			return nil, err
		}
		return []byte(data), nil
	}

	it, err := m.ring.Get(KeyAuthState)
	if err != nil {
		return nil, err
	}
	return it.Data, nil
}

// ClearAuthState removes the stored auth state from the keychain.
// This method is thread-safe.
func (m *Manager) ClearAuthState() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.backend != nil {
		_ = m.backend.Delete(KeyAuthState)
		return nil
	}

	_ = m.ring.Remove(KeyAuthState)
	return nil
}

// SaveDBDSN stores the database DSN in the keychain.
// This method is thread-safe.
func (m *Manager) SaveDBDSN(dsn string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.backend != nil {
		return m.backend.Set(KeyDBDSN, dsn)
	}

	return m.ring.Set(keyring.Item{Key: KeyDBDSN, Data: []byte(dsn)})
}

// LoadDBDSN retrieves the database DSN from the keychain.
// This method is thread-safe.
func (m *Manager) LoadDBDSN() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.backend != nil {
		return m.backend.Get(KeyDBDSN)
	}

	it, err := m.ring.Get(KeyDBDSN)
	if err != nil {
		return "", err
	}
	return string(it.Data), nil
}

// ClearDB removes DB-related secrets from the keychain.
// This method is thread-safe.
func (m *Manager) ClearDB() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.backend != nil {
		_ = m.backend.Delete(KeyDBDSN)
		return nil
	}

	_ = m.ring.Remove(KeyDBDSN)
	return nil
}

// ClearAll removes all secrets from the keychain.
// This method is thread-safe and should be used with caution.
func (m *Manager) ClearAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.backend != nil {
		_ = m.backend.Delete(KeyAccessToken)
		_ = m.backend.Delete(KeyRefreshToken)
		_ = m.backend.Delete(KeyAuthState)
		_ = m.backend.Delete(KeyDBDSN)
		return nil
	}

	_ = m.ring.Remove(KeyAccessToken)
	_ = m.ring.Remove(KeyRefreshToken)
	_ = m.ring.Remove(KeyAuthState)
	_ = m.ring.Remove(KeyDBDSN)
	return nil
}
