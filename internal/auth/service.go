// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package auth provides authentication services for the Seedfast CLI.
// It manages the device authorization flow, token refresh, and session validation.
// Authentication state is persisted in local XDG config directory, while secrets
// (tokens, DSN) are stored in the OS keychain for security.
package auth

import (
	"context"

	"seedfast/cli/internal/backend"
	"seedfast/cli/internal/keychain"
	"seedfast/cli/internal/manifest"
)

// Service centralizes authentication-related operations against the backend
// and local secure storage/state.
type Service struct {
	be backend.API
}

// NewService constructs an auth Service using manifest endpoints.
func NewService(baseURL string, endpoints manifest.HTTPEndpoints) *Service {
	return &Service{be: backend.New(baseURL, endpoints)}
}

// StartLogin begins the device-link login flow.
func (s *Service) StartLogin(ctx context.Context) (authURL string, deviceID string, pollIntervalSeconds int, err error) {
	return s.be.BeginDeviceLink(ctx)
}

// PollLogin attempts to complete login for the given deviceID.
// When tokens are issued, they are saved to secure storage and local state is updated.
// Returns (account, true, nil) on success; (_, false, nil) if still pending.
func (s *Service) PollLogin(ctx context.Context, deviceID string) (string, bool, error) {
	access, refresh, err := s.be.PollDeviceLink(ctx, deviceID)
	if err != nil {
		return "", false, err
	}
	if access == "" {
		return "", false, nil
	}

	// Get keychain manager - this will initialize it if needed
	km, err := keychain.GetManager()
	if err != nil {
		return "", false, err
	}

	if err := km.SaveAuthTokens(access, refresh); err != nil {
		return "", false, err
	}
	userID := ""
	if uid, err := s.be.CheckDevice(ctx, access); err == nil && uid != "" {
		userID = uid
	}
	// Persist minimal state; we store user_id in Account for display
	if userID == "" {
		userID = "user"
	}
	_ = Save(State{LoggedIn: true, Account: userID})
	return userID, true, nil
}

// WhoAmI validates the current access token and returns the account when valid.
// First tries the new /api/cli/me endpoint (which has caching), then falls back
// to CheckDevice, and finally local state if offline.
// If token is expired, attempts to refresh. If refresh fails, logs out the user.
func (s *Service) WhoAmI(ctx context.Context) (string, bool, error) {
	// Try to get keychain manager - if it fails, user is not logged in
	km, err := keychain.GetManager()
	if err != nil {
		return "", false, nil // Keychain unavailable = not logged in
	}

	token, err := km.LoadAccessToken()
	if err == nil && token != "" {
		// Try new /api/cli/me endpoint first (supports caching)
		userData, meErr := s.be.GetMe(ctx, token)
		if meErr == nil && userData != nil {
			// Extract user identifier from various possible fields
			if uid, ok := userData["user_id"].(string); ok && uid != "" {
				return uid, true, nil
			}
			if uid, ok := userData["id"].(string); ok && uid != "" {
				return uid, true, nil
			}
			if email, ok := userData["email"].(string); ok && email != "" {
				return email, true, nil
			}
			// If userData exists but no identifier, still consider it valid
			return "user", true, nil
		}

		// If we got an unauthorized error, try to refresh the token
		if meErr != nil && meErr.Error() == "unauthorized" {
			if refreshed, _ := s.RefreshAccessToken(ctx); refreshed {
				// Retry with new token
				if newToken, err := keychain.MustGetManager().LoadAccessToken(); err == nil && newToken != "" {
					if userData, err := s.be.GetMe(ctx, newToken); err == nil && userData != nil {
						if uid, ok := userData["user_id"].(string); ok && uid != "" {
							return uid, true, nil
						}
						if uid, ok := userData["id"].(string); ok && uid != "" {
							return uid, true, nil
						}
						if email, ok := userData["email"].(string); ok && email != "" {
							return email, true, nil
						}
						return "user", true, nil
					}
				}
			} else {
				// Refresh failed - both tokens expired, logout automatically
				_ = s.ResetLocalAuth()
				return "", false, nil
			}
		}

		// Fallback to legacy CheckDevice endpoint
		if uid, err := s.be.CheckDevice(ctx, token); err == nil && uid != "" {
			return uid, true, nil
		}
	}
	// Final fallback: local state (for offline mode)
	st, err := Load()
	if err != nil {
		return "", false, err
	}
	if st.LoggedIn && st.Account != "" {
		return st.Account, true, nil
	}
	return "", false, nil
}

// Logout performs remote logout (best-effort) and clears local credentials/state.
func (s *Service) Logout(ctx context.Context) error {
	if token, err := keychain.MustGetManager().LoadAccessToken(); err == nil && token != "" {
		_ = s.be.Logout(ctx, token)
	}
	if err := keychain.MustGetManager().ClearAuth(); err != nil {
		return err
	}
	if err := Clear(); err != nil {
		return err
	}
	return nil
}

// ResetLocalAuth clears only local credentials/state (no remote calls).
func (s *Service) ResetLocalAuth() error {
	if err := keychain.MustGetManager().ClearAuth(); err != nil {
		return err
	}
	if err := Clear(); err != nil {
		return err
	}
	return nil
}

// RefreshAccessToken attempts to refresh the access token using the stored refresh token.
// If successful, updates the stored tokens in the keychain.
// Returns true if refresh was successful, false otherwise.
func (s *Service) RefreshAccessToken(ctx context.Context) (bool, error) {
	km := keychain.MustGetManager()

	// Load refresh token
	refreshToken, err := km.LoadRefreshToken()
	if err != nil || refreshToken == "" {
		return false, err
	}

	// Call backend to refresh
	newAccessToken, newRefreshToken, err := s.be.RefreshToken(ctx, refreshToken)
	if err != nil {
		return false, err
	}

	// Save new access token (always returned)
	if err := km.SaveAuthTokens(newAccessToken, ""); err != nil {
		return false, err
	}

	// If a new refresh token was provided, update it too
	if newRefreshToken != "" {
		if err := km.SaveAuthTokens("", newRefreshToken); err != nil {
			return false, err
		}
	}

	return true, nil
}

// GetAccessToken retrieves the current access token.
// Just returns the stored token without validation or refresh.
// For automatic token refresh, use GetValidAccessToken instead.
func (s *Service) GetAccessToken(ctx context.Context) (string, error) {
	token, err := keychain.MustGetManager().LoadAccessToken()
	if err != nil {
		return "", err
	}
	return token, nil
}

// GetValidAccessToken retrieves a valid access token, automatically refreshing if needed.
// If both access and refresh tokens are expired, clears auth state and returns an error.
func (s *Service) GetValidAccessToken(ctx context.Context) (string, error) {
	km := keychain.MustGetManager()
	token, err := km.LoadAccessToken()
	if err != nil {
		return "", err
	}

	// Quick validation: try to use the token
	if _, err := s.be.GetMe(ctx, token); err == nil {
		// Token is valid
		return token, nil
	} else if err.Error() == "unauthorized" {
		// Token expired, try to refresh
		if refreshed, _ := s.RefreshAccessToken(ctx); refreshed {
			// Get the new token
			if newToken, err := km.LoadAccessToken(); err == nil {
				return newToken, nil
			}
		}
		// Refresh failed - both tokens expired
		_ = s.ResetLocalAuth()
		return "", err
	}

	// Network error or other issue - return the token anyway
	return token, nil
}

// WarmCache pre-fetches user data from /api/cli/me to populate the cache.
// This is typically called right after successful login to enable offline whoami.
func (s *Service) WarmCache(ctx context.Context) error {
	token, err := keychain.MustGetManager().LoadAccessToken()
	if err != nil || token == "" {
		return err
	}
	// Call GetMe to populate cache (ignore result, we just want caching side-effect)
	_, _ = s.be.GetMe(ctx, token)
	return nil
}

// GetUserData retrieves full user data from the /api/cli/me endpoint.
// Returns a map with user fields like email, user_id, etc.
func (s *Service) GetUserData(ctx context.Context) (map[string]any, error) {
	token, err := keychain.MustGetManager().LoadAccessToken()
	if err != nil || token == "" {
		return nil, err
	}
	return s.be.GetMe(ctx, token)
}
