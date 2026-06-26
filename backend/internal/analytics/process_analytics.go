package analytics

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ProcessExecutionMetrics represents metrics collected during workflow execution
type ProcessExecutionMetrics struct {
	ID            string                 `json:"id" db:"id"`
	WorkflowID    string                 `json:"workflow_id" db:"workflow_id"`
	WorkflowType  string                 `json:"workflow_type" db:"workflow_type"`
	TenantID      string                 `json:"tenant_id" db:"tenant_id"`
	StepName      string                 `json:"step_name" db:"step_name"`
	StepType      string                 `json:"step_type" db:"step_type"`
	StartTime     time.Time              `json:"start_time" db:"start_time"`
	EndTime       *time.Time             `json:"end_time,omitempty" db:"end_time"`
	Duration      *time.Duration         `json:"duration,omitempty" db:"duration"`
	Status        string                 `json:"status" db:"status"` // running, completed, failed, timeout
	ErrorMessage  *string                `json:"error_message,omitempty" db:"error_message"`
	ResourceUsage map[string]interface{} `json:"resource_usage" db:"resource_usage"`
	Metadata      map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt     time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at" db:"updated_at"`
}

// ProcessBottleneckAnalysis represents identified bottlenecks in processes
type ProcessBottleneckAnalysis struct {
	ID             string        `json:"id" db:"id"`
	WorkflowType   string        `json:"workflow_type" db:"workflow_type"`
	StepName       string        `json:"step_name" db:"step_name"`
	TenantID       string        `json:"tenant_id" db:"tenant_id"`
	BottleneckType string        `json:"bottleneck_type" db:"bottleneck_type"` // duration, failure_rate, resource_contention
	Severity       float64       `json:"severity" db:"severity"`               // 0-1 scale
	AvgDuration    time.Duration `json:"avg_duration" db:"avg_duration"`
	FailureRate    float64       `json:"failure_rate" db:"failure_rate"`
	Recommendation string        `json:"recommendation" db:"recommendation"`
	Confidence     float64       `json:"confidence" db:"confidence"`
	DetectedAt     time.Time     `json:"detected_at" db:"detected_at"`
	LastAnalyzedAt time.Time     `json:"last_analyzed_at" db:"last_analyzed_at"`
}

// ProcessOptimizationRecommendation represents AI-generated optimization suggestions
type ProcessOptimizationRecommendation struct {
	ID             string                 `json:"id" db:"id"`
	WorkflowType   string                 `json:"workflow_type" db:"workflow_type"`
	TenantID       string                 `json:"tenant_id" db:"tenant_id"`
	Title          string                 `json:"title" db:"title"`
	Description    string                 `json:"description" db:"description"`
	Priority       string                 `json:"priority" db:"priority"` // high, medium, low
	ExpectedImpact float64                `json:"expected_impact" db:"expected_impact"`
	Implementation map[string]interface{} `json:"implementation" db:"implementation"`
	Status         string                 `json:"status" db:"status"` // pending, implemented, rejected
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	ImplementedAt  *time.Time             `json:"implemented_at,omitempty" db:"implemented_at"`
}

// ProcessAnalyticsService handles process analytics and optimization
type ProcessAnalyticsService struct {
	db *sqlx.DB
}

// NewProcessAnalyticsService creates a new process analytics service
func NewProcessAnalyticsService(db *sqlx.DB) *ProcessAnalyticsService {
	return &ProcessAnalyticsService{
		db: db,
	}
}

// RecordWorkflowStep records metrics for a workflow step execution
func (s *ProcessAnalyticsService) RecordWorkflowStep(ctx context.Context, metrics *ProcessExecutionMetrics) error {
	if metrics.ID == "" {
		metrics.ID = uuid.New().String()
	}
	metrics.CreatedAt = time.Now()
	metrics.UpdatedAt = time.Now()

	query := `
		INSERT INTO process_execution_metrics (
			id, workflow_id, workflow_type, tenant_id, step_name, step_type,
			start_time, end_time, duration, status, error_message,
			resource_usage, metadata, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	_, err := s.db.ExecContext(ctx, query,
		metrics.ID, metrics.WorkflowID, metrics.WorkflowType, metrics.TenantID,
		metrics.StepName, metrics.StepType, metrics.StartTime, metrics.EndTime,
		metrics.Duration, metrics.Status, metrics.ErrorMessage,
		metrics.ResourceUsage, metrics.Metadata, metrics.CreatedAt, metrics.UpdatedAt,
	)

	return err
}

// UpdateWorkflowStep updates completion metrics for a workflow step
func (s *ProcessAnalyticsService) UpdateWorkflowStep(ctx context.Context, workflowID, stepName string, endTime time.Time, status string, errorMessage *string) error {
	duration := endTime.Sub(endTime) // This would need to be calculated from start time

	query := `
		UPDATE process_execution_metrics
		SET end_time = $1, duration = $2, status = $3, error_message = $4, updated_at = $5
		WHERE workflow_id = $6 AND step_name = $7 AND end_time IS NULL
	`

	_, err := s.db.ExecContext(ctx, query, endTime, duration, status, errorMessage, time.Now(), workflowID, stepName)
	return err
}

// AnalyzeBottlenecks analyzes workflow data to identify bottlenecks
func (s *ProcessAnalyticsService) AnalyzeBottlenecks(ctx context.Context, tenantID string, workflowType string, timeWindow time.Duration) ([]*ProcessBottleneckAnalysis, error) {
	query := `
		WITH step_stats AS (
			SELECT
				step_name,
				step_type,
				AVG(EXTRACT(EPOCH FROM duration)) as avg_duration_seconds,
				COUNT(*) as total_executions,
				COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_count,
				AVG(resource_usage->>'cpu_percent')::float as avg_cpu_usage,
				AVG(resource_usage->>'memory_mb')::float as avg_memory_usage
			FROM process_execution_metrics
			WHERE tenant_id = $1
				AND workflow_type = $2
				AND created_at >= $3
				AND status IN ('completed', 'failed')
			GROUP BY step_name, step_type
		)
		SELECT
			step_name,
			step_type,
			avg_duration_seconds,
			failed_count::float / total_executions as failure_rate,
			avg_cpu_usage,
			avg_memory_usage
		FROM step_stats
		WHERE avg_duration_seconds > (
			SELECT percentile_cont(0.75) WITHIN GROUP (ORDER BY avg_duration_seconds)
			FROM step_stats
		) * 1.5
		OR failed_count::float / total_executions > 0.1
		ORDER BY avg_duration_seconds DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, workflowType, time.Now().Add(-timeWindow))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var bottlenecks []*ProcessBottleneckAnalysis
	for rows.Next() {
		var stepName, stepType string
		var avgDurationSeconds, failureRate, avgCpuUsage, avgMemoryUsage float64

		err := rows.Scan(&stepName, &stepType, &avgDurationSeconds, &failureRate, &avgCpuUsage, &avgMemoryUsage)
		if err != nil {
			continue
		}

		bottleneck := &ProcessBottleneckAnalysis{
			ID:             uuid.New().String(),
			WorkflowType:   workflowType,
			StepName:       stepName,
			TenantID:       tenantID,
			BottleneckType: s.determineBottleneckType(avgDurationSeconds, failureRate, avgCpuUsage, avgMemoryUsage),
			Severity:       s.calculateSeverity(avgDurationSeconds, failureRate, avgCpuUsage, avgMemoryUsage),
			AvgDuration:    time.Duration(avgDurationSeconds * float64(time.Second)),
			FailureRate:    failureRate,
			Recommendation: s.generateRecommendation(avgDurationSeconds, failureRate, avgCpuUsage, avgMemoryUsage),
			Confidence:     0.85, // Placeholder confidence score
			DetectedAt:     time.Now(),
			LastAnalyzedAt: time.Now(),
		}

		bottlenecks = append(bottlenecks, bottleneck)
	}

	return bottlenecks, nil
}

// GenerateOptimizationRecommendations creates AI-powered optimization suggestions
func (s *ProcessAnalyticsService) GenerateOptimizationRecommendations(ctx context.Context, tenantID string) ([]*ProcessOptimizationRecommendation, error) {
	// Get bottleneck data
	bottlenecks, err := s.AnalyzeBottlenecks(ctx, tenantID, "", 7*24*time.Hour) // Last 7 days
	if err != nil {
		return nil, err
	}

	var recommendations []*ProcessOptimizationRecommendation

	for _, bottleneck := range bottlenecks {
		if bottleneck.Severity > 0.7 { // High severity bottlenecks
			rec := &ProcessOptimizationRecommendation{
				ID:             uuid.New().String(),
				WorkflowType:   bottleneck.WorkflowType,
				TenantID:       tenantID,
				Title:          s.generateRecommendationTitle(bottleneck),
				Description:    bottleneck.Recommendation,
				Priority:       s.calculatePriority(bottleneck.Severity),
				ExpectedImpact: bottleneck.Severity * 0.3, // Estimated 30% of severity as impact
				Implementation: s.generateImplementationPlan(bottleneck),
				Status:         "pending",
				CreatedAt:      time.Now(),
			}
			recommendations = append(recommendations, rec)
		}
	}

	// Add pattern-based recommendations
	patternRecs, err := s.generatePatternRecommendations(ctx, tenantID)
	if err == nil {
		recommendations = append(recommendations, patternRecs...)
	}

	return recommendations, nil
}

// Helper methods

func (s *ProcessAnalyticsService) determineBottleneckType(avgDuration float64, failureRate float64, avgCpuUsage float64, avgMemoryUsage float64) string {
	if failureRate > 0.2 {
		return "failure_rate"
	}
	if avgCpuUsage > 80 || avgMemoryUsage > 80 {
		return "resource_contention"
	}
	return "duration"
}

func (s *ProcessAnalyticsService) calculateSeverity(avgDuration float64, failureRate float64, avgCpuUsage float64, avgMemoryUsage float64) float64 {
	durationScore := min(avgDuration/300.0, 1.0) // Normalize to 5 minutes max
	failureScore := failureRate
	resourceScore := max(avgCpuUsage/100.0, avgMemoryUsage/100.0)

	return (durationScore*0.4 + failureScore*0.4 + resourceScore*0.2)
}

func (s *ProcessAnalyticsService) generateRecommendation(avgDuration float64, failureRate float64, avgCpuUsage float64, avgMemoryUsage float64) string {
	if failureRate > 0.2 {
		return "High failure rate detected. Consider implementing retry logic, improving error handling, or reviewing input validation."
	}
	if avgCpuUsage > 80 {
		return "High CPU usage detected. Consider optimizing the step logic, implementing caching, or scaling resources."
	}
	if avgMemoryUsage > 80 {
		return "High memory usage detected. Consider implementing streaming processing, optimizing data structures, or increasing memory allocation."
	}
	if avgDuration > 120 {
		return "Step is taking longer than expected. Consider parallelizing operations, optimizing database queries, or implementing caching."
	}
	return "General optimization recommended. Review step implementation for potential improvements."
}

func (s *ProcessAnalyticsService) generateRecommendationTitle(bottleneck *ProcessBottleneckAnalysis) string {
	switch bottleneck.BottleneckType {
	case "failure_rate":
		return "Reduce Failure Rate in " + bottleneck.StepName
	case "resource_contention":
		return "Optimize Resource Usage in " + bottleneck.StepName
	case "duration":
		return "Improve Performance of " + bottleneck.StepName
	default:
		return "Optimize " + bottleneck.StepName
	}
}

func (s *ProcessAnalyticsService) calculatePriority(severity float64) string {
	if severity > 0.8 {
		return "high"
	}
	if severity > 0.6 {
		return "medium"
	}
	return "low"
}

func (s *ProcessAnalyticsService) generateImplementationPlan(bottleneck *ProcessBottleneckAnalysis) map[string]interface{} {
	plan := map[string]interface{}{
		"estimated_effort": "medium",
		"risk_level":       "low",
		"steps": []string{
			"Analyze current implementation",
			"Implement recommended changes",
			"Test in staging environment",
			"Deploy with monitoring",
		},
	}

	switch bottleneck.BottleneckType {
	case "failure_rate":
		plan["steps"] = []string{
			"Add comprehensive error handling",
			"Implement retry mechanism with exponential backoff",
			"Add input validation",
			"Improve logging for debugging",
		}
	case "resource_contention":
		plan["steps"] = []string{
			"Profile resource usage",
			"Optimize algorithms and data structures",
			"Implement caching where appropriate",
			"Consider horizontal scaling",
		}
	case "duration":
		plan["steps"] = []string{
			"Identify performance bottlenecks",
			"Optimize database queries",
			"Implement parallel processing",
			"Add performance monitoring",
		}
	}

	return plan
}

func (s *ProcessAnalyticsService) generatePatternRecommendations(ctx context.Context, tenantID string) ([]*ProcessOptimizationRecommendation, error) {
	// Analyze workflow patterns for optimization opportunities
	query := `
		SELECT
			workflow_type,
			COUNT(*) as execution_count,
			AVG(EXTRACT(EPOCH FROM duration)) as avg_duration,
			COUNT(CASE WHEN status = 'failed' THEN 1 END)::float / COUNT(*) as failure_rate
		FROM process_execution_metrics
		WHERE tenant_id = $1
			AND created_at >= $2
		GROUP BY workflow_type
		HAVING COUNT(*) > 10
		ORDER BY execution_count DESC
		LIMIT 5
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, time.Now().Add(-30*24*time.Hour))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recommendations []*ProcessOptimizationRecommendation
	for rows.Next() {
		var workflowType string
		var executionCount int
		var avgDuration, failureRate float64

		err := rows.Scan(&workflowType, &executionCount, &avgDuration, &failureRate)
		if err != nil {
			continue
		}

		// Generate pattern-based recommendations
		if executionCount > 100 && avgDuration > 60 {
			rec := &ProcessOptimizationRecommendation{
				ID:             uuid.New().String(),
				WorkflowType:   workflowType,
				TenantID:       tenantID,
				Title:          "Implement Workflow Caching for " + workflowType,
				Description:    "High execution frequency detected. Consider implementing caching to reduce processing time.",
				Priority:       "medium",
				ExpectedImpact: 0.25,
				Implementation: map[string]interface{}{
					"estimated_effort": "medium",
					"risk_level":       "low",
					"steps": []string{
						"Identify cacheable data",
						"Implement Redis caching layer",
						"Add cache invalidation logic",
						"Monitor cache hit rates",
					},
				},
				Status:    "pending",
				CreatedAt: time.Now(),
			}
			recommendations = append(recommendations, rec)
		}
	}

	return recommendations, nil
}

// Utility functions
func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
