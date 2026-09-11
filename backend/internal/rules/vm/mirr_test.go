package vm

import "testing"

// MIRR golden tests: exact-by-construction and independent cross-check.
// The third leg (a real Microsoft-published fixture, matching IRR/XIRR's
// treatment in irr_excel_fixtures_test.go) lives in
// mirr_excel_fixtures_test.go.

func TestMIRR_ExactByConstruction_SingleNegativeSinglePositive(t *testing.T) {
	// A single negative CF at t=0 and a single positive CF at t=n means
	// the finance_rate never enters the calculation (there's only one
	// negative cash flow, undiscounted at t=0) - MIRR collapses to
	// exactly reinvest_rate regardless of finance_rate. 100 growing at
	// 10%/yr for 3 years is exactly 133.1.
	got, err := solveMIRR([]float64{-100, 0, 0, 133.1}, 0.05, 0.10)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got, 0.10, irrTol, "MIRR (single CF pair, exact by construction)")
}

func TestMIRR_ExactByConstruction_IgnoresFinanceRateWithSingleUpfrontOutflow(t *testing.T) {
	// Same shape, different finance_rate - confirms finance_rate truly
	// doesn't affect the result when there's only one negative CF at t=0
	// (PV of that CF at any discount rate is just itself).
	got, err := solveMIRR([]float64{-100, 0, 0, 133.1}, 0.20, 0.10)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got, 0.10, irrTol, "MIRR (finance_rate-insensitivity check)")
}

// TestMIRR_IndependentCrossCheck mirrors a Python re-implementation of the
// same formula (sum-of-discounted-negatives / sum-of-compounded-positives),
// run independently during development:
//
//	def mirr(cfs, finance_rate, reinvest_rate):
//	    n = len(cfs) - 1
//	    pv_neg = sum(cf/(1+finance_rate)**i for i,cf in enumerate(cfs) if cf < 0)
//	    fv_pos = sum(cf*(1+reinvest_rate)**(n-i) for i,cf in enumerate(cfs) if cf > 0)
//	    return (fv_pos / -pv_neg) ** (1/n) - 1
//	mirr([-1000, -200, 300, 800], 0.08, 0.12)  # -> -0.014029232283635618
func TestMIRR_IndependentCrossCheck(t *testing.T) {
	got, err := solveMIRR([]float64{-1000, -200, 300, 800}, 0.08, 0.12)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got, -0.014029232283635618, 1e-9, "MIRR (vs. independent Python cross-check)")
}

func TestMIRR_RequiresBothSigns(t *testing.T) {
	if _, err := solveMIRR([]float64{100, 200}, 0.05, 0.10); err == nil {
		t.Fatal("expected an error for all-positive cash flows")
	}
	if _, err := solveMIRR([]float64{-100, -200}, 0.05, 0.10); err == nil {
		t.Fatal("expected an error for all-negative cash flows")
	}
}

func TestMIRR_NativeFuncRegistration(t *testing.T) {
	spec, ok := LookupFunction("MIRR")
	if !ok {
		t.Fatal("MIRR not registered in Library")
	}
	result, err := spec.Native([]any{[]float64{-100, 0, 0, 133.1}, 0.05, 0.10})
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, result.(float64), 0.10, irrTol, "Library[\"MIRR\"]")
}

func TestMIRR_NotPushdownable(t *testing.T) {
	// MIRR is closed-form (unlike IRR/XIRR) but has no verified SQL
	// emitter yet - Pushdownable must report false, not silently succeed
	// with an untested expansion.
	spec, ok := LookupFunction("MIRR")
	if !ok {
		t.Fatal("MIRR not registered in Library")
	}
	if spec.Pushdownable(DialectStarRocks) {
		t.Fatal("MIRR should not be pushdownable to StarRocks yet - no verified SQL emitter")
	}
}

// TestIRR_ThroughFullExpressionAST (irr_test.go) already proves IRR runs
// through the full RuleNode/Expression path; this proves MIRR does too,
// closing the same loop for a 3-arg function.
func TestMIRR_ThroughFullExpressionAST(t *testing.T) {
	ae := NewAdvancedEvaluator()
	node := RuleNode{
		Type: NodeTypeExpression,
		Expression: &Expression{
			Root: &FuncCall{
				Name: "MIRR",
				Args: []ExprNode{
					&FieldRef{Path: "cash_flows"},
					&Literal{Value: 0.05},
					&Literal{Value: 0.10},
				},
			},
		},
	}
	result, err := ae.EvaluateNumeric(node, map[string]interface{}{
		"cash_flows": []interface{}{-100.0, 0.0, 0.0, 133.1},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, result, 0.10, irrTol, "MIRR through full Expression AST")
}
