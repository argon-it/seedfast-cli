// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"seedfast/cli/internal/manifest"
)

// HTTP implements API client over REST endpoints.
// It provides methods for authentication, user management, and version checking.
// User data is cached in memory to support offline scenarios and reduce API calls.
type HTTP struct {
	// baseURL is the base URL for all HTTP requests (e.g., "https://seedfa.st")
	baseURL string
	// endpoints contains the URL paths for various API endpoints
	endpoints manifest.HTTPEndpoints
	// client is the underlying HTTP client with configured timeout
	client *http.Client
	// meCache stores user data from /api/cli/me for offline access
	meCache map[string]any
	// meCacheTime tracks when the cache was last updated
	meCacheTime time.Time
}

// newHTTP creates a new HTTP client with the given base URL and endpoints.
// It configures a 15-second timeout for all requests.
func newHTTP(baseURL string, endpoints manifest.HTTPEndpoints) *HTTP {
	return &HTTP{
		baseURL:   strings.TrimRight(baseURL, "/"),
		endpoints: endpoints,
		client:    &http.Client{Timeout: 15 * time.Second},
	}
}

// setStandardHeaders adds standard headers to HTTP requests for better compatibility.
func (h *HTTP) setStandardHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "seedfast-cli/1.0")
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json, */*")
	}
}

// GetVersion calls GET /api/version and returns the backend version string when available.
// No authentication required. This can be used to check connectivity to the backend service.
func (h *HTTP) GetVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.baseURL+h.endpoints.Version, nil)
	if err != nil {
		return "", err
	}
	h.setStandardHeaders(req)

	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "unknown", nil
	}
	var out struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Version == "" {
		return "unknown", nil
	}
	return out.Version, nil
}

// GetCLIVersion calls GET /api/cli/version and returns the latest CLI version string.
// No authentication required. This is used to check if the user's CLI is up to date.
func (h *HTTP) GetCLIVersion(ctx context.Context) (string, error) {
	// If CLIVersion endpoint is not configured, return empty string
	if h.endpoints.CLIVersion == "" {
		return "", nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.baseURL+h.endpoints.CLIVersion, nil)
	if err != nil {
		return "", err
	}
	h.setStandardHeaders(req)

	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil
	}
	var out struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	return out.Version, nil
}
