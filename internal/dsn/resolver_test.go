// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package dsn

import (
	"testing"
)

func TestDetectDBType(t *testing.T) {
	tests := []struct {
		name string
		dsn  string
		want DBType
	}{
		{
			name: "postgres scheme",
			dsn:  "postgres://user:pass@localhost/db",
			want: DBTypePostgreSQL,
		},
		{
			name: "postgresql scheme",
			dsn:  "postgresql://user:pass@localhost/db",
			want: DBTypePostgreSQL,
		},
		{
			name: "postgres uppercase",
			dsn:  "POSTGRES://user:pass@localhost/db",
			want: DBTypePostgreSQL,
		},
		{
			name: "mysql scheme",
			dsn:  "mysql://user:pass@localhost/db",
			want: DBTypeMySQL,
		},
		{
			name: "oracle scheme",
			dsn:  "oracle://user:pass@localhost/db",
			want: DBTypeOracle,
		},
		{
			name: "unknown scheme",
			dsn:  "http://example.com",
			want: DBTypeUnknown,
		},
		{
			name: "no scheme",
			dsn:  "user:pass@localhost/db",
			want: DBTypeUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectDBType(tt.dsn)
			if got != tt.want {
				t.Errorf("DetectDBType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := []struct {
		name        string
		dsn         string
		expectError bool
	}{
		{
			name: "valid postgres DSN",
			dsn:  "postgres://user:pass@localhost:5432/testdb",
		},
		{
			name: "valid postgres with special chars",
			dsn:  "postgres://postgres:r^NAbbi^Ym=mTi-tdcNuBjuc^7ENYJ@localhost:5432/lprx",
		},
		{
			name:        "empty DSN",
			dsn:         "",
			expectError: true,
		},
		{
			name:        "MySQL not yet supported",
			dsn:         "mysql://user:pass@localhost/db",
			expectError: true,
		},
		{
			name:        "Oracle not yet supported",
			dsn:         "oracle://user:pass@localhost/db",
			expectError: true,
		},
		{
			name:        "unknown database type",
			dsn:         "mongodb://localhost/db",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized, err := Parse(tt.dsn)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if normalized == "" {
				t.Error("normalized DSN is empty")
			}

			// Verify normalized DSN can be parsed again
			_, err = Parse(normalized)
			if err != nil {
				t.Errorf("normalized DSN failed to parse: %v", err)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name        string
		dsn         string
		expectError bool
	}{
		{
			name: "valid postgres DSN",
			dsn:  "postgres://user:pass@localhost:5432/testdb",
		},
		{
			name:        "invalid postgres DSN",
			dsn:         "postgres://localhost",
			expectError: true,
		},
		{
			name:        "empty DSN",
			dsn:         "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Validate(tt.dsn)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestParseInfo(t *testing.T) {
	dsn := "postgres://testuser:testpass@testhost:5555/testdb?sslmode=require"

	info, err := ParseInfo(dsn)
	if err != nil {
		t.Fatalf("ParseInfo() error = %v", err)
	}

	if info.Type != DBTypePostgreSQL {
		t.Errorf("Type = %v, want %v", info.Type, DBTypePostgreSQL)
	}
	if info.User != "testuser" {
		t.Errorf("User = %v, want testuser", info.User)
	}
	if info.Password != "testpass" {
		t.Errorf("Password = %v, want testpass", info.Password)
	}
	if info.Host != "testhost" {
		t.Errorf("Host = %v, want testhost", info.Host)
	}
	if info.Port != "5555" {
		t.Errorf("Port = %v, want 5555", info.Port)
	}
	if info.Database != "testdb" {
		t.Errorf("Database = %v, want testdb", info.Database)
	}
	if info.Params["sslmode"] != "require" {
		t.Errorf("Params[sslmode] = %v, want require", info.Params["sslmode"])
	}
}
