// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package cmd

import (
	"fmt"

	"seedfast/cli/internal/auth"
	"seedfast/cli/internal/keychain"
	"seedfast/cli/internal/manifest"

	"github.com/spf13/cobra"
)

// logoutCmd represents the logout command for clearing authentication state.
// It removes all saved credentials, tokens, and authentication state from both
// the local system and the backend service (best-effort remote logout).
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove all saved credentials and tokens",
	Long: `The logout command clears all authentication state from the local system,
including access tokens, refresh tokens, and user state. It also attempts to
notify the backend service to invalidate the current session (best-effort).

This command removes:
- Authentication tokens from the OS keychain
- Local authentication state
- Database connection credentials
- Any cached session information`,

	RunE: func(cmd *cobra.Command, args []string) error {
		// Try to logout from backend (best effort - don't fail if offline)
		m, err := manifest.GetEndpoints(cmd.Context())
		if err == nil {
			svc := auth.NewService(m.HTTPBaseURL(), m.HTTP)
			_ = svc.Logout(cmd.Context()) // Ignore error - best effort
		}

		// Always clear local credentials regardless of backend response
		if km, err := keychain.GetManager(); err == nil {
			_ = km.ClearAuth()
			_ = km.ClearDB()
		}
		_ = auth.Clear()

		fmt.Println("✅ All credentials and tokens have been removed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(logoutCmd)
}
