package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// MetricRegistryEntry represents a metric in the semantic registry
type MetricRegistryEntry struct {
	MetricID                 uuid.UUID       `db:"metric_id" json:"metric_id"`
	Name                     string          `db:"name" json:"name"`
	DisplayName              string          `db:"display_name" json:"display_name"`
	Description              *string         `db:"description" json:"description"`
	Domain                   string          `db:"domain" json:"domain"`
	Category                 string          `db:"category" json:"category"`
	MetricType               string          `db:"metric_type" json:"metric_type"` // atomic, derived, composite
	BaseQuery                *string         `db:"base_query" json:"base_query"`
	AggregationFunction      *string         `db:"aggregation_function" json:"aggregation_function"`
	Granularity              []string        `db:"granularity" json:"granularity"`
	ValueColumn              *string         `db:"value_column" json:"value_column"`
	DateColumn               *string         `db:"date_column" json:"date_column"`
	SourceFormula            *string         `db:"source_formula" json:"source_formula"`
	SourceSystem             *string         `db:"source_system" json:"source_system"`
	ComparisonPeriods        json.RawMessage `db:"comparison_periods" json:"comparison_periods"`
	PeriodLabelFormat        string          `db:"period_label_format" json:"period_label_format"`
	SLAFreshnessHours        int             `db:"sla_freshness_hours" json:"sla_freshness_hours"`
	SLACompletenessThreshold float64         `db:"sla_completeness_threshold" json:"sla_completeness_threshold"`
	RefreshSchedule          string          `db:"refresh_schedule" json:"refresh_schedule"`
	OwnerUserID              *uuid.UUID      `db:"owner_user_id" json:"owner_user_id"`
	StewardGroup             *string         `db:"steward_group" json:"steward_group"`
	GoldenPath               bool            `db:"golden_path" json:"golden_path"`
	Status                   string          `db:"status" json:"status"`
	Version                  int             `db:"version" json:"version"`
	CreatedAt                time.Time       `db:"created_at" json:"created_at"`
	UpdatedAt                time.Time       `db:"updated_at" json:"updated_at"`
	CreatedBy                *string         `db:"created_by" json:"created_by"`
	UpdatedBy                *string         `db:"updated_by" json:"updated_by"`
}

// ExecutionLog tracks metric computation execution
type ExecutionLog struct {
	ExecutionID       uuid.UUID       `db:"execution_id" json:"execution_id"`
	MetricID          uuid.UUID       `db:"metric_id" json:"metric_id"`
	Lane              string          `db:"lane" json:"lane"` // real-time, batch
	ExecutionType     string          `db:"execution_type" json:"execution_type"`
	PeriodStart       *time.Time      `db:"period_start" json:"period_start"`
	PeriodEnd         *time.Time      `db:"period_end" json:"period_end"`
	PeriodLabel       *string         `db:"period_label" json:"period_label"`
	Status            string          `db:"status" json:"status"`
	RecordCount       *int            `db:"record_count" json:"record_count"`
	SuccessCount      *int            `db:"success_count" json:"success_count"`
	ErrorCount        *int            `db:"error_count" json:"error_count"`
	CompletenessScore *float64        `db:"completeness_score" json:"completeness_score"`
	FreshnessHours    *float64        `db:"freshness_hours" json:"freshness_hours"`
	ErrorMessage      *string         `db:"error_message" json:"error_message"`
	ErrorDetails      json.RawMessage `db:"error_details" json:"error_details"`
	StartedAt         time.Time       `db:"started_at" json:"started_at"`
	CompletedAt       *time.Time      `db:"completed_at" json:"completed_at"`
	DurationMs        *int            `db:"duration_ms" json:"duration_ms"`
}

// MetricRegistryService handles metric registry operations and orchestration
type MetricRegistryService struct {
	db *sqlx.DB
}

// NewMetricRegistryService creates a new MetricRegistryService
func NewMetricRegistryService(db *sqlx.DB) *MetricRegistryService {
	return &MetricRegistryService{db: db}
}

// GetMetricRegistry retrieves a specific metric from the registry
func (s *MetricRegistryService) GetMetricRegistry(ctx context.Context, metricID uuid.UUID) (*MetricRegistryEntry, error) {
	var metric MetricRegistryEntry
	err := s.db.GetContext(ctx, &metric, `
		SELECT 
			metric_id, name, display_name, description, domain, category,
			metric_type, base_query, aggregation_function, granularity,
			value_column, date_column, source_formula, source_system,
			comparison_periods, period_label_format, sla_freshness_hours,
			sla_completeness_threshold, refresh_schedule, owner_user_id,
			steward_group, golden_path, status, version,
			created_at, updated_at, created_by, updated_by
		FROM semantic_layer.metric_registry
		WHERE metric_id = $1
	`, metricID)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric registry: %w", err)
	}
	return &metric, nil
}

// ListMetricRegistry lists all active metrics in the registry
func (s *MetricRegistryService) ListMetricRegistry(ctx context.Context, domain *string, goldenPathOnly bool) ([]MetricRegistryEntry, error) {
	var metrics []MetricRegistryEntry
	query := `
		SELECT 
			metric_id, name, display_name, description, domain, category,
			metric_type, base_query, aggregation_function, granularity,
			value_column, date_column, source_formula, source_system,
			comparison_periods, period_label_format, sla_freshness_hours,
			sla_completeness_threshold, refresh_schedule, owner_user_id,
			steward_group, golden_path, status, version,
			created_at, updated_at, created_by, updated_by
		FROM semantic_layer.metric_registry
		WHERE status = 'active'
	`
	args := []interface{}{}

	if domain != nil {
		query += ` AND domain = $1`
		args = append(args, *domain)
	}

	if goldenPathOnly {
		query += ` AND golden_path = true`
	}

	query += ` ORDER BY domain, category, name`

	err := s.db.SelectContext(ctx, &metrics, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list metrics: %w", err)
	}
	return metrics, nil
}

// RefreshAtomicMetrics executes the real-time atomic refresh lane
func (s *MetricRegistryService) RefreshAtomicMetrics(ctx context.Context, metricID *uuid.UUID) ([]ExecutionLog, error) {
	var execLogs []ExecutionLog
	query := `SELECT * FROM public.refresh_atomic_metrics($1)`

	rows, err := s.db.QueryxContext(ctx, query, metricID)
	if err != nil {
		return nil, fmt.Errorf("failed to execute atomic refresh: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var log ExecutionLog
		// Map the function output to ExecutionLog
		if err := rows.StructScan(&log); err != nil {
			return nil, fmt.Errorf("failed to scan execution log: %w", err)
		}
		execLogs = append(execLogs, log)
	}

	return execLogs, rows.Err()
}

// ComputeMonthlyPoP executes the batch PoP computation lane
func (s *MetricRegistryService) ComputeMonthlyPoP(ctx context.Context, metricID *uuid.UUID, periodStart, periodEnd *time.Time) (ExecutionLog, error) {
	var log ExecutionLog

	query := `SELECT * FROM public.compute_monthly_pop($1, $2, $3) LIMIT 1`

	err := s.db.GetContext(ctx, &log, query, metricID, periodStart, periodEnd)
	if err != nil {
		return log, fmt.Errorf("failed to compute PoP: %w", err)
	}

	return log, nil
}

// ComputeComparisonPeriods computes YoY, QoQ, PoP comparisons
func (s *MetricRegistryService) ComputeComparisonPeriods(ctx context.Context, metricID *uuid.UUID) (ExecutionLog, error) {
	var log ExecutionLog

	query := `SELECT * FROM public.compute_comparison_periods($1) LIMIT 1`

	err := s.db.GetContext(ctx, &log, query, metricID)
	if err != nil {
		return log, fmt.Errorf("failed to compute comparison periods: %w", err)
	}

	return log, nil
}

// DetectZScoreAnomalies runs anomaly detection with z-score windowing
func (s *MetricRegistryService) DetectZScoreAnomalies(ctx context.Context, metricID *uuid.UUID, threshold float64, windowDays int, minDataPoints int) ([]map[string]interface{}, error) {
	var anomalies []map[string]interface{}

	query := `
		SELECT 
			execution_id, metric_id, error_message
		FROM public.detect_zscore_anomalies($1, $2, $3, $4)
	`

	rows, err := s.db.QueryContext(ctx, query, metricID, threshold, windowDays, minDataPoints)
	if err != nil {
		return nil, fmt.Errorf("failed to detect anomalies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var execID uuid.UUID
		var mID uuid.UUID
		var errMsg *string

		if err := rows.Scan(&execID, &mID, &errMsg); err != nil {
			return nil, fmt.Errorf("failed to scan anomaly result: %w", err)
		}

		anomalies = append(anomalies, map[string]interface{}{
			"execution_id": execID,
			"metric_id":    mID,
			"error":        errMsg,
		})
	}

	return anomalies, rows.Err()
}

// GetExecutionHistory retrieves execution logs for a metric
func (s *MetricRegistryService) GetExecutionHistory(ctx context.Context, metricID uuid.UUID, limit int) ([]ExecutionLog, error) {
	var logs []ExecutionLog

	query := `
		SELECT 
			execution_id, metric_id, lane, execution_type, period_start, period_end,
			period_label, status, record_count, success_count, error_count,
			completeness_score, freshness_hours, error_message, error_details,
			started_at, completed_at, duration_ms
		FROM semantic_layer.metric_execution_log
		WHERE metric_id = $1
		ORDER BY completed_at DESC NULLS LAST
		LIMIT $2
	`

	err := s.db.SelectContext(ctx, &logs, query, metricID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get execution history: %w", err)
	}

	return logs, nil
}

// GetGoldenPathReadiness checks readiness of golden path metrics
func (s *MetricRegistryService) GetGoldenPathReadiness(ctx context.Context) ([]map[string]interface{}, error) {
	var readiness []map[string]interface{}

	query := `
		SELECT 
			metric_id, name, display_name, domain, readiness_status,
			current_value, last_data_date, last_refresh, violation_type, violation_status
		FROM public.golden_path_readiness
		ORDER BY domain, name
	`

	rows, err := s.db.QueryxContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get golden path readiness: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		m := map[string]interface{}{}
		if err := rows.MapScan(m); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		readiness = append(readiness, m)
	}

	return readiness, rows.Err()
}

// RegisterMetric registers a new metric in the semantic registry
func (s *MetricRegistryService) RegisterMetric(ctx context.Context, metric *MetricRegistryEntry) (uuid.UUID, error) {
	metricID := uuid.New()

	query := `
		INSERT INTO semantic_layer.metric_registry (
			metric_id, name, display_name, description, domain, category,
			metric_type, base_query, aggregation_function, granularity,
			value_column, date_column, source_formula, source_system,
			comparison_periods, period_label_format, sla_freshness_hours,
			sla_completeness_threshold, refresh_schedule, owner_user_id,
			steward_group, golden_path, status, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23, $24
		)
	`

	_, err := s.db.ExecContext(ctx, query,
		metricID, metric.Name, metric.DisplayName, metric.Description,
		metric.Domain, metric.Category, metric.MetricType,
		metric.BaseQuery, metric.AggregationFunction, metric.Granularity,
		metric.ValueColumn, metric.DateColumn, metric.SourceFormula,
		metric.SourceSystem, metric.ComparisonPeriods, metric.PeriodLabelFormat,
		metric.SLAFreshnessHours, metric.SLACompletenessThreshold,
		metric.RefreshSchedule, metric.OwnerUserID, metric.StewardGroup,
		metric.GoldenPath, metric.Status, metric.CreatedBy,
	)

	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to register metric: %w", err)
	}

	return metricID, nil
}

// PromoteToGoldenPath marks a metric as golden path
func (s *MetricRegistryService) PromoteToGoldenPath(ctx context.Context, metricID uuid.UUID) error {
	query := `
		UPDATE semantic_layer.metric_registry
		SET golden_path = true, updated_at = NOW()
		WHERE metric_id = $1
	`

	result, err := s.db.ExecContext(ctx, query, metricID)
	if err != nil {
		return fmt.Errorf("failed to promote metric: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil || rows == 0 {
		return fmt.Errorf("metric not found or already golden: %w", err)
	}

	return nil
}
