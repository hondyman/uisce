package mdm

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics"
)

// SemanticViewGenerator automates the creation of relational views from semanticBindings.
type SemanticViewGenerator struct {
	graphService *analytics.SemanticGraphService
}

func NewSemanticViewGenerator(gs *analytics.SemanticGraphService) *SemanticViewGenerator {
	return &SemanticViewGenerator{graphService: gs}
}

// GenerateViewSQL constructs a CREATE OR REPLACE VIEW statement for a given business object.
func (g *SemanticViewGenerator) GenerateViewSQL(ctx context.Context, tenantID uuid.UUID, boID uuid.UUID) (string, error) {
	rootNode, err := g.graphService.GetNodeByID(boID)
	if err != nil {
		return "", err
	}
	if rootNode == nil || rootNode.NodeType != "business_object" {
		return "", fmt.Errorf("root node must be a business_object")
	}

	// 1. Find the physical driver table
	driverTable, err := g.findPhysicalTable(boID)
	if err != nil {
		return "", fmt.Errorf("failed to find physical table for %s: %w", rootNode.NodeName, err)
	}

	viewName := fmt.Sprintf("semantic_views.sv_%s", rootNode.NodeName)
	sql := fmt.Sprintf("CREATE OR REPLACE VIEW %s AS\nSELECT\n  t0.*", viewName)

	// 2. Discover semantic links for JOINs
	edges, err := g.graphService.GetOutgoingEdges(boID)
	if err == nil {
		joinIdx := 1
		for _, edge := range edges {
			if edge.EdgeType == "holds_security" || edge.EdgeType == analytics.EdgeTypeBORelatesToBO {
				targetNode, tErr := g.graphService.GetNodeByID(edge.TargetNodeID)
				if tErr != nil || targetNode == nil {
					continue
				}

				targetTable, pErr := g.findPhysicalTable(edge.TargetNodeID)
				if pErr != nil {
					continue
				}

				// Construct dynamic JOIN logic from edge properties
				props, _ := g.graphService.GetEdgeProperties(boID, edge.TargetNodeID, edge.EdgeType)

				sourceKey := "id"
				if val, ok := props["source_column"].(string); ok {
					sourceKey = val
				} else if edge.EdgeType == "holds_security" {
					sourceKey = "security_id" // Specific domain fallback
				} else {
					sourceKey = fmt.Sprintf("%s_id", targetNode.NodeName) // Generic fallback
				}

				targetKey := "id"
				if val, ok := props["target_column"].(string); ok {
					targetKey = val
				}

				sql += fmt.Sprintf(",\n  t%d.*", joinIdx)
				sql += fmt.Sprintf("\nFROM public.%s t0", driverTable)
				sql += fmt.Sprintf("\nLEFT JOIN public.%s t%d ON t0.%s = t%d.%s", targetTable, joinIdx, sourceKey, joinIdx, targetKey)

				joinIdx++
			}
		}
	} else {
		sql += fmt.Sprintf("\nFROM public.%s t0", driverTable)
	}

	return sql + ";", nil
}

func (g *SemanticViewGenerator) findPhysicalTable(boID uuid.UUID) (string, error) {
	edges, err := g.graphService.GetOutgoingEdges(boID)
	if err != nil {
		return "", err
	}

	for _, edge := range edges {
		if edge.EdgeType == analytics.EdgeTypeBODrivesTable {
			target, err := g.graphService.GetNodeByID(edge.TargetNodeID)
			if err == nil && target != nil {
				return target.NodeName, nil
			}
		}
	}

	// Dynamic inference fallback: if no explicit edge, look for table with same name
	node, _ := g.graphService.GetNodeByID(boID)
	if node != nil {
		return node.NodeName, nil
	}

	return "", fmt.Errorf("physical table mapping not found for node %s", boID)
}
