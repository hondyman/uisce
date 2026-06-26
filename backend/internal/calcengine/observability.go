package calcengine

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// CALC ENGINE OBSERVABILITY - Full Auditing & Performance Monitoring
// ============================================================================
//
// FEATURES:
// - Full audit trail for every calculation (compliance)
// - Latency histograms and P50/P95/P99 percentiles
// - Outlier detection and alerting
// - Query plan analysis for slow queries
// - Tenant-level resource usage tracking
// - Real-time metrics export (Prometheus-compatible)
//
// ============================================================================

// CalcObserver provides full observability for the calc engine
type CalcObserver struct {
	db       *sql.DB
	config   *ObservabilityConfig
	metrics  *MetricsCollector
	auditLog *AuditLogger
	alerter  *AlertManager
	mu       sync.RWMutex
}

// ObservabilityConfig configures observability features
type ObservabilityConfig struct {
	// Audit settings
	AuditEnabled    bool          `yaml:"audit_enabled"`
	AuditRetention  time.Duration `yaml:"audit_retention"`   // How long to keep audit logs
	AuditSampleRate float64       `yaml:"audit_sample_rate"` // 1.0 = 100% of requests

	// Metrics settings
	MetricsEnabled   bool          `yaml:"metrics_enabled"`
	MetricsInterval  time.Duration `yaml:"metrics_interval"`  // Aggregation interval
	HistogramBuckets []float64     `yaml:"histogram_buckets"` // Latency buckets in ms

	// Alerting settings
	AlertEnabled        bool          `yaml:"alert_enabled"`
	LatencyThresholdP95 time.Duration `yaml:"latency_threshold_p95"` // Alert if P95 exceeds
	LatencyThresholdP99 time.Duration `yaml:"latency_threshold_p99"` // Alert if P99 exceeds
	ErrorRateThreshold  float64       `yaml:"error_rate_threshold"`  // Alert if error rate exceeds

	// Outlier detection
	OutlierStdDevMultiple float64 `yaml:"outlier_std_dev_multiple"` // e.g., 3.0 for 3 std devs

	// Slow query logging
	SlowQueryThreshold time.Duration `yaml:"slow_query_threshold"`
	ExplainSlowQueries bool          `yaml:"explain_slow_queries"`
}

// NewCalcObserver creates a new observability instance
func NewCalcObserver(db *sql.DB, config *ObservabilityConfig) *CalcObserver {
	// Set defaults
	if config.AuditRetention == 0 {
		config.AuditRetention = 90 * 24 * time.Hour // 90 days
	}
	if config.AuditSampleRate == 0 {
		config.AuditSampleRate = 1.0 // 100%
	}
	if len(config.HistogramBuckets) == 0 {
		config.HistogramBuckets = []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000} // ms
	}
	if config.LatencyThresholdP95 == 0 {
		config.LatencyThresholdP95 = 500 * time.Millisecond
	}
	if config.LatencyThresholdP99 == 0 {
		config.LatencyThresholdP99 = 2 * time.Second
	}
	if config.OutlierStdDevMultiple == 0 {
		config.OutlierStdDevMultiple = 3.0
	}
	if config.SlowQueryThreshold == 0 {
		config.SlowQueryThreshold = 1 * time.Second
	}

	obs := &CalcObserver{
		db:      db,
		config:  config,
		metrics: NewMetricsCollector(config.HistogramBuckets),
		auditLog: &AuditLogger{
			db:         db,
			sampleRate: config.AuditSampleRate,
		},
		alerter: &AlertManager{
			config: config,
		},
	}

	return obs
}

// ============================================================================
// AUDIT LOGGING - Compliance-grade calculation tracking
// ============================================================================

// AuditLogger records every calculation for compliance
type AuditLogger struct {
	db         *sql.DB
	sampleRate float64
	mu         sync.Mutex
}

// CalcAuditEntry represents a single calculation audit record
type CalcAuditEntry struct {
	// Identity
	AuditID   string `json:"audit_id"`
	RequestID string `json:"request_id"`

	// Tenant context
	TenantID     string `json:"tenant_id"`
	DatasourceID string `json:"datasource_id"`
	UserID       string `json:"user_id,omitempty"`

	// Calculation details
	CalcType   string `json:"calc_type"` // NAV, XIRR, Returns, etc.
	CalcID     string `json:"calc_id,omitempty"`
	MetricName string `json:"metric_name,omitempty"`

	// Input/Output
	InputParams map[string]interface{} `json:"input_params"`
	OutputValue interface{}            `json:"output_value,omitempty"`
	OutputHash  string                 `json:"output_hash,omitempty"` // For large results

	// Execution details
	DataTier  string `json:"data_tier"` // hot, cold, realtime
	QueryMode string `json:"query_mode"`
	CacheHit  bool   `json:"cache_hit"`

	// Performance
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
	RowsScanned  int64         `json:"rows_scanned,omitempty"`
	RowsReturned int64         `json:"rows_returned,omitempty"`
	BytesScanned int64         `json:"bytes_scanned,omitempty"`

	// Status
	Success      bool   `json:"success"`
	ErrorMessage string `json:"error_message,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`

	// Query analysis (for slow queries)
	SQLQuery  string `json:"sql_query,omitempty"`
	QueryPlan string `json:"query_plan,omitempty"`

	// Source tracking
	SourceIP    string `json:"source_ip,omitempty"`
	UserAgent   string `json:"user_agent,omitempty"`
	APIEndpoint string `json:"api_endpoint,omitempty"`
}

// LogCalculation records a calculation to the audit log
func (l *AuditLogger) LogCalculation(ctx context.Context, entry *CalcAuditEntry) error {
	// Sample rate check
	if l.sampleRate < 1.0 {
		// TODO: implement proper sampling
	}

	entry.AuditID = uuid.New().String()
	if entry.RequestID == "" {
		entry.RequestID = uuid.New().String()
	}

	inputJSON, _ := json.Marshal(entry.InputParams)
	outputJSON, _ := json.Marshal(entry.OutputValue)

	query := `
		INSERT INTO semantic_hot.calc_audit_log (
			audit_id, request_id, tenant_id, datasource_id, user_id,
			calc_type, calc_id, metric_name,
			input_params, output_value, output_hash,
			data_tier, query_mode, cache_hit,
			start_time, end_time, duration_ms,
			rows_scanned, rows_returned, bytes_scanned,
			success, error_message, error_code,
			sql_query, query_plan,
			source_ip, user_agent, api_endpoint,
			created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, NOW())
	`

	_, err := l.db.ExecContext(ctx, query,
		entry.AuditID, entry.RequestID, entry.TenantID, entry.DatasourceID, entry.UserID,
		entry.CalcType, entry.CalcID, entry.MetricName,
		string(inputJSON), string(outputJSON), entry.OutputHash,
		entry.DataTier, entry.QueryMode, entry.CacheHit,
		entry.StartTime, entry.EndTime, entry.Duration.Milliseconds(),
		entry.RowsScanned, entry.RowsReturned, entry.BytesScanned,
		entry.Success, entry.ErrorMessage, entry.ErrorCode,
		entry.SQLQuery, entry.QueryPlan,
		entry.SourceIP, entry.UserAgent, entry.APIEndpoint,
	)

	return err
}

// ============================================================================
// METRICS COLLECTOR - Latency histograms and counters
// ============================================================================

// MetricsCollector collects and aggregates performance metrics
type MetricsCollector struct {
	buckets []float64

	// Per-tenant, per-calc-type metrics
	latencies  map[string]*LatencyHistogram
	counters   map[string]*Counter
	errorRates map[string]*ErrorRate

	mu sync.RWMutex
}

// LatencyHistogram tracks latency distribution
type LatencyHistogram struct {
	Buckets     []float64 `json:"buckets"` // Bucket boundaries in ms
	Counts      []int64   `json:"counts"`  // Count per bucket
	Sum         float64   `json:"sum"`     // Total latency sum
	Count       int64     `json:"count"`   // Total request count
	Min         float64   `json:"min"`     // Minimum latency
	Max         float64   `json:"max"`     // Maximum latency
	Values      []float64 `json:"-"`       // Raw values for percentile calc
	LastUpdated time.Time `json:"last_updated"`
}

// Counter tracks request counts
type Counter struct {
	Total       int64     `json:"total"`
	Success     int64     `json:"success"`
	Failure     int64     `json:"failure"`
	CacheHits   int64     `json:"cache_hits"`
	LastUpdated time.Time `json:"last_updated"`
}

// ErrorRate tracks error rates over time
type ErrorRate struct {
	WindowSize  time.Duration `json:"window_size"`
	Errors      int64         `json:"errors"`
	Total       int64         `json:"total"`
	Rate        float64       `json:"rate"`
	LastUpdated time.Time     `json:"last_updated"`
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(buckets []float64) *MetricsCollector {
	return &MetricsCollector{
		buckets:    buckets,
		latencies:  make(map[string]*LatencyHistogram),
		counters:   make(map[string]*Counter),
		errorRates: make(map[string]*ErrorRate),
	}
}

// RecordLatency records a latency measurement
func (m *MetricsCollector) RecordLatency(tenantID, calcType string, duration time.Duration, success bool, cacheHit bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", tenantID, calcType)

	// Get or create histogram
	hist, ok := m.latencies[key]
	if !ok {
		hist = &LatencyHistogram{
			Buckets: m.buckets,
			Counts:  make([]int64, len(m.buckets)+1),
			Min:     float64(duration.Milliseconds()),
			Max:     float64(duration.Milliseconds()),
			Values:  make([]float64, 0, 10000),
		}
		m.latencies[key] = hist
	}

	ms := float64(duration.Milliseconds())

	// Update histogram
	hist.Sum += ms
	hist.Count++
	if ms < hist.Min {
		hist.Min = ms
	}
	if ms > hist.Max {
		hist.Max = ms
	}

	// Find bucket
	placed := false
	for i, boundary := range hist.Buckets {
		if ms <= boundary {
			hist.Counts[i]++
			placed = true
			break
		}
	}
	if !placed {
		hist.Counts[len(hist.Buckets)]++ // Overflow bucket
	}

	// Store raw value for percentile calculation (limit to last 10k)
	if len(hist.Values) < 10000 {
		hist.Values = append(hist.Values, ms)
	} else {
		// Rotate: remove oldest, add newest
		hist.Values = append(hist.Values[1:], ms)
	}

	hist.LastUpdated = time.Now()

	// Update counter
	counter, ok := m.counters[key]
	if !ok {
		counter = &Counter{}
		m.counters[key] = counter
	}
	counter.Total++
	if success {
		counter.Success++
	} else {
		counter.Failure++
	}
	if cacheHit {
		counter.CacheHits++
	}
	counter.LastUpdated = time.Now()

	// Update error rate
	errRate, ok := m.errorRates[key]
	if !ok {
		errRate = &ErrorRate{WindowSize: 5 * time.Minute}
		m.errorRates[key] = errRate
	}
	errRate.Total++
	if !success {
		errRate.Errors++
	}
	errRate.Rate = float64(errRate.Errors) / float64(errRate.Total)
	errRate.LastUpdated = time.Now()
}

// GetPercentiles calculates P50, P95, P99 for a tenant/calc type
func (m *MetricsCollector) GetPercentiles(tenantID, calcType string) *LatencyPercentiles {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantID, calcType)
	hist, ok := m.latencies[key]
	if !ok || len(hist.Values) == 0 {
		return nil
	}

	// Sort values for percentile calculation
	sorted := make([]float64, len(hist.Values))
	copy(sorted, hist.Values)
	sort.Float64s(sorted)

	return &LatencyPercentiles{
		TenantID:  tenantID,
		CalcType:  calcType,
		P50:       percentile(sorted, 50),
		P75:       percentile(sorted, 75),
		P90:       percentile(sorted, 90),
		P95:       percentile(sorted, 95),
		P99:       percentile(sorted, 99),
		P999:      percentile(sorted, 99.9),
		Min:       hist.Min,
		Max:       hist.Max,
		Avg:       hist.Sum / float64(hist.Count),
		Count:     hist.Count,
		Timestamp: time.Now(),
	}
}

// LatencyPercentiles represents calculated percentiles
type LatencyPercentiles struct {
	TenantID  string    `json:"tenant_id"`
	CalcType  string    `json:"calc_type"`
	P50       float64   `json:"p50_ms"`
	P75       float64   `json:"p75_ms"`
	P90       float64   `json:"p90_ms"`
	P95       float64   `json:"p95_ms"`
	P99       float64   `json:"p99_ms"`
	P999      float64   `json:"p999_ms"`
	Min       float64   `json:"min_ms"`
	Max       float64   `json:"max_ms"`
	Avg       float64   `json:"avg_ms"`
	Count     int64     `json:"count"`
	Timestamp time.Time `json:"timestamp"`
}

func percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(float64(len(sorted)-1) * p / 100)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// ============================================================================
// OUTLIER DETECTION
// ============================================================================

// OutlierDetector identifies abnormally slow calculations
type OutlierDetector struct {
	stdDevMultiple float64
}

// Outlier represents a detected outlier
type Outlier struct {
	AuditID     string        `json:"audit_id"`
	TenantID    string        `json:"tenant_id"`
	CalcType    string        `json:"calc_type"`
	Duration    time.Duration `json:"duration"`
	AvgDuration time.Duration `json:"avg_duration"`
	StdDev      time.Duration `json:"std_dev"`
	ZScore      float64       `json:"z_score"`
	DetectedAt  time.Time     `json:"detected_at"`
	Severity    string        `json:"severity"` // warning, critical
}

// DetectOutliers finds calculations that are significantly slower than average
func (m *MetricsCollector) DetectOutliers(tenantID, calcType string, stdDevMultiple float64) []Outlier {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", tenantID, calcType)
	hist, ok := m.latencies[key]
	if !ok || len(hist.Values) < 10 {
		return nil
	}

	// Calculate mean and std dev
	mean := hist.Sum / float64(hist.Count)
	var variance float64
	for _, v := range hist.Values {
		variance += (v - mean) * (v - mean)
	}
	stdDev := variance / float64(len(hist.Values))
	if stdDev > 0 {
		stdDev = float64(int(stdDev)) // sqrt approximation placeholder
	}

	threshold := mean + (stdDevMultiple * stdDev)

	var outliers []Outlier
	for _, v := range hist.Values {
		if v > threshold {
			zScore := (v - mean) / stdDev
			severity := "warning"
			if zScore > 5 {
				severity = "critical"
			}
			outliers = append(outliers, Outlier{
				TenantID:    tenantID,
				CalcType:    calcType,
				Duration:    time.Duration(v) * time.Millisecond,
				AvgDuration: time.Duration(mean) * time.Millisecond,
				StdDev:      time.Duration(stdDev) * time.Millisecond,
				ZScore:      zScore,
				DetectedAt:  time.Now(),
				Severity:    severity,
			})
		}
	}

	return outliers
}

// ============================================================================
// ALERT MANAGER
// ============================================================================

// AlertManager handles alerting for performance issues
type AlertManager struct {
	config   *ObservabilityConfig
	alerts   []Alert
	handlers []AlertHandler
	mu       sync.Mutex
}

// Alert represents a performance alert
type Alert struct {
	AlertID    string                 `json:"alert_id"`
	AlertType  string                 `json:"alert_type"` // latency_p95, latency_p99, error_rate, outlier
	TenantID   string                 `json:"tenant_id"`
	CalcType   string                 `json:"calc_type"`
	Severity   string                 `json:"severity"` // warning, critical
	Message    string                 `json:"message"`
	Value      float64                `json:"value"`
	Threshold  float64                `json:"threshold"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
	AckedAt    *time.Time             `json:"acked_at,omitempty"`
	ResolvedAt *time.Time             `json:"resolved_at,omitempty"`
}

// AlertHandler processes alerts (webhook, email, slack, etc.)
type AlertHandler interface {
	Handle(alert Alert) error
}

// CheckAndAlert evaluates metrics and fires alerts if thresholds exceeded
func (a *AlertManager) CheckAndAlert(percentiles *LatencyPercentiles, errorRate *ErrorRate) []Alert {
	a.mu.Lock()
	defer a.mu.Unlock()

	var alerts []Alert

	// Check P95 latency
	if percentiles != nil && time.Duration(percentiles.P95)*time.Millisecond > a.config.LatencyThresholdP95 {
		alerts = append(alerts, Alert{
			AlertID:   uuid.New().String(),
			AlertType: "latency_p95",
			TenantID:  percentiles.TenantID,
			CalcType:  percentiles.CalcType,
			Severity:  "warning",
			Message:   fmt.Sprintf("P95 latency %.2fms exceeds threshold %v", percentiles.P95, a.config.LatencyThresholdP95),
			Value:     percentiles.P95,
			Threshold: float64(a.config.LatencyThresholdP95.Milliseconds()),
			CreatedAt: time.Now(),
		})
	}

	// Check P99 latency
	if percentiles != nil && time.Duration(percentiles.P99)*time.Millisecond > a.config.LatencyThresholdP99 {
		alerts = append(alerts, Alert{
			AlertID:   uuid.New().String(),
			AlertType: "latency_p99",
			TenantID:  percentiles.TenantID,
			CalcType:  percentiles.CalcType,
			Severity:  "critical",
			Message:   fmt.Sprintf("P99 latency %.2fms exceeds threshold %v", percentiles.P99, a.config.LatencyThresholdP99),
			Value:     percentiles.P99,
			Threshold: float64(a.config.LatencyThresholdP99.Milliseconds()),
			CreatedAt: time.Now(),
		})
	}

	// Check error rate
	if errorRate != nil && errorRate.Rate > a.config.ErrorRateThreshold {
		alerts = append(alerts, Alert{
			AlertID:   uuid.New().String(),
			AlertType: "error_rate",
			Severity:  "critical",
			Message:   fmt.Sprintf("Error rate %.2f%% exceeds threshold %.2f%%", errorRate.Rate*100, a.config.ErrorRateThreshold*100),
			Value:     errorRate.Rate,
			Threshold: a.config.ErrorRateThreshold,
			CreatedAt: time.Now(),
		})
	}

	// Store and dispatch alerts
	for _, alert := range alerts {
		a.alerts = append(a.alerts, alert)
		for _, handler := range a.handlers {
			go handler.Handle(alert)
		}
	}

	return alerts
}

// ============================================================================
// SLOW QUERY ANALYSIS
// ============================================================================

// SlowQueryAnalyzer analyzes slow queries
type SlowQueryAnalyzer struct {
	db        *sql.DB
	threshold time.Duration
}

// SlowQueryReport represents analysis of a slow query
type SlowQueryReport struct {
	AuditID         string                `json:"audit_id"`
	SQLQuery        string                `json:"sql_query"`
	Duration        time.Duration         `json:"duration"`
	QueryPlan       string                `json:"query_plan"`
	Recommendations []string              `json:"recommendations"`
	TableStats      map[string]TableStats `json:"table_stats"`
	Timestamp       time.Time             `json:"timestamp"`
}

// TableStats contains statistics for a table
type TableStats struct {
	TableName   string     `json:"table_name"`
	RowCount    int64      `json:"row_count"`
	DataSize    int64      `json:"data_size_bytes"`
	IndexSize   int64      `json:"index_size_bytes"`
	LastAnalyze *time.Time `json:"last_analyze"`
}

// AnalyzeSlowQuery analyzes a slow query and provides recommendations
func (a *SlowQueryAnalyzer) AnalyzeSlowQuery(ctx context.Context, query string) (*SlowQueryReport, error) {
	report := &SlowQueryReport{
		SQLQuery:  query,
		Timestamp: time.Now(),
	}

	// Get query plan
	explainQuery := fmt.Sprintf("EXPLAIN %s", query)
	rows, err := a.db.QueryContext(ctx, explainQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to explain query: %w", err)
	}
	defer rows.Close()

	var planLines []string
	for rows.Next() {
		var line string
		if err := rows.Scan(&line); err != nil {
			continue
		}
		planLines = append(planLines, line)
	}
	report.QueryPlan = fmt.Sprintf("%v", planLines)

	// Generate recommendations based on query plan
	report.Recommendations = a.generateRecommendations(query, planLines)

	return report, nil
}

func (a *SlowQueryAnalyzer) generateRecommendations(query string, plan []string) []string {
	var recommendations []string

	planStr := fmt.Sprintf("%v", plan)

	// Check for full table scans
	if containsStr(planStr, "FULL SCAN") || containsStr(planStr, "TABLE SCAN") {
		recommendations = append(recommendations, "Consider adding an index to avoid full table scan")
	}

	// Check for missing indexes
	if containsStr(planStr, "NO INDEX") {
		recommendations = append(recommendations, "Query would benefit from an index on filtered columns")
	}

	// Check for large sorts
	if containsStr(planStr, "SORT") && containsStr(planStr, "EXTERNAL") {
		recommendations = append(recommendations, "Large sort operation detected - consider adding ORDER BY index or reducing result set")
	}

	// Check for cross joins
	if containsStr(query, "CROSS JOIN") || (containsStr(query, ",") && !containsStr(query, "JOIN")) {
		recommendations = append(recommendations, "Cross join detected - ensure this is intentional and not a missing WHERE clause")
	}

	// Check for UNION vs UNION ALL
	if containsStr(query, "UNION") && !containsStr(query, "UNION ALL") {
		recommendations = append(recommendations, "Consider using UNION ALL instead of UNION if duplicates are acceptable (faster)")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "No obvious optimization issues found - consider reviewing data volume and indexes")
	}

	return recommendations
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStrHelper(s, substr))
}

func containsStrHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ============================================================================
// PROMETHEUS METRICS EXPORT
// ============================================================================

// PrometheusExporter exports metrics in Prometheus format
type PrometheusExporter struct {
	collector *MetricsCollector
}

// Export returns Prometheus-formatted metrics
func (e *PrometheusExporter) Export() string {
	e.collector.mu.RLock()
	defer e.collector.mu.RUnlock()

	var output string

	// Latency histograms
	output += "# HELP calcengine_latency_ms Calculation latency in milliseconds\n"
	output += "# TYPE calcengine_latency_ms histogram\n"
	for key, hist := range e.collector.latencies {
		for i, boundary := range hist.Buckets {
			output += fmt.Sprintf("calcengine_latency_ms_bucket{key=\"%s\",le=\"%.0f\"} %d\n", key, boundary, hist.Counts[i])
		}
		output += fmt.Sprintf("calcengine_latency_ms_bucket{key=\"%s\",le=\"+Inf\"} %d\n", key, hist.Counts[len(hist.Buckets)])
		output += fmt.Sprintf("calcengine_latency_ms_sum{key=\"%s\"} %.2f\n", key, hist.Sum)
		output += fmt.Sprintf("calcengine_latency_ms_count{key=\"%s\"} %d\n", key, hist.Count)
	}

	// Counters
	output += "\n# HELP calcengine_requests_total Total calculation requests\n"
	output += "# TYPE calcengine_requests_total counter\n"
	for key, counter := range e.collector.counters {
		output += fmt.Sprintf("calcengine_requests_total{key=\"%s\",status=\"success\"} %d\n", key, counter.Success)
		output += fmt.Sprintf("calcengine_requests_total{key=\"%s\",status=\"failure\"} %d\n", key, counter.Failure)
	}

	// Cache hit rate
	output += "\n# HELP calcengine_cache_hits_total Total cache hits\n"
	output += "# TYPE calcengine_cache_hits_total counter\n"
	for key, counter := range e.collector.counters {
		output += fmt.Sprintf("calcengine_cache_hits_total{key=\"%s\"} %d\n", key, counter.CacheHits)
	}

	// Error rates
	output += "\n# HELP calcengine_error_rate Current error rate\n"
	output += "# TYPE calcengine_error_rate gauge\n"
	for key, errRate := range e.collector.errorRates {
		output += fmt.Sprintf("calcengine_error_rate{key=\"%s\"} %.4f\n", key, errRate.Rate)
	}

	return output
}

// ============================================================================
// PERFORMANCE DASHBOARD DATA
// ============================================================================

// PerformanceDashboard provides data for performance monitoring UI
type PerformanceDashboard struct {
	observer *CalcObserver
}

// DashboardData contains all performance data for display
type DashboardData struct {
	Summary      *PerformanceSummary             `json:"summary"`
	ByTenant     map[string]*TenantPerformance   `json:"by_tenant"`
	ByCalcType   map[string]*CalcTypePerformance `json:"by_calc_type"`
	RecentAlerts []Alert                         `json:"recent_alerts"`
	SlowQueries  []SlowQueryReport               `json:"slow_queries"`
	Outliers     []Outlier                       `json:"outliers"`
	Timestamp    time.Time                       `json:"timestamp"`
}

// PerformanceSummary provides high-level performance overview
type PerformanceSummary struct {
	TotalRequests   int64         `json:"total_requests"`
	TotalErrors     int64         `json:"total_errors"`
	ErrorRate       float64       `json:"error_rate"`
	AvgLatency      float64       `json:"avg_latency_ms"`
	P95Latency      float64       `json:"p95_latency_ms"`
	P99Latency      float64       `json:"p99_latency_ms"`
	CacheHitRate    float64       `json:"cache_hit_rate"`
	ActiveTenants   int           `json:"active_tenants"`
	ActiveCalcTypes int           `json:"active_calc_types"`
	Period          time.Duration `json:"period"`
}

// TenantPerformance shows performance for a specific tenant
type TenantPerformance struct {
	TenantID     string   `json:"tenant_id"`
	Requests     int64    `json:"requests"`
	Errors       int64    `json:"errors"`
	ErrorRate    float64  `json:"error_rate"`
	AvgLatency   float64  `json:"avg_latency_ms"`
	P95Latency   float64  `json:"p95_latency_ms"`
	CacheHitRate float64  `json:"cache_hit_rate"`
	TopCalcTypes []string `json:"top_calc_types"`
}

// CalcTypePerformance shows performance for a specific calculation type
type CalcTypePerformance struct {
	CalcType   string  `json:"calc_type"`
	Requests   int64   `json:"requests"`
	Errors     int64   `json:"errors"`
	ErrorRate  float64 `json:"error_rate"`
	AvgLatency float64 `json:"avg_latency_ms"`
	P95Latency float64 `json:"p95_latency_ms"`
	P99Latency float64 `json:"p99_latency_ms"`
}

// GetDashboardData returns all performance data
func (d *PerformanceDashboard) GetDashboardData() *DashboardData {
	d.observer.mu.RLock()
	defer d.observer.mu.RUnlock()

	data := &DashboardData{
		Summary:    &PerformanceSummary{},
		ByTenant:   make(map[string]*TenantPerformance),
		ByCalcType: make(map[string]*CalcTypePerformance),
		Timestamp:  time.Now(),
	}

	// Aggregate metrics
	var totalRequests, totalErrors, totalCacheHits int64
	var totalLatency float64
	var allLatencies []float64

	for _, hist := range d.observer.metrics.latencies {
		totalRequests += hist.Count
		totalLatency += hist.Sum
		allLatencies = append(allLatencies, hist.Values...)
	}

	for _, counter := range d.observer.metrics.counters {
		totalErrors += counter.Failure
		totalCacheHits += counter.CacheHits
	}

	// Summary
	data.Summary.TotalRequests = totalRequests
	data.Summary.TotalErrors = totalErrors
	if totalRequests > 0 {
		data.Summary.ErrorRate = float64(totalErrors) / float64(totalRequests)
		data.Summary.AvgLatency = totalLatency / float64(totalRequests)
		data.Summary.CacheHitRate = float64(totalCacheHits) / float64(totalRequests)
	}

	// Calculate percentiles
	if len(allLatencies) > 0 {
		sort.Float64s(allLatencies)
		data.Summary.P95Latency = percentile(allLatencies, 95)
		data.Summary.P99Latency = percentile(allLatencies, 99)
	}

	data.Summary.ActiveTenants = len(d.observer.metrics.latencies)
	data.Summary.ActiveCalcTypes = len(d.observer.metrics.counters)

	// Recent alerts
	d.observer.alerter.mu.Lock()
	if len(d.observer.alerter.alerts) > 10 {
		data.RecentAlerts = d.observer.alerter.alerts[len(d.observer.alerter.alerts)-10:]
	} else {
		data.RecentAlerts = d.observer.alerter.alerts
	}
	d.observer.alerter.mu.Unlock()

	return data
}
