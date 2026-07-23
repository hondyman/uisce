package ai

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
)

// DependencyHealer detects and repairs broken DAG dependencies
type DependencyHealer struct {
	logger *slog.Logger
}

// NewDependencyHealer creates a new dependency healer
func NewDependencyHealer(logger *slog.Logger) *DependencyHealer {
	return &DependencyHealer{
		logger: logger,
	}
}

// HealingRequest contains the DAG to analyze
type HealingRequest struct {
	TenantID    uuid.UUID     `json:"tenant_id"`
	DAGID       uuid.UUID     `json:"dag_id"`
	DAGName     string        `json:"dag_name"`
	Nodes       []DAGNodeInfo `json:"nodes"`
	Edges       []DAGEdgeInfo `json:"edges"`
	FailureData []NodeFailure `json:"failure_data,omitempty"`
}

// DAGNodeInfo represents a node with metadata
type DAGNodeInfo struct {
	ID            string    `json:"id"`
	JobID         uuid.UUID `json:"job_id"`
	Name          string    `json:"name"`
	AvgDurationMS int64     `json:"avg_duration_ms"`
	FailureRate   float64   `json:"failure_rate"`
}

// DAGEdgeInfo represents an edge with metadata
type DAGEdgeInfo struct {
	FromNodeID   string `json:"from_node_id"`
	ToNodeID     string `json:"to_node_id"`
	Type         string `json:"type"`
	FailureCount int    `json:"failure_count"`
}

// NodeFailure represents failure data for a node
type NodeFailure struct {
	NodeID       string `json:"node_id"`
	FailureCount int    `json:"failure_count"`
	LastError    string `json:"last_error"`
}

// HealingResult contains healing recommendations
type HealingResult struct {
	Issues          []DependencyIssue `json:"issues"`
	Recommendations []HealingAction   `json:"recommendations"`
	RiskScore       float64           `json:"risk_score"` // 0-1
	Confidence      float64           `json:"confidence"`
}

// DependencyIssue describes a detected problem
type DependencyIssue struct {
	IssueType     string   `json:"issue_type"` // missing_edge, circular, orphan, bottleneck
	Description   string   `json:"description"`
	Severity      string   `json:"severity"` // low, medium, high, critical
	AffectedNodes []string `json:"affected_nodes"`
}

// HealingAction is a recommended fix
type HealingAction struct {
	ActionType  string                 `json:"action_type"` // add_edge, remove_edge, add_retry, add_bypass, reorder
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	RiskLevel   string                 `json:"risk_level"`
	Impact      string                 `json:"impact"`
}

// AnalyzeAndHeal detects issues and generates healing recommendations
func (h *DependencyHealer) AnalyzeAndHeal(ctx context.Context, req HealingRequest) (*HealingResult, error) {
	h.logger.Info("Analyzing DAG dependencies",
		"dag_id", req.DAGID,
		"dag_name", req.DAGName,
		"nodes", len(req.Nodes),
		"edges", len(req.Edges),
	)

	result := &HealingResult{
		Confidence: 0.8,
	}

	// Build adjacency structures
	adj := h.buildAdjacency(req.Nodes, req.Edges)

	// Detect orphan nodes (no incoming or outgoing edges)
	orphans := h.detectOrphanNodes(req.Nodes, adj)
	for _, orphan := range orphans {
		result.Issues = append(result.Issues, DependencyIssue{
			IssueType:     "orphan",
			Description:   fmt.Sprintf("Node '%s' has no connections", orphan),
			Severity:      "medium",
			AffectedNodes: []string{orphan},
		})
		result.Recommendations = append(result.Recommendations, HealingAction{
			ActionType:  "add_edge",
			Description: fmt.Sprintf("Connect orphan node '%s' to the workflow", orphan),
			Parameters:  map[string]interface{}{"node_id": orphan},
			RiskLevel:   "low",
			Impact:      "Ensures all nodes execute",
		})
	}

	// Detect bottlenecks (nodes with high fan-in)
	bottlenecks := h.detectBottlenecks(adj)
	for _, bn := range bottlenecks {
		result.Issues = append(result.Issues, DependencyIssue{
			IssueType:     "bottleneck",
			Description:   fmt.Sprintf("Node '%s' is a bottleneck with many dependencies", bn),
			Severity:      "medium",
			AffectedNodes: []string{bn},
		})
		result.Recommendations = append(result.Recommendations, HealingAction{
			ActionType:  "add_bypass",
			Description: fmt.Sprintf("Consider adding bypass for bottleneck '%s'", bn),
			Parameters:  map[string]interface{}{"node_id": bn},
			RiskLevel:   "medium",
			Impact:      "Improves parallel execution",
		})
	}

	// Analyze failure patterns
	failingNodes := h.analyzeFailurePatterns(req.FailureData, req.Nodes)
	for _, fn := range failingNodes {
		result.Issues = append(result.Issues, DependencyIssue{
			IssueType:     "frequent_failure",
			Description:   fmt.Sprintf("Node '%s' has high failure rate", fn.NodeID),
			Severity:      "high",
			AffectedNodes: []string{fn.NodeID},
		})
		result.Recommendations = append(result.Recommendations, HealingAction{
			ActionType:  "add_retry",
			Description: fmt.Sprintf("Add retry policy to frequently failing node '%s'", fn.NodeID),
			Parameters:  map[string]interface{}{"node_id": fn.NodeID, "retry_count": 3},
			RiskLevel:   "low",
			Impact:      "Reduces failure propagation",
		})
	}

	// Detect missing dependencies based on execution order
	missing := h.detectMissingDependencies(req.Nodes, adj)
	for _, m := range missing {
		result.Issues = append(result.Issues, DependencyIssue{
			IssueType:     "missing_edge",
			Description:   fmt.Sprintf("Potential missing dependency: %s → %s", m.from, m.to),
			Severity:      "low",
			AffectedNodes: []string{m.from, m.to},
		})
		result.Recommendations = append(result.Recommendations, HealingAction{
			ActionType:  "add_edge",
			Description: fmt.Sprintf("Add edge from '%s' to '%s'", m.from, m.to),
			Parameters:  map[string]interface{}{"from": m.from, "to": m.to},
			RiskLevel:   "low",
			Impact:      "Ensures correct execution order",
		})
	}

	// Calculate overall risk score
	result.RiskScore = h.calculateRiskScore(result.Issues)

	h.logger.Info("Healing analysis complete",
		"issues", len(result.Issues),
		"recommendations", len(result.Recommendations),
		"risk_score", result.RiskScore,
	)

	return result, nil
}

// Adjacency tracks node connections
type Adjacency struct {
	Outgoing map[string][]string
	Incoming map[string][]string
}

// buildAdjacency constructs adjacency lists
func (h *DependencyHealer) buildAdjacency(nodes []DAGNodeInfo, edges []DAGEdgeInfo) *Adjacency {
	adj := &Adjacency{
		Outgoing: make(map[string][]string),
		Incoming: make(map[string][]string),
	}

	for _, node := range nodes {
		adj.Outgoing[node.ID] = []string{}
		adj.Incoming[node.ID] = []string{}
	}

	for _, edge := range edges {
		adj.Outgoing[edge.FromNodeID] = append(adj.Outgoing[edge.FromNodeID], edge.ToNodeID)
		adj.Incoming[edge.ToNodeID] = append(adj.Incoming[edge.ToNodeID], edge.FromNodeID)
	}

	return adj
}

// detectOrphanNodes finds nodes with no connections
func (h *DependencyHealer) detectOrphanNodes(nodes []DAGNodeInfo, adj *Adjacency) []string {
	var orphans []string
	for _, node := range nodes {
		if len(adj.Outgoing[node.ID]) == 0 && len(adj.Incoming[node.ID]) == 0 {
			orphans = append(orphans, node.ID)
		}
	}
	return orphans
}

// detectBottlenecks finds nodes with high fan-in
func (h *DependencyHealer) detectBottlenecks(adj *Adjacency) []string {
	var bottlenecks []string
	for nodeID, incoming := range adj.Incoming {
		if len(incoming) >= 3 { // 3+ incoming edges = bottleneck
			bottlenecks = append(bottlenecks, nodeID)
		}
	}
	return bottlenecks
}

// analyzeFailurePatterns identifies frequently failing nodes
func (h *DependencyHealer) analyzeFailurePatterns(failures []NodeFailure, nodes []DAGNodeInfo) []NodeFailure {
	var failing []NodeFailure
	for _, f := range failures {
		if f.FailureCount >= 3 { // 3+ failures = problematic
			failing = append(failing, f)
		}
	}
	return failing
}

// MissingEdge represents a potential missing dependency
type MissingEdge struct {
	from string
	to   string
}

// detectMissingDependencies infers missing edges from patterns
func (h *DependencyHealer) detectMissingDependencies(nodes []DAGNodeInfo, adj *Adjacency) []MissingEdge {
	// This is a simplified heuristic - in real implementation would use
	// execution timing analysis and data flow inference
	var missing []MissingEdge

	// Look for nodes that should be connected but aren't
	for i, nodeA := range nodes {
		for j, nodeB := range nodes {
			if i >= j {
				continue
			}
			// If A's name suggests it provides data B needs, suggest edge
			// (This is a simplified example)
			if containsProducer(nodeA.Name) && containsConsumer(nodeB.Name) {
				// Check if edge already exists
				hasEdge := false
				for _, target := range adj.Outgoing[nodeA.ID] {
					if target == nodeB.ID {
						hasEdge = true
						break
					}
				}
				if !hasEdge {
					missing = append(missing, MissingEdge{from: nodeA.ID, to: nodeB.ID})
				}
			}
		}
	}

	return missing
}

// Helper functions for name pattern matching
func containsProducer(name string) bool {
	// Would use more sophisticated matching in real impl
	return false
}

func containsConsumer(name string) bool {
	return false
}

// calculateRiskScore computes overall DAG risk
func (h *DependencyHealer) calculateRiskScore(issues []DependencyIssue) float64 {
	if len(issues) == 0 {
		return 0
	}

	score := 0.0
	for _, issue := range issues {
		switch issue.Severity {
		case "critical":
			score += 0.4
		case "high":
			score += 0.2
		case "medium":
			score += 0.1
		case "low":
			score += 0.05
		}
	}

	if score > 1.0 {
		score = 1.0
	}
	return score
}

// ValidateHealing tests if proposed actions would fix issues
func (h *DependencyHealer) ValidateHealing(ctx context.Context, dag HealingRequest, actions []HealingAction) (bool, []string) {
	var warnings []string
	valid := true

	// Simulate applying actions and check for new issues
	for _, action := range actions {
		switch action.ActionType {
		case "add_edge":
			// Check if this would create a cycle
			from := action.Parameters["from"].(string)
			to := action.Parameters["to"].(string)
			if h.wouldCreateCycle(dag, from, to) {
				warnings = append(warnings, fmt.Sprintf("Adding edge %s→%s would create a cycle", from, to))
				valid = false
			}
		}
	}

	return valid, warnings
}

// wouldCreateCycle checks if adding an edge would create a cycle
func (h *DependencyHealer) wouldCreateCycle(dag HealingRequest, from, to string) bool {
	// Build current adjacency with new edge
	adj := h.buildAdjacency(dag.Nodes, dag.Edges)
	adj.Outgoing[from] = append(adj.Outgoing[from], to)

	// Check if we can reach 'from' starting from 'to'
	visited := make(map[string]bool)
	var dfs func(node string) bool
	dfs = func(node string) bool {
		if node == from {
			return true
		}
		if visited[node] {
			return false
		}
		visited[node] = true
		for _, next := range adj.Outgoing[node] {
			if dfs(next) {
				return true
			}
		}
		return false
	}

	return dfs(to)
}
