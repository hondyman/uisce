package boresolver

import (
	"sync"
)

// Cache defines the interface for an in-memory cache with thread-safe operations.
type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Set(key K, value V)
	Clear()
}

// mapCache is a simple thread-safe in-memory cache using a sync.RWMutex.
type mapCache[K comparable, V any] struct {
	mu sync.RWMutex
	m  map[K]V
}

// NewMapCache creates a new thread-safe in-memory cache.
func NewMapCache[K comparable, V any]() *mapCache[K, V] {
	return &mapCache[K, V]{
		m: make(map[K]V),
	}
}

// Get retrieves a value from the cache.
// Returns the value and a boolean indicating whether the key was found.
func (c *mapCache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	v, ok := c.m[key]
	return v, ok
}

// Set stores a value in the cache.
func (c *mapCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = value
}

// Clear removes all entries from the cache.
func (c *mapCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[K]V)
}

// Size returns the current number of entries in the cache (for debugging).
func (c *mapCache[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.m)
}
