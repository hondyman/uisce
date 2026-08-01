package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/metadata"
)

type LookThroughSQLHandler struct{}

func NewLookThroughSQLHandler() *LookThroughSQLHandler {
	return &LookThroughSQLHandler{}
}

func (h *LookThroughSQLHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/v1/compliance", func(r chi.Router) {
		r.Post("/lookthrough-sql", h.HandleBuildLookThroughSQL)
	})
}

type CompileLookThroughSQLRequest struct {
	TenantID       string `json:"tenant_id"`
	PortfolioID    string `json:"portfolio_id"`
	TargetIssuerID string `json:"target_issuer_id"`
	WatermarkDate  string `json:"watermark_date"`
}

func (h *LookThroughSQLHandler) HandleBuildLookThroughSQL(w http.ResponseWriter, r *http.Request) {
	var req CompileLookThroughSQLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	tenantUUID, err := uuid.Parse(req.TenantID)
	if err != nil {
		http.Error(w, "invalid tenant_id: "+err.Error(), http.StatusBadRequest)
		return
	}
	cfg := metadata.LookThroughQueryConfig{
		TenantID:       tenantUUID,
		PortfolioID:    req.PortfolioID,
		TargetIssuerID: req.TargetIssuerID,
		WatermarkDate:  req.WatermarkDate,
	}
	sql, args, err := metadata.BuildLookThroughExposureSQL(cfg)
	if err != nil {
		http.Error(w, "compile failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"sql": sql, "args": args})
}
