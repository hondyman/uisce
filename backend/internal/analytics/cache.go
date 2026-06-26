package analytics

import (
	"container/list"
	"crypto/sha256"
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
	"time"
)

// CacheEntry represents a cached governance context with versioning
type CacheEntry struct {
	GovernanceContext *GovernanceContext
	ClaimsVersion     int64
	PolicyVersion     int64
	CreatedAt         time.Time
	AccessedAt        int64 // Unix nano timestamp for atomic operations
	ExpiresAt         int64 // Unix nano timestamp, 0 for no expiration
	listElement       *list.Element
}

// ShardedCache provides high-performance, sharded caching for governance contexts
type ShardedCache struct {
	shards    []*cacheShard
	numShards int
	hashFunc  func(string) uint32
	stopChan  chan struct{}
}

type cacheShard struct {
	mu      sync.RWMutex
	entries map[string]*CacheEntry
	lruList *list.List
	size    int64
	maxSize int64
	hits    int64
	misses  int64
}

// GovernanceContext represents the governance context for query generation
type GovernanceContext struct {
	UserID            string
	TenantID          string
	Datasource        string
	AllowedMetrics    []string
	AllowedDimensions []string
	RequiredFilters   []QueryFilter
	AppliedPolicies   []AppliedGovernancePolicy
	AssetMappings     map[string]string
}

// AppliedGovernancePolicy represents a policy that was applied
type AppliedGovernancePolicy struct {
	ID     string
	RuleID string
	Action string
	Reason string
}

// QueryFilter represents a filter that must be applied
type QueryFilter struct {
	Field    string
	Operator string
	Value    interface{}
}

// NewShardedCache creates a new sharded cache with the specified number of shards
func NewShardedCache(numShards int, maxSizePerShard int64) *ShardedCache {
	if numShards <= 0 {
		numShards = 16 // Default to 16 shards
	}

	cache := &ShardedCache{
		shards:    make([]*cacheShard, numShards),
		numShards: numShards,
		hashFunc:  fnvHash,
		stopChan:  make(chan struct{}),
	}

	for i := 0; i < numShards; i++ {
		cache.shards[i] = &cacheShard{
			entries: make(map[string]*CacheEntry),
			lruList: list.New(),
			maxSize: maxSizePerShard,
		}
	}

	go cache.janitor(1 * time.Minute) // Run janitor every minute

	return cache
}

// Get retrieves a cached entry
func (sc *ShardedCache) Get(key string) (*CacheEntry, bool) {
	shard := sc.getShard(key)

	shard.mu.RLock()
	entry, found := shard.entries[key]
	shard.mu.RUnlock()

	if !found {
		atomic.AddInt64(&shard.misses, 1)
		return nil, false
	}

	// Item found, check for expiration.
	now := time.Now().UnixNano()
	if entry.ExpiresAt > 0 && now > entry.ExpiresAt {
		// Entry has expired. Remove it lazily.
		shard.mu.Lock()
		// Double-check it hasn't been removed by another goroutine or the janitor.
		shard.remove(key)
		shard.mu.Unlock()
		atomic.AddInt64(&shard.misses, 1) // Count as a miss
		return nil, false
	}

	shard.mu.Lock() // Acquire write lock to update LRU list
	shard.lruList.MoveToFront(entry.listElement)
	shard.mu.Unlock()

	atomic.AddInt64(&shard.hits, 1)
	atomic.StoreInt64(&entry.AccessedAt, time.Now().UnixNano()) // Atomic update is fine
	return entry, true
}

// Put stores an entry in the cache
func (sc *ShardedCache) Put(key string, entry *CacheEntry, ttl time.Duration) {
	shard := sc.getShard(key)

	if ttl > 0 {
		entry.ExpiresAt = time.Now().Add(ttl).UnixNano()
	} else {
		entry.ExpiresAt = 0 // No expiration
	}

	shard.mu.Lock()
	defer shard.mu.Unlock()

	// If entry already exists, move it to the front
	if existingEntry, ok := shard.entries[key]; ok {
		shard.lruList.MoveToFront(existingEntry.listElement)
		existingEntry.GovernanceContext = entry.GovernanceContext // Update content
		existingEntry.ExpiresAt = entry.ExpiresAt                 // Update expiration
		atomic.StoreInt64(&existingEntry.AccessedAt, time.Now().UnixNano())
		return
	}

	// Check if we need to evict entries
	if atomic.LoadInt64(&shard.size) >= shard.maxSize {
		sc.evictLRU(shard)
	}

	entry.AccessedAt = time.Now().UnixNano()
	element := shard.lruList.PushFront(key)
	entry.listElement = element
	shard.entries[key] = entry

	atomic.AddInt64(&shard.size, 1)
}

// Invalidate removes entries matching the given pattern
func (sc *ShardedCache) Invalidate(pattern func(string) bool) {
	for _, shard := range sc.shards {
		shard.mu.Lock()
		// Collect keys to remove to avoid modifying the map while iterating over it.
		keysToRemove := make([]string, 0)
		for key := range shard.entries {
			if pattern(key) {
				keysToRemove = append(keysToRemove, key)
			}
		}
		for _, key := range keysToRemove {
			shard.remove(key)
		}
		shard.mu.Unlock()
	}
}

// GetStats returns cache statistics
func (sc *ShardedCache) GetStats() map[string]interface{} {
	totalHits := int64(0)
	totalMisses := int64(0)
	totalSize := int64(0)
	shardStats := make([]map[string]interface{}, sc.numShards)

	for i, shard := range sc.shards {
		shard.mu.RLock()
		hits := atomic.LoadInt64(&shard.hits)
		misses := atomic.LoadInt64(&shard.misses)
		size := atomic.LoadInt64(&shard.size)
		shard.mu.RUnlock()

		totalHits += hits
		totalMisses += misses
		totalSize += size

		shardRequests := hits + misses
		shardHitRate := float64(0)
		if shardRequests > 0 {
			shardHitRate = float64(hits) / float64(shardRequests)
		}
		shardStats[i] = map[string]interface{}{
			"shard_id": i,
			"hits":     hits,
			"misses":   misses,
			"size":     size,
			"hit_rate": shardHitRate,
		}
	}

	totalRequests := totalHits + totalMisses
	hitRate := float64(0)
	if totalRequests > 0 {
		hitRate = float64(totalHits) / float64(totalRequests)
	}

	return map[string]interface{}{
		"total_hits":   totalHits,
		"total_misses": totalMisses,
		"total_size":   totalSize,
		"hit_rate":     hitRate,
		"num_shards":   sc.numShards,
		"shard_stats":  shardStats,
	}
}

// evictLRU removes the least recently used entry from a shard
func (sc *ShardedCache) evictLRU(shard *cacheShard) {
	// This is now an O(1) operation
	element := shard.lruList.Back()
	if element != nil {
		keyToRemove := shard.lruList.Remove(element).(string)
		shard.remove(keyToRemove)
	}
}

// remove removes an entry from a shard. The caller must hold the shard's write lock.
func (s *cacheShard) remove(key string) {
	if entry, ok := s.entries[key]; ok {
		s.lruList.Remove(entry.listElement)
		delete(s.entries, key)
		atomic.AddInt64(&s.size, -1)
	}
}

// Stop gracefully stops the cache's background processes (like the janitor).
func (sc *ShardedCache) Stop() {
	close(sc.stopChan)
}

// janitor periodically purges expired items from the cache.
func (sc *ShardedCache) janitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now().UnixNano()
			for _, shard := range sc.shards {
				shard.mu.Lock()
				for key, entry := range shard.entries {
					if entry.ExpiresAt > 0 && now > entry.ExpiresAt {
						shard.remove(key)
					}
				}
				shard.mu.Unlock()
			}
		case <-sc.stopChan:
			return
		}
	}
}

// getShard returns the shard for the given key
func (sc *ShardedCache) getShard(key string) *cacheShard {
	hash := sc.hashFunc(key)
	return sc.shards[hash%uint32(sc.numShards)]
}

// fnvHash computes a 32-bit FNV-1a hash
func fnvHash(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// generateCacheKey creates a stable cache key from tenant, user, and versions
func generateCacheKey(tenantID, userID string, claimsVersion, policyVersion int64) string {
	key := fmt.Sprintf("%s|%s|%d|%d", tenantID, userID, claimsVersion, policyVersion)
	hash := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", hash[:16]) // Use first 16 bytes for shorter key
}

// VersionManager manages version numbers for cache invalidation
type VersionManager struct {
	claimsVersions map[string]int64 // tenantID -> version
	policyVersions map[string]int64 // tenantID -> version
	mu             sync.RWMutex
}

// NewVersionManager creates a new version manager
func NewVersionManager() *VersionManager {
	return &VersionManager{
		claimsVersions: make(map[string]int64),
		policyVersions: make(map[string]int64),
	}
}

// GetClaimsVersion returns the current claims version for a tenant
func (vm *VersionManager) GetClaimsVersion(tenantID string) int64 {
	vm.mu.RLock()
	version := vm.claimsVersions[tenantID]
	vm.mu.RUnlock()
	return version
}

// GetPolicyVersion returns the current policy version for a tenant
func (vm *VersionManager) GetPolicyVersion(tenantID string) int64 {
	vm.mu.RLock()
	version := vm.policyVersions[tenantID]
	vm.mu.RUnlock()
	return version
}

// IncrementClaimsVersion increments the claims version for a tenant
func (vm *VersionManager) IncrementClaimsVersion(tenantID string) int64 {
	vm.mu.Lock()
	vm.claimsVersions[tenantID]++
	version := vm.claimsVersions[tenantID]
	vm.mu.Unlock()
	return version
}

// IncrementPolicyVersion increments the policy version for a tenant
func (vm *VersionManager) IncrementPolicyVersion(tenantID string) int64 {
	vm.mu.Lock()
	vm.policyVersions[tenantID]++
	version := vm.policyVersions[tenantID]
	vm.mu.Unlock()
	return version
}
