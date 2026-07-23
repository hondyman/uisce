package bp

import "sync/atomic"

var (
	hasuraFallbackTriggerEngine   uint64
	hasuraFallbackBranchEvaluator uint64
	hasuraFallbackGeneric         uint64
)

// IncHasuraFallback increments the fallback counter for a given component.
func IncHasuraFallback(component string) {
	switch component {
	case "trigger_engine":
		atomic.AddUint64(&hasuraFallbackTriggerEngine, 1)
	case "branch_evaluator":
		atomic.AddUint64(&hasuraFallbackBranchEvaluator, 1)
	default:
		atomic.AddUint64(&hasuraFallbackGeneric, 1)
	}
}

// GetHasuraFallbackCount returns the current counter for a component.
func GetHasuraFallbackCount(component string) uint64 {
	switch component {
	case "trigger_engine":
		return atomic.LoadUint64(&hasuraFallbackTriggerEngine)
	case "branch_evaluator":
		return atomic.LoadUint64(&hasuraFallbackBranchEvaluator)
	default:
		return atomic.LoadUint64(&hasuraFallbackGeneric)
	}
}

// ResetHasuraFallbacks resets all counters to zero. Useful for tests.
func ResetHasuraFallbacks() {
	atomic.StoreUint64(&hasuraFallbackTriggerEngine, 0)
	atomic.StoreUint64(&hasuraFallbackBranchEvaluator, 0)
	atomic.StoreUint64(&hasuraFallbackGeneric, 0)
}
