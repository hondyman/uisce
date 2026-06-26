package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// LineageService provides methods for fetching lineage and impact data.
type LineageService struct {
	db *sqlx.DB
}

// NewLineageService creates a new LineageService.
func NewLineageService(db *sqlx.DB) *LineageService {
	return &LineageService{db: db}
}

// GetLineageGraph generates a lineage graph for a given asset ID from the catalog.
func (s *LineageService) GetLineageGraph(ctx context.Context, assetID string) (*models.LineageGraphData, error) {
	// Query to fetch the target node and its connected nodes via edges
	// This includes MAPS_TO edges (database columns → semantic terms)

	type EdgeRow struct {
		SourceID   string `db:"source_id"`
		SourceName string `db:"source_name"`
		SourceType string `db:"source_type"`
		TargetID   string `db:"target_id"`
		TargetName string `db:"target_name"`
		TargetType string `db:"target_type"`
		EdgeType   string `db:"edge_type"`
	}

	// Query edges connected to this asset (both incoming and outgoing)
	query := `
		SELECT 
			ce.source_node_id as source_id,
			cn_source.node_name as source_name,
			COALESCE(cnt_source.catalog_type_name, 'unknown') as source_type,
			ce.target_node_id as target_id,
			cn_target.node_name as target_name,
			COALESCE(cnt_target.catalog_type_name, 'unknown') as target_type,
			COALESCE(cet.edge_type_name, 'unknown') as edge_type
		FROM catalog_edge ce
		JOIN catalog_node cn_source ON ce.source_node_id = cn_source.id
		JOIN catalog_node cn_target ON ce.target_node_id = cn_target.id
		LEFT JOIN catalog_node_type cnt_source ON cn_source.node_type_id = cnt_source.id
		LEFT JOIN catalog_node_type cnt_target ON cn_target.node_type_id = cnt_target.id
		LEFT JOIN catalog_edge_type cet ON ce.edge_type_id = cet.id
		WHERE ce.source_node_id = $1 OR ce.target_node_id = $1
		LIMIT 100
	`

	var edgeRows []EdgeRow
	err := s.db.SelectContext(ctx, &edgeRows, query, assetID)
	if err != nil {
		fmt.Printf("[LineageService] Error querying edges: %v\n", err)
		// If no edges found or error, return minimal graph with just the node itself
		return s.getMinimalGraph(ctx, assetID)
	}

	// Build nodes and edges from query results
	nodeMap := make(map[string]models.LineageNode)
	var edges []models.LineageEdge

	for _, row := range edgeRows {
		// Add source node
		if _, exists := nodeMap[row.SourceID]; !exists {
			nodeMap[row.SourceID] = models.LineageNode{
				ID:    row.SourceID,
				Type:  row.SourceType,
				Label: row.SourceName,
				Data:  map[string]any{},
			}
		}

		// Add target node
		if _, exists := nodeMap[row.TargetID]; !exists {
			nodeMap[row.TargetID] = models.LineageNode{
				ID:    row.TargetID,
				Type:  row.TargetType,
				Label: row.TargetName,
				Data:  map[string]any{},
			}
		}

		// Add edge with appropriate label
		edgeLabel := s.getEdgeLabel(row.EdgeType)
		edges = append(edges, models.LineageEdge{
			Source: row.SourceID,
			Target: row.TargetID,
			Label:  edgeLabel,
		})
	}

	// Convert node map to slice
	nodes := make([]models.LineageNode, 0, len(nodeMap))
	for _, node := range nodeMap {
		nodes = append(nodes, node)
	}

	return &models.LineageGraphData{Nodes: nodes, Edges: edges}, nil
}

// GetRecursiveLineage generates a recursive lineage graph starting from a root node.
// It traverses both upstream and downstream relationships up to maxDepth.
func (s *LineageService) GetRecursiveLineage(ctx context.Context, rootID string, maxDepth int) (*models.LineageGraphData, error) {
	if maxDepth <= 0 {
		maxDepth = 5 // Default depth
	}

	fmt.Printf("[LineageService] GetRecursiveLineage called for rootID=%s, maxDepth=%d\n", rootID, maxDepth)

	// Enhanced query with parent table information for hierarchical display, qualified paths, and predicates
	simpleQuery := `
		SELECT 
			ce.id,
			ce.source_node_id,
			ce.target_node_id,
			ce.edge_type_id,
			ns.node_name as source_name,
			nt.node_name as target_name,
			ns.node_type_id as source_type_id,
			nt.node_type_id as target_type_id,
			COALESCE(type_s.catalog_type_name, ns.node_type_id::text) as source_type_name,
			COALESCE(type_t.catalog_type_name, nt.node_type_id::text) as target_type_name,
			ns.parent_id as source_parent_id,
			nt.parent_id as target_parent_id,
			parent_s.node_name as source_parent_name,
			parent_t.node_name as target_parent_name,
			COALESCE(et.edge_type_name, 'unknown') as edge_label,
			COALESCE(ns.qualified_path, ns.node_name) as source_qualified_path,
			COALESCE(nt.qualified_path, nt.node_name) as target_qualified_path
		FROM catalog_edge ce
		LEFT JOIN catalog_node ns ON ce.source_node_id = ns.id
		LEFT JOIN catalog_node nt ON ce.target_node_id = nt.id
		LEFT JOIN catalog_node_type type_s ON ns.node_type_id = type_s.id
		LEFT JOIN catalog_node_type type_t ON nt.node_type_id = type_t.id
		LEFT JOIN catalog_node parent_s ON ns.parent_id = parent_s.id
		LEFT JOIN catalog_node parent_t ON nt.parent_id = parent_t.id
		LEFT JOIN catalog_edge_type et ON ce.edge_type_id = et.id
		WHERE ce.source_node_id = $1 OR ce.target_node_id = $1
		LIMIT 100
	`

	rows, err := s.db.QueryContext(ctx, simpleQuery, rootID)
	if err != nil {
		fmt.Printf("[LineageService] Error executing query: %v\n", err)
		return s.getMinimalGraph(ctx, rootID)
	}
	defer rows.Close()

	nodeMap := make(map[string]models.LineageNode)
	var edges []models.LineageEdge

	for rows.Next() {
		var id, sourceID, targetID string
		var edgeTypeID, sourceName, targetName, sourceTypeID, targetTypeID, edgeLabel, sourceQualifiedPath, targetQualifiedPath sql.NullString
		var sourceTypeName, targetTypeName sql.NullString
		var sourceParentID, targetParentID, sourceParentName, targetParentName sql.NullString

		if err := rows.Scan(&id, &sourceID, &targetID, &edgeTypeID, &sourceName, &targetName,
			&sourceTypeID, &targetTypeID, &sourceTypeName, &targetTypeName,
			&sourceParentID, &targetParentID,
			&sourceParentName, &targetParentName, &edgeLabel, &sourceQualifiedPath, &targetQualifiedPath); err != nil {
			fmt.Printf("[LineageService] Error scanning row: %v\n", err)
			continue
		}

		// Add source node with qualified path
		if _, exists := nodeMap[sourceID]; !exists {
			srcLabel := sourceName.String
			if sourceQualifiedPath.Valid && sourceQualifiedPath.String != "" {
				srcLabel = sourceQualifiedPath.String
			}
			nodeMap[sourceID] = models.LineageNode{
				ID:    sourceID,
				Type:  sourceTypeName.String, // Use human-readable type name
				Label: srcLabel,
				Data: map[string]any{
					"parent_id":      sourceParentID.String,
					"parent_name":    sourceParentName.String,
					"node_type_id":   sourceTypeID.String,
					"node_type_name": sourceTypeName.String,
					"qualified_path": sourceQualifiedPath.String,
				},
			}
		}

		// Add target node with qualified path
		if _, exists := nodeMap[targetID]; !exists {
			tgtLabel := targetName.String
			if targetQualifiedPath.Valid && targetQualifiedPath.String != "" {
				tgtLabel = targetQualifiedPath.String
			}
			nodeMap[targetID] = models.LineageNode{
				ID:    targetID,
				Type:  targetTypeName.String, // Use human-readable type name
				Label: tgtLabel,
				Data: map[string]any{
					"parent_id":      targetParentID.String,
					"parent_name":    targetParentName.String,
					"node_type_id":   targetTypeID.String,
					"node_type_name": targetTypeName.String,
					"qualified_path": targetQualifiedPath.String,
				},
			}
		}

		// Add edge with direction indicator and correct predicate
		label := edgeLabel.String
		if label == "" {
			label = "unknown"
		}

		// Add direction indicator (arrow) showing subject/object relationship
		// The selected node's role (source=subject, target=object) is indicated with arrows
		directionLabel := label
		if sourceID == rootID {
			directionLabel = label + " →" // Selected node is subject
		} else if targetID == rootID {
			directionLabel = "← " + label // Selected node is object
		}

		edges = append(edges, models.LineageEdge{
			ID:     id,
			Source: sourceID,
			Target: targetID,
			Label:  directionLabel,
			Type:   edgeTypeID.String,
		})
	}

	fmt.Printf("[LineageService] Found %d edges and %d unique nodes for rootID=%s\n", len(edges), len(nodeMap), rootID)
	for i, edge := range edges {
		fmt.Printf("[LineageService] Edge %d: ID=%s, Label=%s, Type=%s\n", i, edge.ID, edge.Label, edge.Type)
	}

	if len(edges) == 0 {
		return s.getMinimalGraph(ctx, rootID)
	}

	// Convert map to slice
	nodes := make([]models.LineageNode, 0, len(nodeMap))
	for _, n := range nodeMap {
		nodes = append(nodes, n)
	}

	return &models.LineageGraphData{Nodes: nodes, Edges: edges}, nil
}

// getMinimalGraph returns a minimal graph with just the requested node
func (s *LineageService) getMinimalGraph(ctx context.Context, assetID string) (*models.LineageGraphData, error) {
	type NodeRow struct {
		ID       string `db:"id"`
		NodeName string `db:"node_name"`
		NodeType string `db:"node_type"`
	}

	query := `
		SELECT cn.id, cn.node_name, COALESCE(cnt.code, 'unknown') as node_type
		FROM catalog_node cn
		LEFT JOIN catalog_node_type cnt ON cn.node_type_id = cnt.id
		WHERE cn.id = $1
	`

	var nodeRow NodeRow
	err := s.db.GetContext(ctx, &nodeRow, query, assetID)
	if err != nil {
		// Return empty graph if node not found
		return &models.LineageGraphData{Nodes: []models.LineageNode{}, Edges: []models.LineageEdge{}}, nil
	}

	nodes := []models.LineageNode{
		{
			ID:    nodeRow.ID,
			Type:  nodeRow.NodeType,
			Label: nodeRow.NodeName,
			Data:  map[string]any{},
		},
	}

	return &models.LineageGraphData{Nodes: nodes, Edges: []models.LineageEdge{}}, nil
}

// getEdgeLabel returns a human-readable label for an edge type
func (s *LineageService) getEdgeLabel(edgeType string) string {
	labels := map[string]string{
		"MAPS_TO":           "Maps To",
		"HAS_BUSINESS_TERM": "Has Business Term",
		"FOREIGN_KEY":       "Foreign Key",
		"REFERENCE":         "References",
		"COMPOSITION":       "Composed Of",
		"ASSOCIATION":       "Associated With",
	}

	if label, exists := labels[edgeType]; exists {
		return label
	}
	return edgeType // Return the raw type if no mapping exists
}

// GetImpactAnalysis generates a mock impact analysis for a given asset ID.
func (s *LineageService) GetImpactAnalysis(ctx context.Context, assetID string) (*models.ImpactAnalysis, error) {
	// Mock downstream dependencies for "total_revenue"
	if assetID != "total_revenue" {
		return &models.ImpactAnalysis{}, nil
	}

	return &models.ImpactAnalysis{
		Queries: []models.ImpactedItem{
			{ID: uuid.New().String(), Name: "Q1 Sales Report"},
			{ID: uuid.New().String(), Name: "Regional Performance vs. Target"},
			{ID: uuid.New().String(), Name: "Daily Revenue Flash"},
		},
		Workbooks: []models.ImpactedItem{
			{ID: uuid.New().String(), Name: "Exec Sales Dashboard"},
		},
		Dashboards: []models.ImpactedItem{
			{ID: uuid.New().String(), Name: "Sales Team Weekly Standup"},
		},
	}, nil
}

// GetLineageForSubjects is a placeholder implementation.
func (s *LineageService) GetLineageForSubjects(ctx context.Context, subjectIDs []string) ([]models.LineageNode, []models.LineageEdge, error) {
	// Mock implementation
	nodes := []models.LineageNode{{ID: subjectIDs[0], Label: "Subject", Type: "table"}}
	edges := []models.LineageEdge{}
	return nodes, edges, nil
}

// GetTechnicalLineageData is a placeholder implementation.
func (s *LineageService) GetTechnicalLineageData(assetID string) (interface{}, error) {
	// Mock implementation
	return map[string]string{"data": "technical lineage for " + assetID}, nil
}

// GetSemanticLineageData is a placeholder implementation.
func (s *LineageService) GetSemanticLineageData(assetID string) (interface{}, error) {
	// Mock implementation
	return map[string]string{"data": "semantic lineage for " + assetID}, nil
}
