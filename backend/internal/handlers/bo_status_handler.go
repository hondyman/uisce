package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/analytics"
)

// BOStatusHandler handles BO status requests
type BOStatusHandler struct {
	Service *analytics.BOStatusService
}

// NewBOStatusHandler creates a new status handler
func NewBOStatusHandler(service *analytics.BOStatusService) *BOStatusHandler {
	return &BOStatusHandler{Service: service}
}

// RegisterRoutes registers status routes
func (h *BOStatusHandler) RegisterRoutes(r chi.Router) {
	r.Get("/api/bo/{boId}/status", h.GetBOStatus)
}

// GetBOStatus returns the current status of a BO
// GET /api/bo/:boId/status
func (h *BOStatusHandler) GetBOStatus(w http.ResponseWriter, r *http.Request) {
	boID := chi.URLParam(r, "boId")
	if boID == "" {
		http.Error(w, "boId is required", http.StatusBadRequest)
		return
	}

	status, err := h.Service.GetBOStatus(boID)
	if err != nil {
		http.Error(w, "Failed to get BO status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
