package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCache provides distributed caching capabilities with Redis
type RedisCache struct {
	client     *redis.Client
	localCache *ShardedCache
	ctx        context.Context
	ttl        time.Duration
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	TTL      time.Duration
}

// NewRedisCache creates a new Redis-backed cache with local fallback
func NewRedisCache(config RedisConfig, localCache *ShardedCache) *RedisCache {
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.Addr,
		Password: config.Password,
		DB:       config.DB,
	})

	// Test connection
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Printf("Warning: Redis connection failed, using local cache only: %v", err)
	}

	return &RedisCache{
		client:     rdb,
		localCache: localCache,
		ctx:        ctx,
		ttl:        config.TTL,
	}
}

// Get retrieves an entry from Redis or falls back to local cache
func (rc *RedisCache) Get(key string) (*CacheEntry, bool) {
	// Try Redis first
	if val, err := rc.client.Get(rc.ctx, key).Result(); err == nil {
		var entry CacheEntry
		if err := json.Unmarshal([]byte(val), &entry); err == nil {
			// Update local cache for faster subsequent access
			rc.localCache.Put(key, &entry, rc.ttl)
			return &entry, true
		}
	}

	// Fallback to local cache
	return rc.localCache.Get(key)
}

// Put stores an entry in both Redis and local cache
func (rc *RedisCache) Put(key string, entry *CacheEntry) {
	// Store in local cache first (fast)
	rc.localCache.Put(key, entry, rc.ttl)

	// Store in Redis asynchronously
	go func() {
		if data, err := json.Marshal(entry); err == nil {
			rc.client.Set(rc.ctx, key, data, rc.ttl).Err()
		}
	}()
}

// Invalidate removes entries from both caches
func (rc *RedisCache) Invalidate(pattern func(string) bool) {
	// Invalidate local cache
	rc.localCache.Invalidate(pattern)

	// For Redis, we'll need to scan and delete matching keys
	// This is more complex in Redis, so we'll use a simpler approach
	go func() {
		iter := rc.client.Scan(rc.ctx, 0, "*", 0).Iterator()
		for iter.Next(rc.ctx) {
			key := iter.Val()
			if pattern(key) {
				rc.client.Del(rc.ctx, key)
			}
		}
	}()
}

// GetStats returns combined statistics from both caches
func (cm *CacheManager) GetStats() map[string]interface{} {
	localStats := cm.localCache.GetStats()

	// Get Redis info if available
	redisStats := make(map[string]interface{})
	if cm.redisCache.client != nil {
		info, err := cm.redisCache.client.Info(cm.redisCache.ctx, "stats").Result()
		if err == nil {
			redisStats["redis_info"] = info
		} else {
			redisStats["redis_error"] = err.Error()
		}
	}

	// Combine stats
	combined := make(map[string]interface{})
	for k, v := range localStats {
		combined["local_"+k] = v
	}
	for k, v := range redisStats {
		combined["redis_"+k] = v
	}

	return combined
}

// Close closes the Redis connection
func (rc *RedisCache) Close() error {
	return rc.client.Close()
}

// DistributedLock provides distributed locking using Redis
type DistributedLock struct {
	client *redis.Client
	ctx    context.Context
}

// NewDistributedLock creates a new distributed lock manager
func NewDistributedLock(client *redis.Client) *DistributedLock {
	return &DistributedLock{
		client: client,
		ctx:    context.Background(),
	}
}

// AcquireLock attempts to acquire a distributed lock
func (dl *DistributedLock) AcquireLock(key string, ttl time.Duration) (bool, error) {
	return dl.client.SetNX(dl.ctx, "lock:"+key, "locked", ttl).Result()
}

// ReleaseLock releases a distributed lock
func (dl *DistributedLock) ReleaseLock(key string) error {
	return dl.client.Del(dl.ctx, "lock:"+key).Err()
}

// CacheManager orchestrates multiple cache layers
type CacheManager struct {
	redisCache *RedisCache
	localCache *ShardedCache
	versionMgr *VersionManager
	lockMgr    *DistributedLock
}

// NewCacheManager creates a new cache manager with multiple layers
func NewCacheManager(redisConfig RedisConfig) *CacheManager {
	localCache := NewShardedCache(16, 10000)
	redisCache := NewRedisCache(redisConfig, localCache)
	versionMgr := NewVersionManager()

	var lockMgr *DistributedLock
	if redisCache.client != nil {
		lockMgr = NewDistributedLock(redisCache.client)
	}

	return &CacheManager{
		redisCache: redisCache,
		localCache: localCache,
		versionMgr: versionMgr,
		lockMgr:    lockMgr,
	}
}

// GetGovernanceContext retrieves governance context with multi-layer caching
func (cm *CacheManager) GetGovernanceContext(tenantID, userID string) (*GovernanceContext, error) {
	claimsVersion := cm.versionMgr.GetClaimsVersion(tenantID)
	policyVersion := cm.versionMgr.GetPolicyVersion(tenantID)

	cacheKey := generateCacheKey(tenantID, userID, claimsVersion, policyVersion)

	if entry, found := cm.redisCache.Get(cacheKey); found {
		return entry.GovernanceContext, nil
	}

	// Context not found - would need to be generated by the governance service
	return nil, fmt.Errorf("governance context not found for tenant %s, user %s", tenantID, userID)
}

// UpdateGovernanceContext updates the governance context and invalidates related caches
func (cm *CacheManager) UpdateGovernanceContext(tenantID, userID string, context *GovernanceContext) error {
	claimsVersion := cm.versionMgr.IncrementClaimsVersion(tenantID)
	policyVersion := cm.versionMgr.GetPolicyVersion(tenantID)

	cacheKey := generateCacheKey(tenantID, userID, claimsVersion, policyVersion)

	entry := &CacheEntry{
		GovernanceContext: context,
		ClaimsVersion:     claimsVersion,
		PolicyVersion:     policyVersion,
		CreatedAt:         time.Now(),
	}

	cm.redisCache.Put(cacheKey, entry)
	return nil
}

// InvalidateTenantCaches invalidates all caches for a specific tenant
func (cm *CacheManager) InvalidateTenantCaches(tenantID string) error {
	// Acquire distributed lock to prevent race conditions
	if cm.lockMgr != nil {
		if acquired, _ := cm.lockMgr.AcquireLock("invalidate:"+tenantID, 30*time.Second); !acquired {
			return fmt.Errorf("failed to acquire invalidation lock for tenant %s", tenantID)
		}
		defer cm.lockMgr.ReleaseLock("invalidate:" + tenantID)
	}

	// Invalidate caches for this tenant
	cm.redisCache.Invalidate(func(key string) bool {
		// Parse tenant from cache key (assuming format: tenant|user|claims|policy)
		return len(key) > len(tenantID) && key[:len(tenantID)] == tenantID
	})

	// Increment version to invalidate existing entries
	cm.versionMgr.IncrementClaimsVersion(tenantID)

	return nil
}
