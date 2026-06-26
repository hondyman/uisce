package charts

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// QueryTableNodes retrieves table nodes with their schema information
func QueryTableNodes(ctx context.Context, tx *sql.Tx, datasourceId string) ([]DatabaseAsset, error) {
	tablesQuery := `
		SELECT 
			id, 
			core_id,
			node_name, 
			COALESCE(properties->>'schema', '') as table_schema, 
			qualified_path,
			COALESCE(properties, '{}') as properties
		FROM public.catalog_node
		WHERE tenant_datasource_id = $1 
		  AND node_type_id = $2 -- table type
		ORDER BY qualified_path`

	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := tx.QueryContext(queryCtx, tablesQuery, datasourceId, TABLE_NODE_TYPE_ID)
	if err != nil {
		return nil, fmt.Errorf("query catalog tables: %w", err)
	}
	defer rows.Close()

	var tables []DatabaseAsset
	for rows.Next() {
		var asset DatabaseAsset
		var propertiesJSON []byte

		if err := rows.Scan(&asset.ID, &asset.CoreID, &asset.Name, &asset.Schema, &asset.QualifiedPath, &propertiesJSON); err != nil {
			log.Printf("Error scanning catalog table node: %v", err)
			continue
		}

		// Parse schema from qualified path if not provided
		if asset.Schema == "" && strings.HasPrefix(asset.QualifiedPath, "/") {
			pathParts := strings.Split(strings.TrimPrefix(asset.QualifiedPath, "/"), "/")
			if len(pathParts) >= 1 {
				asset.Schema = pathParts[0]
			}
		}

		if asset.Schema == "" {
			asset.Schema = "unknown"
		}

		asset.NodeType = "table"
		asset.Table = asset.Name

		// Parse properties JSON if needed
		// if len(propertiesJSON) > 0 {
		// 	// You can unmarshal properties here if needed
		// }

		tables = append(tables, asset)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating catalog table rows: %w", err)
	}

	return tables, nil
}

// QueryColumnsForTable retrieves columns for a specific table with qualified paths
func QueryColumnsForTable(ctx context.Context, tx *sql.Tx, datasourceId string, tableAsset DatabaseAsset) ([]ColumnData, error) {
	// Updated query to get better hierarchical information
	columnsQuery := `
		SELECT 
			col.id,
			col.node_name,
			COALESCE(col.properties->>'data_type', 'unknown') as data_type,
			COALESCE((col.properties->>'is_nullable')::boolean, false) as is_nullable,
			COALESCE(col.properties->>'default_value', '') as default_value,
			COALESCE(col.core_id IS NOT NULL, false) as is_core,
			COALESCE((col.properties->>'ordinal_position')::integer, 999) as ordinal_position,
			col.qualified_path,
			col.parent_id,
			COALESCE((col.properties->>'is_primary_key')::boolean, false) as is_primary_key,
			COALESCE((col.properties->>'is_foreign_key')::boolean, false) as is_foreign_key,
			COALESCE(col.properties, '{}'::jsonb) as properties
		FROM public.catalog_node col
		WHERE col.parent_id = $2
		  AND col.tenant_datasource_id = $1 
		  AND col.node_type_id = $3
		ORDER BY ordinal_position`

	timeout := 10 * time.Second
	queryCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	rows, err := tx.QueryContext(queryCtx, columnsQuery, datasourceId, tableAsset.ID, COLUMN_NODE_TYPE_ID)
	if err != nil {
		return nil, fmt.Errorf("query columns for table %s: %w", tableAsset.ID, err)
	}
	defer rows.Close()

	var columns []ColumnData
	for rows.Next() {
		var columnID, columnName, dataType, defaultValue, qualifiedPath, parentID string
		var isNullable, isCore, isPrimaryKey, isForeignKey bool
		var ordinalPosition int
		var propertiesJSON []byte

		if err := rows.Scan(&columnID, &columnName, &dataType, &isNullable, &defaultValue, &isCore, &ordinalPosition, &qualifiedPath, &parentID, &isPrimaryKey, &isForeignKey, &propertiesJSON); err != nil {
			log.Printf("Error scanning column from catalog: %v", err)
			continue
		}

		// Build the qualified path for the column: schema.table.column
		var columnQualifiedPath string
		if tableAsset.Schema != "" && tableAsset.Schema != "unknown" {
			columnQualifiedPath = fmt.Sprintf("%s.%s.%s", tableAsset.Schema, tableAsset.Table, columnName)
		} else {
			columnQualifiedPath = fmt.Sprintf("%s.%s", tableAsset.Table, columnName)
		}

		column := ColumnData{
			ID:            columnID,
			Name:          columnName,
			Type:          dataType,
			IsCore:        isCore,
			Nullable:      isNullable,
			Schema:        tableAsset.Schema,
			Table:         tableAsset.Table,
			QualifiedPath: columnQualifiedPath,
			IsPrimaryKey:  isPrimaryKey,
			IsForeignKey:  isForeignKey,
		}

		if len(propertiesJSON) > 0 {
			var props map[string]interface{}
			if err := json.Unmarshal(propertiesJSON, &props); err == nil {
				column.Properties = props
			}
		}

		if defaultValue != "" {
			column.Default = defaultValue
		}

		columns = append(columns, column)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over column rows for table %s: %w", tableAsset.ID, err)
	}

	log.Printf("Found %d columns for table %s.%s", len(columns), tableAsset.Schema, tableAsset.Table)
	return columns, nil
}

// QueryForeignKeyEdges retrieves foreign key relationships between tables
func QueryForeignKeyEdges(ctx context.Context, tx *sql.Tx, datasourceId string) ([]ReactFlowEdge, error) {
	fkQuery := `
		SELECT 
			ce.id, 
			ce.source_node_id, 
			ce.target_node_id,
			source_node.node_name as source_table,
			target_node.node_name as target_table
		FROM public.catalog_edge ce
		JOIN public.catalog_node source_node ON ce.source_node_id = source_node.id
		JOIN public.catalog_node target_node ON ce.target_node_id = target_node.id
		WHERE ce.tenant_datasource_id = $1 
		  AND ce.properties->>'primary_constraint_name' IS NOT NULL
		  AND source_node.node_type_id = $2
		  AND target_node.node_type_id = $2
		ORDER BY ce.id`

	fkCtx, fkCancel := context.WithTimeout(ctx, 30*time.Second)
	defer fkCancel()

	rows, err := tx.QueryContext(fkCtx, fkQuery, datasourceId, TABLE_NODE_TYPE_ID)
	if err != nil {
		return nil, fmt.Errorf("query catalog edges: %w", err)
	}
	defer rows.Close()

	var edges []ReactFlowEdge
	for rows.Next() {
		var edgeID, sourceNodeID, targetNodeID, sourceTable, targetTable string

		if err := rows.Scan(&edgeID, &sourceNodeID, &targetNodeID, &sourceTable, &targetTable); err != nil {
			log.Printf("Error scanning catalog edge: %v", err)
			continue
		}

		// Parse properties and build edge label
		label := buildForeignKeyLabel(sourceTable, targetTable)

		edge := ReactFlowEdge{
			ID:     edgeID,
			Source: sourceNodeID,
			Target: targetNodeID,
			Type:   "smoothstep",
			Label:  label,
			Data: map[string]interface{}{
				"relationshipType": "foreign_key",
				"sourceTable":      sourceTable,
				"targetTable":      targetTable,
			},
		}

		edges = append(edges, edge)
		log.Printf("Added FK edge: %s -> %s (%s)", sourceTable, targetTable, label)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating catalog edge rows: %w", err)
	}

	return edges, nil
}

// QueryColumnForeignKeyEdges retrieves foreign key relationships between columns
func QueryColumnForeignKeyEdges(ctx context.Context, tx *sql.Tx, datasourceId string) ([]ReactFlowEdge, error) {
	fkQuery := `
		SELECT
			ce.id,
			ce.source_node_id,
			ce.target_node_id,
			source_node.node_name as source_column,
			target_node.node_name as target_column,
			source_parent.node_name as source_table,
			target_parent.node_name as target_table
		FROM public.catalog_edge ce
		JOIN public.catalog_node source_node ON ce.source_node_id = source_node.id
		JOIN public.catalog_node target_node ON ce.target_node_id = target_node.id
		JOIN public.catalog_node source_parent ON source_node.parent_id = source_parent.id
		JOIN public.catalog_node target_parent ON target_node.parent_id = target_parent.id
		WHERE ce.tenant_datasource_id = $1
		  AND ce.properties->>'primary_constraint_name' IS NOT NULL
		  AND source_node.node_type_id = $2 -- column type
		  AND target_node.node_type_id = $2 -- column type
		ORDER BY ce.id`

	fkCtx, fkCancel := context.WithTimeout(ctx, 30*time.Second)
	defer fkCancel()

	rows, err := tx.QueryContext(fkCtx, fkQuery, datasourceId, COLUMN_NODE_TYPE_ID)
	if err != nil {
		return nil, fmt.Errorf("query catalog column edges: %w", err)
	}
	defer rows.Close()

	var edges []ReactFlowEdge
	for rows.Next() {
		var edgeID, sourceNodeID, targetNodeID, sourceColumn, targetColumn, sourceTable, targetTable string

		if err := rows.Scan(&edgeID, &sourceNodeID, &targetNodeID, &sourceColumn, &targetColumn, &sourceTable, &targetTable); err != nil {
			log.Printf("Error scanning catalog column edge: %v", err)
			continue
		}

		label := fmt.Sprintf("%s.%s -> %s.%s", sourceTable, sourceColumn, targetTable, targetColumn)

		edge := ReactFlowEdge{
			ID:     edgeID,
			Source: sourceNodeID,
			Target: targetNodeID,
			Type:   "smoothstep",
			Label:  label,
			Data: map[string]interface{}{
				"relationshipType": "foreign_key",
				"sourceTable":      sourceTable,
				"targetTable":      targetTable,
				"sourceColumn":     sourceColumn,
				"targetColumn":     targetColumn,
			},
		}

		edges = append(edges, edge)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error after iterating catalog column edge rows: %w", err)
	}

	return edges, nil
}

// buildForeignKeyLabel creates a descriptive label for foreign key edges
func buildForeignKeyLabel(sourceTable, targetTable string) string {
	// You can parse the JSON properties here to extract column mapping
	// For now, return a simple label
	return fmt.Sprintf("%s -> %s", sourceTable, targetTable)
}

// QuerySemanticNodes retrieves semantic nodes of a specific type
func QuerySemanticNodes(ctx context.Context, tx *sql.Tx, datasourceId string, nodeType string) ([]SemanticNode, error) {
	query := `
		SELECT 
			n.id, 
			n.node_name, 
			t.catalog_type_name as node_type, 
			n.description, 
			n.qualified_path, 
			n.properties
		FROM public.catalog_node n
		JOIN public.catalog_node_type t ON n.node_type_id = t.id
		WHERE n.tenant_datasource_id = $1 AND t.catalog_type_name = $2`

	rows, err := tx.QueryContext(ctx, query, datasourceId, nodeType)
	if err != nil {
		return nil, fmt.Errorf("query semantic nodes: %w", err)
	}
	defer rows.Close()

	var nodes []SemanticNode
	for rows.Next() {
		var node SemanticNode
		var propertiesJSON []byte
		var qualifiedPath sql.NullString
		var description sql.NullString

		err := rows.Scan(&node.ID, &node.NodeName, &node.NodeType, &description, &qualifiedPath, &propertiesJSON)
		if err != nil {
			log.Printf("Error scanning semantic node: %v", err)
			continue
		}

		if description.Valid {
			node.Description = description.String
		}

		if qualifiedPath.Valid {
			node.QualifiedPath = qualifiedPath.String
		}

		if len(propertiesJSON) > 0 {
			// Unmarshal properties if needed
		} else {
			node.Properties = make(map[string]interface{})
		}

		nodes = append(nodes, node)
	}

	log.Printf("querySemanticNodes for type '%s' found %d nodes.", nodeType, len(nodes))
	return nodes, rows.Err()
}

// HealthCheck performs a basic database connection test
func HealthCheck(ctx context.Context, tx *sql.Tx, datasourceId string) error {
	var testCount int
	err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM public.catalog_node WHERE tenant_datasource_id = $1", datasourceId).Scan(&testCount)
	if err != nil {
		return fmt.Errorf("database connection test failed: %w", err)
	}
	log.Printf("Database connection test: found %d catalog nodes for datasource %s", testCount, datasourceId)
	return nil
}
