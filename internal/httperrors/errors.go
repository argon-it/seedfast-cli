// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

// Package httperrors provides user-friendly error handling for HTTP requests.
package httperrors

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"syscall"

	"github.com/pterm/pterm"
)

// FormatNetworkError converts technical HTTP/network errors into user-friendly messages.
// It detects common error types (timeout, DNS, connection refused, SSL, server errors)
// and displays helpful troubleshooting information.
func FormatNetworkError(err error, context string) error {
	if err == nil {
		return nil
	}

	// Display user-friendly error message with pterm
	displayErrorMessage(err, context)

	// Return wrapped error for logging/debugging
	return fmt.Errorf("network error: %w", err)
}

// displayErrorMessage shows a formatted error message to the user based on error type.
func displayErrorMessage(err error, context string) {
	errStr := err.Error()

	// Check for specific error types
	if isTimeoutError(err) {
		showTimeoutError(context)
		return
	}

	if isDNSError(err) {
		showDNSError(context)
		return
	}

	if isConnectionRefusedError(err) {
		showConnectionRefusedError(context)
		return
	}

	if isSSLError(err) {
		showSSLError(context)
		return
	}

	if isServerError(errStr) {
		showServerError(context, errStr)
		return
	}

	// Generic network error
	showGenericError(context, errStr)
}

// isTimeoutError checks if the error is a timeout error.
func isTimeoutError(err error) bool {
	if err == nil {
		return false
	}

	// Check for timeout in error message
	errStr := strings.ToLower(err.Error())
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") {
		return true
	}

	// Check for net.Error with Timeout()
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	return false
}

// isDNSError checks if the error is a DNS resolution error.
func isDNSError(err error) bool {
	if err == nil {
		return false
	}

	var dnsErr *net.DNSError
	return errors.As(err, &dnsErr)
}

// isConnectionRefusedError checks if the error is a connection refused error.
func isConnectionRefusedError(err error) bool {
	if err == nil {
		return false
	}

	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return errors.Is(opErr.Err, syscall.ECONNREFUSED)
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "connection refused")
}

// isSSLError checks if the error is an SSL/TLS error.
func isSSLError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "tls") ||
		strings.Contains(errStr, "ssl") ||
		strings.Contains(errStr, "certificate") ||
		strings.Contains(errStr, "handshake")
}

// isServerError checks if the error indicates a server-side problem (5xx errors).
func isServerError(errStr string) bool {
	lower := strings.ToLower(errStr)
	return strings.Contains(lower, "500") ||
		strings.Contains(lower, "502") ||
		strings.Contains(lower, "503") ||
		strings.Contains(lower, "504") ||
		strings.Contains(lower, "internal server error") ||
		strings.Contains(lower, "bad gateway") ||
		strings.Contains(lower, "service unavailable") ||
		strings.Contains(lower, "gateway timeout")
}

// showTimeoutError displays a user-friendly timeout error message.
func showTimeoutError(context string) {
	pterm.Printf("⏱️  Connection timeout while %s\n", context)
	pterm.Println()
	pterm.Println("The server took too long to respond. This could mean:")
	pterm.Println("  • Slow internet connection")
	pterm.Println("  • Server is under heavy load")
	pterm.Println("  • Network firewall is blocking the connection")
	pterm.Println()
	pterm.Println("Please try again in a few moments.")
	pterm.Println()
}

// showDNSError displays a user-friendly DNS error message.
func showDNSError(context string) {
	pterm.Printf("🌐 Cannot resolve server address while %s\n", context)
	pterm.Println()
	pterm.Println("Unable to look up seedfa.st. Please check:")
	pterm.Println("  • Your internet connection is working")
	pterm.Println("  • DNS settings are correct")
	pterm.Println("  • No DNS-level blocking (corporate firewall, parental controls)")
	pterm.Println()
}

// showConnectionRefusedError displays a user-friendly connection refused error message.
func showConnectionRefusedError(context string) {
	pterm.Printf("🚫 Connection refused while %s\n", context)
	pterm.Println()
	pterm.Println("The server is not accepting connections. This could mean:")
	pterm.Println("  • The service is temporarily down")
	pterm.Println("  • Firewall is blocking the connection")
	pterm.Println("  • Wrong server address or port")
	pterm.Println()
	pterm.Println("Please try again later or contact support.")
	pterm.Println()
}

// showSSLError displays a user-friendly SSL/TLS error message.
func showSSLError(context string) {
	pterm.Printf("🔒 Secure connection failed while %s\n", context)
	pterm.Println()
	pterm.Println("Cannot establish a secure HTTPS connection. This could mean:")
	pterm.Println("  • SSL/TLS certificate issue")
	pterm.Println("  • Network proxy interfering with HTTPS")
	pterm.Println("  • System clock is incorrect")
	pterm.Println()
	pterm.Println("Try:")
	pterm.Println("  • Check your system date and time")
	pterm.Println("  • Verify network proxy settings")
	pterm.Println()
}

// showServerError displays a user-friendly server error message.
func showServerError(context string, errDetails string) {
	pterm.Printf("⚠️  Server error while %s\n", context)
	pterm.Println()
	pterm.Println("⚠️  The Seedfast server encountered an internal error.")
	pterm.Println()
	pterm.Println("This is not a problem with your setup. The issue is on our end.")
	pterm.Println("  • The service team has been notified")
	pterm.Println("  • Please try again in a few minutes")
	pterm.Println()

	// Show error details if available
	if strings.Contains(strings.ToLower(errDetails), "vercel") {
		pterm.Println("For status updates, visit: https://status.vercel.com")
		pterm.Println()
	}
}

// showGenericError displays a generic error message for unrecognized errors.
func showGenericError(context string, errDetails string) {
	pterm.Printf("❌ Cannot connect to Seedfast service while %s\n", context)
	pterm.Println()
	pterm.Println("Please check:")
	pterm.Println("  • Your internet connection")
	pterm.Println("  • Whether seedfa.st is accessible from your network")
	pterm.Println("  • Firewall settings that might block HTTPS requests")
	pterm.Println()

	// Show abbreviated error details for debugging
	if errDetails != "" {
		shortErr := errDetails
		if len(shortErr) > 100 {
			shortErr = shortErr[:100] + "..."
		}
		pterm.Debug.Printf("Technical details: %s\n", shortErr)
		pterm.Println()
	}
}

// ExtractHostFromURL extracts the hostname from a URL for error messages.
func ExtractHostFromURL(urlStr string) string {
	u, err := url.Parse(urlStr)
	if err != nil || u.Host == "" {
		return "server"
	}
	return u.Host
}
