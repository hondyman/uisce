package boresolver

import (
	"sync"

	lru "github.com/hashicorp/golang-lru/v2"
)

// SQLCacheKey uniquely identifies a resolved SQL expression
type SQLCacheKey struct {
	BOID        string
	CalcID      string // Can be calculation ID or expression hash if ad-hoc
	DialectName string
	VersionHash string // Ensures cache invalidation on metadata change
}

// SQLCacheValue holds the cached SQL and join requirements
type SQLCacheValue struct {
	SQL   string
	Joins []JoinStep
}

// SQLCache implements a thread-safe LRU cache for resolved SQL
type SQLCache struct {
	mu    sync.RWMutex
	store *lru.Cache[SQLCacheKey, SQLCacheValue]
}

// NewSQLCache creates a new cache with the specified size
func NewSQLCache(size int) *SQLCache {
	c, _ := lru.New[SQLCacheKey, SQLCacheValue](size)
	return &SQLCache{store: c}
}

// Get retrieval from cache
func (c *SQLCache) Get(key SQLCacheKey) (SQLCacheValue, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.store.Get(key)
}

// Set adds to cache
func (c *SQLCache) Set(key SQLCacheKey, val SQLCacheValue) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store.Add(key, val)
}
