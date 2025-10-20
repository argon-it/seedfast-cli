// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package cmd

import (
	"fmt"
	"os"

	"seedfast/cli/internal/auth"
	"seedfast/cli/internal/manifest"

	"github.com/spf13/cobra"
)

var (
	verboseMe bool
)

// meCmd represents the me command for displaying current authentication state.
// It shows the currently authenticated account information by validating the current
// session with the backend service.
var meCmd = &cobra.Command{
	Use:   "me",
	Short: "Show current authenticated account",
	Long: `The me command displays information about the currently authenticated account.
It validates the current session by checking with the backend service and shows
the account identifier if authentication is valid.

If no valid session exists, it will indicate that the user is not logged in.
This command is useful for verifying authentication status before running
other commands that require authentication.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		// Enable verbose mode for all modules if --verbose is set
		if verboseMe {
			os.Setenv("SEEDFAST_VERBOSE", "1")
		}

		ctx := cmd.Context()
		st, err := auth.Load()
		if err != nil {
			// If auth state can't be loaded, treat as not logged in
			if verboseMe {
				fmt.Printf("[DEBUG] me: auth.Load() error: %v\n", err)
			}
			fmt.Println("🔒 You're not logged in yet!")
			fmt.Println("   Run 'seedfast login' to get started.")
			return nil
		}

		if verboseMe {
			fmt.Printf("[DEBUG] me: auth.Load() success - LoggedIn: %v, Account: %s\n", st.LoggedIn, st.Account)
		}

		// Check if not logged in
		if !st.LoggedIn {
			fmt.Println("🔒 You're not logged in yet!")
			fmt.Println("   Run 'seedfast login' to get started.")
			return nil
		}

		// Fetch manifest from server
		m, err := manifest.GetEndpoints(ctx)
		if err != nil {
			return err
		}

		svc := auth.NewService(m.HTTPBaseURL(), m.HTTP)

		// Try to get full user data with email
		userData, err := svc.GetUserData(ctx)
		if err == nil && userData != nil {
			// Try to extract email
			if email, ok := userData["email"].(string); ok && email != "" {
				fmt.Println(getMePhrase(email))
				return nil
			}
			// Fallback to user_id
			if userID, ok := userData["user_id"].(string); ok && userID != "" {
				fmt.Println(getMePhrase(userID))
				return nil
			}
			// Fallback to id
			if id, ok := userData["id"].(string); ok && id != "" {
				fmt.Println(getMePhrase(id))
				return nil
			}
		}

		// Fallback: Try WhoAmI
		if account, ok, err := svc.WhoAmI(ctx); err == nil && ok {
			fmt.Println(getMePhrase(account))
			return nil
		}

		// Final fallback to local state
		if st.LoggedIn && st.Account != "" {
			fmt.Println(getMePhrase(st.Account))
			return nil
		}

		fmt.Println("🔒 You're not logged in yet!")
		fmt.Println("   Run 'seedfast login' to get started.")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(meCmd)
	meCmd.Flags().BoolVarP(&verboseMe, "verbose", "v", false, "Enable verbose debug output")
}

// getMePhrase returns a friendly phrase with the user's identifier
func getMePhrase(identifier string) string {
	return fmt.Sprintf("👤 Current user: %s", identifier)
}
