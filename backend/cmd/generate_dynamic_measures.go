//go:build ignore

package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type DynamicMeasure struct {
	Name        string
	Type        string
	SQL         string
	Description string
	SourceEnum  string
	Tags        []string
	Owner       string
	Version     string
	GoldenPath  bool
}

func main() {
	fmt.Println("🔄 Syncing Dynamic Measures from Postgres to Cube")
	fmt.Println("=================================================")

	// Database connection
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:password@localhost:5432/semlayer?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		fmt.Printf("❌ Failed to connect to database: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	ctx := context.Background()

	// Generate measures from orders.status
	fmt.Println("📊 Generating measures from orders.status...")
	measures, err := generateMeasuresFromEnum(ctx, db, "orders", "status")
	if err != nil {
		fmt.Printf("❌ Failed to generate measures: %v\n", err)
		os.Exit(1)
	}

	// Generate measures from products.category
	fmt.Println("📦 Generating measures from products.category...")
	categoryMeasures, err := generateMeasuresFromEnum(ctx, db, "products", "category")
	if err != nil {
		fmt.Printf("❌ Failed to generate category measures: %v\n", err)
		os.Exit(1)
	}
	measures = append(measures, categoryMeasures...)

	// Generate measures from clickstream.device_type
	fmt.Println("📱 Generating measures from clickstream.device_type...")
	deviceMeasures, err := generateMeasuresFromEnum(ctx, db, "clickstream", "device_type")
	if err != nil {
		fmt.Printf("❌ Failed to generate device measures: %v\n", err)
		os.Exit(1)
	}
	measures = append(measures, deviceMeasures...)

	// Sync to catalog
	fmt.Println("🔄 Syncing to catalog...")
	for _, measure := range measures {
		err := syncToCatalog(ctx, db, measure)
		if err != nil {
			fmt.Printf("❌ Failed to sync measure %s: %v\n", measure.Name, err)
			continue
		}
		fmt.Printf("✅ Synced %s to catalog\n", measure.Name)
	}

	// Write Cube YAML
	fmt.Println("📝 Writing Cube schema files...")
	err = writeCubeSchema(measures)
	if err != nil {
		fmt.Printf("❌ Failed to write Cube schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🎉 Dynamic measures sync completed successfully!")
	fmt.Printf("   Generated %d measures\n", len(measures))
	fmt.Println("   Updated catalog with governance metadata")
	fmt.Println("   Created Cube YAML schema files")
}

func generateMeasuresFromEnum(ctx context.Context, db *sql.DB, table, column string) ([]DynamicMeasure, error) {
	// TODO: Refactor to Hasura GraphQL
	// query {
	//   orders(distinct_on: status, where: {status: {_is_null: false}}, order_by: {status: asc}) {
	//     status
	//   }
	// }
	// Or use aggregate for distinct values
	query := fmt.Sprintf("SELECT DISTINCT %s FROM %s WHERE %s IS NOT NULL ORDER BY %s", column, table, column, column)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query %s.%s: %w", table, column, err)
	}
	defer rows.Close()

	var measures []DynamicMeasure
	for rows.Next() {
		var value string
		err := rows.Scan(&value)
		if err != nil {
			return nil, fmt.Errorf("failed to scan value: %w", err)
		}

		// Clean the value for use in measure names
		cleanValue := strings.ReplaceAll(strings.ToLower(value), " ", "_")
		cleanValue = strings.ReplaceAll(cleanValue, "-", "_")

		measure := DynamicMeasure{
			Name:        fmt.Sprintf("total_%s_%s", cleanValue, strings.TrimSuffix(table, "s")),
			Type:        "count",
			SQL:         fmt.Sprintf("CASE WHEN %s = '%s' THEN 1 ELSE 0 END", column, value),
			Description: fmt.Sprintf("Total count of %s records with %s = '%s'", table, column, value),
			SourceEnum:  fmt.Sprintf("%s.%s", table, column),
			Tags:        []string{table, column, cleanValue},
			Owner:       "system",
			Version:     "v1.0",
			GoldenPath:  false,
		}
		measures = append(measures, measure)
	}

	return measures, nil
}

func syncToCatalog(ctx context.Context, db *sql.DB, measure DynamicMeasure) error {
	tagsJSON := fmt.Sprintf(`["%s"]`, strings.Join(measure.Tags, `","`))

	// TODO: Refactor to Hasura GraphQL
	// mutation {
	//   insert_catalog_node_one(
	//     object: {
	//       node_id: "measure_name", node_type: "dynamic_measure", name: "measure_name"
	//       description: "description", schema_def: {name: "...", type: "count", sql: "..."}
	//       version: "v1.0", created_by: "system", tags: ["tag1", "tag2"], golden_path: false
	//     }
	//     on_conflict: {constraint: catalog_node_pkey, update_columns: [description, schema_def, version, updated_at, tags, golden_path]}
	//   ) { node_id }
	// }
	_, err := db.ExecContext(ctx, `
		INSERT INTO public.catalog_node (
			node_id, node_type, name, description, schema_def, version,
			created_by, created_at, updated_at, tags, golden_path
		)
		VALUES ($1, 'dynamic_measure', $2, $3, $4::jsonb, $5, $6, $7, $8, $9::jsonb, $10)
		ON CONFLICT (node_id) DO UPDATE SET
			description = EXCLUDED.description,
			schema_def = EXCLUDED.schema_def,
			version = EXCLUDED.version,
			updated_at = EXCLUDED.updated_at,
			tags = EXCLUDED.tags,
			golden_path = EXCLUDED.golden_path
	`, measure.Name, measure.Name, measure.Description,
		fmt.Sprintf(`{
			"name": "%s",
			"type": "%s",
			"sql": "%s",
			"source_enum": "%s",
			"tags": ["%s"]
		}`, measure.Name, measure.Type, measure.SQL, measure.SourceEnum, strings.Join(measure.Tags, `","`)),
		measure.Version, measure.Owner, time.Now().UTC(), time.Now().UTC(), tagsJSON, measure.GoldenPath)

	return err
}

func writeCubeSchema(measures []DynamicMeasure) error {
	// Group measures by source table
	measureGroups := make(map[string][]DynamicMeasure)

	for _, measure := range measures {
		table := strings.Split(measure.SourceEnum, ".")[0]
		measureGroups[table] = append(measureGroups[table], measure)
	}

	// Write schema files for each table
	for table, tableMeasures := range measureGroups {
		var yamlMeasures []string

		for _, measure := range tableMeasures {
			yamlMeasure := fmt.Sprintf(`  - name: %s
    type: %s
    sql: %s
    description: "%s"`,
				measure.Name, measure.Type, measure.SQL, measure.Description)
			yamlMeasures = append(yamlMeasures, yamlMeasure)
		}

		yamlContent := fmt.Sprintf(`# Auto-generated dynamic measures for %s
# Generated at: %s
# Source: %s

measures:
%s
`, table, time.Now().UTC().Format(time.RFC3339), "generate_dynamic_measures.go", strings.Join(yamlMeasures, "\n"))

		filename := fmt.Sprintf("cube/schema/%s_dynamic_measures.yml", table)
		err := os.MkdirAll("cube/schema", 0755)
		if err != nil {
			return fmt.Errorf("failed to create cube/schema directory: %w", err)
		}

		err = os.WriteFile(filename, []byte(yamlContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}

		fmt.Printf("📝 Wrote %s with %d measures\n", filename, len(tableMeasures))
	}

	return nil
}
