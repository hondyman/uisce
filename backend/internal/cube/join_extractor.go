package cube

import (
	"database/sql"
	"fmt"
	"strings"
)

// DatabaseJoinExtractor extracts potential joins from database foreign key relationships
type DatabaseJoinExtractor struct {
	db *sql.DB
}

// NewDatabaseJoinExtractor creates a new join extractor
func NewDatabaseJoinExtractor(db *sql.DB) *DatabaseJoinExtractor {
	return &DatabaseJoinExtractor{db: db}
}

// JoinSuggestion represents a suggested join relationship between tables
type JoinSuggestion struct {
	SourceTable  string `json:"source_table"`
	TargetTable  string `json:"target_table"`
	SourceColumn string `json:"source_column"`
	TargetColumn string `json:"target_column"`
	Relationship string `json:"relationship"`
	JoinSQL      string `json:"join_sql"`
	Description  string `json:"description"`
}

// ExtractJoins extracts join suggestions from the catalog metadata
func (e *DatabaseJoinExtractor) ExtractJoins(datasourceID string) ([]JoinSuggestion, error) {
	// Query the catalog_edge_vw to get foreign key relationships
	query := `
		SELECT 
			cnv_source.table_name as source_table,
			cnv_target.table_name as target_table,
			cnv_source.column_name as source_column,
			cnv_target.column_name as target_column,
			cev.relationship_type,
			cnv_source.table_name || '.' || cnv_source.column_name as source_ref,
			cnv_target.table_name || '.' || cnv_target.column_name as target_ref
		FROM catalog_edge_vw cev
		JOIN catalog_node_vw cnv_source ON cev.subject_node_id = cnv_source.id
		JOIN catalog_node_vw cnv_target ON cev.object_node_id = cnv_target.id
		WHERE cev.predicate = 'foreign_key'
		AND cnv_source.catalog_type_name = 'column'
		AND cnv_target.catalog_type_name = 'column'
		ORDER BY cnv_source.table_name, cnv_target.table_name
	`

	rows, err := e.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query join relationships: %w", err)
	}
	defer rows.Close()

	var suggestions []JoinSuggestion
	seen := make(map[string]bool) // Track unique relationships

	for rows.Next() {
		var sourceTable, targetTable, sourceColumn, targetColumn, relationshipType, sourceRef, targetRef string
		err := rows.Scan(&sourceTable, &targetTable, &sourceColumn, &targetColumn, &relationshipType, &sourceRef, &targetRef)
		if err != nil {
			continue // Skip malformed rows
		}

		// Create a unique key for this relationship
		key := fmt.Sprintf("%s->%s", sourceRef, targetRef)
		if seen[key] {
			continue // Skip duplicates
		}
		seen[key] = true

		// Determine join relationship type
		relationship := "many_to_one"
		if strings.Contains(strings.ToLower(relationshipType), "one_to_many") {
			relationship = "one_to_many"
		} else if strings.Contains(strings.ToLower(relationshipType), "many_to_many") {
			relationship = "many_to_many"
		} else if strings.Contains(strings.ToLower(relationshipType), "one_to_one") {
			relationship = "one_to_one"
		}

		// Generate Cube.js compatible join SQL
		joinSQL := fmt.Sprintf("{CUBE.%s} = {%s.%s}", sourceColumn, targetTable, targetColumn)

		// Create human-readable description
		description := fmt.Sprintf("Join %s to %s via %s", sourceTable, targetTable, sourceColumn)

		suggestion := JoinSuggestion{
			SourceTable:  sourceTable,
			TargetTable:  targetTable,
			SourceColumn: sourceColumn,
			TargetColumn: targetColumn,
			Relationship: relationship,
			JoinSQL:      joinSQL,
			Description:  description,
		}

		suggestions = append(suggestions, suggestion)
	}

	return suggestions, nil
}

// GenerateJoinDefinitions creates Cube.js compatible join definitions for a cube
func (e *DatabaseJoinExtractor) GenerateJoinDefinitions(tableName string, datasourceID string) (map[string]map[string]any, error) {
	// Get all foreign key relationships where this table is the source
	query := `
		SELECT DISTINCT
			cnv_target.table_name as target_table,
			cnv_source.column_name as source_column,
			cnv_target.column_name as target_column,
			cev.relationship_type
		FROM catalog_edge_vw cev
		JOIN catalog_node_vw cnv_source ON cev.subject_node_id = cnv_source.id
		JOIN catalog_node_vw cnv_target ON cev.object_node_id = cnv_target.id
		WHERE cev.predicate = 'foreign_key'
		AND cnv_source.catalog_type_name = 'column'
		AND cnv_target.catalog_type_name = 'column'
		AND cnv_source.table_name = $1
		ORDER BY cnv_target.table_name
	`

	rows, err := e.db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query joins for table %s: %w", tableName, err)
	}
	defer rows.Close()

	joins := make(map[string]map[string]any)

	for rows.Next() {
		var targetTable, sourceColumn, targetColumn, relationshipType string
		err := rows.Scan(&targetTable, &sourceColumn, &targetColumn, &relationshipType)
		if err != nil {
			continue
		}

		// Determine relationship type for Cube.js
		relationship := "many_to_one"
		if strings.Contains(strings.ToLower(relationshipType), "one_to_many") {
			relationship = "one_to_many"
		} else if strings.Contains(strings.ToLower(relationshipType), "many_to_many") {
			relationship = "many_to_many"
		} else if strings.Contains(strings.ToLower(relationshipType), "one_to_one") {
			relationship = "one_to_one"
		}

		// Generate Cube.js join SQL
		joinSQL := fmt.Sprintf("{CUBE.%s} = {%s.%s}", sourceColumn, targetTable, targetColumn)

		joinDef := map[string]any{
			"relationship": relationship,
			"sql":          joinSQL,
		}

		joins[targetTable] = joinDef
	}

	return joins, nil
}

// TableColumn represents a database table column
type TableColumn struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	IsNullable   bool   `json:"is_nullable"`
	IsPrimaryKey bool   `json:"is_primary_key"`
}

// GetTableColumns retrieves column information for a table
func (e *DatabaseJoinExtractor) GetTableColumns(tableName string) ([]TableColumn, error) {
	query := `
		SELECT 
			column_name,
			COALESCE(data_type, 'string') as data_type,
			COALESCE(description, '') as description,
			COALESCE(is_nullable, false) as is_nullable,
			COALESCE(is_primary_key, false) as is_primary_key
		FROM catalog_node_vw
		WHERE catalog_type_name = 'column'
		AND table_name = $1
		ORDER BY column_name
	`

	rows, err := e.db.Query(query, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns for table %s: %w", tableName, err)
	}
	defer rows.Close()

	var columns []TableColumn
	for rows.Next() {
		var col TableColumn
		err := rows.Scan(&col.Name, &col.Type, &col.Description, &col.IsNullable, &col.IsPrimaryKey)
		if err != nil {
			continue
		}
		columns = append(columns, col)
	}

	return columns, nil
}

// GenerateCubeFromTable creates a complete Cube definition from database metadata
func (e *DatabaseJoinExtractor) GenerateCubeFromTable(tableName string, datasourceID string) (*Cube, error) {
	// Get table columns
	columns, err := e.GetTableColumns(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Generate joins
	joins, err := e.GenerateJoinDefinitions(tableName, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate joins: %w", err)
	}

	// Create dimensions and measures
	dimensions := make(map[string]map[string]any)
	measures := make(map[string]map[string]any)

	for _, col := range columns {
		// Convert database type to Cube.js type
		cubeType := mapDatabaseTypeToCubeType(col.Type)

		if isNumericType(col.Type) && !col.IsPrimaryKey {
			// Create both dimension and measure for numeric columns
			dimDef := map[string]any{
				"sql":         col.Name,
				"type":        cubeType,
				"title":       formatTitle(col.Name),
				"description": col.Description,
				"public":      true,
			}

			// Add primary key attribute if applicable
			if col.IsPrimaryKey {
				dimDef["primary_key"] = true
			}

			dimensions[col.Name] = dimDef

			// Create common measures for numeric columns
			if !col.IsPrimaryKey && strings.Contains(strings.ToLower(col.Name), "amount") ||
				strings.Contains(strings.ToLower(col.Name), "price") ||
				strings.Contains(strings.ToLower(col.Name), "value") {
				measures[fmt.Sprintf("total_%s", col.Name)] = map[string]any{
					"type":        "sum",
					"sql":         col.Name,
					"title":       fmt.Sprintf("Total %s", formatTitle(col.Name)),
					"description": fmt.Sprintf("Sum of %s", col.Description),
					"format":      "currency",
				}
			}
		} else {
			// Create dimension for non-numeric or primary key columns
			dimDef := map[string]any{
				"sql":         col.Name,
				"type":        cubeType,
				"title":       formatTitle(col.Name),
				"description": col.Description,
				"public":      true,
			}

			if col.IsPrimaryKey {
				dimDef["primary_key"] = true
			}

			dimensions[col.Name] = dimDef
		}
	}

	// Always add a count measure
	measures["count"] = map[string]any{
		"type":        "count",
		"sql":         "id",
		"title":       "Count",
		"description": fmt.Sprintf("Count of %s records", tableName),
	}

	// Generate hierarchies based on common patterns
	hierarchies := e.generateHierarchies(columns, dimensions)

	// Generate drill members (commonly used dimensions)
	drillMembers := e.generateDrillMembers(columns)

	// Create the cube
	cube := &Cube{
		Name:         tableName,
		SQLTable:     tableName,
		Title:        formatTitle(tableName),
		Description:  fmt.Sprintf("Auto-generated cube for %s table", tableName),
		Public:       boolPtr(true),
		Dimensions:   dimensions,
		Measures:     measures,
		Joins:        joins,
		Hierarchies:  hierarchies,
		DrillMembers: drillMembers,
	}

	return cube, nil
}

// Helper functions

func mapDatabaseTypeToCubeType(dbType string) string {
	dbType = strings.ToLower(dbType)
	switch {
	case strings.Contains(dbType, "int") || strings.Contains(dbType, "decimal") ||
		strings.Contains(dbType, "numeric") || strings.Contains(dbType, "float") ||
		strings.Contains(dbType, "double"):
		return "number"
	case strings.Contains(dbType, "timestamp") || strings.Contains(dbType, "datetime") ||
		strings.Contains(dbType, "date"):
		return "time"
	case strings.Contains(dbType, "bool"):
		return "boolean"
	default:
		return "string"
	}
}

func isNumericType(dbType string) bool {
	dbType = strings.ToLower(dbType)
	return strings.Contains(dbType, "int") || strings.Contains(dbType, "decimal") ||
		strings.Contains(dbType, "numeric") || strings.Contains(dbType, "float") ||
		strings.Contains(dbType, "double")
}

func formatTitle(name string) string {
	// Convert snake_case to Title Case
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
		}
	}
	return strings.Join(parts, " ")
}

func boolPtr(b bool) *bool {
	return &b
}

// generateHierarchies creates hierarchies based on common patterns in column names
func (e *DatabaseJoinExtractor) generateHierarchies(columns []TableColumn, dimensions map[string]map[string]any) []map[string]any {
	var hierarchies []map[string]any

	// Look for date/time hierarchies
	var timeColumns []string
	for _, col := range columns {
		if strings.Contains(strings.ToLower(col.Type), "date") ||
			strings.Contains(strings.ToLower(col.Type), "time") {
			if _, exists := dimensions[col.Name]; exists {
				timeColumns = append(timeColumns, col.Name)
			}
		}
	}

	// Create date hierarchy if we have date columns
	if len(timeColumns) > 0 {
		for _, dateCol := range timeColumns {
			hierarchy := map[string]any{
				"name":  fmt.Sprintf("%s_hierarchy", dateCol),
				"title": fmt.Sprintf("%s Hierarchy", formatTitle(dateCol)),
				"levels": []map[string]any{
					{
						"name":             fmt.Sprintf("%s_year", dateCol),
						"title":            "Year",
						"dimension":        dateCol,
						"time_granularity": "year",
					},
					{
						"name":             fmt.Sprintf("%s_quarter", dateCol),
						"title":            "Quarter",
						"dimension":        dateCol,
						"time_granularity": "quarter",
					},
					{
						"name":             fmt.Sprintf("%s_month", dateCol),
						"title":            "Month",
						"dimension":        dateCol,
						"time_granularity": "month",
					},
					{
						"name":             fmt.Sprintf("%s_day", dateCol),
						"title":            "Day",
						"dimension":        dateCol,
						"time_granularity": "day",
					},
				},
			}
			hierarchies = append(hierarchies, hierarchy)
		}
	}

	// Look for geographic hierarchies (country, state, city patterns)
	geoColumns := make(map[string][]string)
	for _, col := range columns {
		colName := strings.ToLower(col.Name)
		if strings.Contains(colName, "country") {
			geoColumns["country"] = append(geoColumns["country"], col.Name)
		} else if strings.Contains(colName, "state") || strings.Contains(colName, "region") {
			geoColumns["state"] = append(geoColumns["state"], col.Name)
		} else if strings.Contains(colName, "city") {
			geoColumns["city"] = append(geoColumns["city"], col.Name)
		}
	}

	// Create geographic hierarchy if we have the right columns
	if len(geoColumns["country"]) > 0 && len(geoColumns["state"]) > 0 {
		var levels []map[string]any

		// Add country level
		levels = append(levels, map[string]any{
			"name":      "country",
			"title":     "Country",
			"dimension": geoColumns["country"][0],
		})

		// Add state level
		levels = append(levels, map[string]any{
			"name":      "state",
			"title":     "State/Region",
			"dimension": geoColumns["state"][0],
		})

		// Add city level if available
		if len(geoColumns["city"]) > 0 {
			levels = append(levels, map[string]any{
				"name":      "city",
				"title":     "City",
				"dimension": geoColumns["city"][0],
			})
		}

		hierarchy := map[string]any{
			"name":   "geography",
			"title":  "Geographic Hierarchy",
			"levels": levels,
		}
		hierarchies = append(hierarchies, hierarchy)
	}

	return hierarchies
}

// generateDrillMembers creates a list of commonly used dimensions for drill-downs
func (e *DatabaseJoinExtractor) generateDrillMembers(columns []TableColumn) []string {
	var drillMembers []string

	// Common drill-down patterns
	drillPatterns := []string{
		"name", "title", "description", "type", "category", "status",
		"country", "state", "city", "region", "location",
		"date", "created", "updated", "modified",
		"user", "customer", "client", "account",
		"product", "service", "item", "code", "id",
	}

	for _, col := range columns {
		colNameLower := strings.ToLower(col.Name)

		// Skip numeric columns that are likely measures
		if isNumericType(col.Type) && !col.IsPrimaryKey {
			continue
		}

		// Check if column matches drill-down patterns
		for _, pattern := range drillPatterns {
			if strings.Contains(colNameLower, pattern) {
				drillMembers = append(drillMembers, col.Name)
				break
			}
		}
	}

	// Remove duplicates and limit to reasonable number
	seen := make(map[string]bool)
	var uniqueDrillMembers []string
	for _, member := range drillMembers {
		if !seen[member] && len(uniqueDrillMembers) < 10 {
			seen[member] = true
			uniqueDrillMembers = append(uniqueDrillMembers, member)
		}
	}

	return uniqueDrillMembers
}
