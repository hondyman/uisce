package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
)

// GlobalMetrics represents system-wide observability KPIs
type GlobalMetrics struct {
	CommitSuccessRate  float64 `json:"commitSuccessRate"`
	S3Failures5m       int     `json:"s3Failures5m"`
	IdempotencyHits5m  int     `json:"idempotencyHits5m"`
	RegionsDegraded    int     `json:"regionsDegraded"`
	AvgCommitLatencyMs float64 `json:"avgCommitLatencyMs"`
	P95CommitLatencyMs float64 `json:"p95CommitLatencyMs"`
	ActiveRegions      int     `json:"activeRegions"`
	Timestamp          string  `json:"timestamp"`
}

// RegionHeatmapPoint represents a single heatmap cell
type RegionHeatmapPoint struct {
	Region string  `json:"region"`
	Bucket string  `json:"bucket"`
	Value  float64 `json:"value"`
}

// TenantMetrics represents per-tenant KPIs
type TenantMetrics struct {
	TenantID        string  `json:"tenantId"`
	SuccessRate     float64 `json:"successRate"`
	S3Failures      int     `json:"s3Failures"`
	IdempotencyHits int     `json:"idempotencyHits"`
	AvgLatencyMs    float64 `json:"avgLatencyMs"`
	Timestamp       string  `json:"timestamp"`
}

// PlanRecord represents a single plan in recent plans list
type PlanRecord struct {
	ID        string  `json:"id"`
	Table     string  `json:"table"`
	Region    string  `json:"region"`
	Status    string  `json:"status"`
	Latency   float64 `json:"latency"`
	Timestamp string  `json:"timestamp"`
}

// TimelineEvent represents a single plan event in chronological order
type TimelineEvent struct {
	PlanID    string  `json:"planId"`
	Table     string  `json:"table"`
	Region    string  `json:"region"`
	Status    string  `json:"status"`
	Latency   float64 `json:"latency"`
	Timestamp string  `json:"timestamp"`
}

// SnapshotInfo represents a single Iceberg snapshot
type SnapshotInfo struct {
	SnapshotID       int64  `json:"snapshotId"`
	ParentSnapshotID *int64 `json:"parentSnapshotId,omitempty"`
	Timestamp        string `json:"timestamp"`
	FileCount        int    `json:"fileCount"`
	DataBytes        int64  `json:"dataBytes"`
}

// globalMetricsHandler returns system-wide health KPIs queried from Prometheus
func (s *Server) globalMetricsHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	metrics := GlobalMetrics{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Query commit success rate (5m window)
	successQuery := "(increase(commits_total{status=\"success\"}[5m]) / increase(commits_total[5m])) * 100"
	if result, err := queryPrometheus(ctx, successQuery); err == nil {
		metrics.CommitSuccessRate = getFloatValue(result)
	}

	// Query S3 failures in last 5 minutes
	s3FailuresQuery := "increase(s3_failures_total[5m])"
	if result, err := queryPrometheus(ctx, s3FailuresQuery); err == nil {
		metrics.S3Failures5m = int(getFloatValue(result))
	}

	// Query idempotency hits in last 5 minutes
	idempotencyQuery := "increase(idempotency_hits_total[5m])"
	if result, err := queryPrometheus(ctx, idempotencyQuery); err == nil {
		metrics.IdempotencyHits5m = int(getFloatValue(result))
	}

	// Query number of degraded regions
	degradedRegionsQuery := "count(up{job=\"region_health\"} == 0 or up{job=\"region_health\"} < 1)"
	if result, err := queryPrometheus(ctx, degradedRegionsQuery); err == nil {
		metrics.RegionsDegraded = int(getFloatValue(result))
	}

	// Query average commit latency
	avgLatencyQuery := "avg(rate(commit_latency_milliseconds_sum[5m]) / rate(commit_latency_milliseconds_count[5m]))"
	if result, err := queryPrometheus(ctx, avgLatencyQuery); err == nil {
		metrics.AvgCommitLatencyMs = getFloatValue(result)
	}

	// Query 95th percentile commit latency
	p95LatencyQuery := "histogram_quantile(0.95, rate(commit_latency_milliseconds_bucket[5m]))"
	if result, err := queryPrometheus(ctx, p95LatencyQuery); err == nil {
		metrics.P95CommitLatencyMs = getFloatValue(result)
	}

	// Query number of active regions
	activeRegionsQuery := "count(up{job=\"region_health\"} == 1)"
	if result, err := queryPrometheus(ctx, activeRegionsQuery); err == nil {
		metrics.ActiveRegions = int(getFloatValue(result))
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=30")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

// regionHeatmapHandler returns regional latency heatmap data queried from Prometheus
func (s *Server) regionHeatmapHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	heatmap := make([]RegionHeatmapPoint, 0)

	// Query latency by region and time bucket
	query := "increase(commit_latency_milliseconds_sum{job=\"commits\"}[5m]) / increase(commit_latency_milliseconds_count{job=\"commits\"}[5m])"
	result, err := queryPrometheus(ctx, query)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to query heatmap data: %v", err)})
		return
	}

	// Process results into heatmap grid
	for _, r := range result.Data.Result {
		point := RegionHeatmapPoint{
			Region: r.Metric["region"],
			Bucket: "current",
			Value: getFloatValue(&PrometheusQueryResult{Data: struct {
				ResultType string `json:"resultType"`
				Result     []struct {
					Metric map[string]string `json:"metric"`
					Value  [2]interface{}    `json:"value"`
				} `json:"result"`
			}{Result: []struct {
				Metric map[string]string `json:"metric"`
				Value  [2]interface{}    `json:"value"`
			}{{Metric: r.Metric, Value: r.Value}}}}),
		}
		heatmap = append(heatmap, point)
	}

	// If we have few results, add some historical buckets for context
	if len(heatmap) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(heatmap)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=60")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(heatmap)
}

// tenantMetricsHandler returns metrics for a specific tenant queried from Prometheus
func (s *Server) tenantMetricsHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := chi.URLParam(r, "tenantId")
	if tenantID == "" {
		http.Error(w, "missing tenantId parameter", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	metrics := TenantMetrics{
		TenantID:  tenantID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Query success rate for this tenant
	successQuery := fmt.Sprintf(
		"(increase(commits_total{tenant_id=\"%s\",status=\"success\"}[5m]) / increase(commits_total{tenant_id=\"%s\"}[5m])) * 100",
		sanitizePromQL(tenantID), sanitizePromQL(tenantID),
	)
	if result, err := queryPrometheus(ctx, successQuery); err == nil {
		metrics.SuccessRate = getFloatValue(result)
	}

	// Query S3 failures for this tenant
	s3FailuresQuery := fmt.Sprintf(
		"increase(s3_failures_total{tenant_id=\"%s\"}[5m])",
		sanitizePromQL(tenantID),
	)
	if result, err := queryPrometheus(ctx, s3FailuresQuery); err == nil {
		metrics.S3Failures = int(getFloatValue(result))
	}

	// Query idempotency hits for this tenant
	idempotencyQuery := fmt.Sprintf(
		"increase(idempotency_hits_total{tenant_id=\"%s\"}[5m])",
		sanitizePromQL(tenantID),
	)
	if result, err := queryPrometheus(ctx, idempotencyQuery); err == nil {
		metrics.IdempotencyHits = int(getFloatValue(result))
	}

	// Query average latency for this tenant
	latencyQuery := fmt.Sprintf(
		"avg(rate(commit_latency_milliseconds_sum{tenant_id=\"%s\"}[5m]) / rate(commit_latency_milliseconds_count{tenant_id=\"%s\"}[5m]))",
		sanitizePromQL(tenantID), sanitizePromQL(tenantID),
	)
	if result, err := queryPrometheus(ctx, latencyQuery); err == nil {
		metrics.AvgLatencyMs = getFloatValue(result)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=30")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(metrics)
}

// tenantPlansHandler returns recent plans for a specific tenant queried from Prometheus/database
func (s *Server) tenantPlansHandler(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant")
	if tenantID == "" {
		http.Error(w, "missing tenant query parameter", http.StatusBadRequest)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
		limit = l
	}

	plans := make([]PlanRecord, 0)

	// Query recent plans for this tenant from Prometheus exemplars or time series
	ctx := r.Context()
	query := fmt.Sprintf(
		"topk(%d, group by (plan_id, table, region, status) (max_over_time(commit_status{tenant_id=\"%s\"}[1h])))",
		limit, sanitizePromQL(tenantID),
	)

	result, err := queryPrometheus(ctx, query)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to query plans: %v", err)})
		return
	}

	// Parse results into plan records
	for _, r := range result.Data.Result {
		var plan PlanRecord
		plan.ID = r.Metric["plan_id"]
		plan.Table = r.Metric["table"]
		plan.Region = r.Metric["region"]

		// Get status value
		if val, ok := r.Metric["status"]; ok {
			plan.Status = val
		} else {
			plan.Status = "unknown"
		}

		// Get latency from this result's value
		if v, ok := r.Value[1].(string); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				plan.Latency = f
			}
		}

		plan.Timestamp = time.Now().UTC().Format(time.RFC3339)
		plans = append(plans, plan)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=60")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(plans)
}

// planTimelineHandler returns chronological plan timeline queried from Prometheus
func (s *Server) planTimelineHandler(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	limit := 50
	if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 1000 {
		limit = l
	}

	events := make([]TimelineEvent, 0)

	// Query recent plan events from Prometheus
	ctx := r.Context()
	query := fmt.Sprintf(
		"topk(%d, max_over_time(commit_status[1h]))",
		limit,
	)

	result, err := queryPrometheus(ctx, query)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to query timeline: %v", err)})
		return
	}

	// Parse results into timeline events
	for _, r := range result.Data.Result {
		event := TimelineEvent{
			PlanID:    r.Metric["plan_id"],
			Table:     r.Metric["table"],
			Region:    r.Metric["region"],
			Status:    r.Metric["status"],
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}

		// Parse latency from value
		if v, ok := r.Value[1].(string); ok {
			if f, err := strconv.ParseFloat(v, 64); err == nil {
				event.Latency = f
			}
		}

		events = append(events, event)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=60")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(events)
}

// icebergLineageHandler returns Iceberg snapshot lineage for a table
func (s *Server) icebergLineageHandler(w http.ResponseWriter, r *http.Request) {
	table := r.URL.Query().Get("table")
	if table == "" {
		http.Error(w, "missing table query parameter", http.StatusBadRequest)
		return
	}

	snapshots := make([]SnapshotInfo, 0)

	// Query Iceberg snapshots for this table from Prometheus/metadata store
	ctx := r.Context()
	query := fmt.Sprintf(
		"max_over_time(iceberg_snapshot_metadata{table=\"%s\"}[1h])",
		sanitizePromQL(table),
	)

	result, err := queryPrometheus(ctx, query)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to query snapshots: %v", err)})
		return
	}

	// Parse results into snapshot info
	for _, r := range result.Data.Result {
		var snapshot SnapshotInfo

		// Parse snapshot ID
		if id, err := strconv.ParseInt(r.Metric["snapshot_id"], 10, 64); err == nil {
			snapshot.SnapshotID = id
		}

		// Parse file count
		if count, err := strconv.Atoi(r.Metric["file_count"]); err == nil {
			snapshot.FileCount = count
		}

		// Parse data bytes
		if bytes, err := strconv.ParseInt(r.Metric["data_bytes"], 10, 64); err == nil {
			snapshot.DataBytes = bytes
		}

		snapshot.Timestamp = time.Now().UTC().Format(time.RFC3339)
		snapshots = append(snapshots, snapshot)
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=300")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(snapshots)
}
