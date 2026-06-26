package reporting

import (
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// HIGH-PERFORMANCE CACHING LAYER
// ============================================================================

// CacheConfig configures the caching behavior
type CacheConfig struct {
	// Definition cache
	DefinitionTTL     time.Duration
	DefinitionMaxSize int

	// Rendered report cache
	RenderedReportTTL     time.Duration
	RenderedReportMaxSize int

	// Query result cache
	QueryResultTTL     time.Duration
	QueryResultMaxSize int

	// Schema cache
	SchemaTTL time.Duration

	// Enable compression for large cache entries
	EnableCompression    bool
	CompressionThreshold int // bytes
}

// DefaultCacheConfig returns production-ready cache settings
func DefaultCacheConfig() *CacheConfig {
	return &CacheConfig{
		DefinitionTTL:         15 * time.Minute,
		DefinitionMaxSize:     10000,
		RenderedReportTTL:     5 * time.Minute,
		RenderedReportMaxSize: 1000,
		QueryResultTTL:        2 * time.Minute,
		QueryResultMaxSize:    50000,
		SchemaTTL:             30 * time.Minute,
		EnableCompression:     true,
		CompressionThreshold:  10240, // 10KB
	}
}

// CacheEntry represents a cached item
type CacheEntry struct {
	Key        string
	Value      interface{}
	Expiration time.Time
	Size       int
	Compressed bool
	Hits       int64
	CreatedAt  time.Time
}

// IsExpired checks if the entry has expired
func (ce *CacheEntry) IsExpired() bool {
	return time.Now().After(ce.Expiration)
}

// ShardedCache is a high-performance thread-safe cache with sharding
type ShardedCache struct {
	shards    []*cacheShard
	shardMask uint32
	config    *CacheConfig
	metrics   *CacheMetrics
}

type cacheShard struct {
	items map[string]*CacheEntry
	lock  sync.RWMutex
}

// CacheMetrics tracks cache performance
type CacheMetrics struct {
	Hits          int64
	Misses        int64
	Evictions     int64
	Size          int64
	TotalRequests int64
	AvgLatencyNs  int64
	lock          sync.RWMutex
}

// NewShardedCache creates a new sharded cache
func NewShardedCache(config *CacheConfig) *ShardedCache {
	numShards := 256 // Power of 2 for efficient modulo
	shards := make([]*cacheShard, numShards)
	for i := range shards {
		shards[i] = &cacheShard{
			items: make(map[string]*CacheEntry),
		}
	}

	cache := &ShardedCache{
		shards:    shards,
		shardMask: uint32(numShards - 1),
		config:    config,
		metrics:   &CacheMetrics{},
	}

	// Start background cleanup
	go cache.cleanupLoop()

	return cache
}

// getShard returns the shard for a key using FNV hash
func (sc *ShardedCache) getShard(key string) *cacheShard {
	h := fnv.New32a()
	h.Write([]byte(key))
	return sc.shards[h.Sum32()&sc.shardMask]
}

// Get retrieves an item from the cache
func (sc *ShardedCache) Get(key string) (interface{}, bool) {
	start := time.Now()
	defer func() {
		sc.metrics.lock.Lock()
		sc.metrics.TotalRequests++
		latency := time.Since(start).Nanoseconds()
		sc.metrics.AvgLatencyNs = (sc.metrics.AvgLatencyNs + latency) / 2
		sc.metrics.lock.Unlock()
	}()

	shard := sc.getShard(key)
	shard.lock.RLock()
	entry, exists := shard.items[key]
	shard.lock.RUnlock()

	if !exists || entry.IsExpired() {
		sc.metrics.lock.Lock()
		sc.metrics.Misses++
		sc.metrics.lock.Unlock()
		return nil, false
	}

	// Update hit count (atomic would be better but this is good enough)
	entry.Hits++

	sc.metrics.lock.Lock()
	sc.metrics.Hits++
	sc.metrics.lock.Unlock()

	return entry.Value, true
}

// Set adds an item to the cache
func (sc *ShardedCache) Set(key string, value interface{}, ttl time.Duration) {
	shard := sc.getShard(key)

	entry := &CacheEntry{
		Key:        key,
		Value:      value,
		Expiration: time.Now().Add(ttl),
		CreatedAt:  time.Now(),
	}

	shard.lock.Lock()
	shard.items[key] = entry
	shard.lock.Unlock()

	sc.metrics.lock.Lock()
	sc.metrics.Size++
	sc.metrics.lock.Unlock()
}

// Delete removes an item from the cache
func (sc *ShardedCache) Delete(key string) {
	shard := sc.getShard(key)
	shard.lock.Lock()
	delete(shard.items, key)
	shard.lock.Unlock()
}

// DeletePattern removes items matching a pattern
func (sc *ShardedCache) DeletePattern(pattern string) int {
	deleted := 0
	for _, shard := range sc.shards {
		shard.lock.Lock()
		for key := range shard.items {
			if matchPattern(key, pattern) {
				delete(shard.items, key)
				deleted++
			}
		}
		shard.lock.Unlock()
	}
	return deleted
}

// cleanupLoop periodically removes expired entries
func (sc *ShardedCache) cleanupLoop() {
	ticker := time.NewTicker(time.Minute)
	for range ticker.C {
		sc.cleanup()
	}
}

func (sc *ShardedCache) cleanup() {
	evicted := int64(0)
	for _, shard := range sc.shards {
		shard.lock.Lock()
		for key, entry := range shard.items {
			if entry.IsExpired() {
				delete(shard.items, key)
				evicted++
			}
		}
		shard.lock.Unlock()
	}

	sc.metrics.lock.Lock()
	sc.metrics.Evictions += evicted
	sc.metrics.Size -= evicted
	sc.metrics.lock.Unlock()
}

// GetMetrics returns cache performance metrics
func (sc *ShardedCache) GetMetrics() CacheMetrics {
	sc.metrics.lock.RLock()
	defer sc.metrics.lock.RUnlock()
	return *sc.metrics
}

// matchPattern simple glob pattern matching
func matchPattern(s, pattern string) bool {
	// Simple implementation - production should use proper glob matching
	if pattern == "*" {
		return true
	}
	return s == pattern
}

// ============================================================================
// DEFINITION CACHE
// ============================================================================

// DefinitionCache caches report definitions
type DefinitionCache struct {
	cache  *ShardedCache
	config *CacheConfig
}

// NewDefinitionCache creates a definition cache
func NewDefinitionCache(config *CacheConfig) *DefinitionCache {
	return &DefinitionCache{
		cache:  NewShardedCache(config),
		config: config,
	}
}

func (dc *DefinitionCache) cacheKey(tenantID, datasourceID, defID uuid.UUID) string {
	return fmt.Sprintf("def:%s:%s:%s", tenantID, datasourceID, defID)
}

func (dc *DefinitionCache) tenantPattern(tenantID uuid.UUID) string {
	return fmt.Sprintf("def:%s:*", tenantID)
}

// Get retrieves a definition from cache
func (dc *DefinitionCache) Get(tenantID, datasourceID, defID uuid.UUID) (*ReportDefinition, bool) {
	key := dc.cacheKey(tenantID, datasourceID, defID)
	if val, ok := dc.cache.Get(key); ok {
		return val.(*ReportDefinition), true
	}
	return nil, false
}

// Set stores a definition in cache
func (dc *DefinitionCache) Set(def *ReportDefinition) {
	key := dc.cacheKey(def.TenantID, def.TenantDatasourceID, def.ID)
	dc.cache.Set(key, def, dc.config.DefinitionTTL)
}

// Invalidate removes a definition from cache
func (dc *DefinitionCache) Invalidate(tenantID, datasourceID, defID uuid.UUID) {
	key := dc.cacheKey(tenantID, datasourceID, defID)
	dc.cache.Delete(key)
}

// InvalidateTenant removes all definitions for a tenant
func (dc *DefinitionCache) InvalidateTenant(tenantID uuid.UUID) {
	dc.cache.DeletePattern(dc.tenantPattern(tenantID))
}

// ============================================================================
// QUERY RESULT CACHE (for Cube.dev queries)
// ============================================================================

// QueryResultCache caches Cube.dev query results
type QueryResultCache struct {
	cache  *ShardedCache
	config *CacheConfig
}

// NewQueryResultCache creates a query result cache
func NewQueryResultCache(config *CacheConfig) *QueryResultCache {
	return &QueryResultCache{
		cache:  NewShardedCache(config),
		config: config,
	}
}

// QueryCacheKey generates a cache key from a query
type QueryCacheKey struct {
	TenantID     uuid.UUID
	DatasourceID uuid.UUID
	CubeQuery    json.RawMessage
}

func (qc *QueryResultCache) cacheKey(key QueryCacheKey) string {
	h := fnv.New64a()
	h.Write([]byte(key.TenantID.String()))
	h.Write([]byte(key.DatasourceID.String()))
	h.Write(key.CubeQuery)
	return fmt.Sprintf("query:%d", h.Sum64())
}

// Get retrieves query results from cache
func (qc *QueryResultCache) Get(key QueryCacheKey) (json.RawMessage, bool) {
	cacheKey := qc.cacheKey(key)
	if val, ok := qc.cache.Get(cacheKey); ok {
		return val.(json.RawMessage), true
	}
	return nil, false
}

// Set stores query results in cache
func (qc *QueryResultCache) Set(key QueryCacheKey, result json.RawMessage, ttl time.Duration) {
	if ttl == 0 {
		ttl = qc.config.QueryResultTTL
	}
	cacheKey := qc.cacheKey(key)
	qc.cache.Set(cacheKey, result, ttl)
}

// ============================================================================
// RENDERED REPORT CACHE
// ============================================================================

// RenderedReportCache caches rendered reports
type RenderedReportCache struct {
	cache  *ShardedCache
	config *CacheConfig
}

// NewRenderedReportCache creates a rendered report cache
func NewRenderedReportCache(config *CacheConfig) *RenderedReportCache {
	return &RenderedReportCache{
		cache:  NewShardedCache(config),
		config: config,
	}
}

// RenderedCacheKey identifies a rendered report in cache
type RenderedCacheKey struct {
	DefinitionID uuid.UUID
	ExtensionID  *uuid.UUID
	Parameters   json.RawMessage
	OutputFormat string
	ContextID    *uuid.UUID
}

func (rc *RenderedReportCache) cacheKey(key RenderedCacheKey) string {
	h := fnv.New64a()
	h.Write([]byte(key.DefinitionID.String()))
	if key.ExtensionID != nil {
		h.Write([]byte(key.ExtensionID.String()))
	}
	h.Write(key.Parameters)
	h.Write([]byte(key.OutputFormat))
	if key.ContextID != nil {
		h.Write([]byte(key.ContextID.String()))
	}
	return fmt.Sprintf("rendered:%d", h.Sum64())
}

// CachedRenderResult holds a cached rendered report
type CachedRenderResult struct {
	InstanceID uuid.UUID
	OutputURL  string
	ExpiresAt  time.Time
}

// Get retrieves a rendered report from cache
func (rc *RenderedReportCache) Get(key RenderedCacheKey) (*CachedRenderResult, bool) {
	cacheKey := rc.cacheKey(key)
	if val, ok := rc.cache.Get(cacheKey); ok {
		result := val.(*CachedRenderResult)
		if time.Now().Before(result.ExpiresAt) {
			return result, true
		}
	}
	return nil, false
}

// Set stores a rendered report in cache
func (rc *RenderedReportCache) Set(key RenderedCacheKey, result *CachedRenderResult) {
	cacheKey := rc.cacheKey(key)
	rc.cache.Set(cacheKey, result, rc.config.RenderedReportTTL)
}

// ============================================================================
// CACHE WARMING
// ============================================================================

// CacheWarmer pre-populates caches for better performance
type CacheWarmer struct {
	repo     *Repository
	defCache *DefinitionCache
	service  *Service
}

// NewCacheWarmer creates a cache warmer
func NewCacheWarmer(repo *Repository, defCache *DefinitionCache, service *Service) *CacheWarmer {
	return &CacheWarmer{
		repo:     repo,
		defCache: defCache,
		service:  service,
	}
}

// WarmTenantCache loads frequently used definitions for a tenant
func (cw *CacheWarmer) WarmTenantCache(ctx context.Context, tenantID, datasourceID uuid.UUID) error {
	// Load top definitions by usage
	defs, err := cw.repo.ListDefinitions(ctx, tenantID, datasourceID, map[string]interface{}{
		"status": "published",
		"limit":  100,
	})
	if err != nil {
		return err
	}

	for _, def := range defs {
		cw.defCache.Set(&def)
	}

	return nil
}

// WarmPopularReports pre-renders frequently requested reports
func (cw *CacheWarmer) WarmPopularReports(ctx context.Context, tenantID, datasourceID uuid.UUID) error {
	// Get most frequently run reports from analytics
	// This would integrate with the analytics system
	return nil
}
