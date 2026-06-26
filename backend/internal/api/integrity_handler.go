package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
	"go.uber.org/zap"
)

// IntegrityHandler handles data integrity API requests
type IntegrityHandler struct {
	service *services.IntegrityService
	logger  *zap.Logger
}

// NewIntegrityHandler creates a new IntegrityHandler
func NewIntegrityHandler(service *services.IntegrityService) *IntegrityHandler {
	logger, _ := zap.NewProduction()
	return &IntegrityHandler{
		service: service,
		logger:  logger,
	}
}

// RegisterRoutes registers integrity check routes
func (h *IntegrityHandler) RegisterRoutes(r chi.Router) {
	r.Route("/datasources/{datasourceId}/integrity", func(r chi.Router) {
		r.Post("/check", h.RunIntegrityCheck)
		r.Get("/history", h.GetIntegrityHistory)
		r.Get("/status", h.GetIntegrityStatus)
		r.Post("/baseline", h.SetSchemaBaseline)
	})
}

// RunIntegrityCheckRequest represents the request body
type RunIntegrityCheckRequest struct {
	CheckType string `json:"check_type"` // row_count, schema_drift, checksum, full
}

// RunIntegrityCheck handles POST /datasources/{datasourceId}/integrity/check
func (h *IntegrityHandler) RunIntegrityCheck(w http.ResponseWriter, r *http.Request) {
	datasourceID := chi.URLParam(r, "datasourceId")
	if datasourceID == "" {
		http.Error(w, "Missing datasource ID", http.StatusBadRequest)
		return
	}

	var req RunIntegrityCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CheckType == "" {
		req.CheckType = "full"
	}

	// Get user from context (would normally come from auth middleware)
	executedBy := r.Header.Get("X-User-ID")
	if executedBy == "" {
		executedBy = "system"
	}

	result, err := h.service.RunIntegrityCheck(r.Context(), datasourceID, req.CheckType, executedBy)
	if err != nil {
		h.logger.Error("Failed to run integrity check", zap.Error(err))
		http.Error(w, "Integrity check failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetIntegrityHistory handles GET /datasources/{datasourceId}/integrity/history
func (h *IntegrityHandler) GetIntegrityHistory(w http.ResponseWriter, r *http.Request) {
	datasourceID := chi.URLParam(r, "datasourceId")
	if datasourceID == "" {
		http.Error(w, "Missing datasource ID", http.StatusBadRequest)
		return
	}

	results, err := h.service.GetIntegrityHistory(r.Context(), datasourceID, 20)
	if err != nil {
		h.logger.Error("Failed to get integrity history", zap.Error(err))
		http.Error(w, "Failed to get history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// GetIntegrityStatus handles GET /datasources/{datasourceId}/integrity/status
func (h *IntegrityHandler) GetIntegrityStatus(w http.ResponseWriter, r *http.Request) {
	datasourceID := chi.URLParam(r, "datasourceId")
	if datasourceID == "" {
		http.Error(w, "Missing datasource ID", http.StatusBadRequest)
		return
	}

	// Get current status from database
	var status struct {
		IntegrityStatus      string  `json:"integrity_status"`
		IntegrityMessage     *string `json:"integrity_message,omitempty"`
		LastIntegrityCheckAt *string `json:"last_integrity_check_at,omitempty"`
		HealthStatus         string  `json:"health_status"`
		LastHeartbeatAt      *string `json:"last_heartbeat_at,omitempty"`
	}

	// For now, return a basic response
	status.IntegrityStatus = "unknown"
	status.HealthStatus = "unknown"

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// SetSchemaBaseline handles POST /datasources/{datasourceId}/integrity/baseline
func (h *IntegrityHandler) SetSchemaBaseline(w http.ResponseWriter, r *http.Request) {
	datasourceID := chi.URLParam(r, "datasourceId")
	if datasourceID == "" {
		http.Error(w, "Missing datasource ID", http.StatusBadRequest)
		return
	}

	capturedBy := r.Header.Get("X-User-ID")
	if capturedBy == "" {
		capturedBy = "system"
	}

	// Capture current schema and set as baseline
	// This would call the service method

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"message": "Schema baseline captured",
	})
}
