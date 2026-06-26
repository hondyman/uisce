package simulation

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// Handler handles HTTP requests for simulation
type Handler struct {
	service      Service
	orchestrator *Orchestrator
	logger       *zap.Logger
}

// NewHandler creates a new simulation handler
func NewHandler(svc Service, orch *Orchestrator, logger *zap.Logger) *Handler {
	return &Handler{
		service:      svc,
		orchestrator: orch,
		logger:       logger,
	}
}

// RegisterRoutes registers endpoints
func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/scenarios", h.ListScenarios)
	r.Post("/scenarios", h.CreateScenario)
	r.Get("/scenarios/{id}", h.GetScenario)
	r.Post("/scenarios/{id}/deltas", h.AddDelta)
	r.Get("/scenarios/{id}/deltas", h.GetDeltas)
	r.Post("/scenarios/{id}/run", h.RunSimulation)
	r.Get("/scenarios/{id}/result", h.GetLatestResult)
	r.Post("/scenarios/{id}/changeset", h.CreateChangeSet)
}

// CreateScenario creates a new scenario
func (h *Handler) CreateScenario(w http.ResponseWriter, r *http.Request) {
	var req SimulationScenario
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.TenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID // Middleware should ensure this exists
	if req.BaseAsOf.IsZero() {
		req.BaseAsOf = time.Now().UTC()
	}

	if err := h.service.CreateScenario(r.Context(), &req); err != nil {
		h.logger.Error("failed to create scenario", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

// GetScenario gets a scenario
func (h *Handler) GetScenario(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		http.Error(w, "missing id", http.StatusBadRequest)
		return
	}

	scenario, err := h.service.GetScenario(r.Context(), id)
	if err != nil {
		h.logger.Error("failed to get scenario", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if scenario == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(scenario)
}

// ListScenarios lists scenarios
func (h *Handler) ListScenarios(w http.ResponseWriter, r *http.Request) {
	scenarios, err := h.service.ListScenarios(r.Context())
	if err != nil {
		h.logger.Error("failed to list scenarios", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(scenarios)
}

// AddDelta adds a delta
func (h *Handler) AddDelta(w http.ResponseWriter, r *http.Request) {
	scenarioID := chi.URLParam(r, "id")
	var req SimulationDelta
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.ScenarioID = scenarioID

	if err := h.service.AddDelta(r.Context(), &req); err != nil {
		h.logger.Error("failed to add delta", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(req)
}

// GetDeltas lists deltas for a scenario
func (h *Handler) GetDeltas(w http.ResponseWriter, r *http.Request) {
	scenarioID := chi.URLParam(r, "id")
	deltas, err := h.service.GetDeltas(r.Context(), scenarioID)
	if err != nil {
		h.logger.Error("failed to list deltas", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(deltas)
}

// RunSimulation triggers a run
func (h *Handler) RunSimulation(w http.ResponseWriter, r *http.Request) {
	scenarioID := chi.URLParam(r, "id")

	result, err := h.orchestrator.RunSimulation(r.Context(), scenarioID)
	if err != nil {
		h.logger.Error("failed to run simulation", zap.Error(err))
		http.Error(w, fmt.Sprintf("simulation failed: %v", err), http.StatusInternalServerError)
		return
	}

	if err := h.service.SaveResult(r.Context(), result); err != nil {
		h.logger.Error("failed to save simulation result", zap.Error(err))
		// We continue to return the result to the user even if persistence failed,
		// but ideally this should be a critical error.
		// For now, we log and proceed to return result so UI isn't blocked.
	}

	json.NewEncoder(w).Encode(result)
}

// GetLatestResult retrieves the most recent result for a scenario
func (h *Handler) GetLatestResult(w http.ResponseWriter, r *http.Request) {
	scenarioID := chi.URLParam(r, "id")
	result, err := h.service.GetLatestResult(r.Context(), scenarioID)
	if err != nil {
		h.logger.Error("failed to get latest result", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	if result == nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(result)
}

// CreateChangeSet converts a scenario to a changeset
func (h *Handler) CreateChangeSet(w http.ResponseWriter, r *http.Request) {
	scenarioID := chi.URLParam(r, "id")

	changesetID, err := h.service.CreateChangeSet(r.Context(), scenarioID)
	if err != nil {
		h.logger.Error("failed to create changeset", zap.Error(err))
		http.Error(w, fmt.Sprintf("failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"changeset_id": changesetID})
}
