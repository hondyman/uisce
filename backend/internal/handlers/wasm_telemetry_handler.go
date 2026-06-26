// handlers/observability_rpc_handler.go
package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/mux"
	"github.com/hondyman/semlayer/backend/internal/observability"
)

// WasmTelemetryHandler defines HTTP endpoints for the telemetry/observability domain
type WasmTelemetryHandler struct {
	repo observability.Repository
}

// NewWasmTelemetryHandler creates a new WasmTelemetryHandler
func NewWasmTelemetryHandler(repo observability.Repository) *WasmTelemetryHandler {
	return &WasmTelemetryHandler{repo: repo}
}

// RegisterRoutes registers the observability endpoints on the given router
func (h *WasmTelemetryHandler) RegisterRoutes(r *mux.Router) {
	// ETL Runs
	r.HandleFunc("/etl-runs", h.ListETLRuns).Methods("GET")
	r.HandleFunc("/etl-runs/{etl_run_id}", h.GetETLRun).Methods("GET")

	// WASM Versions
	r.HandleFunc("/wasm-versions", h.ListWASMVersions).Methods("GET")
	r.HandleFunc("/wasm-versions/{id}/activate", h.ActivateWASMVersion).Methods("POST")

	// Lineage Explorers
	r.HandleFunc("/rules/{rule_id}/lineage", h.GetRuleLineage).Methods("GET")
	r.HandleFunc("/scenarios/{scenario_id}/lineage", h.GetScenarioLineage).Methods("GET")
}

// ListETLRuns: GET /api/v1/telemetry/etl-runs
func (h *WasmTelemetryHandler) ListETLRuns(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	status := r.URL.Query().Get("status")
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	limitStr := r.URL.Query().Get("limit")

	limit := int32(100)
	if limitStr != "" {
		if parsed, err := strconv.ParseInt(limitStr, 10, 32); err == nil && parsed > 0 {
			limit = int32(parsed)
		}
	}

	runs, err := h.repo.ListETLRuns(r.Context(), tenantID, status, from, to, limit)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"runs": runs})
}

// GetETLRun: GET /api/v1/telemetry/etl-runs/{etl_run_id}
func (h *WasmTelemetryHandler) GetETLRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["etl_run_id"]

	run, err := h.repo.GetETLRun(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, fmt.Sprintf("ETL Run %s not found: %v", id, err))
		return
	}

	respondWithJSON(w, http.StatusOK, run)
}

// ListWASMVersions: GET /api/v1/telemetry/wasm-versions?module_name=risk-compliance-engine
func (h *WasmTelemetryHandler) ListWASMVersions(w http.ResponseWriter, r *http.Request) {
	module := r.URL.Query().Get("module_name")

	versions, err := h.repo.ListWASMVersions(r.Context(), module)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{"versions": versions})
}

// ActivateWASMVersion: POST /api/v1/telemetry/wasm-versions/{id}/activate
func (h *WasmTelemetryHandler) ActivateWASMVersion(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	err := h.repo.ActivateWASMVersion(r.Context(), id)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]string{"status": "activated"})
}

// GetRuleLineage: GET /api/v1/telemetry/rules/{rule_id}/lineage
func (h *WasmTelemetryHandler) GetRuleLineage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	ruleID := vars["rule_id"]
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	portfolio := r.URL.Query().Get("portfolio_id")

	rows, err := h.repo.GetRuleLineage(r.Context(), ruleID, from, to, portfolio)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"rule_id":     ruleID,
		"evaluations": rows,
	})
}

// GetScenarioLineage: GET /api/v1/telemetry/scenarios/{scenario_id}/lineage
func (h *WasmTelemetryHandler) GetScenarioLineage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	scenarioID := vars["scenario_id"]
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	portfolio := r.URL.Query().Get("portfolio_id")

	rows, err := h.repo.GetScenarioLineage(r.Context(), scenarioID, from, to, portfolio)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, map[string]interface{}{
		"scenario_id": scenarioID,
		"results":     rows,
	})
}
