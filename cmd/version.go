package cmd

import (
	"fmt"

	"seedfast/cli/internal/backend"
	"seedfast/cli/internal/manifest"

	"github.com/spf13/cobra"
)

var (
	// Version holds the CLI version information.
	// This value is typically set at build time using -ldflags.
	Version = "0.0.0-dev"
)

// versionCmd represents the version command for displaying version information.
// It shows both the CLI version and queries the backend service for its version.
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show CLI and backend version information",
	Long: `The version command displays version information for both the Seedfast CLI
and the connected backend service. The CLI version is compiled into the binary,
while the backend version is retrieved dynamically from the backend API.`,

	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		// Fetch manifest from server
		m, err := manifest.GetEndpoints(ctx)
		if err != nil {
			return err
		}

		be := backend.New(m.HTTPBaseURL(), m.HTTP)
		backendVersion, err := be.GetVersion(ctx)
		if err != nil {
			backendVersion = "unknown"
		}
		fmt.Printf("seedfast %s\nbackend %s\n", Version, backendVersion)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
