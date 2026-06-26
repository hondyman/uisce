package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Field struct {
	Name  string `json:"name"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

type BusinessObject struct {
	ID          string
	Name        string
	DisplayName string
	FieldsRaw   []byte
	Fields      []Field
}

func main() {
	dbURL := "postgres://postgres:postgres@localhost/alpha?sslmode=disable"
	db, err := sql.Open("pgx", dbURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	rows, err := db.Query("SELECT name, display_name, fields FROM public.business_objects")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Query failed: %v\n", err)
		os.Exit(1)
	}
	defer rows.Close()

	var businessObjects []BusinessObject
	for rows.Next() {
		var bo BusinessObject
		if err := rows.Scan(&bo.Name, &bo.DisplayName, &bo.FieldsRaw); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to scan row: %v\n", err)
			continue
		}
		if err := json.Unmarshal(bo.FieldsRaw, &bo.Fields); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to unmarshal fields for %s: %v\n", bo.Name, err)
			continue
		}
		businessObjects = append(businessObjects, bo)
	}

	schemaDir := "schema"
	if _, err := os.Stat(schemaDir); os.IsNotExist(err) {
		os.Mkdir(schemaDir, 0755)
	}

	for _, bo := range businessObjects {
		yamlContent := generateYAML(bo)
		err := saveYAMLToFile(schemaDir, bo.Name, yamlContent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save YAML for %s: %v\n", bo.Name, err)
		}

		err = saveModelToRegistry(db, bo, yamlContent)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to save model to registry for %s: %v\n", bo.Name, err)
		}
	}

	fmt.Println("Successfully generated and stored Cube models.")
}

func generateYAML(bo BusinessObject) string {
	var builder strings.Builder
	builder.WriteString("cubes:\n")
	builder.WriteString(fmt.Sprintf("  - name: %s\n", strings.Title(bo.Name)))
	builder.WriteString(fmt.Sprintf("    sql_table: %s\n", bo.Name))
	builder.WriteString(fmt.Sprintf("    title: %s\n", bo.DisplayName))
	builder.WriteString("\n    measures:\n")
	builder.WriteString("      - name: count\n")
	builder.WriteString("        type: count\n")

	// Add a sum measure for currency types
	for _, field := range bo.Fields {
		if field.Type == "currency" {
			builder.WriteString(fmt.Sprintf("      - name: total_%s\n", field.Name))
			builder.WriteString(fmt.Sprintf("        sql: %s\n", field.Name))
			builder.WriteString("        type: sum\n")
		}
	}

	builder.WriteString("\n    dimensions:\n")
	for _, field := range bo.Fields {
		builder.WriteString(fmt.Sprintf("      - name: %s\n", field.Name))
		builder.WriteString(fmt.Sprintf("        sql: %s\n", field.Name))
		cubeType := "string"
		if field.Type == "currency" || field.Type == "number" {
			cubeType = "number"
		} else if field.Type == "date" || field.Type == "datetime" {
			cubeType = "time"
		}
		builder.WriteString(fmt.Sprintf("        type: %s\n", cubeType))
		if field.Name == "id" {
			builder.WriteString("        primary_key: true\n")
		}
		builder.WriteString(fmt.Sprintf("        title: %s\n", field.Label))
	}

	return builder.String()
}

func saveYAMLToFile(schemaDir, name, content string) error {
	filePath := filepath.Join(schemaDir, fmt.Sprintf("%s.yml", strings.Title(name)))
	return os.WriteFile(filePath, []byte(content), 0644)
}

func saveModelToRegistry(db *sql.DB, bo BusinessObject, yamlContent string) error {
	jsonData, err := json.Marshal(yamlContent)
	if err != nil {
		return fmt.Errorf("failed to marshal yaml to json string: %w", err)
	}

	query := `
		INSERT INTO public.template_registry (template_name, template_type, description, template_data, version, node_id, domain)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (node_id) DO UPDATE SET
			template_type = EXCLUDED.template_type,
			description = EXCLUDED.description,
			template_data = EXCLUDED.template_data,
			version = EXCLUDED.version,
			updated_at = NOW();
	`
	_, err = db.Exec(query, bo.Name, "cubejs_model", bo.DisplayName, jsonData, "1.0.0", bo.Name, "business_objects")
	return err
}
