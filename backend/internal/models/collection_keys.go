// Package models hosts shared domain constants that need to be referenced by
// multiple internal packages without creating import cycles.
//
// CollectionKeysForBO declares the relation-scoped collection keys each BO's
// context loader is expected to deliver. This is the single source of truth:
// - analytics/validation_rule_service.go reads it to populate bo-fields (editor
//   only offers what the evaluator can resolve)
// - metadata/shadow_evaluation.go reads it to implement context loaders
//
// Adding a second relation to a BO? Add it here in both places, and the
// drift-guard test will catch any half-implemented delivery.
package models

// CollectionKeysForBO returns the collection key names that a BO's rule-context
// loader is expected to populate for evaluation. Each entry is a top-level
// identifier that appears as a dotted prefix in field references
// (e.g. "OrderAllocations" → "OrderAllocations.target_qty").
func CollectionKeysForBO(boKey string) []string {
	switch boKey {
	case "order":
		return []string{"OrderAllocations"}
	default:
		return nil
	}
}

// IsKnownContextField reports whether f is a known related-row context key.
func IsKnownContextField(f string) bool {
	return knownTransientContextFields[f]
}

var knownTransientContextFields = map[string]bool{
	"sibling_qty_sum":             true,
	"routed_qty":                  true,
	"executed_qty":                true,
	"allocation_target_qty_sum":   true,
	"account_status":              true,
	"account_is_discretionary":    true,
	"placement_routed_sum":        true,
	"duplicate_order_count":       true,
	"order_target_qty":            true,
	"sibling_routed_sum":          true,
	"broker_status":               true,
	"placement_created_at":        true,
	"order_side":                  true,
	"order_limit_price":           true,
	"duplicate_broker_exec_count": true,
	"alloc_fill_sum":              true,
	"parent_exec_qty":             true,
	"parent_exec_price":           true,
	"parent_order_id":             true,
	"alloc_order_id":              true,
	"sibling_alloc_sum":           true,
	"causality_ok":                true,
	"same_order_ok":               true,
}

