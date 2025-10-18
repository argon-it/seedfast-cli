package backend

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// BeginDeviceLink fetches a magic link from /api/cli/get-link.
// It initiates the device authorization flow by requesting a link and device code from the backend.
// Returns the magic link URL, device ID/code, polling interval in seconds, and any error.
func (h *HTTP) BeginDeviceLink(ctx context.Context) (string, string, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.baseURL+h.endpoints.GetLink, nil)
	if err != nil {
		return "", "", 0, err
	}
	resp, err := h.client.Do(req)
	if err != nil {
		return "", "", 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", "", 0, fmt.Errorf("get-link failed: %s", strings.TrimSpace(string(b)))
	}

	// Be liberal in what we accept: decode into a map first
	var raw map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return "", "", 0, err
	}

	link := extractLink(raw)
	if link == "" {
		return "", "", 0, errors.New("empty magic link")
	}

	deviceID := extractDeviceID(raw, link)
	return link, deviceID, 3, nil
}

// extractLink extracts the magic link from the response payload.
func extractLink(raw map[string]any) string {
	if v, ok := raw["link"].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// extractDeviceID extracts the device ID/code from various possible fields in the response.
// It tries multiple common field names and formats to be resilient to backend changes.
func extractDeviceID(raw map[string]any, link string) string {
	// Try common field names
	candidates := []string{
		"device_id", "deviceId", "code", "user_code", "userCode", "device_code", "deviceCode",
	}

	for _, key := range candidates {
		if v, ok := raw[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}

	// Fallback: parse from the link URL
	if deviceID := extractDeviceIDFromURL(link); deviceID != "" {
		return deviceID
	}

	return ""
}

// extractDeviceIDFromURL attempts to extract a device ID from query parameters or path segments.
func extractDeviceIDFromURL(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}

	// Try query parameters
	q := u.Query()
	for _, key := range []string{"device_id", "deviceId", "code"} {
		if v := q.Get(key); v != "" {
			return v
		}
	}

	// Fallback: use last non-empty path segment
	parts := strings.Split(u.Path, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		if p := strings.TrimSpace(parts[i]); p != "" {
			return p
		}
	}

	return ""
}

// PollDeviceLink posts to /api/cli/get-token with { device_id }.
// Returns empty tokens when not ready yet (pending authorization).
// Returns access token and refresh token when the device has been authorized.
func (h *HTTP) PollDeviceLink(ctx context.Context, deviceID string) (string, string, error) {
	// POST JSON strictly as per API: { "device_id": "<uuid>" }
	jsonBody := map[string]string{
		"device_id": deviceID,
	}
	if access, refresh, ok := h.tryVerifyPostJSON(ctx, jsonBody, parseTokensFromBody); ok {
		return access, refresh, nil
	}

	// Pending (not yet authorized)
	return "", "", nil
}

// parseTokensFromBody extracts access and refresh tokens from the response body.
// It supports both JSON responses (with nested structures) and plain text responses.
func parseTokensFromBody(r io.Reader, contentType string) (string, string) {
	lowerCT := strings.ToLower(contentType)
	if strings.Contains(lowerCT, "application/json") || strings.Contains(lowerCT, "+json") || contentType == "" {
		var anyBody any
		dec := json.NewDecoder(r)
		if err := dec.Decode(&anyBody); err == nil {
			var access, refresh string
			walkJSON(anyBody, &access, &refresh)
			return access, refresh
		}
	}

	// Fallback to plain text
	b, _ := io.ReadAll(r)
	token := strings.TrimSpace(string(b))
	if token != "" {
		return token, ""
	}
	return "", ""
}

// walkJSON recursively searches a JSON structure for access and refresh tokens.
// It handles various common field naming conventions.
func walkJSON(node any, access *string, refresh *string) {
	if *access != "" && *refresh != "" {
		return
	}

	switch v := node.(type) {
	case map[string]any:
		for k, vv := range v {
			lk := strings.ToLower(strings.ReplaceAll(k, "_", ""))
			if s, ok := vv.(string); ok {
				val := strings.TrimSpace(s)
				if *access == "" {
					if lk == "accesstoken" || lk == "access" || lk == "token" || lk == "bearer" {
						*access = val
					} else if lk == "authorization" {
						if t := parseBearerToken(val); t != "" {
							*access = t
						}
					}
				}
				if *refresh == "" {
					if lk == "refreshtoken" || lk == "refresh" {
						*refresh = val
					}
				}
			}
			if *access == "" || *refresh == "" {
				walkJSON(vv, access, refresh)
			}
		}
	case []any:
		for _, e := range v {
			if *access == "" || *refresh == "" {
				walkJSON(e, access, refresh)
			}
		}
	}
}

// tryVerifyPostJSON posts JSON to get-token and returns tokens when successful.
// It returns (access, refresh, true) on success, or ("", "", false) if pending or failed.
func (h *HTTP) tryVerifyPostJSON(ctx context.Context, body map[string]string, parse func(io.Reader, string) (string, string)) (string, string, bool) {
	b, err := json.Marshal(body)
	if err != nil {
		return "", "", false
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL+h.endpoints.GetToken, strings.NewReader(string(b)))
	if err != nil {
		return "", "", false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, */*")

	resp, err := h.client.Do(req)
	if err != nil {
		return "", "", false
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		// Check for token in headers first
		if token := findBearerTokenInHeaders(resp.Header); token != "" {
			return token, "", true
		}
		// Parse from body
		access, refresh := parse(resp.Body, resp.Header.Get("Content-Type"))
		if access != "" || refresh != "" {
			return access, refresh, true
		}
		return "", "", false
	case http.StatusNoContent, http.StatusAccepted, http.StatusNotFound, http.StatusBadRequest, http.StatusMethodNotAllowed, http.StatusUnsupportedMediaType:
		// Pending or invalid request
		return "", "", false
	default:
		return "", "", false
	}
}

// CheckDevice calls POST /api/cli/check-device with Authorization: Bearer <token>.
// It verifies the device authorization and returns the user ID if successful.
func (h *HTTP) CheckDevice(ctx context.Context, accessToken string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL+h.endpoints.ConfirmDevice, nil)
	if err != nil {
		return "", err
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		var out map[string]any
		if err := json.NewDecoder(resp.Body).Decode(&out); err == nil {
			if v, ok := out["user_id"].(string); ok && v != "" {
				return v, nil
			}
		}
		return "", errors.New("unexpected response")
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return "", errors.New("unauthorized")
	}

	b, _ := io.ReadAll(resp.Body)
	return "", fmt.Errorf("check-device failed: %d %s", resp.StatusCode, strings.TrimSpace(string(b)))
}

// Logout calls POST /api/cli/logout with Authorization header.
// It invalidates the current access token and clears cached user data.
func (h *HTTP) Logout(ctx context.Context, accessToken string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.baseURL+h.endpoints.Logout, nil)
	if err != nil {
		return err
	}
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Clear cache on logout
	h.meCache = nil
	h.meCacheTime = time.Time{}

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent {
		return nil
	}

	b, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("logout failed: %d %s", resp.StatusCode, strings.TrimSpace(string(b)))
}
