//go:build ignore

package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

type ValidationResult struct {
	NodeID     string   `json:"node_id"`
	NodeType   string   `json:"node_type"`
	IsValid    bool     `json:"is_valid"`
	Errors     []string `json:"errors"`
	Warnings   []string `json:"warnings"`
	SchemaHash string   `json:"schema_hash"`
}

func main() {
	// Database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost/semlayer?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	fmt.Println("🔍 Validating Advanced Modeling Patterns...")

	// Validate dynamic unions
	unionResults := validateDynamicUnions(db)

	// Validate string time dimensions
	timeDimResults := validateStringTimeDimensions(db)

	// Validate custom granularities
	granularityResults := validateCustomGranularities(db)

	// Combine results
	allResults := append(append(unionResults, timeDimResults...), granularityResults...)

	// Generate report
	generateValidationReport(allResults)

	// Update schema hashes
	updateSchemaHashes(db, allResults)

	fmt.Printf("✅ Validation complete! Processed %d modeling patterns\n", len(allResults))
}

func validateDynamicUnions(db *sql.DB) []ValidationResult {
	rows, err := db.Query(`
		SELECT node_id, name, schema_def
		FROM public.catalog_node
		WHERE node_type = 'dynamic_union'
	`)
	if err != nil {
		log.Printf("Failed to fetch unions: %v", err)
		return nil
	}
	defer rows.Close()

	var results []ValidationResult
	for rows.Next() {
		var nodeID, name, schemaJSON string
		rows.Scan(&nodeID, &name, &schemaJSON)

		result := ValidationResult{
			NodeID:   nodeID,
			NodeType: "dynamic_union",
			IsValid:  true,
			Errors:   []string{},
			Warnings: []string{},
		}

		var schema map[string]interface{}
		if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, "Invalid JSON schema")
			results = append(results, result)
			continue
		}

		// Validate required fields
		if schema["source_tables"] == nil {
			result.Errors = append(result.Errors, "Missing source_tables")
			result.IsValid = false
		}

		if schema["union_sql"] == nil {
			result.Errors = append(result.Errors, "Missing union_sql")
			result.IsValid = false
		}

		// Validate source tables exist
		if tables, ok := schema["source_tables"].([]interface{}); ok {
			for _, table := range tables {
				if tableStr, ok := table.(string); ok {
					if !tableExists(db, tableStr) {
						result.Errors = append(result.Errors, fmt.Sprintf("Source table '%s' does not exist", tableStr))
						result.IsValid = false
					}
				}
			}
		}

		// Generate schema hash
		result.SchemaHash = generateSchemaHash(schemaJSON)

		results = append(results, result)
	}

	fmt.Printf("📊 Validated %d dynamic unions\n", len(results))
	return results
}

func validateStringTimeDimensions(db *sql.DB) []ValidationResult {
	rows, err := db.Query(`
		SELECT node_id, name, schema_def
		FROM public.catalog_node
		WHERE node_type = 'string_time_dimension'
	`)
	if err != nil {
		log.Printf("Failed to fetch time dimensions: %v", err)
		return nil
	}
	defer rows.Close()

	var results []ValidationResult
	for rows.Next() {
		var nodeID, name, schemaJSON string
		rows.Scan(&nodeID, &name, &schemaJSON)

		result := ValidationResult{
			NodeID:   nodeID,
			NodeType: "string_time_dimension",
			IsValid:  true,
			Errors:   []string{},
			Warnings: []string{},
		}

		var schema map[string]interface{}
		if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, "Invalid JSON schema")
			results = append(results, result)
			continue
		}

		// Validate required fields
		if schema["source_column"] == nil {
			result.Errors = append(result.Errors, "Missing source_column")
			result.IsValid = false
		}

		if schema["source_table"] == nil {
			result.Errors = append(result.Errors, "Missing source_table")
			result.IsValid = false
		}

		if schema["date_format"] == nil {
			result.Errors = append(result.Errors, "Missing date_format")
			result.IsValid = false
		}

		// Validate source table and column exist
		if table, ok := schema["source_table"].(string); ok {
			if column, ok := schema["source_column"].(string); ok {
				if !tableExists(db, table) {
					result.Errors = append(result.Errors, fmt.Sprintf("Source table '%s' does not exist", table))
					result.IsValid = false
				} else if !columnExists(db, table, column) {
					result.Errors = append(result.Errors, fmt.Sprintf("Source column '%s' does not exist in table '%s'", column, table))
					result.IsValid = false
				}
			}
		}

		// Validate date format
		if format, ok := schema["date_format"].(string); ok {
			if !isValidDateFormat(format) {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Potentially invalid date format: %s", format))
			}
		}

		// Generate schema hash
		result.SchemaHash = generateSchemaHash(schemaJSON)

		results = append(results, result)
	}

	fmt.Printf("📊 Validated %d string time dimensions\n", len(results))
	return results
}

func validateCustomGranularities(db *sql.DB) []ValidationResult {
	rows, err := db.Query(`
		SELECT node_id, name, schema_def
		FROM public.catalog_node
		WHERE node_type = 'custom_granularity'
	`)
	if err != nil {
		log.Printf("Failed to fetch granularities: %v", err)
		return nil
	}
	defer rows.Close()

	var results []ValidationResult
	for rows.Next() {
		var nodeID, name, schemaJSON string
		rows.Scan(&nodeID, &name, &schemaJSON)

		result := ValidationResult{
			NodeID:   nodeID,
			NodeType: "custom_granularity",
			IsValid:  true,
			Errors:   []string{},
			Warnings: []string{},
		}

		var schema map[string]interface{}
		if err := json.Unmarshal([]byte(schemaJSON), &schema); err != nil {
			result.IsValid = false
			result.Errors = append(result.Errors, "Invalid JSON schema")
			results = append(results, result)
			continue
		}

		// Validate required fields
		if schema["dimension"] == nil {
			result.Errors = append(result.Errors, "Missing dimension")
			result.IsValid = false
		}

		if schema["interval"] == nil {
			result.Errors = append(result.Errors, "Missing interval")
			result.IsValid = false
		}

		// Validate interval values
		if interval, ok := schema["interval"].(string); ok {
			validIntervals := []string{"minute", "hour", "day", "week", "month", "quarter", "year"}
			valid := false
			for _, v := range validIntervals {
				if interval == v {
					valid = true
					break
				}
			}
			if !valid {
				result.Errors = append(result.Errors, fmt.Sprintf("Invalid interval: %s", interval))
				result.IsValid = false
			}
		}

		// Validate calendar type
		if calendarType, ok := schema["calendar_type"].(string); ok {
			validTypes := []string{"gregorian", "fiscal", "iso_week", "custom"}
			valid := false
			for _, v := range validTypes {
				if calendarType == v {
					valid = true
					break
				}
			}
			if !valid {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Unknown calendar type: %s", calendarType))
			}
		}

		// Generate schema hash
		result.SchemaHash = generateSchemaHash(schemaJSON)

		results = append(results, result)
	}

	fmt.Printf("📊 Validated %d custom granularities\n", len(results))
	return results
}

func tableExists(db *sql.DB, tableName string) bool {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = $1
		)`
	db.QueryRow(query, tableName).Scan(&exists)
	return exists
}

func columnExists(db *sql.DB, tableName, columnName string) bool {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM information_schema.columns
			WHERE table_schema = 'public'
			AND table_name = $1
			AND column_name = $2
		)`
	db.QueryRow(query, tableName, columnName).Scan(&exists)
	return exists
}

func isValidDateFormat(format string) bool {
	// Basic validation for common date formats
	validFormats := []string{
		"YYYY-MM-DD", "MM/DD/YYYY", "DD/MM/YYYY", "YYYY/MM/DD",
		"MM-DD-YYYY", "DD-MM-YYYY", "YYYY-MM-DD HH:mm:ss",
		"YYYY-MM-DDTHH:mm:ssZ", "MM/DD/YYYY HH:mm:ss",
	}

	for _, validFormat := range validFormats {
		if format == validFormat {
			return true
		}
	}

	// Allow custom formats that contain common date components
	return strings.Contains(format, "YYYY") || strings.Contains(format, "YY") ||
		strings.Contains(format, "MM") || strings.Contains(format, "DD")
}

func generateSchemaHash(schemaJSON string) string {
	hash := sha256.Sum256([]byte(schemaJSON))
	return fmt.Sprintf("%x", hash)
}

func generateValidationReport(results []ValidationResult) {
	validCount := 0
	errorCount := 0
	warningCount := 0

	for _, result := range results {
		if result.IsValid {
			validCount++
		}
		errorCount += len(result.Errors)
		warningCount += len(result.Warnings)
	}

	report := fmt.Sprintf(`Advanced Modeling Patterns Validation Report
==============================================

Summary:
- Total Patterns: %d
- Valid Patterns: %d
- Total Errors: %d
- Total Warnings: %d

Breakdown by Type:
`, len(results), validCount, errorCount, warningCount)

	typeCounts := make(map[string]int)
	for _, result := range results {
		typeCounts[result.NodeType]++
	}

	for nodeType, count := range typeCounts {
		report += fmt.Sprintf("- %s: %d\n", nodeType, count)
	}

	report += "\nDetailed Results:\n"

	for _, result := range results {
		status := "✅ VALID"
		if !result.IsValid {
			status = "❌ INVALID"
		} else if len(result.Warnings) > 0 {
			status = "⚠️  WARNINGS"
		}

		report += fmt.Sprintf("\n%s - %s (%s)\n", status, result.NodeID, result.NodeType)

		for _, err := range result.Errors {
			report += fmt.Sprintf("  ❌ %s\n", err)
		}

		for _, warning := range result.Warnings {
			report += fmt.Sprintf("  ⚠️  %s\n", warning)
		}
	}

	os.WriteFile("validation_report.txt", []byte(report), 0644)
	fmt.Println("📄 Validation report saved to validation_report.txt")
}

func updateSchemaHashes(db *sql.DB, results []ValidationResult) {
	for _, result := range results {
		if result.IsValid {
			_, err := db.Exec(`
				UPDATE public.catalog_node
				SET schema_hash = $1, updated_at = NOW()
				WHERE node_id = $2
			`, result.SchemaHash, result.NodeID)

			if err != nil {
				log.Printf("Failed to update schema hash for %s: %v", result.NodeID, err)
			}
		}
	}

	fmt.Println("🔄 Schema hashes updated for valid patterns")
}
