// backend/internal/rules/condition_evaluator.go (UPDATED)

package rules

import (
	"fmt"
	"reflect"
)

type ConditionEvaluator struct {
	hierarchyResolver *HierarchyResolver
}

func NewConditionEvaluator() *ConditionEvaluator {
	return &ConditionEvaluator{
		hierarchyResolver: NewHierarchyResolver(),
	}
}

// EvaluateWithHierarchy evaluates condition with hierarchy support
func (ce *ConditionEvaluator) EvaluateWithHierarchy(
	condition map[string]interface{},
	data map[string]interface{},
) (bool, error) {

	// Check if this is a hierarchy condition
	if condType, ok := condition["type"].(string); ok && condType == "hierarchy" {
		return ce.evaluateHierarchyCondition(condition, data)
	}

	// Check if this is an aggregation condition
	if condType, ok := condition["type"].(string); ok && condType == "hierarchy_aggregate" {
		return ce.evaluateAggregateCondition(condition, data)
	}

	// Check if this is a simple condition (e.g., for Parent Only rules)
	if condType, ok := condition["type"].(string); ok && condType == "simple" {
		return ce.evaluateSimpleCondition(condition, data)
	}

	// Fall back to regular evaluation
	return ce.Evaluate(condition, data)
}

func (ce *ConditionEvaluator) evaluateHierarchyCondition(
	condition map[string]interface{},
	data map[string]interface{},
) (bool, error) {

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

	var comparisonValue interface{}
	if val, ok := condition["value"]; ok {
		comparisonValue = val
	} else if parentFieldForValue, ok := condition["value_from_parent_field"].(string); ok {
		resolvedParentVal, found := ce.hierarchyResolver.ResolveFieldPath(data, parentFieldForValue)
		if !found {
			return false, fmt.Errorf("failed to resolve parent field for comparison value: %s", parentFieldForValue)
		}
		comparisonValue = resolvedParentVal
	}

	// Resolve sub-entity field for ALL array elements
	subValues, ok := ce.hierarchyResolver.ResolveFieldPathArray(data, subEntity+"."+field)
	if !ok {
		// If path doesn't resolve, it might not be an error, but the condition can't be met.
		// Depending on strictness, you might return true here. For now, let's be strict.
		return false, nil
	}

	// Evaluate condition for each sub-entity
	for _, subValue := range subValues {
		result, err := ce.compareValues(subValue, operator, comparisonValue)
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

func (ce *ConditionEvaluator) evaluateAggregateCondition(
	condition map[string]interface{},
	data map[string]interface{},
) (bool, error) {

	subEntity, ok := condition["sub_entity"].(string)
	if !ok {
		return false, fmt.Errorf("missing sub_entity")
	}

	aggregation, ok := condition["aggregation"].(string)
	if !ok {
		return false, fmt.Errorf("missing aggregation")
	}

	aggregationField, ok := condition["aggregation_field"].(string)
	if !ok {
		return false, fmt.Errorf("missing aggregation_field")
	}

	// The path for aggregation is the sub-entity itself.
	// The field to aggregate on is specified separately.
	aggregatedValue, ok := ce.hierarchyResolver.ResolveWithAggregation(
		data,
		subEntity,
		AggregationType(aggregation),
		aggregationField,
	)
	if !ok {
		// Could be that no items were found to aggregate. Treat as non-passing.
		return false, nil
	}

	parentField := condition["parent_field"].(string)
	parentVal, ok := ce.hierarchyResolver.ResolveFieldPath(data, parentField)
	if !ok {
		return false, fmt.Errorf("failed to resolve parent field: %s", parentField)
	}

	operator := condition["operator"].(string)
	return ce.compareValues(parentVal, operator, aggregatedValue)
}

// evaluateSimpleCondition evaluates a non-hierarchical condition.
func (ce *ConditionEvaluator) evaluateSimpleCondition(
	condition map[string]interface{},
	data map[string]interface{},
) (bool, error) {
	fieldPath, ok := condition["field"].(string)
	if !ok {
		return false, fmt.Errorf("missing field in simple condition")
	}
	operator, ok := condition["operator"].(string)
	if !ok {
		return false, fmt.Errorf("missing operator in simple condition")
	}
	value := condition["value"]

	actualVal, found := ce.hierarchyResolver.ResolveFieldPath(data, fieldPath)
	if !found {
		// If the field doesn't exist, the condition cannot be met.
		return false, nil
	}

	return ce.compareValues(actualVal, operator, value)
}

// compareValues compares two values with an operator.
// It handles numeric comparisons and delegates to reflect.DeepEqual for others.
func (ce *ConditionEvaluator) compareValues(
	actual interface{},
	operator string,
	expected interface{},
) (bool, error) {
	// Special handling for aggregate comparison where types might differ
	if operator == "equals_aggregate" {
		numA, okA := ce.hierarchyResolver.toNumber(actual)
		numB, okB := ce.hierarchyResolver.toNumber(expected)
		if okA && okB {
			return numA == numB, nil
		}
		// If not both numbers, fall back to deep equal
		return reflect.DeepEqual(actual, expected), nil
	}

	// This is a simplified comparison. A full implementation would handle types robustly.
	// For now, we'll delegate numeric comparisons.
	switch operator {
	case "equals", "equal", "==":
		return reflect.DeepEqual(actual, expected), nil

	case "not_equals", "!=":
		return !reflect.DeepEqual(actual, expected), nil

	case "greater_than", ">":
		return ce.isGreaterThan(actual, expected) // Delegates to numeric comparison

	case "less_than", "<":
		return ce.isLessThan(actual, expected) // Delegates to numeric comparison

	case "greater_equal", ">=":
		eq := reflect.DeepEqual(actual, expected)
		gt, _ := ce.isGreaterThan(actual, expected)
		return gt || eq, nil

	case "less_equal", "<=":
		eq := reflect.DeepEqual(actual, expected)
		lt, _ := ce.isLessThan(actual, expected)
		return lt || eq, nil

	default:
		return false, fmt.Errorf("unknown operator: %s", operator)
	}
}

// Dummy Evaluate for fallback
func (ce *ConditionEvaluator) Evaluate(condition map[string]interface{}, data map[string]interface{}) (bool, error) { // This is the original fallback
	// This method is now effectively replaced by evaluateSimpleCondition for "simple" type.
	// If other non-hierarchical types are introduced, they would be handled here.
	return true, nil // Default to true if no specific evaluation logic is found (e.g., for unknown types)
}

func (ce *ConditionEvaluator) isGreaterThan(a, b interface{}) (bool, error) {
	numA, okA := ce.hierarchyResolver.toNumber(a)
	numB, okB := ce.hierarchyResolver.toNumber(b)
	if !okA || !okB {
		return false, fmt.Errorf("cannot compare non-numeric values for greater_than: %v, %v", a, b)
	}
	return numA > numB, nil
}

func (ce *ConditionEvaluator) isLessThan(a, b interface{}) (bool, error) {
	numA, okA := ce.hierarchyResolver.toNumber(a)
	numB, okB := ce.hierarchyResolver.toNumber(b)
	if !okA || !okB {
		return false, fmt.Errorf("cannot compare non-numeric values for less_than: %v, %v", a, b)
	}
	return numA < numB, nil
}
