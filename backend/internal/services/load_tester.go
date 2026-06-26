package services

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/models"
)

// LoadTestConfig configures the load test parameters
type LoadTestConfig struct {
	Duration         time.Duration
	Concurrency      int
	RequestRate      int // requests per second (0 = unlimited)
	TenantCount      int
	UserCount        int
	AssetCount       int
	WarmupDuration   time.Duration
	ProgressInterval time.Duration
}

// LoadTestResults contains the results of a load test
type LoadTestResults struct {
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	Duration           time.Duration
	RequestsPerSecond  float64
	AverageLatency     time.Duration
	P50Latency         time.Duration
	P95Latency         time.Duration
	P99Latency         time.Duration
	ErrorRate          float64
	CacheHitRate       float64
}

// LoadTester performs load testing on the access intelligence service
type LoadTester struct {
	service *AccessIntelligenceService
	config  LoadTestConfig
}

// NewLoadTester creates a new load tester
func NewLoadTester(service *AccessIntelligenceService, config LoadTestConfig) *LoadTester {
	return &LoadTester{
		service: service,
		config:  config,
	}
}

// RunLoadTest executes the load test
func (lt *LoadTester) RunLoadTest(ctx context.Context) (*LoadTestResults, error) {
	logging.GetLogger().Sugar().Infof("Starting load test: duration=%v, concurrency=%d, rate=%d req/s",
		lt.config.Duration, lt.config.Concurrency, lt.config.RequestRate)

	// Warmup phase
	if lt.config.WarmupDuration > 0 {
		logging.GetLogger().Sugar().Infof("Warmup phase: %v", lt.config.WarmupDuration)
		lt.runWarmup(ctx)
	}

	// Test phase
	start := time.Now()
	results := lt.runTestPhase(ctx)
	results.Duration = time.Since(start)

	// Calculate final metrics
	results.RequestsPerSecond = float64(results.TotalRequests) / results.Duration.Seconds()
	results.ErrorRate = float64(results.FailedRequests) / float64(results.TotalRequests) * 100

	logging.GetLogger().Sugar().Infof("Load test completed: %d requests, %.2f req/s, %.2f%% error rate",
		results.TotalRequests, results.RequestsPerSecond, results.ErrorRate)

	return results, nil
}

// runWarmup performs the warmup phase
func (lt *LoadTester) runWarmup(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, lt.config.WarmupDuration)
	defer cancel()

	var wg sync.WaitGroup
	requests := make(chan models.EvaluateAccessRequest, lt.config.Concurrency*10)

	// Start workers
	for i := 0; i < lt.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			lt.warmupWorker(ctx, requests)
		}()
	}

	// Generate requests
	go func() {
		defer close(requests)
		ticker := time.NewTicker(10 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				req := lt.generateRandomRequest()
				select {
				case requests <- req:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	wg.Wait()
	logging.GetLogger().Sugar().Info("Warmup phase completed")
}

// runTestPhase performs the actual test phase
func (lt *LoadTester) runTestPhase(ctx context.Context) *LoadTestResults {
	ctx, cancel := context.WithTimeout(ctx, lt.config.Duration)
	defer cancel()

	results := &LoadTestResults{}
	latencies := make([]time.Duration, 0, 100000)

	var wg sync.WaitGroup
	requests := make(chan models.EvaluateAccessRequest, lt.config.Concurrency*100)

	// Progress reporting
	progressTicker := time.NewTicker(lt.config.ProgressInterval)
	defer progressTicker.Stop()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-progressTicker.C:
				current := atomic.LoadInt64(&results.TotalRequests)
				logging.GetLogger().Sugar().Infof("Progress: %d requests processed", current)
			}
		}
	}()

	// Rate limiter if specified
	var rateLimiter <-chan time.Time
	if lt.config.RequestRate > 0 {
		rateLimiter = time.Tick(time.Second / time.Duration(lt.config.RequestRate))
	}

	// Start workers
	for i := 0; i < lt.config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			lt.testWorker(ctx, requests, results, latencies)
		}(i)
	}

	// Generate requests
	go func() {
		defer close(requests)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				if rateLimiter != nil {
					select {
					case <-ctx.Done():
						return
					case <-rateLimiter:
					}
				}

				req := lt.generateRandomRequest()
				select {
				case requests <- req:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	wg.Wait()

	// Calculate latency percentiles
	results.AverageLatency = lt.calculateAverageLatency(latencies)
	results.P50Latency = lt.calculatePercentileLatency(latencies, 50)
	results.P95Latency = lt.calculatePercentileLatency(latencies, 95)
	results.P99Latency = lt.calculatePercentileLatency(latencies, 99)

	return results
}

// warmupWorker processes requests during warmup
func (lt *LoadTester) warmupWorker(ctx context.Context, requests <-chan models.EvaluateAccessRequest) {
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-requests:
			if !ok {
				return
			}
			// Just make the request, don't track metrics
			lt.service.EvaluateAccess(ctx, req)
		}
	}
}

// testWorker processes requests during test phase
func (lt *LoadTester) testWorker(ctx context.Context, requests <-chan models.EvaluateAccessRequest, results *LoadTestResults, latencies []time.Duration) {
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-requests:
			if !ok {
				return
			}

			start := time.Now()
			_, err := lt.service.EvaluateAccess(ctx, req)
			duration := time.Since(start)

			atomic.AddInt64(&results.TotalRequests, 1)

			if err != nil {
				atomic.AddInt64(&results.FailedRequests, 1)
			} else {
				atomic.AddInt64(&results.SuccessfulRequests, 1)
			}

			// Record latency (with some sampling to avoid memory issues)
			if len(latencies) < cap(latencies) || rand.Float64() < 0.1 {
				latencies = append(latencies, duration)
			}
		}
	}
}

// generateRandomRequest creates a random access evaluation request
func (lt *LoadTester) generateRandomRequest() models.EvaluateAccessRequest {
	tenantID := fmt.Sprintf("tenant_%d", rand.Intn(lt.config.TenantCount)+1)
	userID := fmt.Sprintf("user_%d", rand.Intn(lt.config.UserCount)+1)
	assetID := fmt.Sprintf("asset_%d", rand.Intn(lt.config.AssetCount)+1)

	actions := []string{"read", "write", "query", "delete"}
	action := actions[rand.Intn(len(actions))]

	return models.EvaluateAccessRequest{
		UserID:   userID,
		TenantID: tenantID,
		AssetID:  assetID,
		Action:   action,
	}
}

// calculateAverageLatency calculates the average latency
func (lt *LoadTester) calculateAverageLatency(latencies []time.Duration) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}

	return total / time.Duration(len(latencies))
}

// calculatePercentileLatency calculates the nth percentile latency
func (lt *LoadTester) calculatePercentileLatency(latencies []time.Duration, percentile float64) time.Duration {
	if len(latencies) == 0 {
		return 0
	}

	// Simple sort and pick (not the most efficient for large datasets)
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)

	// Basic bubble sort for simplicity
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j] > sorted[j+1] {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	index := int(float64(len(sorted)-1) * percentile / 100.0)
	return sorted[index]
}

// RunStandardLoadTest runs a standard load test configuration
func RunStandardLoadTest(service *AccessIntelligenceService) (*LoadTestResults, error) {
	config := LoadTestConfig{
		Duration:         5 * time.Minute,
		Concurrency:      50,
		RequestRate:      1000, // 1000 req/s
		TenantCount:      10,
		UserCount:        1000,
		AssetCount:       100,
		WarmupDuration:   30 * time.Second,
		ProgressInterval: 30 * time.Second,
	}

	tester := NewLoadTester(service, config)
	return tester.RunLoadTest(context.Background())
}
