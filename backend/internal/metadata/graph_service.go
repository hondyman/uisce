package metadata

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// CatalogNode represents a node in the metadata graph (Semantic Entity or Physical Table)
type CatalogNode struct {
	ID          uuid.UUID       `db:"id" json:"id"`
	NodeName    string          `db:"node_name" json:"node_name"` // e.g. "Employee" or "hr_employees"
	NodeTypeID  *uuid.UUID      `db:"node_type_id" json:"node_type_id,omitempty"`
	TenantID    string          `db:"tenant_id" json:"tenant_id"`
	Description *string         `db:"description" json:"description,omitempty"`
	Properties  json.RawMessage `db:"properties" json:"properties"` // Stores tableName, primaryKey, etc.
	CreatedAt   time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// CatalogEdge represents a relationship or logic between nodes
type CatalogEdge struct {
	ID         uuid.UUID       `db:"id" json:"id"`
	SourceID   uuid.UUID       `db:"source_id" json:"source_id"`
	TargetID   uuid.UUID       `db:"target_id" json:"target_id"`
	EdgeType   string          `db:"edge_type" json:"edge_type"`   // e.g. "HAS_RELATIONSHIP", "MAPS_TO"
	Properties json.RawMessage `db:"properties" json:"properties"` // Stores joinCondition, validationRule
	CreatedAt  time.Time       `db:"created_at" json:"created_at"`
}

// GraphService handles metadata graph operations
type GraphService struct {
	DB *sqlx.DB
}

// NewGraphService creates a new GraphService
func NewGraphService(db *sqlx.DB) *GraphService {
	return &GraphService{DB: db}
}

// GetNodeByName retrieves a node by its name and tenant
func (s *GraphService) GetNodeByName(ctx context.Context, tenantID, name string) (*CatalogNode, error) {
	var node CatalogNode
	// Note: checking both node_name (from 002700) or name (from 000032) depending on what's actually there.
	// The migration 000032 improved schema uses 'name', 002700 uses 'node_name'.
	// We will assume 'node_name' based on 002700 being the glossary schema, but check migration logic.
	// Actually, 000032 creates 'name', 002700 adds 'node_name'.
	// Let's select 'name' aliased as 'node_name' if 'node_name' is null?
	// To be safe and consistent with the "Glossary" schema in 002700, we use `node_name`.

	// Assuming the schema converged to use `node_name` for the glossary/business term.
	// If the table was created by 000032 it had `name`, 002700 tries to add `node_name`.
	// We'll write the query to support the `node_name` column.
	query := `SELECT id, node_name, tenant_id, properties, created_at, updated_at 
	          FROM catalog_node 
	          WHERE tenant_id = $1 AND node_name = $2`

	err := s.DB.GetContext(ctx, &node, query, tenantID, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, err
	}
	return &node, nil
}

// GetEdges retrieves all outgoing edges for a node
func (s *GraphService) GetEdges(ctx context.Context, nodeID uuid.UUID) ([]CatalogEdge, error) {
	var edges []CatalogEdge
	query := `SELECT id, source_id, target_id, edge_type, properties, created_at 
	          FROM catalog_edge 
	          WHERE source_id = $1`

	err := s.DB.SelectContext(ctx, &edges, query, nodeID)
	return edges, err
}

// PathResult represents a step in a traversal path
type PathResult struct {
	SourceID   uuid.UUID       `db:"source_node_id"`
	TargetID   uuid.UUID       `db:"target_node_id"`
	Path       pq.StringArray  `db:"path"` // Array of Edge IDs
	Depth      int             `db:"depth"`
	Properties json.RawMessage `db:"properties"`
}

// FindPath performs a recursive CTE traversal to find a path between two nodes
func (s *GraphService) FindPath(ctx context.Context, startNodeID, endNodeID uuid.UUID) ([]PathResult, error) {
	query := `
	WITH RECURSIVE graph_path (source_node_id, target_node_id, path, depth) AS (
		-- Base Case: Direct neighbors
		SELECT 
			source_id, 
			target_id, 
			ARRAY[id::text] as path, 
			1 as depth
		FROM catalog_edge
		WHERE source_id = $1

		UNION ALL

		-- Recursive Step: Traverse to next neighbor
		SELECT 
			e.source_id, 
			e.target_id, 
			gp.path || e.id::text, 
			gp.depth + 1
		FROM catalog_edge e
		JOIN graph_path gp ON e.source_id = gp.target_node_id
		WHERE gp.depth < 10 -- Prevent infinite loops
	)
	SELECT source_node_id, target_node_id, path, depth
	FROM graph_path 
	WHERE target_node_id = $2
	ORDER BY depth ASC
	LIMIT 1;
	`

	var results []PathResult
	// NOTE: Because PathResult struct doesn't perfectly match the CTE output column names by default with sqlx unless mapped
	// We re-map in query or struct. Struct tags are db:"source_node_id".
	// The query outputs source_node_id, target_node_id, path, depth.

	err := s.DB.SelectContext(ctx, &results, query, startNodeID, endNodeID)
	return results, err
}

// GetValidationRules fetches validation rules from edges connected to a node
func (s *GraphService) GetValidationRules(ctx context.Context, nodeID uuid.UUID) ([]string, error) {
	edges, err := s.GetEdges(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	var rules []string
	for _, edge := range edges {
		// Look for edges that define validation, e.g., types 'HAS_VALIDATION'
		// OR properties containing "validationRule"
		var props map[string]interface{}
		if len(edge.Properties) > 0 {
			if err := json.Unmarshal(edge.Properties, &props); err == nil {
				if rule, ok := props["validationRule"].(string); ok && rule != "" {
					rules = append(rules, rule)
				}
			}
		}
	}
	return rules, nil
}
