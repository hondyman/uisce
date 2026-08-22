package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/analytics"
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
	r.Get("/api/v1/business-objects/{boId}/status", h.GetBOStatus)
	r.Get("/api/business-objects/{boId}/status", h.GetBOStatus)
}

// GetBOStatus returns the current status of a BO
// GET /api/bo/:boId/status
func (h *BOStatusHandler) GetBOStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	boID := chi.URLParam(r, "boId")
	if boID == "" {
		boID = chi.URLParam(r, "id")
	}
	if boID == "" {
		http.Error(w, `{"error":"boId is required"}`, http.StatusBadRequest)
		return
	}

	if h.Service == nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":                "draft",
			"reason":                "",
			"is_published":          false,
			"can_publish":           true,
			"pending_terms":         []string{},
			"pending_calculations":  []string{},
			"pending_dependencies":  []interface{}{},
			"validation_errors":     []interface{}{},
			"diff_required":         false,
			"import_pending":        false,
			"last_modified":         "",
			"modified_by":           "",
			"version":               "v1",
		})
		return
	}

	status, err := h.Service.GetBOStatus(boID)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":                "draft",
			"reason":                "",
			"is_published":          false,
			"can_publish":           true,
			"pending_terms":         []string{},
			"pending_calculations":  []string{},
			"pending_dependencies":  []interface{}{},
			"validation_errors":     []interface{}{},
			"diff_required":         false,
			"import_pending":        false,
			"last_modified":         "",
			"modified_by":           "",
			"version":               "v1",
		})
		return
	}

	json.NewEncoder(w).Encode(status)
}
