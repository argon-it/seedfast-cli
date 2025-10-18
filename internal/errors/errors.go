// Package errors defines typed errors with categories for user-friendly reporting.
// It provides a structured approach to error handling with machine-readable error kinds
// and human-friendly messages. This enables better error categorization, logging,
// and user experience by providing context-aware error information.
//
// The package supports wrapping underlying errors while maintaining error kind information,
// making it easier to handle different types of failures appropriately.
package errors

import "fmt"

// Kind is a machine-readable error category.
type Kind string

const (
	// HandshakeFailed indicates MCP handshake failure.
	HandshakeFailed Kind = "handshake_failed"
	// HeartbeatFailed indicates heartbeat verification failure.
	HeartbeatFailed Kind = "heartbeat_failed"
	// MCPStartFailed indicates MCP process failed to start.
	MCPStartFailed Kind = "mcp_start_failed"
)

// E wraps an error with kind and human-friendly message.
type E struct {
	Kind    Kind
	Message string
	Err     error
}

func (e *E) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Kind, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Kind, e.Message)
}

func Wrap(kind Kind, msg string, err error) *E { return &E{Kind: kind, Message: msg, Err: err} }
func New(kind Kind, msg string) *E             { return &E{Kind: kind, Message: msg} }
