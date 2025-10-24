// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package logging provides utilities for secure logging and error presentation.
// It includes functions for masking sensitive information in log messages and
// formatting errors for user-friendly display while protecting credentials and secrets.
//
// The package helps ensure that sensitive data like passwords, tokens, and API keys
// are not accidentally exposed in logs or error messages shown to users.
package logging

import (
	"regexp"
	"strings"
)

var (
	rePassword = regexp.MustCompile(`(?i)(password=)([^\s;]+)`)
	reToken    = regexp.MustCompile(`(?i)(token=|bearer\s+)([A-Za-z0-9._-]+)`)
	reDSNPass  = regexp.MustCompile(`(?i)(://)([^:]+):([^@]+)(@)`) // postgres://user:pass@host
	reAPIKey   = regexp.MustCompile(`(?i)(apikey=|api_key=)([^\s;]+)`)
)

// Mask replaces sensitive values in the input string with "*".
// For DSN strings, both username and password are masked.
func Mask(s string) string {
	out := s
	out = rePassword.ReplaceAllString(out, "$1***")
	out = reToken.ReplaceAllString(out, "$1***")
	out = reDSNPass.ReplaceAllString(out, "$1*:*$4")
	out = reAPIKey.ReplaceAllString(out, "$1***")
	// Basic env-like pairs key=VALUE; mask common secret keys
	for _, k := range []string{"PGPASSWORD", "SUPABASE_TOKEN", "ACCESS_TOKEN"} {
		out = strings.ReplaceAll(out, k+"=", k+"=***")
	}
	return out
}
