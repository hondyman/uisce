package api

import (
	"encoding/json"
	"net/http"

	"github.com/hondyman/semlayer/backend/internal/wealth"

	"github.com/go-chi/chi/v5"
	"github.com/shopspring/decimal"
)

// RiskManagementHandlers contains handlers for risk management features
type RiskManagementHandlers struct {
	riskService *wealth.RiskManagementService
}

// NewRiskManagementHandlers creates risk management handlers
func NewRiskManagementHandlers(riskService *wealth.RiskManagementService) *RiskManagementHandlers {
	return &RiskManagementHandlers{
		riskService: riskService,
	}
}

// RegisterRoutes registers all risk management routes
func (h *RiskManagementHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/risk", func(r chi.Router) {
		// Options overlay endpoints
		r.Post("/options/protective-put", h.BuildProtectivePut)
		r.Post("/options/collar", h.BuildCollar)

		// Tail risk endpoints
		r.Post("/tail-risk/analyze", h.AnalyzeTailRisk)

		// Drawdown endpoints
		r.Post("/drawdown/analyze", h.AnalyzeDrawdowns)
	})
}

// ==============================================================================
// OPTIONS OVERLAY HANDLERS
// ==============================================================================

type BuildProtectivePutRequest struct {
	PortfolioID          string          `json:"portfolio_id"`
	FamilyID             string          `json:"family_id"`
	UnderlyingSymbol     string          `json:"underlying_symbol"`
	PositionValue        decimal.Decimal `json:"position_value"`
	DesiredProtectionPct decimal.Decimal `json:"desired_protection_pct"` // e.g., 10 for 10%
	ExpirationMonths     int             `json:"expiration_months"`
}

func (h *RiskManagementHandlers) BuildProtectivePut(w http.ResponseWriter, r *http.Request) {
	var req BuildProtectivePutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	strategy, err := h.riskService.BuildProtectivePutStrategy(
		r.Context(),
		req.PortfolioID,
		req.FamilyID,
		req.UnderlyingSymbol,
		req.PositionValue,
		req.DesiredProtectionPct,
		req.ExpirationMonths,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategy)
}

type BuildCollarRequest struct {
	PortfolioID      string          `json:"portfolio_id"`
	FamilyID         string          `json:"family_id"`
	UnderlyingSymbol string          `json:"underlying_symbol"`
	PositionValue    decimal.Decimal `json:"position_value"`
	ProtectionStrike decimal.Decimal `json:"protection_strike"`
	CallStrike       decimal.Decimal `json:"call_strike"`
	ExpirationMonths int             `json:"expiration_months"`
}

func (h *RiskManagementHandlers) BuildCollar(w http.ResponseWriter, r *http.Request) {
	var req BuildCollarRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	strategy, err := h.riskService.BuildCollarStrategy(
		r.Context(),
		req.PortfolioID,
		req.FamilyID,
		req.UnderlyingSymbol,
		req.PositionValue,
		req.ProtectionStrike,
		req.CallStrike,
		req.ExpirationMonths,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(strategy)
}

// ==============================================================================
// TAIL RISK HANDLERS
// ==============================================================================

type AnalyzeTailRiskRequest struct {
	PortfolioID    string            `json:"portfolio_id"`
	FamilyID       string            `json:"family_id"`
	PortfolioValue decimal.Decimal   `json:"portfolio_value"`
	ReturnHistory  []decimal.Decimal `json:"return_history"` // Historical returns
}

func (h *RiskManagementHandlers) AnalyzeTailRisk(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeTailRiskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	analysis, err := h.riskService.AnalyzeTailRisk(
		r.Context(),
		req.PortfolioID,
		req.FamilyID,
		req.PortfolioValue,
		req.ReturnHistory,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// ==============================================================================
// DRAWDOWN HANDLERS
// ==============================================================================

type AnalyzeDrawdownsRequest struct {
	PortfolioID  string              `json:"portfolio_id"`
	FamilyID     string              `json:"family_id"`
	PriceHistory []wealth.PricePoint `json:"price_history"`
}

func (h *RiskManagementHandlers) AnalyzeDrawdowns(w http.ResponseWriter, r *http.Request) {
	var req AnalyzeDrawdownsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	analysis, err := h.riskService.AnalyzeDrawdowns(
		r.Context(),
		req.PortfolioID,
		req.FamilyID,
		req.PriceHistory,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}
