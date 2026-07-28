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

	sql, err := gen.CompileDeepCalculations(layers, baseQuery, []string{"net_fund_yield"})
	assert.NoError(t, err)

	// Validate the nested structure
	assert.Contains(t, sql, "WITH layer_0 AS (")
	assert.Contains(t, sql, "layer_1 AS (")
	assert.Contains(t, sql, "(total_revenue / total_aum) * 100) AS gross_return")
	assert.Contains(t, sql, "layer_2 AS (")
	assert.Contains(t, sql, "(gross_return - management_fee) AS net_fund_yield")
	assert.Contains(t, sql, "FROM layer_2;")
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
