package vm

import (
	"errors"
	"strings"
	"testing"
)

func resolveIdentity(path string) (string, error) {
	return path, nil
}

func TestCompileToSQL_BinaryArithmetic(t *testing.T) {
	// (revenue - cogs) / revenue, same shape as TVPI/margin-style calc terms.
	expr := &Expression{
		Root: &BinaryExpr{
			Op: "/",
			Left: &BinaryExpr{
				Op:    "-",
				Left:  &FieldRef{Path: "revenue"},
				Right: &FieldRef{Path: "cogs"},
			},
			Right: &FieldRef{Path: "revenue"},
		},
	}
	got, err := CompileToSQL(expr, resolveIdentity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "((revenue - cogs) / revenue)"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCompileToSQL_AggregateFuncCall(t *testing.T) {
	// TVPI: (SUM(cumulative_distributions) + SUM(remaining_value)) / SUM(paid_in_capital)
	expr := &Expression{
		Root: &BinaryExpr{
			Op: "/",
			Left: &BinaryExpr{
				Op:    "+",
				Left:  &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "cumulative_distributions"}}},
				Right: &FuncCall{Name: "sum", Args: []ExprNode{&FieldRef{Path: "remaining_value"}}}, // lowercase, must still resolve
			},
			Right: &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "paid_in_capital"}}},
		},
	}
	got, err := CompileToSQL(expr, resolveIdentity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "((SUM(cumulative_distributions) + SUM(remaining_value)) / SUM(paid_in_capital))"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestCompileToSQL_NPVExpansion(t *testing.T) {
	expr := &Expression{
		Root: &FuncCall{
			Name: "NPV",
			Args: []ExprNode{
				&Literal{Value: 0.08},
				&FieldRef{Path: "cash_flow"},
			},
		},
	}
	got, err := CompileToSQL(expr, resolveIdentity)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(got, "SUM(cash_flow /") || !strings.Contains(got, "POWER(1 + 0.08") {
		t.Errorf("NPV expansion looks wrong: %q", got)
	}
}

func TestCompileToSQL_UnsupportedFunction(t *testing.T) {
	expr := &Expression{
		Root: &FuncCall{
			Name: "XIRR",
			Args: []ExprNode{&FieldRef{Path: "cash_flow"}, &FieldRef{Path: "dates"}},
		},
	}
	_, err := CompileToSQL(expr, resolveIdentity)
	if err == nil {
		t.Fatal("expected error for unsupported function, got nil")
	}
	var unsupported *ErrUnsupportedFunction
	if !errors.As(err, &unsupported) {
		t.Errorf("expected ErrUnsupportedFunction, got %T: %v", err, err)
	}
	if unsupported.Name != "XIRR" {
		t.Errorf("got function name %q, want %q", unsupported.Name, "XIRR")
	}
}

func TestCompileToSQL_FieldResolutionError(t *testing.T) {
	expr := &Expression{Root: &FieldRef{Path: "missing_field"}}
	_, err := CompileToSQL(expr, func(path string) (string, error) {
		return "", errors.New("not found")
	})
	if err == nil {
		t.Fatal("expected error for unresolved field, got nil")
	}
}

func TestCompileToSQL_UnsupportedOperator(t *testing.T) {
	expr := &Expression{
		Root: &BinaryExpr{Op: "%", Left: &Literal{Value: 1}, Right: &Literal{Value: 2}},
	}
	_, err := CompileToSQL(expr, resolveIdentity)
	if err == nil {
		t.Fatal("expected error for unsupported operator, got nil")
	}
}
