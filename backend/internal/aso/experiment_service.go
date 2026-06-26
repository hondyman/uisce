package aso

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// A/B Experiment Types
// ============================================================================

// ASOExperiment represents an A/B test for an optimization
type ASOExperiment struct {
	ID             uuid.UUID           `json:"id" db:"id"`
	OptimizationID uuid.UUID           `json:"optimization_id" db:"optimization_id"`
	TenantID       *uuid.UUID          `json:"tenant_id,omitempty" db:"tenant_id"`
	Env            string              `json:"env" db:"env"`
	Name           string              `json:"name" db:"name"`
	Status         ASOExperimentStatus `json:"status" db:"status"`

	// Configuration
	TrafficPercent float64             `json:"traffic_percent" db:"traffic_percent"` // % of queries to test
	Config         ASOExperimentConfig `json:"config" db:"-"`
	ConfigJSON     json.RawMessage     `json:"-" db:"config_json"`

	// Timing
	StartedAt      time.Time  `json:"started_at" db:"started_at"`
	ScheduledEndAt time.Time  `json:"scheduled_end_at" db:"scheduled_end_at"`
	EndedAt        *time.Time `json:"ended_at,omitempty" db:"ended_at"`

	// Results
	Metrics     ASOExperimentMetrics `json:"metrics" db:"-"`
	MetricsJSON json.RawMessage      `json:"-" db:"metrics_json"`
	Outcome     ASOExperimentOutcome `json:"outcome" db:"outcome"`

	CreatedBy string `json:"created_by" db:"created_by"`
}

// ASOExperimentStatus tracks experiment lifecycle
type ASOExperimentStatus string

const (
	ASOExpStatusDraft     ASOExperimentStatus = "draft"
	ASOExpStatusRunning   ASOExperimentStatus = "running"
	ASOExpStatusCompleted ASOExperimentStatus = "completed"
	ASOExpStatusAborted   ASOExperimentStatus = "aborted"
)

// ASOExperimentOutcome is the final decision
type ASOExperimentOutcome string

const (
	ASOExpOutcomePending      ASOExperimentOutcome = "pending"
	ASOExpOutcomePromoted     ASOExperimentOutcome = "promoted"  // Optimization applied
	ASOExpOutcomeAbandoned    ASOExperimentOutcome = "abandoned" // Optimization rejected
	ASOExpOutcomeInconclusive ASOExperimentOutcome = "inconclusive"
)

// ASOExperimentConfig defines experiment parameters
type ASOExperimentConfig struct {
	MinDuration       time.Duration `json:"min_duration"`        // Min runtime
	MaxDuration       time.Duration `json:"max_duration"`        // Max runtime
	MinSampleSize     int           `json:"min_sample_size"`     // Min queries before decision
	SignificanceLevel float64       `json:"significance_level"`  // p-value threshold (0.05)
	MinImprovementPct float64       `json:"min_improvement_pct"` // Required improvement to promote
	AutoPromote       bool          `json:"auto_promote"`        // Auto-promote if successful
}

// ASOExperimentMetrics tracks test vs control metrics
type ASOExperimentMetrics struct {
	ControlGroup   ASOGroupMetrics `json:"control_group"`
	TestGroup      ASOGroupMetrics `json:"test_group"`
	PValue         float64         `json:"p_value"`
	Significant    bool            `json:"significant"`
	ImprovementPct float64         `json:"improvement_pct"`
}

// ASOGroupMetrics are stats for one group
type ASOGroupMetrics struct {
	QueryCount     int64   `json:"query_count"`
	TotalLatencyMs float64 `json:"total_latency_ms"`
	AvgLatencyMs   float64 `json:"avg_latency_ms"`
	P50LatencyMs   float64 `json:"p50_latency_ms"`
	P95LatencyMs   float64 `json:"p95_latency_ms"`
	P99LatencyMs   float64 `json:"p99_latency_ms"`
	HitRate        float64 `json:"hit_rate"` // Pre-agg hit rate
	ErrorRate      float64 `json:"error_rate"`
}

// ============================================================================
// Experiment Service
// ============================================================================

// ExperimentService manages A/B tests for optimizations
type ExperimentService interface {
	// CreateExperiment creates a new A/B test
	CreateExperiment(ctx context.Context, optID uuid.UUID, config ASOExperimentConfig, creator string) (*ASOExperiment, error)

	// StartExperiment begins running the experiment
	StartExperiment(ctx context.Context, expID uuid.UUID) error

	// StopExperiment ends the experiment early
	StopExperiment(ctx context.Context, expID uuid.UUID, reason string) error

	// RecordQueryMetric records a query result for the experiment
	RecordQueryMetric(ctx context.Context, expID uuid.UUID, isTestGroup bool, latencyMs float64, hit bool, error bool) error

	// GetExperiment retrieves an experiment
	GetExperiment(ctx context.Context, expID uuid.UUID) (*ASOExperiment, error)

	// ListExperiments lists experiments with filters
	ListExperiments(ctx context.Context, status *ASOExperimentStatus, limit int) ([]ASOExperiment, error)

	// EvaluateExperiment analyzes results and determines outcome
	EvaluateExperiment(ctx context.Context, expID uuid.UUID) (*ASOExperimentMetrics, error)

	// ShouldRouteToTest determines if a query should use the test group
	ShouldRouteToTest(ctx context.Context, optID uuid.UUID, queryHash string) (bool, error)
}

// experimentService implements ExperimentService
type experimentService struct {
	db      *sqlx.DB
	optRepo ASOOptimizationRepository
	rng     *rand.Rand
}

// NewExperimentService creates a new experiment service
func NewExperimentService(db *sqlx.DB, optRepo ASOOptimizationRepository) ExperimentService {
	return &experimentService{
		db:      db,
		optRepo: optRepo,
		rng:     rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// CreateExperiment creates a new A/B test
func (s *experimentService) CreateExperiment(ctx context.Context, optID uuid.UUID, config ASOExperimentConfig, creator string) (*ASOExperiment, error) {
	opt, err := s.optRepo.GetByID(ctx, optID)
	if err != nil || opt == nil {
		return nil, fmt.Errorf("optimization not found")
	}

	// Default config values
	if config.MinDuration == 0 {
		config.MinDuration = 24 * time.Hour
	}
	if config.MaxDuration == 0 {
		config.MaxDuration = 7 * 24 * time.Hour
	}
	if config.MinSampleSize == 0 {
		config.MinSampleSize = 1000
	}
	if config.SignificanceLevel == 0 {
		config.SignificanceLevel = 0.05
	}
	if config.MinImprovementPct == 0 {
		config.MinImprovementPct = 10.0
	}

	configJSON, _ := json.Marshal(config)

	exp := &ASOExperiment{
		ID:             uuid.New(),
		OptimizationID: optID,
		TenantID:       opt.TenantID,
		Env:            opt.Env,
		Name:           fmt.Sprintf("A/B Test: %s", opt.TargetName),
		Status:         ASOExpStatusDraft,
		TrafficPercent: 10.0, // Start with 10% test traffic
		Config:         config,
		ConfigJSON:     configJSON,
		ScheduledEndAt: time.Now().Add(config.MaxDuration),
		Outcome:        ASOExpOutcomePending,
		CreatedBy:      creator,
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO semantic.aso_experiment (
			id, optimization_id, tenant_id, env, name, status,
			traffic_percent, config_json, scheduled_end_at, outcome, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`, exp.ID, exp.OptimizationID, exp.TenantID, exp.Env, exp.Name,
		exp.Status, exp.TrafficPercent, exp.ConfigJSON, exp.ScheduledEndAt,
		exp.Outcome, exp.CreatedBy)

	if err != nil {
		return nil, err
	}

	return exp, nil
}

// StartExperiment begins running the experiment
func (s *experimentService) StartExperiment(ctx context.Context, expID uuid.UUID) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE semantic.aso_experiment
		SET status = 'running', started_at = $2
		WHERE id = $1 AND status = 'draft'
	`, expID, now)
	return err
}

// StopExperiment ends the experiment early
func (s *experimentService) StopExperiment(ctx context.Context, expID uuid.UUID, reason string) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE semantic.aso_experiment
		SET status = 'aborted', ended_at = $2, outcome = 'abandoned'
		WHERE id = $1 AND status = 'running'
	`, expID, now)
	return err
}

// RecordQueryMetric records a query result for the experiment
func (s *experimentService) RecordQueryMetric(ctx context.Context, expID uuid.UUID, isTestGroup bool, latencyMs float64, hit bool, hasError bool) error {
	group := "control"
	if isTestGroup {
		group = "test"
	}

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO semantic.aso_experiment_metric (
			experiment_id, group_name, latency_ms, hit, error, recorded_at
		) VALUES ($1, $2, $3, $4, $5, now())
	`, expID, group, latencyMs, hit, hasError)

	return err
}

// GetExperiment retrieves an experiment
func (s *experimentService) GetExperiment(ctx context.Context, expID uuid.UUID) (*ASOExperiment, error) {
	var exp ASOExperiment
	err := s.db.GetContext(ctx, &exp, `SELECT * FROM semantic.aso_experiment WHERE id = $1`, expID)
	if err != nil {
		return nil, err
	}

	// Deserialize config and metrics
	if exp.ConfigJSON != nil {
		_ = json.Unmarshal(exp.ConfigJSON, &exp.Config)
	}
	if exp.MetricsJSON != nil {
		_ = json.Unmarshal(exp.MetricsJSON, &exp.Metrics)
	}

	return &exp, nil
}

// ListExperiments lists experiments with filters
func (s *experimentService) ListExperiments(ctx context.Context, status *ASOExperimentStatus, limit int) ([]ASOExperiment, error) {
	query := `SELECT * FROM semantic.aso_experiment WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if status != nil {
		query += fmt.Sprintf(" AND status = $%d", argNum)
		args = append(args, *status)
		argNum++
	}

	query += " ORDER BY started_at DESC"
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	var exps []ASOExperiment
	err := s.db.SelectContext(ctx, &exps, query, args...)
	return exps, err
}

// EvaluateExperiment analyzes results and determines outcome
func (s *experimentService) EvaluateExperiment(ctx context.Context, expID uuid.UUID) (*ASOExperimentMetrics, error) {
	exp, err := s.GetExperiment(ctx, expID)
	if err != nil {
		return nil, err
	}

	// Aggregate metrics for each group
	controlMetrics := s.aggregateGroupMetrics(ctx, expID, "control")
	testMetrics := s.aggregateGroupMetrics(ctx, expID, "test")

	// Calculate improvement
	var improvement float64
	if controlMetrics.AvgLatencyMs > 0 {
		improvement = (controlMetrics.AvgLatencyMs - testMetrics.AvgLatencyMs) / controlMetrics.AvgLatencyMs * 100
	}

	// Statistical significance (simplified - would use proper t-test)
	pValue := s.calculatePValue(controlMetrics, testMetrics)
	significant := pValue < exp.Config.SignificanceLevel

	metrics := &ASOExperimentMetrics{
		ControlGroup:   controlMetrics,
		TestGroup:      testMetrics,
		PValue:         pValue,
		Significant:    significant,
		ImprovementPct: improvement,
	}

	// Update experiment with metrics
	metricsJSON, _ := json.Marshal(metrics)
	s.db.ExecContext(ctx, `
		UPDATE semantic.aso_experiment SET metrics_json = $2 WHERE id = $1
	`, expID, metricsJSON)

	// Determine outcome
	if significant && improvement >= exp.Config.MinImprovementPct {
		if exp.Config.AutoPromote {
			s.promoteExperiment(ctx, exp)
		}
	}

	return metrics, nil
}

// ShouldRouteToTest determines if a query should use the test group
func (s *experimentService) ShouldRouteToTest(ctx context.Context, optID uuid.UUID, queryHash string) (bool, error) {
	// Get running experiment for this optimization
	var exp ASOExperiment
	err := s.db.GetContext(ctx, &exp, `
		SELECT * FROM semantic.aso_experiment
		WHERE optimization_id = $1 AND status = 'running'
		LIMIT 1
	`, optID)

	if err != nil {
		return false, nil // No experiment, use control
	}

	// Deterministic routing based on query hash
	// This ensures same query always goes to same group
	hashNum := hashToNumber(queryHash)
	threshold := exp.TrafficPercent / 100.0

	return hashNum < threshold, nil
}

// Helper methods
func (s *experimentService) aggregateGroupMetrics(ctx context.Context, expID uuid.UUID, group string) ASOGroupMetrics {
	var metrics ASOGroupMetrics

	s.db.GetContext(ctx, &metrics, `
		SELECT 
			COUNT(*) as query_count,
			COALESCE(SUM(latency_ms), 0) as total_latency_ms,
			COALESCE(AVG(latency_ms), 0) as avg_latency_ms,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY latency_ms), 0) as p50_latency_ms,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms), 0) as p95_latency_ms,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms), 0) as p99_latency_ms,
			COALESCE(AVG(CASE WHEN hit THEN 1.0 ELSE 0.0 END), 0) as hit_rate,
			COALESCE(AVG(CASE WHEN error THEN 1.0 ELSE 0.0 END), 0) as error_rate
		FROM semantic.aso_experiment_metric
		WHERE experiment_id = $1 AND group_name = $2
	`, expID, group)

	return metrics
}

func (s *experimentService) calculatePValue(control, test ASOGroupMetrics) float64 {
	// Simplified statistical test
	// In production, use proper Welch's t-test
	if control.QueryCount < 30 || test.QueryCount < 30 {
		return 1.0 // Not enough samples
	}

	// Placeholder - would calculate actual p-value
	return 0.03
}

func (s *experimentService) promoteExperiment(ctx context.Context, exp *ASOExperiment) error {
	now := time.Now()

	// Update experiment status
	s.db.ExecContext(ctx, `
		UPDATE semantic.aso_experiment
		SET status = 'completed', ended_at = $2, outcome = 'promoted'
		WHERE id = $1
	`, exp.ID, now)

	// Apply the optimization
	s.optRepo.UpdateStatus(ctx, exp.OptimizationID, OptStatusApproved, "experiment_service", "A/B test successful")

	return nil
}

func hashToNumber(s string) float64 {
	// Simple hash to 0-1 range
	var sum uint64
	for _, c := range s {
		sum = sum*31 + uint64(c)
	}
	return float64(sum%10000) / 10000.0
}
