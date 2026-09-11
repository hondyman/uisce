package vm

import (
	"fmt"
	"strconv"
)

// ColumnResolver maps a FieldRef.Path (a semantic term / BO field name) to
// the real physical column expression it should compile to, e.g. resolving
// "PlacementID" against a specific BO's MAPS_TO catalog edges. Callers own
// the resolution strategy (BO-scoped, tenant-scoped, etc.) — this compiler
// only walks the AST and asks for each leaf field.
type ColumnResolver func(fieldPath string) (string, error)

// SQL pushdown functions (which functions CompileToSQL can push down, and
// what SQL each renders to) are registered once, in library.go's
// FunctionSpec.SQLEmit map, alongside each function's native
// implementation — not maintained here as a second, parallel map.
//
// This intentionally does NOT attempt to cover every calculated-term
// formula in the catalog (see docs/oms-calc-engine-handoff.md) — only
// functions that are (a) genuinely computable as a single SQL expression
// over grouped rows and (b) registered with a DialectStarRocks emitter.
// Anything else fails loudly via ErrUnsupportedFunction rather than
// emitting wrong SQL.

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
// plus FuncCall nodes for functions with a DialectStarRocks entry in
// library.go's Library.
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
		spec, ok := LookupFunction(n.Name)
		if !ok || !spec.Pushdownable(DialectStarRocks) {
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
		return spec.SQLEmit[DialectStarRocks](args)

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
