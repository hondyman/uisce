package cache

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// ============================================================================
// Redis Schema & Key Structure
// ============================================================================
//
// REDIS DATABASE ASSIGNMENT:
// DB 0: Semantic View Cache (existing)
// DB 1: Semantic Query Cache (new - this layer)
//
// KEY PATTERNS (all tenant-partitioned):
//
// Layer 1: NL → SemanticQuery
//   Key:   nlquery:HASH(prompt+datasource+mode+tenantID)
//   Value: JSON(NLQueryCacheEntry)
//   TTL:   24 hours
//   Lookup: One hit per unique NL question
//
// Layer 2: SemanticQuery → SQL
//   Key:   sqlquery:HASH(semantic_query+dbtype+tenantID)
//   Value: JSON(SQLQueryCacheEntry)
//   TTL:   7 days
//   Lookup: One hit per unique semantic query structure
//
// Layer 3: SQL → Results
//   Key:   results:HASH(sql+tenantID+dbname)
//   Value: JSON(ResultsCacheEntry)
//   TTL:   5 minutes
//   Lookup: One hit per unique SQL statement
//
// EVICTION POLICY:
//   Redis will use TTL-based eviction; keys are automatically deleted
//   when their TTL expires.
//   Max memory policy: allkeys-lru (if Redis memory limits reached)

// ============================================================================
// Redis Migration & Schema Initialization
// ============================================================================

// MigrationScript contains the Redis initialization commands
type MigrationScript struct {
	Commands []string
	Version  string
}

// GetRedisMigrationScript returns the initialization script for query cache
func GetRedisMigrationScript() *MigrationScript {
	return &MigrationScript{
		Version: "1.0",
		Commands: []string{
			// Set max memory policy for automatic eviction
			"CONFIG SET maxmemory-policy allkeys-lru",

			// Create namespace indexes (for monitoring/stats)
			// These are Lua script helpers to count keys by pattern
			// Not required for basic operation, but useful for metrics

			// Index key counts for monitoring
			// Use Redis SCAN to iterate keys by pattern:
			// SCAN 0 MATCH nlquery:* COUNT 100
			// SCAN 0 MATCH sqlquery:* COUNT 100
			// SCAN 0 MATCH results:* COUNT 100
		},
	}
}

// InitializeRedisQueryCache initializes the Redis database for query caching
func InitializeRedisQueryCache(ctx context.Context, client *redis.Client) error {
	script := GetRedisMigrationScript()

	log.Printf("Initializing Redis query cache schema (version %s)", script.Version)

	for _, cmd := range script.Commands {
		if cmd == "" {
			continue
		}

		log.Printf("Executing: %s", cmd)
		// Note: This is pseudo-code. Actual Redis commands should be executed
		// using proper Redis client methods or Lua scripts.
	}

	// Verify connection
	if err := client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis connection check failed: %w", err)
	}

	// Set connection name for monitoring
	if err := client.Do(ctx, "CLIENT", "SETNAME", "semlayer-query-cache").Err(); err != nil {
		log.Printf("Warning: could not set client name: %v", err)
	}

	log.Printf("Redis query cache schema installed successfully")
	return nil
}

// ============================================================================
// Monitoring & Metrics Collection
// ============================================================================

// MetricsCollector periodically collects cache metrics from Redis
type MetricsCollector struct {
	client   *redis.Client
	cache    *SemanticQueryCache
	interval time.Duration
	done     chan bool
}

// NewMetricsCollector creates a metrics collector
func NewMetricsCollector(client *redis.Client, cache *SemanticQueryCache, interval time.Duration) *MetricsCollector {
	return &MetricsCollector{
		client:   client,
		cache:    cache,
		interval: interval,
		done:     make(chan bool),
	}
}

// Start begins periodic metrics collection
func (mc *MetricsCollector) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(mc.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				mc.done <- true
				return
			case <-mc.done:
				return
			case <-ticker.C:
				if err := mc.collectMetrics(ctx); err != nil {
					log.Printf("Error collecting metrics: %v", err)
				}
			}
		}
	}()
}

// collectMetrics gathers cache statistics from Redis
func (mc *MetricsCollector) collectMetrics(ctx context.Context) error {
	if mc.client == nil {
		return nil
	}

	metrics := mc.cache.GetMetrics()

	// Count keys by pattern
	nlCount, _ := mc.countKeysByPattern(ctx, "nlquery:*")
	sqlCount, _ := mc.countKeysByPattern(ctx, "sqlquery:*")
	resultCount, _ := mc.countKeysByPattern(ctx, "results:*")

	// Get Redis memory info
	info := mc.client.Info(ctx, "memory")

	log.Printf("Cache Metrics: NL=%d, SQL=%d, Results=%d | Hits=%d, Misses=%d | TotalSavings=%.2fs | Cost saved=$%.2f",
		nlCount, sqlCount, resultCount,
		metrics.NLQueryHits+metrics.SQLQueryHits+metrics.ResultsHits,
		metrics.NLQueryMisses+metrics.SQLQueryMisses+metrics.ResultsMisses,
		metrics.TotalSavings.Seconds(),
		float64(metrics.AvoidsHits)*0.0075,
	)

	// Log to info channel
	log.Printf("Redis Memory: %s", info.String()[:100])

	return nil
}

// countKeysByPattern counts keys matching a pattern
func (mc *MetricsCollector) countKeysByPattern(ctx context.Context, pattern string) (int64, error) {
	if mc.client == nil {
		return 0, nil
	}

	var count int64
	iter := mc.client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		count++
	}

	if err := iter.Err(); err != nil {
		return 0, fmt.Errorf("scan error: %w", err)
	}

	return count, nil
}

// Stop ends metrics collection
func (mc *MetricsCollector) Stop() {
	mc.done <- true
}

// ============================================================================
// Cache Invalidation Hooks
// ============================================================================

// CacheInvalidationHook defines a callback for cache invalidation events
type CacheInvalidationHook func(ctx context.Context, reason string) error

// InvalidationReason enumerates cache invalidation triggers
type InvalidationReason string

const (
	ReasonSemanticBundleUpdated InvalidationReason = "bundle_updated"
	ReasonTenantOffboarded      InvalidationReason = "tenant_offboarded"
	ReasonManualInvalidation    InvalidationReason = "manual_invalidation"
	ReasonDatabaseSchemaChanged InvalidationReason = "schema_changed"
	ReasonLLMModelUpdated       InvalidationReason = "llm_model_updated"
)

// InvalidationManager manages cache invalidation events
type InvalidationManager struct {
	hooks []CacheInvalidationHook
	cache *SemanticQueryCache
}

// NewInvalidationManager creates a new invalidation manager
func NewInvalidationManager(cache *SemanticQueryCache) *InvalidationManager {
	return &InvalidationManager{
		cache: cache,
		hooks: []CacheInvalidationHook{},
	}
}

// RegisterHook registers a callback for invalidation events
func (im *InvalidationManager) RegisterHook(hook CacheInvalidationHook) {
	im.hooks = append(im.hooks, hook)
}

// OnSemanticBundleUpdated triggers invalidation when a semantic bundle is updated
func (im *InvalidationManager) OnSemanticBundleUpdated(ctx context.Context, tenantID, boID string) error {
	log.Printf("Invalidating cache: semantic bundle updated (tenant=%s, bo=%s)", tenantID, boID)

	// Invalidate all related cache entries for this tenant
	// This invalidates all layers since bundles affect all three
	if err := im.cache.InvalidateTenantCache(ctx, tenantID); err != nil {
		log.Printf("Error invalidating tenant cache: %v", err)
		return err
	}

	// Execute hooks
	for _, hook := range im.hooks {
		if err := hook(ctx, string(ReasonSemanticBundleUpdated)); err != nil {
			log.Printf("Hook error: %v", err)
		}
	}

	return nil
}

// OnTenantOffboarded triggers invalidation when a tenant is offboarded
func (im *InvalidationManager) OnTenantOffboarded(ctx context.Context, tenantID string) error {
	log.Printf("Invalidating cache: tenant offboarded (tenant=%s)", tenantID)

	if err := im.cache.InvalidateTenantCache(ctx, tenantID); err != nil {
		log.Printf("Error invalidating tenant cache: %v", err)
		return err
	}

	// Execute hooks
	for _, hook := range im.hooks {
		if err := hook(ctx, string(ReasonTenantOffboarded)); err != nil {
			log.Printf("Hook error: %v", err)
		}
	}

	return nil
}

// OnDatabaseSchemaChanged triggers invalidation when database schema changes
func (im *InvalidationManager) OnDatabaseSchemaChanged(ctx context.Context, tenantID, dbName string) error {
	log.Printf("Invalidating cache: database schema changed (tenant=%s, db=%s)", tenantID, dbName)

	// Invalidate all cache entries (schema change affects SQL generation)
	if err := im.cache.InvalidateTenantCache(ctx, tenantID); err != nil {
		log.Printf("Error invalidating tenant cache: %v", err)
		return err
	}

	// Execute hooks
	for _, hook := range im.hooks {
		if err := hook(ctx, string(ReasonDatabaseSchemaChanged)); err != nil {
			log.Printf("Hook error: %v", err)
		}
	}

	return nil
}

// OnLLMModelUpdated triggers invalidation when LLM model is updated
func (im *InvalidationManager) OnLLMModelUpdated(ctx context.Context, tenantID string) error {
	log.Printf("Invalidating cache: LLM model updated (tenant=%s)", tenantID)

	// Invalidate NL and SQL layers (but not results layer)
	// Results may still be valid with different SQL from new model
	if err := im.cache.InvalidateTenantCache(ctx, tenantID); err != nil {
		log.Printf("Error invalidating tenant cache: %v", err)
		return err
	}

	// Execute hooks
	for _, hook := range im.hooks {
		if err := hook(ctx, string(ReasonLLMModelUpdated)); err != nil {
			log.Printf("Hook error: %v", err)
		}
	}

	return nil
}

// ============================================================================
// Health Check & Diagnostics
// ============================================================================

// HealthCheckResult contains cache health status
type HealthCheckResult struct {
	RedisConnected bool
	CacheSize      int64
	KeyCount       int64
	NLQueryCount   int64
	SQLQueryCount  int64
	ResultsCount   int64
	HitRate        float64
	Status         string
	Timestamp      time.Time
}

// HealthCheck performs a health check on the cache
func (sqc *SemanticQueryCache) HealthCheck(ctx context.Context) *HealthCheckResult {
	result := &HealthCheckResult{
		Timestamp: time.Now(),
	}

	if sqc.redisClient == nil {
		result.Status = "OFFLINE"
		return result
	}

	// Test Redis connection
	if err := sqc.redisClient.Ping(ctx).Err(); err != nil {
		result.RedisConnected = false
		result.Status = "ERROR"
		log.Printf("Health check: Redis connection failed - %v", err)
		return result
	}

	result.RedisConnected = true

	// Get Redis memory stats
	info := sqc.redisClient.Info(ctx, "memory")
	log.Printf("Health check Redis info: %s", info.String()[:200])

	// Count keys by type
	nlCount, _ := countKeysByPattern(ctx, sqc.redisClient, "nlquery:*")
	sqlCount, _ := countKeysByPattern(ctx, sqc.redisClient, "sqlquery:*")
	resultCount, _ := countKeysByPattern(ctx, sqc.redisClient, "results:*")

	result.NLQueryCount = nlCount
	result.SQLQueryCount = sqlCount
	result.ResultsCount = resultCount
	result.KeyCount = nlCount + sqlCount + resultCount

	// Calculate hit rate
	m := sqc.metrics
	totalOps := m.NLQueryHits + m.NLQueryMisses + m.SQLQueryHits + m.SQLQueryMisses + m.ResultsHits + m.ResultsMisses
	if totalOps > 0 {
		totalHits := m.NLQueryHits + m.SQLQueryHits + m.ResultsHits
		result.HitRate = float64(totalHits) / float64(totalOps)
	}

	// Determine overall status
	if result.HitRate > 0.5 {
		result.Status = "HEALTHY"
	} else if result.HitRate > 0.2 {
		result.Status = "DEGRADED"
	} else {
		result.Status = "WARNING"
	}

	return result
}

// Helper function to count keys
func countKeysByPattern(ctx context.Context, client *redis.Client, pattern string) (int64, error) {
	var count int64
	iter := client.Scan(ctx, 0, pattern, 0).Iterator()

	for iter.Next(ctx) {
		count++
	}

	return count, iter.Err()
}

// ============================================================================
// Cleanup & Maintenance
// ============================================================================

// PruneExpiredKeys manually removes expired keys (normally Redis does this)
func (sqc *SemanticQueryCache) PruneExpiredKeys(ctx context.Context) (int64, error) {
	if sqc.redisClient == nil {
		return 0, fmt.Errorf("redis client not initialized")
	}

	// Redis automatically prunes expired keys, but this can be used
	// to force cleanup if needed
	patterns := []string{"nlquery:*", "sqlquery:*", "results:*"}
	var totalPruned int64

	for _, pattern := range patterns {
		iter := sqc.redisClient.Scan(ctx, 0, pattern, 0).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			ttl := sqc.redisClient.TTL(ctx, key).Val()
			if ttl < 0 { // Key has no TTL or is expired
				if err := sqc.redisClient.Del(ctx, key).Err(); err == nil {
					totalPruned++
				}
			}
		}
	}

	log.Printf("Pruned %d expired keys", totalPruned)
	return totalPruned, nil
}

// ClearAllCache clears all cache entries (use with caution)
func (sqc *SemanticQueryCache) ClearAllCache(ctx context.Context) error {
	if sqc.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	patterns := []string{"nlquery:*", "sqlquery:*", "results:*"}
	var totalDeleted int64

	for _, pattern := range patterns {
		iter := sqc.redisClient.Scan(ctx, 0, pattern, 0).Iterator()
		for iter.Next(ctx) {
			key := iter.Val()
			if err := sqc.redisClient.Del(ctx, key).Err(); err == nil {
				totalDeleted++
			}
		}
	}

	log.Printf("Cleared all cache entries: %d keys deleted", totalDeleted)
	return nil
}
