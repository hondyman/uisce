package cbo

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// WorkloadTracker tracks query patterns for adaptive optimization
type WorkloadTracker struct {
	db *sqlx.DB
}

// NewWorkloadTracker creates a new workload tracker
func NewWorkloadTracker(db *sqlx.DB) *WorkloadTracker {
	return &WorkloadTracker{db: db}
}

// RecordQuery records a query execution for workload analysis
func (t *WorkloadTracker) RecordQuery(ctx context.Context, record QueryRecord) error {
	query := `
		INSERT INTO cbo_query_log (
			tenant_id, query_hash, query_pattern, execution_path,
			estimated_cost, actual_duration_ms, cache_hit, preagg_used
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err := t.db.ExecContext(ctx, query,
		record.TenantID,
		record.QueryHash,
		record.QueryPattern,
		string(record.ExecutionPath),
		record.EstimatedCost,
		record.ActualDuration,
		record.CacheHit,
		record.PreAggUsed,
	)

	return err
}

// GetTopPatterns returns the most frequently occurring query patterns
func (t *WorkloadTracker) GetTopPatterns(ctx context.Context, tenantID uuid.UUID, limit int) ([]QueryPattern, error) {
	query := `
		SELECT 
			query_pattern,
			COUNT(*) as frequency,
			AVG(actual_duration_ms) as avg_duration,
			AVG(estimated_cost) as avg_cost,
			MAX(created_at) as last_seen
		FROM cbo_query_log
		WHERE tenant_id = $1
		  AND created_at > NOW() - INTERVAL '7 days'
		  AND query_pattern != ''
		GROUP BY query_pattern
		ORDER BY frequency DESC
		LIMIT $2
	`

	var patterns []QueryPattern
	rows, err := t.db.QueryxContext(ctx, query, tenantID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p QueryPattern
		if err := rows.Scan(&p.Pattern, &p.Frequency, &p.AvgDuration, &p.AvgCost, &p.LastSeen); err != nil {
			continue
		}
		// Determine if pattern is optimizable
		p.Optimizable = p.Frequency >= 10 && p.AvgDuration > 1000 // 10+ queries, >1s avg
		patterns = append(patterns, p)
	}

	return patterns, nil
}

// GetSlowQueries returns queries that exceed a duration threshold
func (t *WorkloadTracker) GetSlowQueries(ctx context.Context, tenantID uuid.UUID, thresholdMs int, limit int) ([]QueryRecord, error) {
	query := `
		SELECT id, tenant_id, query_hash, query_pattern, execution_path,
		       estimated_cost, actual_duration_ms, cache_hit, preagg_used, created_at
		FROM cbo_query_log
		WHERE tenant_id = $1
		  AND actual_duration_ms > $2
		  AND created_at > NOW() - INTERVAL '24 hours'
		ORDER BY actual_duration_ms DESC
		LIMIT $3
	`

	var records []QueryRecord
	err := t.db.SelectContext(ctx, &records, query, tenantID, thresholdMs, limit)
	return records, err
}

// SuggestOptimizations analyzes workload and suggests optimizations
func (t *WorkloadTracker) SuggestOptimizations(ctx context.Context, tenantID uuid.UUID) ([]Recommendation, error) {
	var recommendations []Recommendation

	// Get top patterns that are slow
	patterns, err := t.GetTopPatterns(ctx, tenantID, 20)
	if err != nil {
		return nil, err
	}

	for _, p := range patterns {
		if !p.Optimizable {
			continue
		}

		// Suggest pre-aggregation for high-frequency, slow patterns
		if p.Frequency >= 50 && p.AvgDuration > 2000 {
			recommendations = append(recommendations, Recommendation{
				Type:        "create_preagg",
				Priority:    "high",
				Description: "Create pre-aggregation for frequently accessed pattern",
				Impact:      "Expected 80-95% query time reduction",
				SQLHint:     p.Pattern,
				CreatedAt:   time.Now(),
			})
		} else if p.Frequency >= 10 && p.AvgDuration > 1000 {
			recommendations = append(recommendations, Recommendation{
				Type:        "create_preagg",
				Priority:    "medium",
				Description: "Consider pre-aggregation for this query pattern",
				Impact:      "Expected 50-80% query time reduction",
				SQLHint:     p.Pattern,
				CreatedAt:   time.Now(),
			})
		}
	}

	// Check for patterns without cache hits
	noCacheQuery := `
		SELECT query_pattern, COUNT(*) as cnt
		FROM cbo_query_log
		WHERE tenant_id = $1
		  AND cache_hit = false
		  AND created_at > NOW() - INTERVAL '24 hours'
		GROUP BY query_pattern
		HAVING COUNT(*) > 20
		ORDER BY cnt DESC
		LIMIT 5
	`

	rows, err := t.db.QueryxContext(ctx, noCacheQuery, tenantID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var pattern string
			var count int
			if err := rows.Scan(&pattern, &count); err == nil {
				recommendations = append(recommendations, Recommendation{
					Type:        "enable_caching",
					Priority:    "medium",
					Description: "Enable result caching for repeated queries",
					Impact:      "Eliminate redundant computation for repeated queries",
					SQLHint:     pattern,
					CreatedAt:   time.Now(),
				})
			}
		}
	}

	// Check for queries with high join complexity
	highJoinQuery := `
		SELECT query_pattern, AVG(actual_duration_ms) as avg_dur
		FROM cbo_query_log
		WHERE tenant_id = $1
		  AND estimated_cost > 100000
		  AND created_at > NOW() - INTERVAL '7 days'
		GROUP BY query_pattern
		HAVING AVG(actual_duration_ms) > 5000
		LIMIT 5
	`

	rows2, err := t.db.QueryxContext(ctx, highJoinQuery, tenantID)
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var pattern string
			var avgDur float64
			if err := rows2.Scan(&pattern, &avgDur); err == nil {
				recommendations = append(recommendations, Recommendation{
					Type:        "add_index",
					Priority:    "high",
					Description: "Add indexes to reduce join cost for complex queries",
					Impact:      "Expected 40-60% query time reduction",
					SQLHint:     pattern,
					CreatedAt:   time.Now(),
				})
			}
		}
	}

	return recommendations, nil
}

// GetWorkloadSummary returns a summary of recent workload
func (t *WorkloadTracker) GetWorkloadSummary(ctx context.Context, tenantID uuid.UUID, hours int) (*WorkloadSummary, error) {
	query := `
		SELECT 
			COUNT(*) as total_queries,
			SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END) as cache_hits,
			SUM(CASE WHEN preagg_used IS NOT NULL THEN 1 ELSE 0 END) as preagg_hits,
			AVG(actual_duration_ms) as avg_duration,
			MAX(actual_duration_ms) as max_duration,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY actual_duration_ms) as p95_duration,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY actual_duration_ms) as p99_duration
		FROM cbo_query_log
		WHERE tenant_id = $1
		  AND created_at > NOW() - INTERVAL '%d hours'
	`

	var summary WorkloadSummary
	err := t.db.QueryRowxContext(ctx, query, tenantID, hours).Scan(
		&summary.TotalQueries,
		&summary.CacheHits,
		&summary.PreAggHits,
		&summary.AvgDuration,
		&summary.MaxDuration,
		&summary.P95Duration,
		&summary.P99Duration,
	)
	if err != nil {
		return &WorkloadSummary{}, nil
	}

	// Calculate rates
	if summary.TotalQueries > 0 {
		summary.CacheHitRate = float64(summary.CacheHits) / float64(summary.TotalQueries)
		summary.PreAggHitRate = float64(summary.PreAggHits) / float64(summary.TotalQueries)
	}

	return &summary, nil
}

// WorkloadSummary contains aggregated workload statistics
type WorkloadSummary struct {
	TotalQueries  int64   `json:"total_queries"`
	CacheHits     int64   `json:"cache_hits"`
	CacheHitRate  float64 `json:"cache_hit_rate"`
	PreAggHits    int64   `json:"preagg_hits"`
	PreAggHitRate float64 `json:"preagg_hit_rate"`
	AvgDuration   float64 `json:"avg_duration_ms"`
	MaxDuration   float64 `json:"max_duration_ms"`
	P95Duration   float64 `json:"p95_duration_ms"`
	P99Duration   float64 `json:"p99_duration_ms"`
}

// CleanupOldRecords removes old query log entries
func (t *WorkloadTracker) CleanupOldRecords(ctx context.Context, retentionDays int) (int64, error) {
	query := `
		DELETE FROM cbo_query_log
		WHERE created_at < NOW() - INTERVAL '%d days'
	`

	result, err := t.db.ExecContext(ctx, query, retentionDays)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}
