// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package cmd

import (
	"net/url"
	"os"
	"strings"

	"seedfast/cli/internal/auth"
	"seedfast/cli/internal/keychain"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

// dbinfoCmd represents the dbinfo command for displaying database connection information.
// It shows the current database connection string with the password masked for security.
var dbinfoCmd = &cobra.Command{
	Use:   "dbinfo",
	Short: "Show current database connection string",
	Long: `The dbinfo command displays the currently configured database connection string (DSN)
with the password masked for security. This helps verify which database you're connected to
without exposing sensitive credentials.

The password in the DSN will be replaced with *** for security.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		st, err := auth.Load()
		if err != nil || !st.LoggedIn {
			pterm.Println("❌ You need to be logged in to view database connection")
			pterm.Println("   Please run: seedfast login")
			return nil
		}

		// Try to get DSN from env vars first
		dsn := ""
		if env := os.Getenv("SEEDFAST_DSN"); strings.TrimSpace(env) != "" {
			dsn = strings.TrimSpace(env)
			pterm.Println("Using DSN from SEEDFAST_DSN environment variable")
			pterm.Println()
		} else if env := os.Getenv("DATABASE_URL"); strings.TrimSpace(env) != "" {
			dsn = strings.TrimSpace(env)
			pterm.Println("Using DSN from DATABASE_URL environment variable")
			pterm.Println()
		}

		// Fallback to keychain
		if strings.TrimSpace(dsn) == "" {
			km, err := keychain.GetManager()
			if err != nil {
				pterm.Println("❌ Secure storage is not available on this system")
				pterm.Println("   Keychain is only supported on macOS and Windows")
				return err
			}

			dsn, err = km.LoadDBDSN()
			if err != nil {
				pterm.Println("⚠️  No database connection configured")
				pterm.Println("   Please run: seedfast connect")
				return nil
			}

			if strings.TrimSpace(dsn) == "" {
				pterm.Println("⚠️  No database connection configured")
				pterm.Println("   Please run: seedfast connect")
				return nil
			}
			pterm.Println("Using DSN from OS keychain")
			pterm.Println()
		}

		// Mask the password in the DSN
		maskedDSN := maskPassword(dsn)

		// Display the connection info
		pterm.DefaultBox.
			WithTitle(pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint("Database Connection")).
			WithPadding(1).
			Println(maskedDSN)
		pterm.Println()
		pterm.Println("To update this connection, run: seedfast connect")
		pterm.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dbinfoCmd)
}

// maskPassword replaces the password in a PostgreSQL DSN with asterisks.
// It handles the format: postgres://user:password@host:5432/database?params
func maskPassword(dsn string) string {
	// Try to parse as URL
	u, err := url.Parse(dsn)
	if err != nil {
		// If parsing fails, do simple string replacement
		return maskPasswordSimple(dsn)
	}

	// Check if there's a password
	if u.User == nil {
		return dsn
	}

	_, hasPassword := u.User.Password()
	if !hasPassword {
		return dsn
	}

	// Replace password with asterisks
	username := u.User.Username()
	u.User = url.UserPassword(username, "***")

	return u.String()
}

// maskPasswordSimple performs simple string-based password masking for DSNs that don't parse as URLs.
func maskPasswordSimple(dsn string) string {
	// Look for pattern: user:password@
	atIndex := strings.Index(dsn, "@")
	if atIndex == -1 {
		return dsn
	}

	// Find the last colon before @
	beforeAt := dsn[:atIndex]
	colonIndex := strings.LastIndex(beforeAt, ":")

	if colonIndex == -1 {
		return dsn
	}

	// Check if there's a protocol before (like postgres://)
	protocolEnd := strings.Index(dsn, "://")
	if protocolEnd != -1 && colonIndex < protocolEnd+3 {
		// The colon is part of the protocol, not the password separator
		return dsn
	}

	// Replace password
	return dsn[:colonIndex+1] + "***" + dsn[atIndex:]
}
