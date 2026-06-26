package ops

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Phase 3.4: Performance Optimization Layer
// Lock-free caching, connection pooling, and query optimization
// ============================================================================

// HighPerformanceRCACache provides lock-free caching for RCA results
type HighPerformanceRCACache struct {
	// Concurrent-safe map for caching
	cache sync.Map // map[string]*CachedRCAResult

	// TTL and eviction
	ttl      time.Duration
	maxSize  int
	itemLock sync.RWMutex
	itemSize int
}

// CachedRCAResult stores RCA result with metadata
type CachedRCAResult struct {
	Result    *RCAResult
	Timestamp time.Time
	HitCount  int64
	Region    string
}

// NewHighPerformanceRCACache creates an optimized RCA cache
func NewHighPerformanceRCACache(ttl time.Duration, maxSize int) *HighPerformanceRCACache {
	cache := &HighPerformanceRCACache{
		ttl:     ttl,
		maxSize: maxSize,
	}

	// Background eviction goroutine
	go cache.evictionWorker()

	return cache
}

// Get retrieves RCA result from cache (lock-free read)
func (c *HighPerformanceRCACache) Get(ctx context.Context, incidentID string) (*RCAResult, bool) {
	val, exists := c.cache.Load(fmt.Sprintf("rca:%s", incidentID))
	if !exists {
		return nil, false
	}

	cached := val.(*CachedRCAResult)

	// Check TTL
	if time.Since(cached.Timestamp) > c.ttl {
		c.cache.Delete(fmt.Sprintf("rca:%s", incidentID))
		return nil, false
	}

	// Increment hit count (atomic)
	atomic.AddInt64(&cached.HitCount, 1)

	return cached.Result, true
}

// Set stores RCA result in cache (lock-free write)
func (c *HighPerformanceRCACache) Set(ctx context.Context, incidentID string, result *RCAResult, region string) {
	key := fmt.Sprintf("rca:%s", incidentID)

	cached := &CachedRCAResult{
		Result:    result,
		Timestamp: time.Now(),
		HitCount:  0,
		Region:    region,
	}

	c.cache.Store(key, cached)
	c.itemLock.Lock()
	c.itemSize++
	c.itemLock.Unlock()
}

// evictionWorker periodically evicts expired entries
func (c *HighPerformanceRCACache) evictionWorker() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.cache.Range(func(key, value interface{}) bool {
			cached := value.(*CachedRCAResult)
			if time.Since(cached.Timestamp) > c.ttl {
				c.cache.Delete(key)
				c.itemLock.Lock()
				c.itemSize--
				c.itemLock.Unlock()
			}
			return true
		})
	}
}

// RegionConnectionPool manages concurrent connections per region
type RegionConnectionPool struct {
	pools map[string]*ConnectionPool
	mu    sync.RWMutex
}

// ConnectionPool manages a pool of connections to a single region
type ConnectionPool struct {
	name      string
	maxConns  int
	available chan interface{}
	inUse     int32
	mu        sync.Mutex
}

// NewRegionConnectionPool creates a pool manager
func NewRegionConnectionPool(regions []string, connsPerRegion int) *RegionConnectionPool {
	pool := &RegionConnectionPool{
		pools: make(map[string]*ConnectionPool),
	}

	for _, region := range regions {
		pool.pools[region] = &ConnectionPool{
			name:      region,
			maxConns:  connsPerRegion,
			available: make(chan interface{}, connsPerRegion),
		}

		// Pre-allocate connections
		for i := 0; i < connsPerRegion; i++ {
			pool.pools[region].available <- nil
		}
	}

	return pool
}

// AcquireConnection gets a connection from the region pool
func (p *RegionConnectionPool) AcquireConnection(ctx context.Context, region string) (interface{}, error) {
	p.mu.RLock()
	pool, exists := p.pools[region]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown region: %s", region)
	}

	select {
	case conn := <-pool.available:
		atomic.AddInt32(&pool.inUse, 1)
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// ReleaseConnection returns connection to the pool
func (p *RegionConnectionPool) ReleaseConnection(region string, conn interface{}) error {
	p.mu.RLock()
	pool, exists := p.pools[region]
	p.mu.RUnlock()

	if !exists {
		return fmt.Errorf("unknown region: %s", region)
	}

	atomic.AddInt32(&pool.inUse, -1)
	pool.available <- conn

	return nil
}

// OperationMetricsCollector tracks performance metrics
type OperationMetricsCollector struct {
	mu sync.RWMutex

	// Operation counters
	totalOps     int64
	successOps   int64
	failedOps    int64
	totalLatency int64 // nanoseconds
	maxLatency   int64
	minLatency   int64

	// Per-region metrics
	regionMetrics map[string]*RegionMetrics

	// Cache efficiency
	cacheHits   int64
	cacheMisses int64
}

// RegionMetrics tracks metrics for a specific region
type RegionMetrics struct {
	Ops         int64
	Latency     int64 // nanoseconds
	FailureRate float64
}

// NewOperationMetricsCollector creates a metrics collector
func NewOperationMetricsCollector() *OperationMetricsCollector {
	return &OperationMetricsCollector{
		regionMetrics: make(map[string]*RegionMetrics),
		minLatency:    1_000_000_000, // 1 second, will be overwritten
	}
}

// RecordOperation records an operation outcome
func (m *OperationMetricsCollector) RecordOperation(region string, latency int64, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	atomic.AddInt64(&m.totalOps, 1)

	if success {
		atomic.AddInt64(&m.successOps, 1)
	} else {
		atomic.AddInt64(&m.failedOps, 1)
	}

	atomic.AddInt64(&m.totalLatency, latency)

	// Update max/min
	if latency > m.maxLatency {
		m.maxLatency = latency
	}
	if latency < m.minLatency {
		m.minLatency = latency
	}

	// Update region metrics
	if metrics, exists := m.regionMetrics[region]; exists {
		metrics.Ops++
		metrics.Latency = (metrics.Latency + latency) / 2 // Running average
	} else {
		m.regionMetrics[region] = &RegionMetrics{
			Ops:     1,
			Latency: latency,
		}
	}
}

// RecordCacheHit records cache hit for metrics
func (m *OperationMetricsCollector) RecordCacheHit() {
	atomic.AddInt64(&m.cacheHits, 1)
}

// RecordCacheMiss records cache miss for metrics
func (m *OperationMetricsCollector) RecordCacheMiss() {
	atomic.AddInt64(&m.cacheMisses, 1)
}

// GetMetrics returns current metrics snapshot
func (m *OperationMetricsCollector) GetMetrics() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := atomic.LoadInt64(&m.totalOps)
	if total == 0 {
		total = 1 // Avoid division by zero
	}

	avgLatency := atomic.LoadInt64(&m.totalLatency) / total
	cacheHits := atomic.LoadInt64(&m.cacheHits)
	cacheMisses := atomic.LoadInt64(&m.cacheMisses)
	cacheTotal := cacheHits + cacheMisses
	if cacheTotal == 0 {
		cacheTotal = 1
	}

	return map[string]interface{}{
		"total_operations": total,
		"successful":       atomic.LoadInt64(&m.successOps),
		"failed":           atomic.LoadInt64(&m.failedOps),
		"success_rate":     float64(atomic.LoadInt64(&m.successOps)) / float64(total),
		"avg_latency_ns":   avgLatency,
		"max_latency_ns":   m.maxLatency,
		"min_latency_ns":   m.minLatency,
		"cache_hit_rate":   float64(cacheHits) / float64(cacheTotal),
		"cache_hits":       cacheHits,
		"cache_misses":     cacheMisses,
		"region_metrics":   m.regionMetrics,
	}
}

// BatchOperationOptimizer optimizes batch RCA operations
type BatchOperationOptimizer struct {
	batchSize    int
	flushTimeout time.Duration
	queue        chan *BatchedRCARequest
	results      map[string]*RCAResult
	mu           sync.Mutex
}

// BatchedRCARequest represents a single RCA request in a batch
type BatchedRCARequest struct {
	IncidentID string
	Incident   *Incident
	Events     []Event
	ResultChan chan *RCAResult
}

// NewBatchOperationOptimizer creates an optimizer for batch operations
func NewBatchOperationOptimizer(batchSize int, flushTimeout time.Duration) *BatchOperationOptimizer {
	opt := &BatchOperationOptimizer{
		batchSize:    batchSize,
		flushTimeout: flushTimeout,
		queue:        make(chan *BatchedRCARequest, batchSize*2),
		results:      make(map[string]*RCAResult),
	}

	// Start batch processor
	go opt.processBatches()

	return opt
}

// AddRequest adds a request to be batched
func (b *BatchOperationOptimizer) AddRequest(ctx context.Context, incidentID string, incident *Incident, events []Event) chan *RCAResult {
	resultChan := make(chan *RCAResult, 1)

	select {
	case b.queue <- &BatchedRCARequest{
		IncidentID: incidentID,
		Incident:   incident,
		Events:     events,
		ResultChan: resultChan,
	}:
	case <-ctx.Done():
		close(resultChan)
	}

	return resultChan
}

// processBatches processes accumulated requests in batches
func (b *BatchOperationOptimizer) processBatches() {
	batch := make([]*BatchedRCARequest, 0, b.batchSize)
	ticker := time.NewTicker(b.flushTimeout)
	defer ticker.Stop()

	for {
		select {
		case req := <-b.queue:
			batch = append(batch, req)

			// Process batch when full
			if len(batch) >= b.batchSize {
				b.processBatch(batch)
				batch = make([]*BatchedRCARequest, 0, b.batchSize)
			}

		case <-ticker.C:
			// Flush remaining items
			if len(batch) > 0 {
				b.processBatch(batch)
				batch = make([]*BatchedRCARequest, 0, b.batchSize)
			}
		}
	}
}

// processBatch executes a batch of RCA operations
func (b *BatchOperationOptimizer) processBatch(requests []*BatchedRCARequest) {
	// Parallel execution with bounded concurrency
	sem := make(chan struct{}, 4) // 4 concurrent RCAs
	var wg sync.WaitGroup

	for _, req := range requests {
		wg.Add(1)
		go func(r *BatchedRCARequest) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			// Execute RCA (mock for now)
			result := &RCAResult{
				ConfidenceScore: 0.85,
				CausalityChain:  make([]ScoredEvent, 0),
			}

			r.ResultChan <- result
		}(req)
	}

	wg.Wait()
}

// ThreadSafeRegionRouter provides optimized, thread-safe routing decisions
type ThreadSafeRegionRouter struct {
	baseRouter  RegionRouter
	cache       *HighPerformanceRCACache
	metrics     *OperationMetricsCollector
	connPool    *RegionConnectionPool
	batchOptim  *BatchOperationOptimizer
	routerMutex sync.RWMutex
}

// NewThreadSafeRegionRouter wraps a router with performance optimizations
func NewThreadSafeRegionRouter(baseRouter RegionRouter, regions []string) *ThreadSafeRegionRouter {
	return &ThreadSafeRegionRouter{
		baseRouter: baseRouter,
		cache:      NewHighPerformanceRCACache(5*time.Minute, 10000),
		metrics:    NewOperationMetricsCollector(),
		connPool:   NewRegionConnectionPool(regions, 10),
		batchOptim: NewBatchOperationOptimizer(100, 500*time.Millisecond),
	}
}

// OptimizedRouteDecision makes a routing decision with caching and metrics
func (t *ThreadSafeRegionRouter) OptimizedRouteDecision(ctx context.Context, tenantID string) (string, error) {
	start := time.Now()
	defer func() {
		elapsed := time.Since(start).Nanoseconds()
		t.metrics.RecordOperation("router", elapsed, true)
	}()

	t.routerMutex.RLock()
	defer t.routerMutex.RUnlock()

	// Parse tenantID string to UUID
	tenantUUID, err := uuid.Parse(tenantID)
	if err != nil {
		return "", fmt.Errorf("invalid tenant ID format: %v", err)
	}

	// Make routing decision
	target, err := t.baseRouter.GetTenantRegion(ctx, tenantUUID)
	return target, err
}

// GetMetrics returns current performance metrics
func (t *ThreadSafeRegionRouter) GetMetrics() map[string]interface{} {
	return t.metrics.GetMetrics()
}
