package cbo

import (
	"context"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// DBTelemetryRepository provides telemetry data for cost estimation
type DBTelemetryRepository struct {
	db *sqlx.DB
}

// NewDBTelemetryRepository creates a new database-backed telemetry repository
func NewDBTelemetryRepository(db *sqlx.DB) *DBTelemetryRepository {
	return &DBTelemetryRepository{db: db}
}

// GetBOFeatures returns aggregated features for a business object
func (r *DBTelemetryRepository) GetBOFeatures(ctx context.Context, env string, tenantID *uuid.UUID, boName string, window string) (*BOFeatures, error) {
	// First try to get cached features
	query := `
		SELECT 
			env, tenant_id, bo_name, time_window,
			p50_latency_ms, p95_latency_ms, p99_latency_ms,
			avg_scan_bytes, query_count, error_rate,
			cache_hit_rate, preagg_hit_rate, last_query_at
		FROM bo_features
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND bo_name = $3
		  AND time_window = $4
		ORDER BY tenant_id NULLS LAST
		LIMIT 1
	`

	var features BOFeatures
	err := r.db.QueryRowxContext(ctx, query, env, tenantID, boName, window).Scan(
		&features.Env, &features.TenantID, &features.BOName, &features.Window,
		&features.P50LatencyMs, &features.P95LatencyMs, &features.P99LatencyMs,
		&features.AvgScanBytes, &features.QueryCount, &features.ErrorRate,
		&features.CacheHitRate, &features.PreAggHitRate, &features.LastQueryAt,
	)
	if err == nil {
		return &features, nil
	}

	// Otherwise compute from planner telemetry
	return r.computeBOFeatures(ctx, env, tenantID, boName, window)
}

// computeBOFeatures computes BO features from telemetry
func (r *DBTelemetryRepository) computeBOFeatures(ctx context.Context, env string, tenantID *uuid.UUID, boName string, window string) (*BOFeatures, error) {
	query := `
		SELECT 
			PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY actual_latency_ms) AS p50_latency,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY actual_latency_ms) AS p95_latency,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY actual_latency_ms) AS p99_latency,
			COALESCE(AVG(actual_scan_bytes), 0) AS avg_scan_bytes,
			COUNT(*) AS query_count,
			COALESCE(SUM(CASE WHEN success = false THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0) AS error_rate,
			COALESCE(SUM(CASE WHEN plan_type = 'cached' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0) AS cache_hit_rate,
			COALESCE(SUM(CASE WHEN plan_type = 'preagg' THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0) AS preagg_hit_rate,
			MAX(created_at) AS last_query_at
		FROM planner_telemetry
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND bo_name = $3
		  AND created_at >= NOW() - $4::interval
	`

	var features BOFeatures
	features.Env = env
	features.BOName = boName
	features.Window = window
	if tenantID != nil {
		tid := tenantID.String()
		features.TenantID = &tid
	}

	err := r.db.QueryRowxContext(ctx, query, env, tenantID, boName, window).Scan(
		&features.P50LatencyMs, &features.P95LatencyMs, &features.P99LatencyMs,
		&features.AvgScanBytes, &features.QueryCount, &features.ErrorRate,
		&features.CacheHitRate, &features.PreAggHitRate, &features.LastQueryAt,
	)
	if err != nil {
		// Return defaults if no telemetry
		return &BOFeatures{
			Env:           env,
			BOName:        boName,
			Window:        window,
			P50LatencyMs:  200,
			P95LatencyMs:  500,
			P99LatencyMs:  1000,
			AvgScanBytes:  1_000_000,
			QueryCount:    0,
			ErrorRate:     0,
			CacheHitRate:  0,
			PreAggHitRate: 0,
		}, nil
	}

	return &features, nil
}

// GetPreAggFeatures returns aggregated features for a pre-aggregation
func (r *DBTelemetryRepository) GetPreAggFeatures(ctx context.Context, env string, tenantID *uuid.UUID, preAggName string, window string) (*PreAggFeatures, error) {
	// First try to get cached features
	query := `
		SELECT 
			env, tenant_id, preagg_name, time_window,
			avg_speedup, hit_count, miss_count, hit_rate,
			storage_bytes, refresh_frequency_sec, last_refresh_at, avg_freshness_lag_sec
		FROM preagg_features
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND preagg_name = $3
		  AND time_window = $4
		ORDER BY tenant_id NULLS LAST
		LIMIT 1
	`

	var features PreAggFeatures
	err := r.db.QueryRowxContext(ctx, query, env, tenantID, preAggName, window).Scan(
		&features.Env, &features.TenantID, &features.PreAggName, &features.Window,
		&features.AvgSpeedup, &features.HitCount, &features.MissCount, &features.HitRate,
		&features.StorageBytes, &features.RefreshFrequencySec, &features.LastRefreshAt, &features.AvgFreshnessLagSec,
	)
	if err == nil {
		return &features, nil
	}

	// Otherwise compute from telemetry
	return r.computePreAggFeatures(ctx, env, tenantID, preAggName, window)
}

// computePreAggFeatures computes pre-agg features from telemetry
func (r *DBTelemetryRepository) computePreAggFeatures(ctx context.Context, env string, tenantID *uuid.UUID, preAggName string, window string) (*PreAggFeatures, error) {
	// Get hit/miss counts
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE preagg_name = $3) AS hit_count,
			COUNT(*) FILTER (WHERE preagg_name IS NULL OR preagg_name != $3) AS miss_count
		FROM planner_telemetry
		WHERE env = $1
		  AND (tenant_id = $2 OR (tenant_id IS NULL AND $2 IS NULL))
		  AND created_at >= NOW() - $4::interval
	`

	var hitCount, missCount int64
	err := r.db.QueryRowxContext(ctx, query, env, tenantID, preAggName, window).Scan(&hitCount, &missCount)
	if err != nil {
		hitCount = 0
		missCount = 0
	}

	total := hitCount + missCount
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(hitCount) / float64(total)
	}

	features := &PreAggFeatures{
		Env:        env,
		PreAggName: preAggName,
		Window:     window,
		HitCount:   hitCount,
		MissCount:  missCount,
		HitRate:    hitRate,
		AvgSpeedup: 2.0, // Default assumption
	}
	if tenantID != nil {
		tid := tenantID.String()
		features.TenantID = &tid
	}

	return features, nil
}

// RecordPlannerTelemetry records a query execution for telemetry
func (r *DBTelemetryRepository) RecordPlannerTelemetry(ctx context.Context, t *PlannerTelemetryRecord) error {
	query := `
		INSERT INTO planner_telemetry (
			env, tenant_id, bo_name, plan_type, preagg_name, entitlement_strategy,
			estimated_latency_ms, actual_latency_ms, estimated_scan_bytes, actual_scan_bytes,
			slo_satisfied, candidates_evaluated, planning_time_ms, success, error_message, user_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
	`

	_, err := r.db.ExecContext(ctx, query,
		t.Env, t.TenantID, t.BOName, t.PlanType, t.PreAggName, t.EntitlementStrategy,
		t.EstimatedLatencyMs, t.ActualLatencyMs, t.EstimatedScanBytes, t.ActualScanBytes,
		t.SLOSatisfied, t.CandidatesEvaluated, t.PlanningTimeMs, t.Success, t.ErrorMessage, t.UserID,
	)
	return err
}

// SemanticEventRecord represents a record in the semantic_events table
type SemanticEventRecord struct {
	TenantID       uuid.UUID  `json:"tenant_id"`
	Datasource     string     `json:"datasource"`
	SQLFingerprint string     `json:"sql_fingerprint"`
	SQLLatencyMs   float64    `json:"sql_latency_ms"`
	SQLRows        int        `json:"sql_rows"`
	GroupByFields  string     `json:"groupby_fields"` // JSON array
	FilterFields   string     `json:"filter_fields"`  // JSON array
	MeasureFields  string     `json:"measure_fields"` // JSON array
	PreAggID       *uuid.UUID `json:"preagg_id,omitempty"`
	PreAggHit      bool       `json:"preagg_hit"`
}

// RecordSemanticEvent logs a semantic event for the suggestion engine
func (r *DBTelemetryRepository) RecordSemanticEvent(ctx context.Context, e *SemanticEventRecord) error {
	query := `
		INSERT INTO semantic_events (
			tenant_id, datasource, sql_fingerprint,
			sql_latency_ms, sql_rows,
			groupby_fields, filter_fields, measure_fields,
			preagg_id, preagg_hit
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`
	_, err := r.db.ExecContext(ctx, query,
		e.TenantID, e.Datasource, e.SQLFingerprint,
		e.SQLLatencyMs, e.SQLRows,
		e.GroupByFields, e.FilterFields, e.MeasureFields,
		e.PreAggID, e.PreAggHit,
	)
	return err
}

// PlannerTelemetryRecord represents a telemetry record for the planner
type PlannerTelemetryRecord struct {
	Env                 string     `json:"env"`
	TenantID            *uuid.UUID `json:"tenant_id,omitempty"`
	BOName              string     `json:"bo_name"`
	PlanType            string     `json:"plan_type"`
	PreAggName          *string    `json:"preagg_name,omitempty"`
	PreAggID            *uuid.UUID `json:"preagg_id,omitempty"` // Added for semantic_events correlation
	EntitlementStrategy string     `json:"entitlement_strategy"`
	EstimatedLatencyMs  float64    `json:"estimated_latency_ms"`
	ActualLatencyMs     float64    `json:"actual_latency_ms"`
	EstimatedScanBytes  float64    `json:"estimated_scan_bytes"`
	ActualScanBytes     float64    `json:"actual_scan_bytes"`
	SLOSatisfied        bool       `json:"slo_satisfied"`
	CandidatesEvaluated int        `json:"candidates_evaluated"`
	PlanningTimeMs      float64    `json:"planning_time_ms"`
	Success             bool       `json:"success"`
	ErrorMessage        *string    `json:"error_message,omitempty"`
	UserID              string     `json:"user_id"`
}
