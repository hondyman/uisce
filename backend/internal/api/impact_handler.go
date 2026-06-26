package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
	"github.com/hondyman/semlayer/backend/internal/logging"
)

// ImpactHandler handles impact analysis HTTP requests
type ImpactHandler struct {
	impactService *analytics.ImpactService
}

// NewImpactHandler creates a new impact analysis handler
func NewImpactHandler(impactService *analytics.ImpactService) *ImpactHandler {
	return &ImpactHandler{
		impactService: impactService,
	}
}

// GetImpactGraph returns the visual graph representation for a node
// GET /api/impact/graph/{nodeType}/{nodeId}?depth=3
func (h *ImpactHandler) GetImpactGraph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nodeType := analytics.NodeType(chi.URLParam(r, "nodeType"))
	nodeID := chi.URLParam(r, "nodeId")

	// Parse depth parameter (default: 3)
	depth := 3
	if depthStr := r.URL.Query().Get("depth"); depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil && d > 0 && d <= 5 {
			depth = d
		}
	}

	// Get the impact graph
	graph, err := h.impactService.GetImpactGraph(ctx, nodeID, nodeType, depth)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to get impact graph: %v", err)
		http.Error(w, "Failed to retrieve impact graph", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}

// GetImpactExplanation returns textual explanation of impact
// GET /api/impact/explain/{nodeType}/{nodeId}?depth=3
func (h *ImpactHandler) GetImpactExplanation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	nodeType := analytics.NodeType(chi.URLParam(r, "nodeType"))
	nodeID := chi.URLParam(r, "nodeId")

	// Parse depth parameter (default: 3)
	depth := 3
	if depthStr := r.URL.Query().Get("depth"); depthStr != "" {
		if d, err := strconv.Atoi(depthStr); err == nil && d > 0 && d <= 5 {
			depth = d
		}
	}

	// Get the impact graph
	graph, err := h.impactService.GetImpactGraph(ctx, nodeID, nodeType, depth)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to get impact graph: %v", err)
		http.Error(w, "Failed to retrieve impact graph", http.StatusInternalServerError)
		return
	}

	// Generate summary
	summary, err := h.impactService.GetImpactSummary(ctx, graph, nodeID)
	if err != nil {
		logging.GetLogger().Sugar().Errorf("Failed to generate impact summary: %v", err)
		http.Error(w, "Failed to generate impact summary", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// ImpactQueryRequest represents a natural language impact query
type ImpactQueryRequest struct {
	Query      string `json:"query"`
	Context    string `json:"context,omitempty"`
	EntityID   string `json:"entityId,omitempty"`
	EntityType string `json:"entityType,omitempty"`
}

// ImpactQueryResponse represents the response to an NL query
type ImpactQueryResponse struct {
	Answer          string                   `json:"answer"`
	Graph           *analytics.ImpactGraph   `json:"graph,omitempty"`
	Summary         *analytics.ImpactSummary `json:"summary,omitempty"`
	SuggestedAction string                   `json:"suggestedAction,omitempty"`
}

// QueryImpact handles natural language impact queries
// POST /api/impact/query
func (h *ImpactHandler) QueryImpact(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req ImpactQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// For now, return a placeholder response
	// In a full implementation, this would use an LLM to parse the query,
	// extract entities, determine intent, and execute the appropriate graph query
	response := ImpactQueryResponse{
		Answer: "Natural language query processing is not yet implemented. Please use the graph or explanation endpoints directly.",
	}

	// If entity ID/type provided, we can at least show the graph
	if req.EntityID != "" && req.EntityType != "" {
		graph, err := h.impactService.GetImpactGraph(ctx, req.EntityID, analytics.NodeType(req.EntityType), 3)
		if err == nil {
			response.Graph = graph
			summary, _ := h.impactService.GetImpactSummary(ctx, graph, req.EntityID)
			response.Summary = summary
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
