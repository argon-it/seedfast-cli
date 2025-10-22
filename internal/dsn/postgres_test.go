// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package dsn

import (
	"strings"
	"testing"
)

func TestPostgreSQLResolver_Parse(t *testing.T) {
	resolver := NewPostgreSQLResolver()

	tests := []struct {
		name        string
		dsn         string
		wantUser    string
		wantHost    string
		wantPort    string
		wantDB      string
		wantPass    string
		wantParams  map[string]string
		expectError bool
	}{
		{
			name:     "standard postgres scheme",
			dsn:      "postgres://user:pass@localhost:5432/testdb",
			wantUser: "user",
			wantPass: "pass",
			wantHost: "localhost",
			wantPort: "5432",
			wantDB:   "testdb",
		},
		{
			name:     "postgresql scheme",
			dsn:      "postgresql://user:pass@localhost:5432/testdb",
			wantUser: "user",
			wantPass: "pass",
			wantHost: "localhost",
			wantPort: "5432",
			wantDB:   "testdb",
		},
		{
			name:     "password with special characters",
			dsn:      "postgres://postgres:r^NAbbi^Ym=mTi-tdcNuBjuc^7ENYJ@localhost:5432/lprx",
			wantUser: "postgres",
			wantPass: "r^NAbbi^Ym=mTi-tdcNuBjuc^7ENYJ",
			wantHost: "localhost",
			wantPort: "5432",
			wantDB:   "lprx",
		},
		{
			name:     "password with @ symbol",
			dsn:      "postgres://user:p@ssw0rd@example.com:5432/mydb",
			wantUser: "user",
			wantPass: "p@ssw0rd",
			wantHost: "example.com",
			wantPort: "5432",
			wantDB:   "mydb",
		},
		{
			name:     "password with : symbol",
			dsn:      "postgres://admin:p:ass:word@localhost:5432/db",
			wantUser: "admin",
			wantPass: "p:ass:word",
			wantHost: "localhost",
			wantPort: "5432",
			wantDB:   "db",
		},
		{
			name:     "default port omitted",
			dsn:      "postgres://user:pass@localhost/testdb",
			wantUser: "user",
			wantPass: "pass",
			wantHost: "localhost",
			wantPort: "5432",
			wantDB:   "testdb",
		},
		{
			name:     "with sslmode parameter",
			dsn:      "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
			wantUser: "user",
			wantPass: "pass",
			wantHost: "localhost",
			wantPort: "5432",
			wantDB:   "testdb",
			wantParams: map[string]string{
				"sslmode": "disable",
			},
		},
		{
			name:     "multiple parameters",
			dsn:      "postgres://user:pass@localhost:5432/testdb?sslmode=disable&connect_timeout=10",
			wantUser: "user",
			wantPass: "pass",
			wantHost: "localhost",
			wantPort: "5432",
			wantDB:   "testdb",
			wantParams: map[string]string{
				"sslmode":         "disable",
				"connect_timeout": "10",
			},
		},
		{
			name:        "empty DSN",
			dsn:         "",
			expectError: true,
		},
		{
			name:        "missing scheme",
			dsn:         "user:pass@localhost:5432/testdb",
			expectError: true,
		},
		{
			name:        "missing database",
			dsn:         "postgres://user:pass@localhost:5432/",
			expectError: true,
		},
		{
			name:        "missing host",
			dsn:         "postgres://user:pass@:5432/testdb",
			expectError: true,
		},
		{
			name:     "password without user should still work",
			dsn:      "postgres://testuser@localhost:5432/testdb",
			wantUser: "testuser",
			wantPass: "",
			wantHost: "localhost",
			wantPort: "5432",
			wantDB:   "testdb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := resolver.Parse(tt.dsn)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if info.User != tt.wantUser {
				t.Errorf("user = %q, want %q", info.User, tt.wantUser)
			}
			if info.Password != tt.wantPass {
				t.Errorf("password = %q, want %q", info.Password, tt.wantPass)
			}
			if info.Host != tt.wantHost {
				t.Errorf("host = %q, want %q", info.Host, tt.wantHost)
			}
			if info.Port != tt.wantPort {
				t.Errorf("port = %q, want %q", info.Port, tt.wantPort)
			}
			if info.Database != tt.wantDB {
				t.Errorf("database = %q, want %q", info.Database, tt.wantDB)
			}

			if tt.wantParams != nil {
				for key, wantVal := range tt.wantParams {
					gotVal, ok := info.Params[key]
					if !ok {
						t.Errorf("missing param %q", key)
					} else if gotVal != wantVal {
						t.Errorf("param %q = %q, want %q", key, gotVal, wantVal)
					}
				}
			}
		})
	}
}

func TestPostgreSQLResolver_Normalize(t *testing.T) {
	resolver := NewPostgreSQLResolver()

	tests := []struct {
		name        string
		input       string
		wantScheme  string
		wantEncoded bool // whether password should be URL-encoded
	}{
		{
			name:        "special characters in password",
			input:       "postgres://postgres:r^NAbbi^Ym=mTi-tdcNuBjuc^7ENYJ@localhost:5432/lprx",
			wantScheme:  "postgresql://",
			wantEncoded: true,
		},
		{
			name:        "standard password",
			input:       "postgres://user:password123@localhost:5432/testdb",
			wantScheme:  "postgresql://",
			wantEncoded: false,
		},
		{
			name:       "with parameters",
			input:      "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
			wantScheme: "postgresql://",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// First parse
			info, err := resolver.Parse(tt.input)
			if err != nil {
				t.Fatalf("parse failed: %v", err)
			}

			// Then normalize
			normalized, err := resolver.Normalize(info)
			if err != nil {
				t.Fatalf("normalize failed: %v", err)
			}

			// Check it uses canonical scheme
			if !strings.HasPrefix(normalized, tt.wantScheme) {
				t.Errorf("normalized DSN doesn't start with %q: %q", tt.wantScheme, normalized)
			}

			// Try parsing the normalized version - should not fail
			info2, err := resolver.Parse(normalized)
			if err != nil {
				t.Errorf("normalized DSN failed to parse: %v\nDSN: %s", err, normalized)
			}

			// Verify data integrity after normalization round-trip
			if info2.User != info.User {
				t.Errorf("user mismatch after normalization: %q != %q", info2.User, info.User)
			}
			if info2.Password != info.Password {
				t.Errorf("password mismatch after normalization: %q != %q", info2.Password, info.Password)
			}
			if info2.Host != info.Host {
				t.Errorf("host mismatch after normalization: %q != %q", info2.Host, info.Host)
			}
			if info2.Database != info.Database {
				t.Errorf("database mismatch after normalization: %q != %q", info2.Database, info.Database)
			}
		})
	}
}

func TestPostgreSQLResolver_Validate(t *testing.T) {
	resolver := NewPostgreSQLResolver()

	tests := []struct {
		name        string
		dsn         string
		expectError bool
	}{
		{
			name: "valid DSN",
			dsn:  "postgres://user:pass@localhost:5432/testdb",
		},
		{
			name: "valid with special chars",
			dsn:  "postgres://postgres:r^NAbbi^Ym=mTi-tdcNuBjuc^7ENYJ@localhost:5432/lprx",
		},
		{
			name:        "invalid port",
			dsn:         "postgres://user:pass@localhost:abc/testdb",
			expectError: true,
		},
		{
			name:        "missing database",
			dsn:         "postgres://user:pass@localhost:5432/",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := resolver.Validate(tt.dsn)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
