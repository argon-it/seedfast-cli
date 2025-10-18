package manifest

import (
	"context"
	"fmt"

	"github.com/pterm/pterm"
)

// GetEndpoints returns the manifest endpoints, using the RAM cache if available.
// If not cached, it fetches from the server and caches the result.
// This function is the main entry point for retrieving backend configuration.
func GetEndpoints(ctx context.Context) (*Manifest, error) {
	// Check RAM cache first
	if cached := GetCached(); cached != nil {
		return cached, nil
	}

	// Fetch from server
	manifest, err := fetchFromServer(ctx)
	if err != nil {
		return nil, formatServerError(err)
	}

	// Cache in RAM for future calls within this process
	SetCached(manifest)

	return manifest, nil
}

// formatServerError creates user-friendly error messages for manifest fetch failures.
func formatServerError(err error) error {
	pterm.Error.Println("Cannot connect to seedfa.st")
	pterm.Println()
	pterm.Info.Println("Please check:")
	pterm.Println("  • Your internet connection")
	pterm.Println("  • Whether seedfa.st is accessible from your network")
	pterm.Println("  • Firewall settings that might block HTTPS requests")
	pterm.Println()

	return fmt.Errorf("server unreachable: %w", err)
}
