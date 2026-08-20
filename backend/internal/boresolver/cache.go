package boresolver

import (
	"fmt"
	"sync"

	"golang.org/x/sync/singleflight"
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

// SingleFlightCoherentCache provides thread-safe in-memory caching with singleflight deduplication
// to completely prevent thundering herds / cache stampedes on database misses.
type SingleFlightCoherentCache[K comparable, V any] struct {
	mu           sync.RWMutex
	m            map[K]V
	requestGroup singleflight.Group
}

func NewSingleFlightCoherentCache[K comparable, V any]() *SingleFlightCoherentCache[K, V] {
	return &SingleFlightCoherentCache[K, V]{
		m: make(map[K]V),
	}
}

func (c *SingleFlightCoherentCache[K, V]) GetOrFetch(key K, fetchFn func() (V, error)) (V, error) {
	c.mu.RLock()
	if val, ok := c.m[key]; ok {
		c.mu.RUnlock()
		return val, nil
	}
	c.mu.RUnlock()

	// Singleflight request coalescing
	keyStr := fmt.Sprintf("%v", key)
	res, err, _ := c.requestGroup.Do(keyStr, func() (interface{}, error) {
		val, err := fetchFn()
		if err != nil {
			return nil, err
		}
		c.mu.Lock()
		c.m[key] = val
		c.mu.Unlock()
		return val, nil
	})

	if err != nil {
		var zero V
		return zero, err
	}

	return res.(V), nil
}

func (c *SingleFlightCoherentCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[key] = value
}

func (c *SingleFlightCoherentCache[K, V]) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[K]V)
}

