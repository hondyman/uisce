package rules

import (
	"sync"

	"go.starlark.net/starlark"
)

type TenantFnKey struct {
	TenantID string
	RuleID   string
	CoreVer  int // 0 if no core (pure custom)
}

type TenantFnCache struct {
	mu sync.RWMutex
	m  map[TenantFnKey]*starlark.Function
}

func NewTenantFnCache() *TenantFnCache {
	return &TenantFnCache{m: make(map[TenantFnKey]*starlark.Function)}
}

func (c *TenantFnCache) Get(k TenantFnKey) (*starlark.Function, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	fn, ok := c.m[k]
	return fn, ok
}

func (c *TenantFnCache) Set(k TenantFnKey, fn *starlark.Function) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.m[k] = fn
}

func (c *TenantFnCache) ClearForTenant(tenantID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.m {
		if k.TenantID == tenantID {
			delete(c.m, k)
		}
	}
}
