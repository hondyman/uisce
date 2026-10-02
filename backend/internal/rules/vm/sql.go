package vm

import (
	"fmt"
	"strings"
)

// ParamBinder accumulates parameterized literals, emitting $n placeholders.
// Bind order defines parameter order — the pushdown executor binds the
// tenant ID first so it is always $1.
type ParamBinder struct {
	args []any
}

func (b *ParamBinder) Bind(v any) string {
	b.args = append(b.args, v)
	return fmt.Sprintf("$%d", len(b.args))
}

func (b *ParamBinder) Args() []any { return b.args }

var sqlOperatorMap = map[string]string{
	"greater_than":     ">",
	"less_than":        "<",
	"greater_or_equal": ">=",
	"less_or_equal":    "<=",
	"equals":           "=",
	"not_equals":       "<>",
}

// CompileToSQL translates a rule into a PostgreSQL boolean predicate where
// TRUE = record passes. Semantics are pinned to AdvancedEvaluator (see the
// parity table in the design doc); the SQL conformance harness enforces it
// against the same fixtures as the in-memory engine.
func CompileToSQL(node RuleNode, resolve ColumnResolver, b *ParamBinder) (string, error) {
	switch node.Type {
	case NodeTypeCondition:
		if node.Condition == nil {
			return "", fmt.Errorf("sql pushdown: condition node with nil condition")
		}
		return compileConditionSQL(*node.Condition, resolve, b)
	case NodeTypeExpression:
		if node.Expression == nil {
			return "", fmt.Errorf("sql pushdown: expression node with nil expression")
		}
		return compileExprSQL(node.Expression.Root, resolve, b)
	case NodeTypeGroup:
		if node.Group == nil {
			return "", fmt.Errorf("sql pushdown: group node with nil group")
		}
		return compileGroupSQL(*node.Group, resolve, b)
	default:
		return "", fmt.Errorf("sql pushdown: unsupported node type %q", node.Type)
	}
}

func compileConditionSQL(c RuleCondition, resolve ColumnResolver, b *ParamBinder) (string, error) {
	col, err := resolve(c.Field)
	if err != nil {
		return "", err
	}
	op := strings.ToLower(c.Operator)
	switch op {
	case "is_null":
		return fmt.Sprintf("(%s IS NULL)", col), nil
	case "is_not_null":
		return fmt.Sprintf("(%s IS NOT NULL)", col), nil
	case "between":
		var low, high any
		if c.SecondValue != nil {
			low = c.Value
			high = c.SecondValue
		} else if bounds, ok := c.Value.([]any); ok && len(bounds) == 2 {
			low = bounds[0]
			high = bounds[1]
		} else {
			return "", fmt.Errorf("sql pushdown: 'between' requires two bounds, got %v", c.Value)
		}
		inner := fmt.Sprintf("%s BETWEEN %s AND %s", col, b.Bind(low), b.Bind(high))
		return fmt.Sprintf("COALESCE((%s), FALSE)", inner), nil
	case "in":
		vals, ok := c.Value.([]any)
		if !ok {
			return "", fmt.Errorf("sql pushdown: 'in' requires array value, got %T", c.Value)
		}
		if len(vals) == 0 {
			// Empty IN list: no value matches; NULL col also non-matching.
			return "(FALSE)", nil
		}
		ph := make([]string, len(vals))
		for i, v := range vals {
			ph[i] = b.Bind(v)
		}
		return fmt.Sprintf("COALESCE((%s IN (%s)), FALSE)", col, strings.Join(ph, ", ")), nil
	case "contains", "starts_with", "ends_with":
		s, ok := c.Value.(string)
		if !ok {
			return "", fmt.Errorf("sql pushdown: %q requires string value, got %T", op, c.Value)
		}
		esc := escapeLike(s)
		var pat string
		switch op {
		case "contains":
			pat = "%" + esc + "%"
		case "starts_with":
			pat = esc + "%"
		case "ends_with":
			pat = "%" + esc
		}
		return fmt.Sprintf("COALESCE((%s LIKE %s), FALSE)", col, b.Bind(pat)), nil
	}
	std, ok := sqlOperatorMap[op]
	if !ok {
		return "", fmt.Errorf("sql pushdown: operator %q not supported (capability matrix — row-level engine handles it)", c.Operator)
	}
	return fmt.Sprintf("COALESCE((%s %s %s), FALSE)", col, std, b.Bind(c.Value)), nil
}

func compileGroupSQL(g RuleGroup, resolve ColumnResolver, b *ParamBinder) (string, error) {
	switch strings.ToUpper(g.Operator) {
	case "AND", "OR":
		if len(g.Conditions) == 0 {
			return "", fmt.Errorf("sql pushdown: empty %s group", g.Operator)
		}
		parts := make([]string, 0, len(g.Conditions))
		for _, child := range g.Conditions {
			s, err := CompileToSQL(child, resolve, b)
			if err != nil {
				return "", err
			}
			parts = append(parts, s)
		}
		return "(" + strings.Join(parts, " "+strings.ToUpper(g.Operator)+" ") + ")", nil
	case "NOT":
		if len(g.Conditions) != 1 {
			return "", fmt.Errorf("sql pushdown: NOT group must contain exactly one child, got %d", len(g.Conditions))
		}
		s, err := CompileToSQL(g.Conditions[0], resolve, b)
		if err != nil {
			return "", err
		}
		return "(NOT " + s + ")", nil
	default:
		return "", fmt.Errorf("sql pushdown: group operator %q not supported", g.Operator)
	}
}

func compileExprSQL(n ExprNode, resolve ColumnResolver, b *ParamBinder) (string, error) {
	switch x := n.(type) {
	case *FieldRef:
		return resolve(x.Path)
	case *Literal:
		return b.Bind(x.Value), nil
	case *BinaryExpr:
		l, err := compileExprSQL(x.Left, resolve, b)
		if err != nil {
			return "", err
		}
		r, err := compileExprSQL(x.Right, resolve, b)
		if err != nil {
			return "", err
		}
		// Division-by-zero guard: Postgres aborts the whole query on x/0,
		// while the Go evaluator errors per-record. Emit NULL instead, which
		// the executor classifies as rule_error — matching Go semantics.
		if x.Op == "/" {
			return fmt.Sprintf("(CASE WHEN (%s) = 0 THEN NULL ELSE (%s) / (%s) END)", r, l, r), nil
		}
		op, ok := exprOpSQL[x.Op]
		if !ok {
			return "", fmt.Errorf("sql pushdown: expression operator %q not supported", x.Op)
		}
		return fmt.Sprintf("(%s %s %s)", l, op, r), nil
	case *FuncCall:
		return "", fmt.Errorf("sql pushdown: function %q not supported in v1 (capability matrix)", x.Name)
	default:
		return "", fmt.Errorf("sql pushdown: unsupported expression node %T", n)
	}
}

var exprOpSQL = map[string]string{
	"+": "+", "-": "-", "*": "*",
	">": ">", "<": "<", ">=": ">=", "<=": "<=",
	"=": "=", "==": "=", "!=": "<>", "<>": "<>",
	"AND": "AND", "OR": "OR",
}
