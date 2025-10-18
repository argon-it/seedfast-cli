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
	once          sync.Once
)

// Manager provides centralized, thread-safe operations for the OS keychain.
type Manager struct {
	mu   sync.RWMutex
	ring keyring.Keyring
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
func GetManager() (*Manager, error) {
	var err error
	once.Do(func() {
		globalManager, err = NewManager()
	})
	if err != nil {
		return nil, err
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

// openRing opens the OS keyring using a restricted set of backends.
// Only macOS Keychain and Windows Credential Manager are allowed.
func openRing() (keyring.Keyring, error) {
	// Restrict to darwin/windows explicitly; return an error elsewhere.
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return nil, errors.New("secure storage not supported on this OS (macOS/Windows only)")
	}
	cfg := keyring.Config{
		ServiceName:     ServiceName,
		AllowedBackends: []keyring.BackendType{keyring.KeychainBackend, keyring.WinCredBackend},
	}
	// Hint prefixes where supported to minimize namespace collisions
	if runtime.GOOS == "windows" {
		cfg.WinCredPrefix = ServiceName
	}
	return keyring.Open(cfg)
}

// SaveAuthTokens stores access and refresh tokens in the OS keychain.
// This method is thread-safe.
func (m *Manager) SaveAuthTokens(accessToken, refreshToken string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

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

	return m.ring.Set(keyring.Item{Key: KeyAuthState, Data: data})
}

// LoadAuthState retrieves serialized auth state from the keychain.
// This method is thread-safe.
func (m *Manager) LoadAuthState() ([]byte, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

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

	_ = m.ring.Remove(KeyAuthState)
	return nil
}

// SaveDBDSN stores the database DSN in the keychain.
// This method is thread-safe.
func (m *Manager) SaveDBDSN(dsn string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.ring.Set(keyring.Item{Key: KeyDBDSN, Data: []byte(dsn)})
}

// LoadDBDSN retrieves the database DSN from the keychain.
// This method is thread-safe.
func (m *Manager) LoadDBDSN() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

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

	_ = m.ring.Remove(KeyDBDSN)
	return nil
}

// ClearAll removes all secrets from the keychain.
// This method is thread-safe and should be used with caution.
func (m *Manager) ClearAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	_ = m.ring.Remove(KeyAccessToken)
	_ = m.ring.Remove(KeyRefreshToken)
	_ = m.ring.Remove(KeyAuthState)
	_ = m.ring.Remove(KeyDBDSN)
	return nil
}
