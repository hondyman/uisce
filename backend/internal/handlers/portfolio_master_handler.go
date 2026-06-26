package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/portfoliomaster"
)

// PortfolioMasterHandler exposes three endpoints for the Portfolio Master Gold Copy:
//
//	GET  /api/v1/portfolio                              — list golden records
//	GET  /api/v1/portfolio/sources                      — list source registry entries
//	POST /api/v1/portfolio/sources/preferences/{prefId}/simulate — impact simulation
type PortfolioMasterHandler struct {
	svc *portfoliomaster.Service
}

func NewPortfolioMasterHandler(svc *portfoliomaster.Service) *PortfolioMasterHandler {
	return &PortfolioMasterHandler{svc: svc}
}

// RegisterRoutes mounts all portfolio master endpoints.
func (h *PortfolioMasterHandler) RegisterRoutes(r chi.Router) {
	r.Route("/portfolio", func(r chi.Router) {
		r.Get("/", h.ListGoldenRecords)
		r.Route("/sources", func(r chi.Router) {
			r.Get("/", h.ListSourceRegistry)
			r.Post("/preferences/{prefId}/simulate", h.SimulateSourceChange)
		})
	})
}

// ─── Handlers ─────────────────────────────────────────────────────────────────

// ListGoldenRecords returns current portfolio golden records.
// Query params: account_type (optional), as_of (optional RFC3339 date)
func (h *PortfolioMasterHandler) ListGoldenRecords(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	accountType := q.Get("account_type")
	asOf := time.Now()
	if s := q.Get("as_of"); s != "" {
		if t, err := time.Parse("2006-01-02", s); err == nil {
			asOf = t
		}
	}
	records, err := h.svc.GetPortfolioGolden(r.Context(), mustTenantID(r), accountType, asOf)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"portfolios":   records,
		"total":        len(records),
		"account_type": accountType,
		"as_of":        asOf.Format("2006-01-02"),
	})
}

// ListSourceRegistry returns all active source registry entries.
func (h *PortfolioMasterHandler) ListSourceRegistry(w http.ResponseWriter, r *http.Request) {
	sources, err := h.svc.GetSourceRegistry(r.Context(), mustTenantID(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"sources": sources,
		"total":   len(sources),
	})
}

// SimulateSourceChange runs a what-if simulation for a source preference change.
// POST body: { "field": "price"|"quantity", "account_type": "...", "proposed_source": "Bloomberg",
//
//	"old_confidence": 85, "new_confidence": 95 }
func (h *PortfolioMasterHandler) SimulateSourceChange(w http.ResponseWriter, r *http.Request) {
	prefID, err := uuid.Parse(chi.URLParam(r, "prefId"))
	if err != nil {
		http.Error(w, "invalid prefId", http.StatusBadRequest)
		return
	}

	var body struct {
		Field          string `json:"field"` // price | quantity
		AccountType    string `json:"account_type"`
		ProposedSource string `json:"proposed_source"`
		OldConfidence  int    `json:"old_confidence"`
		NewConfidence  int    `json:"new_confidence"`
		AsOfDate       string `json:"as_of_date,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	if body.Field == "" {
		body.Field = "price"
	}

	result, err := h.svc.SimulateSourceChange(
		r.Context(),
		mustTenantID(r),
		prefID,
		body.Field,
		body.AccountType,
		body.ProposedSource,
		body.OldConfidence,
		body.NewConfidence,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, result)
}
