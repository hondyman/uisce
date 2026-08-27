package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/mdm"
)

type SurvivorshipHandler struct {
	engine *mdm.SurvivorshipEngine
}

func NewSurvivorshipHandler(e *mdm.SurvivorshipEngine) *SurvivorshipHandler {
	return &SurvivorshipHandler{engine: e}
}

func (h *SurvivorshipHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/mdm/survivorship", func(r chi.Router) {
		r.Post("/resolve-field", h.HandleResolveField)
	})
}

type ResolveFieldRequest struct {
	TenantID    string                      `json:"tenant_id"`
	EntityType  string                      `json:"entity_type"`
	FieldName   string                      `json:"field_name"`
	FieldSources []mdm.FieldSourceRecord    `json:"field_sources"`
}

type ResolveFieldResponse struct {
	FieldName      string      `json:"field_name"`
	ResolvedValue  interface{} `json:"resolved_value"`
	WinningSource  string      `json:"winning_source"`
	StrategyUsed   string      `json:"strategy_used"`
	EvaluationNote string      `json:"evaluation_note"`
}

func (h *SurvivorshipHandler) HandleResolveField(w http.ResponseWriter, r *http.Request) {
	var req ResolveFieldRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.TenantID == "" {
		http.Error(w, "tenant_id is required", http.StatusBadRequest)
		return
	}
	if req.EntityType == "" {
		http.Error(w, "entity_type is required", http.StatusBadRequest)
		return
	}
	if req.FieldName == "" {
		http.Error(w, "field_name is required", http.StatusBadRequest)
		return
	}
	if len(req.FieldSources) == 0 {
		http.Error(w, "field_sources cannot be empty", http.StatusBadRequest)
		return
	}

	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "invalid tenant_id format", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := h.engine.ResolveField(ctx, tenantUUID, req.EntityType, req.FieldName, req.FieldSources)
	if err != nil {
		http.Error(w, "resolve failed: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ResolveFieldResponse{
		FieldName:      result.FieldName,
		ResolvedValue:  result.ResolvedValue,
		WinningSource:  result.WinningSource,
		StrategyUsed:   result.StrategyUsed,
		EvaluationNote: result.EvaluationNote,
	})
}
