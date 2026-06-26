package analytics

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/lineage"
	"github.com/jmoiron/sqlx"
)

// NodeType represents the type of node in the impact graph
type NodeType string

const (
	NodeTypeBusinessObject  NodeType = "business_object"
	NodeTypeBoField         NodeType = "BO_FIELD"
	NodeTypeSemanticTerm    NodeType = "semantic_term"
	NodeTypeDBColumn        NodeType = "DB_COLUMN"
	NodeTypeAPIEndpoint     NodeType = "API_ENDPOINT"
	NodeTypeBIArtifact      NodeType = "BI_ARTIFACT"
	NodeTypeAIArtifact      NodeType = "AI_ARTIFACT"
	NodeTypeAccessRule      NodeType = "ACCESS_RULE"
	NodeTypeCalculationTerm NodeType = "calculation_term"
)

// EdgeType represents relationship types in the impact graph
type EdgeType string

const (
	EdgeTypeHasField       EdgeType = "HAS_FIELD"
	EdgeTypeBackedByTerm   EdgeType = "BACKED_BY_TERM"
	EdgeTypeBackedByCalc   EdgeType = "BACKED_BY_CALC"
	EdgeTypeMappedToColumn EdgeType = "MAPPED_TO_COLUMN"
	EdgeTypeExposedViaAPI  EdgeType = "EXPOSED_VIA_API"
	EdgeTypeUsedInBI       EdgeType = "USED_IN_BI"
	EdgeTypeUsedInAI       EdgeType = "USED_IN_AI"
	EdgeTypeAppliesToBO    EdgeType = "APPLIES_TO_BO"
	EdgeTypeMasksTerm      EdgeType = "MASKS_TERM"
	EdgeTypeFiltersOnTerm  EdgeType = "FILTERS_ON_TERM"
	EdgeTypeDependsOn      EdgeType = "DEPENDS_ON"
)

// ImpactNode represents a node in the impact analysis graph
type ImpactNode struct {
	ID         string                 `json:"id"`
	Type       NodeType               `json:"type"`
	Label      string                 `json:"label"`
	Properties map[string]interface{} `json:"properties"`
}

// ImpactEdge represents an edge in the impact analysis graph
type ImpactEdge struct {
	From       string                 `json:"source"`
	To         string                 `json:"target"`
	Type       EdgeType               `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// ImpactGraph represents the complete impact analysis result
type ImpactGraph struct {
	Nodes []ImpactNode `json:"nodes"`
	Edges []ImpactEdge `json:"edges"`
}

// ImpactSummary provides textual summary of impact analysis
type ImpactSummary struct {
	TotalNodes        int                     `json:"totalNodes"`
	NodesByType       map[NodeType]int        `json:"nodesByType"`
	AffectedArtifacts map[string][]ImpactNode `json:"affectedArtifacts"`
	Explanation       string                  `json:"explanation"`
	Recommendations   []string                `json:"recommendations,omitempty"`
}

// GraphSchema defines the schema for dynamic traversal
type GraphSchema struct {
	Nodes map[NodeType]NodeSchema `json:"nodes"`
}

// NodeSchema defines allowed edges for each node type
type NodeSchema struct {
	OutgoingEdges []EdgeType `json:"outgoingEdges"`
	IncomingEdges []EdgeType `json:"incomingEdges"`
}

// ImpactService provides impact analysis capabilities
type ImpactService struct {
	db          *sqlx.DB
	lineageRepo lineage.LineageRepository
	graphSchema GraphSchema
}

// NewImpactService creates a new impact analysis service
func NewImpactService(db *sqlx.DB) *ImpactService {
	return &ImpactService{
		db:          db,
		lineageRepo: lineage.NewDBLineageRepository(db),
		graphSchema: GraphSchema{
			Nodes: map[NodeType]NodeSchema{
				NodeTypeBusinessObject: {
					OutgoingEdges: []EdgeType{
						EdgeTypeHasField,
						EdgeTypeExposedViaAPI,
						EdgeTypeUsedInBI,
						EdgeTypeUsedInAI,
					},
					IncomingEdges: []EdgeType{
						EdgeTypeAppliesToBO,
					},
				},
				NodeTypeBoField: {
					OutgoingEdges: []EdgeType{
						EdgeTypeBackedByTerm,
						EdgeTypeBackedByCalc,
					},
					IncomingEdges: []EdgeType{
						EdgeTypeHasField,
					},
				},
				NodeTypeSemanticTerm: {
					OutgoingEdges: []EdgeType{
						EdgeTypeMappedToColumn,
					},
					IncomingEdges: []EdgeType{
						EdgeTypeBackedByTerm,
						EdgeTypeMasksTerm,
						EdgeTypeFiltersOnTerm,
					},
				},
				NodeTypeDBColumn: {
					OutgoingEdges: []EdgeType{},
					IncomingEdges: []EdgeType{
						EdgeTypeMappedToColumn,
					},
				},
				NodeTypeCalculationTerm: {
					OutgoingEdges: []EdgeType{
						EdgeTypeDependsOn,
					},
					IncomingEdges: []EdgeType{
						EdgeTypeBackedByCalc,
					},
				},
			},
		},
	}
}

// GetImpactGraph retrieves impact analysis graph for a given node
func (s *ImpactService) GetImpactGraph(ctx context.Context, nodeID string, nodeType NodeType, depth int) (*ImpactGraph, error) {
	// Use the existing lineage functions to get downstream/upstream graphs
	// We use bidirectional to provide full context (lineage + impact)
	graph, err := s.lineageRepo.FindBiDirectionalGraph(ctx, nodeID, depth)
	if err != nil {
		return nil, fmt.Errorf("failed to get downstream graph: %w", err)
	}

	// Convert lineage graph to impact graph
	impactGraph := &ImpactGraph{
		Nodes: make([]ImpactNode, 0, len(graph.Nodes)),
		Edges: make([]ImpactEdge, 0, len(graph.Edges)),
	}

	// Convert nodes
	for _, node := range graph.Nodes {
		var props map[string]interface{}
		if len(node.Metadata) > 0 {
			json.Unmarshal(node.Metadata, &props)
		}
		if props == nil {
			props = make(map[string]interface{})
		}

		impactGraph.Nodes = append(impactGraph.Nodes, ImpactNode{
			ID:         node.ID,
			Type:       NodeType(node.Type),
			Label:      node.Name,
			Properties: props,
		})
	}

	// Convert edges
	for _, edge := range graph.Edges {
		var props map[string]interface{}
		if len(edge.Metadata) > 0 {
			json.Unmarshal(edge.Metadata, &props)
		}

		impactGraph.Edges = append(impactGraph.Edges, ImpactEdge{
			From:       edge.FromID,
			To:         edge.ToID,
			Type:       EdgeType(edge.Type),
			Properties: props,
		})
	}

	return impactGraph, nil
}

// GetImpactSummary generates a textual summary of impact analysis
func (s *ImpactService) GetImpactSummary(ctx context.Context, graph *ImpactGraph, rootID string) (*ImpactSummary, error) {
	summary := &ImpactSummary{
		TotalNodes:        0,
		NodesByType:       make(map[NodeType]int),
		AffectedArtifacts: make(map[string][]ImpactNode),
	}

	// Build adjacency list for downstream traversal
	adj := make(map[string][]string)
	for _, edge := range graph.Edges {
		adj[edge.From] = append(adj[edge.From], edge.To)
	}

	// BFS to find strictly downstream nodes
	downstream := make(map[string]bool)
	visited := make(map[string]bool)
	queue := []string{rootID}
	visited[rootID] = true

	head := 0
	for head < len(queue) {
		curr := queue[head]
		head++

		for _, neighbor := range adj[curr] {
			if !visited[neighbor] {
				visited[neighbor] = true
				downstream[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	// Count nodes by type and categorize artifacts
	for _, node := range graph.Nodes {
		// Only count downstream nodes
		if downstream[node.ID] {
			summary.TotalNodes++
			summary.NodesByType[node.Type]++

			// Categorize impacted artifacts
			switch node.Type {
			case NodeTypeAPIEndpoint:
				summary.AffectedArtifacts["APIs"] = append(summary.AffectedArtifacts["APIs"], node)
			case NodeTypeBIArtifact:
				summary.AffectedArtifacts["BI Reports"] = append(summary.AffectedArtifacts["BI Reports"], node)
			case NodeTypeAIArtifact:
				summary.AffectedArtifacts["AI Artifacts"] = append(summary.AffectedArtifacts["AI Artifacts"], node)
			case NodeTypeAccessRule:
				summary.AffectedArtifacts["Security Rules"] = append(summary.AffectedArtifacts["Security Rules"], node)
			case NodeTypeBusinessObject:
				summary.AffectedArtifacts["Business Objects"] = append(summary.AffectedArtifacts["Business Objects"], node)
			case "column": // Hardcoded string for now if NodeType not defined as constant
				summary.AffectedArtifacts["Data Columns"] = append(summary.AffectedArtifacts["Data Columns"], node)
			}
		}
	}

	// Generate explanation
	summary.Explanation = s.generateExplanation(summary)

	// Generate recommendations
	summary.Recommendations = s.generateRecommendations(summary)

	return summary, nil
}

// generateExplanation creates a human-readable explanation
func (s *ImpactService) generateExplanation(summary *ImpactSummary) string {
	explanation := fmt.Sprintf("This change impacts %d nodes across the system. ", summary.TotalNodes)

	if len(summary.AffectedArtifacts["APIs"]) > 0 {
		explanation += fmt.Sprintf("%d API endpoints are affected. ", len(summary.AffectedArtifacts["APIs"]))
	}
	if len(summary.AffectedArtifacts["BI Reports"]) > 0 {
		explanation += fmt.Sprintf("%d BI reports/dashboards are affected. ", len(summary.AffectedArtifacts["BI Reports"]))
	}
	if len(summary.AffectedArtifacts["AI Artifacts"]) > 0 {
		explanation += fmt.Sprintf("%d AI artifacts are affected. ", len(summary.AffectedArtifacts["AI Artifacts"]))
	}
	if len(summary.AffectedArtifacts["Security Rules"]) > 0 {
		explanation += fmt.Sprintf("%d security rules are affected. ", len(summary.AffectedArtifacts["Security Rules"]))
	}

	return explanation
}

// generateRecommendations provides actionable recommendations
func (s *ImpactService) generateRecommendations(summary *ImpactSummary) []string {
	recommendations := []string{}

	if summary.TotalNodes > 10 {
		recommendations = append(recommendations, "This is a high-impact change. Consider testing in a staging environment first.")
	}

	if len(summary.AffectedArtifacts["APIs"]) > 0 {
		recommendations = append(recommendations, "Review and update API documentation for affected endpoints.")
	}

	if len(summary.AffectedArtifacts["BI Reports"]) > 0 {
		recommendations = append(recommendations, "Notify BI team and validate affected reports after deployment.")
	}

	if len(summary.AffectedArtifacts["Security Rules"]) > 0 {
		recommendations = append(recommendations, "Security rules are affected. Coordinate with security team before proceeding.")
	}

	return recommendations
}
