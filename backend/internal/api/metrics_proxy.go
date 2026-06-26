package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// PrometheusQueryResult represents a single Prometheus query result
type PrometheusQueryResult struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  [2]interface{}    `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

// CommitMetricsResponse represents commit metrics for a specific plan
type CommitMetricsResponse struct {
	PlanID            string  `json:"planId"`
	CommitLatencyMs   float64 `json:"commitLatencyMs"`
	S3Failures        int     `json:"s3Failures"`
	IdempotencyHits   int     `json:"idempotencyHits"`
	CommitSuccessRate float64 `json:"commitSuccessRate"`
	Table             string  `json:"table"`
	Region            string  `json:"region"`
	Timestamp         string  `json:"timestamp"`
}

// queryPrometheus executes a PromQL query against the Prometheus server
func queryPrometheus(ctx context.Context, query string) (*PrometheusQueryResult, error) {
	prometheusURL := os.Getenv("PROMETHEUS_URL")
	if prometheusURL == "" {
		prometheusURL = "http://prometheus:9090"
	}

	// Build query URL
	queryURL, err := url.Parse(prometheusURL)
	if err != nil {
		return nil, fmt.Errorf("invalid prometheus URL: %w", err)
	}

	queryURL.Path = "/api/v1/query"
	params := url.Values{}
	params.Set("query", query)
	queryURL.RawQuery = params.Encode()

	// Execute query with timeout
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queryURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create prometheus request: %w", err)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("prometheus query failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("prometheus query returned status %d: %s", resp.StatusCode, string(body))
	}

	var result PrometheusQueryResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode prometheus response: %w", err)
	}

	if result.Status != "success" {
		return nil, fmt.Errorf("prometheus query failed with status: %s", result.Status)
	}

	return &result, nil
}

// getFloatValue extracts a float value from prometheus result
func getFloatValue(result *PrometheusQueryResult) float64 {
	if len(result.Data.Result) == 0 || len(result.Data.Result[0].Value) < 2 {
		return 0
	}

	switch v := result.Data.Result[0].Value[1].(type) {
	case string:
		val, _ := strconv.ParseFloat(v, 64)
		return val
	case float64:
		return v
	default:
		return 0
	}
}

// getMetricLabel extracts a label value from the first result
func getMetricLabel(result *PrometheusQueryResult, label string) string {
	if len(result.Data.Result) == 0 {
		return ""
	}
	return result.Data.Result[0].Metric[label]
}

// commitMetricsHandler returns commit-related metrics for a given plan_id by querying Prometheus
func (s *Server) commitMetricsHandler(w http.ResponseWriter, r *http.Request) {
	planID := r.URL.Query().Get("plan_id")
	if planID == "" {
		http.Error(w, "missing required query parameter: plan_id", http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	response := CommitMetricsResponse{
		PlanID:    planID,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	// Query commit latency for this plan
	latencyQuery := fmt.Sprintf(
		"histogram_quantile(0.95, rate(commit_latency_milliseconds_bucket{plan_id=\"%s\"}[5m]))",
		sanitizePromQL(planID),
	)
	latencyResult, err := queryPrometheus(ctx, latencyQuery)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": fmt.Sprintf("failed to query commit latency: %v", err)})
		return
	}
	response.CommitLatencyMs = getFloatValue(latencyResult)

	// Query S3 failures for this plan
	s3FailuresQuery := fmt.Sprintf(
		"increase(s3_failures_total{plan_id=\"%s\"}[5m])",
		sanitizePromQL(planID),
	)
	s3Result, err := queryPrometheus(ctx, s3FailuresQuery)
	if err == nil {
		response.S3Failures = int(getFloatValue(s3Result))
	}

	// Query idempotency hits for this plan
	idempotencyQuery := fmt.Sprintf(
		"increase(idempotency_hits_total{plan_id=\"%s\"}[5m])",
		sanitizePromQL(planID),
	)
	idempotencyResult, err := queryPrometheus(ctx, idempotencyQuery)
	if err == nil {
		response.IdempotencyHits = int(getFloatValue(idempotencyResult))
	}

	// Query commit success rate for this plan
	successRateQuery := fmt.Sprintf(
		"(increase(commits_total{plan_id=\"%s\",status=\"success\"}[5m]) / increase(commits_total{plan_id=\"%s\"}[5m])) * 100",
		sanitizePromQL(planID), sanitizePromQL(planID),
	)
	successResult, err := queryPrometheus(ctx, successRateQuery)
	if err == nil {
		response.CommitSuccessRate = getFloatValue(successResult)
	}

	// Query table and region metadata for this plan
	metadataQuery := fmt.Sprintf(
		"max_over_time(commit_metadata{plan_id=\"%s\"}[1h])",
		sanitizePromQL(planID),
	)
	metadataResult, err := queryPrometheus(ctx, metadataQuery)
	if err == nil {
		response.Table = getMetricLabel(metadataResult, "table")
		response.Region = getMetricLabel(metadataResult, "region")
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// sanitizePromQL escapes special characters in PromQL string literals
func sanitizePromQL(s string) string {
	return strings.ReplaceAll(s, "\"", "\\\"")
}

// ---------------------------
// Versioned metrics endpoint
// ---------------------------

// CommitMetricsV1Response is the v1 response contract for commit metrics
type CommitMetricsV1Response struct {
	Window   string                `json:"window"`
	Global   GlobalCommitMetrics   `json:"global"`
	ByRegion []RegionCommitMetrics `json:"byRegion"`
	ByTenant []TenantCommitMetrics `json:"byTenant"`
}

type GlobalCommitMetrics struct {
	SuccessRate  float64          `json:"successRate"`
	SuccessCount float64          `json:"successCount"`
	FailureCount float64          `json:"failureCount"`
	S3Failures   float64          `json:"s3Failures"`
	Idempotency  float64          `json:"idempotencyHits"`
	LatencyMs    LatencyQuantiles `json:"latencyMs"`
}

// LatencyQuantiles provides p50/p95/p99 latency data in ms
type LatencyQuantiles struct {
	P50 float64 `json:"p50"`
	P95 float64 `json:"p95"`
	P99 float64 `json:"p99"`
}

// RegionCommitMetrics holds metrics aggregated by region
type RegionCommitMetrics struct {
	Region       string           `json:"region"`
	SuccessRate  float64          `json:"successRate"`
	SuccessCount float64          `json:"successCount"`
	FailureCount float64          `json:"failureCount"`
	S3Failures   float64          `json:"s3Failures"`
	LatencyMs    LatencyQuantiles `json:"latencyMs"`
}

// TenantCommitMetrics holds metrics per tenant
type TenantCommitMetrics struct {
	TenantID       string  `json:"tenantId"`
	SuccessRate    float64 `json:"successRate"`
	SuccessCount   float64 `json:"successCount"`
	FailureCount   float64 `json:"failureCount"`
	S3Failures     float64 `json:"s3Failures"`
	IdempotencyHit float64 `json:"idempotencyHits"`
}

// commitMetricsV1Handler implements the versioned metrics API described in the design doc
func (s *Server) commitMetricsV1Handler(w http.ResponseWriter, r *http.Request) {
	window := r.URL.Query().Get("window")
	if window == "" {
		window = "5m"
	}
	tenantID := r.URL.Query().Get("tenant_id")
	region := r.URL.Query().Get("region")

	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	resp := CommitMetricsV1Response{Window: window}

	// Build label filter
	filter := buildLabelFilter(tenantID, region)

	// Global metrics
	resp.Global = buildGlobalMetricsV1(ctx, filter, window)

	// Region metrics (group by region)
	resp.ByRegion = buildRegionMetricsV1(ctx, tenantID, window)

	// Tenant metrics (group by tenant)
	resp.ByTenant = buildTenantMetricsV1(ctx, region, window)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

func buildLabelFilter(tenantID, region string) string {
	labels := []string{}
	if tenantID != "" {
		labels = append(labels, fmt.Sprintf(`tenant_id="%s"`, tenantID))
	}
	if region != "" {
		labels = append(labels, fmt.Sprintf(`region="%s"`, region))
	}
	if len(labels) == 0 {
		return ""
	}
	return "{" + strings.Join(labels, ",") + "}"
}

func buildGlobalMetricsV1(ctx context.Context, filter, window string) GlobalCommitMetrics {
	successQ := fmt.Sprintf(`sum(rate(commit_service_commits_success_total%s[%s]))`, filter, window)
	failQ := fmt.Sprintf(`sum(rate(commit_service_commits_failed_total%s[%s]))`, filter, window)
	s3Q := fmt.Sprintf(`sum(rate(commit_service_s3_validation_failures_total%s[%s]))`, filter, window)
	idemQ := fmt.Sprintf(`sum(rate(commit_service_idempotency_hits_total%s[%s]))`, filter, window)
	p50Q := fmt.Sprintf(`histogram_quantile(0.50, sum(rate(commit_service_commit_latency_ms_bucket%s[%s])) by (le))`, filter, window)
	p95Q := fmt.Sprintf(`histogram_quantile(0.95, sum(rate(commit_service_commit_latency_ms_bucket%s[%s])) by (le))`, filter, window)
	p99Q := fmt.Sprintf(`histogram_quantile(0.99, sum(rate(commit_service_commit_latency_ms_bucket%s[%s])) by (le))`, filter, window)

	success := 0.0
	if r, err := queryPrometheus(ctx, successQ); err == nil {
		success = getFloatValue(r)
	}
	fail := 0.0
	if r, err := queryPrometheus(ctx, failQ); err == nil {
		fail = getFloatValue(r)
	}
	s3 := 0.0
	if r, err := queryPrometheus(ctx, s3Q); err == nil {
		s3 = getFloatValue(r)
	}
	idem := 0.0
	if r, err := queryPrometheus(ctx, idemQ); err == nil {
		idem = getFloatValue(r)
	}
	p50 := 0.0
	if r, err := queryPrometheus(ctx, p50Q); err == nil {
		p50 = getFloatValue(r)
	}
	p95 := 0.0
	if r, err := queryPrometheus(ctx, p95Q); err == nil {
		p95 = getFloatValue(r)
	}
	p99 := 0.0
	if r, err := queryPrometheus(ctx, p99Q); err == nil {
		p99 = getFloatValue(r)
	}

	total := success + fail
	rate := 0.0
	if total > 0 {
		rate = success / total
	}

	return GlobalCommitMetrics{
		SuccessRate:  rate,
		SuccessCount: success,
		FailureCount: fail,
		S3Failures:   s3,
		Idempotency:  idem,
		LatencyMs: LatencyQuantiles{
			P50: p50,
			P95: p95,
			P99: p99,
		},
	}
}

// queryVector returns a map[labelValue]float64 for a query that returns multiple series with a label
func queryVector(ctx context.Context, query string, label string) map[string]float64 {
	out := map[string]float64{}
	res, err := queryPrometheus(ctx, query)
	if err != nil {
		return out
	}

	for _, r := range res.Data.Result {
		val := 0.0
		switch v := r.Value[1].(type) {
		case string:
			f, _ := strconv.ParseFloat(v, 64)
			val = f
		case float64:
			val = v
		}
		key := r.Metric[label]
		out[key] = val
	}
	return out
}

func buildRegionMetricsV1(ctx context.Context, tenantID, window string) []RegionCommitMetrics {
	filter := ""
	if tenantID != "" {
		filter = fmt.Sprintf(`{tenant_id="%s"}`, tenantID)
	}

	successQ := fmt.Sprintf(`sum(rate(commit_service_commits_success_total%s[%s])) by (region)`, filter, window)
	failQ := fmt.Sprintf(`sum(rate(commit_service_commits_failed_total%s[%s])) by (region)`, filter, window)
	s3Q := fmt.Sprintf(`sum(rate(commit_service_s3_validation_failures_total%s[%s])) by (region)`, filter, window)
	p95Q := fmt.Sprintf(`histogram_quantile(0.95, sum(rate(commit_service_commit_latency_ms_bucket%s[%s])) by (region, le))`, filter, window)

	successMap := queryVector(ctx, successQ, "region")
	failMap := queryVector(ctx, failQ, "region")
	s3Map := queryVector(ctx, s3Q, "region")
	p95Map := queryVector(ctx, p95Q, "region")

	regions := map[string]RegionCommitMetrics{}
	for r, s := range successMap {
		rm := regions[r]
		rm.Region = r
		rm.SuccessCount = s
		rm.FailureCount = failMap[r]
		rm.S3Failures = s3Map[r]
		rm.LatencyMs.P95 = p95Map[r]
		regions[r] = rm
	}

	out := []RegionCommitMetrics{}
	for _, v := range regions {
		out = append(out, v)
	}
	return out
}

func buildTenantMetricsV1(ctx context.Context, region, window string) []TenantCommitMetrics {
	filter := ""
	if region != "" {
		filter = fmt.Sprintf(`{region="%s"}`, region)
	}

	successQ := fmt.Sprintf(`sum(rate(commit_service_commits_success_total%s[%s])) by (tenant_id)`, filter, window)
	failQ := fmt.Sprintf(`sum(rate(commit_service_commits_failed_total%s[%s])) by (tenant_id)`, filter, window)
	s3Q := fmt.Sprintf(`sum(rate(commit_service_s3_validation_failures_total%s[%s])) by (tenant_id)`, filter, window)
	idemQ := fmt.Sprintf(`sum(rate(commit_service_idempotency_hits_total%s[%s])) by (tenant_id)`, filter, window)

	successMap := queryVector(ctx, successQ, "tenant_id")
	failMap := queryVector(ctx, failQ, "tenant_id")
	s3Map := queryVector(ctx, s3Q, "tenant_id")
	idemMap := queryVector(ctx, idemQ, "tenant_id")

	out := []TenantCommitMetrics{}
	for tenant, s := range successMap {
		total := s + failMap[tenant]
		rate := 0.0
		if total > 0 {
			rate = s / total
		}
		out = append(out, TenantCommitMetrics{
			TenantID:       tenant,
			SuccessRate:    rate,
			SuccessCount:   s,
			FailureCount:   failMap[tenant],
			S3Failures:     s3Map[tenant],
			IdempotencyHit: idemMap[tenant],
		})
	}
	return out
}
