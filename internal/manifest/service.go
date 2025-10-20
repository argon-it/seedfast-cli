// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package manifest

import (
	"context"

	"seedfast/cli/internal/httperrors"
)

// GetEndpoints returns the manifest endpoints, using the RAM cache if available.
// If not cached, it fetches from the server and caches the result.
// This function is the main entry point for retrieving backend configuration.
func GetEndpoints(ctx context.Context) (*Manifest, error) {
	// Check RAM cache first
	if cached := GetCached(); cached != nil {
		return cached, nil
	}

	// Fetch from server
	manifest, err := fetchFromServer(ctx)
	if err != nil {
		return nil, httperrors.FormatNetworkError(err, "fetching server configuration")
	}

	// Cache in RAM for future calls within this process
	SetCached(manifest)

	return manifest, nil
}
