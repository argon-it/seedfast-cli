// Package sqlexec provides a concurrent SQL execution engine over a pgx connection pool.
// It handles SQL statement execution, result formatting for database seeding operations.
// The package supports both read and write operations with automatic transaction management and
// provides JSON-formatted results suitable for backend communication.
//
// Key features include:
//   - Direct SQL execution with schema-qualified table names
//   - Transaction management for write operations
//   - JSON result formatting with proper type handling
//   - Debug logging for troubleshooting
//   - Support for PostgreSQL-specific data types (UUIDs, byte arrays, etc.)
package sqlexec

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// File logging disabled - debug logs are no longer written to sql_debug.log
func logDebug(format string, args ...interface{}) {
	// No-op: logging disabled
}

// min returns the minimum of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Result represents a normalized SQL result for JSON marshaling.
type Result struct {
	Columns      []string `json:"columns"`
	Rows         [][]any  `json:"rows"`
	RowsAffected int64    `json:"rows_affected,omitempty"`
	Error        string   `json:"error,omitempty"`
}

// MarshalJSON implements custom JSON marshaling for Result to handle pgx types properly.
func (r Result) MarshalJSON() ([]byte, error) {
	type Alias Result
	a := Alias(r)

	// Convert rows to JSON-serializable format
	if len(r.Rows) > 0 {
		serializableRows := make([][]interface{}, len(r.Rows))
		for i, row := range r.Rows {
			serializableRows[i] = make([]interface{}, len(row))
			for j, val := range row {
				// Convert pgx types to JSON-serializable values
				switch v := val.(type) {
				case []byte:
					// Handle UUID and other byte arrays as hex strings
					if len(v) == 16 {
						// UUID format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
						// Use %02x to ensure each byte is exactly 2 hex digits (with leading zeros)
						uuidStr := fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
							v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7],
							v[8], v[9], v[10], v[11], v[12], v[13], v[14], v[15])
						logDebug("UUID conversion: bytes=%v -> string=%s", v, uuidStr)
						serializableRows[i][j] = uuidStr
					} else {
						serializableRows[i][j] = fmt.Sprintf("\\x%x", v)
					}
				case [16]byte:
					// Handle UUID as fixed-size byte array (pgx might return this format)
					uuidStr := fmt.Sprintf("%02x%02x%02x%02x-%02x%02x-%02x%02x-%02x%02x-%02x%02x%02x%02x%02x%02x",
						v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7],
						v[8], v[9], v[10], v[11], v[12], v[13], v[14], v[15])
					logDebug("UUID conversion [16]byte: -> string=%s", uuidStr)
					serializableRows[i][j] = uuidStr
				case nil:
					serializableRows[i][j] = nil
				default:
					// Keep the value as-is for all other types (strings, numbers, etc.)
					serializableRows[i][j] = v
				}
			}
		}
		a.Rows = serializableRows
	}

	jsonBytes, err := json.Marshal(a)
	if err != nil {
		return nil, err
	}

	return jsonBytes, nil
}

// Executor executes SQL statements using a connection pool.
// It integrates schema inspection and SQL fixing capabilities for robust seeding operations.
type Executor struct {
	// Pool is the PostgreSQL connection pool
	Pool *pgxpool.Pool
	// inspector provides schema metadata caching and introspection
	inspector *SchemaInspector
	// fixer applies SQL statement repairs based on schema constraints
	fixer *SQLFixer
}

// New creates an Executor from an existing pgx pool.
// It initializes the schema inspector and SQL fixer for enhanced seeding capabilities.
func New(pool *pgxpool.Pool) *Executor {
	inspector := NewSchemaInspector(pool)
	fixer := NewSQLFixer(inspector)
	return &Executor{
		Pool:      pool,
		inspector: inspector,
		fixer:     fixer,
	}
}

// ExecuteSQL runs an arbitrary SQL statement and returns a JSON payload.
// Read queries return {columns, rows}. Write operations return {} or {error}.
func (e *Executor) ExecuteSQL(ctx context.Context, sql string, isWrite bool) (string, error) {
	return e.ExecuteSQLInSchema(ctx, sql, isWrite, "")
}

// ExecuteSQLInSchema runs SQL with optional schema by setting search_path.
// The schema parameter is optional and only used for backward compatibility.
// In most cases, SQL statements should use schema-qualified table names (e.g., "app.users")
// which PostgreSQL handles natively without needing to set search_path.
func (e *Executor) ExecuteSQLInSchema(ctx context.Context, sql string, isWrite bool, schema string) (string, error) {
	// Fix common seeding issues before execution using schema-aware approach
	fixedSQL, err := e.fixer.FixSeedingSQL(ctx, sql)
	if err != nil {
		logDebug("Failed to fix seeding SQL: %v", err)
		// Continue with original SQL if fixing fails
		fixedSQL = sql
	}
	if fixedSQL != sql {
		logDebug("SQL was modified for seeding fixes using schema-aware approach")
		sql = fixedSQL
	}
	res := Result{
		Columns: []string{},
		Rows:    [][]any{},
	}
	conn, err := e.Pool.Acquire(ctx)
	if err != nil {
		res.Error = err.Error()
		jsonBytes, _ := res.MarshalJSON()
		return string(jsonBytes), nil
	}
	defer conn.Release()

	// Only set search_path if explicitly provided (for backward compatibility).
	// Otherwise, rely on schema-qualified table names in the SQL itself (e.g., "app.users").
	// PostgreSQL natively handles schema-qualified names without requiring search_path.
	if schema != "" {
		logDebug("Using explicitly provided schema: %s", schema)
		_, err = conn.Exec(ctx, "SET search_path TO "+schema)
		if err != nil {
			logDebug("Failed to set search_path: %v", err)
		}
	} else {
		logDebug("No explicit schema provided, relying on schema-qualified table names in SQL")
	}

	if isWrite {
		// Start a transaction for write operations to ensure proper commit
		logDebug("BEGIN transaction for SQL: %s", sql[:min(100, len(sql))])
		tx, err := conn.Begin(ctx)
		if err != nil {
			logDebug("BEGIN transaction failed: %v", err)
			res.Error = err.Error()
			jsonBytes, _ := res.MarshalJSON()
			return string(jsonBytes), nil
		}
		defer tx.Rollback(ctx) // Rollback if commit doesn't happen

		ct, err := tx.Exec(ctx, sql)
		if err != nil {
			logDebug("Exec failed: %v", err)
			res.Error = err.Error()
			jsonBytes, _ := res.MarshalJSON()
			return string(jsonBytes), nil
		}

		res.RowsAffected = ct.RowsAffected()
		logDebug("Exec succeeded, rows affected: %d, attempting COMMIT...", res.RowsAffected)

		// Commit the transaction
		if err := tx.Commit(ctx); err != nil {
			logDebug("COMMIT failed: %v", err)
			res.Error = fmt.Sprintf("commit failed: %v", err)
			jsonBytes, _ := res.MarshalJSON()
			return string(jsonBytes), nil
		}

		logDebug("COMMIT succeeded!")
		jsonBytes, _ := res.MarshalJSON()
		return string(jsonBytes), nil
	}

	rows, err := conn.Query(ctx, sql)
	if err != nil {
		res.Error = err.Error()
		jsonBytes, _ := res.MarshalJSON()
		return string(jsonBytes), nil
	}
	defer rows.Close()

	fds := rows.FieldDescriptions()
	cols := make([]string, len(fds))
	for i, fd := range fds {
		cols[i] = string(fd.Name)
	}
	res.Columns = cols

	for rows.Next() {
		vals, err := rows.Values()
		if err != nil {
			res.Error = err.Error()
			break
		}
		res.Rows = append(res.Rows, vals)
	}
	if rows.Err() != nil {
		res.Error = rows.Err().Error()
	}

	// Use custom MarshalJSON to properly handle pgx types
	jsonBytes, marshalErr := res.MarshalJSON()
	if marshalErr != nil {
		res.Error = fmt.Sprintf("JSON marshal error: %v", marshalErr)
		jsonBytes, _ = json.Marshal(res) // fallback to basic marshal
	}

	// Log complete JSON response for SELECT queries (file log only)
	jsonStr := string(jsonBytes)
	if len(res.Rows) > 0 {
		if len(jsonStr) <= 500 {
			logDebug("SELECT response JSON: %s", jsonStr)
		} else {
			logDebug("SELECT response JSON (first 500 chars): %s...", jsonStr[:500])
		}
	}

	return jsonStr, nil
}
