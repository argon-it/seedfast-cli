// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package sqlexec

import (
	"context"
	"regexp"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SchemaInfo holds information about table constraints and structure.
// It caches database metadata to avoid repeated queries to information_schema.
type SchemaInfo struct {
	// TableName is the fully qualified or unqualified table name
	TableName string
	// PrimaryKeyCols lists primary key column names in order
	PrimaryKeyCols []string
	// AutoIncrement maps column names to whether they use sequences (auto-increment)
	AutoIncrement map[string]bool
	// CheckConstraints maps column names to their check constraint definitions
	CheckConstraints map[string]string
	// EnumValues maps column names to their allowed enum values (extracted from constraints)
	EnumValues map[string][]string
}

// SchemaInspector provides database schema inspection and caching capabilities.
// It queries information_schema to gather metadata about tables, columns, constraints,
// and caches results to minimize database roundtrips.
type SchemaInspector struct {
	// pool is the connection pool for executing schema queries
	pool *pgxpool.Pool
	// cache stores schema information keyed by table name
	cache map[string]*SchemaInfo
	// mu protects concurrent access to the cache
	mu sync.RWMutex
}

// NewSchemaInspector creates a new SchemaInspector with the given connection pool.
func NewSchemaInspector(pool *pgxpool.Pool) *SchemaInspector {
	return &SchemaInspector{
		pool:  pool,
		cache: make(map[string]*SchemaInfo),
	}
}

// GetSchemaInfo retrieves or caches schema information for a table.
// It returns cached data if available, otherwise queries the database and caches the result.
// The tableName can be either "table" or "schema.table".
func (si *SchemaInspector) GetSchemaInfo(ctx context.Context, tableName string) (*SchemaInfo, error) {
	// Check cache first
	si.mu.RLock()
	if info, exists := si.cache[tableName]; exists {
		si.mu.RUnlock()
		return info, nil
	}
	si.mu.RUnlock()

	// Parse table name to extract schema and table
	schema, table := parseTableName(tableName)

	info := &SchemaInfo{
		TableName:        tableName,
		AutoIncrement:    make(map[string]bool),
		CheckConstraints: make(map[string]string),
		EnumValues:       make(map[string][]string),
	}

	// Acquire connection
	conn, err := si.pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	// Get primary key information
	if err := si.loadPrimaryKeys(ctx, conn, schema, table, info); err != nil {
		return nil, err
	}

	// Get auto-increment information for primary key columns
	if err := si.loadAutoIncrementInfo(ctx, conn, schema, table, info); err != nil {
		return nil, err
	}

	// Get check constraints that define enum-like restrictions
	if err := si.loadCheckConstraints(ctx, conn, schema, table, info); err != nil {
		// Non-fatal: continue without check constraints
		logDebug("Failed to load check constraints for %s.%s: %v", schema, table, err)
	}

	// Cache the result
	si.mu.Lock()
	si.cache[tableName] = info
	si.mu.Unlock()

	return info, nil
}

// ClearCache clears all cached schema information.
// This is useful when schema changes are expected.
func (si *SchemaInspector) ClearCache() {
	si.mu.Lock()
	defer si.mu.Unlock()
	si.cache = make(map[string]*SchemaInfo)
}

// parseTableName splits a table name into schema and table components.
// If no schema is specified, it defaults to "public".
func parseTableName(tableName string) (schema string, table string) {
	parts := strings.Split(tableName, ".")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "public", tableName
}

// loadPrimaryKeys queries and populates primary key information for a table.
func (si *SchemaInspector) loadPrimaryKeys(ctx context.Context, conn *pgxpool.Conn, schema, table string, info *SchemaInfo) error {
	pkQuery := `
		SELECT c.column_name
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kc ON tc.constraint_name = kc.constraint_name
		JOIN information_schema.columns c ON kc.table_name = c.table_name AND kc.column_name = c.column_name
		WHERE tc.table_schema = $1 AND tc.table_name = $2 AND tc.constraint_type = 'PRIMARY KEY'
		ORDER BY kc.ordinal_position`

	rows, err := conn.Query(ctx, pkQuery, schema, table)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var colName string
		if err := rows.Scan(&colName); err == nil {
			info.PrimaryKeyCols = append(info.PrimaryKeyCols, colName)
		}
	}

	return rows.Err()
}

// loadAutoIncrementInfo queries and populates auto-increment information for primary key columns.
func (si *SchemaInspector) loadAutoIncrementInfo(ctx context.Context, conn *pgxpool.Conn, schema, table string, info *SchemaInfo) error {
	for _, pkCol := range info.PrimaryKeyCols {
		autoQuery := `
			SELECT c.column_default LIKE '%nextval%'
			FROM information_schema.columns c
			WHERE c.table_schema = $1 AND c.table_name = $2 AND c.column_name = $3`

		autoRows, err := conn.Query(ctx, autoQuery, schema, table, pkCol)
		if err != nil {
			continue // Skip this column on error
		}
		defer autoRows.Close()

		if autoRows.Next() {
			var isAuto bool
			if err := autoRows.Scan(&isAuto); err == nil {
				info.AutoIncrement[pkCol] = isAuto
			}
		}
	}

	return nil
}

// loadCheckConstraints queries and populates check constraint information for a table.
func (si *SchemaInspector) loadCheckConstraints(ctx context.Context, conn *pgxpool.Conn, schema, table string, info *SchemaInfo) error {
	checkQuery := `
		SELECT c.column_name, cc.check_clause
		FROM information_schema.check_constraints cc
		JOIN information_schema.columns c ON cc.constraint_name = c.column_name || '_check'
		WHERE cc.constraint_schema = $1 AND c.table_name = $2`

	checkRows, err := conn.Query(ctx, checkQuery, schema, table)
	if err != nil {
		return err
	}
	defer checkRows.Close()

	for checkRows.Next() {
		var colName, checkClause string
		if err := checkRows.Scan(&colName, &checkClause); err == nil {
			info.CheckConstraints[colName] = checkClause
			// Extract enum values from check constraints like "status IN ('queued','running','done','failed')"
			if enumValues := extractEnumValues(checkClause); len(enumValues) > 0 {
				info.EnumValues[colName] = enumValues
			}
		}
	}

	return checkRows.Err()
}

// extractEnumValues extracts enum values from a check constraint clause.
// It supports patterns like:
//   - "status IN ('queued','running','done','failed')"
//   - "status = ANY (ARRAY['queued'::text, 'running'::text, ...])"
func extractEnumValues(checkClause string) []string {
	// Look for patterns like "status IN ('queued','running','done','failed')"
	inRegex := regexp.MustCompile(`(?i)IN\s*\(\s*([^)]+)\)`)
	if inRegex.MatchString(checkClause) {
		match := inRegex.FindStringSubmatch(checkClause)
		if len(match) > 1 {
			return parseEnumValueList(match[1])
		}
	}

	// Look for patterns like "status = ANY (ARRAY['queued'::text, 'running'::text, ...])"
	anyRegex := regexp.MustCompile(`(?i)=\s*ANY\s*\(\s*ARRAY\s*\[([^\]]+)\]`)
	if anyRegex.MatchString(checkClause) {
		match := anyRegex.FindStringSubmatch(checkClause)
		if len(match) > 1 {
			return parseEnumValueList(match[1])
		}
	}

	return nil
}

// parseEnumValueList parses a comma-separated list of enum values.
// It handles both single and double quotes and trims whitespace.
func parseEnumValueList(valueList string) []string {
	values := strings.Split(valueList, ",")
	var result []string
	for _, val := range values {
		val = strings.TrimSpace(val)
		val = strings.Trim(val, "'\"")
		// Remove type casts like ::text
		if idx := strings.Index(val, "::"); idx >= 0 {
			val = val[:idx]
		}
		if val != "" {
			result = append(result, val)
		}
	}
	return result
}
