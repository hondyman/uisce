//go:build ignore

package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

type DynamicUnion struct {
	NodeID       string            `json:"node_id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	SourceTables []string          `json:"source_tables"`
	UnionType    string            `json:"union_type"`
	TableAliases map[string]string `json:"table_aliases"`
	Owner        string            `json:"owner"`
	Version      string            `json:"version"`
}

type StringTimeDimension struct {
	NodeID          string `json:"node_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	SourceColumn    string `json:"source_column"`
	SourceTable     string `json:"source_table"`
	DateFormat      string `json:"date_format"`
	TimeFormat      string `json:"time_format"`
	Timezone        string `json:"timezone"`
	ParsingFunction string `json:"parsing_function"`
	FallbackValue   string `json:"fallback_value"`
	Owner           string `json:"owner"`
	Version         string `json:"version"`
}

type CustomGranularity struct {
	NodeID       string `json:"node_id"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Dimension    string `json:"dimension"`
	Interval     string `json:"interval"`
	OffsetDays   int    `json:"offset_days"`
	OffsetHours  int    `json:"offset_hours"`
	FiscalLabel  string `json:"fiscal_label"`
	CalendarType string `json:"calendar_type"`
	WeekStartDay string `json:"week_start_day"`
	Owner        string `json:"owner"`
	Version      string `json:"version"`
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

	// Generate Cube.js YAML for dynamic unions
	generateUnionCubes(db)

	// Generate Cube.js YAML for string time dimensions
	generateTimeDimensionCubes(db)

	// Generate Cube.js YAML for custom granularities
	generateGranularityCubes(db)

	fmt.Println("✅ Advanced modeling patterns generated successfully!")
}

func generateUnionCubes(db *sql.DB) {
	rows, err := db.Query(`
		SELECT node_id, name, description, schema_def
		FROM public.catalog_node
		WHERE node_type = 'dynamic_union'
		AND review_status = 'approved'
	`)
	if err != nil {
		log.Printf("Failed to fetch unions: %v", err)
		return
	}
	defer rows.Close()

	var unions []DynamicUnion
	for rows.Next() {
		var nodeID, name, description, schemaJSON string
		rows.Scan(&nodeID, &name, &description, &schemaJSON)

		var schema map[string]interface{}
		json.Unmarshal([]byte(schemaJSON), &schema)

		sourceTables := make([]string, 0)
		if tables, ok := schema["source_tables"].([]interface{}); ok {
			for _, t := range tables {
				if tableStr, ok := t.(string); ok {
					sourceTables = append(sourceTables, tableStr)
				}
			}
		}

		tableAliases := make(map[string]string)
		if aliases, ok := schema["table_aliases"].(map[string]interface{}); ok {
			for k, v := range aliases {
				if aliasStr, ok := v.(string); ok {
					tableAliases[k] = aliasStr
				}
			}
		}

		unions = append(unions, DynamicUnion{
			NodeID:       nodeID,
			Name:         name,
			Description:  description,
			SourceTables: sourceTables,
			UnionType:    getStringValue(schema, "union_type", "UNION ALL"),
			TableAliases: tableAliases,
			Owner:        getStringValue(schema, "owner", "system"),
			Version:      getStringValue(schema, "version", "1.0.0"),
		})
	}

	// Generate Cube.js YAML
	var cubeDefs []string
	for _, union := range unions {
		cubeDef := generateUnionCubeYAML(union)
		cubeDefs = append(cubeDefs, cubeDef)
	}

	output := strings.Join(cubeDefs, "\n\n")
	os.WriteFile("cube/schema/dynamic_unions.yml", []byte(output), 0644)
	fmt.Printf("📄 Generated %d union cubes\n", len(unions))
}

func generateTimeDimensionCubes(db *sql.DB) {
	rows, err := db.Query(`
		SELECT node_id, name, description, schema_def
		FROM public.catalog_node
		WHERE node_type = 'string_time_dimension'
		AND review_status = 'approved'
	`)
	if err != nil {
		log.Printf("Failed to fetch time dimensions: %v", err)
		return
	}
	defer rows.Close()

	var timeDims []StringTimeDimension
	for rows.Next() {
		var nodeID, name, description, schemaJSON string
		rows.Scan(&nodeID, &name, &description, &schemaJSON)

		var schema map[string]interface{}
		json.Unmarshal([]byte(schemaJSON), &schema)

		timeDims = append(timeDims, StringTimeDimension{
			NodeID:          nodeID,
			Name:            name,
			Description:     description,
			SourceColumn:    getStringValue(schema, "source_column", ""),
			SourceTable:     getStringValue(schema, "source_table", ""),
			DateFormat:      getStringValue(schema, "date_format", "YYYY-MM-DD"),
			TimeFormat:      getStringValue(schema, "time_format", ""),
			Timezone:        getStringValue(schema, "timezone", "UTC"),
			ParsingFunction: getStringValue(schema, "parsing_function", "TO_TIMESTAMP"),
			FallbackValue:   getStringValue(schema, "fallback_value", ""),
			Owner:           getStringValue(schema, "owner", "system"),
			Version:         getStringValue(schema, "version", "1.0.0"),
		})
	}

	// Generate Cube.js YAML
	var cubeDefs []string
	for _, timeDim := range timeDims {
		cubeDef := generateTimeDimensionCubeYAML(timeDim)
		cubeDefs = append(cubeDefs, cubeDef)
	}

	output := strings.Join(cubeDefs, "\n\n")
	os.WriteFile("cube/schema/string_time_dimensions.yml", []byte(output), 0644)
	fmt.Printf("📄 Generated %d time dimension cubes\n", len(timeDims))
}

func generateGranularityCubes(db *sql.DB) {
	rows, err := db.Query(`
		SELECT node_id, name, description, schema_def
		FROM public.catalog_node
		WHERE node_type = 'custom_granularity'
		AND review_status = 'approved'
	`)
	if err != nil {
		log.Printf("Failed to fetch granularities: %v", err)
		return
	}
	defer rows.Close()

	var granularities []CustomGranularity
	for rows.Next() {
		var nodeID, name, description, schemaJSON string
		rows.Scan(&nodeID, &name, &description, &schemaJSON)

		var schema map[string]interface{}
		json.Unmarshal([]byte(schemaJSON), &schema)

		granularities = append(granularities, CustomGranularity{
			NodeID:       nodeID,
			Name:         name,
			Description:  description,
			Dimension:    getStringValue(schema, "dimension", ""),
			Interval:     getStringValue(schema, "interval", "month"),
			OffsetDays:   getIntValue(schema, "offset_days", 0),
			OffsetHours:  getIntValue(schema, "offset_hours", 0),
			FiscalLabel:  getStringValue(schema, "fiscal_label", ""),
			CalendarType: getStringValue(schema, "calendar_type", "gregorian"),
			WeekStartDay: getStringValue(schema, "week_start_day", "monday"),
			Owner:        getStringValue(schema, "owner", "system"),
			Version:      getStringValue(schema, "version", "1.0.0"),
		})
	}

	// Generate Cube.js YAML
	var cubeDefs []string
	for _, granularity := range granularities {
		cubeDef := generateGranularityCubeYAML(granularity)
		cubeDefs = append(cubeDefs, cubeDef)
	}

	output := strings.Join(cubeDefs, "\n\n")
	os.WriteFile("cube/schema/custom_granularities.yml", []byte(output), 0644)
	fmt.Printf("📄 Generated %d granularity cubes\n", len(granularities))
}

func generateUnionCubeYAML(union DynamicUnion) string {
	cubeName := strings.ToLower(strings.ReplaceAll(union.Name, " ", "_"))

	var selectParts []string
	for i, table := range union.SourceTables {
		alias := union.TableAliases[table]
		if alias == "" {
			alias = fmt.Sprintf("t%d", i+1)
		}
		selectParts = append(selectParts, fmt.Sprintf("SELECT *, '%s' AS source_table FROM %s AS %s", table, table, alias))
	}

	sql := strings.Join(selectParts, fmt.Sprintf("\n%s\n", union.UnionType))

	return fmt.Sprintf(`cubes:
  - name: %s
    sql: |
%s
    description: "%s"
    meta:
      owner: "%s"
      version: "%s"
      node_id: "%s"

    dimensions:
      - name: source_table
        sql: source_table
        type: string
        description: "Source table for this record"

    # Add your measures and additional dimensions here
    measures: []`,
		cubeName, indentSQL(sql, 6), union.Description, union.Owner, union.Version, union.NodeID)
}

func generateTimeDimensionCubeYAML(timeDim StringTimeDimension) string {
	cubeName := strings.ToLower(strings.ReplaceAll(timeDim.Name, " ", "_"))

	var parsingSQL string
	if timeDim.TimeFormat != "" {
		parsingSQL = fmt.Sprintf("%s(%s, '%s %s', '%s')",
			timeDim.ParsingFunction, timeDim.SourceColumn,
			timeDim.DateFormat, timeDim.TimeFormat, timeDim.Timezone)
	} else {
		parsingSQL = fmt.Sprintf("%s(%s, '%s')",
			timeDim.ParsingFunction, timeDim.SourceColumn, timeDim.DateFormat)
	}

	if timeDim.FallbackValue != "" {
		parsingSQL = fmt.Sprintf("COALESCE(%s, '%s')", parsingSQL, timeDim.FallbackValue)
	}

	return fmt.Sprintf(`cubes:
  - name: %s
    sql: SELECT * FROM %s
    description: "%s"
    meta:
      owner: "%s"
      version: "%s"
      node_id: "%s"

    dimensions:
      - name: parsed_timestamp
        sql: %s
        type: time
        description: "Parsed timestamp from string column"

      - name: original_string
        sql: %s
        type: string
        description: "Original string value"

    # Add your measures here
    measures: []`,
		cubeName, timeDim.SourceTable, timeDim.Description,
		timeDim.Owner, timeDim.Version, timeDim.NodeID,
		parsingSQL, timeDim.SourceColumn)
}

func generateGranularityCubeYAML(granularity CustomGranularity) string {
	cubeName := strings.ToLower(strings.ReplaceAll(granularity.Name, " ", "_"))

	var granularitySQL string
	switch granularity.CalendarType {
	case "fiscal":
		granularitySQL = fmt.Sprintf(`
			CASE
				WHEN EXTRACT(MONTH FROM %s + INTERVAL '%d days') >= 7
				THEN CONCAT('FY', EXTRACT(YEAR FROM %s + INTERVAL '%d days') + 1)
				ELSE CONCAT('FY', EXTRACT(YEAR FROM %s + INTERVAL '%d days'))
			END`, granularity.Dimension, granularity.OffsetDays,
			granularity.Dimension, granularity.OffsetDays,
			granularity.Dimension, granularity.OffsetDays)
	case "iso_week":
		granularitySQL = fmt.Sprintf("EXTRACT(ISOYEAR FROM %s) || '-W' || LPAD(EXTRACT(ISOWEEK FROM %s)::TEXT, 2, '0')",
			granularity.Dimension, granularity.Dimension)
	default:
		if granularity.OffsetDays != 0 || granularity.OffsetHours != 0 {
			granularitySQL = fmt.Sprintf("DATE_TRUNC('%s', %s + INTERVAL '%d days %d hours')",
				granularity.Interval, granularity.Dimension, granularity.OffsetDays, granularity.OffsetHours)
		} else {
			granularitySQL = fmt.Sprintf("DATE_TRUNC('%s', %s)", granularity.Interval, granularity.Dimension)
		}
	}

	return fmt.Sprintf(`cubes:
  - name: %s
    sql: SELECT * FROM your_base_table
    description: "%s"
    meta:
      owner: "%s"
      version: "%s"
      node_id: "%s"

    dimensions:
      - name: %s
        sql: %s
        type: time
        description: "Custom %s granularity"

    granularities:
      - name: %s
        interval: 1 %s
        offset: %d days
        title: "%s"

    # Add your measures here
    measures: []`,
		cubeName, granularity.Description, granularity.Owner, granularity.Version, granularity.NodeID,
		strings.ToLower(strings.ReplaceAll(granularity.Name, " ", "_")),
		strings.TrimSpace(granularitySQL), granularity.Interval,
		strings.ToLower(strings.ReplaceAll(granularity.Name, " ", "_")),
		granularity.Interval, granularity.OffsetDays, granularity.FiscalLabel)
}

func getStringValue(schema map[string]interface{}, key, defaultValue string) string {
	if val, ok := schema[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

func getIntValue(schema map[string]interface{}, key string, defaultValue int) int {
	if val, ok := schema[key]; ok {
		if intVal, ok := val.(float64); ok {
			return int(intVal)
		}
	}
	return defaultValue
}

func indentSQL(sql string, spaces int) string {
	indent := strings.Repeat(" ", spaces)
	lines := strings.Split(sql, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = indent + line
		}
	}
	return strings.Join(lines, "\n")
}
