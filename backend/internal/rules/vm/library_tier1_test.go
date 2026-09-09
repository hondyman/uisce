package vm

import (
	"testing"
	"time"
)

// YEARFRAC: real fixture from Microsoft's published docs -
// https://support.microsoft.com/en-us/office/yearfrac-function-3844141e-c76d-4143-82b6-208454ddc6a8
// (1/1/2012 to 7/30/2012, three bases). Note: this repo's YEARFRAC only
// implements ACT/365, ACT/360, and 30/360 (Excel's basis 3, 2, 0) - not
// Excel's basis 1 (actual/actual, average-year-length dependent) or
// basis 4 (European 30/360) - so only the two Microsoft-published values
// for supported bases are used here as fixtures.
func TestYEARFRAC_ExcelFixture_MicrosoftDocsExample(t *testing.T) {
	cases := []struct {
		basis    string
		expected float64
	}{
		{"30/360", 0.58055556},
		{"ACT/365", 0.57808219},
	}
	for _, c := range cases {
		got, err := yearFrac(mustDate(t, "2012-01-01"), mustDate(t, "2012-07-30"), c.basis)
		if err != nil {
			t.Fatalf("basis %s: %v", c.basis, err)
		}
		assertNear(t, got, c.expected, 1e-6, "YEARFRAC basis "+c.basis+" (vs. Microsoft's published example)")
	}
}

func TestYEARFRAC_ExactByConstruction_ACT360(t *testing.T) {
	// Jan 1 to Jul 1 is exactly 181 actual days (2026 is not a leap year:
	// Jan 31 + Feb 28 + Mar 31 + Apr 30 + May 31 + Jun 30 = 181).
	got, err := yearFrac(mustDate(t, "2026-01-01"), mustDate(t, "2026-07-01"), "ACT/360")
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got, 181.0/360.0, 1e-12, "YEARFRAC ACT/360 (exact by construction)")
}

func TestYEARFRAC_ExactByConstruction_30360_ExactHalfYear(t *testing.T) {
	// 30/360's whole point: Jan 1 to Jul 1 is defined as exactly 6*30=180
	// days regardless of actual calendar length, i.e. exactly half a
	// 360-day year.
	got, err := yearFrac(mustDate(t, "2026-01-01"), mustDate(t, "2026-07-01"), "30/360")
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got, 0.5, 1e-12, "YEARFRAC 30/360 (exact by construction)")
}

func TestYEARFRAC_UnsupportedBasis(t *testing.T) {
	if _, err := yearFrac(mustDate(t, "2026-01-01"), mustDate(t, "2026-07-01"), "actual/actual"); err == nil {
		t.Fatal("expected an error for an unsupported basis")
	}
}

func TestYEARFRAC_NativeFuncRegistration(t *testing.T) {
	spec, ok := LookupFunction("YEARFRAC")
	if !ok {
		t.Fatal("YEARFRAC not registered in Library")
	}
	got, err := spec.Native([]any{"2026-01-01", "2026-07-01", "30/360"})
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got.(float64), 0.5, 1e-12, "Library[\"YEARFRAC\"]")
}

// XNPV: exact-by-construction, cross-checked with the same math as NPV's
// own exact-by-construction cases but over real dates one year apart -
// at exactly 365 days (ACT/365), XNPV should reduce to plain NPV's
// integer-period formula.
func TestXNPV_ExactByConstruction_MatchesNPVAtOneYearSpacing(t *testing.T) {
	spec, ok := LookupFunction("XNPV")
	if !ok {
		t.Fatal("XNPV not registered in Library")
	}
	// -1000 at t=0, +1100 exactly 365 days later, rate=0.10:
	// NPV = -1000 + 1100/1.10 = -1000 + 1000 = 0.
	got, err := spec.Native([]any{0.10,
		[]any{-1000.0, 1100.0},
		[]any{"2026-01-01", "2027-01-01"},
	})
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got.(float64), 0.0, 1e-6, "XNPV (exact by construction, matches NPV at 365-day spacing)")
}

func TestXNPV_MismatchedLengthsError(t *testing.T) {
	spec, _ := LookupFunction("XNPV")
	_, err := spec.Native([]any{0.10, []any{-1000.0, 1100.0}, []any{"2026-01-01"}})
	if err == nil {
		t.Fatal("expected an error for mismatched cash_flows/dates lengths")
	}
}

func TestXNPV_NotPushdownable(t *testing.T) {
	spec, _ := LookupFunction("XNPV")
	if spec.Pushdownable(DialectStarRocks) {
		t.Fatal("XNPV should not be pushdownable yet - date-string parsing isn't a single SQL expression here")
	}
}

// SUMPRODUCT: exact-by-construction (the textbook weighted-average
// numerator use case: avg_price = SUMPRODUCT(qty, price) / SUM(qty)).
func TestSUMPRODUCT_ExactByConstruction(t *testing.T) {
	spec, ok := LookupFunction("SUMPRODUCT")
	if !ok {
		t.Fatal("SUMPRODUCT not registered in Library")
	}
	// qty=[10,20], price=[2,3] -> 10*2 + 20*3 = 20+60 = 80
	got, err := spec.Native([]any{[]any{10.0, 20.0}, []any{2.0, 3.0}})
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got.(float64), 80.0, 1e-12, "SUMPRODUCT (exact by construction)")
}

func TestSUMPRODUCT_Pushdownable(t *testing.T) {
	spec, _ := LookupFunction("SUMPRODUCT")
	if !spec.Pushdownable(DialectStarRocks) {
		t.Fatal("SUMPRODUCT should be pushdownable - SUM(a*b) is a single SQL expression")
	}
	sql, err := spec.SQLEmit[DialectStarRocks]([]string{"qty", "price"})
	if err != nil {
		t.Fatal(err)
	}
	if sql != "SUM(qty * price)" {
		t.Errorf("got %q, want %q", sql, "SUM(qty * price)")
	}
}

func TestCompileToSQL_SUMPRODUCTPushdown(t *testing.T) {
	expr := &Expression{
		Root: &BinaryExpr{
			Op:    "/",
			Left:  &FuncCall{Name: "SUMPRODUCT", Args: []ExprNode{&FieldRef{Path: "qty"}, &FieldRef{Path: "price"}}},
			Right: &FuncCall{Name: "SUM", Args: []ExprNode{&FieldRef{Path: "qty"}}},
		},
	}
	got, err := CompileToSQL(expr, resolveIdentity)
	if err != nil {
		t.Fatal(err)
	}
	want := "(SUM(qty * price) / SUM(qty))"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// LN/EXP/SQRT: exact-by-construction and mutual inverse cross-checks.
func TestLN_EXP_SQRT_ExactByConstruction(t *testing.T) {
	lnSpec, _ := LookupFunction("LN")
	got, err := lnSpec.Native([]any{1.0})
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got.(float64), 0.0, 1e-12, "LN(1)")

	expSpec, _ := LookupFunction("EXP")
	got, err = expSpec.Native([]any{0.0})
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got.(float64), 1.0, 1e-12, "EXP(0)")

	sqrtSpec, _ := LookupFunction("SQRT")
	got, err = sqrtSpec.Native([]any{144.0})
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got.(float64), 12.0, 1e-12, "SQRT(144)")
}

func TestLN_EXP_AreInverses(t *testing.T) {
	// Cross-check: LN(EXP(x)) == x for several x, confirming the two
	// registrations are consistent with each other, not just individually
	// plausible.
	lnSpec, _ := LookupFunction("LN")
	expSpec, _ := LookupFunction("EXP")
	for _, x := range []float64{0.05, 1.0, 2.718281828, 10.0} {
		e, err := expSpec.Native([]any{x})
		if err != nil {
			t.Fatal(err)
		}
		back, err := lnSpec.Native([]any{e})
		if err != nil {
			t.Fatal(err)
		}
		assertNear(t, back.(float64), x, 1e-9, "LN(EXP(x)) == x")
	}
}

func TestLN_EXP_SQRT_Pushdownable(t *testing.T) {
	for _, name := range []string{"LN", "EXP", "SQRT"} {
		spec, ok := LookupFunction(name)
		if !ok {
			t.Fatalf("%s not registered", name)
		}
		if !spec.Pushdownable(DialectStarRocks) {
			t.Errorf("%s should be pushdownable - StarRocks has a native function of the same name", name)
		}
	}
}

func mustDate(t *testing.T, s string) time.Time {
	t.Helper()
	d, err := parseDate(s)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
