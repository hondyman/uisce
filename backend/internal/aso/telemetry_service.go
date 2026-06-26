package aso

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Telemetry Service - Production Integration
// ============================================================================

// TelemetryService provides access to query telemetry data
type TelemetryService interface {
	// GetQueryMetrics returns query metrics for a target
	GetQueryMetrics(ctx context.Context, targetID uuid.UUID, period time.Duration) (*QueryMetrics, error)

	// GetMissRate returns pre-agg miss rate for a target
	GetMissRate(ctx context.Context, targetID uuid.UUID, period time.Duration) (*MissRateMetrics, error)

	// GetLatencyStats returns latency statistics
	GetLatencyStats(ctx context.Context, targetID uuid.UUID, period time.Duration) (*LatencyStats, error)

	// GetRefreshStatus returns refresh job status
	GetRefreshStatus(ctx context.Context, targetID uuid.UUID) (*RefreshStatus, error)

	// GetUsageStats returns usage statistics
	GetUsageStats(ctx context.Context, targetID uuid.UUID, period time.Duration) (*UsageStats, error)

	// GetWorkloadProfile returns workload profile for optimization
	GetWorkloadProfile(ctx context.Context, targetID uuid.UUID) (*WorkloadProfile, error)
}

// QueryMetrics contains aggregated query statistics
type QueryMetrics struct {
	TargetID       uuid.UUID `json:"target_id"`
	Period         string    `json:"period"`
	TotalQueries   int64     `json:"total_queries"`
	QueriesPerDay  float64   `json:"queries_per_day"`
	UniqueUsers    int       `json:"unique_users"`
	UniquePatterns int       `json:"unique_patterns"`
}

// MissRateMetrics contains pre-agg hit/miss data
type MissRateMetrics struct {
	TargetID         uuid.UUID `json:"target_id"`
	Period           string    `json:"period"`
	TotalQueries     int64     `json:"total_queries"`
	HitCount         int64     `json:"hit_count"`
	MissCount        int64     `json:"miss_count"`
	HitRate          float64   `json:"hit_rate"`
	MissRate         float64   `json:"miss_rate"`
	CommonMissGrains []string  `json:"common_miss_grains"` // Grains causing misses
}

// LatencyStats contains latency percentiles
type LatencyStats struct {
	TargetID     uuid.UUID `json:"target_id"`
	Period       string    `json:"period"`
	AvgLatencyMs float64   `json:"avg_latency_ms"`
	P50LatencyMs float64   `json:"p50_latency_ms"`
	P95LatencyMs float64   `json:"p95_latency_ms"`
	P99LatencyMs float64   `json:"p99_latency_ms"`
	MaxLatencyMs float64   `json:"max_latency_ms"`
}

// RefreshStatus contains refresh job information
type RefreshStatus struct {
	TargetID             uuid.UUID  `json:"target_id"`
	LastRefreshAt        *time.Time `json:"last_refresh_at"`
	LastSuccessAt        *time.Time `json:"last_success_at"`
	LastFailureAt        *time.Time `json:"last_failure_at"`
	LastError            string     `json:"last_error,omitempty"`
	ConsecutiveFailures  int        `json:"consecutive_failures"`
	AvgRefreshDurationMs float64    `json:"avg_refresh_duration_ms"`
	RefreshSchedule      string     `json:"refresh_schedule"` // cron expression
}

// UsageStats contains usage patterns
type UsageStats struct {
	TargetID         uuid.UUID `json:"target_id"`
	Period           string    `json:"period"`
	TotalQueries     int64     `json:"total_queries"`
	TrendPercent     float64   `json:"trend_percent"`    // Growth or decline
	PeakHour         int       `json:"peak_hour"`        // 0-23
	PeakDayOfWeek    int       `json:"peak_day_of_week"` // 0=Sunday
	IsUnderUtilized  bool      `json:"is_under_utilized"`
	DaysSinceLastUse int       `json:"days_since_last_use"`
}

// telemetryService implements TelemetryService using query_telemetry table
type telemetryService struct {
	db *sqlx.DB
}

// NewTelemetryService creates a new telemetry service
func NewTelemetryService(db *sqlx.DB) TelemetryService {
	return &telemetryService{db: db}
}

// GetQueryMetrics returns query metrics for a target
func (s *telemetryService) GetQueryMetrics(ctx context.Context, targetID uuid.UUID, period time.Duration) (*QueryMetrics, error) {
	metrics := &QueryMetrics{
		TargetID: targetID,
		Period:   period.String(),
	}

	since := time.Now().Add(-period)

	err := s.db.GetContext(ctx, metrics, `
		SELECT 
			$1::uuid as target_id,
			$3 as period,
			COUNT(*) as total_queries,
			COUNT(*) * 1.0 / GREATEST(EXTRACT(EPOCH FROM (now() - $2::timestamptz)) / 86400, 1) as queries_per_day,
			COUNT(DISTINCT user_id) as unique_users,
			COUNT(DISTINCT query_hash) as unique_patterns
		FROM query_telemetry
		WHERE target_id = $1
		AND created_at >= $2
	`, targetID, since, period.String())

	if err != nil {
		return nil, fmt.Errorf("failed to get query metrics: %w", err)
	}

	return metrics, nil
}

// GetMissRate returns pre-agg miss rate for a target
func (s *telemetryService) GetMissRate(ctx context.Context, targetID uuid.UUID, period time.Duration) (*MissRateMetrics, error) {
	metrics := &MissRateMetrics{
		TargetID: targetID,
		Period:   period.String(),
	}

	since := time.Now().Add(-period)

	err := s.db.GetContext(ctx, metrics, `
		SELECT 
			COUNT(*) as total_queries,
			COUNT(*) FILTER (WHERE preagg_hit = true) as hit_count,
			COUNT(*) FILTER (WHERE preagg_hit = false OR preagg_hit IS NULL) as miss_count,
			COALESCE(AVG(CASE WHEN preagg_hit THEN 1.0 ELSE 0.0 END), 0) as hit_rate,
			COALESCE(AVG(CASE WHEN preagg_hit THEN 0.0 ELSE 1.0 END), 0) as miss_rate
		FROM query_telemetry
		WHERE target_id = $1
		AND created_at >= $2
	`, targetID, since)

	if err != nil {
		return nil, fmt.Errorf("failed to get miss rate: %w", err)
	}

	// Get common miss patterns
	var grains []string
	s.db.SelectContext(ctx, &grains, `
		SELECT DISTINCT grain_columns
		FROM query_telemetry
		WHERE target_id = $1
		AND created_at >= $2
		AND (preagg_hit = false OR preagg_hit IS NULL)
		GROUP BY grain_columns
		ORDER BY COUNT(*) DESC
		LIMIT 5
	`, targetID, since)
	metrics.CommonMissGrains = grains

	return metrics, nil
}

// GetLatencyStats returns latency statistics
func (s *telemetryService) GetLatencyStats(ctx context.Context, targetID uuid.UUID, period time.Duration) (*LatencyStats, error) {
	stats := &LatencyStats{
		TargetID: targetID,
		Period:   period.String(),
	}

	since := time.Now().Add(-period)

	err := s.db.GetContext(ctx, stats, `
		SELECT 
			COALESCE(AVG(latency_ms), 0) as avg_latency_ms,
			COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY latency_ms), 0) as p50_latency_ms,
			COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY latency_ms), 0) as p95_latency_ms,
			COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY latency_ms), 0) as p99_latency_ms,
			COALESCE(MAX(latency_ms), 0) as max_latency_ms
		FROM query_telemetry
		WHERE target_id = $1
		AND created_at >= $2
	`, targetID, since)

	if err != nil {
		return nil, fmt.Errorf("failed to get latency stats: %w", err)
	}

	return stats, nil
}

// GetRefreshStatus returns refresh job status
func (s *telemetryService) GetRefreshStatus(ctx context.Context, targetID uuid.UUID) (*RefreshStatus, error) {
	status := &RefreshStatus{
		TargetID: targetID,
	}

	err := s.db.GetContext(ctx, status, `
		SELECT 
			MAX(started_at) as last_refresh_at,
			MAX(started_at) FILTER (WHERE status = 'success') as last_success_at,
			MAX(started_at) FILTER (WHERE status = 'failed') as last_failure_at,
			(SELECT error_message FROM preagg_refresh_log 
			 WHERE target_id = $1 AND status = 'failed' 
			 ORDER BY started_at DESC LIMIT 1) as last_error,
			(SELECT COUNT(*) FROM (
				SELECT status FROM preagg_refresh_log 
				WHERE target_id = $1 
				ORDER BY started_at DESC 
				LIMIT 10
			) recent WHERE status = 'failed') as consecutive_failures,
			COALESCE(AVG(duration_ms) FILTER (WHERE status = 'success'), 0) as avg_refresh_duration_ms
		FROM preagg_refresh_log
		WHERE target_id = $1
	`, targetID)

	if err != nil {
		// If table doesn't exist or no data, return empty status
		return &RefreshStatus{TargetID: targetID}, nil
	}

	return status, nil
}

// GetUsageStats returns usage statistics
func (s *telemetryService) GetUsageStats(ctx context.Context, targetID uuid.UUID, period time.Duration) (*UsageStats, error) {
	stats := &UsageStats{
		TargetID: targetID,
		Period:   period.String(),
	}

	since := time.Now().Add(-period)
	halfPeriod := time.Now().Add(-period / 2)

	// Get total and trend
	err := s.db.GetContext(ctx, stats, `
		WITH recent AS (
			SELECT COUNT(*) as cnt
			FROM query_telemetry
			WHERE target_id = $1 AND created_at >= $3
		),
		older AS (
			SELECT COUNT(*) as cnt
			FROM query_telemetry
			WHERE target_id = $1 AND created_at >= $2 AND created_at < $3
		)
		SELECT 
			(SELECT cnt FROM recent) as total_queries,
			CASE WHEN (SELECT cnt FROM older) > 0 
				THEN ((SELECT cnt FROM recent) - (SELECT cnt FROM older)) * 100.0 / (SELECT cnt FROM older)
				ELSE 0 
			END as trend_percent
	`, targetID, since, halfPeriod)

	if err != nil {
		return nil, fmt.Errorf("failed to get usage stats: %w", err)
	}

	// Get peak hour and day
	s.db.GetContext(ctx, stats, `
		SELECT 
			EXTRACT(HOUR FROM created_at)::int as peak_hour,
			EXTRACT(DOW FROM created_at)::int as peak_day_of_week
		FROM query_telemetry
		WHERE target_id = $1 AND created_at >= $2
		GROUP BY EXTRACT(HOUR FROM created_at), EXTRACT(DOW FROM created_at)
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`, targetID, since)

	// Check last use
	var lastUsed time.Time
	s.db.GetContext(ctx, &lastUsed, `
		SELECT COALESCE(MAX(created_at), now() - interval '365 days')
		FROM query_telemetry
		WHERE target_id = $1
	`, targetID)
	stats.DaysSinceLastUse = int(time.Since(lastUsed).Hours() / 24)

	// Determine if under-utilized (less than 10 queries/day and declining)
	stats.IsUnderUtilized = stats.TotalQueries < int64(period.Hours()/24*10) && stats.TrendPercent < 0

	return stats, nil
}

// GetWorkloadProfile returns workload profile for optimization
func (s *telemetryService) GetWorkloadProfile(ctx context.Context, targetID uuid.UUID) (*WorkloadProfile, error) {
	profile := &WorkloadProfile{
		BOID:       targetID,
		WindowDays: 7,
	}

	// Get recent stats
	queryMetrics, _ := s.GetQueryMetrics(ctx, targetID, 7*24*time.Hour)
	latencyStats, _ := s.GetLatencyStats(ctx, targetID, 7*24*time.Hour)
	missRate, _ := s.GetMissRate(ctx, targetID, 7*24*time.Hour)

	if queryMetrics != nil {
		profile.QueriesPerDay = queryMetrics.QueriesPerDay
		profile.TotalQueries = queryMetrics.TotalQueries
	}
	if latencyStats != nil {
		profile.AvgDurationMs = latencyStats.AvgLatencyMs
		profile.P95DurationMs = latencyStats.P95LatencyMs
	}
	if missRate != nil {
		profile.PreAggHitRate = missRate.HitRate
		profile.PreAggMissRate = missRate.MissRate
		profile.PreAggMissQueries = missRate.MissCount
	}

	// Get hot grains
	s.db.SelectContext(ctx, &profile.HotGrains, `
		SELECT grain_columns
		FROM query_telemetry
		WHERE target_id = $1
		AND created_at >= now() - interval '7 days'
		GROUP BY grain_columns
		ORDER BY COUNT(*) DESC
		LIMIT 5
	`, targetID)

	// Get hot measures
	s.db.SelectContext(ctx, &profile.HotMeasures, `
		SELECT measure_names
		FROM query_telemetry
		WHERE target_id = $1
		AND created_at >= now() - interval '7 days'
		GROUP BY measure_names
		ORDER BY COUNT(*) DESC
		LIMIT 10
	`, targetID)

	// Get peak hours
	var peakHours []int
	s.db.SelectContext(ctx, &peakHours, `
		SELECT EXTRACT(HOUR FROM created_at)::int as hour
		FROM query_telemetry
		WHERE target_id = $1
		AND created_at >= now() - interval '7 days'
		GROUP BY EXTRACT(HOUR FROM created_at)
		HAVING COUNT(*) > (SELECT COUNT(*) * 0.1 FROM query_telemetry WHERE target_id = $1 AND created_at >= now() - interval '7 days')
		ORDER BY COUNT(*) DESC
		LIMIT 5
	`, targetID)
	profile.PeakHours = peakHours

	return profile, nil
}
