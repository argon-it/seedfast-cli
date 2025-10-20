// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package cmd provides the command-line interface for the Seedfast CLI application.
// It implements various subcommands for database seeding, authentication, and configuration
// using the Cobra CLI framework. The package handles command parsing, execution, and
// provides a rich terminal UI with spinners and progress indicators.
package cmd

import (
	"context"
	"fmt"
	"os"

	"seedfast/cli/internal/backend"
	"seedfast/cli/internal/manifest"

	"github.com/spf13/cobra"
)

var (
	showVersion bool
)

// rootCmd represents the base command when called without any subcommands.
// It serves as the entry point for the Seedfast CLI application.
var rootCmd = &cobra.Command{
	Use:           "seedfast",
	Short:         "Seedfast CLI for database seeding via gRPC bridge",
	Long:          `Seedfast is a command-line tool for database seeding that connects to a backend service via gRPC bridge.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if showVersion {
			ctx := context.Background()
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

			// Check for CLI updates
			latestCLIVersion, err := be.GetCLIVersion(ctx)
			if err == nil && latestCLIVersion != "" && latestCLIVersion != Version {
				fmt.Println()
				fmt.Println("┌──────────────────────────────────────────────────────────┐")
				fmt.Printf("│ ⚠️  A new version of seedfast CLI is available: %-7s │\n", latestCLIVersion)
				fmt.Println("│                                                          │")
				fmt.Println("│ To update, run:                                          │")
				fmt.Println("│   brew upgrade argon-it/tap/seedfast                     │")
				fmt.Println("└──────────────────────────────────────────────────────────┘")
			}

			return nil
		}
		// If no flag is set, show help
		return cmd.Help()
	},
}

// Execute runs the CLI application.
// It executes the root command and handles any errors that occur during execution.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVar(&showVersion, "version", false, "Show CLI and backend version information")
}
