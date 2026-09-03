// Package graphview holds the single normalized graph response shape shared by
// every catalog graph producer (lineage, BO graph, semantic lenses, and the
// tenant-configurable ViewDefinition endpoint), so the frontend has one
// contract to consume regardless of which backend path built the graph.
package graphview

import (
	"encoding/json"

	"github.com/hondyman/uisce/backend/internal/lineage"
)

// ResponseNode represents a node in API response format (compatible with the frontend).
type ResponseNode struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Label      string                 `json:"label"`
	ParentID   *string                `json:"parentId,omitempty"`
	IsCluster  bool                   `json:"isCluster,omitempty"`
	MemberIDs  []string               `json:"memberIds,omitempty"`
	Properties map[string]interface{} `json:"properties"`
}

// ResponseEdge represents an edge in API response format (compatible with the frontend).
type ResponseEdge struct {
	ID         string                 `json:"id,omitempty"`
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// ResponseGraph represents a graph in the frontend-compatible normalized format.
type ResponseGraph struct {
	Nodes []ResponseNode `json:"nodes"`
	Edges []ResponseEdge `json:"edges"`
}

// ConvertLineageGraph converts a lineage.Graph (nodes/edges keyed by FromID/ToID,
// metadata as json.RawMessage) into the normalized ResponseGraph shape.
func ConvertLineageGraph(graph *lineage.Graph) *ResponseGraph {
	resp := &ResponseGraph{
		Nodes: make([]ResponseNode, 0, len(graph.Nodes)),
		Edges: make([]ResponseEdge, 0, len(graph.Edges)),
	}

	for _, node := range graph.Nodes {
		var props map[string]interface{}
		if len(node.Metadata) > 0 {
			json.Unmarshal(node.Metadata, &props)
		}
		if props == nil {
			props = make(map[string]interface{})
		}

		resp.Nodes = append(resp.Nodes, ResponseNode{
			ID:         node.ID,
			Type:       string(node.Type),
			Label:      node.Name,
			Properties: props,
		})
	}

	for _, edge := range graph.Edges {
		var props map[string]interface{}
		if len(edge.Metadata) > 0 {
			json.Unmarshal(edge.Metadata, &props)
		}
		if props == nil {
			props = make(map[string]interface{})
		}

		resp.Edges = append(resp.Edges, ResponseEdge{
			ID:         edge.FromID + "->" + edge.ToID,
			Source:     edge.FromID,
			Target:     edge.ToID,
			Type:       string(edge.Type),
			Properties: props,
		})
	}

	return resp
}
