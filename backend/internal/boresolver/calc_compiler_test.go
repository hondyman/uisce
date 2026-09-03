package boresolver_test

import (
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/stretchr/testify/assert"
)

func TestDeepCalculationEngine_Success(t *testing.T) {
	graph := boresolver.NewCalcGraph()

	// Base Fields (Layer 0)
	graph.AddNode(&boresolver.CalcNode{TermKey: "total_revenue", IsBaseField: true})
	graph.AddNode(&boresolver.CalcNode{TermKey: "total_aum", IsBaseField: true})
	graph.AddNode(&boresolver.CalcNode{TermKey: "management_fee", IsBaseField: true})

	// Level 1 Calculation
	graph.AddNode(&boresolver.CalcNode{
		TermKey:      "gross_return",
		Formula:      "(${total_revenue} / ${total_aum}) * 100",
		Dependencies: []string{"total_revenue", "total_aum"},
	})

	// Level 2 Calculation (Depends on Level 1 and Level 0)
	graph.AddNode(&boresolver.CalcNode{
		TermKey:      "net_fund_yield",
		Formula:      "${gross_return} - ${management_fee}",
		Dependencies: []string{"gross_return", "management_fee"},
	})

	layers, err := graph.ResolveExecutionLayers()
	assert.NoError(t, err)
	assert.Equal(t, 3, len(layers), "Should resolve into 3 execution layers (0, 1, 2)")

	gen := &boresolver.BOSQLGenerator{}
	baseQuery := "SELECT revenue AS total_revenue, aum AS total_aum, fee AS management_fee FROM public.funds WHERE tenant_id = '123'"

	sql, hostNodes, err := gen.CompileDeepCalculations(layers, baseQuery, []string{"net_fund_yield"})
	assert.NoError(t, err)
	assert.Empty(t, hostNodes, "purely arithmetic formulas should never produce host-runtime nodes")

	// Validate the nested structure
	assert.Contains(t, sql, "WITH layer_0 AS (")
	assert.Contains(t, sql, "layer_1 AS (")
	// Division now compiles through Dialect.SafeDiv rather than a raw "/",
	// so it can never emit a divide-by-zero error at query time.
	assert.Contains(t, sql, "CASE WHEN total_aum = 0 THEN NULL ELSE total_revenue / total_aum END")
	assert.Contains(t, sql, "AS gross_return")
	assert.Contains(t, sql, "layer_2 AS (")
	assert.Contains(t, sql, "gross_return - management_fee")
	assert.Contains(t, sql, "AS net_fund_yield")
	assert.Contains(t, sql, "FROM layer_2;")
}

// TestDeepCalculationEngine_HostRuntimeTier verifies that a formula calling
// a host-runtime-only function (xirr) is excluded from the compiled SQL and
// returned separately for the caller to execute via finlib, while pure
// pushdown fields in the same graph still compile to SQL as normal.
func TestDeepCalculationEngine_HostRuntimeTier(t *testing.T) {
	graph := boresolver.NewCalcGraph()

	graph.AddNode(&boresolver.CalcNode{TermKey: "cashflow_amount", IsBaseField: true})
	graph.AddNode(&boresolver.CalcNode{TermKey: "cashflow_date", IsBaseField: true})
	graph.AddNode(&boresolver.CalcNode{TermKey: "management_fee", IsBaseField: true})

	graph.AddNode(&boresolver.CalcNode{
		TermKey:      "fund_xirr",
		Formula:      "xirr(${cashflow_amount}, ${cashflow_date})",
		Dependencies: []string{"cashflow_amount", "cashflow_date"},
	})
	graph.AddNode(&boresolver.CalcNode{
		TermKey:      "fee_x2",
		Formula:      "${management_fee} * 2",
		Dependencies: []string{"management_fee"},
	})

	layers, err := graph.ResolveExecutionLayers()
	assert.NoError(t, err)

	gen := &boresolver.BOSQLGenerator{}
	baseQuery := "SELECT amount AS cashflow_amount, dt AS cashflow_date, fee AS management_fee FROM public.cashflows"

	sql, hostNodes, err := gen.CompileDeepCalculations(layers, baseQuery, []string{"fund_xirr", "fee_x2"})
	assert.NoError(t, err)

	// fund_xirr must NOT appear in the compiled SQL — it isn't SQL-expressible.
	assert.NotContains(t, sql, "fund_xirr")
	assert.NotContains(t, sql, "xirr(")
	// fee_x2 is pure pushdown and must still compile normally.
	assert.Contains(t, sql, "management_fee")
	assert.Contains(t, sql, "AS fee_x2")

	// fund_xirr comes back as a host-runtime node for the caller to execute.
	assert.Len(t, hostNodes, 1)
	assert.Equal(t, "fund_xirr", hostNodes[0].TermKey)
	assert.Equal(t, boresolver.TierHostRuntime, hostNodes[0].Tier)
}

func TestDeepCalculationEngine_CircularDependencyBlock(t *testing.T) {
	graph := boresolver.NewCalcGraph()

	// Malicious Circular Math
	graph.AddNode(&boresolver.CalcNode{
		TermKey:      "metric_a",
		Formula:      "${metric_b} * 2",
		Dependencies: []string{"metric_b"},
	})
	graph.AddNode(&boresolver.CalcNode{
		TermKey:      "metric_b",
		Formula:      "${metric_a} / 2",
		Dependencies: []string{"metric_a"},
	})

	_, err := graph.ResolveExecutionLayers()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "FATAL GOVERNANCE ERROR: Circular dependency")
}
