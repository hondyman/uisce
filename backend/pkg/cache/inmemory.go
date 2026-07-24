package cache

import (
	"sync"
	"time"
)

// InMemoryCache provides a simple thread-safe in-memory cache with TTL.
// Use this instead of Redis for simple key-value caching within a single service.
//
// When to use this:
//   - Session tokens, API response caching
//   - Access decision caching (governance)
//   - Rate limiting counters
//   - Any single-service cache that doesn't need to be shared across instances
//
// When NOT to use this:
//   - Shared state across multiple service instances
//   - Cache that must survive restarts
//   - Pub/sub patterns
//   - Complex data structures (sorted sets, etc.)
//
// For those cases, keep Redis or use a distributed cache.
type InMemoryCache struct {
	mu       sync.RWMutex
	items    map[string]*cacheItem
	maxItems int
	onEvict  func(key string, value []byte)
}

type cacheItem struct {
	value  []byte
	expiry int64 // UnixNano timestamp
}

// Option configures the cache
type Option func(*InMemoryCache)

// WithMaxItems sets the maximum number of items in the cache.
// When exceeded, oldest items are evicted.
func WithMaxItems(max int) Option {
	return func(c *InMemoryCache) {
		c.maxItems = max
	}
}

// WithEvictCallback sets a callback for when items are evicted
func WithEvictCallback(fn func(key string, value []byte)) Option {
	return func(c *InMemoryCache) {
		c.onEvict = fn
	}
}

// New creates a new in-memory cache and starts the cleanup goroutine.
func New(opts ...Option) *InMemoryCache {
	c := &InMemoryCache{
		items:    make(map[string]*cacheItem),
		maxItems: 100000, // Default: 100k items
	}

	for _, opt := range opts {
		opt(c)
	}

	go c.cleanup()
	return c
}

// Set stores a value with the given TTL (in seconds).
// If ttl is 0, the item never expires.
func (c *InMemoryCache) Set(key string, value []byte, ttlSeconds int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if we need to evict
	if len(c.items) >= c.maxItems {
		c.evictOldest()
	}

	var expiry int64
	if ttlSeconds > 0 {
		expiry = time.Now().Add(time.Duration(ttlSeconds) * time.Second).UnixNano()
	}

	c.items[key] = &cacheItem{
		value:  value,
		expiry: expiry,
	}
}

// Get retrieves a value. Returns nil and false if not found or expired.
func (c *InMemoryCache) Get(key string) ([]byte, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, ok := c.items[key]
	if !ok {
		return nil, false
	}

	// Check expiry (0 = never expires)
	if item.expiry > 0 && time.Now().UnixNano() > item.expiry {
		return nil, false
	}

	return item.value, true
}

// Delete removes an item from the cache.
func (c *InMemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.items[key]; ok && c.onEvict != nil {
		c.onEvict(key, item.value)
	}
	delete(c.items, key)
}

// Has checks if a key exists and is not expired.
func (c *InMemoryCache) Has(key string) bool {
	_, ok := c.Get(key)
	return ok
}

// Clear removes all items from the cache.
func (c *InMemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.onEvict != nil {
		for k, v := range c.items {
			c.onEvict(k, v.value)
		}
	}
	c.items = make(map[string]*cacheItem)
}

// Len returns the number of items in the cache (including expired ones).
func (c *InMemoryCache) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Keys returns all non-expired keys.
func (c *InMemoryCache) Keys() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now().UnixNano()
	keys := make([]string, 0, len(c.items))
	for k, v := range c.items {
		if v.expiry == 0 || now <= v.expiry {
			keys = append(keys, k)
		}
	}
	return keys
}

// cleanup periodically removes expired items.
func (c *InMemoryCache) cleanup() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		c.removeExpired()
	}
}

func (c *InMemoryCache) removeExpired() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().UnixNano()
	for k, v := range c.items {
		if v.expiry > 0 && now > v.expiry {
			if c.onEvict != nil {
				c.onEvict(k, v.value)
			}
			delete(c.items, k)
		}
	}
}

// evictOldest removes the item with the earliest expiry.
// Must be called with lock held.
func (c *InMemoryCache) evictOldest() {
	var oldestKey string
	var oldestExpiry int64 = 1<<63 - 1

	for k, v := range c.items {
		exp := v.expiry
		if exp == 0 {
			exp = 1<<63 - 1 // Never-expiring items evicted last
		}
		if exp < oldestExpiry {
			oldestExpiry = exp
			oldestKey = k
		}
	}

	if oldestKey != "" {
		if item, ok := c.items[oldestKey]; ok && c.onEvict != nil {
			c.onEvict(oldestKey, item.value)
		}
		delete(c.items, oldestKey)
	}
}

// Stats returns cache statistics.
type Stats struct {
	Items       int   `json:"items"`
	MaxItems    int   `json:"max_items"`
	ExpiredKeys int   `json:"expired_keys"`
	MemoryBytes int64 `json:"memory_bytes_approx"`
}

func (c *InMemoryCache) Stats() Stats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := time.Now().Unix()
	expired := 0
	var memBytes int64

	for _, v := range c.items {
		if v.expiry > 0 && now > v.expiry {
			expired++
		}
		memBytes += int64(len(v.value)) + 24 // 24 bytes overhead per item (approx)
	}

	return Stats{
		Items:       len(c.items),
		MaxItems:    c.maxItems,
		ExpiredKeys: expired,
		MemoryBytes: memBytes,
	}
}
