package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/lib/pq"
)

// PoPService handles Period-over-Period analysis and anomaly detection
type PoPService struct {
	db           *sql.DB
	hasuraClient HasuraClient
}

// NewPoPService creates a new PoP service instance
func NewPoPService(db *sql.DB) *PoPService {
	return &PoPService{db: db}
}

// NewPoPServiceWithHasura creates a new service with Hasura support
func NewPoPServiceWithHasura(db *sql.DB, hasuraClient HasuraClient) *PoPService {
	return &PoPService{
		db:           db,
		hasuraClient: hasuraClient,
	}
}

// PoPMetric represents a period-over-period metric definition
type PoPMetric struct {
	ID                       string                 `json:"id" db:"id"`
	Name                     string                 `json:"name" db:"name"`
	DisplayName              string                 `json:"display_name" db:"display_name"`
	Description              string                 `json:"description" db:"description"`
	Domain                   string                 `json:"domain" db:"domain"`
	Category                 string                 `json:"category" db:"category"`
	MetricType               string                 `json:"metric_type" db:"metric_type"`
	BaseQuery                string                 `json:"base_query" db:"base_query"`
	AggregationFunction      string                 `json:"aggregation_function" db:"aggregation_function"`
	DateColumn               string                 `json:"date_column" db:"date_column"`
	ValueColumn              string                 `json:"value_column" db:"value_column"`
	Granularity              string                 `json:"granularity" db:"granularity"`
	ComparisonPeriods        []string               `json:"comparison_periods" db:"comparison_periods"`
	OwnerUserID              string                 `json:"owner_user_id" db:"owner_user_id"`
	StewardGroup             string                 `json:"steward_group" db:"steward_group"`
	DataSource               string                 `json:"data_source" db:"data_source"`
	SchemaName               string                 `json:"schema_name" db:"schema_name"`
	TableName                string                 `json:"table_name" db:"table_name"`
	SLAFreshnessHours        int                    `json:"sla_freshness_hours" db:"sla_freshness_hours"`
	SLACompletenessThreshold float64                `json:"sla_completeness_threshold" db:"sla_completeness_threshold"`
	DataQualityChecks        map[string]interface{} `json:"data_quality_checks" db:"data_quality_checks"`
	Status                   string                 `json:"status" db:"status"`
	GoldenPath               bool                   `json:"golden_path" db:"golden_path"`
	Version                  int                    `json:"version" db:"version"`
	CreatedAt                time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time              `json:"updated_at" db:"updated_at"`
	CreatedBy                string                 `json:"created_by" db:"created_by"`
	UpdatedBy                string                 `json:"updated_by" db:"updated_by"`
}

// PoPComputation represents computed PoP values
type PoPComputation struct {
	ID                string    `json:"id" db:"id"`
	MetricID          string    `json:"metric_id" db:"metric_id"`
	PeriodStart       time.Time `json:"period_start" db:"period_start"`
	PeriodEnd         time.Time `json:"period_end" db:"period_end"`
	Granularity       string    `json:"granularity" db:"granularity"`
	PeriodLabel       string    `json:"period_label" db:"period_label"`
	CurrentValue      *float64  `json:"current_value" db:"current_value"`
	PreviousValue     *float64  `json:"previous_value" db:"previous_value"`
	Delta             *float64  `json:"delta" db:"delta"`
	PercentChange     *float64  `json:"percent_change" db:"percent_change"`
	RecordCount       *int      `json:"record_count" db:"record_count"`
	LastUpdated       time.Time `json:"last_updated" db:"last_updated"`
	ComputationStatus string    `json:"computation_status" db:"computation_status"`
}

// PoPAnomaly represents detected anomalies
type PoPAnomaly struct {
	ID               string                 `json:"id" db:"id"`
	MetricID         string                 `json:"metric_id" db:"metric_id"`
	ComputationID    string                 `json:"computation_id" db:"computation_id"`
	AnomalyType      string                 `json:"anomaly_type" db:"anomaly_type"`
	Severity         string                 `json:"severity" db:"severity"`
	Confidence       *float64               `json:"confidence" db:"confidence"`
	ZScore           *float64               `json:"z_score" db:"z_score"`
	ExpectedValue    *float64               `json:"expected_value" db:"expected_value"`
	ExpectedRangeMin *float64               `json:"expected_range_min" db:"expected_range_min"`
	ExpectedRangeMax *float64               `json:"expected_range_max" db:"expected_range_max"`
	ActualValue      *float64               `json:"actual_value" db:"actual_value"`
	DetectionMethod  string                 `json:"detection_method" db:"detection_method"`
	DetectionParams  map[string]interface{} `json:"detection_params" db:"detection_params"`
	DetectedAt       time.Time              `json:"detected_at" db:"detected_at"`
	Status           string                 `json:"status" db:"status"`
	ResolvedAt       *time.Time             `json:"resolved_at" db:"resolved_at"`
	ResolvedBy       *string                `json:"resolved_by" db:"resolved_by"`
	ResolutionNotes  *string                `json:"resolution_notes" db:"resolution_notes"`
}

// StewardReview represents steward review sessions
type StewardReview struct {
	ID                string                   `json:"id" db:"id"`
	MetricID          string                   `json:"metric_id" db:"metric_id"`
	ReviewPeriodStart time.Time                `json:"review_period_start" db:"review_period_start"`
	ReviewPeriodEnd   time.Time                `json:"review_period_end" db:"review_period_end"`
	ReviewerUserID    string                   `json:"reviewer_user_id" db:"reviewer_user_id"`
	ReviewType        string                   `json:"review_type" db:"review_type"`
	OverallRating     *string                  `json:"overall_rating" db:"overall_rating"`
	ReviewNotes       *string                  `json:"review_notes" db:"review_notes"`
	ActionItems       []map[string]interface{} `json:"action_items" db:"action_items"`
	Status            string                   `json:"status" db:"status"`
	DueDate           *time.Time               `json:"due_date" db:"due_date"`
	CompletedAt       *time.Time               `json:"completed_at" db:"completed_at"`
	CreatedAt         time.Time                `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time                `json:"updated_at" db:"updated_at"`
}

// CreatePoPMetric creates a new PoP metric definition
func (s *PoPService) CreatePoPMetric(ctx context.Context, metric *PoPMetric) error {
	comparisonPeriodsJSON, err := json.Marshal(metric.ComparisonPeriods)
	if err != nil {
		return fmt.Errorf("failed to marshal comparison periods: %w", err)
	}

	dataQualityChecksJSON, err := json.Marshal(metric.DataQualityChecks)
	if err != nil {
		return fmt.Errorf("failed to marshal data quality checks: %w", err)
	}

	query := `
		INSERT INTO public.pop_metrics (
			name, display_name, description, domain, category, metric_type,
			base_query, aggregation_function, date_column, value_column,
			granularity, comparison_periods, owner_user_id, steward_group,
			data_source, schema_name, table_name, sla_freshness_hours,
			sla_completeness_threshold, data_quality_checks, status,
			golden_path, version, created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14,
			$15, $16, $17, $18, $19, $20, $21, $22, $23, $24)
		RETURNING id`

	err = s.db.QueryRowContext(ctx, query,
		metric.Name, metric.DisplayName, metric.Description, metric.Domain,
		metric.Category, metric.MetricType, metric.BaseQuery, metric.AggregationFunction,
		metric.DateColumn, metric.ValueColumn, metric.Granularity, comparisonPeriodsJSON,
		metric.OwnerUserID, metric.StewardGroup, metric.DataSource, metric.SchemaName,
		metric.TableName, metric.SLAFreshnessHours, metric.SLACompletenessThreshold,
		dataQualityChecksJSON, metric.Status, metric.GoldenPath, metric.Version,
		metric.CreatedBy).Scan(&metric.ID)

	if err != nil {
		return fmt.Errorf("failed to create PoP metric: %w", err)
	}

	return nil
}

// GetPoPMetrics retrieves PoP metrics with optional filtering
func (s *PoPService) GetPoPMetrics(ctx context.Context, filters map[string]interface{}) ([]PoPMetric, error) {
	query := `
		SELECT id, name, display_name, description, domain, category, metric_type,
			base_query, aggregation_function, date_column, value_column, granularity,
			comparison_periods, owner_user_id, steward_group, data_source, schema_name,
			table_name, sla_freshness_hours, sla_completeness_threshold, data_quality_checks,
			status, golden_path, version, created_at, updated_at, created_by, updated_by
		FROM public.pop_metrics
		WHERE 1=1`

	args := []interface{}{}
	argCount := 0

	if domain, ok := filters["domain"].(string); ok && domain != "" {
		argCount++
		query += fmt.Sprintf(" AND domain = $%d", argCount)
		args = append(args, domain)
	}

	if status, ok := filters["status"].(string); ok && status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, status)
	}

	if goldenPath, ok := filters["golden_path"].(bool); ok {
		argCount++
		query += fmt.Sprintf(" AND golden_path = $%d", argCount)
		args = append(args, goldenPath)
	}

	query += " ORDER BY domain, category, name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query PoP metrics: %w", err)
	}
	defer rows.Close()

	var metrics []PoPMetric
	for rows.Next() {
		var metric PoPMetric
		var comparisonPeriodsJSON, dataQualityChecksJSON []byte

		var desc sql.NullString
		var createdBy sql.NullString
		var updatedBy sql.NullString

		err := rows.Scan(
			&metric.ID, &metric.Name, &metric.DisplayName, &desc,
			&metric.Domain, &metric.Category, &metric.MetricType, &metric.BaseQuery,
			&metric.AggregationFunction, &metric.DateColumn, &metric.ValueColumn,
			&metric.Granularity, &comparisonPeriodsJSON, &metric.OwnerUserID,
			&metric.StewardGroup, &metric.DataSource, &metric.SchemaName,
			&metric.TableName, &metric.SLAFreshnessHours, &metric.SLACompletenessThreshold,
			&dataQualityChecksJSON, &metric.Status, &metric.GoldenPath, &metric.Version,
			&metric.CreatedAt, &metric.UpdatedAt, &createdBy, &updatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan PoP metric: %w", err)
		}

		if desc.Valid {
			metric.Description = desc.String
		} else {
			metric.Description = ""
		}
		if createdBy.Valid {
			metric.CreatedBy = createdBy.String
		} else {
			metric.CreatedBy = ""
		}
		if updatedBy.Valid {
			metric.UpdatedBy = updatedBy.String
		} else {
			metric.UpdatedBy = ""
		}

		// Unmarshal JSON fields
		if err := json.Unmarshal(comparisonPeriodsJSON, &metric.ComparisonPeriods); err != nil {
			log.Printf("Warning: failed to unmarshal comparison periods for metric %s: %v", metric.ID, err)
		}

		if len(dataQualityChecksJSON) > 0 {
			if err := json.Unmarshal(dataQualityChecksJSON, &metric.DataQualityChecks); err != nil {
				log.Printf("Warning: failed to unmarshal data quality checks for metric %s: %v", metric.ID, err)
			}
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

// ComputePoPValues computes period-over-period values for a metric
func (s *PoPService) ComputePoPValues(ctx context.Context, metricID string, periodStart, periodEnd time.Time) error {
	// Get metric definition
	metric, err := s.getPoPMetricByID(ctx, metricID)
	if err != nil {
		return fmt.Errorf("failed to get metric: %w", err)
	}

	// Generate period-specific queries
	currentQuery := s.buildPeriodQuery(metric, periodStart, periodEnd)
	previousQuery := s.buildPreviousPeriodQuery(metric, periodStart, periodEnd)

	// Execute queries
	currentValue, currentCount, err := s.executeMetricQuery(ctx, currentQuery)
	if err != nil {
		return fmt.Errorf("failed to execute current period query: %w", err)
	}

	previousValue, _, err := s.executeMetricQuery(ctx, previousQuery)
	if err != nil {
		return fmt.Errorf("failed to execute previous period query: %w", err)
	}

	// Calculate delta and percent change
	var delta, percentChange *float64
	if currentValue != nil && previousValue != nil {
		d := *currentValue - *previousValue
		delta = &d

		if *previousValue != 0 {
			pc := (*currentValue - *previousValue) / math.Abs(*previousValue) * 100
			percentChange = &pc
		}
	}

	// Store computation result
	computation := &PoPComputation{
		MetricID:          metricID,
		PeriodStart:       periodStart,
		PeriodEnd:         periodEnd,
		Granularity:       metric.Granularity,
		PeriodLabel:       s.formatPeriodLabel(periodStart, periodEnd, metric.Granularity),
		CurrentValue:      currentValue,
		PreviousValue:     previousValue,
		Delta:             delta,
		PercentChange:     percentChange,
		RecordCount:       &currentCount,
		LastUpdated:       time.Now(),
		ComputationStatus: "success",
	}

	return s.storePoPComputation(ctx, computation)
}

// DetectAnomalies runs anomaly detection on recent computations
func (s *PoPService) DetectAnomalies(ctx context.Context, metricID string) error {
	// Get recent computations for the metric
	computations, err := s.getRecentComputations(ctx, metricID, 30) // Last 30 periods
	if err != nil {
		return fmt.Errorf("failed to get recent computations: %w", err)
	}

	if len(computations) < 7 { // Need minimum data points for anomaly detection
		return nil
	}

	// Extract values for analysis
	var values []float64
	var computationIDs []string

	for _, comp := range computations {
		if comp.CurrentValue != nil {
			values = append(values, *comp.CurrentValue)
			computationIDs = append(computationIDs, comp.ID)
		}
	}

	if len(values) < 7 {
		return nil
	}

	// Calculate z-scores
	mean, std := s.calculateMeanStd(values)
	anomalies := s.detectZScoreAnomalies(values, computationIDs, mean, std, 2.5) // 2.5 sigma threshold

	// Store detected anomalies
	for _, anomaly := range anomalies {
		if err := s.storeAnomaly(ctx, &anomaly); err != nil {
			log.Printf("Failed to store anomaly for computation %s: %v", anomaly.ComputationID, err)
		}
	}

	return nil
}

// GetPoPDashboardData retrieves data for the PoP cockpit dashboard
func (s *PoPService) GetPoPDashboardData(ctx context.Context, filters map[string]interface{}) (*PoPDashboardData, error) {
	// Get metrics with latest computations
	metrics, err := s.getMetricsWithLatestComputations(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics with computations: %w", err)
	}

	// Get anomaly summary
	anomalySummary, err := s.getAnomalySummary(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get anomaly summary: %w", err)
	}

	// Get steward review status
	reviewStatus, err := s.getStewardReviewStatus(ctx, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get steward review status: %w", err)
	}

	return &PoPDashboardData{
		Metrics:        metrics,
		AnomalySummary: anomalySummary,
		ReviewStatus:   reviewStatus,
		LastUpdated:    time.Now(),
	}, nil
}

// Helper methods

func (s *PoPService) getPoPMetricByID(ctx context.Context, metricID string) (*PoPMetric, error) {
	query := `
		SELECT id, name, display_name, description, domain, category, metric_type,
			base_query, aggregation_function, date_column, value_column, granularity,
			comparison_periods, owner_user_id, steward_group, data_source, schema_name,
			table_name, sla_freshness_hours, sla_completeness_threshold, data_quality_checks,
			status, golden_path, version, created_at, updated_at, created_by, updated_by
		FROM public.pop_metrics WHERE id = $1`

	var metric PoPMetric
	var comparisonPeriodsJSON, dataQualityChecksJSON []byte

	err := s.db.QueryRowContext(ctx, query, metricID).Scan(
		&metric.ID, &metric.Name, &metric.DisplayName, &metric.Description,
		&metric.Domain, &metric.Category, &metric.MetricType, &metric.BaseQuery,
		&metric.AggregationFunction, &metric.DateColumn, &metric.ValueColumn,
		&metric.Granularity, &comparisonPeriodsJSON, &metric.OwnerUserID,
		&metric.StewardGroup, &metric.DataSource, &metric.SchemaName,
		&metric.TableName, &metric.SLAFreshnessHours, &metric.SLACompletenessThreshold,
		&dataQualityChecksJSON, &metric.Status, &metric.GoldenPath, &metric.Version,
		&metric.CreatedAt, &metric.UpdatedAt, &metric.CreatedBy, &metric.UpdatedBy,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get PoP metric: %w", err)
	}

	// Unmarshal JSON fields
	if err := json.Unmarshal(comparisonPeriodsJSON, &metric.ComparisonPeriods); err != nil {
		return nil, fmt.Errorf("failed to unmarshal comparison periods: %w", err)
	}

	if len(dataQualityChecksJSON) > 0 {
		if err := json.Unmarshal(dataQualityChecksJSON, &metric.DataQualityChecks); err != nil {
			return nil, fmt.Errorf("failed to unmarshal data quality checks: %w", err)
		}
	}

	return &metric, nil
}

func (s *PoPService) buildPeriodQuery(metric *PoPMetric, start, end time.Time) string {
	return fmt.Sprintf(`
		SELECT %s(%s) as value, COUNT(*) as record_count
		FROM (%s) base_query
		WHERE %s >= '%s' AND %s < '%s'`,
		metric.AggregationFunction, metric.ValueColumn, metric.BaseQuery,
		metric.DateColumn, start.Format("2006-01-02"),
		metric.DateColumn, end.Format("2006-01-02"))
}

func (s *PoPService) buildPreviousPeriodQuery(metric *PoPMetric, start, end time.Time) string {
	periodDuration := end.Sub(start)
	previousStart := start.Add(-periodDuration)
	previousEnd := start

	return fmt.Sprintf(`
		SELECT %s(%s) as value, COUNT(*) as record_count
		FROM (%s) base_query
		WHERE %s >= '%s' AND %s < '%s'`,
		metric.AggregationFunction, metric.ValueColumn, metric.BaseQuery,
		metric.DateColumn, previousStart.Format("2006-01-02"),
		metric.DateColumn, previousEnd.Format("2006-01-02"))
}

func (s *PoPService) executeMetricQuery(ctx context.Context, query string) (*float64, int, error) {
	var value *float64
	var count int

	err := s.db.QueryRowContext(ctx, query).Scan(&value, &count)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to execute query: %w", err)
	}

	return value, count, nil
}

func (s *PoPService) storePoPComputation(ctx context.Context, computation *PoPComputation) error {
	query := `
		INSERT INTO public.pop_computations (
			metric_id, period_start, period_end, granularity, period_label,
			current_value, previous_value, delta, percent_change, record_count,
			last_updated, computation_status
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		ON CONFLICT (metric_id, period_start, period_end, granularity)
		DO UPDATE SET
			current_value = EXCLUDED.current_value,
			previous_value = EXCLUDED.previous_value,
			delta = EXCLUDED.delta,
			percent_change = EXCLUDED.percent_change,
			record_count = EXCLUDED.record_count,
			last_updated = EXCLUDED.last_updated,
			computation_status = EXCLUDED.computation_status`

	_, err := s.db.ExecContext(ctx, query,
		computation.MetricID, computation.PeriodStart, computation.PeriodEnd,
		computation.Granularity, computation.PeriodLabel, computation.CurrentValue,
		computation.PreviousValue, computation.Delta, computation.PercentChange,
		computation.RecordCount, computation.LastUpdated, computation.ComputationStatus)

	return err
}

func (s *PoPService) calculateMeanStd(values []float64) (float64, float64) {
	if len(values) == 0 {
		return 0, 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	sumSq := 0.0
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	variance := sumSq / float64(len(values))
	std := math.Sqrt(variance)

	return mean, std
}

func (s *PoPService) detectZScoreAnomalies(values []float64, computationIDs []string, mean, std, threshold float64) []PoPAnomaly {
	var anomalies []PoPAnomaly

	for i, value := range values {
		if std == 0 {
			continue // No variation, can't detect anomalies
		}

		zScore := math.Abs((value - mean) / std)
		if zScore >= threshold {
			severity := "low"
			if zScore >= 3 {
				severity = "high"
			} else if zScore >= 2.5 {
				severity = "medium"
			}

			anomaly := PoPAnomaly{
				ComputationID:   computationIDs[i],
				AnomalyType:     "z_score",
				Severity:        severity,
				Confidence:      &zScore,
				ZScore:          &zScore,
				ExpectedValue:   &mean,
				ActualValue:     &value,
				DetectionMethod: "z_score",
				DetectionParams: map[string]interface{}{
					"threshold": threshold,
					"mean":      mean,
					"std":       std,
				},
				DetectedAt: time.Now(),
				Status:     "open",
			}

			anomalies = append(anomalies, anomaly)
		}
	}

	return anomalies
}

func (s *PoPService) storeAnomaly(ctx context.Context, anomaly *PoPAnomaly) error {
	detectionParamsJSON, err := json.Marshal(anomaly.DetectionParams)
	if err != nil {
		return fmt.Errorf("failed to marshal detection params: %w", err)
	}

	query := `
		INSERT INTO public.pop_anomalies (
			metric_id, computation_id, anomaly_type, severity, confidence,
			z_score, expected_value, expected_range_min, expected_range_max,
			actual_value, detection_method, detection_params, detected_at, status
		) VALUES (
			(SELECT metric_id FROM public.pop_computations WHERE id = $1),
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
		ON CONFLICT (metric_id, computation_id, anomaly_type) DO NOTHING`

	_, err = s.db.ExecContext(ctx, query,
		anomaly.ComputationID, anomaly.AnomalyType, anomaly.Severity, anomaly.Confidence,
		anomaly.ZScore, anomaly.ExpectedValue, anomaly.ExpectedRangeMin, anomaly.ExpectedRangeMax,
		anomaly.ActualValue, anomaly.DetectionMethod, detectionParamsJSON,
		anomaly.DetectedAt, anomaly.Status)

	return err
}

func (s *PoPService) getRecentComputations(ctx context.Context, metricID string, limit int) ([]PoPComputation, error) {
	query := `
		SELECT id, metric_id, period_start, period_end, granularity, period_label,
			current_value, previous_value, delta, percent_change, record_count,
			last_updated, computation_status
		FROM public.pop_computations
		WHERE metric_id = $1
		ORDER BY period_end DESC
		LIMIT $2`

	rows, err := s.db.QueryContext(ctx, query, metricID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent computations: %w", err)
	}
	defer rows.Close()

	var computations []PoPComputation
	for rows.Next() {
		var comp PoPComputation
		err := rows.Scan(
			&comp.ID, &comp.MetricID, &comp.PeriodStart, &comp.PeriodEnd,
			&comp.Granularity, &comp.PeriodLabel, &comp.CurrentValue,
			&comp.PreviousValue, &comp.Delta, &comp.PercentChange,
			&comp.RecordCount, &comp.LastUpdated, &comp.ComputationStatus,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan computation: %w", err)
		}
		computations = append(computations, comp)
	}

	return computations, nil
}

func (s *PoPService) formatPeriodLabel(start, end time.Time, granularity string) string {
	switch granularity {
	case "month":
		return start.Format("2006-01")
	case "quarter":
		year := start.Year()
		quarter := (int(start.Month())-1)/3 + 1
		return fmt.Sprintf("%d-Q%d", year, quarter)
	case "year":
		return start.Format("2006")
	default:
		return fmt.Sprintf("%s to %s", start.Format("2006-01-02"), end.Format("2006-01-02"))
	}
}

// Dashboard data structures
type PoPDashboardData struct {
	Metrics        []PoPMetricWithLatest `json:"metrics"`
	AnomalySummary []AnomalySummary      `json:"anomaly_summary"`
	ReviewStatus   []ReviewStatus        `json:"review_status"`
	LastUpdated    time.Time             `json:"last_updated"`
}

type PoPMetricWithLatest struct {
	PoPMetric
	CurrentValue   *float64  `json:"current_value"`
	PreviousValue  *float64  `json:"previous_value"`
	Delta          *float64  `json:"delta"`
	PercentChange  *float64  `json:"percent_change"`
	PeriodStart    time.Time `json:"period_start"`
	PeriodEnd      time.Time `json:"period_end"`
	LastComputedAt time.Time `json:"last_computed_at"`
	HasAnomalies   bool      `json:"has_anomalies"`
	AnomalyCount   int       `json:"anomaly_count"`
}

type AnomalySummary struct {
	Domain          string    `json:"domain"`
	Category        string    `json:"category"`
	Severity        string    `json:"severity"`
	AnomalyType     string    `json:"anomaly_type"`
	AnomalyCount    int       `json:"anomaly_count"`
	LatestDetection time.Time `json:"latest_detection"`
	AffectedMetrics []string  `json:"affected_metrics"`
}

type ReviewStatus struct {
	MetricID       string     `json:"metric_id"`
	MetricName     string     `json:"metric_name"`
	ReviewStatus   string     `json:"review_status"`
	LastReviewDate *time.Time `json:"last_review_date"`
	DueDate        *time.Time `json:"due_date"`
	OverdueCount   int        `json:"overdue_count"`
}

func (s *PoPService) getMetricsWithLatestComputations(ctx context.Context, filters map[string]interface{}) ([]PoPMetricWithLatest, error) {
	query := `
		SELECT m.*, c.current_value, c.previous_value, c.delta, c.percent_change,
			c.period_start, c.period_end, c.last_updated,
			CASE WHEN a.id IS NOT NULL THEN true ELSE false END as has_anomalies,
			COALESCE(anomaly_counts.count, 0) as anomaly_count
		FROM public.pop_metrics m
		LEFT JOIN public.pop_computations c ON m.id = c.metric_id
			AND c.id = (
				SELECT id FROM public.pop_computations
				WHERE metric_id = m.id
				ORDER BY period_end DESC, last_updated DESC
				LIMIT 1
			)
		LEFT JOIN public.pop_anomalies a ON m.id = a.metric_id AND a.status = 'open'
		LEFT JOIN (
			SELECT metric_id, COUNT(*) as count
			FROM public.pop_anomalies
			WHERE status = 'open'
			GROUP BY metric_id
		) anomaly_counts ON m.id = anomaly_counts.metric_id
		WHERE m.status = 'active'`

	args := []interface{}{}
	argCount := 0

	if domain, ok := filters["domain"].(string); ok && domain != "" {
		argCount++
		query += fmt.Sprintf(" AND m.domain = $%d", argCount)
		args = append(args, domain)
	}

	query += " GROUP BY m.id, c.id, c.current_value, c.previous_value, c.delta, c.percent_change, c.period_start, c.period_end, c.last_updated, a.id, anomaly_counts.count"
	query += " ORDER BY m.domain, m.category, m.name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metrics with latest computations: %w", err)
	}
	defer rows.Close()

	var metrics []PoPMetricWithLatest
	for rows.Next() {
		var metric PoPMetricWithLatest
		var comparisonPeriodsJSON, dataQualityChecksJSON []byte
		var hasAnomalies bool

		err := rows.Scan(
			&metric.ID, &metric.Name, &metric.DisplayName, &metric.Description,
			&metric.Domain, &metric.Category, &metric.MetricType, &metric.BaseQuery,
			&metric.AggregationFunction, &metric.DateColumn, &metric.ValueColumn,
			&metric.Granularity, &comparisonPeriodsJSON, &metric.OwnerUserID,
			&metric.StewardGroup, &metric.DataSource, &metric.SchemaName,
			&metric.TableName, &metric.SLAFreshnessHours, &metric.SLACompletenessThreshold,
			&dataQualityChecksJSON, &metric.Status, &metric.GoldenPath, &metric.Version,
			&metric.CreatedAt, &metric.UpdatedAt, &metric.CreatedBy, &metric.UpdatedBy,
			&metric.CurrentValue, &metric.PreviousValue, &metric.Delta, &metric.PercentChange,
			&metric.PeriodStart, &metric.PeriodEnd, &metric.LastComputedAt,
			&hasAnomalies, &metric.AnomalyCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan metric with latest computation: %w", err)
		}

		metric.HasAnomalies = hasAnomalies

		// Unmarshal JSON fields
		if err := json.Unmarshal(comparisonPeriodsJSON, &metric.ComparisonPeriods); err != nil {
			log.Printf("Warning: failed to unmarshal comparison periods for metric %s: %v", metric.ID, err)
		}

		if len(dataQualityChecksJSON) > 0 {
			if err := json.Unmarshal(dataQualityChecksJSON, &metric.DataQualityChecks); err != nil {
				log.Printf("Warning: failed to unmarshal data quality checks for metric %s: %v", metric.ID, err)
			}
		}

		metrics = append(metrics, metric)
	}

	return metrics, nil
}

func (s *PoPService) getAnomalySummary(ctx context.Context, filters map[string]interface{}) ([]AnomalySummary, error) {
	query := `
		SELECT m.domain, m.category, a.severity, a.anomaly_type,
			COUNT(*) as anomaly_count, MAX(a.detected_at) as latest_detection,
			ARRAY_AGG(DISTINCT m.name) as affected_metrics
		FROM public.pop_anomalies a
		JOIN public.pop_metrics m ON a.metric_id = m.id
		WHERE a.status = 'open'`

	args := []interface{}{}
	argCount := 0

	if domain, ok := filters["domain"].(string); ok && domain != "" {
		argCount++
		query += fmt.Sprintf(" AND m.domain = $%d", argCount)
		args = append(args, domain)
	}

	query += " GROUP BY m.domain, m.category, a.severity, a.anomaly_type"
	query += " ORDER BY m.domain, m.category, a.severity"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query anomaly summary: %w", err)
	}
	defer rows.Close()

	var summaries []AnomalySummary
	for rows.Next() {
		var summary AnomalySummary
		var affectedMetrics []string

		err := rows.Scan(
			&summary.Domain, &summary.Category, &summary.Severity,
			&summary.AnomalyType, &summary.AnomalyCount, &summary.LatestDetection,
			pq.Array(&affectedMetrics),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan anomaly summary: %w", err)
		}

		summary.AffectedMetrics = affectedMetrics
		summaries = append(summaries, summary)
	}

	return summaries, nil
}

func (s *PoPService) getStewardReviewStatus(ctx context.Context, filters map[string]interface{}) ([]ReviewStatus, error) {
	query := `
		SELECT m.id, m.name,
			COALESCE(r.status, 'no_reviews') as review_status,
			MAX(r.completed_at) as last_review_date,
			MAX(r.due_date) as due_date,
			COUNT(CASE WHEN r.due_date < NOW() AND r.status != 'completed' THEN 1 END) as overdue_count
		FROM public.pop_metrics m
		LEFT JOIN public.pop_steward_reviews r ON m.id = r.metric_id
		WHERE m.status = 'active'`

	args := []interface{}{}
	argCount := 0

	if domain, ok := filters["domain"].(string); ok && domain != "" {
		argCount++
		query += fmt.Sprintf(" AND m.domain = $%d", argCount)
		args = append(args, domain)
	}

	query += " GROUP BY m.id, m.name, r.status"
	query += " ORDER BY m.domain, m.category, m.name"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query steward review status: %w", err)
	}
	defer rows.Close()

	var statuses []ReviewStatus
	for rows.Next() {
		var status ReviewStatus

		err := rows.Scan(
			&status.MetricID, &status.MetricName, &status.ReviewStatus,
			&status.LastReviewDate, &status.DueDate, &status.OverdueCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan review status: %w", err)
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}
