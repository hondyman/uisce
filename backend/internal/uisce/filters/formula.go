package filters

import (
	"context"
	"fmt"
	"go/parser"
	"strconv"
)

// FormulaFilter evaluates a simple expression against data
type FormulaFilter struct {
	Expression   string // e.g., "amount > 1000 && quantity < 100"
	ErrorMessage string // Custom error message on failure
}

func (f *FormulaFilter) Name() string {
	return "Formula"
}

func (f *FormulaFilter) Purify(ctx context.Context, data map[string]interface{}) error {
	// Simple expression evaluation
	// In production, you'd use a proper expression engine like Starlark or CEL

	// For now, just check a simple comparison like "amount > 1000"
	// This is a placeholder - real implementation would use an expression evaluator

	if f.Expression == "" {
		return nil
	}

	// Parse simple comparison expressions
	// Format: "field_name operator value"
	// Example: "amount > 1000"

	// For the MVP, we'll just validate that the expression looks parseable
	_, err := parser.ParseExpr(f.Expression)
	if err != nil {
		// Try a simpler approach - just check if it's not empty
		return nil
	}

	// Placeholder: In real implementation, evaluate the expression
	// For now, always pass
	return nil
}

// SimpleCompare performs a simple field comparison
func SimpleCompare(data map[string]interface{}, field string, op string, targetValue float64) error {
	value, ok := data[field]
	if !ok {
		return fmt.Errorf("field '%s' not found", field)
	}

	var numValue float64
	switch v := value.(type) {
	case float64:
		numValue = v
	case int:
		numValue = float64(v)
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("field '%s' is not numeric", field)
		}
		numValue = parsed
	default:
		return fmt.Errorf("field '%s' is not numeric", field)
	}

	var result bool
	switch op {
	case ">":
		result = numValue > targetValue
	case ">=":
		result = numValue >= targetValue
	case "<":
		result = numValue < targetValue
	case "<=":
		result = numValue <= targetValue
	case "==":
		result = numValue == targetValue
	case "!=":
		result = numValue != targetValue
	default:
		return fmt.Errorf("unknown operator: %s", op)
	}

	if !result {
		return fmt.Errorf("formula check failed: %s %s %v", field, op, targetValue)
	}

	return nil
}
