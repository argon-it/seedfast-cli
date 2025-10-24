// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package cmd

import (
	"os"
	"strings"

	"seedfast/cli/internal/auth"
	"seedfast/cli/internal/dsn"
	"seedfast/cli/internal/keychain"
	"seedfast/cli/internal/logging"

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
		rawDSN := ""
		if env := os.Getenv("SEEDFAST_DSN"); strings.TrimSpace(env) != "" {
			rawDSN = strings.TrimSpace(env)
		} else if env := os.Getenv("DATABASE_URL"); strings.TrimSpace(env) != "" {
			rawDSN = strings.TrimSpace(env)
		}

		// Fallback to keychain
		if strings.TrimSpace(rawDSN) == "" {
			km, err := keychain.GetManager()
			if err != nil {
				pterm.Println("❌ Secure storage is not available on this system")
				pterm.Println("   Keychain is only supported on macOS and Windows")
				return err
			}

			rawDSN, err = km.LoadDBDSN()
			if err != nil {
				pterm.Println("⚠️  No database connection configured")
				pterm.Println("   Please run: seedfast connect")
				return nil
			}

			if strings.TrimSpace(rawDSN) == "" {
				pterm.Println("⚠️  No database connection configured")
				pterm.Println("   Please run: seedfast connect")
				return nil
			}
		}

		// Parse and normalize the DSN
		normalizedDSN, err := dsn.Parse(rawDSN)
		if err != nil {
			pterm.Println("❌ Invalid database connection string.")
			if parseErr, ok := err.(*dsn.ParseError); ok {
				pterm.Println("   " + parseErr.Error())
			}
			pterm.Println("   Please run 'seedfast connect' to reconfigure your database.")
			return err
		}

		// Mask the password and username in the DSN
		maskedDSN := logging.Mask(normalizedDSN)
		dbName := deriveDBName(normalizedDSN)

		// Display database connection info (same format as seed command)
		pterm.Println()
		pterm.Println(pterm.NewStyle(pterm.FgLightCyan).Sprint("→ Database:   ") + pterm.NewStyle(pterm.FgCyan, pterm.Bold).Sprint(dbName))
		pterm.Println(pterm.NewStyle(pterm.FgLightCyan).Sprint("→ Connection: ") + pterm.NewStyle(pterm.FgLightBlue).Sprint(maskedDSN))
		pterm.Println()

		return nil
	},
}

func init() {
	rootCmd.AddCommand(dbinfoCmd)
}
