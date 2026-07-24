package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// CapacityMetricsCollector collects and pushes capacity index metrics to Dynatrace
type CapacityMetricsCollector struct {
	dynatraceURL       string
	dynatraceToken     string
	perfMonitor        *PerformanceMonitor
	qosManager         *QoSManager
	tenantConfig       *TenantConfigService
	loadTestEngine     *LoadTestEngine
	collectionInterval time.Duration
	shutdown           chan struct{}
	wg                 sync.WaitGroup
}

// NewCapacityMetricsCollector creates a new collector
func NewCapacityMetricsCollector(
	dynatraceURL, dynatraceToken string,
	perfMonitor *PerformanceMonitor,
	qosManager *QoSManager,
	tenantConfig *TenantConfigService,
	loadTestEngine *LoadTestEngine,
) *CapacityMetricsCollector {
	return &CapacityMetricsCollector{
		dynatraceURL:       dynatraceURL,
		dynatraceToken:     dynatraceToken,
		perfMonitor:        perfMonitor,
		qosManager:         qosManager,
		tenantConfig:       tenantConfig,
		loadTestEngine:     loadTestEngine,
		collectionInterval: 30 * time.Second, // Push every 30 seconds
		shutdown:           make(chan struct{}),
	}
}

// MetricPayload represents the structure for Dynatrace metrics ingest
type MetricPayload struct {
	MetricKey  string            `json:"metricKey"`
	Value      float64           `json:"value"`
	Timestamp  int64             `json:"timestamp"`
	Dimensions map[string]string `json:"dimensions,omitempty"`
}

// calculateGlobalCapacityIndex computes the composite capacity index
func (c *CapacityMetricsCollector) calculateGlobalCapacityIndex(_ context.Context) float64 {
	// Get basic system metrics
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// CPU usage (simplified - in production you'd use a proper CPU monitoring library)
	// For now, we'll use goroutine count as a proxy for concurrency
	activeGoroutines := runtime.NumGoroutine()

	// Calculate utilization ratios (0.0 to 1.0 scale)
	memRatio := float64(memStats.Alloc) / float64(memStats.Sys)
	if memRatio > 1.0 {
		memRatio = 1.0
	}

	concurrencyRatio := float64(activeGoroutines) / 10000.0 // Assume 10k max goroutines
	if concurrencyRatio > 1.0 {
		concurrencyRatio = 1.0
	}

	// Get cache metrics from performance monitor (using available counters)
	cacheHits := atomic.LoadInt64(&c.perfMonitor.cacheHits)
	cacheMisses := atomic.LoadInt64(&c.perfMonitor.cacheMisses)
	totalCacheOps := cacheHits + cacheMisses
	cacheHitRatio := 0.0
	if totalCacheOps > 0 {
		cacheHitRatio = float64(cacheHits) / float64(totalCacheOps)
	}

	// For this example, we'll use cache hit ratio as an inverse capacity indicator
	// Low cache hit ratio indicates high load/stress
	cacheStressRatio := 1.0 - cacheHitRatio

	// Capacity index is the maximum of all utilization ratios
	capacityIndex := max(memRatio, concurrencyRatio, cacheStressRatio)

	// Clamp to 1.0 max
	if capacityIndex > 1.0 {
		capacityIndex = 1.0
	}

	return capacityIndex
}

// calculatePerTenantCapacityIndex computes capacity index per tenant
func (c *CapacityMetricsCollector) calculatePerTenantCapacityIndex(_ context.Context) map[string]float64 {
	tenantMetrics := make(map[string]float64)

	// For this simplified example, we'll create mock per-tenant metrics
	// In a real implementation, you'd track per-tenant metrics in the QoS manager
	mockTenants := []string{"tenant-1", "tenant-2", "tenant-3"}

	for _, tenantID := range mockTenants {
		// Simulate tenant-specific load based on tenant ID hash
		loadFactor := float64(int(tenantID[len(tenantID)-1])%10) / 10.0
		tenantCapacityIndex := loadFactor

		if tenantCapacityIndex > 1.0 {
			tenantCapacityIndex = 1.0
		}

		tenantMetrics[tenantID] = tenantCapacityIndex
	}

	return tenantMetrics
}

// pushMetricsToDynatrace sends metrics to Dynatrace Metrics Ingest API
func (c *CapacityMetricsCollector) pushMetricsToDynatrace(ctx context.Context, metrics []MetricPayload) error {
	if len(metrics) == 0 {
		return nil
	}

	payload, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.dynatraceURL+"/api/v2/metrics/ingest", bytes.NewBuffer(payload))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Api-Token "+c.dynatraceToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("dynatrace API returned status %d", resp.StatusCode)
	}

	return nil
}

// collectAndPushMetrics runs the main collection loop
func (c *CapacityMetricsCollector) collectAndPushMetrics(ctx context.Context) {
	ticker := time.NewTicker(c.collectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.shutdown:
			return
		case <-ticker.C:
			c.collectAndPushOnce(ctx)
		}
	}
}

// collectAndPushOnce performs a single collection and push cycle
func (c *CapacityMetricsCollector) collectAndPushOnce(ctx context.Context) {
	var metrics []MetricPayload
	timestamp := time.Now().UnixMilli()

	// Global capacity index
	globalCapacityIndex := c.calculateGlobalCapacityIndex(ctx)
	metrics = append(metrics, MetricPayload{
		MetricKey: "semlayer.capacity.index",
		Value:     globalCapacityIndex,
		Timestamp: timestamp,
	})

	// Per-tenant capacity indices
	tenantCapacityIndices := c.calculatePerTenantCapacityIndex(ctx)
	for tenantID, capacityIndex := range tenantCapacityIndices {
		metrics = append(metrics, MetricPayload{
			MetricKey: "semlayer.capacity.index",
			Value:     capacityIndex,
			Timestamp: timestamp,
			Dimensions: map[string]string{
				"tenant_id": tenantID,
			},
		})
	}

	// Push to Dynatrace
	if err := c.pushMetricsToDynatrace(ctx, metrics); err != nil {
		log.Printf("Failed to push capacity metrics to Dynatrace: %v", err)
	} else {
		log.Printf("Successfully pushed %d capacity metrics to Dynatrace", len(metrics))
	}
}

// Start begins the metrics collection
func (c *CapacityMetricsCollector) Start(ctx context.Context) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.collectAndPushMetrics(ctx)
	}()
}

// Stop gracefully shuts down the collector
func (c *CapacityMetricsCollector) Stop() {
	close(c.shutdown)
	c.wg.Wait()
}

// Helper function for max of float64 values
func max(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

// Example usage:
//
// collector := NewCapacityMetricsCollector(
//     "https://your-environment.live.dynatrace.com",
//     "your-api-token",
//     perfMonitor,
//     qosManager,
//     tenantConfig,
//     loadTestEngine,
// )
//
// ctx := context.Background()
// collector.Start(ctx)
//
// // Later...
// collector.Stop()
