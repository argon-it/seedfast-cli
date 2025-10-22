// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package logging

import (
	"fmt"
	"strings"

	"github.com/pterm/pterm"
)

// GRPCErrorType represents the category of gRPC error
type GRPCErrorType int

const (
	GRPCErrorUnknown GRPCErrorType = iota
	GRPCErrorNetwork
	GRPCErrorAuth
	GRPCErrorTimeout
	GRPCErrorInternal
	GRPCErrorUnavailable
)

// ParseGRPCError categorizes a gRPC error message
func ParseGRPCError(errMsg string) GRPCErrorType {
	lower := strings.ToLower(errMsg)

	// Check for specific error patterns
	if strings.Contains(lower, "rst_stream") || strings.Contains(lower, "connection reset") {
		return GRPCErrorNetwork
	}
	if strings.Contains(lower, "internal_error") {
		return GRPCErrorInternal
	}
	if strings.Contains(lower, "unavailable") || strings.Contains(lower, "service unavailable") {
		return GRPCErrorUnavailable
	}
	if strings.Contains(lower, "deadline") || strings.Contains(lower, "timeout") {
		return GRPCErrorTimeout
	}
	if strings.Contains(lower, "unauthenticated") || strings.Contains(lower, "unauthorized") {
		return GRPCErrorAuth
	}

	return GRPCErrorUnknown
}

// FormatStreamError formats a gRPC stream error in a user-friendly way
func FormatStreamError(errMsg string) string {
	errType := ParseGRPCError(errMsg)

	var builder strings.Builder

	// Title
	builder.WriteString(pterm.NewStyle(pterm.FgRed, pterm.Bold).Sprint("Connection Lost"))
	builder.WriteString("\n\n")

	// User-friendly description
	switch errType {
	case GRPCErrorNetwork:
		builder.WriteString("The connection to Seedfast was interrupted unexpectedly.\n")
		builder.WriteString("This usually happens when:\n")
		builder.WriteString("  • Your internet connection was disrupted\n")
		builder.WriteString("  • The network path to the service was interrupted\n")
		builder.WriteString("  • A firewall or proxy closed the connection\n")

	case GRPCErrorInternal:
		builder.WriteString("An internal error occurred on the Seedfast service.\n")
		builder.WriteString("This could mean:\n")
		builder.WriteString("  • The service encountered an unexpected issue\n")
		builder.WriteString("  • The service is being updated or restarted\n")
		builder.WriteString("  • There was a temporary problem processing your request\n")

	case GRPCErrorUnavailable:
		builder.WriteString("The Seedfast service is currently unavailable.\n")
		builder.WriteString("Possible reasons:\n")
		builder.WriteString("  • The service is under maintenance\n")
		builder.WriteString("  • The service is temporarily overloaded\n")
		builder.WriteString("  • There's a service outage\n")

	case GRPCErrorTimeout:
		builder.WriteString("The connection to Seedfast timed out.\n")
		builder.WriteString("This could be due to:\n")
		builder.WriteString("  • Slow or unstable internet connection\n")
		builder.WriteString("  • The service taking too long to respond\n")
		builder.WriteString("  • Network latency issues\n")

	case GRPCErrorAuth:
		builder.WriteString("Authentication with Seedfast failed.\n")
		builder.WriteString("To fix this:\n")
		builder.WriteString("  • Run 'seedfast login' to authenticate again\n")
		builder.WriteString("  • Your session may have expired\n")

	default:
		builder.WriteString("The seeding session was interrupted.\n")
		builder.WriteString("This could mean:\n")
		builder.WriteString("  • Network connection dropped\n")
		builder.WriteString("  • Service is restarting or under maintenance\n")
		builder.WriteString("  • Session timeout\n")
	}

	builder.WriteString("\n")

	// Action to take
	if errType == GRPCErrorAuth {
		builder.WriteString(pterm.NewStyle(pterm.FgYellow).Sprint("→ Please run 'seedfast login' and try again"))
	} else {
		builder.WriteString(pterm.NewStyle(pterm.FgYellow).Sprint("→ Please try running 'seedfast seed' again"))
	}

	builder.WriteString("\n")

	// Technical details (optional, for debugging)
	if strings.TrimSpace(errMsg) != "" {
		builder.WriteString("\n")
		builder.WriteString(pterm.NewStyle(pterm.FgGray).Sprint("Technical details: " + errMsg))
	}

	return builder.String()
}

// PresentStreamError displays a formatted stream error
func PresentStreamError(errMsg string) {
	fmt.Println()
	fmt.Println(FormatStreamError(errMsg))
	fmt.Println()
}
