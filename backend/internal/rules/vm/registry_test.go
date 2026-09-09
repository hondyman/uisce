package vm

import "testing"

// The registry is hand-maintained data describing two other hand-
// maintained maps (nativeFuncs, starrocksFuncs) - exactly the shape of
// thing that drifts silently. Cross-check directly rather than trust it.
func TestRegistry_MatchesNativeFuncs(t *testing.T) {
	for name := range nativeFuncs {
		cap, ok := LookupCapability(name)
		if !ok {
			t.Errorf("nativeFuncs has %q but the registry doesn't list it", name)
			continue
		}
		if !cap.Native {
			t.Errorf("registry entry for %q says Native: false, but it's in nativeFuncs", name)
		}
	}
	for _, cap := range Registry {
		if cap.Native {
			if _, ok := nativeFuncs[cap.Name]; !ok {
				t.Errorf("registry says %q is Native, but nativeFuncs has no entry for it", cap.Name)
			}
		}
	}
}

func TestRegistry_MatchesStarrocksFuncs(t *testing.T) {
	for name := range starrocksFuncs {
		cap, ok := LookupCapability(name)
		if !ok {
			t.Errorf("starrocksFuncs has %q but the registry doesn't list it", name)
			continue
		}
		if !cap.Pushdownable {
			t.Errorf("registry entry for %q says Pushdownable: false, but it's in starrocksFuncs", name)
		}
	}
	for _, cap := range Registry {
		if cap.Pushdownable {
			if _, ok := starrocksFuncs[cap.Name]; !ok {
				t.Errorf("registry says %q is Pushdownable, but starrocksFuncs has no entry for it", cap.Name)
			}
		}
	}
}

func TestRegistry_IRRXIRRNotPushdownable(t *testing.T) {
	for _, name := range []string{"IRR", "XIRR"} {
		cap, ok := LookupCapability(name)
		if !ok {
			t.Fatalf("%s not in registry", name)
		}
		if cap.Pushdownable {
			t.Errorf("%s must not be pushdownable - no closed-form SQL expansion exists", name)
		}
		if !cap.WASM || !cap.Native {
			t.Errorf("%s should be both Native and WASM - it's supposed to run at tree-walking speed everywhere", name)
		}
	}
}

// TVPI/DPI/MOIC are compositions, not registered functions - this is the
// proof of that claim, not just the comment saying it. Each evaluates
// correctly through the existing engine (SUM + division, both already
// supported) with zero new function code.
func TestTVPI_DPI_MOIC_AsCompositions(t *testing.T) {
	ae := NewAdvancedEvaluator()

	// TVPI = (Distributions + Residual Value) / Paid-In Capital
	//   Distributions = SUM(distributions[]), Residual = SUM(residual_values[]),
	//   Paid-In = SUM(paid_in[])
	tvpi := RuleNode{
		Type: NodeTypeExpression,
		Expression: &Expression{
			Root: &BinaryExpr{
				Op: "/",
				Left: &BinaryExpr{
					Op:    "+",
					Left:  &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "distributions"}}},
					Right: &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "residual_values"}}},
				},
				Right: &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "paid_in"}}},
			},
		},
	}
	data := map[string]interface{}{
		"distributions":   []interface{}{50.0, 30.0}, // 80
		"residual_values": []interface{}{40.0},       // 40
		"paid_in":         []interface{}{100.0},      // 100
	}
	got, err := ae.EvaluateNumeric(tvpi, data)
	if err != nil {
		t.Fatal(err)
	}
	// TVPI = (80 + 40) / 100 = 1.20
	assertNear(t, got, 1.20, 1e-9, "TVPI composition")

	// DPI = Distributions / Paid-In Capital
	dpi := RuleNode{
		Type: NodeTypeExpression,
		Expression: &Expression{
			Root: &BinaryExpr{
				Op:    "/",
				Left:  &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "distributions"}}},
				Right: &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "paid_in"}}},
			},
		},
	}
	got, err = ae.EvaluateNumeric(dpi, data)
	if err != nil {
		t.Fatal(err)
	}
	// DPI = 80 / 100 = 0.80
	assertNear(t, got, 0.80, 1e-9, "DPI composition")

	// MOIC = Total Value / Invested Capital, same shape as TVPI with
	// different field names - the point being it's the same composition
	// pattern, not a different function.
	moic := RuleNode{
		Type: NodeTypeExpression,
		Expression: &Expression{
			Root: &BinaryExpr{
				Op:    "/",
				Left:  &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "total_value"}}},
				Right: &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "invested_capital"}}},
			},
		},
	}
	moicData := map[string]interface{}{
		"total_value":      []interface{}{240.0},
		"invested_capital": []interface{}{100.0},
	}
	got, err = ae.EvaluateNumeric(moic, moicData)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got, 2.40, 1e-9, "MOIC composition")
}

// The same TVPI composition must also compile to SQL - proving the
// "no new solver needed" claim covers pushdown too, not just native
// evaluation. SUM is already in starrocksFuncs; "/" and "+" are
// evalBinaryExpr's own operators, already shared by CompileToSQL.
func TestTVPI_CompilesToSQL(t *testing.T) {
	expr := &Expression{
		Root: &BinaryExpr{
			Op: "/",
			Left: &BinaryExpr{
				Op:    "+",
				Left:  &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "Distributions"}}},
				Right: &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "ResidualValue"}}},
			},
			Right: &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "PaidInCapital"}}},
		},
	}
	resolve := func(field string) (string, error) {
		cols := map[string]string{
			"Distributions": "distributions", "ResidualValue": "residual_value", "PaidInCapital": "paid_in_capital",
		}
		return cols[field], nil
	}
	sql, err := CompileToSQL(expr, resolve)
	if err != nil {
		t.Fatal(err)
	}
	want := "((SUM(distributions) + SUM(residual_value)) / SUM(paid_in_capital))"
	if sql != want {
		t.Fatalf("got %q, want %q", sql, want)
	}
}
