package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/internal/services"
	"github.com/hondyman/semlayer/backend/internal/wealth"
)

type FinancialHandler struct {
	financialService *wealth.FinancialService
	db               *sql.DB
}

func NewFinancialHandler(financialService *wealth.FinancialService, db *sql.DB) *FinancialHandler {
	return &FinancialHandler{
		financialService: financialService,
		db:               db,
	}
}

// RegisterRoutes registers the routes for FinancialHandler.
func (fh *FinancialHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/financial", func(r chi.Router) {
		r.Post("/irr", fh.CalculateIRR)
		r.Post("/xirr", fh.CalculateXIRR)
		r.Post("/wirr", fh.CalculateWIRR)
		r.Post("/npv", fh.CalculateNPV)
		r.Post("/amortization", fh.CalculateAmortizationPayment)
		r.Post("/ratio", fh.CalculateRatio)
		r.Post("/payback", fh.CalculatePaybackPeriod)
		r.Post("/weighted-sum", fh.CalculateWeightedSum)
		r.Post("/mirr", fh.CalculateMIRR)
		r.Post("/cagr", fh.CalculateCAGR)
		r.Post("/sharpe", fh.CalculateSharpeRatio)
		r.Post("/sum-of-ratios", fh.CalculateSumOfRatios)
		r.Post("/vectorized", fh.CalculateVectorized)
	})
}

// CalculateIRR handles IRR calculation requests
func (fh *FinancialHandler) CalculateIRR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CashFlows []float64 `json:"cash_flows" binding:"required"`
		Guess     float64   `json:"guess,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	irr, err := fh.financialService.IRR(req.CashFlows, req.Guess)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"irr":    irr,
		"method": "newton_raphson",
	})
}

// CalculateXIRR handles XIRR calculation requests
func (fh *FinancialHandler) CalculateXIRR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CashFlows []float64 `json:"cash_flows" binding:"required"`
		Dates     []string  `json:"dates" binding:"required"`
		Guess     float64   `json:"guess,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	xirr, err := fh.financialService.XIRR(req.CashFlows, req.Dates, req.Guess)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"xirr":   xirr,
		"method": "newton_raphson",
	})
}

// CalculateWIRR handles WIRR calculation requests
func (fh *FinancialHandler) CalculateWIRR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CashFlows []float64 `json:"cash_flows" binding:"required"`
		Weights   []float64 `json:"weights" binding:"required"`
		Guess     float64   `json:"guess,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	wirr, err := fh.financialService.WIRR(req.CashFlows, req.Weights, req.Guess)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"wirr":   wirr,
		"method": "weighted_irr",
	})
}

// CalculateNPV handles NPV calculation requests
func (fh *FinancialHandler) CalculateNPV(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Rate      float64   `json:"rate" binding:"required"`
		CashFlows []float64 `json:"cash_flows" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	npv := fh.financialService.NPV(req.Rate, req.CashFlows)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"npv":  npv,
		"rate": req.Rate,
	})
}

// CalculateAmortizationPayment handles amortization calculation requests
func (fh *FinancialHandler) CalculateAmortizationPayment(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Principal float64 `json:"principal" binding:"required"`
		Rate      float64 `json:"rate" binding:"required"`    // Assumed to be annual rate
		Periods   int     `json:"periods" binding:"required"` // Assumed to be in months
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	// Based on common financial conventions and examples, we assume the provided
	// rate is annual and periods are monthly. The rate is converted to a per-period rate.
	monthlyRate := req.Rate / 12

	payment, err := fh.financialService.AmortizationPayment(monthlyRate, req.Periods, req.Principal)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payment": payment,
	})
}

// CalculateRatio handles simple ratio calculations
func (fh *FinancialHandler) CalculateRatio(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Numerator   float64 `json:"numerator" binding:"required"`
		Denominator float64 `json:"denominator" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	ratio, err := fh.financialService.Ratio(req.Numerator, req.Denominator)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()}) // Denominator is zero
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ratio": ratio,
	})
}

// CalculatePaybackPeriod handles payback period calculations
func (fh *FinancialHandler) CalculatePaybackPeriod(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CashFlows []float64 `json:"cash_flows" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	period, err := fh.financialService.PaybackPeriod(req.CashFlows)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"payback_period": period,
	})
}

// CalculateWeightedSum handles weighted sum calculations
func (fh *FinancialHandler) CalculateWeightedSum(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Values  []float64 `json:"values" binding:"required"`
		Weights []float64 `json:"weights" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	result, err := fh.financialService.WeightedSum(req.Values, req.Weights)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"weighted_sum": result,
	})
}

// CalculateMIRR handles MIRR calculation requests
func (fh *FinancialHandler) CalculateMIRR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CashFlows    []float64 `json:"cash_flows" binding:"required"`
		FinanceRate  float64   `json:"finance_rate" binding:"required"`
		ReinvestRate float64   `json:"reinvest_rate" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	mirr, err := fh.financialService.MIRR(req.CashFlows, req.FinanceRate, req.ReinvestRate)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"mirr": mirr})
}

// CalculateCAGR handles Compound Annual Growth Rate calculation requests
func (fh *FinancialHandler) CalculateCAGR(w http.ResponseWriter, r *http.Request) {
	var req struct {
		StartValue float64 `json:"start_value" binding:"required"`
		EndValue   float64 `json:"end_value" binding:"required"`
		Years      float64 `json:"years" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	cagr, err := fh.financialService.CAGR(req.StartValue, req.EndValue, req.Years)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"cagr": cagr})
}

// CalculateSharpeRatio handles Sharpe Ratio calculation requests
func (fh *FinancialHandler) CalculateSharpeRatio(w http.ResponseWriter, r *http.Request) {
	var req struct {
		AverageReturn float64 `json:"average_return" binding:"required"`
		RiskFreeRate  float64 `json:"risk_free_rate" binding:"required"`
		StdDev        float64 `json:"std_dev" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	sharpe, err := fh.financialService.SharpeRatio(req.AverageReturn, req.RiskFreeRate, req.StdDev)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"sharpe_ratio": sharpe})
}

// CalculateSumOfRatios handles sum of ratios calculation requests
func (fh *FinancialHandler) CalculateSumOfRatios(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Components []wealth.SumOfRatiosComponent `json:"components" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	result, err := fh.financialService.SumOfRatios(req.Components)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"sum_of_ratios": result})
}

// CalculateVectorized handles vectorized Excel formula calculations across multiple metrics and entities
func (fh *FinancialHandler) CalculateVectorized(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Metrics  []string `json:"metrics" binding:"required"`
		Entities []string `json:"entities" binding:"required"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	results, err := services.ExecuteVectorizedExcelCalc(req.Metrics, req.Entities, fh.db)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"results": results,
		"batch_info": map[string]interface{}{
			"metric_count":       len(req.Metrics),
			"entity_count":       len(req.Entities),
			"total_calculations": len(req.Metrics) * len(req.Entities),
		},
	})
}
