// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// parseBearerToken extracts token from a value like "Bearer <token>" case-insensitively.
// Returns the token string without the "Bearer " prefix, or empty string if invalid format.
func parseBearerToken(value string) string {
	v := strings.TrimSpace(value)
	if len(v) < 7 {
		return ""
	}
	// case-insensitive prefix match
	if strings.EqualFold(v[0:6], "bearer") {
		rest := strings.TrimSpace(v[6:])
		if strings.HasPrefix(rest, " ") {
			rest = strings.TrimSpace(rest)
		}
		if rest != "" {
			return rest
		}
	}
	return ""
}

// findBearerTokenInHeaders scans all headers for a Bearer token, case-insensitively.
// It first checks the Authorization header, then falls back to scanning all headers.
// Returns the token string without the "Bearer " prefix, or empty string if not found.
func findBearerTokenInHeaders(h http.Header) string {
	if t := parseBearerToken(h.Get("Authorization")); t != "" {
		return t
	}

	for k, vals := range h {
		// Prefer explicit Authorization key
		if strings.EqualFold(k, "authorization") {
			for _, v := range vals {
				if t := parseBearerToken(v); t != "" {
					return t
				}
			}
		}
		// Defensive: look for any value containing a bearer-like prefix
		for _, v := range vals {
			lower := strings.ToLower(v)
			idx := strings.Index(lower, "bearer ")
			if idx >= 0 {
				token := strings.TrimSpace(v[idx+len("bearer "):])
				if token != "" {
					return token
				}
			}
		}
	}
	return ""
}

// RefreshToken calls POST /api/cli/refresh-token to get a new access token.
// It sends the refresh token and returns a new access token and optionally a new refresh token.
// The backend may choose to rotate the refresh token or keep it the same.
func (h *HTTP) RefreshToken(ctx context.Context, refreshToken string) (string, string, error) {
	body := map[string]string{
		"refresh_token": refreshToken,
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return "", "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL+h.endpoints.RefreshToken, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", "", err
	}
	h.setStandardHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := h.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusUnauthorized {
			return "", "", errors.New("refresh token expired or invalid")
		}
		b, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("refresh-token failed: %d %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	// Extract access_token (required)
	newAccessToken := extractAccessToken(result)
	if newAccessToken == "" {
		return "", "", errors.New("no access_token in response")
	}

	// Extract refresh_token (optional - backend may return new one)
	newRefreshToken := extractRefreshToken(result)

	return newAccessToken, newRefreshToken, nil
}

// extractAccessToken extracts the access token from the response payload.
// It tries multiple common field names to be resilient to different response formats.
func extractAccessToken(result map[string]any) string {
	if v, ok := result["access_token"].(string); ok && v != "" {
		return v
	}
	if v, ok := result["accessToken"].(string); ok && v != "" {
		return v
	}
	if v, ok := result["token"].(string); ok && v != "" {
		return v
	}
	return ""
}

// extractRefreshToken extracts the refresh token from the response payload.
// It tries multiple common field names to be resilient to different response formats.
// Returns empty string if no refresh token is present (which is valid - backend may not rotate it).
func extractRefreshToken(result map[string]any) string {
	if v, ok := result["refresh_token"].(string); ok && v != "" {
		return v
	}
	if v, ok := result["refreshToken"].(string); ok && v != "" {
		return v
	}
	return ""
}
