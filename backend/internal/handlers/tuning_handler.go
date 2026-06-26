package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
	models "github.com/hondyman/semlayer/backend/models"
)

// TuningHandler handles API requests related to rule tuning.
type TuningHandler struct {
	service *services.TuningService
}

// NewTuningHandler creates a new TuningHandler.
func NewTuningHandler(s *services.TuningService) *TuningHandler {
	return &TuningHandler{service: s}
}

// RegisterRoutes mounts tuning routes
func (h *TuningHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/tuning", func(r chi.Router) {
		r.Get("/status", h.HandleGetTuningStatus)
		r.Post("/run", h.HandleRunTuning)
		r.Get("/changelog", h.HandleGetTuningChangelog)
		r.Post("/simulate", h.HandleSimulateTuning)
		r.Get("/performance/{rule_id}", h.HandleGetRulePerformance)
	})
}

// HandleGetTuningStatus retrieves the current configuration and metrics for all rules.
func (h *TuningHandler) HandleGetTuningStatus(w http.ResponseWriter, r *http.Request) {
	status, err := h.service.GetTuningStatus(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to get tuning status", "details": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

// HandleRunTuning executes the self-tuning algorithm.
func (h *TuningHandler) HandleRunTuning(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.RunTuning(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to run tuning", "details": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// HandleGetTuningChangelog retrieves the history of rule configuration changes.
func (h *TuningHandler) HandleGetTuningChangelog(w http.ResponseWriter, r *http.Request) {
	// Extract optional query parameters to filter the changelog.
	ruleID := r.URL.Query().Get("rule_id")
	scope := r.URL.Query().Get("scope")

	changelog, err := h.service.GetChangelog(r.Context(), ruleID, scope)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to get tuning changelog", "details": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(changelog)
}

// HandleSimulateTuning runs the tuning algorithm in dry-run mode.
func (h *TuningHandler) HandleSimulateTuning(w http.ResponseWriter, r *http.Request) {
	var req models.TuningSimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	annotations, err := h.service.SimulateTuning(r.Context(), req)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to run tuning simulation", "details": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"changes": annotations})
}

// HandleGetRulePerformance retrieves semantic and operational metrics for a single rule.
func (h *TuningHandler) HandleGetRulePerformance(w http.ResponseWriter, r *http.Request) {
	ruleID := chi.URLParam(r, "rule_id")
	if ruleID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "rule_id parameter is required"})
		return
	}

	performanceData, err := h.service.GetRulePerformance(r.Context(), ruleID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Failed to get rule performance data", "details": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(performanceData)
}
