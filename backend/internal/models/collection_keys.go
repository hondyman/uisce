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
