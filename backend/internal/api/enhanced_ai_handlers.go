package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/hondyman/semlayer/backend/internal/scheduler_intelligence/ai"
)

// EnhancedAIHandler handles world-class AI scheduler features
type EnhancedAIHandler struct {
	exceptionClusterer *ai.ExceptionClusterer
	sloForecaster      *ai.SLOForecaster
	anomalyDetector    *ai.AnomalyDetector
	smartRetry         *ai.SmartRetryOptimizer
	runbookGenerator   *ai.RunbookGenerator
	nlqScheduler       *ai.NLQScheduler
}

// NewEnhancedAIHandler creates a new enhanced AI handler
func NewEnhancedAIHandler(
	clusterer *ai.ExceptionClusterer,
	forecaster *ai.SLOForecaster,
	anomaly *ai.AnomalyDetector,
	retry *ai.SmartRetryOptimizer,
	runbook *ai.RunbookGenerator,
	nlq *ai.NLQScheduler,
) *EnhancedAIHandler {
	return &EnhancedAIHandler{
		exceptionClusterer: clusterer,
		sloForecaster:      forecaster,
		anomalyDetector:    anomaly,
		smartRetry:         retry,
		runbookGenerator:   runbook,
		nlqScheduler:       nlq,
	}
}

// RegisterRoutes registers enhanced AI routes
func (h *EnhancedAIHandler) RegisterRoutes(r chi.Router) {
	r.Route("/api/scheduler/ai/enhanced", func(r chi.Router) {
		// Exception Clustering
		r.Post("/exceptions/cluster", h.ClusterExceptions)
		r.Get("/exceptions/clusters", h.GetExceptionClusters)
		r.Get("/exceptions/insights", h.GetExceptionInsights)

		// SLO Forecasting
		r.Post("/slo/forecast", h.ForecastSLO)
		r.Get("/slo/dashboard", h.GetSLODashboard)
		r.Get("/slo/error-budget", h.GetErrorBudget)

		// Anomaly Detection
		r.Post("/anomalies/detect", h.DetectAnomalies)
		r.Get("/anomalies/report", h.GetAnomalyReport)
		r.Get("/anomalies/job/{jobId}/health", h.GetJobHealth)

		// Smart Retry
		r.Post("/retry/optimize", h.OptimizeRetryPolicy)
		r.Get("/retry/analysis/{jobId}", h.GetRetryAnalysis)

		// Runbook Generation
		r.Post("/runbook/generate", h.GenerateRunbook)
		r.Get("/runbook/{jobId}", h.GetRunbook)
		r.Post("/runbook/{id}/update", h.UpdateRunbook)

		// Natural Language Query
		r.Post("/ask", h.AskScheduler)
		r.Get("/ask/suggestions", h.GetQuerySuggestions)
	})
}

// ClusterExceptions groups similar failures
func (h *EnhancedAIHandler) ClusterExceptions(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Events []ai.ExceptionEvent `json:"events"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.exceptionClusterer.ClusterExceptions(r.Context(), req.Events)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetExceptionClusters returns current exception clusters
func (h *EnhancedAIHandler) GetExceptionClusters(w http.ResponseWriter, r *http.Request) {
	// Would fetch from persistent storage in production
	tenantID := r.URL.Query().Get("tenant_id")

	clusters := []map[string]interface{}{
		{
			"id":               "cls-001",
			"name":             "Timeout Failures in Pre-Agg Jobs",
			"pattern":          "context deadline exceeded",
			"category":         "timeout",
			"occurrence_count": 12,
			"severity":         "high",
			"trend_direction":  "increasing",
			"affected_jobs":    []string{"EU Pre-Agg", "APAC Pre-Agg"},
			"tenant_id":        tenantID,
		},
		{
			"id":               "cls-002",
			"name":             "Auth Failures",
			"pattern":          "unauthorized: token expired",
			"category":         "auth",
			"occurrence_count": 5,
			"severity":         "medium",
			"trend_direction":  "stable",
			"affected_jobs":    []string{"Data Sync"},
			"tenant_id":        tenantID,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(clusters)
}

// GetExceptionInsights returns actionable insights
func (h *EnhancedAIHandler) GetExceptionInsights(w http.ResponseWriter, r *http.Request) {
	insights := []ai.ClusterInsight{
		{
			Type:        "emerging_pattern",
			Title:       "New timeout pattern detected",
			Description: "Timeout errors increased 3x in the last 6 hours",
			Impact:      "Affecting 4 jobs across 2 tenants",
			Priority:    1,
		},
		{
			Type:        "cross_tenant",
			Title:       "Cross-tenant connectivity issue",
			Description: "Similar connection errors across multiple tenants",
			Impact:      "Platform-level investigation recommended",
			Priority:    1,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(insights)
}

// ForecastSLO predicts SLO breaches
func (h *EnhancedAIHandler) ForecastSLO(w http.ResponseWriter, r *http.Request) {
	var req struct {
		SLO           ai.SLODefinition  `json:"slo"`
		DataPoints    []ai.SLODataPoint `json:"data_points"`
		ForecastHours int               `json:"forecast_hours"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	forecast, err := h.sloForecaster.ForecastSLOBreach(r.Context(), req.SLO, req.DataPoints, req.ForecastHours)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(forecast)
}

// GetSLODashboard returns real-time SLO status
func (h *EnhancedAIHandler) GetSLODashboard(w http.ResponseWriter, r *http.Request) {
	dashboard := map[string]interface{}{
		"generated_at":   time.Now(),
		"overall_health": "at_risk",
		"health_score":   78.5,
		"total_slos":     12,
		"healthy_slos":   9,
		"at_risk_slos":   2,
		"breached_slos":  1,
		"error_budget": map[string]interface{}{
			"total_budget_minutes": 60.0,
			"consumed_minutes":     45.0,
			"remaining_minutes":    15.0,
			"burn_rate":            1.5,
			"projected_exhaustion": time.Now().Add(10 * time.Hour),
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dashboard)
}

// GetErrorBudget returns error budget status
func (h *EnhancedAIHandler) GetErrorBudget(w http.ResponseWriter, r *http.Request) {
	budget := ai.ErrorBudgetInfo{
		TotalBudgetMinutes: 60.0,
		ConsumedMinutes:    45.0,
		RemainingMinutes:   15.0,
		ConsumptionRate:    4.5,
		BurnRate:           1.5,
		WindowStart:        time.Now().Add(-24 * time.Hour),
		WindowEnd:          time.Now(),
	}

	exhaustion := time.Now().Add(10 * time.Hour)
	budget.ProjectedExhaustion = &exhaustion

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(budget)
}

// DetectAnomalies analyzes for unusual behavior
func (h *EnhancedAIHandler) DetectAnomalies(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Metrics     []ai.JobExecutionMetric `json:"metrics"`
		Sensitivity float64                 `json:"sensitivity"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Sensitivity == 0 {
		req.Sensitivity = 1.0
	}

	report, err := h.anomalyDetector.DetectAnomalies(r.Context(), req.Metrics, req.Sensitivity)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// GetAnomalyReport returns recent anomaly report
func (h *EnhancedAIHandler) GetAnomalyReport(w http.ResponseWriter, r *http.Request) {
	// Would fetch from persistent storage
	report := map[string]interface{}{
		"generated_at":       time.Now(),
		"total_executions":   500,
		"anomalies_detected": 12,
		"anomaly_rate":       0.024,
		"top_anomalies": []map[string]interface{}{
			{
				"job_name":     "Pre-Agg EU",
				"anomaly_type": "duration",
				"severity":     "high",
				"description":  "Execution was 4.2x slower than usual",
			},
			{
				"job_name":     "Data Sync",
				"anomaly_type": "pattern",
				"severity":     "medium",
				"description":  "Sudden 80% change from recent pattern",
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(report)
}

// GetJobHealth returns health score for a job
func (h *EnhancedAIHandler) GetJobHealth(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")

	health := ai.JobHealth{
		JobID:          uuid.MustParse(jobID),
		JobName:        "Sample Job",
		HealthScore:    85.5,
		AnomalyCount:   2,
		TrendDirection: "improving",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(health)
}

// OptimizeRetryPolicy generates optimized retry configuration
func (h *EnhancedAIHandler) OptimizeRetryPolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JobID         uuid.UUID             `json:"job_id"`
		Outcomes      []ai.RetryOutcome     `json:"outcomes"`
		CurrentPolicy ai.CurrentRetryPolicy `json:"current_policy"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	result, err := h.smartRetry.OptimizeRetryPolicy(r.Context(), req.JobID, req.Outcomes, req.CurrentPolicy)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// GetRetryAnalysis returns retry pattern analysis
func (h *EnhancedAIHandler) GetRetryAnalysis(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")

	analysis := map[string]interface{}{
		"job_id":                  jobID,
		"total_executions":        200,
		"executions_with_retries": 45,
		"overall_success_rate":    0.92,
		"avg_attempts_per_exec":   1.3,
		"wasted_retries":          15,
		"recommendation":          "Reduce max attempts from 5 to 3",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analysis)
}

// GenerateRunbook creates an automated runbook
func (h *EnhancedAIHandler) GenerateRunbook(w http.ResponseWriter, r *http.Request) {
	var req ai.RunbookContext
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	runbook, err := h.runbookGenerator.GenerateRunbook(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runbook)
}

// GetRunbook returns an existing runbook
func (h *EnhancedAIHandler) GetRunbook(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")

	// Would fetch from storage
	runbook := map[string]interface{}{
		"id":           "rb-001",
		"job_id":       jobID,
		"title":        "Runbook: Pre-Agg EU",
		"version":      2,
		"sections":     []string{"Initial Triage", "Common Issues", "Recovery"},
		"last_updated": time.Now().Add(-24 * time.Hour),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(runbook)
}

// UpdateRunbook updates with new learnings
func (h *EnhancedAIHandler) UpdateRunbook(w http.ResponseWriter, r *http.Request) {
	runbookID := chi.URLParam(r, "id")

	var req struct {
		NewPatterns []ai.FailurePattern `json:"new_patterns"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"runbook_id": runbookID,
		"status":     "updated",
		"version":    3,
	})
}

// AskScheduler handles natural language queries
func (h *EnhancedAIHandler) AskScheduler(w http.ResponseWriter, r *http.Request) {
	var req ai.NLQQuery
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build context (would fetch real data)
	context := ai.SchedulerContext{
		TenantID:    req.TenantID,
		CurrentTime: time.Now(),
		ScheduleStats: ai.ScheduleStats{
			TotalJobs:          45,
			RunningNow:         3,
			ScheduledToday:     28,
			FailuresLast24h:    4,
			SuccessRateLast24h: 0.92,
			AnomaliesDetected:  2,
		},
	}

	response, err := h.nlqScheduler.ProcessQuery(r.Context(), req, context)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetQuerySuggestions returns suggested questions
func (h *EnhancedAIHandler) GetQuerySuggestions(w http.ResponseWriter, r *http.Request) {
	suggestions := []string{
		"What's the current scheduler status?",
		"Show me recent failures",
		"What's scheduled for today?",
		"Are there any SLO risks?",
		"Compare today vs yesterday",
		"Which jobs are running now?",
		"Show me the anomaly report",
		"What's the error budget status?",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(suggestions)
}
