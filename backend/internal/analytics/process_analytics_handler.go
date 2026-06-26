package analytics

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// ProcessAnalyticsHandler handles HTTP requests for process analytics
type ProcessAnalyticsHandler struct {
	service *ProcessAnalyticsService
}

// NewProcessAnalyticsHandler creates a new process analytics handler
func NewProcessAnalyticsHandler(service *ProcessAnalyticsService) *ProcessAnalyticsHandler {
	return &ProcessAnalyticsHandler{
		service: service,
	}
}

// GetWorkflowMetrics retrieves metrics for a specific workflow
func (h *ProcessAnalyticsHandler) GetWorkflowMetrics(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing X-Tenant-ID header"})
		return
	}

	workflowType := chi.URLParam(r, "workflowType")
	timeWindowStr := r.URL.Query().Get("timeWindow")

	// Default to 7 days if not specified
	timeWindow := 7 * 24 * time.Hour
	if timeWindowStr != "" {
		if days, err := strconv.Atoi(timeWindowStr); err == nil && days > 0 {
			timeWindow = time.Duration(days) * 24 * time.Hour
		}
	}

	// Query for workflow metrics
	query := `
		SELECT
			step_name,
			step_type,
			COUNT(*) as execution_count,
			AVG(EXTRACT(EPOCH FROM duration)) as avg_duration_seconds,
			MIN(EXTRACT(EPOCH FROM duration)) as min_duration_seconds,
			MAX(EXTRACT(EPOCH FROM duration)) as max_duration_seconds,
			COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_count,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_count,
			AVG(resource_usage->>'cpu_percent')::float as avg_cpu_usage,
			AVG(resource_usage->>'memory_mb')::float as avg_memory_usage
		FROM process_execution_metrics
		WHERE tenant_id = $1
			AND workflow_type = $2
			AND created_at >= $3
		GROUP BY step_name, step_type
		ORDER BY execution_count DESC, avg_duration_seconds DESC
	`

	rows, err := h.service.db.Query(query, tenantID, workflowType, time.Now().Add(-timeWindow))
	if err != nil {
		log.Printf("Failed to query workflow metrics: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve workflow metrics"})
		return
	}
	defer rows.Close()

	var metrics []map[string]interface{}
	for rows.Next() {
		var stepName, stepType string
		var executionCount int
		var avgDuration, minDuration, maxDuration *float64
		var failedCount, completedCount int
		var avgCpuUsage, avgMemoryUsage *float64

		err := rows.Scan(&stepName, &stepType, &executionCount, &avgDuration, &minDuration, &maxDuration,
			&failedCount, &completedCount, &avgCpuUsage, &avgMemoryUsage)
		if err != nil {
			continue
		}

		metric := map[string]interface{}{
			"step_name":        stepName,
			"step_type":        stepType,
			"execution_count":  executionCount,
			"completed_count":  completedCount,
			"failed_count":     failedCount,
			"success_rate":     float64(completedCount) / float64(executionCount),
			"avg_duration_sec": avgDuration,
			"min_duration_sec": minDuration,
			"max_duration_sec": maxDuration,
			"avg_cpu_usage":    avgCpuUsage,
			"avg_memory_usage": avgMemoryUsage,
		}
		metrics = append(metrics, metric)
	}

	response := map[string]interface{}{
		"workflow_type":    workflowType,
		"tenant_id":        tenantID,
		"time_window_days": timeWindow.Hours() / 24,
		"metrics":          metrics,
		"total_steps":      len(metrics),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetBottlenecks analyzes and returns workflow bottlenecks
func (h *ProcessAnalyticsHandler) GetBottlenecks(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing X-Tenant-ID header"})
		return
	}

	workflowType := chi.URLParam(r, "workflowType")
	timeWindowStr := r.URL.Query().Get("timeWindow")

	// Default to 7 days if not specified
	timeWindow := 7 * 24 * time.Hour
	if timeWindowStr != "" {
		if days, err := strconv.Atoi(timeWindowStr); err == nil && days > 0 {
			timeWindow = time.Duration(days) * 24 * time.Hour
		}
	}

	bottlenecks, err := h.service.AnalyzeBottlenecks(r.Context(), tenantID, workflowType, timeWindow)
	if err != nil {
		log.Printf("Failed to analyze bottlenecks: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to analyze bottlenecks"})
		return
	}

	response := map[string]interface{}{
		"workflow_type":     workflowType,
		"tenant_id":         tenantID,
		"time_window_days":  timeWindow.Hours() / 24,
		"bottlenecks":       bottlenecks,
		"total_bottlenecks": len(bottlenecks),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetOptimizationRecommendations generates and returns optimization recommendations
func (h *ProcessAnalyticsHandler) GetOptimizationRecommendations(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing X-Tenant-ID header"})
		return
	}

	workflowType := chi.URLParam(r, "workflowType")
	timeWindowStr := r.URL.Query().Get("timeWindow")

	// Default to 7 days if not specified
	timeWindow := 7 * 24 * time.Hour
	if timeWindowStr != "" {
		if days, err := strconv.Atoi(timeWindowStr); err == nil && days > 0 {
			timeWindow = time.Duration(days) * 24 * time.Hour
		}
	}

	recommendations, err := h.service.GenerateOptimizationRecommendations(r.Context(), tenantID)
	if err != nil {
		log.Printf("Failed to generate optimization recommendations: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to generate recommendations"})
		return
	}

	response := map[string]interface{}{
		"workflow_type":         workflowType,
		"tenant_id":             tenantID,
		"time_window_days":      timeWindow.Hours() / 24,
		"recommendations":       recommendations,
		"total_recommendations": len(recommendations),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// RecordWorkflowStep records metrics for a workflow step execution
func (h *ProcessAnalyticsHandler) RecordWorkflowStep(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing X-Tenant-ID header"})
		return
	}

	var payload struct {
		WorkflowID    string                 `json:"workflow_id"`
		WorkflowType  string                 `json:"workflow_type"`
		StepName      string                 `json:"step_name"`
		StepType      string                 `json:"step_type"`
		StartTime     time.Time              `json:"start_time"`
		EndTime       *time.Time             `json:"end_time,omitempty"`
		Status        string                 `json:"status"`
		ErrorMessage  *string                `json:"error_message,omitempty"`
		ResourceUsage map[string]interface{} `json:"resource_usage,omitempty"`
		Metadata      map[string]interface{} `json:"metadata,omitempty"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON payload"})
		return
	}

	metrics := &ProcessExecutionMetrics{
		WorkflowID:    payload.WorkflowID,
		WorkflowType:  payload.WorkflowType,
		TenantID:      tenantID,
		StepName:      payload.StepName,
		StepType:      payload.StepType,
		StartTime:     payload.StartTime,
		EndTime:       payload.EndTime,
		Status:        payload.Status,
		ErrorMessage:  payload.ErrorMessage,
		ResourceUsage: payload.ResourceUsage,
		Metadata:      payload.Metadata,
	}

	// Calculate duration if both start and end times are provided
	if payload.EndTime != nil {
		duration := payload.EndTime.Sub(payload.StartTime)
		metrics.Duration = &duration
	}

	err := h.service.RecordWorkflowStep(r.Context(), metrics)
	if err != nil {
		log.Printf("Failed to record workflow step: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to record workflow step"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"status": "recorded"})
}

// GetProcessHealthDashboard provides a comprehensive health overview
func (h *ProcessAnalyticsHandler) GetProcessHealthDashboard(w http.ResponseWriter, r *http.Request) {
	claims := jwtmiddleware.GetClaimsFromContext(r)
	if claims == nil {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	tenantID := claims.TenantID
	if tenantID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Missing X-Tenant-ID header"})
		return
	}

	// Get overall workflow statistics
	workflowStatsQuery := `
		SELECT
			workflow_type,
			COUNT(*) as total_executions,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful_executions,
			AVG(EXTRACT(EPOCH FROM duration)) as avg_duration_seconds,
			MAX(EXTRACT(EPOCH FROM duration)) as max_duration_seconds
		FROM process_execution_metrics
		WHERE tenant_id = $1
			AND created_at >= $2
		GROUP BY workflow_type
		ORDER BY total_executions DESC
	`

	rows, err := h.service.db.Query(workflowStatsQuery, tenantID, time.Now().Add(-24*time.Hour))
	if err != nil {
		log.Printf("Failed to query workflow statistics: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to retrieve dashboard data"})
		return
	}
	defer rows.Close()

	var workflowStats []map[string]interface{}
	for rows.Next() {
		var workflowType string
		var totalExecutions, successfulExecutions int
		var avgDuration, maxDuration *float64

		err := rows.Scan(&workflowType, &totalExecutions, &successfulExecutions, &avgDuration, &maxDuration)
		if err != nil {
			continue
		}

		stat := map[string]interface{}{
			"workflow_type":         workflowType,
			"total_executions":      totalExecutions,
			"successful_executions": successfulExecutions,
			"success_rate":          float64(successfulExecutions) / float64(totalExecutions),
			"avg_duration_seconds":  avgDuration,
			"max_duration_seconds":  maxDuration,
		}
		workflowStats = append(workflowStats, stat)
	}

	// Get active bottlenecks
	bottlenecks, _ := h.service.AnalyzeBottlenecks(r.Context(), tenantID, "", 7*24*time.Hour)
	criticalBottlenecks := 0
	for _, b := range bottlenecks {
		if b.Severity > 0.7 {
			criticalBottlenecks++
		}
	}

	// Get pending recommendations
	recs, _ := h.service.GenerateOptimizationRecommendations(r.Context(), tenantID)
	pendingRecommendations := 0
	for _, r := range recs {
		if r.Status == "pending" {
			pendingRecommendations++
		}
	}

	dashboard := map[string]interface{}{
		"tenant_id":               tenantID,
		"period":                  "24 hours",
		"workflow_statistics":     workflowStats,
		"active_bottlenecks":      len(bottlenecks),
		"critical_bottlenecks":    criticalBottlenecks,
		"pending_recommendations": pendingRecommendations,
		"overall_health_score":    h.calculateHealthScore(workflowStats, bottlenecks),
		"generated_at":            time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(dashboard)
}

// calculateHealthScore computes an overall health score from 0-100
func (h *ProcessAnalyticsHandler) calculateHealthScore(workflowStats []map[string]interface{}, bottlenecks []*ProcessBottleneckAnalysis) float64 {
	if len(workflowStats) == 0 {
		return 100.0 // No data means healthy (or no activity)
	}

	// Calculate average success rate
	totalSuccessRate := 0.0
	for _, stat := range workflowStats {
		if successRate, ok := stat["success_rate"].(float64); ok {
			totalSuccessRate += successRate
		}
	}
	avgSuccessRate := totalSuccessRate / float64(len(workflowStats))

	// Calculate bottleneck impact
	bottleneckImpact := 0.0
	for _, bottleneck := range bottlenecks {
		bottleneckImpact += bottleneck.Severity * 0.1 // Each bottleneck reduces score by up to 10%
	}

	// Health score: 70% success rate, 30% bottleneck impact
	healthScore := (avgSuccessRate * 70) + ((1 - bottleneckImpact) * 30)

	// Ensure score is between 0 and 100
	if healthScore < 0 {
		healthScore = 0
	}
	if healthScore > 100 {
		healthScore = 100
	}

	return healthScore
}
