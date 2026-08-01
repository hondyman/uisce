package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
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
		r.Post("/merge", h.HandleMergeToGoldenRecord)
	})
}

type MergeSourcesRequest struct {
	EntityType string                  `json:"entity_type"`
	Sources    []mdm.SourcePayload     `json:"sources"`
	Rules      map[string]mdm.FieldRule `json:"rules"`
}

type MergeSourcesResponse struct {
	GoldenRecord map[string]any `json:"golden_record"`
}

func (h *SurvivorshipHandler) HandleMergeToGoldenRecord(w http.ResponseWriter, r *http.Request) {
	var req MergeSourcesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(req.Sources) == 0 {
		http.Error(w, "sources cannot be empty", http.StatusBadRequest)
		return
	}
	if req.Rules == nil {
		req.Rules = make(map[string]mdm.FieldRule)
	}
	ctx := r.Context()
	golden, err := h.engine.MergeToGoldenRecord(ctx, req.Sources, req.Rules, time.Now())
	if err != nil {
		http.Error(w, "merge failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(MergeSourcesResponse{GoldenRecord: golden})
}
