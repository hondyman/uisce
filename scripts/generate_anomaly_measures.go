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

type AnomalyAwareMeasure struct {
	Name          string
	Type          string
	SQL           string
	Description   string
	SourceTable   string
	SourceColumn  string
	AnomalyConfig AnomalyConfig
	Tags          []string
	Owner         string
	Version       string
	GoldenPath    bool
}

type AnomalyConfig struct {
	Enabled      bool    `json:"enabled"`
	Method       string  `json:"method"`
	Threshold    float64 `json:"threshold"`
	LookbackDays int     `json:"lookback_days"`
}

func main() {
	fmt.Println("🔍 Generating Anomaly-Aware Dynamic Measures")
	fmt.Println("===========================================")

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

	// Generate anomaly-aware measures for different metrics
	fmt.Println("📊 Generating anomaly-aware measures...")

	// Revenue anomaly detection
	revenueMeasures, err := generateRevenueAnomalyMeasures(ctx, db)
	if err != nil {
		fmt.Printf("❌ Failed to generate revenue measures: %v\n", err)
		os.Exit(1)
	}

	// User activity anomaly detection
	activityMeasures, err := generateActivityAnomalyMeasures(ctx, db)
	if err != nil {
		fmt.Printf("❌ Failed to generate activity measures: %v\n", err)
		os.Exit(1)
	}

	// Performance anomaly detection
	performanceMeasures, err := generatePerformanceAnomalyMeasures(ctx, db)
	if err != nil {
		fmt.Printf("❌ Failed to generate performance measures: %v\n", err)
		os.Exit(1)
	}

	allMeasures := append(revenueMeasures, activityMeasures...)
	allMeasures = append(allMeasures, performanceMeasures...)

	// Sync to catalog with anomaly metadata
	fmt.Println("🔄 Syncing anomaly-aware measures to catalog...")
	for _, measure := range allMeasures {
		err := syncAnomalyMeasureToCatalog(ctx, db, measure)
		if err != nil {
			fmt.Printf("❌ Failed to sync measure %s: %v\n", measure.Name, err)
			continue
		}
		fmt.Printf("✅ Synced %s to catalog\n", measure.Name)
	}

	// Write Cube YAML with anomaly dimensions
	fmt.Println("📝 Writing Cube schema files with anomaly support...")
	err = writeAnomalyCubeSchema(allMeasures)
	if err != nil {
		fmt.Printf("❌ Failed to write Cube schema: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("🎉 Anomaly-aware measures generation completed successfully!")
	fmt.Printf("   Generated %d measures\n", len(allMeasures))
	fmt.Println("   Added anomaly detection capabilities")
	fmt.Println("   Created Cube YAML with anomaly dimensions")
}

func generateRevenueAnomalyMeasures(ctx context.Context, db *sql.DB) ([]AnomalyAwareMeasure, error) {
	// Check if we have revenue data
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'transactions' AND column_name = 'revenue'").Scan(&count)
	if err != nil || count == 0 {
		fmt.Println("⚠️  No revenue column found, skipping revenue anomaly measures")
		return []AnomalyAwareMeasure{}, nil
	}

	measures := []AnomalyAwareMeasure{
		{
			Name:         "total_revenue_with_anomaly_flag",
			Type:         "number",
			SQL:          "SUM(revenue)",
			Description:  "Total revenue with anomaly detection based on historical patterns",
			SourceTable:  "transactions",
			SourceColumn: "revenue",
			AnomalyConfig: AnomalyConfig{
				Enabled:      true,
				Method:       "z_score",
				Threshold:    2.5,
				LookbackDays: 30,
			},
			Tags:       []string{"revenue", "anomaly_detection", "financial"},
			Owner:      "system",
			Version:    "v1.0",
			GoldenPath: false,
		},
		{
			Name:         "revenue_anomaly_score",
			Type:         "number",
			SQL:          "ABS(SUM(revenue) - AVG(SUM(revenue)) OVER (ORDER BY DATE_TRUNC('day', created_at) ROWS BETWEEN 29 PRECEDING AND CURRENT ROW)) / STDDEV_POP(SUM(revenue)) OVER (ORDER BY DATE_TRUNC('day', created_at) ROWS BETWEEN 29 PRECEDING AND CURRENT ROW)",
			Description:  "Z-score anomaly detection for revenue patterns",
			SourceTable:  "transactions",
			SourceColumn: "revenue",
			AnomalyConfig: AnomalyConfig{
				Enabled:      true,
				Method:       "z_score",
				Threshold:    2.5,
				LookbackDays: 30,
			},
			Tags:       []string{"revenue", "anomaly_score", "z_score", "financial"},
			Owner:      "system",
			Version:    "v1.0",
			GoldenPath: false,
		},
	}

	return measures, nil
}

func generateActivityAnomalyMeasures(ctx context.Context, db *sql.DB) ([]AnomalyAwareMeasure, error) {
	// Check if we have clickstream data
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'clickstream'").Scan(&count)
	if err != nil || count == 0 {
		fmt.Println("⚠️  No clickstream table found, skipping activity anomaly measures")
		return []AnomalyAwareMeasure{}, nil
	}

	measures := []AnomalyAwareMeasure{
		{
			Name:         "daily_active_users_anomaly",
			Type:         "number",
			SQL:          "COUNT(DISTINCT user_id)",
			Description:  "Daily active users with anomaly detection",
			SourceTable:  "clickstream",
			SourceColumn: "user_id",
			AnomalyConfig: AnomalyConfig{
				Enabled:      true,
				Method:       "iqr",
				Threshold:    1.5,
				LookbackDays: 14,
			},
			Tags:       []string{"users", "activity", "anomaly_detection", "engagement"},
			Owner:      "system",
			Version:    "v1.0",
			GoldenPath: false,
		},
		{
			Name:         "session_duration_anomaly",
			Type:         "number",
			SQL:          "AVG(session_duration)",
			Description:  "Average session duration with anomaly detection",
			SourceTable:  "clickstream",
			SourceColumn: "session_duration",
			AnomalyConfig: AnomalyConfig{
				Enabled:      true,
				Method:       "z_score",
				Threshold:    3.0,
				LookbackDays: 7,
			},
			Tags:       []string{"session", "duration", "anomaly_detection", "engagement"},
			Owner:      "system",
			Version:    "v1.0",
			GoldenPath: false,
		},
	}

	return measures, nil
}

func generatePerformanceAnomalyMeasures(ctx context.Context, db *sql.DB) ([]AnomalyAwareMeasure, error) {
	// Check if we have performance metrics
	var count int
	err := db.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.columns WHERE table_name = 'system_metrics' AND column_name = 'response_time'").Scan(&count)
	if err != nil || count == 0 {
		fmt.Println("⚠️  No performance metrics found, skipping performance anomaly measures")
		return []AnomalyAwareMeasure{}, nil
	}

	measures := []AnomalyAwareMeasure{
		{
			Name:         "avg_response_time_anomaly",
			Type:         "number",
			SQL:          "AVG(response_time)",
			Description:  "Average response time with anomaly detection for performance monitoring",
			SourceTable:  "system_metrics",
			SourceColumn: "response_time",
			AnomalyConfig: AnomalyConfig{
				Enabled:      true,
				Method:       "z_score",
				Threshold:    2.0,
				LookbackDays: 1,
			},
			Tags:       []string{"performance", "response_time", "anomaly_detection", "monitoring"},
			Owner:      "system",
			Version:    "v1.0",
			GoldenPath: false,
		},
		{
			Name:         "error_rate_anomaly",
			Type:         "number",
			SQL:          "SUM(CASE WHEN status_code >= 400 THEN 1 ELSE 0 END)::FLOAT / COUNT(*) * 100",
			Description:  "Error rate percentage with anomaly detection",
			SourceTable:  "system_metrics",
			SourceColumn: "status_code",
			AnomalyConfig: AnomalyConfig{
				Enabled:      true,
				Method:       "threshold",
				Threshold:    5.0, // 5% error rate threshold
				LookbackDays: 1,
			},
			Tags:       []string{"performance", "error_rate", "anomaly_detection", "monitoring"},
			Owner:      "system",
			Version:    "v1.0",
			GoldenPath: false,
		},
	}

	return measures, nil
}

func syncAnomalyMeasureToCatalog(ctx context.Context, db *sql.DB, measure AnomalyAwareMeasure) error {
	tagsJSON := fmt.Sprintf(`["%s"]`, strings.Join(measure.Tags, `","`))

	anomalyConfigJSON := fmt.Sprintf(`{
		"enabled": %t,
		"method": "%s",
		"threshold": %f,
		"lookback_days": %d
	}`, measure.AnomalyConfig.Enabled, measure.AnomalyConfig.Method,
		measure.AnomalyConfig.Threshold, measure.AnomalyConfig.LookbackDays)

	_, err := db.ExecContext(ctx, `
		INSERT INTO public.catalog_node (
			node_id, node_type, name, description, schema_def, version,
			created_by, created_at, updated_at, tags, golden_path
		)
		VALUES ($1, 'anomaly_aware_measure', $2, $3, $4::jsonb, $5, $6, $7, $8, $9::jsonb, $10)
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
			"source_table": "%s",
			"source_column": "%s",
			"anomaly_config": %s,
			"tags": ["%s"]
		}`, measure.Name, measure.Type, measure.SQL, measure.SourceTable,
			measure.SourceColumn, anomalyConfigJSON, strings.Join(measure.Tags, `","`)),
		measure.Version, measure.Owner, time.Now().UTC(), time.Now().UTC(), tagsJSON, measure.GoldenPath)

	return err
}

func writeAnomalyCubeSchema(measures []AnomalyAwareMeasure) error {
	// Group measures by source table
	measureGroups := make(map[string][]AnomalyAwareMeasure)

	for _, measure := range measures {
		measureGroups[measure.SourceTable] = append(measureGroups[measure.SourceTable], measure)
	}

	// Write schema files for each table
	for table, tableMeasures := range measureGroups {
		var yamlMeasures []string
		var yamlDimensions []string

		for _, measure := range tableMeasures {
			// Add the measure
			yamlMeasure := fmt.Sprintf(`  - name: %s
    type: %s
    sql: %s
    description: "%s"`,
				measure.Name, measure.Type, measure.SQL, measure.Description)
			yamlMeasures = append(yamlMeasures, yamlMeasure)

			// Add anomaly flag dimension if anomaly detection is enabled
			if measure.AnomalyConfig.Enabled {
				anomalyDim := fmt.Sprintf(`  - name: %s_anomaly_flag
    sql: |
      CASE
        WHEN %s > %f THEN true
        ELSE false
      END
    type: boolean
    description: "Anomaly flag for %s (%s method)"`,
					measure.Name, getAnomalySQL(measure), measure.AnomalyConfig.Threshold,
					measure.Name, measure.AnomalyConfig.Method)
				yamlDimensions = append(yamlDimensions, anomalyDim)
			}
		}

		yamlContent := fmt.Sprintf(`# Auto-generated anomaly-aware measures for %s
# Generated at: %s
# Includes anomaly detection capabilities

cubes:
  - name: %s_anomaly_cube
    sql: SELECT * FROM %s

    measures:
%s

    dimensions:
%s

# Anomaly detection configuration
# Method: Configured per measure
# Threshold: Configured per measure
# Lookback: Configured per measure
`, table, time.Now().UTC().Format(time.RFC3339), table, table, strings.Join(yamlMeasures, "\n"), strings.Join(yamlDimensions, "\n"))

		filename := fmt.Sprintf("cube/schema/%s_anomaly_measures.yml", table)
		err := os.MkdirAll("cube/schema", 0755)
		if err != nil {
			return fmt.Errorf("failed to create cube/schema directory: %w", err)
		}

		err = os.WriteFile(filename, []byte(yamlContent), 0644)
		if err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}

		fmt.Printf("📝 Wrote %s with %d measures and anomaly dimensions\n", filename, len(tableMeasures))
	}

	return nil
}

func getAnomalySQL(measure AnomalyAwareMeasure) string {
	switch measure.AnomalyConfig.Method {
	case "z_score":
		return fmt.Sprintf("ABS(%s - AVG(%s) OVER (ORDER BY created_at ROWS BETWEEN %d PRECEDING AND CURRENT ROW)) / STDDEV_POP(%s) OVER (ORDER BY created_at ROWS BETWEEN %d PRECEDING AND CURRENT ROW)",
			measure.SQL, measure.SQL, measure.AnomalyConfig.LookbackDays-1, measure.SQL, measure.AnomalyConfig.LookbackDays-1)
	case "iqr":
		return fmt.Sprintf("%s - PERCENTILE_CONT(0.75) WITHIN GROUP (ORDER BY %s) OVER (ORDER BY created_at ROWS BETWEEN %d PRECEDING AND CURRENT ROW)",
			measure.SQL, measure.SQL, measure.AnomalyConfig.LookbackDays-1)
	case "threshold":
		return measure.SQL
	default:
		return measure.SQL
	}
}
