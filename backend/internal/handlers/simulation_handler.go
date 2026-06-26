package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/internal/simulation"
	"github.com/hondyman/semlayer/backend/internal/store"
	"github.com/jmoiron/sqlx"
)

// SimulationHandler handles requests for policy simulations.
type SimulationHandler struct {
	UpgradeService *services.UpgradeService
	DB             *sqlx.DB
}

// NewSimulationHandler creates a new SimulationHandler.
func NewSimulationHandler(us *services.UpgradeService, db *sqlx.DB) *SimulationHandler {
	return &SimulationHandler{UpgradeService: us, DB: db}
}

// RegisterRoutes registers the routes for SimulationHandler.
func (h *SimulationHandler) RegisterRoutes(r chi.Router) {
	r.Post("/api/simulation", h.HandleSimulation)
}

// HandleSimulation processes a request to simulate a policy evaluation.
func (h *SimulationHandler) HandleSimulation(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PolicyID     string `json:"policy_id"`
		FromDS       string `json:"from_ds"`
		ToDS         string `json:"to_ds"`
		MigrationSQL string `json:"migration_sql"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "Invalid request body: " + err.Error()})
		return
	}

	if req.PolicyID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": "policy_id is required"})
		return
	}

	pol, err := store.LoadPolicyFromDB(r.Context(), h.DB, req.PolicyID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": fmt.Sprintf("Failed to load policy: %v", err)})
		return
	}

	simInput := simulation.Input{
		Policy:       pol,
		FromEnv:      req.FromDS,
		ToEnv:        req.ToDS,
		MigrationSQL: req.MigrationSQL,
	}

	result, err := simulation.Run(r.Context(), h.UpgradeService, simInput)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
