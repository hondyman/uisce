package validation

import (
	"fmt"
	"regexp"
	"strconv"
)

// ValidationEngine provides functions to execute validation rules
type ValidationEngine struct{}

// NewValidationEngine creates a new validation engine
func NewValidationEngine() *ValidationEngine {
	return &ValidationEngine{}
}

// ExecutionContext holds the data and context for rule execution
type ExecutionContext struct {
	RuleID       string
	RuleType     string
	TargetEntity string
	Condition    map[string]interface{}
	Data         map[string]interface{}
}

// ExecutionResult represents the result of rule execution
type ExecutionResult struct {
	RuleID  string
	Passed  bool
	Message string
	Details map[string]interface{}
}

// Execute runs a validation rule against data
func (ve *ValidationEngine) Execute(ctx ExecutionContext) ExecutionResult {
	switch ctx.RuleType {
	case "field_format":
		return ve.executeFieldFormat(ctx)
	case "cardinality":
		return ve.executeCardinality(ctx)
	case "uniqueness":
		return ve.executeUniqueness(ctx)
	case "referential_integrity":
		return ve.executeReferentialIntegrity(ctx)
	case "business_logic":
		return ve.executeBusinessLogic(ctx)
	default:
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: fmt.Sprintf("Unknown rule type: %s", ctx.RuleType),
		}
	}
}

// executeFieldFormat validates that a field matches a regex pattern
func (ve *ValidationEngine) executeFieldFormat(ctx ExecutionContext) ExecutionResult {
	field, ok := ctx.Condition["field"].(string)
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: "Field format rule missing 'field' parameter",
		}
	}

	pattern, ok := ctx.Condition["pattern"].(string)
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: "Field format rule missing 'pattern' parameter",
		}
	}

	value, ok := ctx.Data[field]
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: fmt.Sprintf("Field '%s' not found in data", field),
		}
	}

	valueStr, ok := value.(string)
	if !ok {
		valueStr = fmt.Sprintf("%v", value)
	}

	matched, err := regexp.MatchString(pattern, valueStr)
	if err != nil {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: fmt.Sprintf("Invalid regex pattern: %v", err),
		}
	}

	if !matched {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: fmt.Sprintf("Field '%s' with value '%s' does not match pattern '%s'", field, valueStr, pattern),
		}
	}

	return ExecutionResult{
		RuleID:  ctx.RuleID,
		Passed:  true,
		Message: fmt.Sprintf("Field '%s' matches pattern '%s'", field, pattern),
	}
}

// executeCardinality validates count/threshold conditions
func (ve *ValidationEngine) executeCardinality(ctx ExecutionContext) ExecutionResult {
	field, ok := ctx.Condition["field"].(string)
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: "Cardinality rule missing 'field' parameter",
		}
	}

	operator, ok := ctx.Condition["operator"].(string)
	if !ok {
		operator = ">"
	}

	thresholdVal, ok := ctx.Condition["value"]
	if !ok {
		thresholdVal = ctx.Condition["threshold"]
	}

	if thresholdVal == nil {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: "Cardinality rule missing 'value' or 'threshold' parameter",
		}
	}

	// Convert threshold to float64
	var threshold float64
	switch v := thresholdVal.(type) {
	case float64:
		threshold = v
	case int:
		threshold = float64(v)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return ExecutionResult{
				RuleID:  ctx.RuleID,
				Passed:  false,
				Message: fmt.Sprintf("Cannot parse threshold value: %v", err),
			}
		}
		threshold = f
	default:
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: fmt.Sprintf("Invalid threshold type: %T", thresholdVal),
		}
	}

	value, ok := ctx.Data[field]
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: fmt.Sprintf("Field '%s' not found in data", field),
		}
	}

	// Convert value to float64
	var actualValue float64
	switch v := value.(type) {
	case float64:
		actualValue = v
	case int:
		actualValue = float64(v)
	case string:
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return ExecutionResult{
				RuleID:  ctx.RuleID,
				Passed:  false,
				Message: fmt.Sprintf("Cannot parse field value: %v", err),
			}
		}
		actualValue = f
	default:
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: fmt.Sprintf("Invalid value type for cardinality: %T", value),
		}
	}

	// Compare using operator
	var passed bool
	switch operator {
	case ">":
		passed = actualValue > threshold
	case "<":
		passed = actualValue < threshold
	case ">=":
		passed = actualValue >= threshold
	case "<=":
		passed = actualValue <= threshold
	case "==":
		passed = actualValue == threshold
	case "!=":
		passed = actualValue != threshold
	default:
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: fmt.Sprintf("Unknown operator: %s", operator),
		}
	}

	if !passed {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: fmt.Sprintf("Field '%s' value %v does not satisfy %s %v", field, actualValue, operator, threshold),
		}
	}

	return ExecutionResult{
		RuleID:  ctx.RuleID,
		Passed:  true,
		Message: fmt.Sprintf("Field '%s' value %v satisfies %s %v", field, actualValue, operator, threshold),
	}
}

// executeUniqueness validates that a field contains a unique value
func (ve *ValidationEngine) executeUniqueness(ctx ExecutionContext) ExecutionResult {
	field, ok := ctx.Condition["field"].(string)
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: "Uniqueness rule missing 'field' parameter",
		}
	}

	value, ok := ctx.Data[field]
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: fmt.Sprintf("Field '%s' not found in data", field),
		}
	}

	// Note: Actual uniqueness check would require database query against all existing values
	// This is a placeholder that always passes in the execution engine
	// The database query would be done at the API layer when executing the rule

	return ExecutionResult{
		RuleID:  ctx.RuleID,
		Passed:  true,
		Message: fmt.Sprintf("Uniqueness check for field '%s' with value '%v' requires database validation", field, value),
	}
}

// executeReferentialIntegrity validates foreign key relationships
func (ve *ValidationEngine) executeReferentialIntegrity(ctx ExecutionContext) ExecutionResult {
	sourceEntity, ok := ctx.Condition["source_entity"].(string)
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: "Referential integrity rule missing 'source_entity' parameter",
		}
	}

	sourceField, ok := ctx.Condition["source_field"].(string)
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: "Referential integrity rule missing 'source_field' parameter",
		}
	}

	targetEntity, ok := ctx.Condition["target_entity"].(string)
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: "Referential integrity rule missing 'target_entity' parameter",
		}
	}

	targetField, ok := ctx.Condition["target_field"].(string)
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: "Referential integrity rule missing 'target_field' parameter",
		}
	}

	sourceValue, ok := ctx.Data[sourceField]
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: fmt.Sprintf("Source field '%s' not found in data", sourceField),
		}
	}

	// Note: Actual referential integrity check would require database query
	// This is a placeholder

	return ExecutionResult{
		RuleID:  ctx.RuleID,
		Passed:  true,
		Message: fmt.Sprintf("Referential integrity check: %s.%s (%v) -> %s.%s (requires database validation)", sourceEntity, sourceField, sourceValue, targetEntity, targetField),
	}
}

// executeBusinessLogic validates custom business logic conditions
func (ve *ValidationEngine) executeBusinessLogic(ctx ExecutionContext) ExecutionResult {
	// For business logic rules, we evaluate a simple comparison expression
	field, ok := ctx.Condition["field"].(string)
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: "Business logic rule missing 'field' parameter",
		}
	}

	operator, ok := ctx.Condition["operator"].(string)
	if !ok {
		operator = ">"
	}

	expectedVal, ok := ctx.Condition["value"]
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: "Business logic rule missing 'value' parameter",
		}
	}

	actualValue, ok := ctx.Data[field]
	if !ok {
		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  false,
			Message: fmt.Sprintf("Field '%s' not found in data", field),
		}
	}

	// Attempt numeric comparison first
	var actualNum, expectedNum float64
	var numComparison = false

	if av, ok := actualValue.(float64); ok {
		actualNum = av
		numComparison = true
	} else if av, ok := actualValue.(int); ok {
		actualNum = float64(av)
		numComparison = true
	}

	if ev, ok := expectedVal.(float64); ok && numComparison {
		expectedNum = ev
	} else if ev, ok := expectedVal.(int); ok && numComparison {
		expectedNum = float64(ev)
	} else {
		numComparison = false
	}

	if numComparison {
		var passed bool
		switch operator {
		case ">":
			passed = actualNum > expectedNum
		case "<":
			passed = actualNum < expectedNum
		case ">=":
			passed = actualNum >= expectedNum
		case "<=":
			passed = actualNum <= expectedNum
		case "==":
			passed = actualNum == expectedNum
		case "!=":
			passed = actualNum != expectedNum
		default:
			return ExecutionResult{
				RuleID:  ctx.RuleID,
				Passed:  false,
				Message: fmt.Sprintf("Unknown operator: %s", operator),
			}
		}

		if !passed {
			return ExecutionResult{
				RuleID:  ctx.RuleID,
				Passed:  false,
				Message: fmt.Sprintf("Business logic failed: %v %s %v", actualNum, operator, expectedNum),
			}
		}

		return ExecutionResult{
			RuleID:  ctx.RuleID,
			Passed:  true,
			Message: fmt.Sprintf("Business logic passed: %v %s %v", actualNum, operator, expectedNum),
		}
	}

	// String comparison
	actualStr := fmt.Sprintf("%v", actualValue)
	expectedStr := fmt.Sprintf("%v", expectedVal)

	switch operator {
	case "==":
		if actualStr == expectedStr {
			return ExecutionResult{
				RuleID:  ctx.RuleID,
				Passed:  true,
				Message: fmt.Sprintf("Business logic passed: '%s' == '%s'", actualStr, expectedStr),
			}
		}
	case "!=":
		if actualStr != expectedStr {
			return ExecutionResult{
				RuleID:  ctx.RuleID,
				Passed:  true,
				Message: fmt.Sprintf("Business logic passed: '%s' != '%s'", actualStr, expectedStr),
			}
		}
	}

	return ExecutionResult{
		RuleID:  ctx.RuleID,
		Passed:  false,
		Message: fmt.Sprintf("Business logic failed: '%s' %s '%s'", actualStr, operator, expectedStr),
	}
}
