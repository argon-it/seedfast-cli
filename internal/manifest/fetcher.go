// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package manifest

import (
	"context"
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"time"
)

const manifestPublicKeyPEM = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwpuNyHuI8AWtqMzWeajV
wnhoVJ5otYLaeI5ND6GSscI6eKMBSpHWRK/AqA2fU2BQzw9K6kv+mNuFznEN89Zh
INloAIk2BK2weTdKjghuYncCjy0t6qSjuMcmv0YSJYMdPWc5V4D+SxkSjQjWiMRU
fLn9JG1KyGaZ3nhkXmhxHWSGE5tMlh/tZWYqtUX/rSAMxSEtXfEtIniBKZccIAsw
rhJhr/5caHVAbw1m3oPI44VvEV+huiV6MEeFrbEnYWWzCw/GxVwtwg7IYm7zB8oS
MVqf+mMUwoIzRW3qvx0cYeUtH20WEpncI4Dw+RNiBa3lee0YZRlXEuWCOk7nCwFC
fwIDAQAB
-----END PUBLIC KEY-----`

const manifestURL = "https://seedfa.st/cli-endpoints.json"

// fetchFromServer retrieves the manifest from the server with signature verification.
func fetchFromServer(ctx context.Context) (*Manifest, error) {
	client := &http.Client{Timeout: 15 * time.Second}

	req, err := http.NewRequestWithContext(ctx, "GET", manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add User-Agent header for better server compatibility
	req.Header.Set("User-Agent", "seedfast-cli/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Verify signature if header is present
	if sig := resp.Header.Get("X-Manifest-Signature"); sig != "" {
		if err := verifySignature(body, sig); err != nil {
			return nil, fmt.Errorf("signature verification failed: %w", err)
		}
	}

	// Parse JSON
	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest JSON: %w", err)
	}

	// Basic validation
	if manifest.Version == 0 {
		return nil, fmt.Errorf("invalid manifest: missing version field")
	}
	if manifest.GRPC.Agent == "" {
		return nil, fmt.Errorf("invalid manifest: missing grpc.agent field")
	}

	return &manifest, nil
}

// verifySignature validates the RSA-SHA256 signature of the manifest.
func verifySignature(body []byte, signatureB64 string) error {
	// Decode base64 signature
	sig, err := base64.StdEncoding.DecodeString(signatureB64)
	if err != nil {
		return fmt.Errorf("decode signature: %w", err)
	}

	// Parse public key
	block, _ := pem.Decode([]byte(manifestPublicKeyPEM))
	if block == nil {
		return fmt.Errorf("failed to parse PEM block")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("parse public key: %w", err)
	}

	rsaPubKey, ok := pubKey.(*rsa.PublicKey)
	if !ok {
		return fmt.Errorf("not an RSA public key")
	}

	// Compute SHA256 hash of body
	hash := sha256.Sum256(body)

	// Verify signature
	if err := rsa.VerifyPKCS1v15(rsaPubKey, crypto.SHA256, hash[:], sig); err != nil {
		return fmt.Errorf("signature mismatch: %w", err)
	}

	return nil
}
