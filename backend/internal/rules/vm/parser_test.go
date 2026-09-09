package vm

import "testing"

func TestParseExpression_NumberLiteral(t *testing.T) {
	expr, err := ParseExpression("42.5")
	if err != nil {
		t.Fatal(err)
	}
	lit, ok := expr.Root.(*Literal)
	if !ok {
		t.Fatalf("expected *Literal, got %T", expr.Root)
	}
	if lit.Value != 42.5 {
		t.Errorf("got %v, want 42.5", lit.Value)
	}
}

func TestParseExpression_FieldRef(t *testing.T) {
	expr, err := ParseExpression("ExecQuantity")
	if err != nil {
		t.Fatal(err)
	}
	fr, ok := expr.Root.(*FieldRef)
	if !ok {
		t.Fatalf("expected *FieldRef, got %T", expr.Root)
	}
	if fr.Path != "ExecQuantity" {
		t.Errorf("got %q, want %q", fr.Path, "ExecQuantity")
	}
}

func TestParseExpression_DottedFieldRef(t *testing.T) {
	expr, err := ParseExpression("client.risk_score")
	if err != nil {
		t.Fatal(err)
	}
	fr, ok := expr.Root.(*FieldRef)
	if !ok || fr.Path != "client.risk_score" {
		t.Fatalf("got %#v, want FieldRef{client.risk_score}", expr.Root)
	}
}

// TestParseExpression_OperatorPrecedence proves multiplication/division
// bind tighter than addition/subtraction, and that parenthesization
// overrides it - "2 + 3 * 4" should compile to the same structure as if
// hand-built with * nested inside +, not (2+3)*4.
func TestParseExpression_OperatorPrecedence(t *testing.T) {
	expr, err := ParseExpression("2 + 3 * 4")
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.Root.(*BinaryExpr)
	if !ok || be.Op != "+" {
		t.Fatalf("expected top-level '+', got %#v", expr.Root)
	}
	left, ok := be.Left.(*Literal)
	if !ok || left.Value != 2 {
		t.Fatalf("expected left=2, got %#v", be.Left)
	}
	right, ok := be.Right.(*BinaryExpr)
	if !ok || right.Op != "*" {
		t.Fatalf("expected right='3 * 4', got %#v", be.Right)
	}
}

func TestParseExpression_ParenOverridesPrecedence(t *testing.T) {
	expr, err := ParseExpression("(2 + 3) * 4")
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.Root.(*BinaryExpr)
	if !ok || be.Op != "*" {
		t.Fatalf("expected top-level '*', got %#v", expr.Root)
	}
	left, ok := be.Left.(*BinaryExpr)
	if !ok || left.Op != "+" {
		t.Fatalf("expected left='2 + 3', got %#v", be.Left)
	}
}

func TestParseExpression_UnaryMinus(t *testing.T) {
	expr, err := ParseExpression("-5 + 3")
	if err != nil {
		t.Fatal(err)
	}
	// "-5 + 3" == "(0 - 5) + 3" == -2, proven via evaluation rather than
	// asserting the exact desugared AST shape (an implementation detail).
	ae := NewAdvancedEvaluator()
	result, err := ae.EvaluateNumeric(RuleNode{Type: NodeTypeExpression, Expression: expr}, nil)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, result, -2, 1e-12, "-5 + 3")
}

func TestParseExpression_FuncCallSingleArg(t *testing.T) {
	expr, err := ParseExpression("SUM(ExecQuantity)")
	if err != nil {
		t.Fatal(err)
	}
	fc, ok := expr.Root.(*FuncCall)
	if !ok || fc.Name != "SUM" {
		t.Fatalf("expected FuncCall SUM, got %#v", expr.Root)
	}
	if len(fc.Args) != 1 {
		t.Fatalf("expected 1 arg, got %d", len(fc.Args))
	}
	fr, ok := fc.Args[0].(*FieldRef)
	if !ok || fr.Path != "ExecQuantity" {
		t.Fatalf("expected arg FieldRef(ExecQuantity), got %#v", fc.Args[0])
	}
}

func TestParseExpression_FuncCallWithExpressionArg(t *testing.T) {
	// The exact shape cmd/verify_calc_measure hand-built as JSON for
	// "Gross Notional": SUM(ExecQuantity * ExecPrice).
	expr, err := ParseExpression("SUM(ExecQuantity * ExecPrice)")
	if err != nil {
		t.Fatal(err)
	}
	sql, err := CompileToSQL(expr, resolveIdentity)
	if err != nil {
		t.Fatal(err)
	}
	want := "SUM((ExecQuantity * ExecPrice))"
	if sql != want {
		t.Errorf("got %q, want %q", sql, want)
	}
}

func TestParseExpression_MultiArgFuncCall(t *testing.T) {
	expr, err := ParseExpression("NPV(rate, cash_flow)")
	if err != nil {
		t.Fatal(err)
	}
	fc, ok := expr.Root.(*FuncCall)
	if !ok || fc.Name != "NPV" || len(fc.Args) != 2 {
		t.Fatalf("got %#v", expr.Root)
	}
}

func TestParseExpression_NestedFuncCalls(t *testing.T) {
	expr, err := ParseExpression("SUM(distributions) / SUM(paid_in)")
	if err != nil {
		t.Fatal(err)
	}
	be, ok := expr.Root.(*BinaryExpr)
	if !ok || be.Op != "/" {
		t.Fatalf("got %#v", expr.Root)
	}
	if _, ok := be.Left.(*FuncCall); !ok {
		t.Fatalf("expected left to be a FuncCall, got %#v", be.Left)
	}
	if _, ok := be.Right.(*FuncCall); !ok {
		t.Fatalf("expected right to be a FuncCall, got %#v", be.Right)
	}
}

// TestParseExpression_ComparisonEvaluatesBoolean proves the top-level
// grammar's one allowed comparison produces a real boolean RuleNode,
// evaluable via Evaluate (not just EvaluateNumeric) - the shape a
// validation rule authored as text needs, e.g. "XIRR(...) > 0.15".
func TestParseExpression_ComparisonEvaluatesBoolean(t *testing.T) {
	cases := []struct {
		src  string
		want bool
	}{
		{"5 > 3", true},
		{"5 < 3", false},
		{"5 == 5", true},
		{"5 != 5", false},
		{"SUM(qty) >= 10", true},
		{"NOT_EMPTY(name)", true},
	}
	ae := NewAdvancedEvaluator()
	data := map[string]interface{}{
		"qty":  []interface{}{4.0, 6.0},
		"name": "hello",
	}
	for _, c := range cases {
		t.Run(c.src, func(t *testing.T) {
			expr, err := ParseExpression(c.src)
			if err != nil {
				t.Fatal(err)
			}
			got, err := ae.Evaluate(RuleNode{Type: NodeTypeExpression, Expression: expr}, data)
			if err != nil {
				t.Fatal(err)
			}
			if got != c.want {
				t.Errorf("%s: got %v, want %v", c.src, got, c.want)
			}
		})
	}
}

func TestParseExpression_ComparisonNotChainable(t *testing.T) {
	// "1 < 2 < 3" is not valid grammar here - a second comparison isn't
	// consumed, so it trails as unparsed input and the parser reports
	// that rather than silently accepting Python-style chaining (which
	// evalBinaryExpr's bool-returning "<" couldn't feed into another "<"
	// against a float64 right-hand side anyway).
	_, err := ParseExpression("1 < 2 < 3")
	if err == nil {
		t.Fatal("expected a parse error for a chained comparison")
	}
}

func TestParseExpression_SyntaxErrors(t *testing.T) {
	cases := []string{
		"",
		"SUM(",
		"SUM(1,)",
		"1 +",
		"1 + + 2",
		"(1 + 2",
		"1 2",
		"$foo",
	}
	for _, src := range cases {
		t.Run(src, func(t *testing.T) {
			_, err := ParseExpression(src)
			if err == nil {
				t.Fatalf("expected a parse error for %q, got none", src)
			}
			var perr *ParseError
			if pe, ok := err.(*ParseError); ok {
				perr = pe
			}
			if perr == nil {
				t.Fatalf("expected a *ParseError, got %T: %v", err, err)
			}
		})
	}
}

// TestParseExpression_XIRRRoundTrip_MatchesGoldenFixture is the proof
// this parser exists for: the exact Microsoft-published XIRR example
// from irr_excel_fixtures_test.go, authored as TEXT instead of a
// hand-built Go struct, parsed, and evaluated - and it must land on the
// identical published result the hand-built-AST version does. If the
// parser produced a subtly different tree (wrong arg order, wrong
// nesting), this is what would catch it - a structural assertion on the
// parsed AST wouldn't.
func TestParseExpression_XIRRRoundTrip_MatchesGoldenFixture(t *testing.T) {
	expr, err := ParseExpression("XIRR(cash_flows, dates)")
	if err != nil {
		t.Fatal(err)
	}
	ae := NewAdvancedEvaluator()
	result, err := ae.EvaluateNumeric(RuleNode{Type: NodeTypeExpression, Expression: expr}, map[string]interface{}{
		"cash_flows": []interface{}{-10000.0, 2750.0, 4250.0, 3250.0, 2750.0},
		"dates":      []interface{}{0.0, 60.0, 303.0, 411.0, 456.0},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, result, 0.373362535, 1e-6, "parsed XIRR text (vs. Microsoft's published XIRR example)")
}

// TestParseExpression_CompileToSQLRoundTrip proves the parser's output
// is not just evaluable but pushdown-compilable too - the same
// TVPI-shaped expression from TestCompileToSQL_AggregateFuncCall
// (sql_compiler_test.go), authored as text.
func TestParseExpression_CompileToSQLRoundTrip(t *testing.T) {
	expr, err := ParseExpression("(SUM(cumulative_distributions) + SUM(remaining_value)) / SUM(paid_in_capital)")
	if err != nil {
		t.Fatal(err)
	}
	got, err := CompileToSQL(expr, resolveIdentity)
	if err != nil {
		t.Fatal(err)
	}
	want := "((SUM(cumulative_distributions) + SUM(remaining_value)) / SUM(paid_in_capital))"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// TestParseExpression_ErrorHasPosition proves the ParseError reports a
// real, checkable byte offset - the whole reason it's a typed error
// instead of a plain string, since an editor needs a position to
// underline.
func TestParseExpression_ErrorHasPosition(t *testing.T) {
	_, err := ParseExpression("SUM(1 $ 2)")
	perr, ok := err.(*ParseError)
	if !ok {
		t.Fatalf("expected *ParseError, got %T: %v", err, err)
	}
	if perr.Pos != 6 {
		t.Errorf("got Pos=%d, want 6 (the '$' character)", perr.Pos)
	}
}
