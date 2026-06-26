package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/lineage"
)

// LineageHandler handles lineage requests
type LineageHandler struct {
	repo lineage.LineageRepository
}

// NewLineageHandler creates a new handler
func NewLineageHandler(repo lineage.LineageRepository) *LineageHandler {
	return &LineageHandler{
		repo: repo,
	}
}

// ResponseNode represents a node in API response format (compatible with frontend)
type ResponseNode struct {
	ID         string                 `json:"id"`
	Type       string                 `json:"type"`
	Label      string                 `json:"label"`
	Properties map[string]interface{} `json:"properties"`
}

// ResponseEdge represents an edge in API response format (compatible with frontend)
type ResponseEdge struct {
	ID         string                 `json:"id,omitempty"`
	Source     string                 `json:"source"`
	Target     string                 `json:"target"`
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}

// ResponseGraph represents the graph in frontend-compatible format
type ResponseGraph struct {
	Nodes []ResponseNode `json:"nodes"`
	Edges []ResponseEdge `json:"edges"`
}

// convertToResponseGraph converts lineage.Graph to frontend-compatible format
func convertToResponseGraph(graph *lineage.Graph) *ResponseGraph {
	resp := &ResponseGraph{
		Nodes: make([]ResponseNode, 0, len(graph.Nodes)),
		Edges: make([]ResponseEdge, 0, len(graph.Edges)),
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

		resp.Nodes = append(resp.Nodes, ResponseNode{
			ID:         node.ID,
			Type:       string(node.Type),
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

// RegisterRoutes registers lineage routes
func (h *LineageHandler) RegisterRoutes(r chi.Router) {
	r.Route("/lineage", func(r chi.Router) {
		r.Get("/node/{id}/graph", h.GetDependencyGraph)
		r.Get("/node/{id}/impact", h.GetImpactAnalysis)
		r.Get("/dual", h.GetDualLineage)
	})
}

// GetDependencyGraph returns the upstream dependencies (Lineage/Provenance)
func (h *LineageHandler) GetDependencyGraph(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	depth, _ := strconv.Atoi(r.URL.Query().Get("depth"))
	if depth == 0 {
		depth = 3
	}

	ctx := r.Context()

	graph, err := h.repo.FindBiDirectionalGraph(ctx, id, depth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to frontend-compatible format
	responseGraph := convertToResponseGraph(graph)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseGraph)
}

// GetImpactAnalysis returns downstream impact
func (h *LineageHandler) GetImpactAnalysis(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	depth, _ := strconv.Atoi(r.URL.Query().Get("depth"))
	if depth == 0 {
		depth = 5
	}

	ctx := r.Context()

	graph, err := h.repo.FindDownstreamGraph(ctx, id, depth)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to frontend-compatible format
	responseGraph := convertToResponseGraph(graph)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(responseGraph)
}

// GetDualLineage handles the combined technical and semantic lineage request
func (h *LineageHandler) GetDualLineage(w http.ResponseWriter, r *http.Request) {
	datasourceID := r.URL.Query().Get("datasourceId")
	if datasourceID == "" {
		datasourceID = r.URL.Query().Get("datasource_id")
	}
	assetID := r.URL.Query().Get("asset_id")

	ctx := r.Context()

	var graph *lineage.Graph
	var err error

	if datasourceID != "" {
		graph, err = h.repo.FindGraphByDatasource(ctx, datasourceID)
	} else if assetID != "" {
		// Fallback to upstream graph if only asset_id is provided
		graph, err = h.repo.FindUpstreamGraph(ctx, assetID, 5)
	} else {
		http.Error(w, "datasourceId or asset_id is required", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Transform graph to the expected DualData format
	// For simplicity, we return the same graph for both technical and semantic,
	// allowing the frontend to filter based on node types.
	response := map[string]interface{}{
		"technicalData": map[string]interface{}{
			"nodes":    graph.Nodes,
			"edges":    graph.Edges,
			"viewport": map[string]interface{}{},
			"metadata": map[string]interface{}{},
		},
		"semanticData": map[string]interface{}{
			"nodes":    graph.Nodes,
			"edges":    graph.Edges,
			"viewport": map[string]interface{}{},
			"metadata": map[string]interface{}{},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
