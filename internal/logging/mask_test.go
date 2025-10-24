// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package logging

import (
	"testing"
)

func TestMask(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "PostgreSQL DSN with username and password",
			input:    "postgresql://myuser:mypassword@localhost:5432/mydb",
			expected: "postgresql://*:*@localhost:5432/mydb",
		},
		{
			name:     "Postgres DSN with username and password",
			input:    "postgres://admin:Secret123@localhost/testdb",
			expected: "postgres://*:*@localhost/testdb",
		},
		{
			name:     "DSN with special characters in password",
			input:    "postgresql://user:P%40ssw0rd!@host:5432/db",
			expected: "postgresql://*:*@host:5432/db",
		},
		{
			name:     "Password parameter",
			input:    "password=secret123",
			expected: "password=***",
		},
		{
			name:     "Token",
			input:    "token=abc123xyz",
			expected: "token=***",
		},
		{
			name:     "API Key",
			input:    "apikey=sk_test_123456",
			expected: "apikey=***",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Mask(tt.input)
			if result != tt.expected {
				t.Errorf("Mask() = %v, want %v", result, tt.expected)
			}
		})
	}
}