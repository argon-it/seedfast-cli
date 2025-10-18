// Package seeding provides state management and coordination for database seeding operations.
package seeding

import (
	"sync"
	"unicode/utf8"
)

// ProgressState tracks the seeding progress for all tables in the current session.
// It maintains information about which tables are active, completed, or failed.
type ProgressState struct {
	// Active maps table names to their remaining task count
	Active map[string]int
	// Completed contains the set of successfully seeded tables
	Completed map[string]struct{}
	// Failed maps table names to failure reasons
	Failed map[string]string
	// Order preserves the sequence in which tables were started
	Order []string
	// Expected contains the set of tables that are planned to be seeded
	Expected map[string]struct{}
	// DoneTables tracks successfully completed tables in order of completion
	DoneTables []string
	// mu protects concurrent access to all fields
	mu sync.Mutex
}

// NewProgressState creates a new ProgressState with initialized maps.
func NewProgressState() *ProgressState {
	return &ProgressState{
		Active:     make(map[string]int),
		Completed:  make(map[string]struct{}),
		Failed:     make(map[string]string),
		Order:      []string{},
		Expected:   make(map[string]struct{}),
		DoneTables: []string{},
	}
}

// Reset clears all progress state, preparing for a new seeding session.
// This is useful when the backend proposes a new plan mid-session.
func (ps *ProgressState) Reset() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.Active = make(map[string]int)
	ps.Completed = make(map[string]struct{})
	ps.Failed = make(map[string]string)
	ps.Order = nil
	ps.Expected = make(map[string]struct{})
	ps.DoneTables = nil
}

// AddExpected marks a table as expected to be seeded.
// This is typically called when the backend proposes a seeding plan.
func (ps *ProgressState) AddExpected(tableName string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.Expected[tableName] = struct{}{}
}

// AddExpectedBatch marks multiple tables as expected to be seeded.
func (ps *ProgressState) AddExpectedBatch(tableNames []string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	for _, name := range tableNames {
		ps.Expected[name] = struct{}{}
	}
}

// StartTable marks a table as active with its initial remaining task count.
// If the table wasn't in the expected list, it's automatically added.
func (ps *ProgressState) StartTable(tableName string, remaining int) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if _, exists := ps.Active[tableName]; !exists {
		ps.Order = append(ps.Order, tableName)
	}
	ps.Active[tableName] = remaining

	// Auto-add to expected if not already present
	if _, exists := ps.Expected[tableName]; !exists {
		ps.Expected[tableName] = struct{}{}
	}
}

// CompleteTable marks a table as successfully completed.
func (ps *ProgressState) CompleteTable(tableName string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.Active, tableName)
	ps.Completed[tableName] = struct{}{}
	ps.DoneTables = append(ps.DoneTables, tableName)
}

// FailTable marks a table as failed with a reason.
func (ps *ProgressState) FailTable(tableName string, reason string) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	delete(ps.Active, tableName)
	ps.Failed[tableName] = reason
}

// CompleteAllActive marks all currently active tables as completed.
// This is used when the backend signals workflow completion but some tables are still marked as active.
func (ps *ProgressState) CompleteAllActive() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	for name := range ps.Active {
		delete(ps.Active, name)
		ps.Completed[name] = struct{}{}

		// Auto-add to expected if not already present
		if _, exists := ps.Expected[name]; !exists {
			ps.Expected[name] = struct{}{}
		}
	}
}

// ExpectedCount returns the total number of expected tables.
func (ps *ProgressState) ExpectedCount() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.Expected)
}

// CompletedCount returns the total number of completed tables.
func (ps *ProgressState) CompletedCount() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.Completed)
}

// FailedCount returns the total number of failed tables.
func (ps *ProgressState) FailedCount() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.Failed)
}

// HasFailures returns true if any table has failed.
func (ps *ProgressState) HasFailures() bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.Failed) > 0
}

// IsFullyCompleted returns true if all expected tables have been completed.
func (ps *ProgressState) IsFullyCompleted() bool {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	expectedCount := len(ps.Expected)
	return expectedCount > 0 && len(ps.Completed) == expectedCount
}

// GetDoneTableCount returns the number of successfully completed tables.
func (ps *ProgressState) GetDoneTableCount() int {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return len(ps.DoneTables)
}

// RenderState holds the UI rendering state for the seeding progress display.
// It tracks animation frames, display dimensions, and cached rendering output.
type RenderState struct {
	// FrameIdx is the current animation frame index for spinners
	FrameIdx int
	// MaxLineLen tracks the maximum line length to prevent flickering
	MaxLineLen int
	// LastRendered caches the last rendered content to avoid unnecessary updates
	LastRendered string
	// mu protects concurrent access to rendering state
	mu sync.Mutex
}

// NewRenderState creates a new RenderState with default values.
func NewRenderState() *RenderState {
	return &RenderState{
		FrameIdx:     0,
		MaxLineLen:   0,
		LastRendered: "",
	}
}

// IncrementFrame advances the animation frame index.
func (rs *RenderState) IncrementFrame() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.FrameIdx++
}

// UpdateMaxLineLen updates the maximum line length if the new length is greater.
func (rs *RenderState) UpdateMaxLineLen(newLen int) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if newLen > rs.MaxLineLen {
		rs.MaxLineLen = newLen
	}
}

// GetMaxLineLen returns the current maximum line length.
func (rs *RenderState) GetMaxLineLen() int {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.MaxLineLen
}

// GetFrameIdx returns the current frame index.
func (rs *RenderState) GetFrameIdx() int {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.FrameIdx
}

// SetLastRendered updates the cached rendering output.
func (rs *RenderState) SetLastRendered(content string) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.LastRendered = content
}

// GetLastRendered returns the cached rendering output.
func (rs *RenderState) GetLastRendered() string {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return rs.LastRendered
}

// Reset clears the rendering state for a new session.
func (rs *RenderState) Reset() {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	rs.MaxLineLen = 0
	rs.LastRendered = ""
}

// FormatLine formats a line for display with proper padding to prevent flickering.
// It ensures all lines are padded to the same maximum length.
func (rs *RenderState) FormatLine(line string) string {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	lineLen := utf8.RuneCountInString(line)
	if lineLen > rs.MaxLineLen {
		rs.MaxLineLen = lineLen
	}

	if pad := rs.MaxLineLen - lineLen; pad > 0 {
		return line + repeatSpaces(pad)
	}
	return line
}

// repeatSpaces returns a string of n spaces.
func repeatSpaces(n int) string {
	if n <= 0 {
		return ""
	}
	// Use a builder for efficiency with larger n
	if n > 100 {
		b := make([]byte, n)
		for i := range b {
			b[i] = ' '
		}
		return string(b)
	}
	// For small n, simple string repetition is fine
	result := ""
	for i := 0; i < n; i++ {
		result += " "
	}
	return result
}
