package schema

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt" // Imported for more detailed error messages

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// ColumnInfo defines the structure for a single column's metadata.
type ColumnInfo struct {
	ColumnName string `json:"column_name"`
	DataType   string `json:"data_type"`
}

// TableInfo defines the structure for a single table, including its columns.
type TableInfo struct {
	TableName string       `json:"table_name"`
	Columns   []ColumnInfo `json:"columns"`
}

// GetSchema retrieves the schema information for all tables within a specified schema.
// It returns a JSON string representing an array of TableInfo structs.
func GetSchema(db *sql.DB) (string, error) {
	// This query retrieves table and column information specifically from the 'sml' schema.
	// It's designed to be efficient by joining and ordering in the database.
	const query = `
        SELECT
            t.table_name,
            c.column_name,
            c.data_type
        FROM
            information_schema.tables AS t
        JOIN
            information_schema.columns AS c ON t.table_name = c.table_name AND t.table_schema = c.table_schema
        WHERE
            t.table_schema = 'sml' AND t.table_type = 'BASE TABLE'
        ORDER BY
            t.table_name, c.ordinal_position;
    `

	// Use a background context for this database operation.
	ctx := context.Background()
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Error querying schema from the database: %v", err)
		// Return a more specific error instead of just the original.
		return "", fmt.Errorf("failed to execute schema query: %w", err)
	}
	// 'defer rows.Close()' is crucial to ensure the database connection is released.
	defer rows.Close()

	// Use a map to group columns by table name efficiently.
	schemaMap := make(map[string][]ColumnInfo)
	for rows.Next() {
		var tableName, columnName, dataType string
		// Scan each row for the required data.
		if err := rows.Scan(&tableName, &columnName, &dataType); err != nil {
			logging.GetLogger().Sugar().Errorf("Error scanning row from schema query result: %v", err)
			// If scanning fails, stop immediately and return the error.
			return "", fmt.Errorf("failed to scan schema row: %w", err)
		}
		// Append the column information to the correct table's slice.
		schemaMap[tableName] = append(schemaMap[tableName], ColumnInfo{
			ColumnName: columnName,
			DataType:   dataType,
		})
	}

	// After the loop, check if any errors occurred during iteration.
	if err = rows.Err(); err != nil {
		logging.GetLogger().Sugar().Errorf("An error occurred during row iteration: %v", err)
		return "", fmt.Errorf("error iterating over schema rows: %w", err)
	}

	// Convert the map into a slice of TableInfo for stable JSON ordering.
	schema := make([]TableInfo, 0, len(schemaMap))
	for name, columns := range schemaMap {
		schema = append(schema, TableInfo{
			TableName: name,
			Columns:   columns,
		})
	}

	// Marshal the final slice into a JSON byte array.
	jsonBytes, err := json.MarshalIndent(schema, "", "  ") // Using MarshalIndent for pretty-printing
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Error marshaling schema to JSON: %v", err)
		return "", fmt.Errorf("failed to marshal schema to JSON: %w", err)
	}

	// Return the JSON as a string.
	return string(jsonBytes), nil
}
