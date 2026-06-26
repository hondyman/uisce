package benchmarking

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/models"
)

// ============================================================================
// Performance Scoring Service
// ============================================================================

type ScoringService struct {
	db *sql.DB
}

func NewScoringService(db *sql.DB) *ScoringService {
	return &ScoringService{db: db}
}

// ============================================================================
// Main Scoring Function
// ============================================================================

// CalculatePerformanceScore computes the overall performance score and dimension scores
// for a given workflow type based on historical execution data and industry benchmarks.
func (s *ScoringService) CalculatePerformanceScore(
	ctx context.Context,
	tenantID uuid.UUID,
	workflowType string,
	industry string,
) (*models.PerformanceScore, error) {
	// Get workflow metrics
	metrics, err := s.getWorkflowMetrics(ctx, tenantID, workflowType)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow metrics: %w", err)
	}

	// Get industry benchmark
	benchmark, err := s.getIndustryBenchmark(ctx, industry, workflowType)
	if err != nil {
		return nil, fmt.Errorf("failed to get industry benchmark: %w", err)
	}

	// Calculate dimension scores
	efficiencyScore := s.calculateEfficiencyScore(metrics, benchmark)
	qualityScore := s.calculateQualityScore(metrics, benchmark)
	speedScore := s.calculateSpeedScore(metrics, benchmark)
	automationScore := s.calculateAutomationScore(metrics, benchmark)
	complianceScore := s.calculateComplianceScore(metrics)

	// Calculate weighted overall score
	// Weights: Efficiency 25%, Quality 25%, Speed 20%, Automation 15%, Compliance 15%
	overallScore := int(math.Round(
		float64(efficiencyScore)*0.25 +
			float64(qualityScore)*0.25 +
			float64(speedScore)*0.20 +
			float64(automationScore)*0.15 +
			float64(complianceScore)*0.15,
	))

	// Assign grade
	grade := assignGrade(overallScore)

	// Calculate percentile (if we have peer data)
	percentile := s.calculatePercentile(ctx, tenantID, workflowType, overallScore)

	// Create performance score record
	score := &models.PerformanceScore{
		ID:              uuid.New(),
		TenantID:        tenantID,
		WorkflowType:    workflowType,
		OverallScore:    overallScore,
		Grade:           grade,
		Percentile:      &percentile,
		EfficiencyScore: efficiencyScore,
		QualityScore:    qualityScore,
		SpeedScore:      speedScore,
		AutomationScore: automationScore,
		ComplianceScore: complianceScore,
		Industry:        &industry,
		SampleSize:      metrics.SampleSize,
		ConfidenceLevel: 0.95,
		CalculatedAt:    time.Now(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Save to database
	if err := s.savePerformanceScore(ctx, score); err != nil {
		return nil, fmt.Errorf("failed to save performance score: %w", err)
	}

	return score, nil
}

// ============================================================================
// Dimension Score Calculations
// ============================================================================

// calculateEfficiencyScore measures resource utilization and cost-effectiveness
// Score = 100 - (cost_vs_benchmark_percent * 0.5 + resource_utilization_gap * 0.5)
func (s *ScoringService) calculateEfficiencyScore(metrics *WorkflowMetrics, benchmark *models.IndustryBenchmark) int {
	if benchmark.MedianCostPerProcess == nil || *benchmark.MedianCostPerProcess == 0 {
		return 70 // Default score if no benchmark data
	}

	// Cost comparison (lower is better)
	costRatio := metrics.AvgCostPerProcess / *benchmark.MedianCostPerProcess
	costScore := 100.0
	if costRatio > 1.0 {
		// Over benchmark - penalize
		costScore = math.Max(0, 100.0-((costRatio-1.0)*50.0))
	} else {
		// Under benchmark - bonus
		costScore = math.Min(100.0, 100.0+((1.0-costRatio)*20.0))
	}

	// Resource utilization (higher is better)
	resourceScore := metrics.ResourceUtilization * 100.0

	// Weighted combination
	score := int(math.Round(costScore*0.5 + resourceScore*0.5))
	return clampScore(score)
}

// calculateQualityScore measures success rate, error rate, and rework
// Score = success_rate * 0.6 + (100 - error_rate) * 0.3 + (100 - rework_rate) * 0.1
func (s *ScoringService) calculateQualityScore(metrics *WorkflowMetrics, benchmark *models.IndustryBenchmark) int {
	successScore := metrics.SuccessRate * 100.0
	errorScore := (1.0 - metrics.ErrorRate) * 100.0
	reworkScore := (1.0 - metrics.ReworkRate) * 100.0

	// Apply benchmark comparison bonus
	if benchmark.MedianSuccessRate != nil {
		successRatio := metrics.SuccessRate / *benchmark.MedianSuccessRate
		if successRatio > 1.0 {
			successScore = math.Min(100.0, successScore+(successRatio-1.0)*10.0)
		}
	}

	score := int(math.Round(successScore*0.6 + errorScore*0.3 + reworkScore*0.1))
	return clampScore(score)
}

// calculateSpeedScore measures execution speed relative to benchmark
// Score = 100 - ((actual_duration - benchmark) / benchmark * 100)
func (s *ScoringService) calculateSpeedScore(metrics *WorkflowMetrics, benchmark *models.IndustryBenchmark) int {
	if benchmark.MedianDurationMinutes == nil || *benchmark.MedianDurationMinutes == 0 {
		return 70 // Default score if no benchmark data
	}

	durationRatio := metrics.AvgDurationMinutes / *benchmark.MedianDurationMinutes

	var score float64
	if durationRatio <= 1.0 {
		// Faster than benchmark - bonus
		score = 100.0 + ((1.0 - durationRatio) * 20.0)
	} else {
		// Slower than benchmark - penalty
		score = 100.0 - ((durationRatio - 1.0) * 50.0)
	}

	// Factor in cycle time variance (consistency bonus)
	if metrics.DurationStdDev < metrics.AvgDurationMinutes*0.2 {
		score += 5 // Consistent execution bonus
	}

	return clampScore(int(math.Round(score)))
}

// calculateAutomationScore measures level of automation vs manual work
// Score = automation_rate * 0.7 + (100 - manual_touchpoints_ratio) * 0.3
func (s *ScoringService) calculateAutomationScore(metrics *WorkflowMetrics, benchmark *models.IndustryBenchmark) int {
	automationScore := metrics.AutomationRate * 100.0

	// Manual touchpoint penalty
	manualTouchpointScore := 100.0
	if metrics.TotalSteps > 0 {
		manualRatio := float64(metrics.ManualTouchpoints) / float64(metrics.TotalSteps)
		manualTouchpointScore = (1.0 - manualRatio) * 100.0
	}

	// Compare against industry benchmark
	if benchmark.MedianAutomationRate != nil {
		automationRatio := metrics.AutomationRate / *benchmark.MedianAutomationRate
		if automationRatio > 1.0 {
			automationScore = math.Min(100.0, automationScore+(automationRatio-1.0)*15.0)
		}
	}

	score := int(math.Round(automationScore*0.7 + manualTouchpointScore*0.3))
	return clampScore(score)
}

// calculateComplianceScore measures regulatory compliance and audit readiness
// Score = audit_coverage * 0.4 + (100 - violation_rate) * 0.4 + documentation_score * 0.2
func (s *ScoringService) calculateComplianceScore(metrics *WorkflowMetrics) int {
	auditScore := metrics.AuditCoverage * 100.0
	violationScore := (1.0 - metrics.ViolationRate) * 100.0
	documentationScore := metrics.DocumentationCompleteness * 100.0

	score := int(math.Round(auditScore*0.4 + violationScore*0.4 + documentationScore*0.2))
	return clampScore(score)
}

// ============================================================================
// Grade Assignment
// ============================================================================

func assignGrade(score int) string {
	switch {
	case score >= 97:
		return "A+"
	case score >= 93:
		return "A"
	case score >= 90:
		return "B+"
	case score >= 83:
		return "B"
	case score >= 77:
		return "C+"
	case score >= 73:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

// ============================================================================
// Percentile Calculation
// ============================================================================

func (s *ScoringService) calculatePercentile(
	ctx context.Context,
	tenantID uuid.UUID,
	workflowType string,
	score int,
) int {
	query := `
		SELECT COUNT(*) as total,
		       COUNT(CASE WHEN overall_score < $1 THEN 1 END) as lower_count
		FROM bp_performance_scores
		WHERE workflow_type = $2
		  AND tenant_id != $3
	`

	var total, lowerCount int
	err := s.db.QueryRowContext(ctx, query, score, workflowType, tenantID).Scan(&total, &lowerCount)
	if err != nil || total == 0 {
		return 50 // Default to median if no peer data
	}

	percentile := int(math.Round(float64(lowerCount) / float64(total) * 100.0))
	return percentile
}

// ============================================================================
// Helper Functions
// ============================================================================

func clampScore(score int) int {
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

// ============================================================================
// Data Models
// ============================================================================

type WorkflowMetrics struct {
	// Core Metrics
	AvgDurationMinutes float64
	DurationStdDev     float64
	SuccessRate        float64
	ErrorRate          float64
	ReworkRate         float64
	AvgCostPerProcess  float64

	// Efficiency Metrics
	ResourceUtilization float64

	// Automation Metrics
	AutomationRate    float64
	ManualTouchpoints int
	TotalSteps        int

	// Compliance Metrics
	AuditCoverage             float64
	ViolationRate             float64
	DocumentationCompleteness float64

	// Metadata
	SampleSize int
}

// ============================================================================
// Database Operations
// ============================================================================

func (s *ScoringService) getWorkflowMetrics(
	ctx context.Context,
	tenantID uuid.UUID,
	workflowType string,
) (*WorkflowMetrics, error) {
	// Query workflow execution data from last 90 days
	query := `
		SELECT
			AVG(duration_minutes) as avg_duration,
			STDDEV(duration_minutes) as duration_stddev,
			AVG(CASE WHEN status = 'completed' THEN 1.0 ELSE 0.0 END) as success_rate,
			AVG(CASE WHEN status = 'failed' THEN 1.0 ELSE 0.0 END) as error_rate,
			COUNT(*) as sample_size
		FROM bp_workflow_executions
		WHERE tenant_id = $1
		  AND workflow_type = $2
		  AND created_at > NOW() - INTERVAL '90 days'
	`

	metrics := &WorkflowMetrics{}
	var stdDev sql.NullFloat64

	err := s.db.QueryRowContext(ctx, query, tenantID, workflowType).Scan(
		&metrics.AvgDurationMinutes,
		&stdDev,
		&metrics.SuccessRate,
		&metrics.ErrorRate,
		&metrics.SampleSize,
	)
	if err != nil {
		return nil, err
	}

	if stdDev.Valid {
		metrics.DurationStdDev = stdDev.Float64
	}

	// TODO: Query additional metrics from other tables
	// For now, set reasonable defaults
	metrics.ReworkRate = 0.05
	metrics.AvgCostPerProcess = 100.0
	metrics.ResourceUtilization = 0.75
	metrics.AutomationRate = 0.60
	metrics.ManualTouchpoints = 3
	metrics.TotalSteps = 10
	metrics.AuditCoverage = 0.85
	metrics.ViolationRate = 0.02
	metrics.DocumentationCompleteness = 0.90

	return metrics, nil
}

func (s *ScoringService) getIndustryBenchmark(
	ctx context.Context,
	industry string,
	processType string,
) (*models.IndustryBenchmark, error) {
	query := `
		SELECT * FROM bp_industry_benchmarks
		WHERE industry = $1 AND process_type = $2
	`

	benchmark := &models.IndustryBenchmark{}
	err := s.db.QueryRowContext(ctx, query, industry, processType).Scan(
		&benchmark.ID,
		&benchmark.Industry,
		&benchmark.ProcessType,
		&benchmark.MedianDurationMinutes,
		&benchmark.TopQuartileDurationMinutes,
		&benchmark.MedianSuccessRate,
		&benchmark.TopQuartileSuccessRate,
		&benchmark.MedianCostPerProcess,
		&benchmark.TopQuartileCostPerProcess,
		&benchmark.MedianAutomationRate,
		&benchmark.TopQuartileAutomationRate,
		&benchmark.SampleSize,
		&benchmark.LastUpdated,
		&benchmark.DataSource,
		&benchmark.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return benchmark, nil
}

func (s *ScoringService) savePerformanceScore(ctx context.Context, score *models.PerformanceScore) error {
	query := `
		INSERT INTO bp_performance_scores (
			id, tenant_id, workflow_type, overall_score, grade, percentile,
			efficiency_score, quality_score, speed_score, automation_score, compliance_score,
			industry, sample_size, confidence_level, calculated_at, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (tenant_id, workflow_type)
		DO UPDATE SET
			overall_score = EXCLUDED.overall_score,
			grade = EXCLUDED.grade,
			percentile = EXCLUDED.percentile,
			efficiency_score = EXCLUDED.efficiency_score,
			quality_score = EXCLUDED.quality_score,
			speed_score = EXCLUDED.speed_score,
			automation_score = EXCLUDED.automation_score,
			compliance_score = EXCLUDED.compliance_score,
			industry = EXCLUDED.industry,
			sample_size = EXCLUDED.sample_size,
			calculated_at = EXCLUDED.calculated_at,
			updated_at = EXCLUDED.updated_at
	`

	_, err := s.db.ExecContext(ctx, query,
		score.ID,
		score.TenantID,
		score.WorkflowType,
		score.OverallScore,
		score.Grade,
		score.Percentile,
		score.EfficiencyScore,
		score.QualityScore,
		score.SpeedScore,
		score.AutomationScore,
		score.ComplianceScore,
		score.Industry,
		score.SampleSize,
		score.ConfidenceLevel,
		score.CalculatedAt,
		score.CreatedAt,
		score.UpdatedAt,
	)

	return err
}
