// Package main is the entry point for the Seedfast CLI application.
// It provides database seeding capabilities through a gRPC bridge interface.
package main

import (
	"seedfast/cli/cmd"
)

// main is the entry point for the Seedfast CLI application.
// It initializes and executes the command-line interface.
func main() {
	cmd.Execute()
}
