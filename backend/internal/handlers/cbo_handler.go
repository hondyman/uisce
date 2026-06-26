package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/cbo"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// CBOHandler handles Cost-Based Optimizer API requests
type CBOHandler struct {
	router          *cbo.QueryRouter
	workloadTracker *cbo.WorkloadTracker
	costEstimator   *cbo.CostEstimator
}

// NewCBOHandler creates a new CBO handler
func NewCBOHandler(
	router *cbo.QueryRouter,
	workloadTracker *cbo.WorkloadTracker,
	costEstimator *cbo.CostEstimator,
) *CBOHandler {
	return &CBOHandler{
		router:          router,
		workloadTracker: workloadTracker,
		costEstimator:   costEstimator,
	}
}

// RegisterRoutes registers CBO routes
func (h *CBOHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/cbo", func(r chi.Router) {
		r.Post("/plan", h.GetQueryPlan)
		r.Get("/stats", h.GetStats)
		r.Get("/patterns", h.GetTopPatterns)
		r.Get("/recommendations", h.GetRecommendations)
		r.Post("/analyze", h.AnalyzeQuery)
		r.Get("/workload", h.GetWorkloadSummary)
	})
}

// GetQueryPlan returns the optimal execution plan for a semantic query
func (h *CBOHandler) GetQueryPlan(w http.ResponseWriter, r *http.Request) {
	var query cbo.SemanticQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	plan, err := h.router.Route(ctx, &query)
	if err != nil {
		http.Error(w, "Failed to generate query plan: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(plan)
}

// GetStats returns CBO performance statistics
func (h *CBOHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	tenantID, _ := uuid.Parse(tenantIDStr)

	ctx := r.Context()
	stats, err := h.router.GetStats(ctx, tenantID)
	if err != nil {
		http.Error(w, "Failed to get stats: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// GetTopPatterns returns the most frequently occurring query patterns
func (h *CBOHandler) GetTopPatterns(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	tenantID, _ := uuid.Parse(tenantIDStr)

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	ctx := r.Context()
	patterns, err := h.workloadTracker.GetTopPatterns(ctx, tenantID, limit)
	if err != nil {
		http.Error(w, "Failed to get patterns: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(patterns)
}

// GetRecommendations returns optimization recommendations based on workload analysis
func (h *CBOHandler) GetRecommendations(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	tenantID, _ := uuid.Parse(tenantIDStr)

	ctx := r.Context()
	recommendations, err := h.workloadTracker.SuggestOptimizations(ctx, tenantID)
	if err != nil {
		http.Error(w, "Failed to get recommendations: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(recommendations)
}

// AnalyzeQuery performs cost analysis on a query without executing it
func (h *CBOHandler) AnalyzeQuery(w http.ResponseWriter, r *http.Request) {
	var query cbo.SemanticQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get cost estimate
	cost, factors, err := h.costEstimator.EstimateCost(ctx, &query)
	if err != nil {
		http.Error(w, "Failed to analyze query: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Get execution plan
	plan, _ := h.router.Route(ctx, &query)

	response := map[string]interface{}{
		"query_hash":     cbo.HashQuery(&query),
		"query_pattern":  cbo.ExtractPattern(&query),
		"estimated_cost": cost,
		"cost_factors":   factors,
		"execution_plan": plan,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetWorkloadSummary returns a summary of recent query workload
func (h *CBOHandler) GetWorkloadSummary(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := jwtmiddleware.GetClaimsFromContext(r).TenantID
	tenantID, _ := uuid.Parse(tenantIDStr)

	hours := 24
	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil && parsed > 0 {
			hours = parsed
		}
	}

	ctx := r.Context()
	summary, err := h.workloadTracker.GetWorkloadSummary(ctx, tenantID, hours)
	if err != nil {
		http.Error(w, "Failed to get workload summary: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}
