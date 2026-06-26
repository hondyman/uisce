package rules

import (
	"sync"

	"go.starlark.net/starlark"
)

type CoreFnKey struct {
	CoreRuleID string
	Version    int
}

type CoreFnCache struct {
	mu sync.RWMutex
	m  map[CoreFnKey]*starlark.Function
}

func NewCoreFnCache() *CoreFnCache {
	return &CoreFnCache{m: make(map[CoreFnKey]*starlark.Function)}
}

func (c *CoreFnCache) Get(k CoreFnKey) (*starlark.Function, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	fn, ok := c.m[k]
	return fn, ok
}

func (c *CoreFnCache) Set(k CoreFnKey, fn *starlark.Function) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[k] = fn
}

func (c *CoreFnCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m = make(map[CoreFnKey]*starlark.Function)
}
