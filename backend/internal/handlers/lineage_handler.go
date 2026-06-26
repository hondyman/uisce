package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// LineageHandler handles API requests for lineage and impact analysis.
// It now includes handlers for both graph-based and subject-based lineage.
type LineageHandler struct {
	service *services.LineageService
}

// LineageResponse represents the response structure for lineage endpoints
type LineageResponse struct {
	Nodes interface{} `json:"nodes,omitempty"`
	Edges interface{} `json:"edges,omitempty"`
	Error string      `json:"error,omitempty"`
}

type LineageRequest struct {
	Input struct {
		SubjectIDs []string `json:"subject_ids"`
	} `json:"input"`
}

// NewLineageHandler creates a new LineageHandler.
func NewLineageHandler(service *services.LineageService) *LineageHandler {
	return &LineageHandler{service: service}
}

// RegisterRoutes registers the routes for LineageHandler.
func (h *LineageHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/lineage", func(r chi.Router) {
		r.Get("/graph/{asset_id}", h.HandleGetLineageGraph)
		r.Get("/impact/{asset_id}", h.HandleGetImpactAnalysis)
		r.Post("/", h.HandleLineage)
		r.Get("/technical", h.HandleTechnicalLineage)
		r.Get("/semantic", h.HandleSemanticLineage)
		r.Get("/dual", h.HandleDualLineage)
	})
}

// HandleGetLineageGraph retrieves the lineage graph for an asset.
func (h *LineageHandler) HandleGetLineageGraph(w http.ResponseWriter, r *http.Request) {
	assetID := chi.URLParam(r, "asset_id")
	graph, err := h.service.GetLineageGraph(r.Context(), assetID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to retrieve lineage graph"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(graph)
}

// HandleGetImpactAnalysis retrieves the downstream impact of an asset.
func (h *LineageHandler) HandleGetImpactAnalysis(w http.ResponseWriter, r *http.Request) {
	assetID := chi.URLParam(r, "asset_id")
	impact, err := h.service.GetImpactAnalysis(r.Context(), assetID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to retrieve impact analysis"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(impact)
}

// HandleLineage handles requests for lineage based on subject IDs.
func (h *LineageHandler) HandleLineage(w http.ResponseWriter, r *http.Request) {
	var req LineageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request format"})
		return
	}

	if len(req.Input.SubjectIDs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "subject_ids are required"})
		return
	}

	nodes, edges, err := h.service.GetLineageForSubjects(r.Context(), req.Input.SubjectIDs)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": fmt.Sprintf("Failed to retrieve lineage data: %v", err)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	})
}

// HandleTechnicalLineage handles technical lineage requests
func (h *LineageHandler) HandleTechnicalLineage(w http.ResponseWriter, r *http.Request) {
	assetID := r.URL.Query().Get("asset_id")
	if assetID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "asset_id query parameter is required"})
		return
	}

	technicalData, err := h.service.GetTechnicalLineageData(assetID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": fmt.Sprintf("Failed to retrieve technical lineage data: %v", err)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"technicalData": technicalData})
}

// HandleSemanticLineage handles semantic lineage requests
func (h *LineageHandler) HandleSemanticLineage(w http.ResponseWriter, r *http.Request) {
	assetID := r.URL.Query().Get("asset_id")
	if assetID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "asset_id query parameter is required"})
		return
	}

	semanticData, err := h.service.GetSemanticLineageData(assetID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": fmt.Sprintf("Failed to retrieve semantic lineage data: %v", err)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"semanticData": semanticData})
}

// HandleDualLineage handles requests for both lineage types
func (h *LineageHandler) HandleDualLineage(w http.ResponseWriter, r *http.Request) {
	assetID := r.URL.Query().Get("asset_id")
	if assetID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "asset_id query parameter is required"})
		return
	}

	technicalData, err := h.service.GetTechnicalLineageData(assetID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": fmt.Sprintf("Failed to retrieve technical lineage data: %v", err)})
		return
	}

	semanticData, err := h.service.GetSemanticLineageData(assetID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": fmt.Sprintf("Failed to retrieve semantic lineage data: %v", err)})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"technicalData": technicalData,
		"semanticData":  semanticData,
	})
}
