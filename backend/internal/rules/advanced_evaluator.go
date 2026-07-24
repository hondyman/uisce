package rules

import (
	"fmt"
)

// AdvancedEvaluator handles recursive evaluation of complex rule structures
type AdvancedEvaluator struct {
	baseEvaluator *ConditionEvaluator
}

// NewAdvancedEvaluator creates a new evaluator instance
func NewAdvancedEvaluator() *AdvancedEvaluator {
	return &AdvancedEvaluator{
		baseEvaluator: NewConditionEvaluator(),
	}
}

// Evaluate checks if the data satisfies the rule structure
func (ae *AdvancedEvaluator) Evaluate(node RuleNode, data map[string]interface{}) (bool, error) {
	switch node.Type {
	case NodeTypeGroup:
		if node.Group == nil {
			return false, fmt.Errorf("group node is nil")
		}
		return ae.evaluateGroup(node.Group, data)
	case NodeTypeCondition:
		if node.Condition == nil {
			return false, fmt.Errorf("condition node is nil")
		}
		return ae.evaluateCondition(node.Condition, data)
	default:
		return false, fmt.Errorf("unknown node type: %s", node.Type)
	}
}

func (ae *AdvancedEvaluator) evaluateGroup(group *RuleGroup, data map[string]interface{}) (bool, error) {
	if len(group.Conditions) == 0 {
		return true, nil // Empty group is considered true (or should it be false? usually true as "no constraints")
	}

	switch group.Operator {
	case "AND":
		for _, child := range group.Conditions {
			result, err := ae.Evaluate(child, data)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil // Short-circuit
			}
		}
		return true, nil

	case "OR":
		for _, child := range group.Conditions {
			result, err := ae.Evaluate(child, data)
			if err != nil {
				return false, err
			}
			if result {
				return true, nil // Short-circuit
			}
		}
		return false, nil

	case "NOT":
		// NOT usually applies to a single child or treats children as an implicit AND
		// For simplicity, let's assume it negates the result of an implicit AND of its children
		for _, child := range group.Conditions {
			result, err := ae.Evaluate(child, data)
			if err != nil {
				return false, err
			}
			if !result {
				// If any child is false, the AND is false, so NOT(AND) is true
				return true, nil
			}
		}
		// All children are true, so AND is true, so NOT(AND) is false
		return false, nil

	default:
		return false, fmt.Errorf("unknown group operator: %s", group.Operator)
	}
}

func (ae *AdvancedEvaluator) evaluateCondition(cond *RuleCondition, data map[string]interface{}) (bool, error) {
	// Convert RuleCondition to the map format expected by ConditionEvaluator
	// This allows us to reuse the existing logic including hierarchy support

	// Determine the field to use. If FieldPath is present (cross-entity), use it.
	// The ConditionEvaluator might need updates to handle dot-notation if it doesn't already fully support it in all paths,
	// but HierarchyResolver usually handles it.
	field := cond.Field
	if cond.FieldPath != "" {
		field = cond.FieldPath
	}

	// Construct a map that mimics the old structure
	conditionMap := map[string]interface{}{
		"type":     "simple", // Default to simple, but we might need to detect hierarchy
		"field":    field,
		"operator": cond.Operator,
		"value":    cond.Value,
	}

	// If it's a hierarchy condition (e.g. has sub_entity), we might need to map it differently.
	// For now, let's assume the "simple" evaluator with HierarchyResolver's dot notation support is sufficient for most cases.
	// Looking at WorldClassConditionBuilder, it produces flat fields or dot-notation fields.

	// Pass to base evaluator
	return ae.baseEvaluator.EvaluateWithHierarchy(conditionMap, data)
}
