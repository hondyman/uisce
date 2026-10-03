package vm

import "testing"

// C1/9.1: the VM now owns metric semantics. These tests pin the properties the
// querybuilder implementations used to guarantee, so the ownership move cannot
// quietly change meaning. The end-to-end proof that the *delegated* path is
// unchanged lives in internal/querybuilder's golden corpus.

func TestDeriveDecomposable(t *testing.T) {
	cases := []struct {
		name string
		expr MetricExpression
		want bool
	}{
		{"sum is decomposable", MetricExpression{Kind: "aggregation", Fn: "sum"}, true},
		{"count is decomposable", MetricExpression{Kind: "aggregation", Fn: "count"}, true},
		{"min is decomposable", MetricExpression{Kind: "aggregation", Fn: "min"}, true},
		{"max is decomposable", MetricExpression{Kind: "aggregation", Fn: "max"}, true},
		{"avg is NOT decomposable", MetricExpression{Kind: "aggregation", Fn: "avg"}, false},
		{"kind case/whitespace insensitive", MetricExpression{Kind: "  AGGREGATION ", Fn: " SUM "}, true},
		{"plain addition is decomposable", MetricExpression{Kind: "formula", Formula: "a + b"}, true},
		{"division is NOT decomposable", MetricExpression{Kind: "formula", Formula: "a / b"}, false},
		{"avg() in a formula is NOT decomposable", MetricExpression{Kind: "formula", Formula: "avg(x) + b"}, false},
		{"derived is NOT decomposable", MetricExpression{Kind: "derived", NumeratorID: "rev", DenominatorID: "cost"}, false},
		{"unknown kind is NOT decomposable", MetricExpression{Kind: "mystery"}, false},
		{"empty is NOT decomposable", MetricExpression{}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := DeriveDecomposable(tc.expr); got != tc.want {
				t.Fatalf("DeriveDecomposable(%+v) = %v, want %v", tc.expr, got, tc.want)
			}
		})
	}
}

func TestNormalizeFormulaForHash(t *testing.T) {
	// Expectations match the pre-port behaviour exactly. Parens are dropped
	// without inserting a separator, so "SUM(a)" normalizes to "suma" -- that is
	// the existing (slightly lossy) canonicalization, preserved verbatim so the
	// port does not silently change any content hash already deployed.
	cases := []struct{ in, want string }{
		{"SUM(a)", "suma"},
		{"  sum(  a )  ", "sum a"},
		{"a+b", "a+b"},
		{"a   +    b", "a + b"},
		{"SUM(price * qty) * @fx_rate", "sumprice * qty * @fx_rate"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := NormalizeFormulaForHash(tc.in); got != tc.want {
			t.Errorf("NormalizeFormulaForHash(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestComputeMetricContentHash_StabilityAndSensitivity(t *testing.T) {
	base := MetricContentInput{
		Name: "gross_margin",
		BOID: "bo-1",
		Expression: MetricExpression{
			Kind: "derived", NumeratorID: "revenue", DenominatorID: "cost",
		},
		GrainAllowlist: []string{"region", "order_date"},
	}

	// Formatting-only differences must not change the hash.
	reformatted := base
	reformatted.Expression.Kind = "  DERIVED  "
	reformatted.GrainAllowlist = []string{" Order_Date ", "REGION"}
	if ComputeMetricContentHash(base) != ComputeMetricContentHash(reformatted) {
		t.Error("hash changed under formatting-only differences; it must be canonical")
	}

	// Grain order must not matter (it is a set).
	reordered := base
	reordered.GrainAllowlist = []string{"order_date", "region"}
	if ComputeMetricContentHash(base) != ComputeMetricContentHash(reordered) {
		t.Error("grain order changed the hash; grains are a set and must sort")
	}

	// ADR-026: operand order IS semantic. Inverting a ratio must change the
	// hash, or the two opposite metrics share a cube identity and a cache entry.
	inverted := base
	inverted.Expression = MetricExpression{
		Kind: "derived", NumeratorID: "cost", DenominatorID: "revenue",
	}
	if ComputeMetricContentHash(base) == ComputeMetricContentHash(inverted) {
		t.Error("inverted ratio shares a content hash; numerator/denominator must be distinct")
	}

	// Named operands must not collide with the ordered form either.
	orderedForm := base
	orderedForm.Expression = MetricExpression{
		Kind: "derived", BaseMetricIDs: []string{"revenue", "cost"},
	}
	if ComputeMetricContentHash(base) == ComputeMetricContentHash(orderedForm) {
		t.Error("named-operand and ordered-operand forms collide; both are hashed distinctly")
	}
}

func TestComputeMetricContentHash_VariablesAreOrderInvariant(t *testing.T) {
	a := MetricContentInput{
		Name: "m", BOID: "bo",
		Variables: []MetricVariable{
			{Name: "fx_rate", Type: VariableTypeNumber},
			{Name: "as_of", Type: VariableTypeDate},
		},
	}
	b := a
	b.Variables = []MetricVariable{a.Variables[1], a.Variables[0]}

	if ComputeMetricContentHash(a) != ComputeMetricContentHash(b) {
		t.Error("variable order changed the hash; variables are a set and must sort by name")
	}

	// A changed variable must change the hash.
	c := a
	c.Variables = []MetricVariable{
		{Name: "fx_rate", Type: VariableTypeString}, // type changed
		{Name: "as_of", Type: VariableTypeDate},
	}
	if ComputeMetricContentHash(a) == ComputeMetricContentHash(c) {
		t.Error("variable type change did not affect the hash")
	}
}
