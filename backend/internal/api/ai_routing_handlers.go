package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/backend/pkg/ai_routing"
)

// AIRoutingHandlers manages AI routing API endpoints
type AIRoutingHandlers struct {
	router            *ai_routing.IntelligentRouter
	feedbackCollector *ai_routing.FeedbackCollector
	metricsCollector  *ai_routing.RoutingMetricsCollector
}

// NewAIRoutingHandlers creates routing handler
func NewAIRoutingHandlers(
	router *ai_routing.IntelligentRouter,
	feedbackCollector *ai_routing.FeedbackCollector,
	metricsCollector *ai_routing.RoutingMetricsCollector,
) *AIRoutingHandlers {
	return &AIRoutingHandlers{
		router:            router,
		feedbackCollector: feedbackCollector,
		metricsCollector:  metricsCollector,
	}
}

// RegisterRoutes registers all AI routing endpoints
func (h *AIRoutingHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/ai-routing", func(r chi.Router) {
		r.Post("/route", h.RouteWorkflow)
		r.Get("/metrics", h.GetMetrics)
		r.Get("/live-decisions", h.GetLiveDecisions)
		r.Post("/feedback/outcome", h.RecordOutcome)
		r.Get("/branch-performance", h.GetBranchPerformance)
		r.Get("/decision-history/{workflowID}", h.GetDecisionHistory)
		r.Get("/model-performance", h.GetModelPerformance)
	})
}

// RouteWorkflow handles routing decision request
// POST /api/ai-routing/route
func (h *AIRoutingHandlers) RouteWorkflow(w http.ResponseWriter, r *http.Request) {
	var req ai_routing.RoutingRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate tenant context
	if req.TenantID == "" || req.DatasourceID == "" {
		http.Error(w, "Tenant and datasource ID required", http.StatusBadRequest)
		return
	}

	// Execute routing decision
	decision, err := h.router.Route(r.Context(), req)
	if err != nil {
		log.Printf("Routing error: %v", err)
		http.Error(w, fmt.Sprintf("Routing failed: %v", err), http.StatusInternalServerError)
		return
	}

	// Store decision for audit trail
	if err := h.feedbackCollector.StoreRoutingDecision(r.Context(), decision, req); err != nil {
		log.Printf("Failed to store decision: %v", err)
		// Don't fail the request, just log the error
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(decision)
}

// GetMetrics returns current routing metrics
// GET /api/ai-routing/metrics
func (h *AIRoutingHandlers) GetMetrics(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	metrics := h.metricsCollector.GetMetrics()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// GetLiveDecisions returns recent routing decisions
// GET /api/ai-routing/live-decisions
func (h *AIRoutingHandlers) GetLiveDecisions(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 10
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	if limit > 100 {
		limit = 100
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	// In production, fetch from database
	// For now, return mock data with collection from metricsCollector
	response := map[string]interface{}{
		"decisions": []interface{}{},
		"limit":     limit,
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// RecordOutcome records workflow completion for feedback
// POST /api/ai-routing/feedback/outcome
func (h *AIRoutingHandlers) RecordOutcome(w http.ResponseWriter, r *http.Request) {
	var outcome ai_routing.WorkflowOutcome

	if err := json.NewDecoder(r.Body).Decode(&outcome); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	outcome.Timestamp = time.Now()

	// Store outcome for RL training
	if err := h.feedbackCollector.StoreWorkflowOutcome(r.Context(), outcome); err != nil {
		log.Printf("Failed to store outcome: %v", err)
		http.Error(w, "Failed to record outcome", http.StatusInternalServerError)
		return
	}

	// Log to metrics
	h.metricsCollector.LogOutcome(outcome)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":      "recorded",
		"workflow_id": outcome.WorkflowID,
	})
}

// GetBranchPerformance returns performance metrics per branch
// GET /api/ai-routing/branch-performance
func (h *AIRoutingHandlers) GetBranchPerformance(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	// Fetch from database
	metrics, err := h.feedbackCollector.GetBranchPerformance(r.Context(), tenantID)
	if err != nil {
		log.Printf("Failed to get branch performance: %v", err)
		http.Error(w, "Failed to retrieve metrics", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// GetDecisionHistory returns routing decisions for a workflow
// GET /api/ai-routing/decision-history/{workflowID}
func (h *AIRoutingHandlers) GetDecisionHistory(w http.ResponseWriter, r *http.Request) {
	workflowID := chi.URLParam(r, "workflowID")
	if workflowID == "" {
		http.Error(w, "workflowID required", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
		limit = l
	}

	decisions, err := h.feedbackCollector.GetDecisionHistory(r.Context(), workflowID, limit)
	if err != nil {
		log.Printf("Failed to get decision history: %v", err)
		http.Error(w, "Failed to retrieve decisions", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(decisions)
}

// GetModelPerformance returns ML model performance metrics
// GET /api/ai-routing/model-performance
func (h *AIRoutingHandlers) GetModelPerformance(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	// Fetch current metrics
	metrics := h.metricsCollector.GetMetrics()

	// Format response
	response := map[string]interface{}{
		"overall_accuracy":       metrics.OverallAccuracy,
		"avg_decision_time_ms":   metrics.AvgDecisionTimeMs,
		"model_agreement_rate":   metrics.ModelAgreementRate,
		"workflows_routed_today": metrics.WorkflowsRoutedToday,
		"branch_distribution":    metrics.BranchDistribution,
		"model_performance":      metrics.ModelPerformance,
		"rl_episodes":            metrics.RLEpisodes,
		"rl_epsilon":             metrics.RLEpsilon,
		"rl_avg_q_value":         metrics.RLAvgQValue,
		"rl_last_reward":         metrics.RLLastReward,
		"last_retrain_time":      metrics.LastRetrainTime,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Note: tenant+datasource extraction is handled by the shared request helpers
// and the frontend shim. No local helper is required here.

// InitializeAIRouting initializes the AI routing system
func InitializeAIRouting() (*AIRoutingHandlers, error) {
	// Create individual components
	predictiveModel := ai_routing.NewPredictiveRoutingModel("") // Empty endpoint = use heuristic
	rlAgent := ai_routing.NewRLRoutingAgent()
	sentimentAnalyzer := ai_routing.NewSentimentClassifier()
	ruleEngine := ai_routing.NewHybridRuleEngine()
	metricsCollector := ai_routing.NewRoutingMetricsCollector()

	// Create intelligent router
	router := ai_routing.NewIntelligentRouter(
		predictiveModel,
		rlAgent,
		sentimentAnalyzer,
		ruleEngine,
		metricsCollector,
	)

	// Note: FeedbackCollector requires DB connection
	// This will be injected during API setup
	var feedbackCollector *ai_routing.FeedbackCollector

	return &AIRoutingHandlers{
		router:            router,
		feedbackCollector: feedbackCollector,
		metricsCollector:  metricsCollector,
	}, nil
}
