package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jmoiron/sqlx"
)

// CubeSyncService generates Cube.js schema files from the catalog.
type CubeSyncService struct {
	db         *sqlx.DB
	schemaPath string
}

// NewCubeSyncService creates a new CubeSyncService.
func NewCubeSyncService(db *sqlx.DB, schemaPath string) *CubeSyncService {
	return &CubeSyncService{
		db:         db,
		schemaPath: schemaPath,
	}
}

// CatalogNode represents a node in the catalog.
type CatalogNode struct {
	ID            string          `db:"id"`
	NodeName      string          `db:"node_name"`
	QualifiedPath string          `db:"qualified_path"`
	NodeType      string          `db:"node_type"` // Joined from catalog_node_type
	Description   *string         `db:"description"`
	Properties    json.RawMessage `db:"properties"`
}

// CatalogEdge represents a relationship between nodes.
type CatalogEdge struct {
	SourceID         string `db:"source_node_id"`
	TargetID         string `db:"target_node_id"`
	RelationshipType string `db:"relationship_type"`
	TargetName       string `db:"target_name"` // Joined
}

// SyncSchema generates Cube schema files for all tables/views in the catalog.
func (s *CubeSyncService) SyncSchema(ctx context.Context, tenantID string) error {
	// 1. Fetch all relevant nodes (tables, views, semantic models)
	query := `
		SELECT n.id, n.node_name, n.qualified_path, nt.name as node_type, n.description, n.properties
		FROM catalog_node n
		JOIN catalog_node_type nt ON n.node_type_id = nt.id
		WHERE n.tenant_id = $1 AND nt.name IN ('Table', 'View', 'Semantic Model', 'Calculation')
	`
	var nodes []CatalogNode
	err := s.db.SelectContext(ctx, &nodes, query, tenantID)
	if err != nil {
		return fmt.Errorf("failed to fetch catalog nodes: %w", err)
	}

	// 2. Ensure output directory exists
	if err := os.MkdirAll(s.schemaPath, 0755); err != nil {
		return fmt.Errorf("failed to create schema directory: %w", err)
	}

	// 3. Generate a file for each node (simplification: one cube per file)
	for _, node := range nodes {
		cubeDef, err := s.generateCubeDefinition(ctx, node)
		if err != nil {
			// Log error but continue? For now, return error.
			return fmt.Errorf("failed to generate cube for %s: %w", node.NodeName, err)
		}

		filename := fmt.Sprintf("%s.js", sanitizeFilename(node.NodeName))
		path := filepath.Join(s.schemaPath, filename)
		if err := os.WriteFile(path, []byte(cubeDef), 0644); err != nil {
			return fmt.Errorf("failed to write file %s: %w", path, err)
		}
	}

	return nil
}

func (s *CubeSyncService) generateCubeDefinition(ctx context.Context, node CatalogNode) (string, error) {
	cubeName := sanitizeCubeName(node.NodeName)
	sqlTable := node.QualifiedPath

	// Parse properties for additional metadata
	var props map[string]interface{}
	if len(node.Properties) > 0 {
		_ = json.Unmarshal(node.Properties, &props)
		if val, ok := props["sql_table"].(string); ok {
			sqlTable = val
		}
	}

	// Fetch edges (joins) for this node
	joins, err := s.fetchJoins(ctx, node.ID)
	if err != nil {
		return "", fmt.Errorf("failed to fetch joins: %w", err)
	}

	// Fetch columns/dimensions from properties
	dimensions := s.parseDimensions(props)
	measures := s.parseMeasures(props)

	sb := strings.Builder{}
	sb.WriteString(fmt.Sprintf("cube(`%s`, {\n", cubeName))
	sb.WriteString(fmt.Sprintf("  sql: `SELECT * FROM %s`,\n", sqlTable))
	
	// Joins
	sb.WriteString("\n  joins: {\n")
	for _, join := range joins {
		sb.WriteString(fmt.Sprintf("    %s: {\n", sanitizeCubeName(join.TargetName)))
		sb.WriteString(fmt.Sprintf("      sql: `${CUBE}.id = ${%s}.id`,\n", sanitizeCubeName(join.TargetName)))
		sb.WriteString(fmt.Sprintf("      relationship: `%s`\n", mapRelationshipType(join.RelationshipType)))
		sb.WriteString("    },\n")
	}
	sb.WriteString("  },\n")

	// Measures
	sb.WriteString("\n  measures: {\n")
	if len(measures) == 0 {
		// Default count measure
		sb.WriteString("    count: {\n")
		sb.WriteString("      type: `count`,\n")
		sb.WriteString("      drillMembers: []\n")
		sb.WriteString("    }\n")
	} else {
		for _, measure := range measures {
			sb.WriteString(fmt.Sprintf("    %s: {\n", sanitizeFieldName(measure.Name)))
			sb.WriteString(fmt.Sprintf("      sql: `%s`,\n", measure.SQL))
			sb.WriteString(fmt.Sprintf("      type: `%s`\n", measure.Type))
			sb.WriteString("    },\n")
		}
	}
	sb.WriteString("  },\n")

	// Dimensions
	sb.WriteString("\n  dimensions: {\n")
	if len(dimensions) == 0 {
		// Default id dimension
		sb.WriteString("    id: {\n")
		sb.WriteString("      sql: `id`,\n")
		sb.WriteString("      type: `string`,\n")
		sb.WriteString("      primaryKey: true\n")
		sb.WriteString("    }\n")
	} else {
		for i, dim := range dimensions {
			sb.WriteString(fmt.Sprintf("    %s: {\n", sanitizeFieldName(dim.Name)))
			sb.WriteString(fmt.Sprintf("      sql: `%s`,\n", dim.SQL))
			sb.WriteString(fmt.Sprintf("      type: `%s`", dim.Type))
			if dim.IsPrimaryKey {
				sb.WriteString(",\n      primaryKey: true")
			}
			sb.WriteString("\n    }")
			if i < len(dimensions)-1 {
				sb.WriteString(",")
			}
			sb.WriteString("\n")
		}
	}
	sb.WriteString("  }\n")
	sb.WriteString("});\n")

	return sb.String(), nil
}

type CubeDimension struct {
	Name         string
	SQL          string
	Type         string
	IsPrimaryKey bool
}

type CubeMeasure struct {
	Name string
	SQL  string
	Type string
}

func (s *CubeSyncService) fetchJoins(ctx context.Context, nodeID string) ([]CatalogEdge, error) {
	query := `
		SELECT ce.source_node_id, ce.target_node_id, ce.relationship_type, n.node_name as target_name
		FROM catalog_edge ce
		JOIN catalog_node n ON ce.target_node_id = n.id
		WHERE ce.source_node_id = $1
	`
	var edges []CatalogEdge
	err := s.db.SelectContext(ctx, &edges, query, nodeID)
	return edges, err
}

func (s *CubeSyncService) parseDimensions(props map[string]interface{}) []CubeDimension {
	var dimensions []CubeDimension
	
	// Look for a "columns" or "dimensions" array in properties
	if cols, ok := props["columns"].([]interface{}); ok {
		for _, col := range cols {
			if colMap, ok := col.(map[string]interface{}); ok {
				dim := CubeDimension{
					Name: getStringProp(colMap, "name"),
					SQL:  getStringProp(colMap, "name"), // Default SQL to column name
					Type: mapDataTypeToCube(getStringProp(colMap, "type")),
				}
				if pkVal, ok := colMap["is_primary_key"].(bool); ok {
					dim.IsPrimaryKey = pkVal
				}
				dimensions = append(dimensions, dim)
			}
		}
	}
	
	return dimensions
}

func (s *CubeSyncService) parseMeasures(props map[string]interface{}) []CubeMeasure {
	var measures []CubeMeasure
	
	// Look for a "measures" array in properties
	if meas, ok := props["measures"].([]interface{}); ok {
		for _, m := range meas {
			if measMap, ok := m.(map[string]interface{}); ok {
				measure := CubeMeasure{
					Name: getStringProp(measMap, "name"),
					SQL:  getStringProp(measMap, "sql"),
					Type: getStringProp(measMap, "type"),
				}
				measures = append(measures, measure)
			}
		}
	}
	
	return measures
}

func getStringProp(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func mapDataTypeToCube(dataType string) string {
	switch strings.ToLower(dataType) {
	case "string", "text", "varchar":
		return "string"
	case "number", "int", "integer", "bigint", "decimal", "numeric":
		return "number"
	case "boolean", "bool":
		return "boolean"
	case "timestamp", "datetime", "date":
		return "time"
	default:
		return "string"
	}
}

func mapRelationshipType(relType string) string {
	switch strings.ToLower(relType) {
	case "one_to_many", "has_many":
		return "hasMany"
	case "belongs_to", "many_to_one":
		return "belongsTo"
	case "one_to_one", "has_one":
		return "hasOne"
	default:
		return "belongsTo"
	}
}

func sanitizeFieldName(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "_"))
}

func sanitizeFilename(name string) string {
	return strings.ToLower(strings.ReplaceAll(name, " ", "_"))
}

func sanitizeCubeName(name string) string {
	// Cube names usually CamelCase or snake_case. Let's stick to the node name but safe.
	return strings.ReplaceAll(name, " ", "")
}
