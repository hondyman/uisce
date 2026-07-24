package services

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/jmoiron/sqlx/types"
)

// LayoutType defines the type of layout to be used for the lineage chart.
type LayoutType string

const (
	HierarchicalLayout LayoutType = "hierarchical"
	FlatLayout         LayoutType = "flat"
)

// EnhancedLineageData holds the data for the lineage chart, including the layout type.
type EnhancedLineageData struct {
	Layout LayoutType      `json:"layout"`
	Nodes  []ReactFlowNode `json:"nodes"`
	Edges  []LineageEdge   `json:"edges"`
}

// BuildHierarchicalTechnicalLineageChart builds a hierarchical lineage chart.
func BuildHierarchicalTechnicalLineageChart(db *sql.DB, tenantID, assetID string) (*EnhancedLineageData, error) {
	// Forcing hierarchical view for demonstration
	return buildHierarchicalView(db, tenantID, assetID)
}

// GetEnhancedLineageData determines the layout and returns the appropriate lineage data.
func GetEnhancedLineageData(db *sql.DB, tenantID, assetID string) (*EnhancedLineageData, error) {
	if shouldUseHierarchicalLayout(assetID) {
		return buildHierarchicalView(db, tenantID, assetID)
	}
	return buildFlatView(db, tenantID, assetID)
}

// GetTechnicalLineage fetches technical lineage data.
// In a real implementation, this would query the database.
// This is a placeholder to satisfy the compiler.
func GetTechnicalLineage(db *sql.DB, tenantID, assetID string) (*LineageData, error) {
	return &LineageData{
		Assets: []LineageAsset{},
		Edges:  []LineageEdge{},
	}, nil
}

// shouldUseHierarchicalLayout determines if a hierarchical layout should be used based on the asset type.
func shouldUseHierarchicalLayout(assetID string) bool {
	// Smart layout detection based on asset type or qualified path
	return strings.Contains(assetID, "column_") || strings.Contains(assetID, "table_")
}

func buildHierarchicalView(db *sql.DB, tenantID, assetID string) (*EnhancedLineageData, error) {
	lineageData, err := GetTechnicalLineage(db, tenantID, assetID)
	if err != nil {
		return nil, err
	}

	hierarchy := BuildHierarchicalDatabaseStructure(lineageData.Assets)
	nodes := ConvertHierarchyToReactFlow(hierarchy, assetID)

	// Create a map for quick node lookup
	nodeMap := make(map[string]bool)
	for _, node := range nodes {
		nodeMap[node.ID] = true
	}

	// Filter edges to only include those connecting existing nodes
	var filteredEdges []LineageEdge
	for _, edge := range lineageData.Edges {
		if _, sourceExists := nodeMap[edge.Source]; sourceExists {
			if _, targetExists := nodeMap[edge.Target]; targetExists {
				filteredEdges = append(filteredEdges, edge)
			}
		}
	}

	return &EnhancedLineageData{
		Layout: HierarchicalLayout,
		Nodes:  nodes,
		Edges:  filteredEdges,
	}, nil
}

func buildFlatView(db *sql.DB, tenantID, assetID string) (*EnhancedLineageData, error) {
	lineageData, err := GetTechnicalLineage(db, tenantID, assetID)
	if err != nil {
		return nil, err
	}

	var nodes []ReactFlowNode
	for _, asset := range lineageData.Assets {
		nodeData, _ := json.Marshal(map[string]interface{}{"label": asset.Name})
		nodes = append(nodes, ReactFlowNode{
			ID:       asset.ID,
			Type:     "default",
			Data:     types.JSONText(nodeData),
			Position: NodePosition{X: 0, Y: 0}, // Position will be set by the frontend
		})
	}

	return &EnhancedLineageData{
		Layout: FlatLayout,
		Nodes:  nodes,
		Edges:  lineageData.Edges,
	}, nil
}
