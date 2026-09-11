package vm

import (
	"fmt"
	"strings"
)

type AdvancedEvaluator struct {
	baseEvaluator *ConditionEvaluator
}

func NewAdvancedEvaluator() *AdvancedEvaluator {
	return &AdvancedEvaluator{
		baseEvaluator: NewConditionEvaluator(),
	}
}

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
	case NodeTypeExpression:
		if node.Expression == nil {
			return false, fmt.Errorf("expression node is nil")
		}
		return ae.evaluateExpression(node.Expression, data)
	default:
		return false, fmt.Errorf("unknown node type: %s", node.Type)
	}
}

// EvaluateNumeric evaluates a RuleNode of type "expression" to a numeric
// result rather than a boolean - the entry point for calculated terms
// (which produce a value, not a pass/fail), as opposed to Evaluate (used by
// validation/MDM rules, which produce a boolean).
func (ae *AdvancedEvaluator) EvaluateNumeric(node RuleNode, data map[string]interface{}) (float64, error) {
	if node.Type != NodeTypeExpression || node.Expression == nil {
		return 0, fmt.Errorf("EvaluateNumeric requires an expression node, got %s", node.Type)
	}
	result, err := ae.evalExprNode(node.Expression.Root, data)
	if err != nil {
		return 0, err
	}
	f, ok := result.(float64)
	if !ok {
		return 0, fmt.Errorf("expression did not evaluate to a number: %T", result)
	}
	return f, nil
}

func (ae *AdvancedEvaluator) evaluateGroup(group *RuleGroup, data map[string]interface{}) (bool, error) {
	if len(group.Conditions) == 0 {
		return true, nil
	}

	switch group.Operator {
	case "AND":
		for _, child := range group.Conditions {
			result, err := ae.Evaluate(child, data)
			if err != nil {
				return false, err
			}
			if !result {
				return false, nil
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
				return true, nil
			}
		}
		return false, nil

	case "NOT":
		for _, child := range group.Conditions {
			result, err := ae.Evaluate(child, data)
			if err != nil {
				return false, err
			}
			if !result {
				return true, nil
			}
		}
		return false, nil

	default:
		return false, fmt.Errorf("unknown group operator: %s", group.Operator)
	}
}

func (ae *AdvancedEvaluator) evaluateCondition(cond *RuleCondition, data map[string]interface{}) (bool, error) {
	field := cond.Field
	if cond.FieldPath != "" {
		field = cond.FieldPath
	}
	conditionMap := map[string]interface{}{
		"type":     "simple",
		"field":    field,
		"operator": cond.Operator,
		"value":    cond.Value,
	}
	return ae.baseEvaluator.EvaluateWithHierarchy(conditionMap, data)
}

func (ae *AdvancedEvaluator) evaluateExpression(expr *Expression, data map[string]interface{}) (bool, error) {
	if expr == nil || expr.Root == nil {
		return false, fmt.Errorf("nil expression")
	}
	result, err := ae.evalExprNode(expr.Root, data)
	if err != nil {
		return false, err
	}
	if b, ok := result.(bool); ok {
		return b, nil
	}
	return false, fmt.Errorf("expression did not evaluate to bool: %T", result)
}

func (ae *AdvancedEvaluator) evalExprNode(node ExprNode, data map[string]interface{}) (any, error) {
	switch n := node.(type) {
	case *BinaryExpr:
		return ae.evalBinaryExpr(n, data)
	case *FieldRef:
		return ae.evalFieldRef(n, data)
	case *Literal:
		return n.Value, nil
	case *FuncCall:
		return ae.evalFuncCall(n, data)
	default:
		return nil, fmt.Errorf("unknown ExprNode type: %T", node)
	}
}

func (ae *AdvancedEvaluator) evalBinaryExpr(be *BinaryExpr, data map[string]interface{}) (any, error) {
	lVal, err := ae.evalExprNode(be.Left, data)
	if err != nil {
		return nil, err
	}
	rVal, err := ae.evalExprNode(be.Right, data)
	if err != nil {
		return nil, err
	}

	a, b, ok := toFloat64(lVal, rVal)
	if !ok {
		return nil, fmt.Errorf("expression operands not numeric: %T, %T", lVal, rVal)
	}

	switch be.Op {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	case "/":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return a / b, nil
	case "==":
		return a == b, nil
	case "!=":
		return a != b, nil
	case ">":
		return a > b, nil
	case "<":
		return a < b, nil
	case ">=":
		return a >= b, nil
	case "<=":
		return a <= b, nil
	default:
		return nil, fmt.Errorf("unsupported expression operator: %s", be.Op)
	}
}

func (ae *AdvancedEvaluator) evalFieldRef(fr *FieldRef, data map[string]interface{}) (any, error) {
	val, found := ae.baseEvaluator.GetFieldValue(fr.Path, data)
	if !found {
		// GetFieldValue (HierarchyResolver.ResolveFieldPath) conflates
		// "key absent" with "key present, value is nil" - both navigate
		// to a nil interface and both come back not-found. That's a real
		// distinction for a top-level field: a SQL NULL column (present
		// key, nil value - the exact case NOT_EMPTY exists to detect)
		// must not error the same way a genuinely missing field does.
		// Scoped to the top-level (no ".") case deliberately - it's the
		// only shape this evaluator's own callers (FuncCall predicates
		// like NOT_EMPTY reading a BO record's own columns) actually hit,
		// and it leaves nested-path resolution (which HierarchyResolver
		// still owns) untouched.
		if !strings.Contains(fr.Path, ".") {
			if v, ok := data[fr.Path]; ok {
				return v, nil
			}
		}
		return nil, fmt.Errorf("field not found: %s", fr.Path)
	}
	return val, nil
}

// Function implementations (SUM/AVG/.../MIRR/format predicates) live in
// library.go's FunctionSpec registry (Library) - the single source of
// truth for native evaluation, SQL pushdown, and editor metadata alike.
// evalFuncCall below just dispatches into it.

func aggFold(args []any, init float64, fold func(acc, v float64) float64) (any, error) {
	vals, err := requireFloatSlice(args)
	if err != nil {
		return nil, err
	}
	acc := init
	for _, v := range vals {
		acc = fold(acc, v)
	}
	return acc, nil
}

// requireFloatSlice flattens the evaluated args into a single []float64,
// accepting either scalars or []float64/[]any-of-numbers per arg (so both
// SUM(field) where field is an array, and SUM(a, b, c) work).
func requireFloatSlice(args []any) ([]float64, error) {
	var out []float64
	for _, a := range args {
		switch v := a.(type) {
		case float64:
			out = append(out, v)
		case []float64:
			out = append(out, v...)
		case []any:
			for _, e := range v {
				f, ok := e.(float64)
				if !ok {
					return nil, fmt.Errorf("non-numeric array element: %T", e)
				}
				out = append(out, f)
			}
		default:
			return nil, fmt.Errorf("expected number or array of numbers, got %T", a)
		}
	}
	return out, nil
}

func (ae *AdvancedEvaluator) evalFuncCall(fc *FuncCall, data map[string]interface{}) (any, error) {
	spec, ok := LookupFunction(fc.Name)
	if !ok {
		return nil, fmt.Errorf("no native evaluator registered for function %q", fc.Name)
	}
	args := make([]any, 0, len(fc.Args))
	for _, a := range fc.Args {
		v, err := ae.evalExprNode(a, data)
		if err != nil {
			return nil, err
		}
		args = append(args, v)
	}
	return spec.Native(args)
}

func toFloat64(a, b any) (float64, float64, bool) {
	var af, bf float64
	ok := true
	switch va := a.(type) {
	case float64:
		af = va
	case int:
		af = float64(va)
	case int64:
		af = float64(va)
	default:
		ok = false
	}
	switch vb := b.(type) {
	case float64:
		bf = vb
	case int:
		bf = float64(vb)
	case int64:
		bf = float64(vb)
	default:
		ok = false
	}
	if !ok {
		return 0, 0, false
	}
	return af, bf, true
}
