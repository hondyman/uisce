package charts

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"
)

// ModelInfo holds basic information about a semantic model for chart enrichment.
type ModelInfo struct {
	ID          string     `db:"id" json:"id"`
	Title       string     `db:"title" json:"title"`
	Status      string     `db:"status" json:"status"`
	Version     int        `db:"version" json:"version"`
	PublishedAt *time.Time `db:"published_at" json:"publishedAt,omitempty"`
}

// queryModelMetadataForDatasource fetches existing model metadata for a given datasource.
func queryModelMetadataForDatasource(ctx context.Context, tx *sql.Tx, datasourceId string) (map[string]ModelInfo, error) {
	query := `
		SELECT id, model_key, title, status, version, published_at
		FROM public.fabric_defn
		WHERE tenant_datasource_id = $1 AND is_current = true
	`
	rows, err := tx.QueryContext(ctx, query, datasourceId)
	if err != nil {
		if strings.Contains(err.Error(), `relation "public.fabric_defn" does not exist`) {
			log.Println("INFO: public.fabric_defn table not found, skipping model metadata enrichment.")
			return make(map[string]ModelInfo), nil
		}
		return nil, fmt.Errorf("query fabric definitions: %w", err)
	}
	defer rows.Close()

	modelMap := make(map[string]ModelInfo)
	for rows.Next() {
		var modelKey string
		var info ModelInfo
		if err := rows.Scan(&info.ID, &modelKey, &info.Title, &info.Status, &info.Version, &info.PublishedAt); err != nil {
			log.Printf("Error scanning fabric definition: %v", err)
			continue
		}
		modelMap[modelKey] = info
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating fabric definitions: %w", err)
	}

	log.Printf("Found %d existing models for datasource %s", len(modelMap), datasourceId)
	return modelMap, nil
}

// BuildTechnicalLineageChart constructs a technical lineage chart with proper qualified paths
func BuildTechnicalLineageChart(ctx context.Context, db *sql.DB, datasourceId string, isGoldCopy bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	log.Printf("Building technical lineage chart with fresh data for datasource %s", datasourceId)

	// Health check
	if err := HealthCheck(ctx, tx, datasourceId); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	chart := &TechnicalLineageChart{
		Nodes: []ReactFlowNode{},
		Edges: []ReactFlowEdge{},
		Metadata: map[string]interface{}{
			"databaseEdgeCount": 0,
			"generatedAt":       time.Now().Format(time.RFC3339),
			"chartType":         "technical_lineage",
		},
	}

	if err := populateTechnicalData(ctx, tx, datasourceId, chart); err != nil {
		return fmt.Errorf("populate technical lineage data: %w", err)
	}

	return saveChart(ctx, tx, datasourceId, "technical_lineage_chart", chart)
}

// populateTechnicalData populates a technical chart with database schema data
func populateTechnicalData(ctx context.Context, tx *sql.Tx, datasourceId string, chart *TechnicalLineageChart) error {
	// Get all tables
	tables, err := QueryTableNodes(ctx, tx, datasourceId)
	if err != nil {
		return fmt.Errorf("query table nodes: %w", err)
	}

	log.Printf("Found %d tables for datasource %s", len(tables), datasourceId)

	// Fetch existing model metadata for this datasource
	modelMeta, err := queryModelMetadataForDatasource(ctx, tx, datasourceId)
	if err != nil {
		// Log as a warning but don't fail the entire chart generation
		log.Printf("Warning: could not fetch model metadata: %v", err)
	}

	// Process each table and its columns
	nodeIndex := 0
	for _, table := range tables {
		// Get columns for this table with proper qualified paths
		columns, err := QueryColumnsForTable(ctx, tx, datasourceId, table)
		if err != nil {
			return fmt.Errorf("query columns for table %s: %w", table.Name, err)
		}

		// Create enhanced column data with qualified paths
		enhancedColumns := make([]map[string]interface{}, len(columns))
		for i, col := range columns {
			enhancedColumns[i] = map[string]interface{}{
				"name":          col.Name,
				"type":          col.Type,
				"isCore":        col.IsCore,
				"nullable":      col.Nullable,
				"default":       col.Default,
				"schema":        col.Schema,
				"table":         col.Table,
				"qualifiedPath": col.QualifiedPath, // This is what your frontend needs!
				"isPrimaryKey":  col.IsPrimaryKey,
				"isForeignKey":  col.IsForeignKey,
				"properties":    col.Properties,
			}
		}

		// Build qualified table path
		var tableQualifiedPath string
		if table.Schema != "" && table.Schema != "unknown" {
			tableQualifiedPath = fmt.Sprintf("%s.%s", table.Schema, table.Name)
		} else {
			tableQualifiedPath = table.Name
		}

		// Check if a model exists for this table
		var modelInfo map[string]interface{}
		if info, ok := modelMeta[table.QualifiedPath]; ok {
			modelInfo = map[string]interface{}{
				"exists": true, "title": info.Title, "status": info.Status, "version": info.Version, "publishedAt": info.PublishedAt,
			}
		}

		// Create the table node with enhanced data
		node := ReactFlowNode{
			ID:   table.ID,
			Type: "table",
			Position: map[string]float64{
				"x": float64((nodeIndex % 5) * 250),
				"y": float64((nodeIndex / 5) * 150),
			},
			Data: map[string]interface{}{
				"label":      table.Name,
				"tableName":  tableQualifiedPath,
				"schemaName": table.Schema,
				"schema":     table.Schema,
				"nodeType":   "table",
				"nodeId":     table.ID,
				"isCore":     table.CoreID.Valid,
				// Include the core_id when available so the frontend can detect gold/core mappings
				"core_id": func() interface{} {
					if table.CoreID.Valid {
						return table.CoreID.UUID.String()
					}
					return nil
				}(),
				"columns":       enhancedColumns, // Enhanced columns with qualified paths
				"qualifiedPath": tableQualifiedPath,
				// Additional metadata for hover tooltips
				"description": fmt.Sprintf("Table: %s", tableQualifiedPath),
				"columnCount": len(columns),
				"modelInfo":   modelInfo, // Add model info here
			},
		}

		chart.Nodes = append(chart.Nodes, node)
		log.Printf("Created enhanced node for table %s with %d columns (qualified path: %s)",
			table.Name, len(columns), tableQualifiedPath)

		// Log sample column data for debugging
		if len(columns) > 0 {
			log.Printf("Sample column qualified path: %s", columns[0].QualifiedPath)
		}

		nodeIndex++
	}

	// Get foreign key edges
	edges, err := QueryForeignKeyEdges(ctx, tx, datasourceId)
	if err != nil {
		return fmt.Errorf("query foreign key edges: %w", err)
	}
	chart.Edges = append(chart.Edges, edges...)

	// Update metadata
	chart.Metadata["databaseEdgeCount"] = len(chart.Edges)
	chart.Metadata["totalNodes"] = len(chart.Nodes)
	chart.Metadata["datasourceId"] = datasourceId
	chart.Metadata["generatedAt"] = time.Now().Format(time.RFC3339)

	log.Printf("Completed technical data population: %d nodes, %d edges", len(chart.Nodes), len(chart.Edges))
	return nil
}

// BuildERDChart constructs a basic ERD chart from the database schema
func BuildERDChart(ctx context.Context, db *sql.DB, datasourceId string, isGoldCopy bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := HealthCheck(ctx, tx, datasourceId); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	chart := &TechnicalLineageChart{
		Nodes: []ReactFlowNode{},
		Edges: []ReactFlowEdge{},
		Metadata: map[string]interface{}{
			"databaseEdgeCount": 0,
			"generatedAt":       time.Now().Format(time.RFC3339),
			"chartType":         "erd",
		},
	}

	if err := populateTechnicalData(ctx, tx, datasourceId, chart); err != nil {
		return fmt.Errorf("populate technical data: %w", err)
	}

	return saveChart(ctx, tx, datasourceId, "erd_chart", chart)
}

// BuildEnhancedERDChart constructs an enhanced ERD chart
func BuildEnhancedERDChart(ctx context.Context, db *sql.DB, datasourceId string, isGoldCopy bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Delete existing enhanced chart
	_, err = tx.ExecContext(ctx, `
		DELETE FROM public.tenant_chart 
		WHERE tenant_datasource_id = $1 AND chart_name = 'enhanced_erd_chart'`,
		datasourceId)
	if err != nil {
		return fmt.Errorf("delete existing enhanced chart: %w", err)
	}

	// Create new enhanced chart from basic ERD chart
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.tenant_chart (tenant_datasource_id, chart_name, chart, updated_at)
		SELECT tenant_datasource_id, 'enhanced_erd_chart', chart, NOW()
		FROM public.tenant_chart 
		WHERE tenant_datasource_id = $1 AND chart_name = 'erd_chart'`,
		datasourceId)
	if err != nil {
		return fmt.Errorf("save enhanced ERD chart: %w", err)
	}

	log.Printf("Successfully built enhanced ERD chart for datasource %s", datasourceId)
	return tx.Commit()
}
