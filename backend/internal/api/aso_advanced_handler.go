package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/aso"
)

// ASOAdvancedHandler handles Phase 8 advanced ASO endpoints
type ASOAdvancedHandler struct {
	experimentService aso.ExperimentService
	simulationService aso.SimulationService
	mlService         aso.MLScoringService
}

// NewASOAdvancedHandler creates a new advanced handler
func NewASOAdvancedHandler(
	experimentService aso.ExperimentService,
	simulationService aso.SimulationService,
	mlService aso.MLScoringService,
) *ASOAdvancedHandler {
	return &ASOAdvancedHandler{
		experimentService: experimentService,
		simulationService: simulationService,
		mlService:         mlService,
	}
}

// RegisterASOAdvancedRoutes registers advanced ASO routes
func RegisterASOAdvancedRoutes(r chi.Router, h *ASOAdvancedHandler) {
	r.Route("/aso", func(r chi.Router) {
		// A/B Experiments
		r.Get("/experiments", h.ListExperiments)
		r.Post("/experiments", h.CreateExperiment)
		r.Get("/experiments/{id}", h.GetExperiment)
		r.Post("/experiments/{id}/start", h.StartExperiment)
		r.Post("/experiments/{id}/stop", h.StopExperiment)
		r.Post("/experiments/{id}/evaluate", h.EvaluateExperiment)

		// Simulation
		r.Post("/simulate", h.RunSimulation)
		r.Post("/simulate/optimization/{id}", h.SimulateOptimization)
		r.Post("/simulate/rollback/{id}", h.SimulateRollback)

		// ML Scoring
		r.Get("/ml/score/{id}", h.GetMLScore)
		r.Get("/ml/stats", h.GetMLStats)
		r.Post("/ml/retrain", h.RetrainModel)
	})
}

// ============================================================================
// A/B Experiment Endpoints
// ============================================================================

// ListExperiments lists all experiments
func (h *ASOAdvancedHandler) ListExperiments(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var status *aso.ASOExperimentStatus
	if s := r.URL.Query().Get("status"); s != "" {
		st := aso.ASOExperimentStatus(s)
		status = &st
	}

	experiments, err := h.experimentService.ListExperiments(ctx, status, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(experiments)
}

// CreateExperimentRequest is the request for creating an experiment
type CreateExperimentRequest struct {
	OptimizationID   string  `json:"optimization_id"`
	TrafficPercent   float64 `json:"traffic_percent,omitempty"`
	MinDurationHours int     `json:"min_duration_hours,omitempty"`
	MaxDurationDays  int     `json:"max_duration_days,omitempty"`
	MinSampleSize    int     `json:"min_sample_size,omitempty"`
	AutoPromote      bool    `json:"auto_promote"`
}

// CreateExperiment creates a new A/B experiment
func (h *ASOAdvancedHandler) CreateExperiment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req CreateExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	optID, err := uuid.Parse(req.OptimizationID)
	if err != nil {
		http.Error(w, "Invalid optimization ID", http.StatusBadRequest)
		return
	}

	creator := r.Header.Get("X-User-ID")
	if creator == "" {
		creator = "api_user"
	}

	config := aso.ASOExperimentConfig{
		AutoPromote: req.AutoPromote,
	}

	experiment, err := h.experimentService.CreateExperiment(ctx, optID, config, creator)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(experiment)
}

// GetExperiment retrieves an experiment
func (h *ASOAdvancedHandler) GetExperiment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid experiment ID", http.StatusBadRequest)
		return
	}

	experiment, err := h.experimentService.GetExperiment(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(experiment)
}

// StartExperiment starts an experiment
func (h *ASOAdvancedHandler) StartExperiment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid experiment ID", http.StatusBadRequest)
		return
	}

	if err := h.experimentService.StartExperiment(ctx, id); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	experiment, _ := h.experimentService.GetExperiment(ctx, id)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(experiment)
}

// StopExperiment stops an experiment
func (h *ASOAdvancedHandler) StopExperiment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid experiment ID", http.StatusBadRequest)
		return
	}

	if err := h.experimentService.StopExperiment(ctx, id, "Manual stop"); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// EvaluateExperiment evaluates experiment results
func (h *ASOAdvancedHandler) EvaluateExperiment(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid experiment ID", http.StatusBadRequest)
		return
	}

	metrics, err := h.experimentService.EvaluateExperiment(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// ============================================================================
// Simulation Endpoints
// ============================================================================

// RunSimulation runs a what-if simulation
func (h *ASOAdvancedHandler) RunSimulation(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var req aso.SimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.simulationService.Simulate(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// SimulateOptimization simulates applying an optimization
func (h *ASOAdvancedHandler) SimulateOptimization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid optimization ID", http.StatusBadRequest)
		return
	}

	result, err := h.simulationService.SimulateOptimization(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// SimulateRollback simulates rolling back an optimization
func (h *ASOAdvancedHandler) SimulateRollback(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid optimization ID", http.StatusBadRequest)
		return
	}

	result, err := h.simulationService.SimulateRollback(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// ============================================================================
// ML Scoring Endpoints
// ============================================================================

// GetMLScore gets ML score for an optimization
func (h *ASOAdvancedHandler) GetMLScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")

	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Invalid optimization ID", http.StatusBadRequest)
		return
	}

	score, err := h.mlService.ScoreOptimization(ctx, id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(score)
}

// GetMLStats gets ML model statistics
func (h *ASOAdvancedHandler) GetMLStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.mlService.GetModelStats(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// RetrainModel triggers model retraining
func (h *ASOAdvancedHandler) RetrainModel(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	if err := h.mlService.RetrainModel(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "retraining_started"})
}
