package vm

import (
	"fmt"
	"strconv"
	"strings"
)

// ColumnResolver maps a FieldRef.Path (a semantic term / BO field name) to
// the real physical column expression it should compile to, e.g. resolving
// "PlacementID" against a specific BO's MAPS_TO catalog edges. Callers own
// the resolution strategy (BO-scoped, tenant-scoped, etc.) — this compiler
// only walks the AST and asks for each leaf field.
type ColumnResolver func(fieldPath string) (string, error)

// sqlFunc renders a FuncCall's already-compiled argument SQL into a full
// SQL expression for that function, in the target dialect.
type sqlFunc func(args []string) (string, error)

// starrocksFuncs is the set of functions CompileToSQL can push down to
// StarRocks. Pure aggregates map straight through; NPV is hand-expanded
// because StarRocks has no native NPV() function.
//
// This intentionally does NOT attempt to cover every calculated-term
// formula in the catalog (see docs/oms-calc-engine-handoff.md) — only
// functions that are (a) genuinely computable as a single SQL expression
// over grouped rows and (b) registered here explicitly. Anything else
// fails loudly via ErrUnsupportedFunction rather than emitting wrong SQL.
var starrocksFuncs = map[string]sqlFunc{
	"SUM": aggPassthrough("SUM"),
	"AVG": aggPassthrough("AVG"),
	"MIN": aggPassthrough("MIN"),
	"MAX": aggPassthrough("MAX"),
	// NPV(rate, cash_flow) over grouped rows, discounting each row's
	// cash_flow by its ordinal position within the group (ROW_NUMBER,
	// 0-indexed) — the standard finite-cash-flow-series NPV formula
	// SUM(cash_flow_i / (1+rate)^i). Requires `rate` to be a constant
	// literal (StarRocks window functions can't reference an aggregate
	// rate per-row here); a per-row/variable rate needs a different
	// formulation and isn't supported by this expansion.
	"NPV": func(args []string) (string, error) {
		if len(args) != 2 {
			return "", fmt.Errorf("NPV expects 2 args (rate, cash_flow), got %d", len(args))
		}
		rate, cashFlow := args[0], args[1]
		return fmt.Sprintf(
			"SUM(%s / POWER(1 + %s, ROW_NUMBER() OVER (ORDER BY %s) - 1))",
			cashFlow, rate, cashFlow,
		), nil
	},
}

func aggPassthrough(sqlName string) sqlFunc {
	return func(args []string) (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("%s expects 1 arg, got %d", sqlName, len(args))
		}
		return fmt.Sprintf("%s(%s)", sqlName, args[0]), nil
	}
}

// ErrUnsupportedFunction is wrapped into the error returned by CompileToSQL
// when the AST references a function this compiler doesn't know how to
// push down to SQL.
type ErrUnsupportedFunction struct {
	Name string
}

func (e *ErrUnsupportedFunction) Error() string {
	return fmt.Sprintf("no SQL pushdown registered for function %q", e.Name)
}

// CompileToSQL walks a rule/calc Expression AST and emits an equivalent SQL
// expression, resolving each FieldRef via resolveColumn. It supports the
// same binary operators as the VM compiler (vm_compiler.go's compileExprNode)
// plus FuncCall nodes for functions registered in starrocksFuncs.
func CompileToSQL(expr *Expression, resolveColumn ColumnResolver) (string, error) {
	if expr == nil || expr.Root == nil {
		return "", fmt.Errorf("nil expression")
	}
	return compileNodeToSQL(expr.Root, resolveColumn)
}

func compileNodeToSQL(node ExprNode, resolveColumn ColumnResolver) (string, error) {
	switch n := node.(type) {
	case *BinaryExpr:
		left, err := compileNodeToSQL(n.Left, resolveColumn)
		if err != nil {
			return "", err
		}
		right, err := compileNodeToSQL(n.Right, resolveColumn)
		if err != nil {
			return "", err
		}
		op, err := binarySQLOp(n.Op)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s %s %s)", left, op, right), nil

	case *FieldRef:
		col, err := resolveColumn(n.Path)
		if err != nil {
			return "", fmt.Errorf("resolving field %q: %w", n.Path, err)
		}
		return col, nil

	case *Literal:
		return strconv.FormatFloat(n.Value, 'g', -1, 64), nil

	case *FuncCall:
		fn, ok := starrocksFuncs[strings.ToUpper(n.Name)]
		if !ok {
			return "", &ErrUnsupportedFunction{Name: n.Name}
		}
		args := make([]string, 0, len(n.Args))
		for _, a := range n.Args {
			argSQL, err := compileNodeToSQL(a, resolveColumn)
			if err != nil {
				return "", err
			}
			args = append(args, argSQL)
		}
		return fn(args)

	default:
		return "", fmt.Errorf("unknown ExprNode type: %T", node)
	}
}

func binarySQLOp(op string) (string, error) {
	switch op {
	case "+", "-", "*", "/":
		return op, nil
	case "==":
		return "=", nil
	case "!=":
		return "<>", nil
	case ">", "<", ">=", "<=":
		return op, nil
	default:
		return "", fmt.Errorf("unsupported expression operator: %s", op)
	}
}
