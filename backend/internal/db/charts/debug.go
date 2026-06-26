package charts

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

// DebugChartData logs detailed chart information
func DebugChartData(ctx context.Context, db *sql.DB, datasourceId string) error {
	log.Printf("--- Starting Chart Debug for Datasource: %s ---", datasourceId)

	charts, err := ListChartsForDatasource(ctx, db, datasourceId)
	if err != nil {
		log.Printf("DEBUG: Error listing charts: %v", err)
		return fmt.Errorf("failed to list charts for debugging: %w", err)
	}

	log.Printf("Found %d charts for datasource %s", len(charts), datasourceId)

	for _, chartInfo := range charts {
		chartName := chartInfo.Name
		lineageType := mapDBChartNameToLineageType(chartName)
		if lineageType == "" {
			log.Printf("DEBUG: Unknown chart name found, cannot determine lineage type: %s", chartName)
			continue
		}

		log.Printf("--- Debugging Chart: %s (type: %s) ---", chartName, lineageType)

		var compressedData []byte
		err := db.QueryRowContext(ctx,
			`SELECT chart FROM public.tenant_chart 
			 WHERE tenant_datasource_id = $1 AND chart_name = $2`,
			datasourceId, chartName).Scan(&compressedData)

		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("DEBUG: No chart data found for %s", chartName)
			} else {
				log.Printf("DEBUG: Error getting chart data for %s: %v", chartName, err)
			}
			continue
		}

		log.Printf("Chart %s: Compressed size: %d bytes", chartName, len(compressedData))

		data, err := ParseChartData(compressedData, lineageType)
		if err != nil {
			log.Printf("DEBUG: Error parsing chart data for %s: %v", chartName, err)
			decompressed, derr := decompressData(compressedData)
			if derr == nil {
				preview := string(decompressed)
				if len(preview) > 200 {
					preview = preview[:200] + "..."
				}
				log.Printf("Decompressed data (string preview): %s", preview)
			}
			continue
		}

		switch d := data.(type) {
		case TechnicalLineageChart:
			log.Printf("Type: TechnicalLineageChart, Nodes: %d, Edges: %d", len(d.Nodes), len(d.Edges))
			log.Printf("Metadata: %+v", d.Metadata)

			// Debug column data in nodes
			for i, node := range d.Nodes {
				if i >= 3 { // Only show first 3 nodes to avoid spam
					break
				}
				if columns, ok := node.Data["columns"].([]map[string]interface{}); ok {
					log.Printf("Node %s has %d columns", node.ID[:8], len(columns))
					if len(columns) > 0 {
						log.Printf("Sample column: %+v", columns[0])
					}
				}
			}

		case SemanticLineageChart:
			log.Printf("Type: SemanticLineageChart, BusinessTerms: %d, SemanticTerms: %d, SemanticColumns: %d, DatabaseColumns: %d, Edges: %d",
				len(d.BusinessTerms), len(d.SemanticTerms), len(d.SemanticColumns), len(d.DatabaseColumns), len(d.Edges))
			log.Printf("Metadata: %+v", d.Metadata)

			// Debug database columns for qualified paths
			for i, col := range d.DatabaseColumns {
				if i >= 3 { // Only show first 3 columns to avoid spam
					break
				}
				log.Printf("Database Column: Name=%s, QualifiedPath=%s, Properties=%+v",
					col.NodeName, col.QualifiedPath, col.Properties)
			}

		default:
			log.Printf("Type: Unknown (%T)", d)
		}
	}

	log.Printf("--- Finished Chart Debug for Datasource: %s ---", datasourceId)
	return nil
}
