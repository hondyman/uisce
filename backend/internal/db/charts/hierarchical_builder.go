package charts

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
)

// BuildHierarchicalDatabaseStructure creates a hierarchical representation of the database
func BuildHierarchicalDatabaseStructure(ctx context.Context, tx *sql.Tx, datasourceId string) (*DatabaseHierarchy, error) {
	hierarchy := &DatabaseHierarchy{
		Schemas: make(map[string]*SchemaNode),
	}

	// Get all tables with their schemas
	tables, err := QueryTableNodes(ctx, tx, datasourceId)
	if err != nil {
		return nil, fmt.Errorf("query table nodes: %w", err)
	}

	schemaPositions := make(map[string]int)
	schemaIndex := 0

	for _, table := range tables {
		schema := table.Schema
		if schema == "" {
			schema = "default"
		}

		// Create schema node if it doesn't exist
		if _, exists := hierarchy.Schemas[schema]; !exists {
			schemaPositions[schema] = schemaIndex
			hierarchy.Schemas[schema] = &SchemaNode{
				ID:     fmt.Sprintf("schema_%s", schema),
				Name:   schema,
				Tables: make(map[string]*TableNode),
				Position: Position{
					X: float64(schemaIndex * 500),
					Y: 0,
				},
			}
			schemaIndex++
		}

		// Create table node
		tableNode := &TableNode{
			ID:      table.ID,
			Name:    table.Name,
			Schema:  schema,
			Columns: make(map[string]*ColumnNode),
			Position: Position{
				X: float64(50 + (len(hierarchy.Schemas[schema].Tables) * 300)), // Add padding
				Y: 100,                                                         // Position below schema label
			},
		}

		// Get columns for this table
		columns, err := QueryColumnsForTable(ctx, tx, datasourceId, table)
		if err != nil {
			log.Printf("Error getting columns for table %s: %v", table.Name, err)
		} else {
			for i, column := range columns {
				columnNode := &ColumnNode{
					ID:       column.ID,
					Name:     column.Name,
					Table:    table.Name,
					Schema:   schema,
					DataType: column.Type,
					Position: Position{
						X: float64(25 + (i * 150)), // Add padding and stagger
						Y: 50,                      // Position inside the table container
					},
				}
				tableNode.Columns[column.Name] = columnNode
			}
		}

		hierarchy.Schemas[schema].Tables[table.Name] = tableNode
	}

	return hierarchy, nil
}

// ConvertHierarchyToReactFlow converts the hierarchical structure to ReactFlow nodes
func ConvertHierarchyToReactFlow(hierarchy *DatabaseHierarchy, selectedAsset *EnhancedSelectedAsset) *HierarchicalLayout {
	layout := &HierarchicalLayout{
		Nodes:     []EnhancedReactFlowNode{},
		Edges:     []ReactFlowEdge{},
		Viewport:  map[string]interface{}{"x": 0, "y": 0, "zoom": 0.8},
		Metadata:  map[string]interface{}{},
		Hierarchy: make(map[string][]string),
	}

	// Track which schema/table contains the selected asset
	var selectedSchema, selectedTable string
	if selectedAsset != nil {
		parts := strings.Split(selectedAsset.QualifiedPath, ".")
		if len(parts) >= 2 {
			selectedSchema = parts[0]
			selectedTable = parts[1]
		}
	}

	for schemaName, schema := range hierarchy.Schemas {
		// Determine if this schema should be expanded
		isSelectedSchema := selectedSchema == schemaName

		// Create schema container node
		schemaNode := EnhancedReactFlowNode{
			ReactFlowNode: ReactFlowNode{
				ID:   schema.ID,
				Type: "schemaContainer",
				Position: map[string]float64{
					"x": schema.Position.X,
					"y": schema.Position.Y,
				},
				Data: map[string]interface{}{
					"label":         schemaName,
					"nodeType":      "schema",
					"qualifiedPath": schemaName,
					"isContainer":   true,
					"level":         0,
					"expanded":      isSelectedSchema,
					"childCount":    len(schema.Tables),
					"children":      []string{},
				},
			},
		}

		childSchemaIds := []string{}

		// Add tables within this schema
		for tableName, table := range schema.Tables {
			isSelectedTable := selectedTable == tableName

			tableNode := EnhancedReactFlowNode{
				ReactFlowNode: ReactFlowNode{
					ID:   table.ID,
					Type: "tableContainer",
					Position: map[string]float64{
						"x": table.Position.X,
						"y": table.Position.Y,
					},
					Data: map[string]interface{}{
						"label":         tableName,
						"nodeType":      "table",
						"qualifiedPath": fmt.Sprintf("%s.%s", schemaName, tableName),
						"schema":        schemaName,
						"isContainer":   true,
						"level":         1,
						"expanded":      isSelectedTable,
						"childCount":    len(table.Columns),
						"children":      []string{},
					},
				},
				ParentNode: &schema.ID,
				Extent:     stringPtr("parent"),
			}

			childTableIds := []string{}

			// Add columns within this table
			for columnName, column := range table.Columns {
				isSelectedColumn := selectedAsset != nil &&
					selectedAsset.Type == "column" &&
					selectedAsset.Name == columnName &&
					selectedSchema == schemaName &&
					selectedTable == tableName

				columnNode := EnhancedReactFlowNode{
					ReactFlowNode: ReactFlowNode{
						ID:   column.ID,
						Type: "hoverableNode", // Use a generic, styleable node type
						Position: map[string]float64{
							"x": column.Position.X,
							"y": column.Position.Y,
						},
						Data: map[string]interface{}{
							"label":         columnName,
							"nodeType":      "column",
							"qualifiedPath": fmt.Sprintf("%s.%s.%s", schemaName, tableName, columnName),
							"schema":        schemaName,
							"table":         tableName,
							"dataType":      column.DataType,
							"isContainer":   false,
							"level":         2,
							"isCenter":      isSelectedColumn,
						},
					},
					ParentNode: &table.ID,
					Extent:     stringPtr("parent"),
				}

				layout.Nodes = append(layout.Nodes, columnNode)
				childTableIds = append(childTableIds, column.ID)
			}

			// Update table's children
			if children, ok := tableNode.Data["children"].([]string); ok {
				tableNode.Data["children"] = append(children, childTableIds...)
			}
			layout.Hierarchy[table.ID] = childTableIds

			layout.Nodes = append(layout.Nodes, tableNode)
			childSchemaIds = append(childSchemaIds, table.ID)
		}

		// Update schema's children
		if children, ok := schemaNode.Data["children"].([]string); ok {
			schemaNode.Data["children"] = append(children, childSchemaIds...)
		}
		layout.Hierarchy[schema.ID] = childSchemaIds

		layout.Nodes = append(layout.Nodes, schemaNode)
	}

	// Add metadata
	layout.Metadata["totalSchemas"] = len(hierarchy.Schemas)
	layout.Metadata["hierarchicalLayout"] = true
	layout.Metadata["selectedAsset"] = selectedAsset

	return layout
}

// EnhancedSelectedAsset represents a selected asset with hierarchical information
type EnhancedSelectedAsset struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Type          string `json:"type"`
	QualifiedPath string `json:"qualifiedPath"`
	Schema        string `json:"schema,omitempty"`
	Table         string `json:"table,omitempty"`
	Column        string `json:"column,omitempty"`
}

// ParseQualifiedPath extracts hierarchy information from a qualified path
func ParseQualifiedPath(qualifiedPath string) (schema, table, column string) {
	parts := strings.Split(qualifiedPath, ".")
	switch len(parts) {
	case 3:
		return parts[0], parts[1], parts[2]
	case 2:
		return "", parts[0], parts[1]
	case 1:
		return "", "", parts[0]
	default:
		return "", "", ""
	}
}

// BuildHierarchicalTechnicalLineage creates a hierarchical technical lineage chart
func BuildHierarchicalTechnicalLineage(ctx context.Context, tx *sql.Tx, datasourceId string, selectedAsset *EnhancedSelectedAsset) (*HierarchicalLayout, error) {
	// Build the complete database hierarchy
	hierarchy, err := BuildHierarchicalDatabaseStructure(ctx, tx, datasourceId)
	if err != nil {
		return nil, fmt.Errorf("build hierarchical structure: %w", err)
	}

	// Convert to ReactFlow format with proper nesting
	layout := ConvertHierarchyToReactFlow(hierarchy, selectedAsset)

	// Add foreign key edges between columns for a more detailed hierarchical view
	edges, err := QueryColumnForeignKeyEdges(ctx, tx, datasourceId)
	if err != nil {
		log.Printf("Error querying column foreign key edges: %v", err)
	} else {
		layout.Edges = append(layout.Edges, edges...)
	}

	return layout, nil
}

// stringPtr returns a pointer to a string
func stringPtr(s string) *string {
	return &s
}
