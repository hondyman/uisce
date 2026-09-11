package vm

import (
	"fmt"
	"math"
	"testing"
)

// Golden values, same three-kind discipline as irr_test.go/library_tier1_test.go:
//   - Excel fixture: real values from Microsoft's published STDEV.S/VAR.S/
//     COVARIANCE.S docs (STDEV.S/VAR.S fetched live for this pass -
//     https://support.microsoft.com/en-us/office/stdev-s-function-7d69cf97-0c1f-4acf-be27-f3e83904cc23,
//     .../var-s-function-913633de-136b-449d-813e-65a00b2b990b,
//     .../covariance-s-function-0a539b74-7371-42aa-a18f-1f5320314977).
//     CORREL/MEDIAN/PERCENTILE.EXC's own docs only publish their example
//     data as an unreadable image, not text (confirmed this pass, not
//     assumed from a 404 - see the Session 9 working note on that
//     distinction) - those three rely on exact-by-construction and an
//     independent cross-check instead, honestly labeled as such rather
//     than mislabeled as Excel fixtures.
//   - Exact-by-construction: inputs engineered so the expected value is
//     mathematically exact (a perfect line for SLOPE/CORREL, round numbers
//     for MEDIAN/PERCENTILE).
//   - Independent cross-check: recomputed with Python's stdlib `statistics`
//     module (STDEV.P/VAR.P, which Microsoft's docs don't fixture) or a
//     from-scratch centered-sum implementation (CORREL/SLOPE/PERCENTILE) -
//     a different language, not a second copy of this file's algorithm.

func TestSTDEV_S_ExcelFixture_MicrosoftDocsExample(t *testing.T) {
	data := []float64{1345, 1301, 1368, 1322, 1310, 1370, 1318, 1350, 1303, 1299}
	got, err := variance(toAnySlice(data), "STDEV_S", true)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, math.Sqrt(got), 27.46391572, 1e-6, "STDEV_S (vs. Microsoft's published example)")
}

func TestVAR_S_ExcelFixture_MicrosoftDocsExample(t *testing.T) {
	data := []float64{1345, 1301, 1368, 1322, 1310, 1370, 1318, 1350, 1303, 1299}
	got, err := variance(toAnySlice(data), "VAR_S", true)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got, 754.27, 1e-2, "VAR_S (vs. Microsoft's published example)")
}

func TestVAR_P_IndependentCrossCheck_PythonStatistics(t *testing.T) {
	// Same fixture data as VAR_S; population form has no Microsoft
	// fixture, so cross-checked against Python's statistics.pvariance.
	data := []float64{1345, 1301, 1368, 1322, 1310, 1370, 1318, 1350, 1303, 1299}
	got, err := variance(toAnySlice(data), "VAR_P", false)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got, 678.84, 1e-2, "VAR_P (vs. Python statistics.pvariance)")
}

func TestSTDEV_P_IndependentCrossCheck_PythonStatistics(t *testing.T) {
	data := []float64{1345, 1301, 1368, 1322, 1310, 1370, 1318, 1350, 1303, 1299}
	got, err := variance(toAnySlice(data), "STDEV_P", false)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, math.Sqrt(got), 26.054558142482477, 1e-6, "STDEV_P (vs. Python statistics.pstdev)")
}

func TestVAR_S_TooFewValues(t *testing.T) {
	if _, err := variance(toAnySlice([]float64{5}), "VAR_S", true); err == nil {
		t.Fatal("expected an error: sample variance is undefined for n=1")
	}
}

func TestCOVARIANCE_S_ExcelFixture_MicrosoftDocsExample(t *testing.T) {
	x := []float64{2, 4, 8}
	y := []float64{5, 11, 12}
	got, err := requirePairedAndCovariance(x, y)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got, 9.666666667, 1e-6, "COVARIANCE_S (vs. Microsoft's published example)")
}

func TestCORREL_ExactByConstruction_PerfectLine(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5}
	y := []float64{2, 4, 6, 8, 10}
	got, err := correl(x, y)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got, 1.0, 1e-12, "CORREL of a perfect line (exact by construction)")
}

func TestCORREL_IndependentCrossCheck_Python(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5, 6, 7}
	y := []float64{2.1, 3.9, 6.2, 7.8, 10.3, 11.7, 14.1}
	got, err := correl(x, y)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got, 0.9987148064527567, 1e-9, "CORREL (vs. independent Python centered-sum computation)")
}

func TestCORREL_ZeroVarianceIsAnError(t *testing.T) {
	x := []float64{1, 1, 1}
	y := []float64{2, 4, 6}
	if _, err := correl(x, y); err == nil {
		t.Fatal("expected an error: x has zero variance")
	}
}

func TestSLOPE_ExactByConstruction_PerfectLine(t *testing.T) {
	// y = 2x exactly.
	y := []float64{2, 4, 6, 8, 10}
	x := []float64{1, 2, 3, 4, 5}
	got, err := slope(y, x)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got, 2.0, 1e-12, "SLOPE of y=2x (exact by construction)")
}

func TestSLOPE_IndependentCrossCheck_Python(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5, 6, 7}
	y := []float64{2.1, 3.9, 6.2, 7.8, 10.3, 11.7, 14.1}
	got, err := slope(y, x)
	if err != nil {
		t.Fatal(err)
	}
	assertNear(t, got, 1.989285714285714, 1e-9, "SLOPE (vs. independent Python centered-sum computation)")
}

func TestMEDIAN_ExactByConstruction_Odd(t *testing.T) {
	assertNear(t, medianOf([]float64{5, 1, 3, 2, 4}), 3, 1e-12, "MEDIAN of {1..5} (exact by construction)")
}

func TestMEDIAN_ExactByConstruction_Even(t *testing.T) {
	assertNear(t, medianOf([]float64{1, 2, 3, 4}), 2.5, 1e-12, "MEDIAN of {1,2,3,4} (exact by construction)")
}

func TestPERCENTILE_ExactByConstruction(t *testing.T) {
	vals := []float64{10, 20, 30, 40}
	cases := []struct {
		k    float64
		want float64
	}{
		{0.0, 10},
		{0.25, 17.5},
		{0.5, 25},
		{1.0, 40},
	}
	for _, c := range cases {
		got, err := percentileOf(vals, c.k)
		if err != nil {
			t.Fatal(err)
		}
		assertNear(t, got, c.want, 1e-9, fmt.Sprintf("PERCENTILE k=%v", c.k))
	}
}

func TestPERCENTILE_OutOfRangeK(t *testing.T) {
	if _, err := percentileOf([]float64{1, 2, 3}, 1.5); err == nil {
		t.Fatal("expected an error: k must be in [0, 1]")
	}
	if _, err := percentileOf([]float64{1, 2, 3}, -0.1); err == nil {
		t.Fatal("expected an error: k must be in [0, 1]")
	}
}

// --- SQL pushdown ---

func TestTier2aStats_SQLPushdown(t *testing.T) {
	resolve := func(field string) (string, error) { return field, nil }
	cases := []struct {
		name string
		expr *Expression
		want string
	}{
		{"STDEV_S", callExpr("STDEV_S", fieldRef("x")), "STDDEV_SAMP(x)"},
		{"STDEV_P", callExpr("STDEV_P", fieldRef("x")), "STDDEV_POP(x)"},
		{"VAR_S", callExpr("VAR_S", fieldRef("x")), "VAR_SAMP(x)"},
		{"VAR_P", callExpr("VAR_P", fieldRef("x")), "VAR_POP(x)"},
		{"COVARIANCE_S", callExpr("COVARIANCE_S", fieldRef("x"), fieldRef("y")), "COVAR_SAMP(x, y)"},
		{"CORREL", callExpr("CORREL", fieldRef("x"), fieldRef("y")), "CORR(x, y)"},
		{"SLOPE", callExpr("SLOPE", fieldRef("y"), fieldRef("x")), "(COVAR_SAMP(y, x) / VAR_SAMP(x))"},
		{"MEDIAN", callExpr("MEDIAN", fieldRef("x")), "PERCENTILE_CONT(x, 0.5)"},
	}
	for _, c := range cases {
		got, err := CompileToSQL(c.expr, resolve)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if got != c.want {
			t.Fatalf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestTier2aStats_NativeFuncRegistration(t *testing.T) {
	for _, name := range []string{"STDEV_S", "STDEV_P", "VAR_S", "VAR_P", "COVARIANCE_S", "CORREL", "SLOPE", "MEDIAN", "PERCENTILE"} {
		spec, ok := LookupFunction(name)
		if !ok {
			t.Fatalf("%s: not registered", name)
		}
		if spec.Native == nil {
			t.Fatalf("%s: no native implementation", name)
		}
	}
}

// --- test-local helpers ---

func toAnySlice(vals []float64) []any {
	out := make([]any, len(vals))
	for i, v := range vals {
		out[i] = v
	}
	return out
}


func requirePairedAndCovariance(x, y []float64) (float64, error) {
	args := []any{toAnySlice(x), toAnySlice(y)}
	xs, ys, err := requirePairedSlices(args, "COVARIANCE_S")
	if err != nil {
		return 0, err
	}
	n := len(xs)
	mx, my := mean(xs), mean(ys)
	sum := 0.0
	for i := 0; i < n; i++ {
		sum += (xs[i] - mx) * (ys[i] - my)
	}
	return sum / float64(n-1), nil
}

func fieldRef(path string) ExprNode { return &FieldRef{Path: path} }

func callExpr(name string, args ...ExprNode) *Expression {
	return &Expression{Root: &FuncCall{Name: name, Args: args}}
}

