package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// MetricsCollector holds all Prometheus metrics for the calendar service
type MetricsCollector struct {
	// Cache metrics
	CacheHits      prometheus.Counter
	CacheMisses    prometheus.Counter
	CacheEvictions prometheus.Counter
	CacheSize      prometheus.Gauge
	CacheHitRate   prometheus.Gauge

	// Query metrics
	QueryDuration   prometheus.Histogram
	QueryErrors     prometheus.Counter
	QueriesInFlight prometheus.Gauge

	// Profile resolution metrics
	ProfileResolutions prometheus.Counter
	ResolutionDuration prometheus.Histogram
	ResolutionErrors   prometheus.Counter

	// Holiday metrics
	HolidayCount  prometheus.Gauge
	BlackoutCount prometheus.Gauge
	ProfileCount  prometheus.Gauge

	// HTTP metrics
	RequestDuration  prometheus.Histogram
	RequestErrors    prometheus.Counter
	RequestsInFlight prometheus.Gauge

	// RRULE expansion metrics
	RRuleExpansions   prometheus.Counter
	RRuleErrors       prometheus.Counter
	ExpansionDuration prometheus.Histogram

	// CDC metrics
	CDCEventsProcessed *prometheus.CounterVec
}

// NewMetricsCollector creates and registers all metrics
func NewMetricsCollector(namespace, subsystem string) *MetricsCollector {
	return &MetricsCollector{
		// Cache metrics
		CacheHits: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_hits_total",
			Help:      "Total number of cache hits",
		}),
		CacheMisses: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_misses_total",
			Help:      "Total number of cache misses",
		}),
		CacheEvictions: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_evictions_total",
			Help:      "Total number of cache evictions",
		}),
		CacheSize: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_size_bytes",
			Help:      "Current cache size in bytes",
		}),
		CacheHitRate: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cache_hit_rate",
			Help:      "Cache hit rate (0-1)",
		}),

		// Query metrics
		QueryDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "query_duration_seconds",
			Help:      "Query execution duration in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		}),
		QueryErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "query_errors_total",
			Help:      "Total number of query errors",
		}),
		QueriesInFlight: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "queries_in_flight",
			Help:      "Number of queries currently in flight",
		}),

		// Profile resolution metrics
		ProfileResolutions: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "profile_resolutions_total",
			Help:      "Total number of profile resolutions",
		}),
		ResolutionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "resolution_duration_seconds",
			Help:      "Profile resolution duration in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		}),
		ResolutionErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "resolution_errors_total",
			Help:      "Total number of resolution errors",
		}),

		// Holiday metrics
		HolidayCount: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "holidays_total",
			Help:      "Total number of holidays in system",
		}),
		BlackoutCount: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "blackouts_total",
			Help:      "Total number of blackouts in system",
		}),
		ProfileCount: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "profiles_total",
			Help:      "Total number of profiles in system",
		}),

		// HTTP metrics
		RequestDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		}),
		RequestErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "request_errors_total",
			Help:      "Total number of HTTP request errors",
		}),
		RequestsInFlight: promauto.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "requests_in_flight",
			Help:      "Number of requests currently in flight",
		}),

		// RRULE expansion metrics
		RRuleExpansions: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "rrule_expansions_total",
			Help:      "Total number of RRULE expansions",
		}),
		RRuleErrors: promauto.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "rrule_errors_total",
			Help:      "Total number of RRULE expansion errors",
		}),
		ExpansionDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "expansion_duration_seconds",
			Help:      "RRULE expansion duration in seconds",
			Buckets:   []float64{.001, .005, .01, .025, .05, .1, .25, .5, 1},
		}),

		// CDC metrics
		CDCEventsProcessed: promauto.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "cdc_events_processed_total",
			Help:      "Total number of CDC events processed",
		}, []string{"table", "operation"}),
	}
}

// RecordCacheHit records a cache hit
func (m *MetricsCollector) RecordCacheHit() {
	m.CacheHits.Inc()
	m.updateCacheHitRate()
}

// RecordCacheMiss records a cache miss
func (m *MetricsCollector) RecordCacheMiss() {
	m.CacheMisses.Inc()
	m.updateCacheHitRate()
}

// updateCacheHitRate calculates and updates cache hit rate
func (m *MetricsCollector) updateCacheHitRate() {
	// In real implementation, would use Prometheus query to get metric values
	// For now, this is a placeholder
}

// RecordQueryDuration records query execution time
func (m *MetricsCollector) RecordQueryDuration(duration float64) {
	m.QueryDuration.Observe(duration)
}

// RecordResolutionDuration records profile resolution time
func (m *MetricsCollector) RecordResolutionDuration(duration float64) {
	m.ResolutionDuration.Observe(duration)
	m.ProfileResolutions.Inc()
}

// RecordExpansionDuration records RRULE expansion time
func (m *MetricsCollector) RecordExpansionDuration(duration float64) {
	m.ExpansionDuration.Observe(duration)
	m.RRuleExpansions.Inc()
}

// RecordProfileResolution records a profile resolution
func (m *MetricsCollector) RecordProfileResolution() {
	m.ProfileResolutions.Inc()
}

// RecordResolutionError records a profile resolution error
func (m *MetricsCollector) RecordResolutionError() {
	m.ResolutionErrors.Inc()
}

// RecordCDCEvent records a CDC event processing metric
func (m *MetricsCollector) RecordCDCEvent(table, operation string) {
	m.CDCEventsProcessed.WithLabelValues(table, operation).Inc()
}
