package discovery

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CacheEntry holds cached data with expiration
type CacheEntry struct {
	Value     interface{}
	ExpiresAt time.Time
	CreatedAt time.Time
}

// QueryCache provides thread-safe query result caching
type QueryCache struct {
	mu      sync.RWMutex
	cache   map[string]*CacheEntry
	maxSize int
	ttl     time.Duration
	hits    int64
	misses  int64
}

// NewQueryCache creates a new query cache with specified TTL
func NewQueryCache(ttl time.Duration, maxSize int) *QueryCache {
	qc := &QueryCache{
		cache:   make(map[string]*CacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}

	// Start cleanup goroutine
	go qc.cleanup()

	return qc
}

// Set stores a value in cache with automatic expiration
func (qc *QueryCache) Set(query string, value interface{}) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	// If cache is full, evict oldest entry
	if len(qc.cache) >= qc.maxSize {
		qc.evictOldest()
	}

	key := qc.hashQuery(query)
	qc.cache[key] = &CacheEntry{
		Value:     value,
		ExpiresAt: time.Now().Add(qc.ttl),
		CreatedAt: time.Now(),
	}
}

// Get retrieves a value from cache if not expired
func (qc *QueryCache) Get(query string) (interface{}, bool) {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	key := qc.hashQuery(query)
	entry, exists := qc.cache[key]

	if !exists {
		qc.misses++
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		qc.misses++
		return nil, false
	}

	qc.hits++
	return entry.Value, true
}

// Invalidate removes a cached query
func (qc *QueryCache) Invalidate(query string) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	key := qc.hashQuery(query)
	delete(qc.cache, key)
}

// InvalidatePattern removes all cache entries matching a pattern
func (qc *QueryCache) InvalidatePattern(pattern string) {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	for key := range qc.cache {
		// Simple pattern matching on key (which is hash of query)
		// In production, could use more sophisticated pattern matching
		if pattern == "*" {
			delete(qc.cache, key)
		}
	}
}

// Clear empties the entire cache
func (qc *QueryCache) Clear() {
	qc.mu.Lock()
	defer qc.mu.Unlock()

	qc.cache = make(map[string]*CacheEntry)
	qc.hits = 0
	qc.misses = 0
}

// Stats returns cache statistics
func (qc *QueryCache) Stats() map[string]interface{} {
	qc.mu.RLock()
	defer qc.mu.RUnlock()

	total := qc.hits + qc.misses
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(qc.hits) / float64(total) * 100
	}

	return map[string]interface{}{
		"size":     len(qc.cache),
		"max_size": qc.maxSize,
		"hits":     qc.hits,
		"misses":   qc.misses,
		"total":    total,
		"hit_rate": fmt.Sprintf("%.1f%%", hitRate),
	}
}

// cleanup removes expired entries periodically
func (qc *QueryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		qc.mu.Lock()
		now := time.Now()
		for key, entry := range qc.cache {
			if now.After(entry.ExpiresAt) {
				delete(qc.cache, key)
			}
		}
		qc.mu.Unlock()
	}
}

// evictOldest removes the oldest cache entry (LRU eviction)
func (qc *QueryCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range qc.cache {
		if oldestTime.IsZero() || entry.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(qc.cache, oldestKey)
	}
}

// hashQuery generates a cache key from query
func (qc *QueryCache) hashQuery(query string) string {
	hash := md5.Sum([]byte(query))
	return fmt.Sprintf("%x", hash)
}

// CacheableQuery wraps a query with cache key information
type CacheableQuery struct {
	Query        string
	CacheTTL     time.Duration
	InvalidateOn []string // Patterns to invalidate on (e.g., "approval", "rejection")
}

// QueryCacheDecorator wraps a data fetching function with caching
type QueryCacheDecorator struct {
	cache *QueryCache
}

// NewQueryCacheDecorator creates a new cache decorator
func NewQueryCacheDecorator(cache *QueryCache) *QueryCacheDecorator {
	return &QueryCacheDecorator{
		cache: cache,
	}
}

// Execute runs a query with caching
func (qcd *QueryCacheDecorator) Execute(query string, fetchFunc func() (interface{}, error)) (interface{}, error) {
	// Check cache first
	if cached, found := qcd.cache.Get(query); found {
		return cached, nil
	}

	// Cache miss, execute query
	result, err := fetchFunc()
	if err != nil {
		return nil, err
	}

	// Store in cache
	qcd.cache.Set(query, result)

	return result, nil
}

// ExecuteJSON is like Execute but for JSON-able results
func (qcd *QueryCacheDecorator) ExecuteJSON(query string, fetchFunc func() (interface{}, error)) (json.RawMessage, error) {
	result, err := qcd.Execute(query, fetchFunc)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return json.RawMessage(data), nil
}
