package charts

import (
	"log"
)

// convertSemanticToReactFlow converts semantic lineage data to ReactFlow format
func convertSemanticToReactFlow(semanticChart *SemanticLineageChart) *TechnicalLineageChart {
	reactFlowChart := &TechnicalLineageChart{
		Nodes:    []ReactFlowNode{},
		Edges:    []ReactFlowEdge{},
		Viewport: semanticChart.Viewport,
		Metadata: semanticChart.Metadata,
	}

	if reactFlowChart.Metadata == nil {
		reactFlowChart.Metadata = make(map[string]interface{})
	}
	reactFlowChart.Metadata["chartType"] = "semantic_lineage"
	reactFlowChart.Metadata["originalFormat"] = "semantic"

	var yPos float64 = 0
	const nodeSpacing = 300
	const layerSpacing = 200

	// Convert business terms
	for i, term := range semanticChart.BusinessTerms {
		node := ReactFlowNode{
			ID:   term.ID.String(),
			Type: "businessTerm",
			Position: map[string]float64{
				"x": float64(i%4) * nodeSpacing,
				"y": yPos,
			},
			Data: map[string]interface{}{
				"label":         term.NodeName,
				"nodeType":      term.NodeType,
				"description":   term.Description,
				"qualifiedPath": term.QualifiedPath,
				"properties":    term.Properties,
				"id":            term.ID.String(),
				"layer":         "business",
			},
		}
		reactFlowChart.Nodes = append(reactFlowChart.Nodes, node)
		if (i+1)%4 == 0 {
			yPos += layerSpacing
		}
	}

	// Convert semantic terms
	yPos += layerSpacing * 2
	for i, term := range semanticChart.SemanticTerms {
		node := ReactFlowNode{
			ID:   term.ID.String(),
			Type: "semanticTerm",
			Position: map[string]float64{
				"x": float64(i%4) * nodeSpacing,
				"y": yPos,
			},
			Data: map[string]interface{}{
				"label":         term.NodeName,
				"nodeType":      term.NodeType,
				"description":   term.Description,
				"qualifiedPath": term.QualifiedPath,
				"properties":    term.Properties,
				"id":            term.ID.String(),
				"layer":         "semantic",
			},
		}
		reactFlowChart.Nodes = append(reactFlowChart.Nodes, node)
		if (i+1)%4 == 0 {
			yPos += layerSpacing
		}
	}

	// Convert semantic columns
	yPos += layerSpacing * 2
	for i, column := range semanticChart.SemanticColumns {
		node := ReactFlowNode{
			ID:   column.ID.String(),
			Type: "semanticColumn",
			Position: map[string]float64{
				"x": float64(i%4) * nodeSpacing,
				"y": yPos,
			},
			Data: map[string]interface{}{
				"label":         column.NodeName,
				"nodeType":      column.NodeType,
				"description":   column.Description,
				"qualifiedPath": column.QualifiedPath,
				"properties":    column.Properties,
				"id":            column.ID.String(),
				"layer":         "semantic_column",
			},
		}
		reactFlowChart.Nodes = append(reactFlowChart.Nodes, node)
		if (i+1)%4 == 0 {
			yPos += layerSpacing
		}
	}

	// Convert database columns with enhanced qualified paths
	yPos += layerSpacing * 2
	for i, column := range semanticChart.DatabaseColumns {
		// Extract schema, table, and column from properties for enhanced hover
		schema, _ := column.Properties["schema"].(string)
		table, _ := column.Properties["table"].(string)
		columnName, _ := column.Properties["column"].(string)

		node := ReactFlowNode{
			ID:   column.ID.String(),
			Type: "databaseColumn",
			Position: map[string]float64{
				"x": float64(i%4) * nodeSpacing,
				"y": yPos,
			},
			Data: map[string]interface{}{
				"label":         column.NodeName,
				"nodeType":      column.NodeType,
				"description":   column.Description,
				"qualifiedPath": column.QualifiedPath, // This now contains schema.table.column
				"properties":    column.Properties,
				"id":            column.ID.String(),
				"layer":         "database",
				// Enhanced data for frontend tooltips
				"schema":     schema,
				"table":      table,
				"tableName":  table,
				"column":     columnName,
				"columnName": columnName,
			},
		}
		reactFlowChart.Nodes = append(reactFlowChart.Nodes, node)
		if (i+1)%4 == 0 {
			yPos += layerSpacing
		}
	}

	// Convert edges
	for _, edge := range semanticChart.Edges {
		label := edge.RelationshipType
		if predicate, ok := edge.Properties["predicate"].(string); ok {
			label = predicate
		}

		reactFlowEdge := ReactFlowEdge{
			ID:     edge.ID.String(),
			Source: edge.SourceID.String(),
			Target: edge.TargetID.String(),
			Type:   getEdgeTypeForRelationship(edge.RelationshipType),
			Label:  label,
			Data: map[string]interface{}{
				"relationshipType": edge.RelationshipType,
				"properties":       edge.Properties,
				"edgeTypeId":       edge.EdgeType,
			},
		}

		if edge.RelationshipType == "defines" || edge.RelationshipType == "implements" {
			reactFlowEdge.Animated = true
		}

		reactFlowChart.Edges = append(reactFlowChart.Edges, reactFlowEdge)
	}

	// Update metadata
	reactFlowChart.Metadata["semanticEdgeCount"] = len(reactFlowChart.Edges)
	reactFlowChart.Metadata["totalNodes"] = len(reactFlowChart.Nodes)
	reactFlowChart.Metadata["businessTermCount"] = len(semanticChart.BusinessTerms)
	reactFlowChart.Metadata["semanticTermCount"] = len(semanticChart.SemanticTerms)
	reactFlowChart.Metadata["databaseColumnCount"] = len(semanticChart.DatabaseColumns)

	log.Printf("Converted semantic chart to ReactFlow: %d nodes, %d edges", len(reactFlowChart.Nodes), len(reactFlowChart.Edges))
	return reactFlowChart
}

// getEdgeTypeForRelationship maps semantic relationships to ReactFlow edge types
func getEdgeTypeForRelationship(relationshipType string) string {
	switch relationshipType {
	case "defines":
		return "smoothstep"
	case "implements":
		return "step"
	case "maps_to":
		return "straight"
	default:
		return "default"
	}
}
