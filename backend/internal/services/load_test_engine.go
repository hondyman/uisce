package services

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// LoadTestScenario represents different load testing scenarios
type LoadTestScenario struct {
	Name             string
	Description      string
	Duration         time.Duration
	Concurrency      int
	RPS              int                // requests per second target
	TenantDist       map[string]float64 // tenant -> probability distribution
	CacheHitRate     float64            // target cache hit rate (0.0-1.0)
	InvalidationFreq time.Duration      // how often to trigger invalidations
}

// LoadTestResult contains results from a load test run
type LoadTestResult struct {
	ScenarioName   string
	Duration       time.Duration
	TotalRequests  int64
	SuccessfulReqs int64
	FailedReqs     int64
	P50Latency     time.Duration
	P95Latency     time.Duration
	P99Latency     time.Duration
	AvgLatency     time.Duration
	TargetRPS      int
	ActualRPS      float64
	CacheHitRate   float64
	QoSDenials     int64
	BreakerTrips   int64
	ErrorRate      float64
	StartTime      time.Time
	EndTime        time.Time
}

// LoadTestEngine manages load testing execution
type LoadTestEngine struct {
	scenarios []*LoadTestScenario
	results   []*LoadTestResult
	running   bool
	mu        sync.RWMutex

	// Test targets
	accessSvc   *AccessIntelligenceService
	perfMonitor *PerformanceMonitor

	// Load generation
	workers  []*LoadWorker
	workerWg sync.WaitGroup

	// Metrics collection
	metrics struct {
		totalRequests  int64
		successfulReqs int64
		failedReqs     int64
		latencies      []time.Duration
		latenciesMu    sync.Mutex
	}
}

// LoadWorker represents a single load testing worker
type LoadWorker struct {
	id        int
	engine    *LoadTestEngine
	scenario  *LoadTestScenario
	stopChan  chan struct{}
	requests  int64
	latencies []time.Duration
	mu        sync.Mutex
}

// NewLoadTestEngine creates a new load testing engine
func NewLoadTestEngine(accessSvc *AccessIntelligenceService, perfMonitor *PerformanceMonitor) *LoadTestEngine {
	return &LoadTestEngine{
		scenarios:   make([]*LoadTestScenario, 0),
		results:     make([]*LoadTestResult, 0),
		accessSvc:   accessSvc,
		perfMonitor: perfMonitor,
	}
}

// AddScenario adds a load testing scenario
func (lte *LoadTestEngine) AddScenario(scenario *LoadTestScenario) {
	lte.mu.Lock()
	defer lte.mu.Unlock()
	lte.scenarios = append(lte.scenarios, scenario)
}

// RunScenario executes a specific load testing scenario
func (lte *LoadTestEngine) RunScenario(ctx context.Context, scenarioName string) (*LoadTestResult, error) {
	lte.mu.RLock()
	var scenario *LoadTestScenario
	for _, s := range lte.scenarios {
		if s.Name == scenarioName {
			scenario = s
			break
		}
	}
	lte.mu.RUnlock()

	if scenario == nil {
		return nil, fmt.Errorf("scenario %s not found", scenarioName)
	}

	logging.GetLogger().Sugar().Infof("Starting load test scenario: %s", scenario.Name)
	logging.GetLogger().Sugar().Infof("Duration: %v, Concurrency: %d, Target RPS: %d",
		scenario.Duration, scenario.Concurrency, scenario.RPS)

	result := &LoadTestResult{
		ScenarioName: scenario.Name,
		Duration:     scenario.Duration,
		TargetRPS:    scenario.RPS,
		StartTime:    time.Now(),
	}

	// Reset metrics
	atomic.StoreInt64(&lte.metrics.totalRequests, 0)
	atomic.StoreInt64(&lte.metrics.successfulReqs, 0)
	atomic.StoreInt64(&lte.metrics.failedReqs, 0)
	lte.metrics.latenciesMu.Lock()
	lte.metrics.latencies = lte.metrics.latencies[:0]
	lte.metrics.latenciesMu.Unlock()

	// Start invalidation simulator if configured
	var cancelInvalidation context.CancelFunc
	if scenario.InvalidationFreq > 0 {
		var invalidationCtx context.Context
		invalidationCtx, cancelInvalidation = context.WithCancel(ctx)
		go lte.runInvalidationSimulator(invalidationCtx, scenario)
	}

	// Start load workers
	lte.startWorkers(scenario)

	// Wait for scenario duration or context cancellation
	scenarioCtx, cancel := context.WithTimeout(ctx, scenario.Duration)
	defer cancel()

	<-scenarioCtx.Done()

	// Stop workers
	lte.stopWorkers()

	// Stop invalidation simulator
	if cancelInvalidation != nil {
		cancelInvalidation()
	}

	// Collect results
	result.EndTime = time.Now()
	result.TotalRequests = atomic.LoadInt64(&lte.metrics.totalRequests)
	result.SuccessfulReqs = atomic.LoadInt64(&lte.metrics.successfulReqs)
	result.FailedReqs = atomic.LoadInt64(&lte.metrics.failedReqs)

	// Calculate latencies
	lte.metrics.latenciesMu.Lock()
	if len(lte.metrics.latencies) > 0 {
		result.P50Latency, result.P95Latency, result.P99Latency = lte.calculatePercentiles(lte.metrics.latencies)
		totalLatency := time.Duration(0)
		for _, lat := range lte.metrics.latencies {
			totalLatency += lat
		}
		result.AvgLatency = totalLatency / time.Duration(len(lte.metrics.latencies))
	}
	lte.metrics.latenciesMu.Unlock()

	// Calculate rates
	actualDuration := result.EndTime.Sub(result.StartTime)
	result.ActualRPS = float64(result.TotalRequests) / actualDuration.Seconds()
	result.ErrorRate = float64(result.FailedReqs) / float64(result.TotalRequests) * 100

	// Get QoS metrics from performance monitor
	if perfStats := lte.perfMonitor.GetStats(); perfStats != nil {
		if cacheHitRate, ok := perfStats["cache_hit_rate"].(string); ok {
			fmt.Sscanf(cacheHitRate, "%f%%", &result.CacheHitRate)
		}
		result.QoSDenials = lte.perfMonitor.qosTokenDenials
		result.BreakerTrips = lte.perfMonitor.qosBreakerTrips
	}

	lte.mu.Lock()
	lte.results = append(lte.results, result)
	lte.mu.Unlock()

	logging.GetLogger().Sugar().Infof("Load test completed: %d requests, %.1f RPS, %.2f%% error rate, P95: %v",
		result.TotalRequests, result.ActualRPS, result.ErrorRate, result.P95Latency)

	return result, nil
}

// startWorkers starts the load testing workers
func (lte *LoadTestEngine) startWorkers(scenario *LoadTestScenario) {
	lte.workers = make([]*LoadWorker, scenario.Concurrency)

	for i := 0; i < scenario.Concurrency; i++ {
		worker := &LoadWorker{
			id:        i,
			engine:    lte,
			scenario:  scenario,
			stopChan:  make(chan struct{}),
			latencies: make([]time.Duration, 0, 1000),
		}
		lte.workers[i] = worker
		lte.workerWg.Add(1)
		go worker.run()
	}
}

// stopWorkers stops all load testing workers
func (lte *LoadTestEngine) stopWorkers() {
	for _, worker := range lte.workers {
		close(worker.stopChan)
	}
	lte.workerWg.Wait()
}

// runInvalidationSimulator simulates cache invalidations during load test
func (lte *LoadTestEngine) runInvalidationSimulator(ctx context.Context, scenario *LoadTestScenario) {
	ticker := time.NewTicker(scenario.InvalidationFreq)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Simulate invalidation by triggering cache refresh
			// This would normally be done by the cache invalidation system
			logging.GetLogger().Sugar().Debug("Simulating cache invalidation")
			lte.perfMonitor.RecordCacheInvalidation()
		}
	}
}

// calculatePercentiles calculates p50, p95, p99 from latency samples
func (lte *LoadTestEngine) calculatePercentiles(latencies []time.Duration) (time.Duration, time.Duration, time.Duration) {
	if len(latencies) == 0 {
		return 0, 0, 0
	}

	// Simple sorting for percentile calculation
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

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

// run executes the load testing loop for a worker
func (w *LoadWorker) run() {
	defer w.engine.workerWg.Done()

	ticker := time.NewTicker(time.Second / time.Duration(w.scenario.RPS/w.scenario.Concurrency))
	defer ticker.Stop()

	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.executeRequest()
		}
	}
}

// executeRequest executes a single test request
func (w *LoadWorker) executeRequest() {
	atomic.AddInt64(&w.engine.metrics.totalRequests, 1)

	// Select random tenant based on distribution
	tenantID := w.selectTenant()

	start := time.Now()

	// Simulate cache behavior based on target hit rate
	isCacheHit := rand.Float64() < w.scenario.CacheHitRate

	if isCacheHit {
		w.engine.perfMonitor.RecordTenantCacheHit(tenantID)
	} else {
		w.engine.perfMonitor.RecordTenantCacheMiss(tenantID)
	}

	// Execute the actual request
	_, err := w.engine.accessSvc.GetEffectiveClaims(context.Background(), "test-user", tenantID)

	duration := time.Since(start)

	w.mu.Lock()
	w.latencies = append(w.latencies, duration)
	w.mu.Unlock()

	w.engine.metrics.latenciesMu.Lock()
	w.engine.metrics.latencies = append(w.engine.metrics.latencies, duration)
	w.engine.metrics.latenciesMu.Unlock()

	if err != nil {
		atomic.AddInt64(&w.engine.metrics.failedReqs, 1)

		// Check if it's a QoS error
		if qosErr, ok := err.(*QoSError); ok {
			switch qosErr.Code {
			case "rate_limit_exceeded":
				w.engine.perfMonitor.RecordTenantQoSDenial(tenantID)
			case "circuit_breaker_open":
				w.engine.perfMonitor.RecordTenantQoSBreakerTrip(tenantID)
			}
		}
	} else {
		atomic.AddInt64(&w.engine.metrics.successfulReqs, 1)
	}

	atomic.AddInt64(&w.requests, 1)
}

// selectTenant selects a tenant based on the scenario's distribution
func (w *LoadWorker) selectTenant() string {
	if len(w.scenario.TenantDist) == 0 {
		return "default-tenant"
	}

	r := rand.Float64()
	cumulative := 0.0

	for tenantID, prob := range w.scenario.TenantDist {
		cumulative += prob
		if r <= cumulative {
			return tenantID
		}
	}

	// Fallback to first tenant
	for tenantID := range w.scenario.TenantDist {
		return tenantID
	}

	return "default-tenant"
}

// GetResults returns all load test results
func (lte *LoadTestEngine) GetResults() []*LoadTestResult {
	lte.mu.RLock()
	defer lte.mu.RUnlock()

	results := make([]*LoadTestResult, len(lte.results))
	copy(results, lte.results)
	return results
}

// CreateDefaultScenarios creates a set of default load testing scenarios
func (lte *LoadTestEngine) CreateDefaultScenarios() {
	// 1x Peak Load Scenario
	lte.AddScenario(&LoadTestScenario{
		Name:        "1x-peak-load",
		Description: "Simulate 1x expected peak load",
		Duration:    5 * time.Minute,
		Concurrency: 50,
		RPS:         1000,
		TenantDist: map[string]float64{
			"gold-tenant":   0.1,
			"silver-tenant": 0.3,
			"bronze-tenant": 0.6,
		},
		CacheHitRate:     0.85,
		InvalidationFreq: 30 * time.Second,
	})

	// 2x Peak Load Scenario
	lte.AddScenario(&LoadTestScenario{
		Name:        "2x-peak-load",
		Description: "Simulate 2x expected peak load with stress",
		Duration:    3 * time.Minute,
		Concurrency: 100,
		RPS:         2000,
		TenantDist: map[string]float64{
			"gold-tenant":   0.1,
			"silver-tenant": 0.3,
			"bronze-tenant": 0.6,
		},
		CacheHitRate:     0.75,
		InvalidationFreq: 15 * time.Second,
	})

	// Cache Invalidation Storm Scenario
	lte.AddScenario(&LoadTestScenario{
		Name:        "cache-invalidation-storm",
		Description: "Test behavior during frequent cache invalidations",
		Duration:    2 * time.Minute,
		Concurrency: 30,
		RPS:         500,
		TenantDist: map[string]float64{
			"gold-tenant":   0.2,
			"silver-tenant": 0.3,
			"bronze-tenant": 0.5,
		},
		CacheHitRate:     0.60,
		InvalidationFreq: 5 * time.Second,
	})

	// QoS Stress Test Scenario
	lte.AddScenario(&LoadTestScenario{
		Name:        "qos-stress-test",
		Description: "Test QoS controls under heavy load",
		Duration:    4 * time.Minute,
		Concurrency: 80,
		RPS:         1500,
		TenantDist: map[string]float64{
			"gold-tenant":   0.05,
			"silver-tenant": 0.15,
			"bronze-tenant": 0.80, // Heavy bronze tenant load
		},
		CacheHitRate:     0.70,
		InvalidationFreq: 20 * time.Second,
	})
}
