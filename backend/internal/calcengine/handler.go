package calcengine

import (
	"encoding/json"
	"net/http"
	"time"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// CalcEngineHandler provides HTTP endpoints for the calc engine
type CalcEngineHandler struct {
	engine *UnifiedCalcEngine
}

// NewCalcEngineHandler creates a new calc engine HTTP handler
func NewCalcEngineHandler(engine *UnifiedCalcEngine) *CalcEngineHandler {
	return &CalcEngineHandler{engine: engine}
}

// RegisterRoutes registers the calc engine routes
func (h *CalcEngineHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/calc/execute", h.withTenantContext(h.handleCalculate))
	mux.HandleFunc("/api/calc/nav", h.withTenantContext(h.handleNAV))
	mux.HandleFunc("/api/calc/returns", h.withTenantContext(h.handleReturns))
	mux.HandleFunc("/api/calc/risk", h.withTenantContext(h.handleRisk))
	mux.HandleFunc("/api/calc/holdings", h.withTenantContext(h.handleHoldings))
	mux.HandleFunc("/api/calc/performance", h.withTenantContext(h.handlePerformance))
	mux.HandleFunc("/api/calc/cache/stats", h.handleCacheStats)
	mux.HandleFunc("/api/calc/cache/invalidate", h.withTenantContext(h.handleCacheInvalidate))
}

// TenantContext holds tenant information extracted from request
type TenantContext struct {
	TenantID     string
	DatasourceID string
	UserID       string
}

// withTenantContext middleware extracts tenant context from headers
func (h *CalcEngineHandler) withTenantContext(next func(w http.ResponseWriter, r *http.Request, tc *TenantContext)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
		datasourceID := r.Header.Get("X-Tenant-Datasource-ID")

		// Also check query params (for backward compat)
		if tenantID == "" {
			tenantID = r.URL.Query().Get("tenant_id")
		}
		if datasourceID == "" {
			datasourceID = r.URL.Query().Get("datasource_id")
		}

		if tenantID == "" || datasourceID == "" {
			h.errorResponse(w, http.StatusBadRequest, "tenant_id and datasource_id required")
			return
		}

		tc := &TenantContext{
			TenantID:     tenantID,
			DatasourceID: datasourceID,
			UserID:       r.Header.Get("X-User-ID"),
		}

		next(w, r, tc)
	}
}

// handleCalculate executes a generic calculation
// POST /api/calc/execute
func (h *CalcEngineHandler) handleCalculate(w http.ResponseWriter, r *http.Request, tc *TenantContext) {
	if r.Method != http.MethodPost {
		h.errorResponse(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	var req CalcRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Inject tenant context
	req.TenantID = tc.TenantID
	req.DatasourceID = tc.DatasourceID

	// Execute calculation
	result, err := h.engine.Calculate(r.Context(), &req)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, result)
}

// handleNAV calculates Net Asset Value
// GET /api/calc/nav?portfolio_id=xxx&as_of_date=2024-01-01
func (h *CalcEngineHandler) handleNAV(w http.ResponseWriter, r *http.Request, tc *TenantContext) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" {
		h.errorResponse(w, http.StatusBadRequest, "portfolio_id required")
		return
	}

	asOfDate := time.Now()
	if dateStr := r.URL.Query().Get("as_of_date"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			asOfDate = t
		}
	}

	result, err := h.engine.CalculateNAV(r.Context(), tc.TenantID, tc.DatasourceID, portfolioID, asOfDate)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, result)
}

// handleReturns calculates portfolio returns
// GET /api/calc/returns?portfolio_id=xxx&start_date=2024-01-01&end_date=2024-12-01&type=monthly
func (h *CalcEngineHandler) handleReturns(w http.ResponseWriter, r *http.Request, tc *TenantContext) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" {
		h.errorResponse(w, http.StatusBadRequest, "portfolio_id required")
		return
	}

	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()
	returnType := "daily"

	if dateStr := r.URL.Query().Get("start_date"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			startDate = t
		}
	}
	if dateStr := r.URL.Query().Get("end_date"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			endDate = t
		}
	}
	if rt := r.URL.Query().Get("type"); rt != "" {
		returnType = rt
	}

	result, err := h.engine.CalculateReturns(r.Context(), tc.TenantID, tc.DatasourceID,
		portfolioID, startDate, endDate, returnType)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, result)
}

// handleRisk calculates risk metrics
// GET /api/calc/risk?portfolio_id=xxx&metrics=var,sharpe,volatility
func (h *CalcEngineHandler) handleRisk(w http.ResponseWriter, r *http.Request, tc *TenantContext) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" {
		h.errorResponse(w, http.StatusBadRequest, "portfolio_id required")
		return
	}

	// Parse metrics list
	metricsStr := r.URL.Query().Get("metrics")
	metrics := []string{"var", "volatility", "sharpe"}
	if metricsStr != "" {
		metrics = splitAndTrim(metricsStr)
	}

	params := map[string]interface{}{}
	if lookback := r.URL.Query().Get("lookback_days"); lookback != "" {
		params["lookback_days"] = lookback
	}

	result, err := h.engine.CalculateRiskMetrics(r.Context(), tc.TenantID, tc.DatasourceID,
		portfolioID, metrics, params)
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, result)
}

// handleHoldings retrieves holdings with calculated values
// GET /api/calc/holdings?portfolio_id=xxx&as_of_date=2024-01-01
func (h *CalcEngineHandler) handleHoldings(w http.ResponseWriter, r *http.Request, tc *TenantContext) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" {
		h.errorResponse(w, http.StatusBadRequest, "portfolio_id required")
		return
	}

	asOfDate := time.Now()
	if dateStr := r.URL.Query().Get("as_of_date"); dateStr != "" {
		if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			asOfDate = t
		}
	}

	result, err := h.engine.Calculate(r.Context(), &CalcRequest{
		TenantID:     tc.TenantID,
		DatasourceID: tc.DatasourceID,
		MetricName:   "Holdings",
		Mode:         ModeRealtime,
		Params: map[string]interface{}{
			"portfolio_id": portfolioID,
			"as_of_date":   asOfDate,
		},
	})
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, result)
}

// handlePerformance retrieves performance metrics
// GET /api/calc/performance?portfolio_id=xxx
func (h *CalcEngineHandler) handlePerformance(w http.ResponseWriter, r *http.Request, tc *TenantContext) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	portfolioID := r.URL.Query().Get("portfolio_id")
	if portfolioID == "" {
		h.errorResponse(w, http.StatusBadRequest, "portfolio_id required")
		return
	}

	result, err := h.engine.Calculate(r.Context(), &CalcRequest{
		TenantID:     tc.TenantID,
		DatasourceID: tc.DatasourceID,
		MetricName:   "Performance",
		Mode:         ModeRealtime,
		Params: map[string]interface{}{
			"portfolio_id": portfolioID,
		},
	})
	if err != nil {
		h.errorResponse(w, http.StatusInternalServerError, err.Error())
		return
	}

	h.jsonResponse(w, http.StatusOK, result)
}

// handleCacheStats returns cache statistics
// GET /api/calc/cache/stats
func (h *CalcEngineHandler) handleCacheStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.errorResponse(w, http.StatusMethodNotAllowed, "GET required")
		return
	}

	stats := h.engine.resultCache.Stats()
	h.jsonResponse(w, http.StatusOK, stats)
}

// handleCacheInvalidate invalidates cache for a tenant
// POST /api/calc/cache/invalidate
func (h *CalcEngineHandler) handleCacheInvalidate(w http.ResponseWriter, r *http.Request, tc *TenantContext) {
	if r.Method != http.MethodPost {
		h.errorResponse(w, http.StatusMethodNotAllowed, "POST required")
		return
	}

	h.engine.resultCache.InvalidateForTenant(tc.TenantID, tc.DatasourceID)
	h.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

// Helper methods

func (h *CalcEngineHandler) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *CalcEngineHandler) errorResponse(w http.ResponseWriter, status int, message string) {
	h.jsonResponse(w, status, map[string]string{"error": message})
}

func splitAndTrim(s string) []string {
	var result []string
	var current string
	for _, c := range s {
		if c == ',' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else if c != ' ' {
			current += string(c)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}
