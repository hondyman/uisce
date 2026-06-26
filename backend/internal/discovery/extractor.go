package discovery

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// MetricExtractor discovers feature candidates from time-series metrics
type MetricExtractor struct {
	prometheusURL string
	logger        *log.Logger
}

// MetricInfo represents metadata about a discovered metric
type MetricInfo struct {
	MetricName    string
	MetricType    string // gauge, counter, histogram, summary
	Help          string
	Labels        map[string][]string // label_name -> [label_values]
	SampleValue   float64
	Cardinality   int64
	DiscoveryTime time.Time
	IsAggregated  bool // true if metric is already aggregated (sum, rate, etc)
}

// NewMetricExtractor creates a new metric extractor
func NewMetricExtractor(prometheusURL string, logger *log.Logger) *MetricExtractor {
	return &MetricExtractor{
		prometheusURL: prometheusURL,
		logger:        logger,
	}
}

// GetAvailableMetrics queries Prometheus for all available metrics
func (me *MetricExtractor) GetAvailableMetrics(ctx context.Context) ([]MetricInfo, error) {
	// In real implementation, this would call: GET /api/v1/label/__name__/values
	// For now, return simulated metrics

	commonMetrics := []string{
		"http_requests_total",
		"http_request_duration_seconds",
		"http_requests_in_progress",
		"database_connection_pool_active_connections",
		"database_pool_connections_open_total",
		"database_queries_total",
		"database_query_duration_seconds",
		"cache_hits_total",
		"cache_misses_total",
		"worker_pool_active_workers",
		"worker_pool_queue_size",
		"grpc_requests_total",
		"grpc_request_duration_seconds",
		"kafka_messages_produced_total",
		"kafka_messages_consumed_total",
		"kubernetes_pod_cpu_cores_used",
		"kubernetes_pod_memory_bytes_used",
		"kubernetes_pod_network_in_bytes",
		"kubernetes_pod_network_out_bytes",
	}

	var metrics []MetricInfo
	for _, name := range commonMetrics {
		metricType := me.inferMetricType(name)
		mi := MetricInfo{
			MetricName:    name,
			MetricType:    metricType,
			Labels:        mapMetricLabels(name),
			DiscoveryTime: time.Now(),
			Cardinality:   me.estimateMetricCardinality(name),
		}
		metrics = append(metrics, mi)
	}

	return metrics, nil
}

// extractRateMetrics identifies rate-based features from metrics
func (me *MetricExtractor) extractRateMetrics(metricName string) string {
	if strings.Contains(strings.ToLower(metricName), "total") ||
		strings.Contains(strings.ToLower(metricName), "count") {
		return "rate"
	}
	return "gauge"
}

// extractLatencyPercentiles derives latency percentile features
func (me *MetricExtractor) extractLatencyPercentiles(metricName string) []string {
	percentiles := []string{}

	if strings.Contains(strings.ToLower(metricName), "duration") ||
		strings.Contains(strings.ToLower(metricName), "latency") {
		percentiles = []string{"p50", "p95", "p99", "p99.9"}
	}

	return percentiles
}

// inferMetricType infers metric type from name
func (me *MetricExtractor) inferMetricType(metricName string) string {
	lower := strings.ToLower(metricName)

	if strings.Contains(lower, "total") || strings.Contains(lower, "count") {
		return "counter"
	}
	if strings.Contains(lower, "duration") || strings.Contains(lower, "latency") {
		return "histogram"
	}
	if strings.Contains(lower, "in_progress") || strings.Contains(lower, "active") {
		return "gauge"
	}

	return "gauge" // default
}

// mapMetricLabels returns typical labels for a metric
func mapMetricLabels(metricName string) map[string][]string {
	labels := make(map[string][]string)
	lower := strings.ToLower(metricName)

	// HTTP metrics
	if strings.Contains(lower, "http") {
		labels["method"] = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}
		labels["status"] = []string{"200", "400", "404", "500", "503"}
		labels["endpoint"] = []string{"/api/v1/users", "/api/v1/incidents", "/api/v1/actions"}
	}

	// Database metrics
	if strings.Contains(lower, "database") || strings.Contains(lower, "db_") {
		labels["database"] = []string{"semlayer", "analytics", "cache"}
		labels["operation"] = []string{"SELECT", "INSERT", "UPDATE", "DELETE"}
	}

	// Worker metrics
	if strings.Contains(lower, "worker") {
		labels["worker_type"] = []string{"incident_detector", "action_executor", "feature_compute"}
		labels["pool_name"] = []string{"primary", "secondary"}
	}

	// Kubernetes metrics
	if strings.Contains(lower, "kubernetes") || strings.Contains(lower, "pod_") {
		labels["namespace"] = []string{"default", "semlayer", "kube-system"}
		labels["pod"] = []string{"incident-detector-0", "action-executor-0"}
		labels["container"] = []string{"main", "sidecar"}
	}

	// Common labels
	labels["le"] = []string{".005", ".01", ".025", ".05", ".1", ".25", ".5", "1", "2.5", "5", "10"}

	return labels
}

// estimateMetricCardinality estimates the number of time series for a metric
func (me *MetricExtractor) estimateMetricCardinality(metricName string) int64 {
	lower := strings.ToLower(metricName)

	// Rough cardinality estimates
	if strings.Contains(lower, "http") {
		return 5 * 5 * 5 // method * status * endpoint = 125
	}
	if strings.Contains(lower, "database") {
		return 3 * 4 // database * operation = 12
	}
	if strings.Contains(lower, "worker") {
		return 3 * 2 // worker_type * pool = 6
	}
	if strings.Contains(lower, "pod_") {
		return 5 * 2 * 2 // pod * container * resource = 20
	}

	return 1 // Single time series
}

// DeriveCompositeFeatures creates derived features from metrics
func (me *MetricExtractor) DeriveCompositeFeatures(metrics []MetricInfo) []string {
	derived := []string{}

	// Error rate = error_requests / total_requests
	if me.hasMetric(metrics, "http_requests_total") {
		derived = append(derived, "error_rate_http", "success_rate_http")
	}

	// P99 Latency
	if me.hasMetric(metrics, "http_request_duration_seconds") {
		derived = append(derived, "p99_latency_seconds", "p95_latency_seconds", "avg_latency_seconds")
	}

	// DB connection utilization = active / pool_size
	if me.hasMetric(metrics, "database_connection_pool_active_connections") {
		derived = append(derived, "db_connection_utilization_pct")
	}

	// Cache hit rate
	if me.hasMetric(metrics, "cache_hits_total") && me.hasMetric(metrics, "cache_misses_total") {
		derived = append(derived, "cache_hit_rate")
	}

	// CPU utilization %
	if me.hasMetric(metrics, "kubernetes_pod_cpu_cores_used") {
		derived = append(derived, "pod_cpu_utilization_pct")
	}

	// Memory utilization %
	if me.hasMetric(metrics, "kubernetes_pod_memory_bytes_used") {
		derived = append(derived, "pod_memory_utilization_pct")
	}

	return derived
}

// hasMetric checks if a metric exists in the list
func (me *MetricExtractor) hasMetric(metrics []MetricInfo, name string) bool {
	for _, m := range metrics {
		if m.MetricName == name {
			return true
		}
	}
	return false
}

// ConvertToFeatureCandidates converts metrics to feature candidates
func (me *MetricExtractor) ConvertToFeatureCandidates(metrics []MetricInfo, derived []string) []models.FeatureCandidate {
	candidates := make([]models.FeatureCandidate, 0, len(metrics)+len(derived))

	// Base metrics
	for _, metric := range metrics {
		// Create one candidate per label combination (sampled)
		cardinalityScore := 1.0
		if metric.Cardinality > 1000 {
			cardinalityScore = 0.5 // High cardinality is less useful
		}

		candidates = append(candidates, models.FeatureCandidate{
			Name:           metric.MetricName,
			SourceDatabase: "prometheus",
			SourceField:    metric.MetricName,
			DataType:       "float",
			Completeness:   0.99, // Metrics are usually very complete
			Cardinality:    metric.Cardinality,
			BusinessValue:  0,
			TechnicalScore: cardinalityScore,
			DiscoveredAt:   metric.DiscoveryTime,
			Status:         "candidate",
		})

		// Create candidates for each label value
		for labelName, labelValues := range metric.Labels {
			candidates = append(candidates, models.FeatureCandidate{
				Name:           fmt.Sprintf("%s_by_%s", metric.MetricName, labelName),
				SourceDatabase: "prometheus",
				SourceField:    labelName,
				DataType:       "categorical",
				Completeness:   0.95,
				Cardinality:    int64(len(labelValues)),
				BusinessValue:  0,
				TechnicalScore: 0.8,
				DiscoveredAt:   metric.DiscoveryTime,
				Status:         "candidate",
			})
		}
	}

	// Derived features
	for _, derivedName := range derived {
		candidates = append(candidates, models.FeatureCandidate{
			Name:           derivedName,
			SourceDatabase: "prometheus_derived",
			SourceField:    derivedName,
			DataType:       "float",
			Completeness:   0.98,
			Cardinality:    -1,
			BusinessValue:  0.8, // Derived features often have higher business value
			TechnicalScore: 0.9,
			DiscoveredAt:   time.Now(),
			Status:         "candidate",
		})
	}

	return candidates
}

// GetMetricAnomalies identifies metrics with unusual patterns (useful for drift detection)
func (me *MetricExtractor) GetMetricAnomalies(ctx context.Context) (map[string]interface{}, error) {
	anomalies := make(map[string]interface{})

	// In real implementation, would query Prometheus /query_range
	// and identify metrics with sudden changes

	return anomalies, nil
}
