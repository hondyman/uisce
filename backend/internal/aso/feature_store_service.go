package aso

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Feature Store Types
// ============================================================================

// BOFeatures contains aggregated workload features for a Business Object
type BOFeatures struct {
	Env                   string    `json:"env" db:"env"`
	TenantID              uuid.UUID `json:"tenant_id" db:"tenant_id"`
	BOID                  uuid.UUID `json:"bo_id" db:"bo_id"`
	BOName                string    `json:"bo_name" db:"bo_name"`
	Window                string    `json:"window" db:"window"`
	Queries               int       `json:"queries" db:"queries"`
	QueriesPerDay         float64   `json:"queries_per_day" db:"queries_per_day"`
	DistinctUsers         int       `json:"distinct_users" db:"distinct_users"`
	DistinctQueryPatterns int       `json:"distinct_query_patterns" db:"distinct_query_patterns"`
	AvgLatencyMs          float64   `json:"avg_latency_ms" db:"avg_latency_ms"`
	P50LatencyMs          float64   `json:"p50_latency_ms" db:"p50_latency_ms"`
	P95LatencyMs          float64   `json:"p95_latency_ms" db:"p95_latency_ms"`
	P99LatencyMs          float64   `json:"p99_latency_ms" db:"p99_latency_ms"`
	AvgScanBytes          float64   `json:"avg_scan_bytes" db:"avg_scan_bytes"`
	AvgRowsScanned        float64   `json:"avg_rows_scanned" db:"avg_rows_scanned"`
	PreAggHitRate         float64   `json:"preagg_hit_rate" db:"preagg_hit_rate"`
	PreAggMissRate        float64   `json:"preagg_miss_rate" db:"preagg_miss_rate"`
	PreAggMissQueries     int       `json:"preagg_miss_queries" db:"preagg_miss_queries"`
	PeakHour              *int      `json:"peak_hour,omitempty" db:"peak_hour"`
	PeakDayOfWeek         *int      `json:"peak_day_of_week,omitempty" db:"peak_day_of_week"`
	LastUpdated           time.Time `json:"last_updated" db:"last_updated"`
}

// PreAggFeatures contains performance features for a pre-aggregation
type PreAggFeatures struct {
	Env                 string     `json:"env" db:"env"`
	TenantID            *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	PreAggID            uuid.UUID  `json:"preagg_id" db:"preagg_id"`
	PreAggName          string     `json:"preagg_name" db:"preagg_name"`
	Window              string     `json:"window" db:"window"`
	QueriesAccelerated  int        `json:"queries_accelerated" db:"queries_accelerated"`
	AvgSpeedup          float64    `json:"avg_speedup" db:"avg_speedup"`
	HitRate             float64    `json:"hit_rate" db:"hit_rate"`
	StorageBytes        int64      `json:"storage_bytes" db:"storage_bytes"`
	RowCount            int64      `json:"row_count" db:"row_count"`
	RefreshCostMs       float64    `json:"refresh_cost_ms" db:"refresh_cost_ms"`
	RefreshFrequencySec int        `json:"refresh_frequency_sec" db:"refresh_frequency_sec"`
	LastRefreshAt       *time.Time `json:"last_refresh_at,omitempty" db:"last_refresh_at"`
	RefreshFailureCount int        `json:"refresh_failure_count" db:"refresh_failure_count"`
	UsageTrendPct       float64    `json:"usage_trend_pct" db:"usage_trend_pct"`
	DaysSinceLastUse    int        `json:"days_since_last_use" db:"days_since_last_use"`
	LastUpdated         time.Time  `json:"last_updated" db:"last_updated"`
}

// OptimizationFeatures is the complete feature vector for ML scoring
type OptimizationFeatures struct {
	OptimizationID uuid.UUID  `json:"optimization_id" db:"optimization_id"`
	Env            string     `json:"env" db:"env"`
	TenantID       *uuid.UUID `json:"tenant_id,omitempty" db:"tenant_id"`
	Type           string     `json:"type" db:"type"`
	TargetType     string     `json:"target_type" db:"target_type"`
	TargetID       uuid.UUID  `json:"target_id" db:"target_id"`
	TargetName     string     `json:"target_name" db:"target_name"`
	BOName         *string    `json:"bo_name,omitempty" db:"bo_name"`
	Window         string     `json:"window" db:"window"`

	// BO Features
	BOQueries        *int     `json:"bo_queries,omitempty" db:"bo_queries"`
	BOQueriesPerDay  *float64 `json:"bo_queries_per_day,omitempty" db:"bo_queries_per_day"`
	BODistinctUsers  *int     `json:"bo_distinct_users,omitempty" db:"bo_distinct_users"`
	BOAvgLatencyMs   *float64 `json:"bo_avg_latency_ms,omitempty" db:"bo_avg_latency_ms"`
	BOP95LatencyMs   *float64 `json:"bo_p95_latency_ms,omitempty" db:"bo_p95_latency_ms"`
	BOAvgScanBytes   *float64 `json:"bo_avg_scan_bytes,omitempty" db:"bo_avg_scan_bytes"`
	BOPreAggMissRate *float64 `json:"bo_preagg_miss_rate,omitempty" db:"bo_preagg_miss_rate"`

	// PreAgg Features
	PreAggQueriesAccelerated  *int     `json:"preagg_queries_accelerated,omitempty" db:"preagg_queries_accelerated"`
	PreAggAvgSpeedup          *float64 `json:"preagg_avg_speedup,omitempty" db:"preagg_avg_speedup"`
	PreAggStorageBytes        *int64   `json:"preagg_storage_bytes,omitempty" db:"preagg_storage_bytes"`
	PreAggRefreshCostMs       *float64 `json:"preagg_refresh_cost_ms,omitempty" db:"preagg_refresh_cost_ms"`
	PreAggRefreshFrequencySec *int     `json:"preagg_refresh_frequency_sec,omitempty" db:"preagg_refresh_frequency_sec"`
	PreAggHitRate             *float64 `json:"preagg_hit_rate,omitempty" db:"preagg_hit_rate"`
	PreAggUsageTrendPct       *float64 `json:"preagg_usage_trend_pct,omitempty" db:"preagg_usage_trend_pct"`

	// Simulation Predictions
	SimExpectedSpeedup     *float64 `json:"sim_expected_speedup,omitempty" db:"sim_expected_speedup"`
	SimExpectedCostSavings *float64 `json:"sim_expected_cost_savings,omitempty" db:"sim_expected_cost_savings"`
	SimQueriesImproved     *int     `json:"sim_queries_improved,omitempty" db:"sim_queries_improved"`
	SimQueriesRegressed    *int     `json:"sim_queries_regressed,omitempty" db:"sim_queries_regressed"`
	SimHitRateBefore       *float64 `json:"sim_hit_rate_before,omitempty" db:"sim_hit_rate_before"`
	SimHitRateAfter        *float64 `json:"sim_hit_rate_after,omitempty" db:"sim_hit_rate_after"`

	// ML Predictions
	MLScore                *float64        `json:"ml_score,omitempty" db:"ml_score"`
	MLPredictedSpeedup     *float64        `json:"ml_predicted_speedup,omitempty" db:"ml_predicted_speedup"`
	MLPredictedCostSavings *float64        `json:"ml_predicted_cost_savings,omitempty" db:"ml_predicted_cost_savings"`
	MLRiskScore            *float64        `json:"ml_risk_score,omitempty" db:"ml_risk_score"`
	MLConfidence           *float64        `json:"ml_confidence,omitempty" db:"ml_confidence"`
	MLTopFactors           json.RawMessage `json:"ml_top_factors,omitempty" db:"ml_top_factors"`

	// Labels
	RealizedSpeedup          *float64   `json:"realized_speedup,omitempty" db:"realized_speedup"`
	RealizedCostSavings      *float64   `json:"realized_cost_savings,omitempty" db:"realized_cost_savings"`
	RealizedRegression       *bool      `json:"realized_regression,omitempty" db:"realized_regression"`
	RealizedLatencyChangePct *float64   `json:"realized_latency_change_pct,omitempty" db:"realized_latency_change_pct"`
	RealizedHitRateChange    *float64   `json:"realized_hit_rate_change,omitempty" db:"realized_hit_rate_change"`
	LabelRecordedAt          *time.Time `json:"label_recorded_at,omitempty" db:"label_recorded_at"`
	LabelReady               bool       `json:"label_ready" db:"label_ready"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// TopFactor represents a feature's contribution to the ML score
type TopFactor struct {
	Feature   string  `json:"feature"`
	Weight    float64 `json:"weight"`
	Direction string  `json:"direction"` // positive, negative
}

// ============================================================================
// Feature Store Service
// ============================================================================

// FeatureStoreService manages ML feature computation and storage
type FeatureStoreService interface {
	// Feature Computation
	ComputeBOFeatures(ctx context.Context, env string, tenantID uuid.UUID, window string) error
	ComputePreAggFeatures(ctx context.Context, env string, tenantID *uuid.UUID, window string) error
	ComputeAllFeatures(ctx context.Context, env string, window string) error

	// Feature Retrieval
	GetBOFeatures(ctx context.Context, env string, tenantID, boID uuid.UUID, window string) (*BOFeatures, error)
	GetPreAggFeatures(ctx context.Context, env string, preAggID uuid.UUID, window string) (*PreAggFeatures, error)

	// Optimization Features
	BuildOptimizationFeatures(ctx context.Context, opt *ASOOptimization) (*OptimizationFeatures, error)
	SaveOptimizationFeatures(ctx context.Context, features *OptimizationFeatures) error

	// Label Recording
	RecordLabels(ctx context.Context, optID uuid.UUID, speedup, costSavings float64, regression bool) error
}

type featureStoreService struct {
	db        *sqlx.DB
	telemetry TelemetryService
	config    *ASOConfig
}

// NewFeatureStoreService creates a new feature store service
func NewFeatureStoreService(db *sqlx.DB, telemetry TelemetryService, config *ASOConfig) FeatureStoreService {
	if config == nil {
		config = DefaultConfig()
	}
	return &featureStoreService{
		db:        db,
		telemetry: telemetry,
		config:    config,
	}
}

// ComputeBOFeatures aggregates BO workload features from telemetry
func (s *featureStoreService) ComputeBOFeatures(ctx context.Context, env string, tenantID uuid.UUID, window string) error {
	// Parse window duration
	windowDuration := parseDuration(window)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO aso.bo_features (
			env, tenant_id, bo_id, bo_name, window,
			queries, queries_per_day, distinct_users, distinct_query_patterns,
			avg_latency_ms, p50_latency_ms, p95_latency_ms, p99_latency_ms,
			avg_scan_bytes, avg_rows_scanned,
			preagg_hit_rate, preagg_miss_rate, preagg_miss_queries,
			peak_hour, peak_day_of_week, last_updated
		)
		SELECT
			$1 AS env,
			$2::uuid AS tenant_id,
			bo_id,
			bo_name,
			$3 AS window,
			COUNT(*) AS queries,
			COUNT(*) * 1.0 / GREATEST($4, 1) AS queries_per_day,
			COUNT(DISTINCT user_id) AS distinct_users,
			COUNT(DISTINCT query_hash) AS distinct_query_patterns,
			COALESCE(AVG(latency_ms), 0) AS avg_latency_ms,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY latency_ms), 0) AS p50_latency_ms,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms), 0) AS p95_latency_ms,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms), 0) AS p99_latency_ms,
			COALESCE(AVG(scan_bytes), 0) AS avg_scan_bytes,
			COALESCE(AVG(rows_scanned), 0) AS avg_rows_scanned,
			COALESCE(AVG(CASE WHEN preagg_hit THEN 1.0 ELSE 0.0 END), 0) AS preagg_hit_rate,
			COALESCE(AVG(CASE WHEN preagg_hit THEN 0.0 ELSE 1.0 END), 0) AS preagg_miss_rate,
			COUNT(*) FILTER (WHERE NOT preagg_hit OR preagg_hit IS NULL) AS preagg_miss_queries,
			MODE() WITHIN GROUP (ORDER BY EXTRACT(HOUR FROM created_at)::int) AS peak_hour,
			MODE() WITHIN GROUP (ORDER BY EXTRACT(DOW FROM created_at)::int) AS peak_day_of_week,
			now() AS last_updated
		FROM query_telemetry
		WHERE env = $1
		  AND tenant_id = $2
		  AND created_at >= now() - $5::interval
		GROUP BY bo_id, bo_name
		ON CONFLICT (env, tenant_id, bo_id, window)
		DO UPDATE SET
			queries = EXCLUDED.queries,
			queries_per_day = EXCLUDED.queries_per_day,
			distinct_users = EXCLUDED.distinct_users,
			distinct_query_patterns = EXCLUDED.distinct_query_patterns,
			avg_latency_ms = EXCLUDED.avg_latency_ms,
			p50_latency_ms = EXCLUDED.p50_latency_ms,
			p95_latency_ms = EXCLUDED.p95_latency_ms,
			p99_latency_ms = EXCLUDED.p99_latency_ms,
			avg_scan_bytes = EXCLUDED.avg_scan_bytes,
			avg_rows_scanned = EXCLUDED.avg_rows_scanned,
			preagg_hit_rate = EXCLUDED.preagg_hit_rate,
			preagg_miss_rate = EXCLUDED.preagg_miss_rate,
			preagg_miss_queries = EXCLUDED.preagg_miss_queries,
			peak_hour = EXCLUDED.peak_hour,
			peak_day_of_week = EXCLUDED.peak_day_of_week,
			last_updated = EXCLUDED.last_updated
	`, env, tenantID, window, windowDuration.Hours()/24, window)

	return err
}

// ComputePreAggFeatures aggregates pre-agg performance features
func (s *featureStoreService) ComputePreAggFeatures(ctx context.Context, env string, tenantID *uuid.UUID, window string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO aso.preagg_features (
			env, tenant_id, preagg_id, preagg_name, window,
			queries_accelerated, avg_speedup, hit_rate,
			storage_bytes, row_count,
			refresh_cost_ms, refresh_frequency_sec, last_refresh_at, refresh_failure_count,
			usage_trend_pct, days_since_last_use, last_updated
		)
		SELECT
			$1 AS env,
			$2::uuid AS tenant_id,
			p.id AS preagg_id,
			p.name AS preagg_name,
			$3 AS window,
			COALESCE(qt.queries_accelerated, 0),
			COALESCE(qt.avg_speedup, 1.0),
			COALESCE(qt.hit_rate, 0),
			COALESCE(p.storage_bytes, 0),
			COALESCE(p.row_count, 0),
			COALESCE(r.avg_refresh_ms, 0),
			COALESCE(EXTRACT(EPOCH FROM p.refresh_interval)::int, 3600),
			r.last_refresh,
			COALESCE(r.failure_count, 0),
			COALESCE(qt.trend_pct, 0),
			COALESCE(EXTRACT(DAY FROM now() - qt.last_query)::int, 365),
			now() AS last_updated
		FROM semantic.preagg_metadata p
		LEFT JOIN LATERAL (
			SELECT 
				COUNT(*) AS queries_accelerated,
				AVG(speedup_factor) AS avg_speedup,
				AVG(CASE WHEN preagg_hit THEN 1.0 ELSE 0.0 END) AS hit_rate,
				MAX(created_at) AS last_query,
				-- Trend: compare last 7 days to previous 7 days
				(COUNT(*) FILTER (WHERE created_at >= now() - interval '7 days') -
				 COUNT(*) FILTER (WHERE created_at >= now() - interval '14 days' AND created_at < now() - interval '7 days')
				) * 100.0 / NULLIF(COUNT(*) FILTER (WHERE created_at >= now() - interval '14 days' AND created_at < now() - interval '7 days'), 0) AS trend_pct
			FROM query_telemetry
			WHERE preagg_id = p.id
			  AND created_at >= now() - $4::interval
		) qt ON true
		LEFT JOIN LATERAL (
			SELECT 
				AVG(duration_ms) AS avg_refresh_ms,
				MAX(started_at) AS last_refresh,
				COUNT(*) FILTER (WHERE status = 'failed') AS failure_count
			FROM preagg_refresh_log
			WHERE preagg_id = p.id
			  AND started_at >= now() - $4::interval
		) r ON true
		WHERE p.env = $1
		  AND ($2::uuid IS NULL OR p.tenant_id = $2)
		ON CONFLICT (env, preagg_id, window)
		DO UPDATE SET
			queries_accelerated = EXCLUDED.queries_accelerated,
			avg_speedup = EXCLUDED.avg_speedup,
			hit_rate = EXCLUDED.hit_rate,
			storage_bytes = EXCLUDED.storage_bytes,
			row_count = EXCLUDED.row_count,
			refresh_cost_ms = EXCLUDED.refresh_cost_ms,
			refresh_frequency_sec = EXCLUDED.refresh_frequency_sec,
			last_refresh_at = EXCLUDED.last_refresh_at,
			refresh_failure_count = EXCLUDED.refresh_failure_count,
			usage_trend_pct = EXCLUDED.usage_trend_pct,
			days_since_last_use = EXCLUDED.days_since_last_use,
			last_updated = EXCLUDED.last_updated
	`, env, tenantID, window, window)

	return err
}

// ComputeAllFeatures runs feature computation for all tenants
func (s *featureStoreService) ComputeAllFeatures(ctx context.Context, env string, window string) error {
	// Get all tenants
	var tenantIDs []uuid.UUID
	err := s.db.SelectContext(ctx, &tenantIDs, `
		SELECT DISTINCT id FROM tenants WHERE env = $1
	`, env)
	if err != nil {
		return err
	}

	// Compute BO features for each tenant
	for _, tid := range tenantIDs {
		if err := s.ComputeBOFeatures(ctx, env, tid, window); err != nil {
			// Log but continue
			continue
		}
	}

	// Compute pre-agg features (core + tenant)
	if err := s.ComputePreAggFeatures(ctx, env, nil, window); err != nil {
		return err
	}

	return nil
}

// GetBOFeatures retrieves BO features
func (s *featureStoreService) GetBOFeatures(ctx context.Context, env string, tenantID, boID uuid.UUID, window string) (*BOFeatures, error) {
	var features BOFeatures
	err := s.db.GetContext(ctx, &features, `
		SELECT * FROM aso.bo_features
		WHERE env = $1 AND tenant_id = $2 AND bo_id = $3 AND window = $4
	`, env, tenantID, boID, window)
	if err != nil {
		return nil, err
	}
	return &features, nil
}

// GetPreAggFeatures retrieves pre-agg features
func (s *featureStoreService) GetPreAggFeatures(ctx context.Context, env string, preAggID uuid.UUID, window string) (*PreAggFeatures, error) {
	var features PreAggFeatures
	err := s.db.GetContext(ctx, &features, `
		SELECT * FROM aso.preagg_features
		WHERE env = $1 AND preagg_id = $2 AND window = $3
	`, env, preAggID, window)
	if err != nil {
		return nil, err
	}
	return &features, nil
}

// BuildOptimizationFeatures constructs the feature vector for an optimization
func (s *featureStoreService) BuildOptimizationFeatures(ctx context.Context, opt *ASOOptimization) (*OptimizationFeatures, error) {
	features := &OptimizationFeatures{
		OptimizationID: opt.ID,
		Env:            opt.Env,
		TenantID:       opt.TenantID,
		Type:           string(opt.OptimizationType),
		TargetType:     string(opt.TargetType),
		TargetID:       opt.TargetID,
		TargetName:     opt.TargetName,
		Window:         "7d",
		CreatedAt:      time.Now(),
	}

	// Get BO features if available
	if opt.TenantID != nil {
		boFeatures, err := s.GetBOFeatures(ctx, opt.Env, *opt.TenantID, opt.TargetID, "7d")
		if err == nil && boFeatures != nil {
			features.BOName = &boFeatures.BOName
			features.BOQueries = &boFeatures.Queries
			features.BOQueriesPerDay = &boFeatures.QueriesPerDay
			features.BODistinctUsers = &boFeatures.DistinctUsers
			features.BOAvgLatencyMs = &boFeatures.AvgLatencyMs
			features.BOP95LatencyMs = &boFeatures.P95LatencyMs
			features.BOAvgScanBytes = &boFeatures.AvgScanBytes
			features.BOPreAggMissRate = &boFeatures.PreAggMissRate
		}
	}

	// Get pre-agg features if target is a pre-agg
	if opt.TargetType == TargetTypePreAgg {
		paFeatures, err := s.GetPreAggFeatures(ctx, opt.Env, opt.TargetID, "7d")
		if err == nil && paFeatures != nil {
			features.PreAggQueriesAccelerated = &paFeatures.QueriesAccelerated
			features.PreAggAvgSpeedup = &paFeatures.AvgSpeedup
			features.PreAggStorageBytes = &paFeatures.StorageBytes
			features.PreAggRefreshCostMs = &paFeatures.RefreshCostMs
			features.PreAggRefreshFrequencySec = &paFeatures.RefreshFrequencySec
			features.PreAggHitRate = &paFeatures.HitRate
			features.PreAggUsageTrendPct = &paFeatures.UsageTrendPct
		}
	}

	return features, nil
}

// SaveOptimizationFeatures persists the feature vector
func (s *featureStoreService) SaveOptimizationFeatures(ctx context.Context, features *OptimizationFeatures) error {
	topFactorsJSON, _ := json.Marshal(features.MLTopFactors)

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO aso.optimization_features (
			optimization_id, env, tenant_id, type, target_type, target_id, target_name, bo_name, window,
			bo_queries, bo_queries_per_day, bo_distinct_users, bo_avg_latency_ms, bo_p95_latency_ms, bo_avg_scan_bytes, bo_preagg_miss_rate,
			preagg_queries_accelerated, preagg_avg_speedup, preagg_storage_bytes, preagg_refresh_cost_ms, preagg_refresh_frequency_sec, preagg_hit_rate, preagg_usage_trend_pct,
			sim_expected_speedup, sim_expected_cost_savings, sim_queries_improved, sim_queries_regressed, sim_hit_rate_before, sim_hit_rate_after,
			ml_score, ml_predicted_speedup, ml_predicted_cost_savings, ml_risk_score, ml_confidence, ml_top_factors,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9,
			$10, $11, $12, $13, $14, $15, $16,
			$17, $18, $19, $20, $21, $22, $23,
			$24, $25, $26, $27, $28, $29,
			$30, $31, $32, $33, $34, $35,
			$36
		)
		ON CONFLICT (optimization_id) DO UPDATE SET
			ml_score = EXCLUDED.ml_score,
			ml_predicted_speedup = EXCLUDED.ml_predicted_speedup,
			ml_predicted_cost_savings = EXCLUDED.ml_predicted_cost_savings,
			ml_risk_score = EXCLUDED.ml_risk_score,
			ml_confidence = EXCLUDED.ml_confidence,
			ml_top_factors = EXCLUDED.ml_top_factors
	`, features.OptimizationID, features.Env, features.TenantID, features.Type, features.TargetType, features.TargetID, features.TargetName, features.BOName, features.Window,
		features.BOQueries, features.BOQueriesPerDay, features.BODistinctUsers, features.BOAvgLatencyMs, features.BOP95LatencyMs, features.BOAvgScanBytes, features.BOPreAggMissRate,
		features.PreAggQueriesAccelerated, features.PreAggAvgSpeedup, features.PreAggStorageBytes, features.PreAggRefreshCostMs, features.PreAggRefreshFrequencySec, features.PreAggHitRate, features.PreAggUsageTrendPct,
		features.SimExpectedSpeedup, features.SimExpectedCostSavings, features.SimQueriesImproved, features.SimQueriesRegressed, features.SimHitRateBefore, features.SimHitRateAfter,
		features.MLScore, features.MLPredictedSpeedup, features.MLPredictedCostSavings, features.MLRiskScore, features.MLConfidence, topFactorsJSON,
		features.CreatedAt)

	return err
}

// RecordLabels records actual outcomes for ML training
func (s *featureStoreService) RecordLabels(ctx context.Context, optID uuid.UUID, speedup, costSavings float64, regression bool) error {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
		UPDATE aso.optimization_features
		SET realized_speedup = $2,
		    realized_cost_savings = $3,
		    realized_regression = $4,
		    label_recorded_at = $5,
		    label_ready = true
		WHERE optimization_id = $1
	`, optID, speedup, costSavings, regression, now)

	return err
}

// parseDuration converts window string to time.Duration
func parseDuration(window string) time.Duration {
	switch window {
	case "7d":
		return 7 * 24 * time.Hour
	case "30d":
		return 30 * 24 * time.Hour
	case "90d":
		return 90 * 24 * time.Hour
	default:
		return 7 * 24 * time.Hour
	}
}
