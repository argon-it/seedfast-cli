// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package dsn

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

// PostgreSQLResolver handles PostgreSQL DSN parsing and normalization
type PostgreSQLResolver struct{}

// NewPostgreSQLResolver creates a new PostgreSQL resolver
func NewPostgreSQLResolver() *PostgreSQLResolver {
	return &PostgreSQLResolver{}
}

// Parse parses a PostgreSQL DSN string and returns normalized DSN info
func (r *PostgreSQLResolver) Parse(dsn string) (*DSNInfo, error) {
	if dsn == "" {
		return nil, NewParseError(dsn, "empty DSN", "provide a valid PostgreSQL connection string")
	}

	// Detect scheme (postgres:// or postgresql://)
	scheme := ""
	remainder := dsn
	if strings.HasPrefix(dsn, "postgresql://") {
		scheme = "postgresql"
		remainder = strings.TrimPrefix(dsn, "postgresql://")
	} else if strings.HasPrefix(dsn, "postgres://") {
		scheme = "postgres"
		remainder = strings.TrimPrefix(dsn, "postgres://")
	} else {
		return nil, NewParseError(dsn, "missing or invalid scheme", "use postgres:// or postgresql://")
	}

	// Try standard URL parsing first
	parsed, err := url.Parse(dsn)
	if err == nil && parsed.User != nil {
		// Standard parsing worked - extract info
		return r.extractFromURL(parsed, dsn)
	}

	// Standard parsing failed - likely due to special characters in password
	// Use manual parsing with regex
	return r.manualParse(scheme, remainder, dsn)
}

// extractFromURL extracts DSN info from a successfully parsed URL
func (r *PostgreSQLResolver) extractFromURL(parsed *url.URL, originalDSN string) (*DSNInfo, error) {
	info := &DSNInfo{
		Type:     DBTypePostgreSQL,
		Host:     parsed.Hostname(),
		Port:     parsed.Port(),
		User:     parsed.User.Username(),
		Database: strings.TrimSpace(strings.TrimPrefix(parsed.Path, "/")),
		Params:   make(map[string]string),
		Original: originalDSN,
	}

	password, _ := parsed.User.Password()
	info.Password = password

	// Extract query parameters
	for key, values := range parsed.Query() {
		if len(values) > 0 {
			info.Params[key] = values[0]
		}
	}

	if info.Port == "" {
		info.Port = "5432"
	}

	// Validate essential fields
	if strings.TrimSpace(info.User) == "" {
		return nil, NewParseError(originalDSN, "missing username", "provide username in format postgres://user:password@host/database")
	}
	if strings.TrimSpace(info.Host) == "" {
		return nil, NewParseError(originalDSN, "missing host", "provide host in format postgres://user:password@host/database")
	}
	if strings.TrimSpace(info.Database) == "" {
		return nil, NewParseError(originalDSN, "missing database name", "provide database in format postgres://user:password@host/database")
	}

	return info, nil
}

// manualParse manually parses a DSN when standard URL parsing fails
// This handles cases where special characters in password aren't URL-encoded
func (r *PostgreSQLResolver) manualParse(scheme, remainder, originalDSN string) (*DSNInfo, error) {
	// Pattern: [user[:password]@]host[:port]/database[?params]
	// We'll parse this step by step to handle unencoded special chars

	info := &DSNInfo{
		Type:     DBTypePostgreSQL,
		Port:     "5432",
		Params:   make(map[string]string),
		Original: originalDSN,
	}

	// Split by @ to separate auth and host
	atIndex := strings.Index(remainder, "@")
	if atIndex == -1 {
		return nil, NewParseError(originalDSN, "missing @ separator", "format should be postgres://user:password@host:port/database")
	}

	authPart := remainder[:atIndex]
	hostAndDB := remainder[atIndex+1:]

	// Parse auth part (user:password)
	colonIndex := strings.Index(authPart, ":")
	if colonIndex == -1 {
		info.User = authPart
		info.Password = ""
	} else {
		info.User = authPart[:colonIndex]
		info.Password = authPart[colonIndex+1:]
	}

	// Parse host and database
	// Format: host[:port]/database[?params]
	slashIndex := strings.Index(hostAndDB, "/")
	if slashIndex == -1 {
		return nil, NewParseError(originalDSN, "missing / before database name", "format should be postgres://user:password@host:port/database")
	}

	hostPart := hostAndDB[:slashIndex]
	dbAndParams := hostAndDB[slashIndex+1:]

	// Parse host:port
	if strings.Contains(hostPart, ":") {
		parts := strings.SplitN(hostPart, ":", 2)
		info.Host = parts[0]
		info.Port = parts[1]
	} else {
		info.Host = hostPart
	}

	// Parse database and params
	questionIndex := strings.Index(dbAndParams, "?")
	if questionIndex == -1 {
		info.Database = strings.TrimSpace(dbAndParams)
	} else {
		info.Database = strings.TrimSpace(dbAndParams[:questionIndex])
		paramStr := dbAndParams[questionIndex+1:]

		// Parse query parameters
		for _, param := range strings.Split(paramStr, "&") {
			if kv := strings.SplitN(param, "=", 2); len(kv) == 2 {
				info.Params[kv[0]] = kv[1]
			}
		}
	}

	// Validate essential fields
	if strings.TrimSpace(info.User) == "" {
		return nil, NewParseError(originalDSN, "missing username", "provide username in format postgres://user:password@host/database")
	}
	if strings.TrimSpace(info.Host) == "" {
		return nil, NewParseError(originalDSN, "missing host", "provide host in format postgres://user:password@host/database")
	}
	if strings.TrimSpace(info.Database) == "" {
		return nil, NewParseError(originalDSN, "missing database name", "provide database in format postgres://user:password@host/database")
	}

	return info, nil
}

// Normalize converts DSN info to a properly formatted connection string
func (r *PostgreSQLResolver) Normalize(info *DSNInfo) (string, error) {
	if info == nil {
		return "", NewParseError("", "nil DSN info", "")
	}

	// Build normalized DSN with proper URL encoding
	var builder strings.Builder

	// Use postgresql:// as canonical scheme
	builder.WriteString("postgresql://")

	// Encode username and password
	if info.User != "" {
		builder.WriteString(url.QueryEscape(info.User))
		if info.Password != "" {
			builder.WriteString(":")
			builder.WriteString(url.QueryEscape(info.Password))
		}
		builder.WriteString("@")
	}

	// Add host
	builder.WriteString(info.Host)

	// Add port if not default
	if info.Port != "" && info.Port != "5432" {
		builder.WriteString(":")
		builder.WriteString(info.Port)
	} else if info.Port == "5432" {
		builder.WriteString(":5432")
	}

	// Add database
	builder.WriteString("/")
	builder.WriteString(info.Database)

	// Add parameters
	if len(info.Params) > 0 {
		builder.WriteString("?")
		first := true
		for key, value := range info.Params {
			if !first {
				builder.WriteString("&")
			}
			builder.WriteString(url.QueryEscape(key))
			builder.WriteString("=")
			builder.WriteString(url.QueryEscape(value))
			first = false
		}
	}

	return builder.String(), nil
}

// Validate checks if the DSN is valid for PostgreSQL
func (r *PostgreSQLResolver) Validate(dsn string) error {
	info, err := r.Parse(dsn)
	if err != nil {
		return err
	}

	// Additional validation
	if info.Host == "" {
		return NewParseError(dsn, "empty host", "")
	}
	if info.Database == "" {
		return NewParseError(dsn, "empty database name", "")
	}
	if info.User == "" {
		return NewParseError(dsn, "empty username", "")
	}

	// Validate port is numeric if present
	if info.Port != "" {
		matched, _ := regexp.MatchString(`^\d+$`, info.Port)
		if !matched {
			return NewParseError(dsn, fmt.Sprintf("invalid port number: %s", info.Port), "port must be numeric")
		}
	}

	return nil
}
