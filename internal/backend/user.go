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
	"time"
)

// GetMe calls GET /api/cli/me with Authorization header.
// Results are cached in memory for 10 minutes to support offline mode and reduce API calls.
// Returns user data as a map, or an error if the request fails and no cached data is available.
func (h *HTTP) GetMe(ctx context.Context, accessToken string) (map[string]any, error) {
	// Check cache first (10 minute TTL)
	if h.meCache != nil && time.Since(h.meCacheTime) < 10*time.Minute {
		return h.meCache, nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.baseURL+h.endpoints.Me, nil)
	if err != nil {
		// If we have cached data, return it even if request creation fails
		if h.meCache != nil {
			return h.meCache, nil
		}
		return nil, err
	}
	h.setStandardHeaders(req)
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		// Network error - return cached data if available
		if h.meCache != nil {
			return h.meCache, nil
		}
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Non-OK status - return cached data if available
		if h.meCache != nil {
			return h.meCache, nil
		}
		if resp.StatusCode == http.StatusUnauthorized {
			return nil, errors.New("unauthorized")
		}
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get-me failed: %d %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var userData map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&userData); err != nil {
		// Decode error - return cached data if available
		if h.meCache != nil {
			return h.meCache, nil
		}
		return nil, err
	}

	// Update cache
	h.meCache = userData
	h.meCacheTime = time.Now()

	return userData, nil
}
