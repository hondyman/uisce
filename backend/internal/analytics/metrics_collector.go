package analytics

import (
	"fmt"
	"sync"
	"time"
)

// MetricsCollector provides comprehensive metrics collection and analysis
type MetricsCollector struct {
	queryMetrics    map[string]*QueryMetrics
	cacheMetrics    map[string]*CacheMetrics
	systemMetrics   *SystemMetrics
	mu              sync.RWMutex
	retentionPeriod time.Duration
}

// QueryMetrics tracks query performance and optimization data
type QueryMetrics struct {
	QueryID              string
	QueryText            string
	ExecutionCount       int64
	TotalExecutionTime   time.Duration
	AverageExecutionTime time.Duration
	LastExecutionTime    time.Time
	CacheHitCount        int64
	CacheMissCount       int64
	ErrorCount           int64
	OptimizationCount    int64
	EstimatedCost        float64
	ActualCost           float64
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	CacheName     string
	HitCount      int64
	MissCount     int64
	EvictionCount int64
	Size          int64
	HitRate       float64
	LastUpdated   time.Time
}

// SystemMetrics tracks system-wide performance
type SystemMetrics struct {
	TotalQueries      int64
	ActiveConnections int64
	MemoryUsage       int64
	CPUUsage          float64
	Uptime            time.Duration
	StartTime         time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(retentionPeriod time.Duration) *MetricsCollector {
	return &MetricsCollector{
		queryMetrics:    make(map[string]*QueryMetrics),
		cacheMetrics:    make(map[string]*CacheMetrics),
		systemMetrics:   &SystemMetrics{StartTime: time.Now()},
		retentionPeriod: retentionPeriod,
	}
}

// RecordQueryExecution records a query execution
func (mc *MetricsCollector) RecordQueryExecution(queryID, queryText string, executionTime time.Duration, cacheHit bool, errorOccurred bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics, exists := mc.queryMetrics[queryID]
	if !exists {
		metrics = &QueryMetrics{
			QueryID:   queryID,
			QueryText: queryText,
		}
		mc.queryMetrics[queryID] = metrics
	}

	metrics.ExecutionCount++
	metrics.TotalExecutionTime += executionTime
	metrics.AverageExecutionTime = metrics.TotalExecutionTime / time.Duration(metrics.ExecutionCount)
	metrics.LastExecutionTime = time.Now()

	if cacheHit {
		metrics.CacheHitCount++
	} else {
		metrics.CacheMissCount++
	}

	if errorOccurred {
		metrics.ErrorCount++
	}

	mc.systemMetrics.TotalQueries++
}

// RecordQueryOptimization records query optimization metrics
func (mc *MetricsCollector) RecordQueryOptimization(queryID string, estimatedCost float64, optimizationTime time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics, exists := mc.queryMetrics[queryID]
	if !exists {
		metrics = &QueryMetrics{QueryID: queryID}
		mc.queryMetrics[queryID] = metrics
	}

	metrics.OptimizationCount++
	metrics.EstimatedCost = estimatedCost
}

// RecordCacheOperation records cache hit/miss operations
func (mc *MetricsCollector) RecordCacheOperation(cacheName string, hit bool, size int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics, exists := mc.cacheMetrics[cacheName]
	if !exists {
		metrics = &CacheMetrics{CacheName: cacheName}
		mc.cacheMetrics[cacheName] = metrics
	}

	if hit {
		metrics.HitCount++
	} else {
		metrics.MissCount++
	}

	metrics.Size = size
	metrics.LastUpdated = time.Now()

	total := metrics.HitCount + metrics.MissCount
	if total > 0 {
		metrics.HitRate = float64(metrics.HitCount) / float64(total)
	}
}

// RecordCacheEviction records cache eviction events
func (mc *MetricsCollector) RecordCacheEviction(cacheName string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	metrics, exists := mc.cacheMetrics[cacheName]
	if !exists {
		metrics = &CacheMetrics{CacheName: cacheName}
		mc.cacheMetrics[cacheName] = metrics
	}

	metrics.EvictionCount++
}

// UpdateSystemMetrics updates system-wide metrics
func (mc *MetricsCollector) UpdateSystemMetrics(activeConnections, memoryUsage int64, cpuUsage float64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.systemMetrics.ActiveConnections = activeConnections
	mc.systemMetrics.MemoryUsage = memoryUsage
	mc.systemMetrics.CPUUsage = cpuUsage
	mc.systemMetrics.Uptime = time.Since(mc.systemMetrics.StartTime)
}

// GetQueryMetrics returns query performance metrics
func (mc *MetricsCollector) GetQueryMetrics(queryID string) (*QueryMetrics, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics, exists := mc.queryMetrics[queryID]
	return metrics, exists
}

// GetTopQueries returns the top N queries by execution time
func (mc *MetricsCollector) GetTopQueries(limit int) []*QueryMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var queries []*QueryMetrics
	for _, metrics := range mc.queryMetrics {
		queries = append(queries, metrics)
	}

	// Sort by average execution time (descending)
	for i := 0; i < len(queries)-1; i++ {
		for j := i + 1; j < len(queries); j++ {
			if queries[i].AverageExecutionTime < queries[j].AverageExecutionTime {
				queries[i], queries[j] = queries[j], queries[i]
			}
		}
	}

	if len(queries) > limit {
		queries = queries[:limit]
	}

	return queries
}

// GetCacheMetrics returns cache performance metrics
func (mc *MetricsCollector) GetCacheMetrics(cacheName string) (*CacheMetrics, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics, exists := mc.cacheMetrics[cacheName]
	return metrics, exists
}

// GetSystemMetrics returns system-wide metrics
func (mc *MetricsCollector) GetSystemMetrics() *SystemMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a copy to prevent external modification
	return &SystemMetrics{
		TotalQueries:      mc.systemMetrics.TotalQueries,
		ActiveConnections: mc.systemMetrics.ActiveConnections,
		MemoryUsage:       mc.systemMetrics.MemoryUsage,
		CPUUsage:          mc.systemMetrics.CPUUsage,
		Uptime:            mc.systemMetrics.Uptime,
		StartTime:         mc.systemMetrics.StartTime,
	}
}

// GetPerformanceSummary returns a comprehensive performance summary
func (mc *MetricsCollector) GetPerformanceSummary() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	summary := make(map[string]interface{})

	// Query performance summary
	totalQueries := int64(0)
	totalExecutionTime := time.Duration(0)
	totalErrors := int64(0)

	for _, metrics := range mc.queryMetrics {
		totalQueries += metrics.ExecutionCount
		totalExecutionTime += metrics.TotalExecutionTime
		totalErrors += metrics.ErrorCount
	}

	summary["total_queries"] = totalQueries
	summary["total_execution_time"] = totalExecutionTime.String()
	if totalQueries > 0 {
		summary["average_query_time"] = (totalExecutionTime / time.Duration(totalQueries)).String()
		summary["error_rate"] = float64(totalErrors) / float64(totalQueries)
	}

	// Cache performance summary
	totalCacheHits := int64(0)
	totalCacheMisses := int64(0)

	for _, metrics := range mc.cacheMetrics {
		totalCacheHits += metrics.HitCount
		totalCacheMisses += metrics.MissCount
	}

	totalCacheOps := totalCacheHits + totalCacheMisses
	summary["cache_operations"] = totalCacheOps
	if totalCacheOps > 0 {
		summary["cache_hit_rate"] = float64(totalCacheHits) / float64(totalCacheOps)
	}

	// System metrics
	summary["system_metrics"] = mc.systemMetrics

	return summary
}

// CleanupOldMetrics removes metrics older than the retention period
func (mc *MetricsCollector) CleanupOldMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cutoffTime := time.Now().Add(-mc.retentionPeriod)

	// Clean up old query metrics
	for queryID, metrics := range mc.queryMetrics {
		if metrics.LastExecutionTime.Before(cutoffTime) {
			delete(mc.queryMetrics, queryID)
		}
	}

	// Clean up old cache metrics
	for cacheName, metrics := range mc.cacheMetrics {
		if metrics.LastUpdated.Before(cutoffTime) {
			delete(mc.cacheMetrics, cacheName)
		}
	}
}

// ExportMetrics exports metrics in a structured format for external analysis
func (mc *MetricsCollector) ExportMetrics() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	export := make(map[string]interface{})

	// Export query metrics
	queryMetrics := make(map[string]interface{})
	for queryID, metrics := range mc.queryMetrics {
		queryMetrics[queryID] = map[string]interface{}{
			"query_text":             metrics.QueryText,
			"execution_count":        metrics.ExecutionCount,
			"total_execution_time":   metrics.TotalExecutionTime.String(),
			"average_execution_time": metrics.AverageExecutionTime.String(),
			"last_execution_time":    metrics.LastExecutionTime,
			"cache_hit_count":        metrics.CacheHitCount,
			"cache_miss_count":       metrics.CacheMissCount,
			"error_count":            metrics.ErrorCount,
			"optimization_count":     metrics.OptimizationCount,
			"estimated_cost":         metrics.EstimatedCost,
			"actual_cost":            metrics.ActualCost,
		}
	}
	export["query_metrics"] = queryMetrics

	// Export cache metrics
	cacheMetrics := make(map[string]interface{})
	for cacheName, metrics := range mc.cacheMetrics {
		cacheMetrics[cacheName] = map[string]interface{}{
			"hit_count":      metrics.HitCount,
			"miss_count":     metrics.MissCount,
			"eviction_count": metrics.EvictionCount,
			"size":           metrics.Size,
			"hit_rate":       metrics.HitRate,
			"last_updated":   metrics.LastUpdated,
		}
	}
	export["cache_metrics"] = cacheMetrics

	// Export system metrics
	export["system_metrics"] = mc.systemMetrics

	return export
}

// GetQueryOptimizationRecommendations provides optimization suggestions
func (mc *MetricsCollector) GetQueryOptimizationRecommendations() []string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	recommendations := []string{}

	// Analyze slow queries
	topQueries := mc.GetTopQueries(10)
	for _, query := range topQueries {
		if query.AverageExecutionTime > 5*time.Second {
			recommendations = append(recommendations,
				fmt.Sprintf("Query %s is slow (avg: %v). Consider optimization.",
					query.QueryID, query.AverageExecutionTime))
		}
	}

	// Analyze cache performance
	for cacheName, metrics := range mc.cacheMetrics {
		if metrics.HitRate < 0.5 {
			recommendations = append(recommendations,
				fmt.Sprintf("Cache %s has low hit rate (%.2f%%). Consider cache strategy optimization.",
					cacheName, metrics.HitRate*100))
		}
	}

	// Analyze error rates
	totalQueries := mc.systemMetrics.TotalQueries
	if totalQueries > 0 {
		errorRate := float64(mc.getTotalErrors()) / float64(totalQueries)
		if errorRate > 0.05 {
			recommendations = append(recommendations,
				fmt.Sprintf("High error rate detected (%.2f%%). Review error handling.",
					errorRate*100))
		}
	}

	return recommendations
}

func (mc *MetricsCollector) getTotalErrors() int64 {
	total := int64(0)
	for _, metrics := range mc.queryMetrics {
		total += metrics.ErrorCount
	}
	return total
}
