package charts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// SemanticNode represents a node in the semantic lineage graph.

// BuildSemanticLineageChart constructs a semantic lineage chart and converts it to ReactFlow format
func BuildSemanticLineageChart(ctx context.Context, db *sql.DB, datasourceId string, isGoldCopy bool) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	log.Printf("Building semantic lineage chart for datasource %s", datasourceId)

	var semanticChart *SemanticLineageChart
	if chart, err := querySemanticDataFromTables(ctx, tx, datasourceId); err == nil && chart != nil {
		semanticChart = chart
		log.Printf("Using semantic data from tables")
	} else {
		return fmt.Errorf("failed to get semantic data from tables: %w", err)
	}

	reactFlowChart := convertSemanticToReactFlow(semanticChart)

	// Save both formats
	semanticJSON, err := json.Marshal(semanticChart)
	if err != nil {
		return fmt.Errorf("marshal original semantic chart: %w", err)
	}

	chartJSON, err := json.Marshal(reactFlowChart)
	if err != nil {
		return fmt.Errorf("marshal semantic lineage chart: %w", err)
	}

	log.Printf("=== SEMANTIC LINEAGE CHART DATA ===")
	log.Printf("Original semantic JSON size: %d bytes", len(semanticJSON))
	log.Printf("ReactFlow format JSON size: %d bytes", len(chartJSON))
	log.Printf("ReactFlow structure: %d nodes, %d edges", len(reactFlowChart.Nodes), len(reactFlowChart.Edges))

	compressedChart, err := compressData(chartJSON)
	if err != nil {
		return fmt.Errorf("compress semantic lineage chart: %w", err)
	}

	compressedSemanticChart, err := compressData(semanticJSON)
	if err != nil {
		return fmt.Errorf("compress original semantic chart: %w", err)
	}

	// Delete existing semantic lineage chart
	_, err = tx.ExecContext(ctx, `
		DELETE FROM public.tenant_chart 
		WHERE tenant_datasource_id = $1 AND chart_name = $2`,
		datasourceId, "semantic_lineage_chart")
	if err != nil {
		return fmt.Errorf("delete existing semantic lineage chart: %w", err)
	}

	// Save ReactFlow format
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.tenant_chart (tenant_datasource_id, chart_name, chart, updated_at)
		VALUES ($1, $2, $3, NOW())`,
		datasourceId, "semantic_lineage_chart", compressedChart)
	if err != nil {
		return fmt.Errorf("save semantic lineage chart: %w", err)
	}

	// Delete existing raw semantic chart
	_, err = tx.ExecContext(ctx, `
		DELETE FROM public.tenant_chart 
		WHERE tenant_datasource_id = $1 AND chart_name = $2`,
		datasourceId, "semantic_lineage_raw")
	if err != nil {
		return fmt.Errorf("delete existing raw semantic lineage chart: %w", err)
	}

	// Save raw format
	_, err = tx.ExecContext(ctx, `
		INSERT INTO public.tenant_chart (tenant_datasource_id, chart_name, chart, updated_at)
		VALUES ($1, $2, $3, NOW())`,
		datasourceId, "semantic_lineage_raw", compressedSemanticChart)
	if err != nil {
		return fmt.Errorf("save raw semantic lineage chart: %w", err)
	}

	log.Printf("Successfully built semantic lineage chart with %d total nodes and %d edges", len(reactFlowChart.Nodes), len(reactFlowChart.Edges))
	return tx.Commit()
}

// querySemanticDataFromTables retrieves semantic data from database tables
func querySemanticDataFromTables(ctx context.Context, tx *sql.Tx, datasourceId string) (*SemanticLineageChart, error) {
	chart := &SemanticLineageChart{
		BusinessTerms:   []SemanticNode{},
		SemanticTerms:   []SemanticNode{},
		SemanticColumns: []SemanticNode{},
		DatabaseColumns: []SemanticNode{},
		Edges:           []SemanticEdge{},
		Viewport:        map[string]interface{}{"x": 0, "y": 0, "zoom": 1},
		Metadata:        map[string]interface{}{},
	}

	// Query business terms
	businessTerms, err := QuerySemanticNodes(ctx, tx, datasourceId, "business_term")
	if err != nil {
		return nil, fmt.Errorf("failed to query business terms: %w", err)
	}
	chart.BusinessTerms = businessTerms

	// Query semantic terms
	semanticTerms, err := QuerySemanticNodes(ctx, tx, datasourceId, "semantic_term")
	if err != nil {
		return nil, fmt.Errorf("failed to query semantic terms: %w", err)
	}
	chart.SemanticTerms = semanticTerms

	// Query semantic columns
	semanticColumns, err := QuerySemanticNodes(ctx, tx, datasourceId, "semantic_column")
	if err != nil {
		return nil, fmt.Errorf("failed to query semantic columns: %w", err)
	}
	chart.SemanticColumns = semanticColumns

	// Query database columns with enhanced qualified paths
	databaseColumns, err := queryDatabaseColumnNodes(ctx, tx, datasourceId)
	if err != nil {
		return nil, fmt.Errorf("failed to query database columns: %w", err)
	}
	chart.DatabaseColumns = databaseColumns

	// Query semantic edges
	edges, err := querySemanticEdges(ctx, tx, datasourceId)
	if err != nil {
		return nil, fmt.Errorf("failed to query semantic edges: %w", err)
	}
	chart.Edges = edges

	chart.Metadata["semanticEdgeCount"] = len(edges)
	totalNodes := len(businessTerms) + len(semanticTerms) + len(semanticColumns) + len(databaseColumns)
	chart.Metadata["totalNodes"] = totalNodes

	log.Printf("Generated semantic lineage: %d nodes, %d edges", totalNodes, len(edges))
	return chart, nil
}

// queryDatabaseColumnNodes retrieves database column nodes with proper qualified paths
func queryDatabaseColumnNodes(ctx context.Context, tx *sql.Tx, datasourceId string) ([]SemanticNode, error) {
	// Query catalog_node directly as catalog_node_vw might not be reliable
	query := `
		SELECT 
			id, 
			node_name, 
			'database_column' as node_type, 
			description, 
			qualified_path,
			properties
		FROM public.catalog_node
		WHERE tenant_datasource_id = $1 
		  AND node_type_id = $2`

	rows, err := tx.QueryContext(ctx, query, datasourceId, COLUMN_NODE_TYPE_ID)
	if err != nil {
		return nil, fmt.Errorf("query database column nodes: %w", err)
	}
	defer rows.Close()

	var nodes []SemanticNode
	for rows.Next() {
		var node SemanticNode
		var propertiesJSON []byte
		var qualifiedPath sql.NullString
		var description sql.NullString

		err := rows.Scan(&node.ID, &node.NodeName, &node.NodeType, &description,
			&qualifiedPath, &propertiesJSON)
		if err != nil {
			log.Printf("Error scanning database column node: %v", err)
			continue
		}

		if description.Valid {
			node.Description = description.String
		}

		if qualifiedPath.Valid {
			node.QualifiedPath = qualifiedPath.String
		} else {
			node.QualifiedPath = node.NodeName
		}

		// Parse properties
		if len(propertiesJSON) > 0 {
			if err := json.Unmarshal(propertiesJSON, &node.Properties); err != nil {
				node.Properties = make(map[string]interface{})
			}
		} else {
			node.Properties = make(map[string]interface{})
		}

		// Initialize if null
		if node.Properties == nil {
			node.Properties = make(map[string]interface{})
		}

		// Parse schema/table/column from qualified path
		// Expect format: schema.table.column
		parts := strings.Split(node.QualifiedPath, ".")
		var schema, table, column string
		if len(parts) >= 3 {
			schema = parts[0]
			table = parts[1]
			column = parts[2]
		} else if len(parts) == 2 {
			schema = "public" // default
			table = parts[0]
			column = parts[1]
		} else {
			column = node.NodeName
		}

		// Add enhanced metadata for frontend
		node.Properties["schema"] = schema
		node.Properties["table"] = table
		node.Properties["column"] = column // or node.NodeName

		// If we missed schema/table parsing, try to extract from properties if they exist there
		// (SemanticMappingService puts DataType there, but maybe not schema/table)

		nodes = append(nodes, node)
	}

	log.Printf("queryDatabaseColumnNodes found %d nodes", len(nodes))
	return nodes, rows.Err()
}

// querySemanticEdges retrieves semantic edges
func querySemanticEdges(ctx context.Context, tx *sql.Tx, datasourceId string) ([]SemanticEdge, error) {
	// Query catalog_edge with join to get edge type name
	// Also query by tenant_id to catch edges that may be stored under different datasource IDs
	// (e.g., context datasource vs actual datasource)
	query := `
		SELECT ce.id, ce.source_node_id, ce.target_node_id, 
			   COALESCE(ce.edge_type, 'unknown') as edge_type,
			   ce.properties
		FROM public.catalog_edge ce
		WHERE ce.tenant_datasource_id = $1
		   OR ce.tenant_id IN (
		       SELECT DISTINCT tenant_id FROM public.catalog_node WHERE tenant_datasource_id = $1
		   )`

	rows, err := tx.QueryContext(ctx, query, datasourceId)
	if err != nil {
		return nil, fmt.Errorf("query semantic edges: %w", err)
	}
	defer rows.Close()

	var edges []SemanticEdge
	for rows.Next() {
		var edge SemanticEdge
		var propertiesJSON []byte

		err := rows.Scan(&edge.ID, &edge.SourceID, &edge.TargetID, &edge.EdgeType, &propertiesJSON)
		if err != nil {
			log.Printf("Error scanning semantic edge: %v", err)
			continue
		}

		// Map EdgeType (text) to RelationshipType (text)
		edge.RelationshipType = strings.ToLower(edge.EdgeType)

		// Parse properties from JSON
		edge.Properties = make(map[string]interface{})
		if len(propertiesJSON) > 0 {
			if err := json.Unmarshal(propertiesJSON, &edge.Properties); err != nil {
				log.Printf("Error parsing edge properties: %v", err)
			}
		}

		edges = append(edges, edge)
	}

	log.Printf("querySemanticEdges found %d edges for datasource %s", len(edges), datasourceId)
	return edges, rows.Err()
}
