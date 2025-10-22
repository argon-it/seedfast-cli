// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package dsn

import "fmt"

// DBType represents the type of database
type DBType string

const (
	DBTypePostgreSQL DBType = "postgresql"
	DBTypeMySQL      DBType = "mysql"
	DBTypeOracle     DBType = "oracle"
	DBTypeUnknown    DBType = "unknown"
)

// DSNInfo contains parsed information from a DSN string
type DSNInfo struct {
	Type     DBType
	Host     string
	Port     string
	User     string
	Password string
	Database string
	Params   map[string]string
	Original string
}

// String returns the normalized DSN string
func (d *DSNInfo) String() string {
	return d.Original
}

// Resolver is an interface for database-specific DSN resolution
type Resolver interface {
	// Parse parses a DSN string and returns normalized DSN info
	Parse(dsn string) (*DSNInfo, error)

	// Normalize converts DSN info to a properly formatted connection string
	Normalize(info *DSNInfo) (string, error)

	// Validate checks if the DSN is valid for the database type
	Validate(dsn string) error
}

// ParseError represents an error that occurred during DSN parsing
type ParseError struct {
	DSN     string
	Reason  string
	Hint    string
}

func (e *ParseError) Error() string {
	if e.Hint != "" {
		return fmt.Sprintf("invalid DSN format: %s\nHint: %s", e.Reason, e.Hint)
	}
	return fmt.Sprintf("invalid DSN format: %s", e.Reason)
}

// NewParseError creates a new ParseError
func NewParseError(dsn, reason, hint string) *ParseError {
	return &ParseError{
		DSN:    dsn,
		Reason: reason,
		Hint:   hint,
	}
}
