// Package seeding defines event structures and rendering utilities used to display
// seeding progress in the CLI while bridging between the MCP (Model Context Protocol)
// and the backend service. It provides types for representing different stages of
// the seeding process and utilities for rendering progress information to the terminal.
//
// The package supports various event types including batch planning, progress updates,
// and table status changes, enabling rich terminal UI feedback during database seeding operations.
package seeding

// EventType enumerates known seeding event kinds.
type EventType string

const (
	// EventMCPLog carries a line of MCP-side log output.
	EventMCPLog EventType = "mcp_log"
	// EventSeedState describes a high-level seeding state change.
	EventSeedState EventType = "seed_state"
	// EventBatchPlan provides the planned batches and tables.
	EventBatchPlan EventType = "batch_plan"
	// EventBatchProgress updates which batch is running and indices.
	EventBatchProgress EventType = "batch_progress"
	// EventTableStatus updates a specific table status within a batch.
	EventTableStatus EventType = "table_status"
)

// Batch describes a batch with its tables.
type Batch struct {
	Name   string   `json:"name"`
	Tables []string `json:"tables"`
}

// Event is a generic container for seeding UI events.
// Only a subset of fields is set depending on Type.
type Event struct {
	Type EventType `json:"type"`

	// Common textual message (e.g., MCP log line or state label)
	Message string `json:"message,omitempty"`

	// Batch plan
	Batches []Batch `json:"batches,omitempty"`

	// Batch progress
	BatchIndex int      `json:"batch_index,omitempty"` // 1-based
	BatchTotal int      `json:"batch_total,omitempty"`
	BatchName  string   `json:"batch_name,omitempty"`
	Done       []string `json:"done,omitempty"`
	Queued     []string `json:"queued,omitempty"`

	// Table status
	TableBatch string `json:"table_batch,omitempty"`
	TableName  string `json:"table_name,omitempty"`
	TableState string `json:"table_state,omitempty"` // pending|running|done|error
}
