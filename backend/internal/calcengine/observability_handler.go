package calcengine

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

// ============================================================================
// OBSERVABILITY HTTP HANDLERS
// ============================================================================

// ObservabilityHandler provides HTTP endpoints for calc engine observability
type ObservabilityHandler struct {
	observer  *CalcObserver
	dashboard *PerformanceDashboard
}

// NewObservabilityHandler creates a new observability handler
func NewObservabilityHandler(observer *CalcObserver) *ObservabilityHandler {
	return &ObservabilityHandler{
		observer: observer,
		dashboard: &PerformanceDashboard{
			observer: observer,
		},
	}
}

// RegisterRoutes registers observability HTTP routes
func (h *ObservabilityHandler) RegisterRoutes(mux *http.ServeMux) {
	// Prometheus metrics endpoint
	mux.HandleFunc("/api/calcengine/metrics", h.handlePrometheusMetrics)

	// Dashboard data
	mux.HandleFunc("/api/calcengine/dashboard", h.handleDashboard)

	// Latency percentiles
	mux.HandleFunc("/api/calcengine/latency", h.handleLatencyPercentiles)

	// Alerts
	mux.HandleFunc("/api/calcengine/alerts", h.handleAlerts)

	// Slow queries
	mux.HandleFunc("/api/calcengine/slow-queries", h.handleSlowQueries)

	// Outliers
	mux.HandleFunc("/api/calcengine/outliers", h.handleOutliers)

	// Audit log query
	mux.HandleFunc("/api/calcengine/audit", h.handleAuditQuery)

	// Health check with performance summary
	mux.HandleFunc("/api/calcengine/health", h.handleHealth)
}

// handlePrometheusMetrics returns metrics in Prometheus format
func (h *ObservabilityHandler) handlePrometheusMetrics(w http.ResponseWriter, r *http.Request) {
	exporter := &PrometheusExporter{collector: h.observer.metrics}
	metrics := exporter.Export()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Write([]byte(metrics))
}

// handleDashboard returns full dashboard data
func (h *ObservabilityHandler) handleDashboard(w http.ResponseWriter, r *http.Request) {
	data := h.dashboard.GetDashboardData()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// handleLatencyPercentiles returns latency percentiles for a tenant/calc type
func (h *ObservabilityHandler) handleLatencyPercentiles(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	calcType := r.URL.Query().Get("calc_type")

	if tenantID == "" {
		// Return all percentiles
		h.observer.metrics.mu.RLock()
		defer h.observer.metrics.mu.RUnlock()

		var allPercentiles []*LatencyPercentiles
		seen := make(map[string]bool)

		for key := range h.observer.metrics.latencies {
			if !seen[key] {
				// Parse tenant:calcType from key
				// For now, return the raw data
				allPercentiles = append(allPercentiles, h.observer.metrics.GetPercentiles("", key))
				seen[key] = true
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(allPercentiles)
		return
	}

	percentiles := h.observer.metrics.GetPercentiles(tenantID, calcType)
	if percentiles == nil {
		http.Error(w, "No data found for specified tenant/calc type", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(percentiles)
}

// AlertsResponse contains alerts with pagination
type AlertsResponse struct {
	Alerts   []Alert `json:"alerts"`
	Total    int     `json:"total"`
	Page     int     `json:"page"`
	PageSize int     `json:"page_size"`
}

// handleAlerts returns recent alerts
func (h *ObservabilityHandler) handleAlerts(w http.ResponseWriter, r *http.Request) {
	h.observer.alerter.mu.Lock()
	defer h.observer.alerter.mu.Unlock()

	// Parse pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Filter by severity
	severity := r.URL.Query().Get("severity")

	var filtered []Alert
	for _, alert := range h.observer.alerter.alerts {
		if severity == "" || alert.Severity == severity {
			filtered = append(filtered, alert)
		}
	}

	// Paginate
	start := (page - 1) * pageSize
	end := start + pageSize
	if start > len(filtered) {
		start = len(filtered)
	}
	if end > len(filtered) {
		end = len(filtered)
	}

	response := AlertsResponse{
		Alerts:   filtered[start:end],
		Total:    len(filtered),
		Page:     page,
		PageSize: pageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSlowQueries returns slow query analysis
func (h *ObservabilityHandler) handleSlowQueries(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")

	// Query slow queries from database
	query := `
		SELECT slow_query_id, audit_id, tenant_id, datasource_id, calc_type,
		       duration_ms, sql_query, query_plan, recommendations,
		       full_scan_detected, created_at
		FROM semantic_hot.calc_slow_queries
		WHERE 1=1
	`
	args := []interface{}{}

	if tenantID != "" {
		query += " AND tenant_id = ?"
		args = append(args, tenantID)
	}
	if datasourceID != "" {
		query += " AND datasource_id = ?"
		args = append(args, datasourceID)
	}

	query += " ORDER BY duration_ms DESC LIMIT 100"

	rows, err := h.observer.db.QueryContext(ctx, query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var slowQueries []SlowQueryReport
	for rows.Next() {
		var sq SlowQueryReport
		var recommendations string
		var fullScan bool
		var createdAt time.Time

		if err := rows.Scan(
			&sq.AuditID, &sq.AuditID, &sq.AuditID, &sq.AuditID, &sq.AuditID,
			&sq.Duration, &sq.SQLQuery, &sq.QueryPlan, &recommendations,
			&fullScan, &createdAt,
		); err != nil {
			continue
		}

		sq.Timestamp = createdAt
		json.Unmarshal([]byte(recommendations), &sq.Recommendations)
		slowQueries = append(slowQueries, sq)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(slowQueries)
}

// handleOutliers returns detected outliers
func (h *ObservabilityHandler) handleOutliers(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	calcType := r.URL.Query().Get("calc_type")

	stdDevMultiple := h.observer.config.OutlierStdDevMultiple
	if mult := r.URL.Query().Get("std_dev_multiple"); mult != "" {
		if parsed, err := strconv.ParseFloat(mult, 64); err == nil {
			stdDevMultiple = parsed
		}
	}

	var allOutliers []Outlier

	if tenantID != "" && calcType != "" {
		// Specific tenant/calc type
		outliers := h.observer.metrics.DetectOutliers(tenantID, calcType, stdDevMultiple)
		allOutliers = append(allOutliers, outliers...)
	} else {
		// All tenant/calc types
		h.observer.metrics.mu.RLock()
		for key := range h.observer.metrics.latencies {
			outliers := h.observer.metrics.DetectOutliers("", key, stdDevMultiple)
			allOutliers = append(allOutliers, outliers...)
		}
		h.observer.metrics.mu.RUnlock()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allOutliers)
}

// AuditQueryRequest represents an audit log query
type AuditQueryRequest struct {
	TenantID     string    `json:"tenant_id"`
	DatasourceID string    `json:"datasource_id"`
	CalcType     string    `json:"calc_type"`
	UserID       string    `json:"user_id"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	SuccessOnly  *bool     `json:"success_only"`
	MinDuration  int64     `json:"min_duration_ms"`
	Page         int       `json:"page"`
	PageSize     int       `json:"page_size"`
}

// AuditQueryResponse contains audit log results
type AuditQueryResponse struct {
	Entries  []CalcAuditEntry `json:"entries"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

// handleAuditQuery queries the audit log
func (h *ObservabilityHandler) handleAuditQuery(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse query parameters
	tenantID := r.URL.Query().Get("tenant_id")
	datasourceID := r.URL.Query().Get("datasource_id")
	calcType := r.URL.Query().Get("calc_type")
	userID := r.URL.Query().Get("user_id")

	// Time range
	startTime := time.Now().Add(-24 * time.Hour) // Default: last 24 hours
	if st := r.URL.Query().Get("start_time"); st != "" {
		if parsed, err := time.Parse(time.RFC3339, st); err == nil {
			startTime = parsed
		}
	}

	endTime := time.Now()
	if et := r.URL.Query().Get("end_time"); et != "" {
		if parsed, err := time.Parse(time.RFC3339, et); err == nil {
			endTime = parsed
		}
	}

	// Filters
	minDuration, _ := strconv.ParseInt(r.URL.Query().Get("min_duration_ms"), 10, 64)

	// Pagination
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if pageSize < 1 || pageSize > 100 {
		pageSize = 50
	}

	// Build query
	query := `
		SELECT 
			audit_id, request_id, tenant_id, datasource_id, user_id,
			calc_type, calc_id, metric_name,
			input_params, output_value,
			data_tier, query_mode, cache_hit,
			start_time, end_time, duration_ms,
			rows_scanned, rows_returned, bytes_scanned,
			success, error_message, error_code,
			source_ip, api_endpoint
		FROM semantic_hot.calc_audit_log
		WHERE start_time >= ? AND start_time <= ?
	`
	args := []interface{}{startTime, endTime}

	if tenantID != "" {
		query += " AND tenant_id = ?"
		args = append(args, tenantID)
	}
	if datasourceID != "" {
		query += " AND datasource_id = ?"
		args = append(args, datasourceID)
	}
	if calcType != "" {
		query += " AND calc_type = ?"
		args = append(args, calcType)
	}
	if userID != "" {
		query += " AND user_id = ?"
		args = append(args, userID)
	}
	if minDuration > 0 {
		query += " AND duration_ms >= ?"
		args = append(args, minDuration)
	}

	// Count total
	countQuery := "SELECT COUNT(*) FROM (" + query + ") t"
	var total int64
	h.observer.db.QueryRowContext(ctx, countQuery, args...).Scan(&total)

	// Add pagination
	offset := (page - 1) * pageSize
	query += " ORDER BY start_time DESC LIMIT ? OFFSET ?"
	args = append(args, pageSize, offset)

	rows, err := h.observer.db.QueryContext(ctx, query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var entries []CalcAuditEntry
	for rows.Next() {
		var entry CalcAuditEntry
		var inputJSON, outputJSON string
		var durationMS int64

		if err := rows.Scan(
			&entry.AuditID, &entry.RequestID, &entry.TenantID, &entry.DatasourceID, &entry.UserID,
			&entry.CalcType, &entry.CalcID, &entry.MetricName,
			&inputJSON, &outputJSON,
			&entry.DataTier, &entry.QueryMode, &entry.CacheHit,
			&entry.StartTime, &entry.EndTime, &durationMS,
			&entry.RowsScanned, &entry.RowsReturned, &entry.BytesScanned,
			&entry.Success, &entry.ErrorMessage, &entry.ErrorCode,
			&entry.SourceIP, &entry.APIEndpoint,
		); err != nil {
			continue
		}

		entry.Duration = time.Duration(durationMS) * time.Millisecond
		json.Unmarshal([]byte(inputJSON), &entry.InputParams)
		json.Unmarshal([]byte(outputJSON), &entry.OutputValue)

		entries = append(entries, entry)
	}

	response := AuditQueryResponse{
		Entries:  entries,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HealthResponse contains health status and performance summary
type HealthResponse struct {
	Status       string              `json:"status"` // healthy, degraded, unhealthy
	Timestamp    time.Time           `json:"timestamp"`
	Performance  *PerformanceSummary `json:"performance"`
	ActiveAlerts int                 `json:"active_alerts"`
	Checks       []HealthCheck       `json:"checks"`
}

// HealthCheck represents a single health check
type HealthCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // pass, warn, fail
	Message string `json:"message,omitempty"`
	Value   string `json:"value,omitempty"`
}

// handleHealth returns health status with performance summary
func (h *ObservabilityHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	data := h.dashboard.GetDashboardData()

	response := HealthResponse{
		Status:      "healthy",
		Timestamp:   time.Now(),
		Performance: data.Summary,
		Checks:      []HealthCheck{},
	}

	// Check error rate
	if data.Summary.ErrorRate > 0.1 {
		response.Status = "unhealthy"
		response.Checks = append(response.Checks, HealthCheck{
			Name:    "error_rate",
			Status:  "fail",
			Message: "Error rate exceeds 10%",
			Value:   strconv.FormatFloat(data.Summary.ErrorRate*100, 'f', 2, 64) + "%",
		})
	} else if data.Summary.ErrorRate > 0.01 {
		response.Status = "degraded"
		response.Checks = append(response.Checks, HealthCheck{
			Name:    "error_rate",
			Status:  "warn",
			Message: "Error rate exceeds 1%",
			Value:   strconv.FormatFloat(data.Summary.ErrorRate*100, 'f', 2, 64) + "%",
		})
	} else {
		response.Checks = append(response.Checks, HealthCheck{
			Name:   "error_rate",
			Status: "pass",
			Value:  strconv.FormatFloat(data.Summary.ErrorRate*100, 'f', 2, 64) + "%",
		})
	}

	// Check P95 latency
	if data.Summary.P95Latency > 2000 { // 2 seconds
		response.Status = "unhealthy"
		response.Checks = append(response.Checks, HealthCheck{
			Name:    "latency_p95",
			Status:  "fail",
			Message: "P95 latency exceeds 2000ms",
			Value:   strconv.FormatFloat(data.Summary.P95Latency, 'f', 0, 64) + "ms",
		})
	} else if data.Summary.P95Latency > 500 { // 500ms
		if response.Status == "healthy" {
			response.Status = "degraded"
		}
		response.Checks = append(response.Checks, HealthCheck{
			Name:    "latency_p95",
			Status:  "warn",
			Message: "P95 latency exceeds 500ms",
			Value:   strconv.FormatFloat(data.Summary.P95Latency, 'f', 0, 64) + "ms",
		})
	} else {
		response.Checks = append(response.Checks, HealthCheck{
			Name:   "latency_p95",
			Status: "pass",
			Value:  strconv.FormatFloat(data.Summary.P95Latency, 'f', 0, 64) + "ms",
		})
	}

	// Check cache hit rate
	if data.Summary.CacheHitRate < 0.5 && data.Summary.TotalRequests > 100 {
		if response.Status == "healthy" {
			response.Status = "degraded"
		}
		response.Checks = append(response.Checks, HealthCheck{
			Name:    "cache_hit_rate",
			Status:  "warn",
			Message: "Cache hit rate below 50%",
			Value:   strconv.FormatFloat(data.Summary.CacheHitRate*100, 'f', 1, 64) + "%",
		})
	} else {
		response.Checks = append(response.Checks, HealthCheck{
			Name:   "cache_hit_rate",
			Status: "pass",
			Value:  strconv.FormatFloat(data.Summary.CacheHitRate*100, 'f', 1, 64) + "%",
		})
	}

	// Count active alerts
	h.observer.alerter.mu.Lock()
	for _, alert := range h.observer.alerter.alerts {
		if alert.ResolvedAt == nil {
			response.ActiveAlerts++
		}
	}
	h.observer.alerter.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
