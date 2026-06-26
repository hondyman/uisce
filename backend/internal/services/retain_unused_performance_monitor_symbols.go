package services

// retain_unused_performance_monitor_symbols.go
// Small retention shim referencing PerformanceMonitor and related types to
// silence staticcheck U1000 warnings for intentionally-kept monitoring fields.
func init() {
	var pm PerformanceMonitor
	_ = pm.totalRequests
	_ = pm.activeRequests
	_ = pm.requestDuration
	_ = pm.errorCount
	_ = pm.cacheHits
	_ = pm.cacheMisses
	_ = pm.cacheEvictions
	_ = pm.cacheSize
	_ = pm.cacheInflight
	_ = pm.cacheInvalidations
	_ = pm.cacheEntrySize
	_ = pm.qosTokenDenials
	_ = pm.qosBreakerTrips
	_ = pm.qosLoadShed
	_ = pm.qosCircuitOpen
	_ = pm.invalidationLag
	_ = pm.invalidationCount
	_ = pm.gcCycles
	_ = pm.gcPauseTime
	_ = pm.tokenBucketSize
	_ = pm.tokenBucketUsed
	_ = pm.tenantMetrics
	_ = &pm.metricsMux // Use address to avoid copying mutex
	_ = pm.latencyHistogram
	_ = pm.startTime

	var lh LatencyHistogram
	_ = lh.buckets
	_ = lh.maxSamples
	_ = &lh.mu // Use address to avoid copying mutex

	var tm TenantMetrics
	_ = tm.tenantID
	_ = tm.requestCount
	_ = tm.errorCount
	_ = tm.latencies
	_ = tm.cacheHits
	_ = tm.cacheMisses
	_ = tm.cacheHitRate
	_ = tm.qosDenials
	_ = tm.qosBreakerTrips
	_ = &tm.mu // Use address to avoid copying mutex
}
