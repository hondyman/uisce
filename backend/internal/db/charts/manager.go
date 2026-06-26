package charts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
)

// saveChart saves a chart to the database with compression
func saveChart(ctx context.Context, tx *sql.Tx, datasourceId string, chartName string, chart interface{}) error {
	chartJSON, err := json.Marshal(chart)
	if err != nil {
		return fmt.Errorf("marshal chart: %w", err)
	}

	log.Printf("=== CHART DATA BEFORE COMPRESSION ===")
	log.Printf("Chart JSON size: %d bytes", len(chartJSON))
	if len(chartJSON) > 1000 {
		log.Printf("Chart JSON preview (first 1000 chars): %s...", string(chartJSON[:1000]))
	} else {
		log.Printf("Chart JSON full content: %s", string(chartJSON))
	}

	compressedChart, err := compressData(chartJSON)
	if err != nil {
		return fmt.Errorf("compress chart: %w", err)
	}

	// TODO: Replace with Hasura GraphQL mutation (upsert pattern):
	//   mutation { insert_tenant_chart_one(object: {tenant_datasource_id: $datasource_id, chart_name: $name, chart: $compressed}, on_conflict: {constraint: tenant_chart_pkey, update_columns: [chart, updated_at]}) { chart_name } }
	//   Note: chart field is bytea (compressed), NOW() becomes now()
	// Delete existing chart first (upsert workaround)
	_, err = tx.ExecContext(ctx, `
		DELETE FROM public.tenant_chart 
		WHERE tenant_datasource_id = $1 AND chart_name = $2`,
		datasourceId, chartName)
	if err != nil {
		return fmt.Errorf("delete existing chart: %w", err)
	}

	// Insert new chart
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.tenant_chart (tenant_datasource_id, chart_name, chart, updated_at)
		VALUES ($1, $2, $3, NOW())`,
		datasourceId, chartName, compressedChart)
	if err != nil {
		return fmt.Errorf("save chart: %w", err)
	}

	log.Printf("Successfully saved chart %s for datasource %s", chartName, datasourceId)
	return tx.Commit()
}

// GetLineageData retrieves lineage data from the database
func GetLineageData(ctx context.Context, db *sql.DB, datasourceId string, lineageType string) ([]byte, error) {
	var chartName string
	switch lineageType {
	case "technical":
		chartName = "technical_lineage_chart"
	case "semantic":
		chartName = "semantic_lineage_chart"
	case "enhanced":
		chartName = "enhanced_erd_chart"
	default:
		chartName = "erd_chart"
	}

	var chartData []byte
	// TODO: Replace with Hasura GraphQL query:
	//   query { tenant_chart(where: {tenant_datasource_id: {_eq: $datasource_id}, chart_name: {_eq: $chart_name}}) { chart } }
	//   Handle sql.ErrNoRows equivalent (empty array)
	err := db.QueryRowContext(ctx,
		`SELECT chart FROM public.tenant_chart 
		 WHERE tenant_datasource_id = $1 AND chart_name = $2`,
		datasourceId, chartName).Scan(&chartData)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no chart found for tenant datasource %s with name %s", datasourceId, chartName)
		}
		return nil, fmt.Errorf("get lineage data: %w", err)
	}

	return chartData, nil
}

// ParseChartData decompresses and parses chart data
func ParseChartData(compressedData []byte, chartType string) (interface{}, error) {
	decompressedData, err := decompressData(compressedData)
	if err != nil {
		return nil, fmt.Errorf("decompress chart data: %w", err)
	}

	switch chartType {
	case "technical", "erd", "enhanced":
		var chart TechnicalLineageChart
		if err := json.Unmarshal(decompressedData, &chart); err != nil {
			return nil, fmt.Errorf("unmarshal technical chart: %w", err)
		}
		return chart, nil
	case "semantic":
		var chart SemanticLineageChart
		if err := json.Unmarshal(decompressedData, &chart); err != nil {
			return nil, fmt.Errorf("unmarshal semantic chart: %w", err)
		}
		return chart, nil
	default:
		return nil, fmt.Errorf("unknown chart type: %s", chartType)
	}
}

// ListChartsForDatasource lists all charts for a datasource
func ListChartsForDatasource(ctx context.Context, db *sql.DB, datasourceId string) ([]ChartInfo, error) {
	// TODO: Replace with Hasura GraphQL query:
	//   query { tenant_chart(where: {tenant_datasource_id: {_eq: $datasource_id}}, order_by: {chart_name: asc}) { chart_name created_at updated_at } }
	//   Calculate size_bytes client-side or add computed column: length(chart)
	rows, err := db.QueryContext(ctx, `
		SELECT chart_name, 
		       length(chart) as size_bytes,
		       created_at, 
		       updated_at
		FROM public.tenant_chart 
		WHERE tenant_datasource_id = $1
		ORDER BY chart_name`, datasourceId)
	if err != nil {
		return nil, fmt.Errorf("query chart list: %w", err)
	}
	defer rows.Close()

	var charts []ChartInfo
	for rows.Next() {
		var chart ChartInfo
		if err := rows.Scan(&chart.Name, &chart.SizeBytes, &chart.CreatedAt, &chart.UpdatedAt); err != nil {
			log.Printf("Error scanning chart row: %v", err)
			continue
		}

		chart.Type = mapDBChartNameToLineageType(chart.Name)
		charts = append(charts, chart)
	}

	return charts, rows.Err()
}

// RefreshAllCharts rebuilds all charts for a datasource
func RefreshAllCharts(ctx context.Context, db *sql.DB, datasourceId string, isGoldCopy bool) error {
	log.Printf("Refreshing all charts for datasource %s", datasourceId)

	if err := BuildERDChart(ctx, db, datasourceId, isGoldCopy); err != nil {
		log.Printf("Failed to build ERD chart: %v", err)
		return fmt.Errorf("build ERD chart: %w", err)
	}

	if err := BuildEnhancedERDChart(ctx, db, datasourceId, isGoldCopy); err != nil {
		log.Printf("Failed to build enhanced ERD chart: %v", err)
		return fmt.Errorf("build enhanced ERD chart: %w", err)
	}

	if err := BuildTechnicalLineageChart(ctx, db, datasourceId, isGoldCopy); err != nil {
		log.Printf("Failed to build technical lineage chart: %v", err)
		return fmt.Errorf("build technical lineage chart: %w", err)
	}

	if err := BuildSemanticLineageChart(ctx, db, datasourceId, isGoldCopy); err != nil {
		log.Printf("Failed to build semantic lineage chart: %v", err)
		return fmt.Errorf("build semantic lineage chart: %w", err)
	}

	log.Printf("Successfully refreshed all charts for datasource %s", datasourceId)
	return nil
}

// DeleteChartData removes a chart from the database
func DeleteChartData(ctx context.Context, db *sql.DB, datasourceId string, chartType string) error {
	var chartName string
	switch chartType {
	case "technical":
		chartName = "technical_lineage_chart"
	case "semantic":
		chartName = "semantic_lineage_chart"
	case "enhanced":
		chartName = "enhanced_erd_chart"
	default:
		chartName = "erd_chart"
	}

	// TODO: Replace with Hasura GraphQL mutation:
	//   mutation { delete_tenant_chart(where: {tenant_datasource_id: {_eq: $datasource_id}, chart_name: {_eq: $chart_name}}) { affected_rows } }
	_, err := db.ExecContext(ctx,
		`DELETE FROM public.tenant_chart 
		 WHERE tenant_datasource_id = $1 AND chart_name = $2`,
		datasourceId, chartName)
	if err != nil {
		return fmt.Errorf("delete chart %s for datasource %s: %w", chartName, datasourceId, err)
	}

	log.Printf("Successfully deleted chart %s for datasource %s", chartName, datasourceId)
	return nil
}

// ValidateChartIntegrity checks the existence and validity of charts
func ValidateChartIntegrity(ctx context.Context, db *sql.DB, datasourceId string) (map[string]bool, error) {
	expectedCharts := []string{"erd_chart", "enhanced_erd_chart", "technical_lineage_chart", "semantic_lineage_chart"}
	results := make(map[string]bool)

	for _, chartName := range expectedCharts {
		var exists bool
		// TODO: Replace with Hasura GraphQL query:
		//   query { tenant_chart_aggregate(where: {tenant_datasource_id: {_eq: $datasource_id}, chart_name: {_eq: $chart_name}}) { aggregate { count } } }
		//   exists = count > 0
		err := db.QueryRowContext(ctx,
			`SELECT EXISTS(SELECT 1 FROM public.tenant_chart 
			 WHERE tenant_datasource_id = $1 AND chart_name = $2)`,
			datasourceId, chartName).Scan(&exists)
		if err != nil {
			log.Printf("Error checking chart %s: %v", chartName, err)
			results[chartName] = false
		} else {
			results[chartName] = exists
		}
	}

	return results, nil
}

// mapDBChartNameToLineageType maps database chart names to lineage types
func mapDBChartNameToLineageType(chartName string) string {
	switch chartName {
	case "erd_chart":
		return "erd"
	case "enhanced_erd_chart":
		return "enhanced"
	case "technical_lineage_chart":
		return "technical"
	case "semantic_lineage_chart":
		return "semantic"
	case "semantic_lineage_raw":
		return "semantic_raw"
	default:
		return chartName
	}
}
