package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
)

// IndexHandler handles API requests for search index management.
type IndexHandler struct {
	service *services.IndexService
}

// NewIndexHandler creates a new IndexHandler.
func NewIndexHandler(service *services.IndexService) *IndexHandler {
	return &IndexHandler{service: service}
}

// RegisterRoutes registers the routes for IndexHandler.
func (h *IndexHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/index", func(r chi.Router) {
		r.Post("/refresh", h.HandleRefreshIndex)
		r.Get("/monitor", h.HandleGetIndexMonitorSnapshot)
	})
}

// HandleRefreshIndex triggers a re-indexing process.
func (h *IndexHandler) HandleRefreshIndex(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AssetType string  `json:"asset_type"` // e.g., "query", "workbook", or empty for all
		AssetID   *string `json:"asset_id,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request payload"})
		return
	}

	// In a real app, you would check for admin privileges here.

	summary, err := h.service.RefreshAssetIndex(r.Context(), req.AssetType, req.AssetID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to refresh index", "details": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"status": "success", "summary": summary})
}

// HandleGetIndexMonitorSnapshot retrieves the index monitor dashboard data.
func (h *IndexHandler) HandleGetIndexMonitorSnapshot(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.service.GetIndexMonitorSnapshot(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to get index monitor snapshot", "details": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}
