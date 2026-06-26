package rules

import (
	"fmt"
)

// ============================================================================
// HIERARCHY EVALUATION FUNCTIONS
// ============================================================================

// EvaluateHierarchyCondition evaluates a hierarchy-based validation rule
func EvaluateHierarchyCondition(
	condition map[string]interface{},
	data map[string]interface{},
) (bool, error) {

	// Check if this is a hierarchy condition
	if condType, ok := condition["type"].(string); ok {
		switch condType {
		case "hierarchy":
			return evaluateHierarchyCondition(condition, data)
		case "hierarchy_aggregate":
			return evaluateAggregateCondition(condition, data)
		default:
			return false, fmt.Errorf("unknown condition type: %s", condType)
		}
	}

	return false, fmt.Errorf("missing condition type")
}

func evaluateHierarchyCondition(
	condition map[string]interface{},
	data map[string]interface{},
) (bool, error) {

	resolver := NewHierarchyResolver()

	subEntity, ok := condition["sub_entity"].(string)
	if !ok {
		return false, fmt.Errorf("missing sub_entity in hierarchy condition")
	}

	field, ok := condition["field"].(string)
	if !ok {
		return false, fmt.Errorf("missing field in hierarchy condition")
	}

	operator, ok := condition["operator"].(string)
	if !ok {
		return false, fmt.Errorf("missing operator in hierarchy condition")
	}

	value := condition["value"]

	// Resolve sub-entity field for ALL array elements
	subValues, ok := resolver.ResolveFieldPathArray(data, subEntity+"."+field)
	if !ok {
		return false, fmt.Errorf("failed to resolve path: %s.%s", subEntity, field)
	}

	// Evaluate condition for each sub-entity
	for _, subValue := range subValues {
		result, err := compareValues(subValue, operator, value)
		if err != nil {
			return false, err
		}

		// If ANY sub-entity fails, rule fails
		if !result {
			return false, nil
		}
	}

	return true, nil
}

func evaluateAggregateCondition(
	condition map[string]interface{},
	data map[string]interface{},
) (bool, error) {

	resolver := NewHierarchyResolver()

	subEntity, ok := condition["sub_entity"].(string)
	if !ok {
		return false, fmt.Errorf("missing sub_entity")
	}

	aggregation, ok := condition["aggregation"].(string)
	if !ok {
		return false, fmt.Errorf("missing aggregation")
	}

	field, ok := condition["aggregation_field"].(string)
	if !ok {
		return false, fmt.Errorf("missing aggregation_field")
	}

	aggregated, ok := resolver.ResolveWithAggregation(
		data,
		subEntity+"."+field,
		AggregationType(aggregation),
		field,
	)
	if !ok {
		return false, fmt.Errorf("failed to aggregate")
	}

	parentField, ok := condition["parent_field"].(string)
	if !ok {
		return false, fmt.Errorf("missing parent_field")
	}

	parentVal, ok := resolver.ResolveFieldPath(data, parentField)
	if !ok {
		return false, fmt.Errorf("failed to resolve parent field: %s", parentField)
	}

	operator, ok := condition["operator"].(string)
	if !ok {
		return false, fmt.Errorf("missing operator")
	}

	return compareValues(aggregated, operator, parentVal)
}

// compareValues compares two values with operator
func compareValues(
	actual interface{},
	operator string,
	expected interface{},
) (bool, error) {

	switch operator {
	case "equals", "equal", "==":
		return actual == expected, nil

	case "not_equals", "!=":
		return actual != expected, nil

	case "greater_than", ">":
		return isGreaterThan(actual, expected)

	case "less_than", "<":
		return isLessThan(actual, expected)

	case "greater_equal", ">=":
		gt, _ := isGreaterThan(actual, expected)
		eq := actual == expected
		return gt || eq, nil

	case "less_equal", "<=":
		lt, _ := isLessThan(actual, expected)
		eq := actual == expected
		return lt || eq, nil

	default:
		return false, fmt.Errorf("unknown operator: %s", operator)
	}
}

func isGreaterThan(a, b interface{}) (bool, error) {
	resolver := NewHierarchyResolver()
	numA, ok := resolver.toNumber(a)
	if !ok {
		return false, fmt.Errorf("cannot convert to number: %v", a)
	}

	numB, ok := resolver.toNumber(b)
	if !ok {
		return false, fmt.Errorf("cannot convert to number: %v", b)
	}

	return numA > numB, nil
}

func isLessThan(a, b interface{}) (bool, error) {
	resolver := NewHierarchyResolver()
	numA, ok := resolver.toNumber(a)
	if !ok {
		return false, fmt.Errorf("cannot convert to number: %v", a)
	}

	numB, ok := resolver.toNumber(b)
	if !ok {
		return false, fmt.Errorf("cannot convert to number: %v", b)
	}

	return numA < numB, nil
}
