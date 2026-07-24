package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx/types"
)

// LineageAsset represents a single asset in the lineage graph.
type LineageAsset struct {
	ID            string
	Name          string
	QualifiedPath string
}

// LineageEdge represents a relationship between two assets.
type LineageEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// LineageData contains all assets and edges for a lineage view.
type LineageData struct {
	Assets []LineageAsset
	Edges  []LineageEdge
}

const (
	schemaContainerNodeType = "schemaContainer"
	tableContainerNodeType  = "tableContainer"
	columnNodeType          = "column"
)

// BuildHierarchicalDatabaseStructure creates the complete database hierarchy from a list of assets.
func BuildHierarchicalDatabaseStructure(assets []LineageAsset) *DatabaseHierarchy {
	hierarchy := &DatabaseHierarchy{Databases: []Database{}}
	dbMap := make(map[string]*Database)

	for _, asset := range assets {
		parts := strings.Split(asset.QualifiedPath, ".")
		if len(parts) < 3 {
			continue // Expecting at least db.schema.table
		}

		dbName, schemaName, tableName := parts[0], parts[1], parts[2]
		columnName := ""
		if len(parts) > 3 {
			columnName = parts[3]
		}

		db, ok := dbMap[dbName]
		if !ok {
			newDb := Database{Name: dbName, Schemas: []Schema{}}
			hierarchy.Databases = append(hierarchy.Databases, newDb)
			db = &hierarchy.Databases[len(hierarchy.Databases)-1]
			dbMap[dbName] = db
		}

		schemaMap := make(map[string]*Schema)
		var currentSchema *Schema
		for i := range db.Schemas {
			if db.Schemas[i].Name == schemaName {
				currentSchema = &db.Schemas[i]
				break
			}
		}

		if currentSchema == nil {
			newSchema := Schema{Name: schemaName, Tables: []Table{}}
			db.Schemas = append(db.Schemas, newSchema)
			currentSchema = &db.Schemas[len(db.Schemas)-1]
			schemaMap[schemaName] = currentSchema
		}

		tableMap := make(map[string]*Table)
		var currentTable *Table
		for i := range currentSchema.Tables {
			if currentSchema.Tables[i].Name == tableName {
				currentTable = &currentSchema.Tables[i]
				break
			}
		}

		if currentTable == nil {
			newTable := Table{Name: tableName, Columns: []Column{}}
			currentSchema.Tables = append(currentSchema.Tables, newTable)
			currentTable = &currentSchema.Tables[len(currentSchema.Tables)-1]
			tableMap[tableName] = currentTable
		}

		if columnName != "" {
			currentTable.Columns = append(currentTable.Columns, Column{Name: columnName, Type: "unknown"})
		}
	}

	return hierarchy
}

// ConvertHierarchyToReactFlow converts the database hierarchy to a ReactFlow format.
func ConvertHierarchyToReactFlow(hierarchy *DatabaseHierarchy, selectedAssetID string) []ReactFlowNode {
	nodes := []ReactFlowNode{}
	xOffset := 0.0

	for _, db := range hierarchy.Databases {
		for _, schema := range db.Schemas {
			schemaID := fmt.Sprintf("schema_%s", schema.Name)
			schemaData, _ := json.Marshal(map[string]interface{}{"label": schema.Name})
			nodes = append(nodes, ReactFlowNode{
				ID:       schemaID,
				Type:     schemaContainerNodeType,
				Data:     types.JSONText(schemaData),
				Position: NodePosition{X: xOffset, Y: 0},
			})

			yOffset := 100.0
			for _, table := range schema.Tables {
				tableID := fmt.Sprintf("table_%s_%s", schema.Name, table.Name)
				tableData, _ := json.Marshal(map[string]interface{}{"label": table.Name})
				nodes = append(nodes, ReactFlowNode{
					ID:       tableID,
					Type:     tableContainerNodeType,
					Data:     types.JSONText(tableData),
					Position: NodePosition{X: 50, Y: yOffset},
					ParentID: &schemaID,
					Extent:   "parent",
				})

				colXOffset := 50.0
				for _, column := range table.Columns {
					columnID := fmt.Sprintf("column_%s_%s_%s", schema.Name, table.Name, column.Name)
					isPrimary := columnID == selectedAssetID
					columnData, _ := json.Marshal(map[string]interface{}{
						"label":     column.Name,
						"isPrimary": isPrimary,
					})
					nodes = append(nodes, ReactFlowNode{
						ID:       columnID,
						Type:     columnNodeType,
						Data:     types.JSONText(columnData),
						Position: NodePosition{X: colXOffset, Y: 50},
						ParentID: &tableID,
						Extent:   "parent",
					})
					colXOffset += 150
				}
				yOffset += 200
			}
			xOffset += 400
		}
	}
	return nodes
}
