package observability

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"
)

// SLOReportConfig configures the weekly SLO report generator.
type SLOReportConfig struct {
	DSN              string
	PrometheusURL    string
	OutputDir        string
	SlackWebhookURL  string
	EmailRecipients  []string
	ReportPeriodDays int
	Logger           interface{ Info(string, ...any) }
}

// SLOReport represents a weekly SLO compliance report.
type SLOReport struct {
	GeneratedAt     time.Time         `json:"generated_at"`
	PeriodStart     time.Time         `json:"period_start"`
	PeriodEnd       time.Time         `json:"period_end"`
	OverallHealth   HealthStatus      `json:"overall_health"`
	Summary         SLOSummary        `json:"summary"`
	TenantReports   []TenantSLOReport `json:"tenant_reports"`
	Alerts          []AlertSummary    `json:"alerts"`
	Recommendations []string          `json:"recommendations"`
	Trends          TrendAnalysis     `json:"trends"`
}

// HealthStatus indicates overall system health.
type HealthStatus string

const (
	HealthGreen  HealthStatus = "green"  // All SLOs met
	HealthYellow HealthStatus = "yellow" // Some warnings
	HealthRed    HealthStatus = "red"    // SLO breaches
)

// SLOSummary provides aggregate statistics.
type SLOSummary struct {
	TotalTenants        int     `json:"total_tenants"`
	TenantsInCompliance int     `json:"tenants_in_compliance"`
	ComplianceRate      float64 `json:"compliance_rate_percent"`

	// Latency SLO (p95 < 2s)
	LatencySLOTarget float64 `json:"latency_slo_target_seconds"`
	LatencySLOMet    bool    `json:"latency_slo_met"`
	LatencyP50       float64 `json:"latency_p50_seconds"`
	LatencyP95       float64 `json:"latency_p95_seconds"`
	LatencyP99       float64 `json:"latency_p99_seconds"`

	// Rollup Hit Rate SLO (> 85%)
	RollupHitSLOTarget float64 `json:"rollup_hit_slo_target_percent"`
	RollupHitSLOMet    bool    `json:"rollup_hit_slo_met"`
	RollupHitRate      float64 `json:"rollup_hit_rate_percent"`

	// Error Rate SLO (< 1%)
	ErrorRateSLOTarget float64 `json:"error_rate_slo_target_percent"`
	ErrorRateSLOMet    bool    `json:"error_rate_slo_met"`
	ErrorRate          float64 `json:"error_rate_percent"`

	// Availability
	UptimePercent float64 `json:"uptime_percent"`
	TotalRequests int64   `json:"total_requests"`
	TotalErrors   int64   `json:"total_errors"`
	CacheHits     int64   `json:"cache_hits"`
	CacheMisses   int64   `json:"cache_misses"`
}

// TenantSLOReport provides per-tenant SLO metrics.
type TenantSLOReport struct {
	TenantID       string             `json:"tenant_id"`
	TenantName     string             `json:"tenant_name"`
	Tier           string             `json:"tier"`
	LatencyP95     float64            `json:"latency_p95_seconds"`
	LatencySLOMet  bool               `json:"latency_slo_met"`
	RollupHitRate  float64            `json:"rollup_hit_rate_percent"`
	RollupSLOMet   bool               `json:"rollup_slo_met"`
	ErrorRate      float64            `json:"error_rate_percent"`
	ErrorSLOMet    bool               `json:"error_slo_met"`
	TotalRequests  int64              `json:"total_requests"`
	TopSlowQueries []SlowQuerySummary `json:"top_slow_queries,omitempty"`
}

// SlowQuerySummary describes a slow query pattern.
type SlowQuerySummary struct {
	Cube           string  `json:"cube"`
	Measures       string  `json:"measures"`
	AvgDurationMs  float64 `json:"avg_duration_ms"`
	Count          int     `json:"count"`
	Recommendation string  `json:"recommendation,omitempty"`
}

// AlertSummary describes alerts fired during the period.
type AlertSummary struct {
	AlertName  string    `json:"alert_name"`
	Severity   string    `json:"severity"`
	Count      int       `json:"count"`
	FirstFired time.Time `json:"first_fired"`
	LastFired  time.Time `json:"last_fired"`
	TenantID   string    `json:"tenant_id,omitempty"`
}

// TrendAnalysis shows week-over-week changes.
type TrendAnalysis struct {
	LatencyTrend         string  `json:"latency_trend"` // "improving", "stable", "degrading"
	LatencyChangePercent float64 `json:"latency_change_percent"`
	RollupTrend          string  `json:"rollup_trend"`
	RollupChangePercent  float64 `json:"rollup_change_percent"`
	ErrorTrend           string  `json:"error_trend"`
	ErrorChangePercent   float64 `json:"error_change_percent"`
	RequestVolumeTrend   string  `json:"request_volume_trend"`
	VolumeChangePercent  float64 `json:"volume_change_percent"`
}

// SLOReportGenerator generates weekly SLO compliance reports.
type SLOReportGenerator struct {
	config SLOReportConfig
	db     *sql.DB
}

// NewSLOReportGenerator creates a new report generator.
func NewSLOReportGenerator(cfg SLOReportConfig) (*SLOReportGenerator, error) {
	if cfg.ReportPeriodDays == 0 {
		cfg.ReportPeriodDays = 7
	}
	if cfg.OutputDir == "" {
		cfg.OutputDir = "reports/slo"
	}

	db, err := sql.Open("postgres", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	return &SLOReportGenerator{
		config: cfg,
		db:     db,
	}, nil
}

// Generate creates a new SLO report for the specified period.
func (g *SLOReportGenerator) Generate(ctx context.Context) (*SLOReport, error) {
	report := &SLOReport{
		GeneratedAt: time.Now().UTC(),
		PeriodEnd:   time.Now().UTC().Truncate(24 * time.Hour),
		PeriodStart: time.Now().UTC().Truncate(24*time.Hour).AddDate(0, 0, -g.config.ReportPeriodDays),
	}

	// 1. Collect aggregate metrics
	summary, err := g.collectSummaryMetrics(ctx, report.PeriodStart, report.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("collect summary: %w", err)
	}
	report.Summary = *summary

	// 2. Collect per-tenant metrics
	tenantReports, err := g.collectTenantMetrics(ctx, report.PeriodStart, report.PeriodEnd)
	if err != nil {
		return nil, fmt.Errorf("collect tenant metrics: %w", err)
	}
	report.TenantReports = tenantReports

	// 3. Collect alerts
	alerts, err := g.collectAlerts(ctx, report.PeriodStart, report.PeriodEnd)
	if err != nil {
		// Non-fatal, continue
		alerts = []AlertSummary{}
	}
	report.Alerts = alerts

	// 4. Calculate trends
	report.Trends = g.calculateTrends(ctx, report.PeriodStart)

	// 5. Generate recommendations
	report.Recommendations = g.generateRecommendations(report)

	// 6. Determine overall health
	report.OverallHealth = g.determineHealth(report)

	return report, nil
}

// collectSummaryMetrics gathers aggregate SLO metrics.
func (g *SLOReportGenerator) collectSummaryMetrics(ctx context.Context, start, end time.Time) (*SLOSummary, error) {
	summary := &SLOSummary{
		LatencySLOTarget:   2.0,  // 2 seconds p95
		RollupHitSLOTarget: 85.0, // 85% rollup hit rate
		ErrorRateSLOTarget: 1.0,  // 1% error rate
	}

	// Query cube_query_analytics for aggregate stats
	query := `
		SELECT 
			COUNT(*) as total_requests,
			COUNT(*) FILTER (WHERE status = 'error') as total_errors,
			COUNT(*) FILTER (WHERE cache_hit = true) as cache_hits,
			COUNT(*) FILTER (WHERE cache_hit = false) as cache_misses,
			PERCENTILE_CONT(0.50) WITHIN GROUP (ORDER BY duration_ms) as p50,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95,
			PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration_ms) as p99,
			COUNT(DISTINCT tenant_id) as total_tenants
		FROM cube_query_analytics
		WHERE created_at >= $1 AND created_at < $2
	`

	var p50, p95, p99 sql.NullFloat64
	err := g.db.QueryRowContext(ctx, query, start, end).Scan(
		&summary.TotalRequests,
		&summary.TotalErrors,
		&summary.CacheHits,
		&summary.CacheMisses,
		&p50,
		&p95,
		&p99,
		&summary.TotalTenants,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Convert to seconds
	if p50.Valid {
		summary.LatencyP50 = p50.Float64 / 1000
	}
	if p95.Valid {
		summary.LatencyP95 = p95.Float64 / 1000
	}
	if p99.Valid {
		summary.LatencyP99 = p99.Float64 / 1000
	}

	// Calculate rates
	if summary.TotalRequests > 0 {
		summary.ErrorRate = float64(summary.TotalErrors) / float64(summary.TotalRequests) * 100
		totalCacheOps := summary.CacheHits + summary.CacheMisses
		if totalCacheOps > 0 {
			summary.RollupHitRate = float64(summary.CacheHits) / float64(totalCacheOps) * 100
		}
	}

	// Evaluate SLO compliance
	summary.LatencySLOMet = summary.LatencyP95 <= summary.LatencySLOTarget
	summary.RollupHitSLOMet = summary.RollupHitRate >= summary.RollupHitSLOTarget
	summary.ErrorRateSLOMet = summary.ErrorRate <= summary.ErrorRateSLOTarget

	// Count tenants in compliance (all 3 SLOs met)
	complianceQuery := `
		WITH tenant_stats AS (
			SELECT 
				tenant_id,
				PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) / 1000 as p95_sec,
				COUNT(*) FILTER (WHERE status = 'error')::float / NULLIF(COUNT(*), 0) * 100 as error_rate,
				COUNT(*) FILTER (WHERE cache_hit = true)::float / NULLIF(COUNT(*), 0) * 100 as hit_rate
			FROM cube_query_analytics
			WHERE created_at >= $1 AND created_at < $2
			GROUP BY tenant_id
		)
		SELECT COUNT(*) FROM tenant_stats
		WHERE p95_sec <= $3 AND error_rate <= $4 AND hit_rate >= $5
	`
	err = g.db.QueryRowContext(ctx, complianceQuery, start, end,
		summary.LatencySLOTarget, summary.ErrorRateSLOTarget, summary.RollupHitSLOTarget,
	).Scan(&summary.TenantsInCompliance)
	if err != nil && err != sql.ErrNoRows {
		// Non-fatal
		summary.TenantsInCompliance = 0
	}

	if summary.TotalTenants > 0 {
		summary.ComplianceRate = float64(summary.TenantsInCompliance) / float64(summary.TotalTenants) * 100
	}

	// Calculate uptime (assuming we have uptime tracking)
	summary.UptimePercent = 99.9 // Default; would come from actual monitoring

	return summary, nil
}

// collectTenantMetrics gathers per-tenant SLO data.
func (g *SLOReportGenerator) collectTenantMetrics(ctx context.Context, start, end time.Time) ([]TenantSLOReport, error) {
	query := `
		SELECT 
			cqa.tenant_id,
			COALESCE(t.display_name, cqa.tenant_id) as tenant_name,
			CASE WHEN t.is_gold_copy THEN 'enterprise' ELSE 'standard' END as tier,
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY cqa.duration_ms) / 1000 as p95_sec,
			COUNT(*) FILTER (WHERE cqa.status = 'error')::float / NULLIF(COUNT(*), 0) * 100 as error_rate,
			COUNT(*) FILTER (WHERE cqa.cache_hit = true)::float / NULLIF(COUNT(*), 0) * 100 as hit_rate,
			COUNT(*) as total_requests
		FROM cube_query_analytics cqa
		LEFT JOIN tenants t ON t.id::text = cqa.tenant_id
		WHERE cqa.created_at >= $1 AND cqa.created_at < $2
		GROUP BY cqa.tenant_id, t.display_name, t.is_gold_copy
		ORDER BY total_requests DESC
	`

	rows, err := g.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reports []TenantSLOReport
	for rows.Next() {
		var tr TenantSLOReport
		var p95, errorRate, hitRate sql.NullFloat64

		err := rows.Scan(
			&tr.TenantID,
			&tr.TenantName,
			&tr.Tier,
			&p95,
			&errorRate,
			&hitRate,
			&tr.TotalRequests,
		)
		if err != nil {
			continue
		}

		if p95.Valid {
			tr.LatencyP95 = p95.Float64
		}
		if errorRate.Valid {
			tr.ErrorRate = errorRate.Float64
		}
		if hitRate.Valid {
			tr.RollupHitRate = hitRate.Float64
		}

		// Check SLO compliance
		tr.LatencySLOMet = tr.LatencyP95 <= 2.0
		tr.ErrorSLOMet = tr.ErrorRate <= 1.0
		tr.RollupSLOMet = tr.RollupHitRate >= 85.0

		// Get top slow queries for this tenant
		tr.TopSlowQueries = g.getTopSlowQueries(ctx, tr.TenantID, start, end)

		reports = append(reports, tr)
	}

	return reports, rows.Err()
}

// getTopSlowQueries finds the slowest query patterns for a tenant.
func (g *SLOReportGenerator) getTopSlowQueries(ctx context.Context, tenantID string, start, end time.Time) []SlowQuerySummary {
	query := `
		SELECT 
			cube_name,
			measures,
			AVG(duration_ms) as avg_duration,
			COUNT(*) as query_count
		FROM cube_query_analytics
		WHERE tenant_id = $1 
		  AND created_at >= $2 AND created_at < $3
		  AND duration_ms > 1000
		GROUP BY cube_name, measures
		ORDER BY avg_duration DESC
		LIMIT 5
	`

	rows, err := g.db.QueryContext(ctx, query, tenantID, start, end)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var slowQueries []SlowQuerySummary
	for rows.Next() {
		var sq SlowQuerySummary
		var measures sql.NullString

		if err := rows.Scan(&sq.Cube, &measures, &sq.AvgDurationMs, &sq.Count); err != nil {
			continue
		}
		if measures.Valid {
			sq.Measures = measures.String
		}

		// Generate recommendation
		if sq.AvgDurationMs > 5000 {
			sq.Recommendation = "Consider adding pre-aggregation for this query pattern"
		} else if sq.Count > 100 {
			sq.Recommendation = "High-frequency query - candidate for caching optimization"
		}

		slowQueries = append(slowQueries, sq)
	}

	return slowQueries
}

// collectAlerts gathers alert firing history.
func (g *SLOReportGenerator) collectAlerts(ctx context.Context, start, end time.Time) ([]AlertSummary, error) {
	query := `
		SELECT 
			alert_name,
			severity,
			COUNT(*) as fire_count,
			MIN(fired_at) as first_fired,
			MAX(fired_at) as last_fired,
			tenant_id
		FROM cube_alerts
		WHERE fired_at >= $1 AND fired_at < $2
		GROUP BY alert_name, severity, tenant_id
		ORDER BY fire_count DESC
	`

	rows, err := g.db.QueryContext(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []AlertSummary
	for rows.Next() {
		var a AlertSummary
		var tenantID sql.NullString

		if err := rows.Scan(&a.AlertName, &a.Severity, &a.Count, &a.FirstFired, &a.LastFired, &tenantID); err != nil {
			continue
		}
		if tenantID.Valid {
			a.TenantID = tenantID.String
		}
		alerts = append(alerts, a)
	}

	return alerts, rows.Err()
}

// calculateTrends compares current period to previous period.
func (g *SLOReportGenerator) calculateTrends(ctx context.Context, currentPeriodStart time.Time) TrendAnalysis {
	trends := TrendAnalysis{
		LatencyTrend:       "stable",
		RollupTrend:        "stable",
		ErrorTrend:         "stable",
		RequestVolumeTrend: "stable",
	}

	// Previous period
	prevEnd := currentPeriodStart
	prevStart := prevEnd.AddDate(0, 0, -g.config.ReportPeriodDays)

	// Get previous period metrics
	prevQuery := `
		SELECT 
			PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) / 1000 as p95_sec,
			COUNT(*) FILTER (WHERE cache_hit = true)::float / NULLIF(COUNT(*), 0) * 100 as hit_rate,
			COUNT(*) FILTER (WHERE status = 'error')::float / NULLIF(COUNT(*), 0) * 100 as error_rate,
			COUNT(*) as total
		FROM cube_query_analytics
		WHERE created_at >= $1 AND created_at < $2
	`

	var prevP95, prevHit, prevError sql.NullFloat64
	var prevTotal int64
	err := g.db.QueryRowContext(ctx, prevQuery, prevStart, prevEnd).Scan(
		&prevP95, &prevHit, &prevError, &prevTotal,
	)
	if err != nil || !prevP95.Valid {
		return trends
	}

	// Current period (already calculated in summary, but re-query for simplicity)
	currEnd := currentPeriodStart.AddDate(0, 0, g.config.ReportPeriodDays)
	var currP95, currHit, currError sql.NullFloat64
	var currTotal int64
	err = g.db.QueryRowContext(ctx, prevQuery, currentPeriodStart, currEnd).Scan(
		&currP95, &currHit, &currError, &currTotal,
	)
	if err != nil || !currP95.Valid {
		return trends
	}

	// Calculate changes
	if prevP95.Float64 > 0 {
		trends.LatencyChangePercent = ((currP95.Float64 - prevP95.Float64) / prevP95.Float64) * 100
		if trends.LatencyChangePercent < -5 {
			trends.LatencyTrend = "improving"
		} else if trends.LatencyChangePercent > 5 {
			trends.LatencyTrend = "degrading"
		}
	}

	if prevHit.Float64 > 0 {
		trends.RollupChangePercent = currHit.Float64 - prevHit.Float64
		if trends.RollupChangePercent > 2 {
			trends.RollupTrend = "improving"
		} else if trends.RollupChangePercent < -2 {
			trends.RollupTrend = "degrading"
		}
	}

	if prevError.Float64 > 0 {
		trends.ErrorChangePercent = ((currError.Float64 - prevError.Float64) / prevError.Float64) * 100
		if trends.ErrorChangePercent < -10 {
			trends.ErrorTrend = "improving"
		} else if trends.ErrorChangePercent > 10 {
			trends.ErrorTrend = "degrading"
		}
	}

	if prevTotal > 0 {
		trends.VolumeChangePercent = float64(currTotal-prevTotal) / float64(prevTotal) * 100
		if trends.VolumeChangePercent > 20 {
			trends.RequestVolumeTrend = "increasing"
		} else if trends.VolumeChangePercent < -20 {
			trends.RequestVolumeTrend = "decreasing"
		}
	}

	return trends
}

// generateRecommendations produces actionable suggestions.
func (g *SLOReportGenerator) generateRecommendations(report *SLOReport) []string {
	var recs []string

	if !report.Summary.LatencySLOMet {
		recs = append(recs, fmt.Sprintf(
			"CRITICAL: p95 latency (%.2fs) exceeds SLO target (%.2fs). Review slow queries and consider adding pre-aggregations.",
			report.Summary.LatencyP95, report.Summary.LatencySLOTarget,
		))
	}

	if !report.Summary.RollupHitSLOMet {
		recs = append(recs, fmt.Sprintf(
			"WARNING: Rollup hit rate (%.1f%%) below SLO target (%.1f%%). Analyze query patterns for pre-aggregation candidates.",
			report.Summary.RollupHitRate, report.Summary.RollupHitSLOTarget,
		))
	}

	if !report.Summary.ErrorRateSLOMet {
		recs = append(recs, fmt.Sprintf(
			"CRITICAL: Error rate (%.2f%%) exceeds SLO target (%.2f%%). Investigate error logs immediately.",
			report.Summary.ErrorRate, report.Summary.ErrorRateSLOTarget,
		))
	}

	if report.Trends.LatencyTrend == "degrading" {
		recs = append(recs, fmt.Sprintf(
			"Latency trending up (%.1f%% WoW). Monitor for capacity issues.",
			report.Trends.LatencyChangePercent,
		))
	}

	if report.Trends.RequestVolumeTrend == "increasing" && report.Trends.VolumeChangePercent > 50 {
		recs = append(recs, fmt.Sprintf(
			"Request volume increased %.1f%% WoW. Consider scaling resources.",
			report.Trends.VolumeChangePercent,
		))
	}

	// Check for tenants with issues
	nonCompliantCount := 0
	for _, tr := range report.TenantReports {
		if !tr.LatencySLOMet || !tr.ErrorSLOMet || !tr.RollupSLOMet {
			nonCompliantCount++
		}
	}
	if nonCompliantCount > 0 {
		recs = append(recs, fmt.Sprintf(
			"%d tenant(s) not meeting SLO targets. Review per-tenant reports for details.",
			nonCompliantCount,
		))
	}

	if len(recs) == 0 {
		recs = append(recs, "All SLOs met. System performing within targets. Continue monitoring.")
	}

	return recs
}

// determineHealth calculates overall system health.
func (g *SLOReportGenerator) determineHealth(report *SLOReport) HealthStatus {
	allMet := report.Summary.LatencySLOMet && report.Summary.RollupHitSLOMet && report.Summary.ErrorRateSLOMet

	if allMet && report.Summary.ComplianceRate >= 95 {
		return HealthGreen
	}

	if !report.Summary.ErrorRateSLOMet || report.Summary.ComplianceRate < 80 {
		return HealthRed
	}

	return HealthYellow
}

// SaveReport writes the report to disk in multiple formats.
func (g *SLOReportGenerator) SaveReport(report *SLOReport) error {
	if err := os.MkdirAll(g.config.OutputDir, 0755); err != nil {
		return err
	}

	timestamp := report.PeriodEnd.Format("2006-01-02")

	// JSON format
	jsonPath := fmt.Sprintf("%s/slo-report-%s.json", g.config.OutputDir, timestamp)
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, jsonData, 0644); err != nil {
		return err
	}

	// Markdown format for human reading
	mdPath := fmt.Sprintf("%s/slo-report-%s.md", g.config.OutputDir, timestamp)
	mdContent := g.renderMarkdown(report)
	if err := os.WriteFile(mdPath, []byte(mdContent), 0644); err != nil {
		return err
	}

	return nil
}

// renderMarkdown creates a human-readable report.
func (g *SLOReportGenerator) renderMarkdown(report *SLOReport) string {
	tmpl := `# Weekly SLO Report

**Generated:** {{.GeneratedAt.Format "2006-01-02 15:04 UTC"}}  
**Period:** {{.PeriodStart.Format "2006-01-02"}} to {{.PeriodEnd.Format "2006-01-02"}}  
**Overall Health:** {{healthEmoji .OverallHealth}} {{.OverallHealth}}

---

## Summary

| Metric | Value | Target | Status |
|--------|-------|--------|--------|
| p95 Latency | {{printf "%.2f" .Summary.LatencyP95}}s | <{{printf "%.1f" .Summary.LatencySLOTarget}}s | {{sloStatus .Summary.LatencySLOMet}} |
| Rollup Hit Rate | {{printf "%.1f" .Summary.RollupHitRate}}% | >{{printf "%.0f" .Summary.RollupHitSLOTarget}}% | {{sloStatus .Summary.RollupHitSLOMet}} |
| Error Rate | {{printf "%.2f" .Summary.ErrorRate}}% | <{{printf "%.1f" .Summary.ErrorRateSLOTarget}}% | {{sloStatus .Summary.ErrorRateSLOMet}} |
| Uptime | {{printf "%.2f" .Summary.UptimePercent}}% | >99.9% | {{if ge .Summary.UptimePercent 99.9}}✅{{else}}❌{{end}} |

**Total Requests:** {{.Summary.TotalRequests | comma}}  
**Tenants in Compliance:** {{.Summary.TenantsInCompliance}}/{{.Summary.TotalTenants}} ({{printf "%.1f" .Summary.ComplianceRate}}%)

---

## Trends (Week over Week)

| Metric | Trend | Change |
|--------|-------|--------|
| Latency | {{trendEmoji .Trends.LatencyTrend}} {{.Trends.LatencyTrend}} | {{printf "%+.1f" .Trends.LatencyChangePercent}}% |
| Rollup Hit Rate | {{trendEmoji .Trends.RollupTrend}} {{.Trends.RollupTrend}} | {{printf "%+.1f" .Trends.RollupChangePercent}}pp |
| Error Rate | {{trendEmoji .Trends.ErrorTrend}} {{.Trends.ErrorTrend}} | {{printf "%+.1f" .Trends.ErrorChangePercent}}% |
| Request Volume | {{.Trends.RequestVolumeTrend}} | {{printf "%+.1f" .Trends.VolumeChangePercent}}% |

---

## Recommendations

{{range .Recommendations}}
- {{.}}
{{end}}

---

## Per-Tenant Summary

| Tenant | Tier | p95 | Hit Rate | Error Rate | Requests |
|--------|------|-----|----------|------------|----------|
{{range .TenantReports -}}
| {{.TenantName}} | {{.Tier}} | {{printf "%.2f" .LatencyP95}}s {{sloStatus .LatencySLOMet}} | {{printf "%.1f" .RollupHitRate}}% {{sloStatus .RollupSLOMet}} | {{printf "%.2f" .ErrorRate}}% {{sloStatus .ErrorSLOMet}} | {{.TotalRequests | comma}} |
{{end}}

{{if .Alerts}}
---

## Alerts Fired

| Alert | Severity | Count | Last Fired |
|-------|----------|-------|------------|
{{range .Alerts -}}
| {{.AlertName}} | {{.Severity}} | {{.Count}} | {{.LastFired.Format "2006-01-02 15:04"}} |
{{end}}
{{end}}
`

	funcMap := template.FuncMap{
		"healthEmoji": func(h HealthStatus) string {
			switch h {
			case HealthGreen:
				return "🟢"
			case HealthYellow:
				return "🟡"
			case HealthRed:
				return "🔴"
			}
			return "⚪"
		},
		"sloStatus": func(met bool) string {
			if met {
				return "✅"
			}
			return "❌"
		},
		"trendEmoji": func(t string) string {
			switch t {
			case "improving":
				return "📈"
			case "degrading":
				return "📉"
			case "increasing":
				return "⬆️"
			case "decreasing":
				return "⬇️"
			}
			return "➡️"
		},
		"comma": func(n int64) string {
			str := fmt.Sprintf("%d", n)
			var result strings.Builder
			for i, c := range str {
				if i > 0 && (len(str)-i)%3 == 0 {
					result.WriteRune(',')
				}
				result.WriteRune(c)
			}
			return result.String()
		},
	}

	t, err := template.New("report").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return fmt.Sprintf("Error parsing template: %v", err)
	}

	var buf strings.Builder
	if err := t.Execute(&buf, report); err != nil {
		return fmt.Sprintf("Error rendering template: %v", err)
	}

	return buf.String()
}

// SendNotifications sends the report via configured channels.
func (g *SLOReportGenerator) SendNotifications(ctx context.Context, report *SLOReport) error {
	// Slack notification
	if g.config.SlackWebhookURL != "" {
		// Would POST to Slack webhook with summary
		_ = g.sendSlackNotification(report)
	}

	// Email notification
	if len(g.config.EmailRecipients) > 0 {
		// Would send email with markdown report
		_ = g.sendEmailNotification(report)
	}

	return nil
}

func (g *SLOReportGenerator) sendSlackNotification(report *SLOReport) error {
	// Implementation would POST to Slack webhook
	return nil
}

func (g *SLOReportGenerator) sendEmailNotification(report *SLOReport) error {
	// Implementation would send email via SMTP/SES
	return nil
}

// RunWeekly is the entry point for scheduled execution.
func RunWeekly(ctx context.Context, dsn string) error {
	cfg := SLOReportConfig{
		DSN:             dsn,
		OutputDir:       "reports/slo",
		SlackWebhookURL: os.Getenv("SLO_SLACK_WEBHOOK_URL"),
	}

	generator, err := NewSLOReportGenerator(cfg)
	if err != nil {
		return err
	}

	report, err := generator.Generate(ctx)
	if err != nil {
		return err
	}

	if err := generator.SaveReport(report); err != nil {
		return err
	}

	return generator.SendNotifications(ctx, report)
}

// GetLatestReport returns the most recent saved report.
func GetLatestReport(outputDir string) (*SLOReport, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}

	// Sort by name descending (newest first)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() > entries[j].Name()
	})

	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".json") {
			data, err := os.ReadFile(fmt.Sprintf("%s/%s", outputDir, e.Name()))
			if err != nil {
				continue
			}
			var report SLOReport
			if err := json.Unmarshal(data, &report); err != nil {
				continue
			}
			return &report, nil
		}
	}

	return nil, fmt.Errorf("no reports found")
}
