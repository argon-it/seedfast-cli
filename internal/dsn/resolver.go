// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package dsn

import (
	"strings"
)

// DetectDBType detects the database type from a DSN string
func DetectDBType(dsn string) DBType {
	lower := strings.ToLower(dsn)

	if strings.HasPrefix(lower, "postgres://") || strings.HasPrefix(lower, "postgresql://") {
		return DBTypePostgreSQL
	}
	if strings.HasPrefix(lower, "mysql://") {
		return DBTypeMySQL
	}
	if strings.HasPrefix(lower, "oracle://") {
		return DBTypeOracle
	}

	return DBTypeUnknown
}

// Parse parses a DSN string and returns normalized connection string
// This is the main entry point for DSN parsing
func Parse(dsn string) (string, error) {
	if dsn == "" {
		return "", NewParseError(dsn, "empty DSN", "provide a valid database connection string")
	}

	dbType := DetectDBType(dsn)

	var resolver Resolver
	switch dbType {
	case DBTypePostgreSQL:
		resolver = NewPostgreSQLResolver()
	case DBTypeMySQL:
		return "", NewParseError(dsn, "MySQL support not yet implemented", "use PostgreSQL for now")
	case DBTypeOracle:
		return "", NewParseError(dsn, "Oracle support not yet implemented", "use PostgreSQL for now")
	default:
		return "", NewParseError(dsn, "unknown database type", "use postgres://, mysql://, or oracle://")
	}

	info, err := resolver.Parse(dsn)
	if err != nil {
		return "", err
	}

	normalized, err := resolver.Normalize(info)
	if err != nil {
		return "", err
	}

	return normalized, nil
}

// Validate validates a DSN string without normalizing it
func Validate(dsn string) error {
	if dsn == "" {
		return NewParseError(dsn, "empty DSN", "provide a valid database connection string")
	}

	dbType := DetectDBType(dsn)

	var resolver Resolver
	switch dbType {
	case DBTypePostgreSQL:
		resolver = NewPostgreSQLResolver()
	case DBTypeMySQL:
		return NewParseError(dsn, "MySQL support not yet implemented", "use PostgreSQL for now")
	case DBTypeOracle:
		return NewParseError(dsn, "Oracle support not yet implemented", "use PostgreSQL for now")
	default:
		return NewParseError(dsn, "unknown database type", "use postgres://, mysql://, or oracle://")
	}

	return resolver.Validate(dsn)
}

// ParseInfo parses a DSN string and returns detailed DSN info
// Useful for inspecting connection details
func ParseInfo(dsn string) (*DSNInfo, error) {
	if dsn == "" {
		return nil, NewParseError(dsn, "empty DSN", "provide a valid database connection string")
	}

	dbType := DetectDBType(dsn)

	var resolver Resolver
	switch dbType {
	case DBTypePostgreSQL:
		resolver = NewPostgreSQLResolver()
	case DBTypeMySQL:
		return nil, NewParseError(dsn, "MySQL support not yet implemented", "use PostgreSQL for now")
	case DBTypeOracle:
		return nil, NewParseError(dsn, "Oracle support not yet implemented", "use PostgreSQL for now")
	default:
		return nil, NewParseError(dsn, "unknown database type", "use postgres://, mysql://, or oracle://")
	}

	return resolver.Parse(dsn)
}
