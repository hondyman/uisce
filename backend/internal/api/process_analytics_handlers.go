package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

// ProcessAnalyticsHandlers provides HTTP handlers for process analytics endpoints
type ProcessAnalyticsHandlers struct {
	db *sqlx.DB
}

// NewProcessAnalyticsHandlers creates a new instance of analytics handlers
func NewProcessAnalyticsHandlers(db *sqlx.DB) *ProcessAnalyticsHandlers {
	return &ProcessAnalyticsHandlers{db: db}
}

// ============================================================================
// REQUEST/RESPONSE TYPES
// ============================================================================

type ProcessMetric struct {
	ID            string                 `json:"id" db:"id"`
	WorkflowID    string                 `json:"workflow_id" db:"workflow_id"`
	WorkflowType  string                 `json:"workflow_type" db:"workflow_type"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	StepName      string                 `json:"step_name" db:"step_name"`
	StepType      string                 `json:"step_type" db:"step_type"`
	StartTime     time.Time              `json:"start_time" db:"start_time"`
	EndTime       *time.Time             `json:"end_time" db:"end_time"`
	Duration      *string                `json:"duration" db:"duration"` // PostgreSQL interval as string
	Status        string                 `json:"status" db:"status"`
	ErrorMessage  *string                `json:"error_message" db:"error_message"`
	ResourceUsage map[string]interface{} `json:"resource_usage" db:"resource_usage"`
	Metadata      map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
}

type BottleneckAnalysis struct {
	ID             string    `json:"id" db:"id"`
	WorkflowType   string    `json:"workflow_type" db:"workflow_type"`
	StepName       string    `json:"step_name" db:"step_name"`
	TenantID       string    `json:"tenant_id" db:"tenant_id"`
	BottleneckType string    `json:"bottleneck_type" db:"bottleneck_type"`
	Severity       float64   `json:"severity" db:"severity"`
	AvgDuration    string    `json:"avg_duration" db:"avg_duration"`
	FailureRate    float64   `json:"failure_rate" db:"failure_rate"`
	Recommendation string    `json:"recommendation" db:"recommendation"`
	Confidence     float64   `json:"confidence" db:"confidence"`
	DetectedAt     time.Time `json:"detected_at" db:"detected_at"`
	LastAnalyzedAt time.Time `json:"last_analyzed_at" db:"last_analyzed_at"`
}

type OptimizationRecommendation struct {
	ID             string                 `json:"id" db:"id"`
	WorkflowType   string                 `json:"workflow_type" db:"workflow_type"`
	TenantID       string                 `json:"tenant_id" db:"tenant_id"`
	Title          string                 `json:"title" db:"title"`
	Description    string                 `json:"description" db:"description"`
	Priority       string                 `json:"priority" db:"priority"`
	ExpectedImpact float64                `json:"expected_impact" db:"expected_impact"`
	Implementation map[string]interface{} `json:"implementation" db:"implementation"`
	Status         string                 `json:"status" db:"status"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	ImplementedAt  *time.Time             `json:"implemented_at" db:"implemented_at"`
}

type ProcessDashboardStats struct {
	TotalWorkflows       int                  `json:"total_workflows"`
	ActiveWorkflows      int                  `json:"active_workflows"`
	CompletedWorkflows   int                  `json:"completed_workflows"`
	FailedWorkflows      int                  `json:"failed_workflows"`
	AvgDurationMinutes   float64              `json:"avg_duration_minutes"`
	SuccessRate          float64              `json:"success_rate"`
	ActiveBottlenecks    int                  `json:"active_bottlenecks"`
	PendingOptimizations int                  `json:"pending_optimizations"`
	TrendData            []TrendDataPoint     `json:"trend_data"`
	TopBottlenecks       []BottleneckAnalysis `json:"top_bottlenecks"`
}

type TrendDataPoint struct {
	Date           string  `json:"date"`
	TotalWorkflows int     `json:"total_workflows"`
	SuccessRate    float64 `json:"success_rate"`
	AvgDuration    float64 `json:"avg_duration"`
}

type StepPerformance struct {
	StepName       string  `json:"step_name"`
	StepType       string  `json:"step_type"`
	ExecutionCount int     `json:"execution_count"`
	AvgDuration    float64 `json:"avg_duration_minutes"`
	SuccessRate    float64 `json:"success_rate"`
	IsBottleneck   bool    `json:"is_bottleneck"`
}

type PredictedDuration struct {
	WorkflowType       string  `json:"workflow_type"`
	PredictedMinutes   float64 `json:"predicted_minutes"`
	ConfidenceInterval struct {
		Lower float64 `json:"lower"`
		Upper float64 `json:"upper"`
	} `json:"confidence_interval"`
	Factors []PredictionFactor `json:"factors"`
}

type PredictionFactor struct {
	Name   string  `json:"name"`
	Impact float64 `json:"impact"`
}

// ============================================================================
// ROUTE HANDLERS
// ============================================================================

// GetDashboardStats retrieves overall process analytics dashboard statistics
func (h *ProcessAnalyticsHandlers) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	ctx := r.Context()
	stats, err := h.calculateDashboardStats(ctx, tenantID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, stats)
}

// GetBottlenecks retrieves identified bottlenecks with filtering
func (h *ProcessAnalyticsHandlers) GetBottlenecks(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	workflowType := r.URL.Query().Get("workflow_type")

	if tenantID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	ctx := r.Context()

	query := `
		SELECT id, workflow_type, step_name, tenant_id, bottleneck_type, severity, 
		       avg_duration, failure_rate, recommendation, confidence, detected_at, last_analyzed_at
		FROM process_bottleneck_analysis
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}

	if workflowType != "" {
		query += " AND workflow_type = $2"
		args = append(args, workflowType)
	}

	query += " ORDER BY severity DESC, detected_at DESC LIMIT 50"

	var bottlenecks []BottleneckAnalysis
	err := h.db.SelectContext(ctx, &bottlenecks, query, args...)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, bottlenecks)
}

// GetOptimizationRecommendations retrieves AI-generated recommendations
func (h *ProcessAnalyticsHandlers) GetOptimizationRecommendations(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	status := r.URL.Query().Get("status") // pending, implemented, rejected, in_progress

	if tenantID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	ctx := r.Context()

	query := `
		SELECT id, workflow_type, tenant_id, title, description, priority, 
		       expected_impact, implementation, status, created_at, implemented_at
		FROM process_optimization_recommendations
		WHERE tenant_id = $1
	`
	args := []interface{}{tenantID}

	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}

	query += " ORDER BY priority DESC, expected_impact DESC, created_at DESC LIMIT 50"

	var recommendations []OptimizationRecommendation
	err := h.db.SelectContext(ctx, &recommendations, query, args...)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, recommendations)
}

// GetStepPerformance retrieves detailed performance metrics for individual steps
func (h *ProcessAnalyticsHandlers) GetStepPerformance(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	workflowType := r.URL.Query().Get("workflow_type")

	if tenantID == "" || workflowType == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id and workflow_type required"})
		return
	}

	ctx := r.Context()
	performance, err := h.calculateStepPerformance(ctx, tenantID, workflowType)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, performance)
}

// PredictWorkflowDuration uses ML to predict workflow completion time
func (h *ProcessAnalyticsHandlers) PredictWorkflowDuration(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	workflowType := r.URL.Query().Get("workflow_type")

	if tenantID == "" || workflowType == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id and workflow_type required"})
		return
	}

	ctx := r.Context()
	prediction, err := h.predictDuration(ctx, tenantID, workflowType)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, prediction)
}

// RunBottleneckAnalysis triggers ML-based bottleneck detection
func (h *ProcessAnalyticsHandlers) RunBottleneckAnalysis(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")

	if tenantID == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "tenant_id required"})
		return
	}

	ctx := r.Context()
	err := h.detectBottlenecks(ctx, tenantID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Bottleneck analysis completed"})
}

// ============================================================================
// ANALYTICS CALCULATION FUNCTIONS
// ============================================================================

func (h *ProcessAnalyticsHandlers) calculateDashboardStats(ctx context.Context, tenantID string) (*ProcessDashboardStats, error) {
	stats := &ProcessDashboardStats{}

	// Get overall workflow counts
	err := h.db.GetContext(ctx, stats, `
		SELECT 
			COUNT(DISTINCT workflow_id) as total_workflows,
			COUNT(DISTINCT CASE WHEN status = 'running' THEN workflow_id END) as active_workflows,
			COUNT(DISTINCT CASE WHEN status = 'completed' THEN workflow_id END) as completed_workflows,
			COUNT(DISTINCT CASE WHEN status = 'failed' THEN workflow_id END) as failed_workflows
		FROM process_execution_metrics
		WHERE tenant_id = $1 AND created_at > NOW() - INTERVAL '30 days'
	`, tenantID)
	if err != nil {
		return nil, err
	}

	// Calculate average duration and success rate
	var avgDuration sql.NullFloat64
	var successRate sql.NullFloat64
	err = h.db.QueryRowContext(ctx, `
		WITH workflow_durations AS (
			SELECT 
				workflow_id,
				EXTRACT(EPOCH FROM (MAX(end_time) - MIN(start_time)))/60 as duration_minutes,
				BOOL_AND(status = 'completed') as all_completed
			FROM process_execution_metrics
			WHERE tenant_id = $1 AND end_time IS NOT NULL AND created_at > NOW() - INTERVAL '30 days'
			GROUP BY workflow_id
		)
		SELECT 
			AVG(duration_minutes) as avg_duration,
			AVG(CASE WHEN all_completed THEN 1.0 ELSE 0.0 END) as success_rate
		FROM workflow_durations
	`, tenantID).Scan(&avgDuration, &successRate)

	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	if avgDuration.Valid {
		stats.AvgDurationMinutes = avgDuration.Float64
	}
	if successRate.Valid {
		stats.SuccessRate = successRate.Float64
	}

	// Get active bottlenecks count
	err = h.db.GetContext(ctx, &stats.ActiveBottlenecks, `
		SELECT COUNT(*) FROM process_bottleneck_analysis
		WHERE tenant_id = $1 AND detected_at > NOW() - INTERVAL '7 days'
	`, tenantID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get pending optimizations count
	err = h.db.GetContext(ctx, &stats.PendingOptimizations, `
		SELECT COUNT(*) FROM process_optimization_recommendations
		WHERE tenant_id = $1 AND status = 'pending'
	`, tenantID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Get trend data (last 14 days)
	rows, err := h.db.QueryContext(ctx, `
		WITH daily_metrics AS (
			SELECT 
				DATE(created_at) as date,
				COUNT(DISTINCT workflow_id) as total_workflows,
				COUNT(DISTINCT CASE WHEN status = 'completed' THEN workflow_id END) as completed_workflows,
				AVG(EXTRACT(EPOCH FROM duration)/60) as avg_duration
			FROM process_execution_metrics
			WHERE tenant_id = $1 AND created_at > NOW() - INTERVAL '14 days'
			GROUP BY DATE(created_at)
		)
		SELECT 
			date,
			total_workflows,
			CASE WHEN total_workflows > 0 THEN completed_workflows::float / total_workflows ELSE 0 END as success_rate,
			COALESCE(avg_duration, 0) as avg_duration
		FROM daily_metrics
		ORDER BY date ASC
	`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stats.TrendData = []TrendDataPoint{}
	for rows.Next() {
		var point TrendDataPoint
		var date time.Time
		err := rows.Scan(&date, &point.TotalWorkflows, &point.SuccessRate, &point.AvgDuration)
		if err != nil {
			return nil, err
		}
		point.Date = date.Format("2006-01-02")
		stats.TrendData = append(stats.TrendData, point)
	}

	// Get top bottlenecks
	err = h.db.SelectContext(ctx, &stats.TopBottlenecks, `
		SELECT id, workflow_type, step_name, tenant_id, bottleneck_type, severity, 
		       avg_duration, failure_rate, recommendation, confidence, detected_at, last_analyzed_at
		FROM process_bottleneck_analysis
		WHERE tenant_id = $1
		ORDER BY severity DESC
		LIMIT 5
	`, tenantID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return stats, nil
}

func (h *ProcessAnalyticsHandlers) calculateStepPerformance(ctx context.Context, tenantID, workflowType string) ([]StepPerformance, error) {
	var performance []StepPerformance

	rows, err := h.db.QueryContext(ctx, `
		SELECT 
			step_name,
			step_type,
			COUNT(*) as execution_count,
			AVG(EXTRACT(EPOCH FROM duration)/60) as avg_duration_minutes,
			AVG(CASE WHEN status = 'completed' THEN 1.0 ELSE 0.0 END) as success_rate
		FROM process_execution_metrics
		WHERE tenant_id = $1 AND workflow_type = $2 AND duration IS NOT NULL
		GROUP BY step_name, step_type
		ORDER BY avg_duration_minutes DESC
	`, tenantID, workflowType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Get bottleneck steps
	bottleneckSteps := make(map[string]bool)
	bottleneckRows, err := h.db.QueryContext(ctx, `
		SELECT DISTINCT step_name 
		FROM process_bottleneck_analysis
		WHERE tenant_id = $1 AND workflow_type = $2
	`, tenantID, workflowType)
	if err == nil {
		defer bottleneckRows.Close()
		for bottleneckRows.Next() {
			var stepName string
			if err := bottleneckRows.Scan(&stepName); err == nil {
				bottleneckSteps[stepName] = true
			}
		}
	}

	for rows.Next() {
		var perf StepPerformance
		err := rows.Scan(&perf.StepName, &perf.StepType, &perf.ExecutionCount, &perf.AvgDuration, &perf.SuccessRate)
		if err != nil {
			return nil, err
		}
		perf.IsBottleneck = bottleneckSteps[perf.StepName]
		performance = append(performance, perf)
	}

	return performance, nil
}

// ============================================================================
// PREDICTIVE ANALYTICS - ML ALGORITHMS
// ============================================================================

func (h *ProcessAnalyticsHandlers) predictDuration(ctx context.Context, tenantID, workflowType string) (*PredictedDuration, error) {
	// Simple linear regression based on historical data
	var historicalDurations []float64

	rows, err := h.db.QueryContext(ctx, `
		SELECT EXTRACT(EPOCH FROM (MAX(end_time) - MIN(start_time)))/60 as duration_minutes
		FROM process_execution_metrics
		WHERE tenant_id = $1 AND workflow_type = $2 AND end_time IS NOT NULL
		GROUP BY workflow_id
		ORDER BY MAX(created_at) DESC
		LIMIT 100
	`, tenantID, workflowType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var duration float64
		if err := rows.Scan(&duration); err == nil {
			historicalDurations = append(historicalDurations, duration)
		}
	}

	if len(historicalDurations) == 0 {
		return &PredictedDuration{
			WorkflowType:     workflowType,
			PredictedMinutes: 0,
			Factors:          []PredictionFactor{},
		}, nil
	}

	// Calculate mean and standard deviation
	mean := calculateMean(historicalDurations)
	stdDev := calculateStdDev(historicalDurations, mean)

	// Apply weighted moving average with recent data having more weight
	weightedAvg := mean
	if len(historicalDurations) >= 5 {
		recent := historicalDurations[:5]
		weightedAvg = calculateMean(recent)
	}

	prediction := &PredictedDuration{
		WorkflowType:     workflowType,
		PredictedMinutes: weightedAvg,
		Factors:          []PredictionFactor{},
	}

	// 95% confidence interval
	prediction.ConfidenceInterval.Lower = math.Max(0, weightedAvg-(1.96*stdDev))
	prediction.ConfidenceInterval.Upper = weightedAvg + (1.96 * stdDev)

	// Identify prediction factors
	factors, _ := h.identifyPredictionFactors(ctx, tenantID, workflowType)
	prediction.Factors = factors

	return prediction, nil
}

func (h *ProcessAnalyticsHandlers) identifyPredictionFactors(ctx context.Context, tenantID, workflowType string) ([]PredictionFactor, error) {
	factors := []PredictionFactor{}

	// Factor 1: Parallel execution
	var parallelSteps int
	h.db.GetContext(ctx, &parallelSteps, `
		SELECT COUNT(DISTINCT step_name)
		FROM process_execution_metrics
		WHERE tenant_id = $1 AND workflow_type = $2 
			AND metadata->>'execution_mode' = 'parallel'
		LIMIT 1
	`, tenantID, workflowType)

	if parallelSteps > 0 {
		factors = append(factors, PredictionFactor{
			Name:   "Parallel Execution",
			Impact: -0.3, // Reduces time by 30%
		})
	}

	// Factor 2: Historical bottlenecks
	var bottleneckCount int
	h.db.GetContext(ctx, &bottleneckCount, `
		SELECT COUNT(*) FROM process_bottleneck_analysis
		WHERE tenant_id = $1 AND workflow_type = $2 AND severity > 0.6
	`, tenantID, workflowType)

	if bottleneckCount > 0 {
		factors = append(factors, PredictionFactor{
			Name:   fmt.Sprintf("%d Active Bottlenecks", bottleneckCount),
			Impact: 0.25 * float64(bottleneckCount), // Each bottleneck adds 25% time
		})
	}

	// Factor 3: Time of day
	hour := time.Now().Hour()
	if hour >= 9 && hour <= 17 {
		factors = append(factors, PredictionFactor{
			Name:   "Peak Business Hours",
			Impact: 0.15, // 15% slower during peak hours
		})
	}

	return factors, nil
}

// detectBottlenecks performs ML-based bottleneck detection
func (h *ProcessAnalyticsHandlers) detectBottlenecks(ctx context.Context, tenantID string) error {
	// Analyze each workflow type and step combination
	rows, err := h.db.QueryContext(ctx, `
		SELECT 
			workflow_type,
			step_name,
			step_type,
			AVG(EXTRACT(EPOCH FROM duration)) as avg_duration_seconds,
			AVG(CASE WHEN status = 'failed' THEN 1.0 ELSE 0.0 END) as failure_rate,
			COUNT(*) as sample_size
		FROM process_execution_metrics
		WHERE tenant_id = $1 AND duration IS NOT NULL AND created_at > NOW() - INTERVAL '7 days'
		GROUP BY workflow_type, step_name, step_type
		HAVING COUNT(*) >= 5
	`, tenantID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var workflowType, stepName, stepType string
		var avgDuration, failureRate float64
		var sampleSize int

		err := rows.Scan(&workflowType, &stepName, &stepType, &avgDuration, &failureRate, &sampleSize)
		if err != nil {
			continue
		}

		// Detect duration bottlenecks (steps taking > 80th percentile)
		var p80Duration float64
		h.db.GetContext(ctx, &p80Duration, `
			SELECT PERCENTILE_CONT(0.8) WITHIN GROUP (ORDER BY EXTRACT(EPOCH FROM duration))
			FROM process_execution_metrics
			WHERE tenant_id = $1 AND workflow_type = $2 AND step_type = $3 AND duration IS NOT NULL
		`, tenantID, workflowType, stepType)

		isDurationBottleneck := avgDuration > p80Duration && p80Duration > 0
		isFailureBottleneck := failureRate > 0.1 // More than 10% failure rate

		if isDurationBottleneck || isFailureBottleneck {
			severity := 0.0
			bottleneckType := ""
			recommendation := ""

			if isDurationBottleneck {
				severity = math.Min(1.0, (avgDuration/p80Duration)-0.8) // 0.8 to 1.0 scale
				bottleneckType = "duration"
				recommendation = fmt.Sprintf("Consider parallelizing this step or optimizing the underlying operation. Current duration (%.1f min) is %.0f%% above average.",
					avgDuration/60, ((avgDuration/p80Duration)-1)*100)
			}

			if isFailureBottleneck {
				severity = math.Max(severity, failureRate) // Use failure rate as severity
				bottleneckType = "failure_rate"
				recommendation = fmt.Sprintf("High failure rate detected (%.1f%%). Review error logs and add retry logic or validation.", failureRate*100)
			}

			// Insert or update bottleneck analysis
			_, err = h.db.ExecContext(ctx, `
				INSERT INTO process_bottleneck_analysis 
					(workflow_type, step_name, tenant_id, bottleneck_type, severity, avg_duration, failure_rate, recommendation, confidence, detected_at, last_analyzed_at)
				VALUES 
					($1, $2, $3, $4, $5, make_interval(secs => $6), $7, $8, $9, NOW(), NOW())
				ON CONFLICT (workflow_type, step_name, tenant_id, bottleneck_type)
				DO UPDATE SET 
					severity = $5,
					avg_duration = make_interval(secs => $6),
					failure_rate = $7,
					recommendation = $8,
					confidence = $9,
					last_analyzed_at = NOW()
				WHERE process_bottleneck_analysis.workflow_type = $1
			`, workflowType, stepName, tenantID, bottleneckType, severity, avgDuration, failureRate, recommendation, 0.85)

			if err != nil {
				fmt.Printf("Error inserting bottleneck: %v\n", err)
			}

			// Generate optimization recommendation
			h.generateOptimizationRecommendation(ctx, tenantID, workflowType, stepName, bottleneckType, severity)
		}
	}

	return nil
}

func (h *ProcessAnalyticsHandlers) generateOptimizationRecommendation(ctx context.Context, tenantID, workflowType, stepName, bottleneckType string, severity float64) {
	priority := "low"
	if severity > 0.7 {
		priority = "high"
	} else if severity > 0.4 {
		priority = "medium"
	}

	title := fmt.Sprintf("Optimize %s in %s workflow", stepName, workflowType)
	description := ""
	implementation := map[string]interface{}{}

	if bottleneckType == "duration" {
		description = fmt.Sprintf("The '%s' step is taking significantly longer than expected. Consider implementing parallel execution or caching.", stepName)
		implementation = map[string]interface{}{
			"type":                 "parallel_execution",
			"steps":                []string{stepName},
			"expected_improvement": "30-40% reduction in duration",
		}
	} else {
		description = fmt.Sprintf("The '%s' step has a high failure rate. Review error patterns and add validation or retry logic.", stepName)
		implementation = map[string]interface{}{
			"type":        "add_retry_logic",
			"max_retries": 3,
			"backoff":     "exponential",
		}
	}

	// Check if recommendation already exists
	var existingID string
	err := h.db.GetContext(ctx, &existingID, `
		SELECT id FROM process_optimization_recommendations
		WHERE tenant_id = $1 AND workflow_type = $2 AND title = $3 AND status = 'pending'
		LIMIT 1
	`, tenantID, workflowType, title)

	if err == sql.ErrNoRows {
		// Insert new recommendation
		implJSON, _ := json.Marshal(implementation)
		_, err = h.db.ExecContext(ctx, `
			INSERT INTO process_optimization_recommendations 
				(workflow_type, tenant_id, title, description, priority, expected_impact, implementation, status, created_at, updated_at)
			VALUES 
				($1, $2, $3, $4, $5, $6, $7, 'pending', NOW(), NOW())
		`, workflowType, tenantID, title, description, priority, severity, implJSON)

		if err != nil {
			fmt.Printf("Error inserting optimization recommendation: %v\n", err)
		}
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	return math.Sqrt(variance / float64(len(values)))
}

// RegisterRoutes registers all process analytics routes
func (h *ProcessAnalyticsHandlers) RegisterRoutes(r chi.Router) {
	r.Route("/api/process-analytics", func(r chi.Router) {
		r.Get("/dashboard", h.GetDashboardStats)
		r.Get("/bottlenecks", h.GetBottlenecks)
		r.Get("/recommendations", h.GetOptimizationRecommendations)
		r.Get("/step-performance", h.GetStepPerformance)
		r.Get("/predict-duration", h.PredictWorkflowDuration)
		r.Post("/analyze-bottlenecks", h.RunBottleneckAnalysis)
	})
}
