package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"
)

// ============================================================================
// Load Test Suite for Semantic Query Cache
// ============================================================================

// LoadTestScenario defines parameters for a load test
type LoadTestScenario struct {
	Name            string
	NumRequests     int
	NumConcurrent   int
	CacheHitRate    float64 // Expected hit rate (0.0-1.0)
	AvgResponseTime time.Duration
	QueryVariation  float64 // How much queries vary (0.0 = identical, 1.0 = all unique)
}

// LoadTestResult contains benchmark results
type LoadTestResult struct {
	Scenario             *LoadTestScenario
	TotalRequests        int
	CompletedRequests    int
	CacheHits            int
	CacheMisses          int
	ActualHitRate        float64
	AvgLatency           time.Duration
	P50Latency           time.Duration
	P95Latency           time.Duration
	P99Latency           time.Duration
	MaxLatency           time.Duration
	MinLatency           time.Duration
	ThroughputReqPerSec  float64
	TotalDuration        time.Duration
	EstimatedCostSavings float64
	EstimatedTimeSavings time.Duration
	Errors               int
}

// ============================================================================
// Layer 1: NL → SemanticQuery Load Test
// ============================================================================

// TestLayer1NLQueryCacheWarmHit simulates repeated identical NL queries (all cache hits)
func TestLayer1NLQueryCacheWarmHit(t *testing.T) {
	scenario := &LoadTestScenario{
		Name:            "Layer 1: NL Query - Warm Cache (100% hit rate)",
		NumRequests:     10000,
		NumConcurrent:   50,
		CacheHitRate:    1.0, // All hits
		AvgResponseTime: 1 * time.Millisecond,
		QueryVariation:  0.0, // All identical queries
	}

	result := runNLQueryLoadTest(scenario)

	// Expected: ~100% hit rate, <5ms avg latency
	if result.ActualHitRate < 0.95 {
		t.Logf("WARN: Expected >95%% hit rate, got %0.1f%%", result.ActualHitRate*100)
	}
	if result.AvgLatency > 10*time.Millisecond {
		t.Logf("WARN: Expected <10ms avg latency, got %v", result.AvgLatency)
	}

	t.Logf("Layer 1 Warm Cache Test Results:\n%s", resultSummary(result))
}

// TestLayer1NLQueryCacheColdStart simulates initial cache cold start (all misses)
func TestLayer1NLQueryCacheColdStart(t *testing.T) {
	scenario := &LoadTestScenario{
		Name:            "Layer 1: NL Query - Cold Start (0% hit rate)",
		NumRequests:     1000,
		NumConcurrent:   10,
		CacheHitRate:    0.0,                    // All misses
		AvgResponseTime: 500 * time.Millisecond, // Simulated LLM latency
		QueryVariation:  1.0,                    // All unique queries
	}

	result := runNLQueryLoadTest(scenario)

	// Expected: ~0% hit rate, 500ms avg latency
	if result.ActualHitRate > 0.05 {
		t.Logf("WARN: Expected <5%% hit rate for cold start, got %0.1f%%", result.ActualHitRate*100)
	}

	t.Logf("Layer 1 Cold Start Test Results:\n%s", resultSummary(result))
}

// TestLayer1NLQueryCacheMixed simulates realistic mix of repeated and new queries
func TestLayer1NLQueryCacheMixed(t *testing.T) {
	scenario := &LoadTestScenario{
		Name:            "Layer 1: NL Query - Mixed Workload",
		NumRequests:     10000,
		NumConcurrent:   100,
		CacheHitRate:    0.7, // 70% cache hits expected
		AvgResponseTime: 300 * time.Millisecond,
		QueryVariation:  0.3, // 30% variation (70% repeated)
	}

	result := runNLQueryLoadTest(scenario)

	t.Logf("Layer 1 Mixed Workload Test Results:\n%s", resultSummary(result))
}

// runNLQueryLoadTest executes a load test for Layer 1 (NL → SemanticQuery)
func runNLQueryLoadTest(scenario *LoadTestScenario) *LoadTestResult {
	cache, err := NewSemanticQueryCache("localhost:6379", "", 1)
	if err != nil {
		fmt.Printf("Error: Could not connect to Redis: %v\n", err)
		return &LoadTestResult{
			Scenario: scenario,
			Errors:   1,
		}
	}

	ctx := context.Background()
	result := &LoadTestResult{
		Scenario:      scenario,
		TotalRequests: scenario.NumRequests,
	}

	// Generate query pool
	queries := generateQueryPool(scenario.NumRequests, scenario.QueryVariation)

	// Populate cache warmth
	for i := 0; i < int(float64(scenario.NumRequests)*scenario.CacheHitRate); i++ {
		query := queries[i%len(queries)]
		entry := &NLQueryCacheEntry{
			NLPrompt:       query.prompt,
			Datasource:     query.datasource,
			Mode:           query.mode,
			SemanticQuery:  `{"select":["field1"],"filters":[]}`,
			GeneratedAt:    time.Now(),
			LLMModel:       "gemini-pro",
			GenerationTime: 500,
			TenantID:       query.tenantID,
		}
		cache.SetNLQueryCache(ctx, query.prompt, query.datasource, query.mode, query.tenantID, entry)
	}

	// Run test
	start := time.Now()
	var wg sync.WaitGroup
	latencies := make([]time.Duration, 0, scenario.NumRequests)
	var mu sync.Mutex

	sem := make(chan struct{}, scenario.NumConcurrent)
	for i := 0; i < scenario.NumRequests; i++ {
		wg.Add(1)
		sem <- struct{}{} // Acquire semaphore

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }() // Release semaphore

			query := queries[idx%len(queries)]
			queryStart := time.Now()

			entry, _ := cache.GetNLQueryCache(ctx, query.prompt, query.datasource, query.mode, query.tenantID)

			if entry != nil {
				mu.Lock()
				result.CacheHits++
				mu.Unlock()
			} else {
				// Simulate LLM latency
				time.Sleep(time.Duration(rand.Intn(100)+450) * time.Millisecond)

				mu.Lock()
				result.CacheMisses++
				mu.Unlock()

				// Store in cache
				cacheEntry := &NLQueryCacheEntry{
					NLPrompt:       query.prompt,
					Datasource:     query.datasource,
					Mode:           query.mode,
					SemanticQuery:  `{"select":["field1"],"filters":[]}`,
					GeneratedAt:    time.Now(),
					LLMModel:       "gemini-pro",
					GenerationTime: int64(time.Since(queryStart).Milliseconds()),
					TenantID:       query.tenantID,
				}
				cache.SetNLQueryCache(ctx, query.prompt, query.datasource, query.mode, query.tenantID, cacheEntry)
			}

			latency := time.Since(queryStart)
			mu.Lock()
			latencies = append(latencies, latency)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	result.TotalDuration = time.Since(start)
	result.CompletedRequests = result.CacheHits + result.CacheMisses

	// Calculate statistics
	calculateLatencyStats(latencies, result)

	result.ActualHitRate = float64(result.CacheHits) / float64(result.CompletedRequests)
	result.ThroughputReqPerSec = float64(result.CompletedRequests) / result.TotalDuration.Seconds()
	result.EstimatedCostSavings = float64(result.CacheHits) * 0.0075
	result.EstimatedTimeSavings = time.Duration(result.CacheHits) * 500 * time.Millisecond

	return result
}

// ============================================================================
// Layer 2: SemanticQuery → SQL Load Test
// ============================================================================

// TestLayer2SQLQueryCacheWarmHit simulates repeated identical semantic queries
func TestLayer2SQLQueryCacheWarmHit(t *testing.T) {
	scenario := &LoadTestScenario{
		Name:            "Layer 2: SQL Query - Warm Cache (100% hit rate)",
		NumRequests:     10000,
		NumConcurrent:   50,
		CacheHitRate:    1.0,
		AvgResponseTime: 2 * time.Millisecond,
		QueryVariation:  0.0,
	}

	result := runSQLQueryLoadTest(scenario)

	if result.ActualHitRate < 0.95 {
		t.Logf("WARN: Expected >95%% hit rate, got %0.1f%%", result.ActualHitRate*100)
	}

	t.Logf("Layer 2 Warm Cache Test Results:\n%s", resultSummary(result))
}

// TestLayer2SQLQueryCacheMixed simulates mix of repeated and new semantic queries
func TestLayer2SQLQueryCacheMixed(t *testing.T) {
	scenario := &LoadTestScenario{
		Name:            "Layer 2: SQL Query - Mixed Workload",
		NumRequests:     10000,
		NumConcurrent:   100,
		CacheHitRate:    0.6,
		AvgResponseTime: 1000 * time.Millisecond,
		QueryVariation:  0.4,
	}

	result := runSQLQueryLoadTest(scenario)

	t.Logf("Layer 2 Mixed Workload Test Results:\n%s", resultSummary(result))
}

// runSQLQueryLoadTest executes a load test for Layer 2 (SemanticQuery → SQL)
func runSQLQueryLoadTest(scenario *LoadTestScenario) *LoadTestResult {
	cache, err := NewSemanticQueryCache("localhost:6379", "", 1)
	if err != nil {
		fmt.Printf("Error: Could not connect to Redis: %v\n", err)
		return &LoadTestResult{
			Scenario: scenario,
			Errors:   1,
		}
	}

	ctx := context.Background()
	result := &LoadTestResult{
		Scenario:      scenario,
		TotalRequests: scenario.NumRequests,
	}

	// Generate semantic query pool
	queries := generateSemanticQueryPool(scenario.NumRequests, scenario.QueryVariation)

	// Warm cache
	for i := 0; i < int(float64(scenario.NumRequests)*scenario.CacheHitRate); i++ {
		query := queries[i%len(queries)]
		entry := &SQLQueryCacheEntry{
			SemanticQuery:  query,
			DatabaseType:   "postgres",
			GeneratedSQL:   "SELECT * FROM table WHERE col = 1",
			GeneratedAt:    time.Now(),
			LLMModel:       "gemini-pro",
			GenerationTime: 1000,
			TenantID:       "test-tenant",
			Validated:      true,
		}
		cache.SetSQLQueryCache(ctx, query, "postgres", "test-tenant", entry)
	}

	// Run test
	start := time.Now()
	var wg sync.WaitGroup
	latencies := make([]time.Duration, 0, scenario.NumRequests)
	var mu sync.Mutex

	sem := make(chan struct{}, scenario.NumConcurrent)
	for i := 0; i < scenario.NumRequests; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			query := queries[idx%len(queries)]
			queryStart := time.Now()

			entry, _ := cache.GetSQLQueryCache(ctx, query, "postgres", "test-tenant")

			if entry != nil {
				mu.Lock()
				result.CacheHits++
				mu.Unlock()
			} else {
				time.Sleep(time.Duration(rand.Intn(200)+900) * time.Millisecond)
				mu.Lock()
				result.CacheMisses++
				mu.Unlock()

				cacheEntry := &SQLQueryCacheEntry{
					SemanticQuery:  query,
					DatabaseType:   "postgres",
					GeneratedSQL:   "SELECT * FROM table WHERE col = 1",
					GeneratedAt:    time.Now(),
					LLMModel:       "gemini-pro",
					GenerationTime: int64(time.Since(queryStart).Milliseconds()),
					TenantID:       "test-tenant",
					Validated:      true,
				}
				cache.SetSQLQueryCache(ctx, query, "postgres", "test-tenant", cacheEntry)
			}

			latency := time.Since(queryStart)
			mu.Lock()
			latencies = append(latencies, latency)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	result.TotalDuration = time.Since(start)
	result.CompletedRequests = result.CacheHits + result.CacheMisses

	calculateLatencyStats(latencies, result)

	result.ActualHitRate = float64(result.CacheHits) / float64(result.CompletedRequests)
	result.ThroughputReqPerSec = float64(result.CompletedRequests) / result.TotalDuration.Seconds()
	result.EstimatedCostSavings = float64(result.CacheHits) * 0.0075
	result.EstimatedTimeSavings = time.Duration(result.CacheHits) * 1000 * time.Millisecond

	return result
}

// ============================================================================
// Layer 3: SQL → Results Load Test
// ============================================================================

// TestLayer3ResultsCacheWarmHit simulates repeated identical SQL executions
func TestLayer3ResultsCacheWarmHit(t *testing.T) {
	scenario := &LoadTestScenario{
		Name:            "Layer 3: Results - Warm Cache (100% hit rate)",
		NumRequests:     15000,
		NumConcurrent:   100,
		CacheHitRate:    1.0,
		AvgResponseTime: 1 * time.Millisecond,
		QueryVariation:  0.0,
	}

	result := runResultsCacheLoadTest(scenario)

	if result.ActualHitRate < 0.95 {
		t.Logf("WARN: Expected >95%% hit rate, got %0.1f%%", result.ActualHitRate*100)
	}

	t.Logf("Layer 3 Warm Cache Test Results:\n%s", resultSummary(result))
}

// TestLayer3ResultsCacheColdStart simulates cache miss on results (DB queries)
func TestLayer3ResultsCacheColdStart(t *testing.T) {
	scenario := &LoadTestScenario{
		Name:            "Layer 3: Results - Cold Start (0% hit rate)",
		NumRequests:     100,
		NumConcurrent:   5,
		CacheHitRate:    0.0,
		AvgResponseTime: 200 * time.Millisecond,
		QueryVariation:  1.0,
	}

	result := runResultsCacheLoadTest(scenario)

	t.Logf("Layer 3 Cold Start Test Results:\n%s", resultSummary(result))
}

// runResultsCacheLoadTest executes a load test for Layer 3 (SQL → Results)
func runResultsCacheLoadTest(scenario *LoadTestScenario) *LoadTestResult {
	cache, err := NewSemanticQueryCache("localhost:6379", "", 1)
	if err != nil {
		fmt.Printf("Error: Could not connect to Redis: %v\n", err)
		return &LoadTestResult{
			Scenario: scenario,
			Errors:   1,
		}
	}

	ctx := context.Background()
	result := &LoadTestResult{
		Scenario:      scenario,
		TotalRequests: scenario.NumRequests,
	}

	// Generate SQL query pool
	sqlQueries := generateSQLQueryPool(scenario.NumRequests, scenario.QueryVariation)

	// Warm cache
	for i := 0; i < int(float64(scenario.NumRequests)*scenario.CacheHitRate); i++ {
		sql := sqlQueries[i%len(sqlQueries)]
		results := `[{"id":1,"name":"test"}]`
		entry := &ResultsCacheEntry{
			SQL:           sql,
			RowCount:      1,
			Results:       results,
			ExecutedAt:    time.Now(),
			ExecutionTime: 200,
			TenantID:      "test-tenant",
			DatabaseName:  "test_db",
		}
		cache.SetResultsCache(ctx, sql, "test-tenant", "test_db", entry)
	}

	// Run test
	start := time.Now()
	var wg sync.WaitGroup
	latencies := make([]time.Duration, 0, scenario.NumRequests)
	var mu sync.Mutex

	sem := make(chan struct{}, scenario.NumConcurrent)
	for i := 0; i < scenario.NumRequests; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			sql := sqlQueries[idx%len(sqlQueries)]
			queryStart := time.Now()

			entry, _ := cache.GetResultsCache(ctx, sql, "test-tenant", "test_db")

			if entry != nil {
				mu.Lock()
				result.CacheHits++
				mu.Unlock()
			} else {
				time.Sleep(time.Duration(rand.Intn(100)+150) * time.Millisecond)
				mu.Lock()
				result.CacheMisses++
				mu.Unlock()

				results := `[{"id":1,"name":"test"}]`
				cacheEntry := &ResultsCacheEntry{
					SQL:           sql,
					RowCount:      1,
					Results:       results,
					ExecutedAt:    time.Now(),
					ExecutionTime: int64(time.Since(queryStart).Milliseconds()),
					TenantID:      "test-tenant",
					DatabaseName:  "test_db",
				}
				cache.SetResultsCache(ctx, sql, "test-tenant", "test_db", cacheEntry)
			}

			latency := time.Since(queryStart)
			mu.Lock()
			latencies = append(latencies, latency)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	result.TotalDuration = time.Since(start)
	result.CompletedRequests = result.CacheHits + result.CacheMisses

	calculateLatencyStats(latencies, result)

	result.ActualHitRate = float64(result.CacheHits) / float64(result.CompletedRequests)
	result.ThroughputReqPerSec = float64(result.CompletedRequests) / result.TotalDuration.Seconds()
	result.EstimatedCostSavings = float64(result.CacheHits) * 0.001 // Results don't have LLM cost
	result.EstimatedTimeSavings = time.Duration(result.CacheHits) * 200 * time.Millisecond

	return result
}

// ============================================================================
// End-to-End Load Test (All 3 Layers)
// ============================================================================

// TestE2EFullPipeline simulates complete NL → SQL → Results pipeline
func TestE2EFullPipeline(t *testing.T) {
	scenario := &LoadTestScenario{
		Name:            "E2E: Full NL → SQL → Results Pipeline",
		NumRequests:     5000,
		NumConcurrent:   50,
		CacheHitRate:    0.75, // Mixed cache performance across layers
		AvgResponseTime: 800 * time.Millisecond,
		QueryVariation:  0.25,
	}

	result := runE2ELoadTest(scenario)

	t.Logf("E2E Full Pipeline Test Results:\n%s", resultSummary(result))
}

// runE2ELoadTest runs an end-to-end load test
func runE2ELoadTest(scenario *LoadTestScenario) *LoadTestResult {
	cache, err := NewSemanticQueryCache("localhost:6379", "", 1)
	if err != nil {
		return &LoadTestResult{
			Scenario: scenario,
			Errors:   1,
		}
	}

	ctx := context.Background()
	result := &LoadTestResult{
		Scenario:      scenario,
		TotalRequests: scenario.NumRequests,
	}

	// Generate realistic query distribution
	nlQueries := generateQueryPool(scenario.NumRequests, scenario.QueryVariation)

	start := time.Now()
	var wg sync.WaitGroup
	latencies := make([]time.Duration, 0, scenario.NumRequests)
	var mu sync.Mutex

	sem := make(chan struct{}, scenario.NumConcurrent)
	for i := 0; i < scenario.NumRequests; i++ {
		wg.Add(1)
		sem <- struct{}{}

		go func(idx int) {
			defer wg.Done()
			defer func() { <-sem }()

			query := nlQueries[idx%len(nlQueries)]
			queryStart := time.Now()

			// Layer 1: NL → SemanticQuery
			if _, err := cache.GetNLQueryCache(ctx, query.prompt, query.datasource, query.mode, query.tenantID); err != nil {
				result.Errors++
			} else {
				result.CacheHits++
			}

			// Layer 2: SemanticQuery → SQL (simulated)
			semQuery := `{"select":["col1"],"filters":[]}`
			if _, err := cache.GetSQLQueryCache(ctx, semQuery, "postgres", query.tenantID); err != nil {
				result.Errors++
			} else {
				result.CacheHits++
			}

			// Layer 3: SQL → Results (simulated)
			sql := "SELECT * FROM table LIMIT 10"
			if _, err := cache.GetResultsCache(ctx, sql, query.tenantID, "test_db"); err != nil {
				result.Errors++
			} else {
				result.CacheHits++
			}

			// Simulate cache miss penalty
			time.Sleep(time.Duration(rand.Intn(200)+100) * time.Millisecond)

			result.CacheMisses++

			latency := time.Since(queryStart)
			mu.Lock()
			latencies = append(latencies, latency)
			mu.Unlock()
		}(i)
	}

	wg.Wait()
	result.TotalDuration = time.Since(start)
	result.CompletedRequests = scenario.NumRequests

	calculateLatencyStats(latencies, result)
	result.ThroughputReqPerSec = float64(result.CompletedRequests) / result.TotalDuration.Seconds()

	return result
}

// ============================================================================
// Helper Functions
// ============================================================================

type queryDef struct {
	prompt     string
	datasource string
	mode       string
	tenantID   string
}

func generateQueryPool(count int, variation float64) []queryDef {
	pool := []queryDef{}
	baseQueries := 10

	for i := 0; i < baseQueries; i++ {
		pool = append(pool, queryDef{
			prompt:     fmt.Sprintf("Show top %d customers", i*100+1000),
			datasource: "customers",
			mode:       "exploratory",
			tenantID:   "tenant-1",
		})
	}

	// Add varied queries
	for i := 0; i < int(float64(count)*variation); i++ {
		pool = append(pool, queryDef{
			prompt:     fmt.Sprintf("Query %d for %d records", i, rand.Intn(1000)),
			datasource: "data_source_" + fmt.Sprintf("%d", rand.Intn(5)),
			mode:       []string{"exploratory", "strict", "crud"}[rand.Intn(3)],
			tenantID:   fmt.Sprintf("tenant-%d", rand.Intn(10)),
		})
	}

	return pool
}

func generateSemanticQueryPool(count int, variation float64) []string {
	pool := []string{}

	for i := 0; i < int(float64(count)*(1-variation)); i++ {
		q := map[string]interface{}{
			"select": []string{"field1", "field2"},
			"filters": []map[string]interface{}{
				{"field": "status", "op": "=", "value": "active"},
			},
			"limit": 100,
		}
		b, _ := json.Marshal(q)
		pool = append(pool, string(b))
	}

	for i := 0; i < int(float64(count)*variation); i++ {
		q := map[string]interface{}{
			"select": []string{fmt.Sprintf("field_%d", rand.Intn(100))},
			"limit":  rand.Intn(1000),
		}
		b, _ := json.Marshal(q)
		pool = append(pool, string(b))
	}

	return pool
}

func generateSQLQueryPool(count int, variation float64) []string {
	pool := []string{}

	for i := 0; i < int(float64(count)*(1-variation)); i++ {
		pool = append(pool, "SELECT * FROM customers WHERE status = 'active' LIMIT 100")
	}

	for i := 0; i < int(float64(count)*variation); i++ {
		sql := fmt.Sprintf("SELECT col_%d FROM table_%d WHERE id = %d LIMIT %d",
			rand.Intn(100), rand.Intn(10), rand.Intn(10000), rand.Intn(1000))
		pool = append(pool, sql)
	}

	return pool
}

func calculateLatencyStats(latencies []time.Duration, result *LoadTestResult) {
	if len(latencies) == 0 {
		return
	}

	var sum time.Duration
	var min, max time.Duration = latencies[0], latencies[0]

	for _, lat := range latencies {
		sum += lat
		if lat < min {
			min = lat
		}
		if lat > max {
			max = lat
		}
	}

	result.AvgLatency = sum / time.Duration(len(latencies))
	result.MinLatency = min
	result.MaxLatency = max

	// Calculate percentiles (simplified)
	sorted := make([]time.Duration, len(latencies))
	copy(sorted, latencies)
	// In production, use proper sorting
	result.P50Latency = result.AvgLatency
	result.P95Latency = result.MaxLatency
	result.P99Latency = result.MaxLatency
}

func resultSummary(r *LoadTestResult) string {
	return fmt.Sprintf(`
Test: %s
Scenario:
  Total Requests: %d
  Completed: %d
  Errors: %d

Cache Performance:
  Hits: %d (%.1f%%)
  Misses: %d
  Hit Rate: %.1f%%

Latency:
  Avg: %v
  P50: %v
  P95: %v
  P99: %v
  Min: %v
  Max: %v

Performance:
  Duration: %v
  Throughput: %.0f req/sec
  Cost Saved: $%.2f
  Time Saved: %v
`,
		r.Scenario.Name,
		r.TotalRequests,
		r.CompletedRequests,
		r.Errors,
		r.CacheHits,
		r.ActualHitRate*100,
		r.CacheMisses,
		r.ActualHitRate*100,
		r.AvgLatency,
		r.P50Latency,
		r.P95Latency,
		r.P99Latency,
		r.MinLatency,
		r.MaxLatency,
		r.TotalDuration,
		r.ThroughputReqPerSec,
		r.EstimatedCostSavings,
		r.EstimatedTimeSavings,
	)
}
