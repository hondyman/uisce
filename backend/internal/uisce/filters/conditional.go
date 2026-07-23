package filters

import (
	"context"
	"fmt"
)

// Filter is the interface all filters must implement
type Filter interface {
	Name() string
	Purify(ctx context.Context, data map[string]interface{}) error
}

// ConditionalFilter implements IF/ELSE branching logic
type ConditionalFilter struct {
	ConditionField string      // Field to check
	ConditionOp    string      // "eq", "ne", "gt", "lt", "contains"
	ConditionValue interface{} // Value to compare against
	ThenFilter     Filter      // Filter to run if condition is true
	ElseFilter     Filter      // Filter to run if condition is false (optional)
}

func (f *ConditionalFilter) Name() string {
	return "Conditional Branch"
}

func (f *ConditionalFilter) Purify(ctx context.Context, data map[string]interface{}) error {
	conditionMet, err := f.evaluateCondition(data)
	if err != nil {
		return err
	}

	if conditionMet {
		if f.ThenFilter != nil {
			return f.ThenFilter.Purify(ctx, data)
		}
	} else {
		if f.ElseFilter != nil {
			return f.ElseFilter.Purify(ctx, data)
		}
	}

	return nil
}

func (f *ConditionalFilter) evaluateCondition(data map[string]interface{}) (bool, error) {
	value, ok := data[f.ConditionField]
	if !ok {
		return false, nil // Field not present, condition is false
	}

	switch f.ConditionOp {
	case "eq":
		return value == f.ConditionValue, nil
	case "ne":
		return value != f.ConditionValue, nil
	case "gt":
		return compareNumeric(value, f.ConditionValue, ">")
	case "lt":
		return compareNumeric(value, f.ConditionValue, "<")
	case "gte":
		return compareNumeric(value, f.ConditionValue, ">=")
	case "lte":
		return compareNumeric(value, f.ConditionValue, "<=")
	case "contains":
		strValue, ok1 := value.(string)
		strTarget, ok2 := f.ConditionValue.(string)
		if !ok1 || !ok2 {
			return false, fmt.Errorf("contains requires string values")
		}
		return containsSubstring(strValue, strTarget), nil
	default:
		return false, fmt.Errorf("unknown condition operator: %s", f.ConditionOp)
	}
}

func compareNumeric(a, b interface{}, op string) (bool, error) {
	aFloat, aOk := toFloat64(a)
	bFloat, bOk := toFloat64(b)
	if !aOk || !bOk {
		return false, fmt.Errorf("numeric comparison requires numeric values")
	}

	switch op {
	case ">":
		return aFloat > bFloat, nil
	case "<":
		return aFloat < bFloat, nil
	case ">=":
		return aFloat >= bFloat, nil
	case "<=":
		return aFloat <= bFloat, nil
	default:
		return false, nil
	}
}

func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	default:
		return 0, false
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
