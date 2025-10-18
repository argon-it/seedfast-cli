// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package backend provides interfaces and implementations for communicating with the Seedfast backend service.
// It defines the API contract for authentication, version checking, and other backend operations.
// The package includes both interface definitions and HTTP-based implementations.
package backend

import "context"

// API defines backend operations the CLI depends on.
// Implementations may call real HTTP/WS endpoints or provide mocks for tests.
type API interface {
	GetVersion(ctx context.Context) (string, error)
	BeginDeviceLink(ctx context.Context) (authURL string, userCode string, pollIntervalSeconds int, err error)
	PollDeviceLink(ctx context.Context, userCode string) (accessToken string, refreshToken string, err error)
	// CheckDevice validates the current access token with the backend and
	// returns the associated user identifier when available.
	CheckDevice(ctx context.Context, accessToken string) (userID string, err error)
	// Logout invalidates the current access token on the backend.
	Logout(ctx context.Context, accessToken string) error
	// GetMe retrieves the current user's information from the backend.
	// Returns user data as a map containing fields like user_id, email, etc.
	GetMe(ctx context.Context, accessToken string) (map[string]any, error)
	// RefreshToken exchanges a refresh token for a new access token.
	// Returns new access token and optionally a new refresh token.
	RefreshToken(ctx context.Context, refreshToken string) (newAccessToken string, newRefreshToken string, err error)
}
