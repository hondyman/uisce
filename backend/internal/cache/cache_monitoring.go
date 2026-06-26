package cache

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// ============================================================================
// Cache Monitoring & Observability
// ============================================================================

// CacheMonitor tracks cache hit/miss ratios and performance metrics
type CacheMonitor struct {
	cache           *SemanticQueryCache
	mu              sync.RWMutex
	metricsSnapshot *CacheMetricsSnapshot
	updateInterval  time.Duration
	stopChan        chan struct{}
	lastSnapshot    time.Time
	snapshotHistory []CacheMetricsSnapshot // Keep last 100 snapshots
	maxHistorySize  int
}

// CacheMetricsSnapshot represents a point-in-time cache metrics snapshot
type CacheMetricsSnapshot struct {
	Timestamp          time.Time
	NLQueryHits        int64
	NLQueryMisses      int64
	SQLQueryHits       int64
	SQLQueryMisses     int64
	ResultsHits        int64
	ResultsMisses      int64
	AvoidsHits         int64
	TotalSavingsMs     int64
	AvgLatencySavedMs  float64 // Per cache hit
	EstimatedCostSaved float64 // Dollar amount
}

// NewCacheMonitor creates a new cache monitor
func NewCacheMonitor(cache *SemanticQueryCache, updateInterval time.Duration) *CacheMonitor {
	return &CacheMonitor{
		cache:           cache,
		updateInterval:  updateInterval,
		stopChan:        make(chan struct{}),
		maxHistorySize:  100,
		snapshotHistory: []CacheMetricsSnapshot{},
	}
}

// Start begins monitoring the cache
func (cm *CacheMonitor) Start() {
	go func() {
		ticker := time.NewTicker(cm.updateInterval)
		defer ticker.Stop()

		for {
			select {
			case <-cm.stopChan:
				log.Printf("Cache monitor stopped")
				return
			case <-ticker.C:
				cm.snapshotMetrics()
			}
		}
	}()

	log.Printf("Cache monitor started (interval: %v)", cm.updateInterval)
}

// Stop stops the cache monitor
func (cm *CacheMonitor) Stop() {
	close(cm.stopChan)
}

// snapshotMetrics captures current metrics
func (cm *CacheMonitor) snapshotMetrics() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	metrics := cm.cache.GetMetrics()

	// Calculate derived metrics
	totalHits := metrics.NLQueryHits + metrics.SQLQueryHits + metrics.ResultsHits
	totalMisses := metrics.NLQueryMisses + metrics.SQLQueryMisses + metrics.ResultsMisses
	totalOps := totalHits + totalMisses

	avgLatencySaved := float64(0)
	if totalHits > 0 {
		avgLatencySaved = float64(metrics.TotalSavings.Milliseconds()) / float64(totalHits)
	}

	// Estimate cost savings ($0.0075 per LLM call avoided)
	estimatedCost := float64(metrics.AvoidsHits) * 0.0075

	snapshot := CacheMetricsSnapshot{
		Timestamp:          time.Now(),
		NLQueryHits:        metrics.NLQueryHits,
		NLQueryMisses:      metrics.NLQueryMisses,
		SQLQueryHits:       metrics.SQLQueryHits,
		SQLQueryMisses:     metrics.SQLQueryMisses,
		ResultsHits:        metrics.ResultsHits,
		ResultsMisses:      metrics.ResultsMisses,
		AvoidsHits:         metrics.AvoidsHits,
		TotalSavingsMs:     metrics.TotalSavings.Milliseconds(),
		AvgLatencySavedMs:  avgLatencySaved,
		EstimatedCostSaved: estimatedCost,
	}

	cm.metricsSnapshot = &snapshot

	// Maintain history
	cm.snapshotHistory = append(cm.snapshotHistory, snapshot)
	if len(cm.snapshotHistory) > cm.maxHistorySize {
		cm.snapshotHistory = cm.snapshotHistory[1:]
	}

	// Log significant metrics
	if totalOps > 0 {
		nlHitRate := float64(metrics.NLQueryHits) / float64(metrics.NLQueryHits+metrics.NLQueryMisses)
		sqlHitRate := float64(metrics.SQLQueryHits) / float64(metrics.SQLQueryHits+metrics.SQLQueryMisses)
		resultsHitRate := float64(metrics.ResultsHits) / float64(metrics.ResultsHits+metrics.ResultsMisses)

		log.Printf("Cache Metrics [%s]: NL=%.1f%% SQL=%.1f%% Results=%.1f%% | Avoided=%d | Saved=%dms (~$%.2f) | AvgLatency=%.1fms",
			time.Now().Format("15:04:05"),
			nlHitRate*100, sqlHitRate*100, resultsHitRate*100,
			metrics.AvoidsHits,
			metrics.TotalSavings.Milliseconds(),
			estimatedCost,
			avgLatencySaved,
		)
	}
}

// GetSnapshot returns the latest metrics snapshot
func (cm *CacheMonitor) GetSnapshot() *CacheMetricsSnapshot {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return cm.metricsSnapshot
}

// GetHistory returns the metrics history
func (cm *CacheMonitor) GetHistory(limit int) []CacheMetricsSnapshot {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if limit > len(cm.snapshotHistory) {
		limit = len(cm.snapshotHistory)
	}

	result := make([]CacheMetricsSnapshot, limit)
	copy(result, cm.snapshotHistory[len(cm.snapshotHistory)-limit:])

	return result
}

// GetPerformanceReport generates a detailed performance report
func (cm *CacheMonitor) GetPerformanceReport() map[string]interface{} {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.metricsSnapshot == nil {
		return map[string]interface{}{
			"status": "no_data",
		}
	}

	s := *cm.metricsSnapshot

	totalHits := s.NLQueryHits + s.SQLQueryHits + s.ResultsHits
	totalMisses := s.NLQueryMisses + s.SQLQueryMisses + s.ResultsMisses
	totalOps := totalHits + totalMisses

	var nlHitRate, sqlHitRate, resultsHitRate, overallHitRate float64
	if s.NLQueryHits+s.NLQueryMisses > 0 {
		nlHitRate = float64(s.NLQueryHits) / float64(s.NLQueryHits+s.NLQueryMisses)
	}
	if s.SQLQueryHits+s.SQLQueryMisses > 0 {
		sqlHitRate = float64(s.SQLQueryHits) / float64(s.SQLQueryHits+s.SQLQueryMisses)
	}
	if s.ResultsHits+s.ResultsMisses > 0 {
		resultsHitRate = float64(s.ResultsHits) / float64(s.ResultsHits+s.ResultsMisses)
	}
	if totalOps > 0 {
		overallHitRate = float64(totalHits) / float64(totalOps)
	}

	return map[string]interface{}{
		"timestamp":        s.Timestamp.String(),
		"total_operations": totalOps,
		"total_hits":       totalHits,
		"total_misses":     totalMisses,
		"overall_hit_rate": fmt.Sprintf("%.2f%%", overallHitRate*100),
		"layer_1_nl_query": map[string]interface{}{
			"hits":     s.NLQueryHits,
			"misses":   s.NLQueryMisses,
			"hit_rate": fmt.Sprintf("%.2f%%", nlHitRate*100),
		},
		"layer_2_sql_query": map[string]interface{}{
			"hits":     s.SQLQueryHits,
			"misses":   s.SQLQueryMisses,
			"hit_rate": fmt.Sprintf("%.2f%%", sqlHitRate*100),
		},
		"layer_3_results": map[string]interface{}{
			"hits":     s.ResultsHits,
			"misses":   s.ResultsMisses,
			"hit_rate": fmt.Sprintf("%.2f%%", resultsHitRate*100),
		},
		"performance": map[string]interface{}{
			"llm_calls_avoided":    s.AvoidsHits,
			"total_savings_ms":     s.TotalSavingsMs,
			"total_savings_sec":    float64(s.TotalSavingsMs) / 1000,
			"avg_latency_saved_ms": fmt.Sprintf("%.1f", s.AvgLatencySavedMs),
			"estimated_cost_saved": fmt.Sprintf("$%.2f", s.EstimatedCostSaved),
		},
		"projections": map[string]interface{}{
			"daily_cost_savings":   fmt.Sprintf("$%.2f", s.EstimatedCostSaved*24), // Very rough estimate
			"monthly_cost_savings": fmt.Sprintf("$%.2f", s.EstimatedCostSaved*24*30),
			"daily_latency_saved":  fmt.Sprintf("%.1f hours", float64(s.TotalSavingsMs)*24/3600/1000),
		},
	}
}

// ============================================================================
// Prometheus Metrics Export
// ============================================================================

// PrometheusMetrics generates metrics in Prometheus format
func (cm *CacheMonitor) PrometheusMetrics() string {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if cm.metricsSnapshot == nil {
		return ""
	}

	s := *cm.metricsSnapshot

	// Format as Prometheus text-based metrics
	metrics := fmt.Sprintf(`# HELP semlayer_cache_nl_query_hits Total NL query cache hits
# TYPE semlayer_cache_nl_query_hits counter
semlayer_cache_nl_query_hits %d

# HELP semlayer_cache_nl_query_misses Total NL query cache misses
# TYPE semlayer_cache_nl_query_misses counter
semlayer_cache_nl_query_misses %d

# HELP semlayer_cache_sql_query_hits Total SQL query cache hits
# TYPE semlayer_cache_sql_query_hits counter
semlayer_cache_sql_query_hits %d

# HELP semlayer_cache_sql_query_misses Total SQL query cache misses
# TYPE semlayer_cache_sql_query_misses counter
semlayer_cache_sql_query_misses %d

# HELP semlayer_cache_results_hits Total results cache hits
# TYPE semlayer_cache_results_hits counter
semlayer_cache_results_hits %d

# HELP semlayer_cache_results_misses Total results cache misses
# TYPE semlayer_cache_results_misses counter
semlayer_cache_results_misses %d

# HELP semlayer_cache_llm_calls_avoided Total LLM calls avoided via caching
# TYPE semlayer_cache_llm_calls_avoided counter
semlayer_cache_llm_calls_avoided %d

# HELP semlayer_cache_total_savings_ms Total latency saved in milliseconds
# TYPE semlayer_cache_total_savings_ms counter
semlayer_cache_total_savings_ms %d

# HELP semlayer_cache_estimated_cost_saved Estimated cost savings in USD
# TYPE semlayer_cache_estimated_cost_saved gauge
semlayer_cache_estimated_cost_saved %.2f

# HELP semlayer_cache_avg_latency_saved_ms Average latency saved per hit in ms
# TYPE semlayer_cache_avg_latency_saved_ms gauge
semlayer_cache_avg_latency_saved_ms %.2f
`,
		s.NLQueryHits,
		s.NLQueryMisses,
		s.SQLQueryHits,
		s.SQLQueryMisses,
		s.ResultsHits,
		s.ResultsMisses,
		s.AvoidsHits,
		s.TotalSavingsMs,
		s.EstimatedCostSaved,
		s.AvgLatencySavedMs,
	)

	return metrics
}

// ============================================================================
// Alert Threshold Configuration
// ============================================================================

// AlertThreshold defines when cache performance alerts should trigger
type AlertThreshold struct {
	MinHitRate       float64       // e.g., 0.5 = 50%
	MaxCacheMissRate float64       // e.g., 0.5 = 50%
	MinSavingsPerHit int64         // milliseconds
	CheckInterval    time.Duration // How often to check thresholds
}

// DefaultAlertThresholds returns recommended alert thresholds
func DefaultAlertThresholds() *AlertThreshold {
	return &AlertThreshold{
		MinHitRate:       0.4, // Alert if < 40% hit rate
		MaxCacheMissRate: 0.6, // Alert if > 60% miss rate
		MinSavingsPerHit: 50,  // Alert if < 50ms saved per hit
		CheckInterval:    1 * time.Minute,
	}
}

// AlertChecker monitors cache metrics against thresholds
type AlertChecker struct {
	monitor    *CacheMonitor
	thresholds *AlertThreshold
	alerts     []AlertEvent
	mu         sync.RWMutex
	stopChan   chan struct{}
}

// AlertEvent represents a triggered alert
type AlertEvent struct {
	Timestamp time.Time
	Severity  string // "warning", "critical"
	Message   string
}

// NewAlertChecker creates a new alert checker
func NewAlertChecker(monitor *CacheMonitor, thresholds *AlertThreshold) *AlertChecker {
	if thresholds == nil {
		thresholds = DefaultAlertThresholds()
	}

	return &AlertChecker{
		monitor:    monitor,
		thresholds: thresholds,
		alerts:     []AlertEvent{},
		stopChan:   make(chan struct{}),
	}
}

// Start begins checking alert thresholds
func (ac *AlertChecker) Start() {
	go func() {
		ticker := time.NewTicker(ac.thresholds.CheckInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ac.stopChan:
				log.Printf("Alert checker stopped")
				return
			case <-ticker.C:
				ac.checkThresholds()
			}
		}
	}()

	log.Printf("Alert checker started")
}

// Stop stops the alert checker
func (ac *AlertChecker) Stop() {
	close(ac.stopChan)
}

// checkThresholds evaluates current metrics against thresholds
func (ac *AlertChecker) checkThresholds() {
	snapshot := ac.monitor.GetSnapshot()
	if snapshot == nil {
		return
	}

	totalHits := snapshot.NLQueryHits + snapshot.SQLQueryHits + snapshot.ResultsHits
	totalMisses := snapshot.NLQueryMisses + snapshot.SQLQueryMisses + snapshot.ResultsMisses
	totalOps := totalHits + totalMisses

	if totalOps == 0 {
		return
	}

	hitRate := float64(totalHits) / float64(totalOps)

	// Check hit rate threshold
	if hitRate < ac.thresholds.MinHitRate {
		event := AlertEvent{
			Timestamp: time.Now(),
			Severity:  "warning",
			Message:   fmt.Sprintf("Cache hit rate low: %.1f%% (threshold: %.1f%%)", hitRate*100, ac.thresholds.MinHitRate*100),
		}
		ac.recordAlert(event)
		log.Printf("ALERT: %s", event.Message)
	}

	// Check average savings per hit
	if totalHits > 0 && snapshot.TotalSavingsMs > 0 {
		avgSavings := snapshot.TotalSavingsMs / totalHits
		if avgSavings < ac.thresholds.MinSavingsPerHit {
			event := AlertEvent{
				Timestamp: time.Now(),
				Severity:  "warning",
				Message:   fmt.Sprintf("Low cache benefit: %.0fms saved per hit (threshold: %dms)", float64(avgSavings), ac.thresholds.MinSavingsPerHit),
			}
			ac.recordAlert(event)
			log.Printf("ALERT: %s", event.Message)
		}
	}
}

// recordAlert records an alert event
func (ac *AlertChecker) recordAlert(event AlertEvent) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	ac.alerts = append(ac.alerts, event)

	// Keep last 100 alerts
	if len(ac.alerts) > 100 {
		ac.alerts = ac.alerts[1:]
	}
}

// GetAlerts returns recent alerts
func (ac *AlertChecker) GetAlerts(limit int) []AlertEvent {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	if limit > len(ac.alerts) {
		limit = len(ac.alerts)
	}

	result := make([]AlertEvent, limit)
	copy(result, ac.alerts[len(ac.alerts)-limit:])

	return result
}

// ============================================================================
// Endpoint for Monitoring Dashboard
// ============================================================================

// GetMonitoringEndpointHandler returns handler data for monitoring dashboard
// This would be used in an HTTP handler like: GET /api/admin/cache-metrics
func (cm *CacheMonitor) GetMonitoringEndpointHandler() map[string]interface{} {
	return map[string]interface{}{
		"report":     cm.GetPerformanceReport(),
		"history":    cm.GetHistory(20),
		"prometheus": cm.PrometheusMetrics(),
	}
}
