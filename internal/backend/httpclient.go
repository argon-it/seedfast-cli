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
// It configures a 10-second timeout for all requests.
func newHTTP(baseURL string, endpoints manifest.HTTPEndpoints) *HTTP {
	return &HTTP{
		baseURL:   strings.TrimRight(baseURL, "/"),
		endpoints: endpoints,
		client:    &http.Client{Timeout: 10 * time.Second},
	}
}

// GetVersion calls GET /api/version and returns the version string when available.
// No authentication required. This can be used to check connectivity to the backend service.
func (h *HTTP) GetVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.baseURL+h.endpoints.Version, nil)
	if err != nil {
		return "", err
	}
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
