package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/graphview"
	"github.com/hondyman/uisce/backend/internal/handlers"
	"github.com/hondyman/uisce/backend/internal/lineage"
)

// LineageHandler handles lineage requests
type LineageHandler struct {
	repo         lineage.LineageRepository
	securityDeps handlers.SecurityContextDeps
}

// NewLineageHandler creates a new handler
func NewLineageHandler(repo lineage.LineageRepository, securityDeps handlers.SecurityContextDeps) *LineageHandler {
	return &LineageHandler{
		repo:         repo,
		securityDeps: securityDeps,
	}
}

// ResponseNode/ResponseEdge/ResponseGraph moved to internal/graphview so every
// catalog graph producer (lineage, BO graph, view-definitions) shares one wire shape.
type ResponseNode = graphview.ResponseNode
type ResponseEdge = graphview.ResponseEdge
type ResponseGraph = graphview.ResponseGraph

// convertToResponseGraph converts lineage.Graph to frontend-compatible format
func convertToResponseGraph(graph *lineage.Graph) *ResponseGraph {
	return graphview.ConvertLineageGraph(graph)
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
	datasourceID := r.Header.Get("X-Tenant-Datasource-ID")
	if datasourceID == "" {
		secCtx, _, err := handlers.SecurityContextFromRequest(r, "", "", h.securityDeps)
		if err == nil {
			datasourceID = secCtx.DatasourceID
		}
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
