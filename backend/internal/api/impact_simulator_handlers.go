package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/uisce/backend/internal/lineage"
)

type ImpactSimulatorHandler struct {
	simulator *lineage.ImpactSimulator
}

func NewImpactSimulatorHandler(svc *lineage.LineageService) *ImpactSimulatorHandler {
	return &ImpactSimulatorHandler{
		simulator: lineage.NewImpactSimulator(svc),
	}
}

func (h *ImpactSimulatorHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/lineage", func(r chi.Router) {
		r.Post("/simulate-impact", h.SimulateImpact)
		r.Get("/node/{id}/blast-radius", h.GetBlastRadius)
	})
}

type SimulateImpactRequest struct {
	TargetNode string `json:"target_node"`
	Action    string `json:"action"`
	Depth     int    `json:"depth,omitempty"`
}

func (h *ImpactSimulatorHandler) SimulateImpact(w http.ResponseWriter, r *http.Request) {
	tenantID := extractValidatedTenantID(w, r)
	if tenantID == "" {
		return
	}

	var req SimulateImpactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid request body: "+err.Error(), "decode_error", nil)
		return
	}

	if req.TargetNode == "" {
		writeJSONError(w, http.StatusBadRequest, "target_node is required", "missing_param", nil)
		return
	}

	if req.Action == "" {
		req.Action = "DEPRECATE_OR_MODIFY"
	}

	defaultDepth := 5
	if v := os.Getenv("IMPACT_SIMULATOR_DEFAULT_DEPTH"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			defaultDepth = parsed
		}
	}
	depth := req.Depth
	if depth == 0 {
		depth = defaultDepth
	}

	report, err := h.simulator.SimulateImpact(r.Context(), tenantID, req.TargetNode, req.Action, depth)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "impact simulation failed: "+err.Error(), "simulation_error", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

func (h *ImpactSimulatorHandler) GetBlastRadius(w http.ResponseWriter, r *http.Request) {
	tenantID := extractValidatedTenantID(w, r)
	if tenantID == "" {
		return
	}

	nodeID := chi.URLParam(r, "id")
	if nodeID == "" {
		writeJSONError(w, http.StatusBadRequest, "node id is required", "missing_param", nil)
		return
	}

	depth := 5
	if d := r.URL.Query().Get("depth"); d != "" {
		if parsed, err := strconv.Atoi(d); err == nil && parsed > 0 {
			depth = parsed
		}
	}

	report, err := h.simulator.SimulateImpact(r.Context(), tenantID, nodeID, "DEPRECATE_OR_MODIFY", depth)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "blast radius computation failed: "+err.Error(), "computation_error", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}
