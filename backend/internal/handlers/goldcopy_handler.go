package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/goldcopy"
)

// GoldCopyHandler exposes Portfolio Master gold copy endpoints.
//
//	GET  /api/v1/goldcopy/portfolio                    — list current portfolio master records
//	GET  /api/v1/goldcopy/portfolio/{portfolioId}      — single record with lineage
//	POST /api/v1/goldcopy/portfolio/build              — trigger gold copy build
type GoldCopyHandler struct {
	engine *goldcopy.Engine
	repo   *goldcopy.Repository
}

func NewGoldCopyHandler(engine *goldcopy.Engine, repo *goldcopy.Repository) *GoldCopyHandler {
	return &GoldCopyHandler{engine: engine, repo: repo}
}

// RegisterRoutes mounts all gold copy endpoints under /api/v1/goldcopy.
func (h *GoldCopyHandler) RegisterRoutes(r chi.Router) {
	r.Route("/v1/goldcopy", func(r chi.Router) {
		r.Route("/portfolio", func(r chi.Router) {
			r.Get("/", h.ListPortfolioMasters)
			r.Post("/build", h.BuildGoldCopy)
			r.Get("/{portfolioId}", h.GetPortfolioMaster)
			r.Get("/{portfolioId}/lineage", h.GetPortfolioLineage)
		})
	})
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// ListPortfolioMasters returns current portfolio master gold copies.
// Query params: portfolio_type (optional)
func (h *GoldCopyHandler) ListPortfolioMasters(w http.ResponseWriter, r *http.Request) {
	tenantID := mustTenantID(r)
	portfolioType := r.URL.Query().Get("portfolio_type")

	records, err := h.repo.ListCurrentPortfolioMasters(r.Context(), tenantID, portfolioType)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if records == nil {
		records = []*goldcopy.PortfolioMasterRecord{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"portfolios":     records,
		"total":          len(records),
		"portfolio_type": portfolioType,
		"as_of":          time.Now().Format("2006-01-02T15:04:05Z"),
	})
}

// GetPortfolioMaster returns a single portfolio master record.
// Includes the full lineage for that record.
func (h *GoldCopyHandler) GetPortfolioMaster(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioId")
	tenantID := mustTenantID(r)

	rec, err := h.repo.GetCurrentPortfolioMaster(r.Context(), tenantID, portfolioID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rec == nil {
		http.Error(w, "portfolio not found", http.StatusNotFound)
		return
	}

	lineage, err := h.repo.GetLineageForEntity(r.Context(), tenantID, rec.ID)
	if err != nil {
		lineage = nil // non-fatal
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"portfolio": rec,
		"lineage":   lineage,
	})
}

// GetPortfolioLineage returns the lineage for a portfolio master record.
func (h *GoldCopyHandler) GetPortfolioLineage(w http.ResponseWriter, r *http.Request) {
	portfolioID := chi.URLParam(r, "portfolioId")
	tenantID := mustTenantID(r)

	rec, err := h.repo.GetCurrentPortfolioMaster(r.Context(), tenantID, portfolioID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if rec == nil {
		http.Error(w, "portfolio not found", http.StatusNotFound)
		return
	}

	lineage, err := h.repo.GetLineageForEntity(r.Context(), tenantID, rec.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"portfolio_id": portfolioID,
		"entity_id":    rec.ID,
		"lineage":      lineage,
	})
}

// BuildGoldCopy triggers a gold copy build for a portfolio cluster.
// POST body:
//
//	{
//	  "portfolio_id": "PF001",
//	  "raw_records": [
//	    { "portfolio_id": "PF001", "source_system": "Bloomberg", "effective_date": "2026-02-21",
//	      "quality_score": 95, "fields": { "portfolio_name": "Acme Growth Fund", ... } }
//	  ]
//	}
func (h *GoldCopyHandler) BuildGoldCopy(w http.ResponseWriter, r *http.Request) {
	tenantID := mustTenantID(r)

	var body struct {
		PortfolioID string                         `json:"portfolio_id"`
		RawRecords  []*goldcopy.RawPortfolioRecord `json:"raw_records"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(body.RawRecords) == 0 {
		http.Error(w, "raw_records must not be empty", http.StatusBadRequest)
		return
	}

	result, err := h.engine.BuildPortfolioGoldCopy(r.Context(), tenantID, body.RawRecords)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	status := http.StatusOK
	if !result.Success {
		status = http.StatusUnprocessableEntity
	}
	writeJSON(w, status, result)
}

// ─── Verify mustTenantID and writeJSON exist in the handlers package ──────────
// Both are defined in source_preference_handler.go (mustTenantID) and
// portfolio_master_handler.go (writeJSON) — no redeclaration needed.

// mustTenantIDGC safely extracts a tenant ID from context; falls back to a
// deterministic nil-safe UUID so the handler doesn't panic in dev/test.
func mustTenantIDGC(r *http.Request) uuid.UUID {
	return mustTenantID(r) // delegate to existing helper in the handlers package
}
