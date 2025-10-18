// Copyright (c) 2025 Seedfast
// Licensed under the MIT License. See LICENSE file in the project root for details.

package sqlexec

import (
	"context"
	"fmt"
	"regexp"
	"strings"
)

// SQLFixer provides SQL statement repair functionality for database seeding operations.
// It can fix common issues like explicit ID values for auto-increment columns
// and invalid enum constraint values.
type SQLFixer struct {
	inspector *SchemaInspector
}

// NewSQLFixer creates a new SQLFixer with the given schema inspector.
func NewSQLFixer(inspector *SchemaInspector) *SQLFixer {
	return &SQLFixer{
		inspector: inspector,
	}
}

// FixSeedingSQL fixes common seeding issues in SQL statements based on schema information.
// It handles:
//  1. Removing explicit ID values for auto-incrementing primary keys
//  2. Fixing enum constraint violations by replacing invalid values
// Returns the fixed SQL statement and any error encountered during analysis.
func (f *SQLFixer) FixSeedingSQL(ctx context.Context, sql string) (string, error) {
	originalSQL := sql

	tableName, columns, valuesStart, hasIdColumn := parseInsertStatement(sql)
	if tableName == "" {
		return sql, nil // Not an INSERT statement, no fixes needed
	}

	logDebug("Analyzing INSERT statement for table: %s", tableName)

	// Get schema information for the table
	schemaInfo, err := f.inspector.GetSchemaInfo(ctx, tableName)
	if err != nil {
		logDebug("Failed to get schema info for %s: %v", tableName, err)
		return sql, nil // Continue without fixing if we can't get schema info
	}

	fixed := false

	// Fix 1: Remove explicit ID values for auto-incrementing primary keys
	if hasIdColumn && len(schemaInfo.PrimaryKeyCols) > 0 {
		for _, pkCol := range schemaInfo.PrimaryKeyCols {
			if isAuto, exists := schemaInfo.AutoIncrement[pkCol]; exists && isAuto {
				if strings.ToLower(pkCol) == "id" {
					logDebug("Removing explicit ID from auto-increment column for table: %s", tableName)
					sql, _ = removeIdColumnAndValue(sql, columns)
					fixed = true
					break
				}
			}
		}
	}

	// Fix 2: Fix enum constraint violations
	for i, col := range columns {
		if enumValues, exists := schemaInfo.EnumValues[col]; exists && len(enumValues) > 0 {
			// Extract the value at position i from the VALUES clause
			if valuesStart >= 0 {
				value, newSQL, err := extractAndFixValue(sql, valuesStart, i)
				if err == nil && value != "" {
					// Check if the value is valid
					isValid := false
					for _, validValue := range enumValues {
						if strings.EqualFold(value, validValue) {
							isValid = true
							break
						}
					}

					if !isValid {
						logDebug("Fixing invalid enum value '%s' for column %s in table %s", value, col, tableName)
						// Replace with first valid value (could be improved to be smarter)
						replacement := enumValues[0]
						sql = strings.Replace(newSQL, "'"+value+"'", "'"+replacement+"'", 1)
						fixed = true
					}
				}
			}
		}
	}

	if fixed {
		logDebug("Fixed SQL statement for table %s", tableName)
		logDebug("Original: %s", originalSQL)
		logDebug("Fixed: %s", sql)
	}

	return sql, nil
}

// parseInsertStatement extracts table name and column information from INSERT statements.
// Returns:
//   - tableName: the target table name
//   - columns: list of column names in the INSERT
//   - valuesStart: byte offset where the VALUES clause begins
//   - hasIdColumn: true if an "id" column is present
func parseInsertStatement(sql string) (tableName string, columns []string, valuesStart int, hasIdColumn bool) {
	insertRegex := regexp.MustCompile(`(?i)INSERT\s+INTO\s+([^\s(]+)\s*\(([^)]+)\)\s*VALUES`)
	match := insertRegex.FindStringSubmatch(sql)
	if len(match) < 3 {
		return "", nil, -1, false
	}

	tableName = match[1]
	columnsStr := match[2]

	// Parse column list
	columns = parseColumnList(columnsStr)

	// Check if ID column is present
	hasIdColumn = false
	for _, col := range columns {
		if strings.ToLower(col) == "id" {
			hasIdColumn = true
			break
		}
	}

	// Find VALUES clause start
	valuesRegex := regexp.MustCompile(`(?i)\s+VALUES\s*\(`)
	valuesMatch := valuesRegex.FindStringIndex(sql)
	if len(valuesMatch) >= 2 {
		valuesStart = valuesMatch[0]
	}

	return tableName, columns, valuesStart, hasIdColumn
}

// parseColumnList parses a comma-separated list of column names.
// It handles quoted identifiers and trims whitespace.
func parseColumnList(columnsStr string) []string {
	// Simple parsing - split by comma and trim spaces
	// This is a simplified version; a full parser would handle quoted identifiers better
	columns := strings.Split(columnsStr, ",")
	var result []string
	for _, col := range columns {
		col = strings.TrimSpace(col)
		if col != "" {
			result = append(result, col)
		}
	}
	return result
}

// removeIdColumnAndValue removes the ID column and its corresponding value from an INSERT statement.
// This is used to allow the database to auto-generate ID values using sequences.
func removeIdColumnAndValue(sql string, columns []string) (string, error) {
	// Find the ID column index
	idIndex := -1
	for i, col := range columns {
		if strings.ToLower(col) == "id" {
			idIndex = i
			break
		}
	}

	if idIndex == -1 {
		return sql, nil // No ID column found
	}

	// Remove ID from column list
	newColumns := make([]string, 0, len(columns)-1)
	for i, col := range columns {
		if i != idIndex {
			newColumns = append(newColumns, col)
		}
	}

	// Remove corresponding value from VALUES clause
	valuesRegex := regexp.MustCompile(`(?i)VALUES\s*\(([^)]+)\)`)
	match := valuesRegex.FindStringSubmatch(sql)
	if len(match) < 2 {
		return sql, fmt.Errorf("could not parse VALUES clause")
	}

	valuesStr := match[1]
	values := parseColumnList(valuesStr)

	if len(values) <= idIndex {
		return sql, fmt.Errorf("VALUES clause has fewer values than columns")
	}

	newValues := make([]string, 0, len(values)-1)
	for i, val := range values {
		if i != idIndex {
			newValues = append(newValues, val)
		}
	}

	// Reconstruct the SQL
	newColumnsStr := strings.Join(newColumns, ", ")
	newValuesStr := strings.Join(newValues, ", ")

	// Replace VALUES clause
	result := valuesRegex.ReplaceAllString(sql, "VALUES ("+newValuesStr+")")

	// Replace column list - extract table name first
	insertRegex := regexp.MustCompile(`(?i)INSERT\s+INTO\s+([^\s(]+)\s*\(([^)]+)\)`)
	match = insertRegex.FindStringSubmatch(result)
	if len(match) >= 3 {
		tableName := match[1]
		result = insertRegex.ReplaceAllString(result, "INSERT INTO "+tableName+" ("+newColumnsStr+")")
	}

	return result, nil
}

// extractAndFixValue extracts a value at a specific position from a VALUES clause.
// This is used to inspect and potentially fix individual column values.
// Returns the extracted value, the SQL (potentially modified), and any error.
func extractAndFixValue(sql string, valuesStart int, valueIndex int) (string, string, error) {
	valuesPart := sql[valuesStart:]
	// Simple parsing to find the Nth value - this could be improved for complex cases
	parenCount := 0
	valueStart := -1
	currentIndex := 0

	for i, char := range valuesPart {
		switch char {
		case '(':
			parenCount++
			if parenCount == 1 && valueStart == -1 {
				valueStart = i + 1 // Start after the opening paren
			}
		case ')':
			parenCount--
			if parenCount == 0 {
				// End of VALUES clause
				return "", sql, nil
			}
		case ',':
			if parenCount == 1 && currentIndex == valueIndex {
				// Found the end of our target value
				value := strings.TrimSpace(valuesPart[valueStart:i])
				return value, sql, nil
			}
			if parenCount == 1 {
				currentIndex++
				valueStart = i + 1
			}
		}
	}

	return "", sql, fmt.Errorf("could not extract value at index %d", valueIndex)
}
