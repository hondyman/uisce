package analytics

import (
	"context"
	"database/sql"
	"fmt"
)

// CoverageDashboardService provides pre-aggregation coverage metrics.
type CoverageDashboardService struct {
	db *sql.DB // Trino connection for querying query_telemetry
}

// NewCoverageDashboardService creates a new coverage dashboard service.
func NewCoverageDashboardService(db *sql.DB) *CoverageDashboardService {
	return &CoverageDashboardService{db: db}
}

// CoverageMetric represents pre-agg coverage for a datasource.
type CoverageMetric struct {
	TenantID        string  `json:"tenant_id"`
	Datasource      string  `json:"datasource"`
	TotalQueries    int64   `json:"total_queries"`
	PreAggHits      int64   `json:"preagg_hits"`
	CoverageRatio   float64 `json:"coverage_ratio"`
	AvgLatencySaved float64 `json:"avg_latency_saved_ms"`
}

// GetCoverageByDatasource returns pre-agg coverage metrics per datasource.
// This query analyzes the last N days of telemetry to compute hit ratios.
func (s *CoverageDashboardService) GetCoverageByDatasource(ctx context.Context, tenantID string, daysBack int) ([]CoverageMetric, error) {
	query := `
		SELECT
			tenant_id,
			bo_name AS datasource,
			COUNT(*) AS total_queries,
			SUM(CASE WHEN preagg_hit THEN 1 ELSE 0 END) AS preagg_hits,
			CAST(SUM(CASE WHEN preagg_hit THEN 1 ELSE 0 END) AS DOUBLE) / NULLIF(COUNT(*), 0) AS coverage_ratio,
			AVG(CASE WHEN preagg_hit THEN duration_ms ELSE NULL END) AS avg_latency_saved_ms
		FROM query_telemetry
		WHERE tenant_id = ?
			AND created_at > NOW() - INTERVAL ? DAY
		GROUP BY tenant_id, bo_name
		ORDER BY coverage_ratio DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, daysBack)
	if err != nil {
		return nil, fmt.Errorf("coverage query failed: %w", err)
	}
	defer rows.Close()

	var metrics []CoverageMetric
	for rows.Next() {
		var m CoverageMetric
		if err := rows.Scan(&m.TenantID, &m.Datasource, &m.TotalQueries, &m.PreAggHits, &m.CoverageRatio, &m.AvgLatencySaved); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, rows.Err()
}

// GetGlobalCoverage returns coverage across all tenants (for Ops dashboard).
func (s *CoverageDashboardService) GetGlobalCoverage(ctx context.Context, daysBack int) ([]CoverageMetric, error) {
	query := `
		SELECT
			tenant_id,
			bo_name AS datasource,
			COUNT(*) AS total_queries,
			SUM(CASE WHEN preagg_hit THEN 1 ELSE 0 END) AS preagg_hits,
			CAST(SUM(CASE WHEN preagg_hit THEN 1 ELSE 0 END) AS DOUBLE) / NULLIF(COUNT(*), 0) AS coverage_ratio,
			AVG(CASE WHEN preagg_hit THEN duration_ms ELSE NULL END) AS avg_latency_saved_ms
		FROM query_telemetry
		WHERE created_at > NOW() - INTERVAL ? DAY
		GROUP BY tenant_id, bo_name
		ORDER BY tenant_id, coverage_ratio DESC
	`

	rows, err := s.db.QueryContext(ctx, query, daysBack)
	if err != nil {
		return nil, fmt.Errorf("global coverage query failed: %w", err)
	}
	defer rows.Close()

	var metrics []CoverageMetric
	for rows.Next() {
		var m CoverageMetric
		if err := rows.Scan(&m.TenantID, &m.Datasource, &m.TotalQueries, &m.PreAggHits, &m.CoverageRatio, &m.AvgLatencySaved); err != nil {
			return nil, err
		}
		metrics = append(metrics, m)
	}

	return metrics, rows.Err()
}

// GetCoverageTrend returns daily coverage trend for a datasource.
func (s *CoverageDashboardService) GetCoverageTrend(ctx context.Context, tenantID, datasource string, daysBack int) ([]DailyCoverage, error) {
	query := `
		SELECT
			DATE(created_at) AS date,
			COUNT(*) AS total_queries,
			SUM(CASE WHEN preagg_hit THEN 1 ELSE 0 END) AS preagg_hits,
			CAST(SUM(CASE WHEN preagg_hit THEN 1 ELSE 0 END) AS DOUBLE) / NULLIF(COUNT(*), 0) AS coverage_ratio
		FROM query_telemetry
		WHERE tenant_id = ?
			AND bo_name = ?
			AND created_at > NOW() - INTERVAL ? DAY
		GROUP BY DATE(created_at)
		ORDER BY date DESC
	`

	rows, err := s.db.QueryContext(ctx, query, tenantID, datasource, daysBack)
	if err != nil {
		return nil, fmt.Errorf("coverage trend query failed: %w", err)
	}
	defer rows.Close()

	var trend []DailyCoverage
	for rows.Next() {
		var d DailyCoverage
		if err := rows.Scan(&d.Date, &d.TotalQueries, &d.PreAggHits, &d.CoverageRatio); err != nil {
			return nil, err
		}
		trend = append(trend, d)
	}

	return trend, rows.Err()
}

// DailyCoverage represents coverage for a single day.
type DailyCoverage struct {
	Date          string  `json:"date"`
	TotalQueries  int64   `json:"total_queries"`
	PreAggHits    int64   `json:"preagg_hits"`
	CoverageRatio float64 `json:"coverage_ratio"`
}
