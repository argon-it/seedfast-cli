// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package cmd provides the command-line interface for the Seedfast CLI application.
// It implements various subcommands for database seeding, authentication, and configuration
// using the Cobra CLI framework. The package handles command parsing, execution, and
// provides a rich terminal UI with spinners and progress indicators.
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands.
// It serves as the entry point for the Seedfast CLI application.
var rootCmd = &cobra.Command{
	Use:           "seedfast",
	Short:         "Seedfast CLI for database seeding via gRPC bridge",
	Long:          `Seedfast is a command-line tool for database seeding that connects to a backend service via gRPC bridge.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the CLI application.
// It executes the root command and handles any errors that occur during execution.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
