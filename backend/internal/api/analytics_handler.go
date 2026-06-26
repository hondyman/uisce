package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/analytics/factor"
)

type AnalyticsHandler struct {
	factorService      *factor.Service
	regressionService  *factor.RegressionService
	attributionService *factor.AttributionService
}

func NewAnalyticsHandler(
	factorService *factor.Service,
	regressionService *factor.RegressionService,
	attributionService *factor.AttributionService,
) *AnalyticsHandler {
	return &AnalyticsHandler{
		factorService:      factorService,
		regressionService:  regressionService,
		attributionService: attributionService,
	}
}

func (h *AnalyticsHandler) RegisterRoutes(r chi.Router) {
	r.Route("/analytics/factors", func(r chi.Router) {
		r.Get("/exposure/{portfolioID}", h.GetFactorExposure)
		r.Get("/attribution/{portfolioID}", h.GetAttribution)
	})
}

// GetFactorExposure calculates and returns the beta exposure of a portfolio to various factors
func (h *AnalyticsHandler) GetFactorExposure(w http.ResponseWriter, r *http.Request) {
	portfolioIDStr := chi.URLParam(r, "portfolioID")
	_, err := uuid.Parse(portfolioIDStr)
	if err != nil {
		http.Error(w, "invalid portfolioID", http.StatusBadRequest)
		return
	}

	// Mock data for MVP - in real app, fetch returns from DB
	// Portfolio Returns (last 10 days)
	portfolioReturns := []float64{0.01, -0.005, 0.002, 0.015, -0.01, 0.005, 0.008, -0.002, 0.012, 0.003}
	
	// Factor Returns (Market, Size, Value)
	factorReturns := [][]float64{
		{0.008, -0.002, 0.001}, // Day 1
		{-0.004, 0.001, 0.000}, // Day 2
		{0.001, 0.000, 0.001},  // ...
		{0.012, -0.003, 0.002},
		{-0.008, 0.002, -0.001},
		{0.004, 0.001, 0.000},
		{0.006, -0.001, 0.001},
		{-0.001, 0.000, 0.000},
		{0.010, -0.002, 0.003},
		{0.002, 0.001, 0.000},
	}

	// Calculate Beta
	// Note: We are using the RegressionService stub which returns mock betas
	results, err := h.regressionService.CalculateRollingBeta(portfolioReturns, factorReturns, 5)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the latest beta
	if len(results) > 0 {
		latest := results[len(results)-1]
		response := map[string]interface{}{
			"portfolio_id": portfolioIDStr,
			"betas": map[string]float64{
				"Market": latest.Betas[0],
				"Size":   latest.Betas[1],
				"Value":  latest.Betas[2],
			},
			"r_squared": latest.RSquared,
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	http.Error(w, "insufficient data", http.StatusBadRequest)
}

// GetAttribution decomposes portfolio returns into factor contributions
func (h *AnalyticsHandler) GetAttribution(w http.ResponseWriter, r *http.Request) {
	portfolioIDStr := chi.URLParam(r, "portfolioID")
	_, err := uuid.Parse(portfolioIDStr)
	if err != nil {
		http.Error(w, "invalid portfolioID", http.StatusBadRequest)
		return
	}

	// Mock data for MVP
	portfolioReturns := []float64{0.01, -0.005, 0.002, 0.015, -0.01, 0.005, 0.008, -0.002, 0.012, 0.003}
	factorReturnsMap := map[string][]float64{
		"Market": {0.008, -0.004, 0.001, 0.012, -0.008, 0.004, 0.006, -0.001, 0.010, 0.002},
		"Size":   {-0.002, 0.001, 0.000, -0.003, 0.002, 0.001, -0.001, 0.000, -0.002, 0.001},
		"Value":  {0.001, 0.000, 0.001, 0.002, -0.001, 0.000, 0.001, 0.000, 0.003, 0.000},
	}

	result, err := h.attributionService.DecomposeReturns(portfolioReturns, factorReturnsMap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(result)
}
