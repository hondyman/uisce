package querybuilder

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func revenueMetric() MetricDefinition {
	return MetricDefinition{
		ID:   "m_revenue",
		Name: "Revenue",
		BOID: "bo_sales",
		Expression: MetricExpression{
			Kind:       "aggregation",
			Fn:         "sum",
			TermNodeID: "revenue",
		},
		GrainAllowlist: []string{"country", "product", "order_date"},
		Decomposable:   true,
	}
}

func unitsMetric() MetricDefinition {
	return MetricDefinition{
		ID:   "m_units",
		Name: "Units Sold",
		BOID: "bo_sales",
		Expression: MetricExpression{
			Kind:       "aggregation",
			Fn:         "sum",
			TermNodeID: "units_sold",
		},
		GrainAllowlist: []string{"country", "product", "order_date"},
		Decomposable:   true,
	}
}

func costMetric() MetricDefinition {
	return MetricDefinition{
		ID:   "m_cost",
		Name: "Cost",
		BOID: "bo_sales",
		Expression: MetricExpression{
			Kind:       "aggregation",
			Fn:         "sum",
			TermNodeID: "cost",
		},
		GrainAllowlist: []string{"country", "product", "order_date"},
		Decomposable:   true,
	}
}

func ddlFixture() (CubeDefinition, map[string]MetricDefinition) {
	cube := sampleCube()
	cube.Grains = [][]string{{"country", "product", "order_date"}}
	cube.Dimensions = []CubeDimension{
		{TermNodeID: "country"},
		{TermNodeID: "product"},
	}
	lookup := map[string]MetricDefinition{
		"m_revenue": revenueMetric(),
		"m_units":   unitsMetric(),
	}
	return cube, lookup
}

// TestGenerateCubeMaterializationDDL_MatchesMetricDefinitions is acceptance
// test #7 and the gate for item 3: the generated DDL must reference each
// metric's OWN compiled expression, never a hardcoded default.
//
// This is the specific bug class GenerateMVDDL embodies. There, a measure
// defaults to SUM(notional) and an "oms." schema is assumed, so a cube built
// from it would compile, parse, and quietly aggregate the wrong column.
func TestGenerateCubeMaterializationDDL_MatchesMetricDefinitions(t *testing.T) {
	gen := NewCubeDDLGenerator("starrocks")
	cube, lookup := ddlFixture()

	got, err := gen.GenerateCubeMaterializationDDL(
		"tenant_a", false, cube,
		[]string{"country", "product", "order_date"},
		"oms.sales", lookup, nil)
	require.NoError(t, err)

	// Each metric's real measure expression must appear.
	assert.Contains(t, got.DDL, "SUM(",
		"measures must be compiled aggregates, not literals")
	assert.Contains(t, strings.ToLower(got.DDL), "revenue",
		"the revenue metric's own term must appear in the DDL")
	assert.Contains(t, strings.ToLower(got.DDL), "units_sold",
		"the units metric's own term must appear in the DDL")

	// One physical measure column per declared metric.
	assert.Equal(t, map[string]string{
		"m_revenue": "m_revenue",
		"m_units":   "m_units",
	}, got.MeasureColumns,
		"every declared metric must get its own physical column")

	// The source table is the resolved binding, not an assumed schema prefix.
	assert.Contains(t, got.DDL, "FROM oms.sales")

	// The hardcoded default must not leak in for a metric that is not notional.
	assert.NotContains(t, got.DDL, "SUM(notional)",
		"must not fall back to GenerateMVDDL's hardcoded SUM(notional) default")
}

// TestGenerateCubeMaterializationDDL_RejectsHardcodedDefault proves the
// generator refuses to emit a hardcoded default measure even if a compiled
// expression somehow produced one.
func TestGenerateCubeMaterializationDDL_RejectsHardcodedDefault(t *testing.T) {
	gen := NewCubeDDLGenerator("starrocks")
	cube, lookup := ddlFixture()

	// A metric whose expression compiles to SUM(notional) but which does not
	// target the notional term is exactly the silent-wrong-answer shape.
	lookup["m_revenue"] = MetricDefinition{
		ID:             "m_revenue",
		Name:           "Revenue",
		BOID:           "bo_sales",
		Expression:     MetricExpression{Kind: "formula", Formula: "SUM(notional)"},
		GrainAllowlist: []string{"country", "product", "order_date"},
	}

	_, err := gen.GenerateCubeMaterializationDDL(
		"tenant_a", false, cube,
		[]string{"country", "product", "order_date"},
		"oms.sales", lookup, nil)
	require.Error(t, err, "a hardcoded default measure must be rejected, not emitted")
	assert.Contains(t, err.Error(), "hardcoded default measure")
}

// TestGenerateCubeMaterializationDDL_AllowsGenuineNotional confirms the check
// does not fire on a metric that legitimately aggregates notional.
func TestGenerateCubeMaterializationDDL_AllowsGenuineNotional(t *testing.T) {
	gen := NewCubeDDLGenerator("starrocks")
	cube, lookup := ddlFixture()
	lookup["m_revenue"] = MetricDefinition{
		ID:             "m_revenue",
		Name:           "Notional",
		BOID:           "bo_sales",
		Expression:     MetricExpression{Kind: "aggregation", Fn: "sum", TermNodeID: "notional"},
		GrainAllowlist: []string{"country", "product", "order_date"},
	}

	got, err := gen.GenerateCubeMaterializationDDL(
		"tenant_a", false, cube,
		[]string{"country", "product", "order_date"},
		"oms.sales", lookup, nil)
	require.NoError(t, err, "a metric that genuinely targets notional is legitimate")
	assert.Contains(t, got.DDL, "notional")
}

// TestGenerateCubeMaterializationDDL_MissingMetricIsAnError verifies a missing
// metric is an error rather than a silently omitted column.
func TestGenerateCubeMaterializationDDL_MissingMetricIsAnError(t *testing.T) {
	gen := NewCubeDDLGenerator("starrocks")
	cube, lookup := ddlFixture()
	delete(lookup, "m_units")

	_, err := gen.GenerateCubeMaterializationDDL(
		"tenant_a", false, cube,
		[]string{"country", "product", "order_date"},
		"oms.sales", lookup, nil)
	require.ErrorIs(t, err, ErrCubeUnknownMetric)
}

// TestGenerateCubeMaterializationDDL_Deterministic verifies content-hash
// stability, which is what makes deploy idempotent.
func TestGenerateCubeMaterializationDDL_Deterministic(t *testing.T) {
	gen := NewCubeDDLGenerator("starrocks")
	cube, lookup := ddlFixture()

	a, err := gen.GenerateCubeMaterializationDDL("tenant_a", false, cube,
		[]string{"product", "country", "order_date"}, "oms.sales", lookup, nil)
	require.NoError(t, err)
	b, err := gen.GenerateCubeMaterializationDDL("tenant_a", false, cube,
		[]string{"country", "order_date", "product"}, "oms.sales", lookup, nil)
	require.NoError(t, err)

	assert.Equal(t, a.ContentHash, b.ContentHash,
		"grain member order must not change the DDL")
	assert.Equal(t, a.MaterializationName, b.MaterializationName)
	assert.Len(t, a.ContentHash, 64)
}

// TestCubeMaterializationName_GoldIsShared verifies the multi-tenant economics:
// a gold copy object is named identically regardless of tenant, while a
// tenant-specific object is namespaced.
func TestCubeMaterializationName_GoldIsShared(t *testing.T) {
	cube := sampleCube()
	cols := []string{"country", "order_date"}

	goldA := CubeMaterializationName("tenant_a", true, cube, cols)
	goldB := CubeMaterializationName("tenant_b", true, cube, cols)
	assert.Equal(t, goldA, goldB,
		"gold materializations are shared by every vanilla adopter")
	assert.Contains(t, goldA, "cube_gold_")

	own := CubeMaterializationName("tenant_a", false, cube, cols)
	assert.NotEqual(t, goldA, own)
	assert.Contains(t, own, "cube_tenant_a_")
}

// TestGenerateCubeMaterializationDDL_ValidatesInput covers the input guards.
func TestGenerateCubeMaterializationDDL_ValidatesInput(t *testing.T) {
	gen := NewCubeDDLGenerator("starrocks")
	cube, lookup := ddlFixture()
	grain := []string{"country", "product", "order_date"}

	t.Run("empty grain rejected", func(t *testing.T) {
		_, err := gen.GenerateCubeMaterializationDDL("t", false, cube, nil, "oms.sales", lookup, nil)
		require.ErrorIs(t, err, ErrCubeNoGrains)
	})

	t.Run("missing source table rejected", func(t *testing.T) {
		_, err := gen.GenerateCubeMaterializationDDL("t", false, cube, grain, "  ", lookup, nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "source table")
	})

	t.Run("structurally invalid cube rejected", func(t *testing.T) {
		bad := cube
		bad.MetricIDs = nil
		_, err := gen.GenerateCubeMaterializationDDL("t", false, bad, grain, "oms.sales", lookup, nil)
		require.ErrorIs(t, err, ErrCubeNoMetrics)
	})
}

// TestGenerateCubeMaterializationDDL_DerivedMetricKeepsOperandOrder closes the
// gap that let a derived-metric defect survive: every other test in this file
// used an aggregation or a formula, and the cube DDL generator is the ONLY
// production consumer of CompileMetric. A derived ratio therefore never went
// through the one path that turns a metric into StarRocks DDL.
//
// The generated measure must carry revenue over cost, because that is the
// order the metric declared. It used to emit cost over revenue, because the
// compiler sorted the operands and the numerator was whichever ID sorted
// first. See ADR-025.
func TestGenerateCubeMaterializationDDL_DerivedMetricKeepsOperandOrder(t *testing.T) {
	gen := NewCubeDDLGenerator("starrocks")
	cube, lookup := ddlFixture()
	lookup["m_cost"] = costMetric()

	grain := []string{"country", "product", "order_date"}
	schema := "oms.sales"

	t.Run("declared order is numerator over denominator", func(t *testing.T) {
		cube.MetricIDs = []string{"m_margin"}
		lookup["m_margin"] = MetricDefinition{
			ID:   "m_margin",
			Name: "Margin Ratio",
			BOID: "bo_sales",
			Expression: MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_revenue", "m_cost"},
			},
			GrainAllowlist: []string{"country", "product", "order_date"},
			Decomposable:   true,
		}

		ddl, err := gen.GenerateCubeMaterializationDDL("t", false, cube, grain, schema, lookup, nil)
		require.NoError(t, err)
		assert.Contains(t, ddl.DDL, "SUM(t0.revenue)) / NULLIF((SUM(t0.cost)",
			"a ratio declared revenue/cost must materialize as revenue over cost")
		assert.NotContains(t, ddl.DDL, "SUM(t0.cost)) / NULLIF((SUM(t0.revenue)")
	})

	t.Run("explicit operands override baseMetricIds order", func(t *testing.T) {
		lookup["m_margin"] = MetricDefinition{
			ID:   "m_margin",
			Name: "Margin Ratio",
			BOID: "bo_sales",
			Expression: MetricExpression{
				Kind:          "derived",
				BaseMetricIDs: []string{"m_cost", "m_revenue"},
				NumeratorID:   "m_revenue",
				DenominatorID: "m_cost",
			},
			GrainAllowlist: []string{"country", "product", "order_date"},
			Decomposable:   true,
		}

		ddl, err := gen.GenerateCubeMaterializationDDL("t", false, cube, grain, schema, lookup, nil)
		require.NoError(t, err)
		assert.Contains(t, ddl.DDL, "SUM(t0.revenue)) / NULLIF((SUM(t0.cost)")
	})

	t.Run("formula with an unspaced operator keeps both operands", func(t *testing.T) {
		cube.MetricIDs = []string{"m_fx"}
		lookup["m_fx"] = MetricDefinition{
			ID:   "m_fx",
			Name: "Revenue In USD",
			BOID: "bo_sales",
			Expression: MetricExpression{
				Kind:    "formula",
				Formula: "SUM(t0.revenue)*@fx_rate",
			},
			Variables: []MetricVariable{
				{Name: "fx_rate", Type: "number", DefaultValue: 1.0},
			},
			GrainAllowlist: []string{"country", "product", "order_date"},
			Decomposable:   true,
		}

		// The multiplication survives compilation - the point of the
		// tokenizer fix - but the measure is still parameterized, and a
		// materialized view cannot bind $1. So generation must refuse it with
		// a real reason rather than emitting DDL StarRocks cannot run.
		_, err := gen.GenerateCubeMaterializationDDL("t", false, cube, grain, schema, lookup, map[string]interface{}{"fx_rate": 1.2})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parameterized formula")
		assert.Contains(t, err.Error(), "self-contained")
	})

	t.Run("an unparameterized formula still materializes with its operator intact", func(t *testing.T) {
		cube.MetricIDs = []string{"m_plain"}
		lookup["m_plain"] = MetricDefinition{
			ID:   "m_plain",
			Name: "Revenue Scaled",
			BOID: "bo_sales",
			Expression: MetricExpression{
				Kind:    "formula",
				Formula: "SUM(t0.revenue)*2",
			},
			GrainAllowlist: []string{"country", "product", "order_date"},
			Decomposable:   true,
		}

		ddl, err := gen.GenerateCubeMaterializationDDL("t", false, cube, grain, schema, lookup, nil)
		require.NoError(t, err)
		assert.Contains(t, ddl.DDL, "SUM(t0.revenue)*2",
			"a literal-only formula must keep its operator and be materializable")
	})
}
