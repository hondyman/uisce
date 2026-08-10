package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/services"
)

// AnomalyHandler handles API requests related to data anomalies.
type AnomalyHandler struct {
	service       *services.AnomalyService
	securityDeps  SecurityContextDeps
}

// NewAnomalyHandler creates a new AnomalyHandler.
func NewAnomalyHandler(service *services.AnomalyService, securityDeps SecurityContextDeps) *AnomalyHandler {
	return &AnomalyHandler{service: service, securityDeps: securityDeps}
}

// RegisterRoutes registers the routes for AnomalyHandler.
func (h *AnomalyHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/anomalies", h.HandleListAnomalies)
}

// HandleListAnomalies retrieves a list of detected anomalies.
func (h *AnomalyHandler) HandleListAnomalies(w http.ResponseWriter, r *http.Request) {
	secCtx, ctx, err := SecurityContextFromRequest(r, "", "", h.securityDeps)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	datasourceID := secCtx.DatasourceID
	metric := r.URL.Query().Get("metric")

	anomalies, err := h.service.ListAnomalies(ctx, datasourceID, metric)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve anomalies"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(anomalies)
}
