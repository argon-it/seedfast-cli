package backend

import (
	"seedfast/cli/internal/manifest"
)

// New creates a backend API implementation with manifest endpoints.
// Returns HTTP client (real backend).
func New(baseURL string, endpoints manifest.HTTPEndpoints) API {
	return newHTTP(baseURL, endpoints)
}
