package rules

import (
	"sync"
	"sync/atomic"

	vm "github.com/hondyman/uisce/backend/internal/rules/vm"
)

// EngineState is an immutable snapshot of the rule engine's schema and
// compile cache. Created by Rewarm() (or SetInitialState at startup) and
// never mutated thereafter. The atomic.Pointer swap is the only way to
// publish a new state — readers see a consistent view at all times.
type EngineState struct {
	Syms            *vm.SymbolDict
	Enums           *vm.EnumDict
	Cache           *sync.Map // map[cacheKey]*vm.CompileResult
	Version         int       // externally-supplied schema version
	Revision        uint64    // monotonic counter incremented on each Rewarm
	LastUsedUnixNano int64   // Unix nano timestamp of last getState() access; used for TTL eviction
}

// CacheSize returns the number of cached compile results.
func (s *EngineState) CacheSize() int {
	if s == nil || s.Cache == nil {
		return 0
	}
	n := 0
	s.Cache.Range(func(_, _ any) bool {
		n++
		return true
	})
	return n
}

// EngineMetrics holds cumulative counters that survive state swaps.
// These are business-level metrics (cache hit rate, fallback rate, etc.)
// and must not be lost when an EngineState is replaced. Fields are
// unexported; use the typed accessor methods.
type EngineMetrics struct {
	cacheHits     atomic.Uint64
	cacheMisses   atomic.Uint64
	fallbacks     atomic.Uint64
	vmPathCount   atomic.Uint64
	compileErrors atomic.Uint64
}

// CacheHits returns the cumulative cache-hit count.
func (m *EngineMetrics) CacheHits() uint64 { return m.cacheHits.Load() }

// CacheMisses returns the cumulative cache-miss count.
func (m *EngineMetrics) CacheMisses() uint64 { return m.cacheMisses.Load() }

// Fallbacks returns the cumulative fallback-to-recursive count.
func (m *EngineMetrics) Fallbacks() uint64 { return m.fallbacks.Load() }

// VMPathCount returns the cumulative VM-path count.
func (m *EngineMetrics) VMPathCount() uint64 { return m.vmPathCount.Load() }

// CompileErrors returns the cumulative compile-error count.
func (m *EngineMetrics) CompileErrors() uint64 { return m.compileErrors.Load() }

// EngineMetricsSnapshot is a point-in-time copy of the metrics.
type EngineMetricsSnapshot struct {
	CacheHits, CacheMisses, Fallbacks, VMPathCount, CompileErrors uint64
}

// Snapshot returns a value-typed snapshot of the metrics.
func (m *EngineMetrics) Snapshot() EngineMetricsSnapshot {
	return EngineMetricsSnapshot{
		CacheHits:     m.cacheHits.Load(),
		CacheMisses:   m.cacheMisses.Load(),
		Fallbacks:     m.fallbacks.Load(),
		VMPathCount:   m.vmPathCount.Load(),
		CompileErrors: m.compileErrors.Load(),
	}
}

// EvalTrace provides observability into a single evaluation. Returned
// alongside the boolean result so SREs can log/alert on fallback rates.
type EvalTrace struct {
	RuleID    string // the versioned cache key actually used (ruleID@version)
	UsedVM    bool   // true if the VM fast path executed
	Fallback  string // reason for fallback; empty if UsedVM is true
	Revision  uint64 // which EngineState.Revision served this call
	IsTenant  bool   // true if evaluated against a tenant-specific state
	TenantID  string // tenant that was charged with this evaluation (empty for core)
}