package boresolver_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	br "github.com/hondyman/uisce/backend/internal/boresolver"
)

// TestCalcInCalc_HostRuntimeReferencesHostRuntime proves "a calc in a
// calc" for host-runtime formulas: net_customer_xirr = customer_xirr -
// hurdle_rate, where customer_xirr is itself a host-runtime aggregate
// (xirr over a cash-flow series) and hurdle_rate is a plain per-entity
// base field. Tier propagation must classify net_customer_xirr as
// host-runtime too (it references customer_xirr, whose value never exists
// as a SQL column), and HostRuntimeExecutor must compose the two: compute
// customer_xirr first, then reuse that scalar for net_customer_xirr
// without re-fetching cash flows.
func TestCalcInCalc_HostRuntimeReferencesHostRuntime(t *testing.T) {
	graph := br.NewCalcGraph()
	graph.AddNode(&br.CalcNode{TermKey: "cashflow_amount", IsBaseField: true})
	graph.AddNode(&br.CalcNode{TermKey: "cashflow_date", IsBaseField: true})
	graph.AddNode(&br.CalcNode{TermKey: "hurdle_rate", IsBaseField: true})
	graph.AddNode(&br.CalcNode{
		TermKey:      "customer_xirr",
		Formula:      "xirr(${cashflow_amount}, ${cashflow_date})",
		Dependencies: []string{"cashflow_amount", "cashflow_date"},
	})
	graph.AddNode(&br.CalcNode{
		TermKey:      "net_customer_xirr",
		Formula:      "${customer_xirr} - ${hurdle_rate}",
		Dependencies: []string{"customer_xirr", "hurdle_rate"},
	})

	layers, err := graph.ResolveExecutionLayers()
	require.NoError(t, err)

	gen := &br.BOSQLGenerator{}
	sql, hostNodes, err := gen.CompileDeepCalculations(layers, "SELECT 1", []string{"net_customer_xirr"})
	require.NoError(t, err)

	// Neither host-runtime term should leak into the compiled SQL.
	assert.NotContains(t, sql, "customer_xirr")
	assert.NotContains(t, sql, "net_customer_xirr")

	require.Len(t, hostNodes, 2, "both customer_xirr and the poisoned net_customer_xirr must be cut from SQL")
	assert.Equal(t, "customer_xirr", hostNodes[0].TermKey, "dependency order: customer_xirr must be evaluated before net_customer_xirr")
	assert.Equal(t, "net_customer_xirr", hostNodes[1].TermKey)
	assert.Equal(t, br.TierHostRuntime, hostNodes[1].Tier, "net_customer_xirr must be poisoned to host-runtime even though it's pure arithmetic")

	source := &fakeRowSource{
		tenantRows: map[string]map[string][]br.CalcRow{
			"tenant-a": {
				"fund-1": {
					{"cashflow_amount": -10000.0, "cashflow_date": date("2008-01-01"), "hurdle_rate": 0.05},
					{"cashflow_amount": 2750.0, "cashflow_date": date("2008-03-01"), "hurdle_rate": 0.05},
					{"cashflow_amount": 4250.0, "cashflow_date": date("2008-10-30"), "hurdle_rate": 0.05},
					{"cashflow_amount": 3250.0, "cashflow_date": date("2009-02-15"), "hurdle_rate": 0.05},
					{"cashflow_amount": 2750.0, "cashflow_date": date("2009-04-01"), "hurdle_rate": 0.05},
				},
			},
		},
	}
	executor := &br.HostRuntimeExecutor{Rows: source}

	results, err := executor.Execute(context.Background(), "tenant-a", hostNodes)
	require.NoError(t, err)
	require.Len(t, results, 2)

	byTerm := map[string]br.HostRuntimeResult{}
	for _, r := range results {
		require.NoError(t, r.Err)
		byTerm[r.TermKey] = r
	}

	const xirr = 0.373362535
	assert.InDelta(t, xirr, byTerm["customer_xirr"].Value, 1e-6)
	assert.InDelta(t, xirr-0.05, byTerm["net_customer_xirr"].Value, 1e-6, "net_customer_xirr must equal customer_xirr - hurdle_rate")

	// Two fetches total: one for the cash-flow series (customer_xirr) and
	// one for hurdle_rate (a new base field net_customer_xirr needs) — NOT
	// three, because net_customer_xirr reused customer_xirr's
	// already-computed scalar instead of re-fetching and re-solving XIRR.
	assert.Equal(t, []string{"tenant-a", "tenant-a"}, source.calls)
}

// TestCalcInCalc_PushdownDependingOnPushdown_AlreadyWorks documents the
// baseline that already worked before this change: pure-SQL "calc in a
// calc" via the CTE layering (net_fund_yield depends on gross_return,
// itself a calc, not a base field) — see TestDeepCalculationEngine_Success
// in calc_compiler_test.go for the full assertion; this just re-confirms
// zero host-runtime nodes are produced for an all-pushdown graph.
func TestCalcInCalc_PushdownDependingOnPushdown_AlreadyWorks(t *testing.T) {
	graph := br.NewCalcGraph()
	graph.AddNode(&br.CalcNode{TermKey: "total_revenue", IsBaseField: true})
	graph.AddNode(&br.CalcNode{TermKey: "total_aum", IsBaseField: true})
	graph.AddNode(&br.CalcNode{
		TermKey:      "gross_return",
		Formula:      "${total_revenue} / ${total_aum}",
		Dependencies: []string{"total_revenue", "total_aum"},
	})
	graph.AddNode(&br.CalcNode{
		TermKey:      "gross_return_pct",
		Formula:      "${gross_return} * 100",
		Dependencies: []string{"gross_return"},
	})

	layers, err := graph.ResolveExecutionLayers()
	require.NoError(t, err)

	gen := &br.BOSQLGenerator{}
	_, hostNodes, err := gen.CompileDeepCalculations(layers, "SELECT 1", []string{"gross_return_pct"})
	require.NoError(t, err)
	assert.Empty(t, hostNodes)
}
