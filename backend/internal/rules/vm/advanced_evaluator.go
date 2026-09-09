package vm

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
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

// nativeFuncs is the native (tree-walking) function registry, evaluated
// directly against a single record's data - the counterpart to
// starrocksFuncs in sql_compiler.go. Aggregate functions (SUM/AVG/MIN/MAX)
// expect their argument to resolve to a []float64 (e.g. a FieldRef pointing
// at an array-valued field), since there is no row set to aggregate over
// here. NPV mirrors the SQL expansion: SUM(cf_i / (1+rate)^i).
var nativeFuncs = map[string]func(args []any) (any, error){
	"SUM": func(args []any) (any, error) {
		return aggFold(args, 0, func(acc, v float64) float64 { return acc + v })
	},
	"AVG": func(args []any) (any, error) {
		vals, err := requireFloatSlice(args)
		if err != nil {
			return nil, err
		}
		if len(vals) == 0 {
			return 0.0, nil
		}
		sum := 0.0
		for _, v := range vals {
			sum += v
		}
		return sum / float64(len(vals)), nil
	},
	"MIN": func(args []any) (any, error) {
		vals, err := requireFloatSlice(args)
		if err != nil || len(vals) == 0 {
			return nil, err
		}
		m := vals[0]
		for _, v := range vals[1:] {
			if v < m {
				m = v
			}
		}
		return m, nil
	},
	"MAX": func(args []any) (any, error) {
		vals, err := requireFloatSlice(args)
		if err != nil || len(vals) == 0 {
			return nil, err
		}
		m := vals[0]
		for _, v := range vals[1:] {
			if v > m {
				m = v
			}
		}
		return m, nil
	},
	"NPV": func(args []any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("NPV expects 2 args (rate, cash_flows), got %d", len(args))
		}
		rate, ok := args[0].(float64)
		if !ok {
			return nil, fmt.Errorf("NPV rate must be numeric, got %T", args[0])
		}
		cashFlows, err := requireFloatSlice(args[1:])
		if err != nil {
			return nil, err
		}
		npv := 0.0
		for i, cf := range cashFlows {
			npv += cf / math.Pow(1+rate, float64(i))
		}
		return npv, nil
	},

	// Field-format predicates, added for the catalog_validation_rules ->
	// rule_ast migration (backend/cmd/migrate_validation_rules). These are
	// the FuncCall side of the 12-operator vocabulary found in that
	// table's condition_json: pure comparisons (greater_than, ...) map to
	// Condition nodes via ConditionEvaluator; format/type validators
	// (is_uuid, is_date, max_length, ...) had no home in either Condition
	// or the arithmetic ExprNode set, so they're predicates here instead -
	// the function registry growing exactly the kind of function it was
	// designed for, now authorable in the editor like SUM/NPV. All treat a
	// present-but-JSON-null field value as failing the predicate (false,
	// no error) rather than a type error - a genuinely absent field is a
	// separate case, already an error from evalFieldRef before these ever
	// run.
	"NOT_EMPTY": func(args []any) (any, error) {
		v, err := require1(args, "NOT_EMPTY")
		if err != nil {
			return nil, err
		}
		if v == nil {
			return false, nil
		}
		s, ok := v.(string)
		return !ok || s != "", nil
	},
	"IS_INTEGER": func(args []any) (any, error) {
		v, err := require1(args, "IS_INTEGER")
		if err != nil {
			return nil, err
		}
		switch n := v.(type) {
		case int, int32, int64:
			return true, nil
		case float64:
			return n == math.Trunc(n), nil
		case string:
			_, err := strconv.ParseInt(n, 10, 64)
			return err == nil, nil
		default:
			return false, nil
		}
	},
	"IS_NUMBER": func(args []any) (any, error) {
		v, err := require1(args, "IS_NUMBER")
		if err != nil {
			return nil, err
		}
		switch n := v.(type) {
		case int, int32, int64, float32, float64:
			return true, nil
		case string:
			_, err := strconv.ParseFloat(n, 64)
			return err == nil, nil
		default:
			return false, nil
		}
	},
	"IS_BOOLEAN": func(args []any) (any, error) {
		v, err := require1(args, "IS_BOOLEAN")
		if err != nil {
			return nil, err
		}
		switch b := v.(type) {
		case bool:
			return true, nil
		case string:
			return b == "true" || b == "false", nil
		default:
			return false, nil
		}
	},
	"IS_UUID": func(args []any) (any, error) {
		v, err := require1(args, "IS_UUID")
		if err != nil {
			return nil, err
		}
		s, ok := v.(string)
		if !ok {
			return false, nil
		}
		return uuidPattern.MatchString(s), nil
	},
	"IS_DATE": func(args []any) (any, error) {
		v, err := require1(args, "IS_DATE")
		if err != nil {
			return nil, err
		}
		s, ok := v.(string)
		if !ok {
			return false, nil
		}
		_, parseErr := time.Parse("2006-01-02", s)
		return parseErr == nil, nil
	},
	"IS_DATETIME": func(args []any) (any, error) {
		v, err := require1(args, "IS_DATETIME")
		if err != nil {
			return nil, err
		}
		s, ok := v.(string)
		if !ok {
			return false, nil
		}
		_, parseErr := time.Parse(time.RFC3339, s)
		return parseErr == nil, nil
	},
	"IS_JSON": func(args []any) (any, error) {
		v, err := require1(args, "IS_JSON")
		if err != nil {
			return nil, err
		}
		s, ok := v.(string)
		if !ok {
			return false, nil
		}
		return json.Valid([]byte(s)), nil
	},
	"MAX_LENGTH": func(args []any) (any, error) {
		if len(args) != 2 {
			return nil, fmt.Errorf("MAX_LENGTH expects 2 args (field, max), got %d", len(args))
		}
		max, ok := args[1].(float64)
		if !ok {
			return nil, fmt.Errorf("MAX_LENGTH's second arg must be numeric, got %T", args[1])
		}
		if args[0] == nil {
			return true, nil
		}
		s, ok := args[0].(string)
		if !ok {
			return false, nil
		}
		return float64(len(s)) <= max, nil
	},
}

// require1 validates a predicate received exactly one argument and returns
// it, nil-safe (a resolved FieldRef for an absent/null field comes through
// as a nil any, which is a valid input to these predicates, not an error).
func require1(args []any, fnName string) (any, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("%s expects 1 arg, got %d", fnName, len(args))
	}
	return args[0], nil
}

var uuidPattern = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

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
	fn, ok := nativeFuncs[strings.ToUpper(fc.Name)]
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
	return fn(args)
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
