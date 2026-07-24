package calcengine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

// ResultCache provides a tenant-isolated, TTL-based cache for calculation results
type ResultCache struct {
	items      map[string]*cacheEntry
	maxEntries int
	defaultTTL time.Duration
	mu         sync.RWMutex
}

type cacheEntry struct {
	response  *CalcResponse
	expiresAt time.Time
	hits      int64
}

// NewResultCache creates a new result cache
func NewResultCache(maxEntries int, defaultTTL time.Duration) *ResultCache {
	cache := &ResultCache{
		items:      make(map[string]*cacheEntry),
		maxEntries: maxEntries,
		defaultTTL: defaultTTL,
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Get retrieves a cached result
func (c *ResultCache) Get(req *CalcRequest) *CalcResponse {
	key := c.buildKey(req)

	c.mu.RLock()
	entry, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return nil
	}

	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil
	}

	// Update hit count
	c.mu.Lock()
	entry.hits++
	c.mu.Unlock()

	// Return a copy to prevent mutation
	result := *entry.response
	return &result
}

// Set stores a result in the cache
func (c *ResultCache) Set(req *CalcRequest, response *CalcResponse) {
	key := c.buildKey(req)

	c.mu.Lock()
	defer c.mu.Unlock()

	// Evict if at capacity
	if len(c.items) >= c.maxEntries {
		c.evictLRU()
	}

	c.items[key] = &cacheEntry{
		response:  response,
		expiresAt: time.Now().Add(c.defaultTTL),
		hits:      0,
	}
}

// SetWithTTL stores a result with custom TTL
func (c *ResultCache) SetWithTTL(req *CalcRequest, response *CalcResponse, ttl time.Duration) {
	key := c.buildKey(req)

	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.items) >= c.maxEntries {
		c.evictLRU()
	}

	c.items[key] = &cacheEntry{
		response:  response,
		expiresAt: time.Now().Add(ttl),
		hits:      0,
	}
}

// Invalidate removes a specific cache entry
func (c *ResultCache) Invalidate(req *CalcRequest) {
	key := c.buildKey(req)

	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// InvalidateForTenant removes all cache entries for a tenant
func (c *ResultCache) InvalidateForTenant(tenantID, datasourceID string) {
	prefix := tenantID + ":" + datasourceID + ":"

	c.mu.Lock()
	defer c.mu.Unlock()

	for key := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.items, key)
		}
	}
}

// Clear removes all cache entries
func (c *ResultCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*cacheEntry)
}

// Stats returns cache statistics
func (c *ResultCache) Stats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var totalHits int64
	var expired int
	now := time.Now()

	for _, entry := range c.items {
		totalHits += entry.hits
		if now.After(entry.expiresAt) {
			expired++
		}
	}

	return CacheStats{
		Entries:     len(c.items),
		MaxEntries:  c.maxEntries,
		TotalHits:   totalHits,
		ExpiredKeys: expired,
	}
}

// CacheStats contains cache statistics
type CacheStats struct {
	Entries     int   `json:"entries"`
	MaxEntries  int   `json:"max_entries"`
	TotalHits   int64 `json:"total_hits"`
	ExpiredKeys int   `json:"expired_keys"`
}

// buildKey creates a cache key from a request
func (c *ResultCache) buildKey(req *CalcRequest) string {
	// Key format: tenant:datasource:hash(request)
	// This ensures tenant isolation in cache

	// Create deterministic hash of request
	keyData := struct {
		CalcID     string                 `json:"calc_id"`
		MetricName string                 `json:"metric_name"`
		Params     map[string]interface{} `json:"params"`
		Mode       QueryMode              `json:"mode"`
	}{
		CalcID:     req.CalculationID,
		MetricName: req.MetricName,
		Params:     req.Params,
		Mode:       req.Mode,
	}

	jsonBytes, _ := json.Marshal(keyData)
	hash := sha256.Sum256(jsonBytes)
	hashStr := hex.EncodeToString(hash[:8]) // Use first 8 bytes for shorter key

	return req.TenantID + ":" + req.DatasourceID + ":" + hashStr
}

// evictLRU removes the least recently used entries
func (c *ResultCache) evictLRU() {
	// Simple eviction: remove expired first, then lowest hit count
	now := time.Now()
	var lowestHits int64 = -1
	var lowestKey string

	// First pass: remove expired
	for key, entry := range c.items {
		if now.After(entry.expiresAt) {
			delete(c.items, key)
		} else if lowestHits < 0 || entry.hits < lowestHits {
			lowestHits = entry.hits
			lowestKey = key
		}
	}

	// If still at capacity, remove lowest hit
	if len(c.items) >= c.maxEntries && lowestKey != "" {
		delete(c.items, lowestKey)
	}
}

// cleanup periodically removes expired entries
func (c *ResultCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.items {
			if now.After(entry.expiresAt) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}
