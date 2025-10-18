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

var (
	// flagConfigPath specifies the path to the configuration file.
	flagConfigPath string
	// flagLogLevel sets the logging level for the application.
	flagLogLevel string
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

// init initializes the root command with persistent flags.
// It adds configuration and logging level flags that can be used with all subcommands.
func init() {
	rootCmd.PersistentFlags().StringVar(&flagConfigPath, "config", "", "Path to the configuration file")
	rootCmd.PersistentFlags().StringVar(&flagLogLevel, "log-level", "info", "Set the logging level (debug, info, warn, error)")
}
