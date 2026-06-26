package services

import (
	"context"
	"expvar"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Import for pprof endpoints
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// PerformanceMonitor provides real-time performance monitoring and metrics
type PerformanceMonitor struct {
	// Request metrics
	totalRequests   int64
	activeRequests  int64
	requestDuration int64 // nanoseconds
	errorCount      int64

	// Cache metrics
	cacheHits          int64
	cacheMisses        int64
	cacheEvictions     int64
	cacheSize          int64
	cacheInflight      int64
	cacheInvalidations int64
	cacheEntrySize     int64 // bytes

	// QoS metrics
	qosTokenDenials int64
	qosBreakerTrips int64
	qosLoadShed     int64
	qosCircuitOpen  int64

	// Invalidation lag tracking
	invalidationLag   int64 // nanoseconds
	invalidationCount int64

	// GC metrics
	gcCycles    int64
	gcPauseTime int64 // nanoseconds

	// Concurrency metrics
	tokenBucketSize int64
	tokenBucketUsed int64

	// Audit metrics
	auditEventsQueued    int64
	auditEventsProcessed int64
	auditEventsDropped   int64

	// Performance targets (in milliseconds)
	targets struct {
		p50 time.Duration
		p95 time.Duration
		p99 time.Duration
	}

	// Per-tenant tracking with enhanced metrics
	tenantMetrics map[string]*TenantMetrics
	metricsMux    sync.RWMutex

	// Latency histograms for percentiles
	latencyHistogram *LatencyHistogram

	startTime time.Time
}

// LatencyHistogram tracks latency distributions for percentile calculations
type LatencyHistogram struct {
	buckets    map[string][]time.Duration // tenant -> latencies
	mu         sync.RWMutex
	maxSamples int
}

// TenantMetrics tracks performance metrics per tenant
type TenantMetrics struct {
	tenantID     string
	requestCount int64
	errorCount   int64
	totalLatency time.Duration
	latencies    []time.Duration // For percentile calculation
	lastReset    time.Time

	// Cache metrics per tenant
	cacheHits    int64
	cacheMisses  int64
	cacheHitRate float64

	// QoS metrics per tenant
	qosDenials      int64
	qosBreakerTrips int64

	// Invalidation metrics
	invalidationLag   time.Duration
	invalidationCount int64

	mu sync.RWMutex
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor() *PerformanceMonitor {
	pm := &PerformanceMonitor{
		startTime:     time.Now(),
		tenantMetrics: make(map[string]*TenantMetrics),
		latencyHistogram: &LatencyHistogram{
			buckets:    make(map[string][]time.Duration),
			maxSamples: 10000, // Keep last 10k samples per tenant
		},
	}

	// Set performance targets
	pm.targets.p50 = 2 * time.Millisecond
	pm.targets.p95 = 6 * time.Millisecond
	pm.targets.p99 = 12 * time.Millisecond

	// Publish metrics to expvar
	expvar.Publish("semlayer_requests_total", expvar.Func(func() interface{} { return atomic.LoadInt64(&pm.totalRequests) }))
	expvar.Publish("semlayer_requests_active", expvar.Func(func() interface{} { return atomic.LoadInt64(&pm.activeRequests) }))
	expvar.Publish("semlayer_errors_total", expvar.Func(func() interface{} { return atomic.LoadInt64(&pm.errorCount) }))
	expvar.Publish("semlayer_cache_hits", expvar.Func(func() interface{} { return atomic.LoadInt64(&pm.cacheHits) }))
	expvar.Publish("semlayer_cache_misses", expvar.Func(func() interface{} { return atomic.LoadInt64(&pm.cacheMisses) }))
	expvar.Publish("semlayer_cache_evictions", expvar.Func(func() interface{} { return atomic.LoadInt64(&pm.cacheEvictions) }))
	expvar.Publish("semlayer_cache_size", expvar.Func(func() interface{} { return atomic.LoadInt64(&pm.cacheSize) }))
	expvar.Publish("semlayer_qps_denials", expvar.Func(func() interface{} { return atomic.LoadInt64(&pm.qosTokenDenials) }))
	expvar.Publish("semlayer_breaker_trips", expvar.Func(func() interface{} { return atomic.LoadInt64(&pm.qosBreakerTrips) }))
	expvar.Publish("semlayer_gc_cycles", expvar.Func(func() interface{} { return atomic.LoadInt64(&pm.gcCycles) }))

	return pm
}

// StartMonitoring begins continuous monitoring
func (pm *PerformanceMonitor) StartMonitoring(ctx context.Context) {
	// Start GC monitoring
	go pm.monitorGC(ctx)

	// Start periodic stats logging
	go pm.logStats(ctx)

	logging.GetLogger().Sugar().Info("Performance monitoring started")
}

// RecordRequest records a request for monitoring
func (pm *PerformanceMonitor) RecordRequest(duration time.Duration, hadError bool) {
	atomic.AddInt64(&pm.totalRequests, 1)
	atomic.AddInt64(&pm.requestDuration, duration.Nanoseconds())

	if hadError {
		atomic.AddInt64(&pm.errorCount, 1)
	}
}

// RecordActiveRequest increments/decrements active request count
func (pm *PerformanceMonitor) RecordActiveRequest(delta int64) {
	atomic.AddInt64(&pm.activeRequests, delta)
}

// RecordCacheHit records a cache hit
func (pm *PerformanceMonitor) RecordCacheHit() {
	atomic.AddInt64(&pm.cacheHits, 1)
}

// RecordCacheMiss records a cache miss
func (pm *PerformanceMonitor) RecordCacheMiss() {
	atomic.AddInt64(&pm.cacheMisses, 1)
}

// RecordCacheEviction records a cache eviction
func (pm *PerformanceMonitor) RecordCacheEviction() {
	atomic.AddInt64(&pm.cacheEvictions, 1)
}

// RecordCacheSize records current cache size
func (pm *PerformanceMonitor) RecordCacheSize(size int64) {
	atomic.StoreInt64(&pm.cacheSize, size)
}

// RecordCacheInflight records number of in-flight cache refreshes
func (pm *PerformanceMonitor) RecordCacheInflight(delta int64) {
	atomic.AddInt64(&pm.cacheInflight, delta)
}

// RecordCacheInvalidation records cache invalidation events
func (pm *PerformanceMonitor) RecordCacheInvalidation() {
	atomic.AddInt64(&pm.cacheInvalidations, 1)
}

// RecordQoSTokenDenial records token bucket denials
func (pm *PerformanceMonitor) RecordQoSTokenDenial() {
	atomic.AddInt64(&pm.qosTokenDenials, 1)
}

// RecordQoSBreakerTrip records circuit breaker trips
func (pm *PerformanceMonitor) RecordQoSBreakerTrip() {
	atomic.AddInt64(&pm.qosBreakerTrips, 1)
}

// RecordQoSLoadShed records load shedding events
func (pm *PerformanceMonitor) RecordQoSLoadShed() {
	atomic.AddInt64(&pm.qosLoadShed, 1)
}

// RecordQoSCircuitOpen records when circuit breaker opens
func (pm *PerformanceMonitor) RecordQoSCircuitOpen() {
	atomic.AddInt64(&pm.qosCircuitOpen, 1)
}

// RecordInvalidationLag records the lag between invalidation trigger and cache update
func (pm *PerformanceMonitor) RecordInvalidationLag(lag time.Duration) {
	atomic.AddInt64(&pm.invalidationLag, lag.Nanoseconds())
	atomic.AddInt64(&pm.invalidationCount, 1)
}

// RecordTenantCacheHit records cache hit for specific tenant
func (pm *PerformanceMonitor) RecordTenantCacheHit(tenantID string) {
	atomic.AddInt64(&pm.cacheHits, 1)
	tm := pm.getTenantMetrics(tenantID)
	atomic.AddInt64(&tm.cacheHits, 1)
	pm.updateCacheHitRate(tm)
}

// RecordTenantCacheMiss records cache miss for specific tenant
func (pm *PerformanceMonitor) RecordTenantCacheMiss(tenantID string) {
	atomic.AddInt64(&pm.cacheMisses, 1)
	tm := pm.getTenantMetrics(tenantID)
	atomic.AddInt64(&tm.cacheMisses, 1)
	pm.updateCacheHitRate(tm)
}

// RecordTenantQoSDenial records QoS denial for specific tenant
func (pm *PerformanceMonitor) RecordTenantQoSDenial(tenantID string) {
	atomic.AddInt64(&pm.qosTokenDenials, 1)
	tm := pm.getTenantMetrics(tenantID)
	atomic.AddInt64(&tm.qosDenials, 1)
}

// GetProfileResults is a no-op compatibility method so PerformanceMonitor can
// be passed where an httpapi.ProfilerService is expected by the HTTP API
// bootstrap (the real profiler implementation lives in the profiler package).
func (pm *PerformanceMonitor) GetProfileResults(jobID string) (interface{}, error) {
	// Not implemented by the performance monitor; return nil to satisfy the
	// interface required during server wiring in tests.
	return nil, nil
}

// GetProfileStatus is a no-op compatibility method so PerformanceMonitor can
// be passed where an httpapi.ProfilerService is expected by the HTTP API
// bootstrap (the real profiler implementation lives in the profiler package).
// It returns nil, nil to avoid importing the httpapi package and causing an
// import cycle during compilation.
func (pm *PerformanceMonitor) GetProfileStatus(jobID string) (interface{}, error) {
	return nil, nil
}

// StartProfile is a no-op compatibility method so PerformanceMonitor can be
// passed where an httpapi.ProfilerService is expected by the HTTP API
// bootstrap. The real profiler implementation lives in the profiler package.
func (pm *PerformanceMonitor) StartProfile(ctx context.Context, job interface{}) error {
	// No-op for compatibility
	return nil
}

// RecordTenantQoSBreakerTrip records circuit breaker trip for specific tenant
func (pm *PerformanceMonitor) RecordTenantQoSBreakerTrip(tenantID string) {
	atomic.AddInt64(&pm.qosBreakerTrips, 1)
	tm := pm.getTenantMetrics(tenantID)
	atomic.AddInt64(&tm.qosBreakerTrips, 1)
}

// RecordTenantInvalidationLag records invalidation lag for specific tenant
func (pm *PerformanceMonitor) RecordTenantInvalidationLag(tenantID string, lag time.Duration) {
	atomic.AddInt64(&pm.invalidationLag, lag.Nanoseconds())
	atomic.AddInt64(&pm.invalidationCount, 1)

	tm := pm.getTenantMetrics(tenantID)
	tm.mu.Lock()
	tm.invalidationLag += lag
	tm.invalidationCount++
	tm.mu.Unlock()
}

// updateCacheHitRate updates cache hit rate for tenant metrics
func (pm *PerformanceMonitor) updateCacheHitRate(tm *TenantMetrics) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	total := tm.cacheHits + tm.cacheMisses
	if total > 0 {
		tm.cacheHitRate = float64(tm.cacheHits) / float64(total)
	}
}

// RecordAuditEvent records audit event metrics
func (pm *PerformanceMonitor) RecordAuditEvent(queued, processed, dropped int64) {
	atomic.AddInt64(&pm.auditEventsQueued, queued)
	atomic.AddInt64(&pm.auditEventsProcessed, processed)
	atomic.AddInt64(&pm.auditEventsDropped, dropped)
}

// GetStats returns current performance statistics
func (pm *PerformanceMonitor) GetStats() map[string]interface{} {
	totalReqs := atomic.LoadInt64(&pm.totalRequests)
	totalDuration := atomic.LoadInt64(&pm.requestDuration)

	avgDuration := time.Duration(0)
	if totalReqs > 0 {
		avgDuration = time.Duration(totalDuration / totalReqs)
	}

	totalCache := atomic.LoadInt64(&pm.cacheHits) + atomic.LoadInt64(&pm.cacheMisses)
	cacheHitRate := float64(0)
	if totalCache > 0 {
		cacheHitRate = float64(atomic.LoadInt64(&pm.cacheHits)) / float64(totalCache)
	}

	return map[string]interface{}{
		"uptime_seconds":           time.Since(pm.startTime).Seconds(),
		"total_requests":           totalReqs,
		"active_requests":          atomic.LoadInt64(&pm.activeRequests),
		"average_request_duration": avgDuration.String(),
		"error_count":              atomic.LoadInt64(&pm.errorCount),
		"cache_hits":               atomic.LoadInt64(&pm.cacheHits),
		"cache_misses":             atomic.LoadInt64(&pm.cacheMisses),
		"cache_hit_rate":           fmt.Sprintf("%.2f%%", cacheHitRate*100),
		"cache_evictions":          atomic.LoadInt64(&pm.cacheEvictions),
		"cache_size":               atomic.LoadInt64(&pm.cacheSize),
		"cache_inflight":           atomic.LoadInt64(&pm.cacheInflight),
		"cache_invalidations":      atomic.LoadInt64(&pm.cacheInvalidations),
		"qos_token_denials":        atomic.LoadInt64(&pm.qosTokenDenials),
		"qos_breaker_trips":        atomic.LoadInt64(&pm.qosBreakerTrips),
		"qos_load_shed":            atomic.LoadInt64(&pm.qosLoadShed),
		"qos_circuit_open":         atomic.LoadInt64(&pm.qosCircuitOpen),
		"invalidation_count":       atomic.LoadInt64(&pm.invalidationCount),
		"invalidation_avg_lag":     pm.getAvgInvalidationLag().String(),
		"gc_cycles":                atomic.LoadInt64(&pm.gcCycles),
		"audit_events_queued":      atomic.LoadInt64(&pm.auditEventsQueued),
		"audit_events_processed":   atomic.LoadInt64(&pm.auditEventsProcessed),
		"audit_events_dropped":     atomic.LoadInt64(&pm.auditEventsDropped),
		"go_routines":              runtime.NumGoroutine(),
		"go_gc_stats":              pm.getGCStats(),
	}
}

// logStats periodically logs performance statistics
func (pm *PerformanceMonitor) logStats(ctx context.Context) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			stats := pm.GetStats()
			logging.GetLogger().Sugar().Infof("Performance Stats: requests=%d, avg_duration=%s, cache_hit_rate=%s, active=%d, errors=%d",
				stats["total_requests"],
				stats["average_request_duration"],
				stats["cache_hit_rate"],
				stats["active_requests"],
				stats["error_count"])
		}
	}
}

// getAvgInvalidationLag calculates average invalidation lag
func (pm *PerformanceMonitor) getAvgInvalidationLag() time.Duration {
	count := atomic.LoadInt64(&pm.invalidationCount)
	if count == 0 {
		return 0
	}
	totalLag := atomic.LoadInt64(&pm.invalidationLag)
	return time.Duration(totalLag / count)
}

// getGCStats returns detailed GC statistics
func (pm *PerformanceMonitor) getGCStats() map[string]interface{} {
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)

	return map[string]interface{}{
		"alloc_bytes":         stats.Alloc,
		"total_alloc_bytes":   stats.TotalAlloc,
		"sys_bytes":           stats.Sys,
		"heap_alloc_bytes":    stats.HeapAlloc,
		"heap_sys_bytes":      stats.HeapSys,
		"heap_idle_bytes":     stats.HeapIdle,
		"heap_released_bytes": stats.HeapReleased,
		"gc_cycles":           stats.NumGC,
		"gc_pause_total_ns":   stats.PauseTotalNs,
		"gc_pause_recent_ns":  stats.PauseNs[(stats.NumGC+255)%256],
		"next_gc_bytes":       stats.NextGC,
		"num_goroutines":      runtime.NumGoroutine(),
	}
}

// monitorGC monitors garbage collection cycles
func (pm *PerformanceMonitor) monitorGC(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	var lastGCCycles uint32
	var lastGCPauseTime uint64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			var stats runtime.MemStats
			runtime.ReadMemStats(&stats)

			// Count GC cycles
			if stats.NumGC > lastGCCycles {
				atomic.AddInt64(&pm.gcCycles, int64(stats.NumGC-lastGCCycles))
				lastGCCycles = stats.NumGC
			}

			// Track GC pause time
			if stats.PauseTotalNs > lastGCPauseTime {
				atomic.StoreInt64(&pm.gcPauseTime, int64(stats.PauseTotalNs))
				lastGCPauseTime = stats.PauseTotalNs
			}
		}
	}
}

// SetupPprofEndpoints sets up pprof HTTP endpoints
func SetupPprofEndpoints() {
	// pprof endpoints are automatically available at /debug/pprof/
	// This function can be extended to add custom endpoints
	logging.GetLogger().Sugar().Info("pprof endpoints available at /debug/pprof/")
}

// Middleware for request monitoring
func PerformanceMiddleware(pm *PerformanceMonitor) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			pm.RecordActiveRequest(1)
			defer pm.RecordActiveRequest(-1)

			// Call the next handler
			next.ServeHTTP(w, r)

			// Record metrics with error detection
			duration := time.Since(start)

			// TODO: Proper error detection would require response wrapper
			// For now, assume no error
			pm.RecordRequest(duration, false)
		})
	}
}

// RecordTenantLatency records latency for tenant-specific performance tracking
func (pm *PerformanceMonitor) RecordTenantLatency(tenantID string, duration time.Duration, success bool) {
	tm := pm.getTenantMetrics(tenantID)

	tm.mu.Lock()
	tm.requestCount++
	if !success {
		tm.errorCount++
	}
	tm.totalLatency += duration
	tm.latencies = append(tm.latencies, duration)
	tm.mu.Unlock()

	// Check performance targets
	pm.checkPerformanceTargets(tenantID, duration)
}

// GetTenantPerformanceSnapshot returns performance metrics for a specific tenant
func (pm *PerformanceMonitor) GetTenantPerformanceSnapshot(tenantID string) map[string]interface{} {
	tm := pm.getTenantMetrics(tenantID)

	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.requestCount == 0 {
		return map[string]interface{}{
			"tenant_id":     tenantID,
			"request_count": 0,
			"error_rate":    0.0,
			"p50_latency":   "0ms",
			"p95_latency":   "0ms",
			"p99_latency":   "0ms",
			"avg_latency":   "0ms",
		}
	}

	// Calculate percentiles
	p50, p95, p99 := pm.calculatePercentiles(tm.latencies)

	return map[string]interface{}{
		"tenant_id":        tenantID,
		"request_count":    tm.requestCount,
		"error_count":      tm.errorCount,
		"error_rate":       fmt.Sprintf("%.2f%%", float64(tm.errorCount)/float64(tm.requestCount)*100),
		"p50_latency":      p50.String(),
		"p95_latency":      p95.String(),
		"p99_latency":      p99.String(),
		"avg_latency":      (tm.totalLatency / time.Duration(tm.requestCount)).String(),
		"meets_p50_target": p50 <= pm.targets.p50,
		"meets_p95_target": p95 <= pm.targets.p95,
		"meets_p99_target": p99 <= pm.targets.p99,
	}
}

// checkPerformanceTargets checks if latency targets are being met
func (pm *PerformanceMonitor) checkPerformanceTargets(tenantID string, duration time.Duration) {
	if duration > pm.targets.p99 {
		logging.GetLogger().Sugar().Warnf("PERFORMANCE BREACH: Tenant %s exceeded P99 target: %v > %v",
			tenantID, duration, pm.targets.p99)
	} else if duration > pm.targets.p95 {
		logging.GetLogger().Sugar().Warnf("PERFORMANCE WARNING: Tenant %s exceeded P95 target: %v > %v",
			tenantID, duration, pm.targets.p95)
	}
}

// calculatePercentiles calculates p50, p95, p99 from latency samples
func (pm *PerformanceMonitor) calculatePercentiles(latencies []time.Duration) (time.Duration, time.Duration, time.Duration) {
	if len(latencies) == 0 {
		return 0, 0, 0
	}

	// Simple percentile calculation (production would use more sophisticated method)
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Sort latencies
	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	p50 := sorted[len(sorted)/2]
	p95 := sorted[int(float64(len(sorted))*0.95)]
	p99 := sorted[int(float64(len(sorted))*0.99)]

	return p50, p95, p99
}

// getTenantMetrics gets or creates tenant metrics tracker
func (pm *PerformanceMonitor) getTenantMetrics(tenantID string) *TenantMetrics {
	pm.metricsMux.RLock()
	tm, ok := pm.tenantMetrics[tenantID]
	pm.metricsMux.RUnlock()

	if !ok {
		pm.metricsMux.Lock()
		defer pm.metricsMux.Unlock()

		// Double-check after acquiring write lock
		if tm, ok := pm.tenantMetrics[tenantID]; ok {
			return tm
		}

		tm = &TenantMetrics{
			tenantID:  tenantID,
			latencies: make([]time.Duration, 0, 1000), // Pre-allocate capacity
			lastReset: time.Now(),
		}
		pm.tenantMetrics[tenantID] = tm
	}

	return tm
}

// GetPerformanceTargets returns the current performance targets
func (pm *PerformanceMonitor) GetPerformanceTargets() map[string]time.Duration {
	return map[string]time.Duration{
		"p50": pm.targets.p50,
		"p95": pm.targets.p95,
		"p99": pm.targets.p99,
	}
}

// ResetTenantMetrics resets metrics for a tenant
func (pm *PerformanceMonitor) ResetTenantMetrics(tenantID string) {
	tm := pm.getTenantMetrics(tenantID)

	tm.mu.Lock()
	tm.requestCount = 0
	tm.errorCount = 0
	tm.totalLatency = 0
	tm.latencies = tm.latencies[:0] // Clear slice but keep capacity
	tm.lastReset = time.Now()
	tm.mu.Unlock()
}
