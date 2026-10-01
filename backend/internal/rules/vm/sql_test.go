package vm

import (
	"strings"
	"testing"
)

func testResolver(cols map[string]string) ColumnResolver {
	return func(f string) (string, error) {
		if c, ok := cols[f]; ok {
			return `"` + c + `"`, nil
		}
		return "", nil // caller decides; tests use known fields
	}
}

func mustCompile(t *testing.T, node RuleNode) (string, []any) {
	t.Helper()
	b := &ParamBinder{}
	b.Bind("tenant") // reserve $1 for tenant, matching production compile path
	s, err := CompileToSQL(node, testResolver(map[string]string{
		"TargetQuantity": "target_quantity", "LimitPrice": "limit_price", "Side": "side",
	}), b)
	if err != nil {
		t.Fatal(err)
	}
	return s, b.Args()
}

func TestCompileConditionCOALESCE(t *testing.T) {
	node := RuleNode{Type: NodeTypeCondition, Condition: &RuleCondition{
		ID: "c", Field: "TargetQuantity", Operator: "greater_than", Value: 100.0,
	}}
	s, args := mustCompile(t, node)
	if !strings.Contains(s, `COALESCE(("target_quantity" > $2), FALSE)`) {
		t.Fatalf("unexpected sql: %s", s)
	}
	if len(args) != 2 || args[1] != 100.0 {
		t.Fatalf("args: %v", args)
	}
}

func TestCompileIsNullNotCoalesced(t *testing.T) {
	node := RuleNode{Type: NodeTypeCondition, Condition: &RuleCondition{
		ID: "c", Field: "Side", Operator: "is_null",
	}}
	s, _ := mustCompile(t, node)
	if s != `("side" IS NULL)` {
		t.Fatalf("is_null must not be COALESCE-wrapped: %s", s)
	}
}

func TestCompileGroupKleene(t *testing.T) {
	node := RuleNode{Type: NodeTypeGroup, Group: &RuleGroup{ID: "g", Operator: "AND", Conditions: []RuleNode{
		{Type: NodeTypeCondition, Condition: &RuleCondition{ID: "a", Field: "TargetQuantity", Operator: "greater_than", Value: 1.0}},
		{Type: NodeTypeCondition, Condition: &RuleCondition{ID: "b", Field: "LimitPrice", Operator: "less_than", Value: 9.0}},
	}}}
	s, _ := mustCompile(t, node)
	if !strings.Contains(s, ` AND `) {
		t.Fatalf("expected AND join: %s", s)
	}
}

func TestCompileUnsupportedOperatorFailsLoud(t *testing.T) {
	node := RuleNode{Type: NodeTypeCondition, Condition: &RuleCondition{
		ID: "c", Field: "Side", Operator: "regex_match", Value: "^B",
	}}
	b := &ParamBinder{}
	if _, err := CompileToSQL(node, testResolver(nil), b); err == nil {
		t.Fatal("unsupported operator must fail loud (capability matrix)")
	}
}

func TestCompileFuncCallFailsLoud(t *testing.T) {
	node := RuleNode{Type: NodeTypeExpression, Expression: &Expression{Root: &FuncCall{
		Name: "SUM", Args: []ExprNode{&FieldRef{Path: "OrderAllocations"}},
	}}}
	b := &ParamBinder{}
	if _, err := CompileToSQL(node, testResolver(nil), b); err == nil {
		t.Fatal("FuncCall must fail loud in v1")
	}
}

func TestDivisionGuard(t *testing.T) {
	node := RuleNode{Type: NodeTypeExpression, Expression: &Expression{Root: &BinaryExpr{
		Op: ">", Left: &BinaryExpr{Op: "/", Left: &FieldRef{Path: "TargetQuantity"}, Right: &Literal{Value: 0}},
		Right: &Literal{Value: 1},
	}}}
	s, _ := mustCompile(t, node)
	if !strings.Contains(s, "CASE WHEN") {
		t.Fatalf("division must be zero-guarded: %s", s)
	}
}
