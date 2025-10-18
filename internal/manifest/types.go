// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package manifest handles dynamic backend endpoint configuration.
package manifest

import (
	"net/url"
	"strings"
)

// Manifest represents the endpoint configuration from the server.
type Manifest struct {
	Version int           `json:"version"`
	GRPC    GRPCEndpoints `json:"grpc"`
	HTTP    HTTPEndpoints `json:"http"`
}

// GRPCEndpoints contains gRPC service addresses.
type GRPCEndpoints struct {
	Agent string `json:"agent_origin"` // Full URL with scheme (e.g., "https://agent.example.com" or "grpc://localhost:50051")
}

// HTTPEndpoints contains REST API endpoint paths.
type HTTPEndpoints struct {
	ConfirmDevice string `json:"device_confirm"` // e.g., "/api/cli/confirm-device"
	GetToken      string `json:"token_issue"`      // e.g., "/api/cli/get-token"
	GetLink       string `json:"device_get_link"`       // e.g., "/api/cli/get-link"
	RefreshToken  string `json:"token_refresh"`  // e.g., "/api/cli/refresh-token"
	Logout        string `json:"device_logout"`         // e.g., "/api/cli/logout"
	Me            string `json:"account_whoami"`             // e.g., "/api/cli/me"
	Health        string `json:"health"`         // e.g., "/api/health"
	Version       string `json:"version"`        // e.g., "/api/version"
}

// HTTPBaseURL extracts the base URL from the gRPC agent address.
// This assumes the HTTP API is hosted on the same domain as the gRPC service.
func (m *Manifest) HTTPBaseURL() string {
	u, err := url.Parse(m.GRPC.Agent)
	if err != nil {
		return ""
	}

	// Construct base URL from scheme + host
	scheme := u.Scheme
	if scheme == "grpc" || scheme == "grpcs" {
		// Map gRPC schemes to HTTP
		if scheme == "grpcs" {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

    host := u.Host
    if strings.HasPrefix(host, "agent.") {
        host = strings.TrimPrefix(host, "agent.")
    }

    base := scheme + "://" + host
	return strings.TrimRight(base, "/")
}

// GRPCAddress extracts the host:port from the agent URL.
func (m *Manifest) GRPCAddress() string {
	u, err := url.Parse(m.GRPC.Agent)
	if err != nil {
		return ""
	}
	return u.Host
}
