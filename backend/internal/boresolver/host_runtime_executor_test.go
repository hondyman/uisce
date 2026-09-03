package boresolver_test

import (
	"context"
	"testing"
	"time"

	"github.com/hondyman/uisce/backend/internal/boresolver"
	"github.com/stretchr/testify/assert"
)

// fakeRowSource is a tenant-scoped, in-memory RowSource used to prove the
// on-demand execution path end-to-end without a real database. A pipeline
// (precalc) RowSource would implement the same interface against a batch
// query instead.
type fakeRowSource struct {
	tenantRows map[string]map[string][]boresolver.CalcRow // tenantID -> entityID -> rows
	calls      []string                                   // tenantIDs FetchRows was called with, for assertion
}

func (f *fakeRowSource) FetchRows(ctx context.Context, tenantID string, termKeys []string) (map[string][]boresolver.CalcRow, error) {
	f.calls = append(f.calls, tenantID)
	return f.tenantRows[tenantID], nil
}

// TestHostRuntimeExecutor_EndToEnd builds a calc graph with one host-runtime
// XIRR field and one pushdown field, compiles it (cutting the XIRR node out
// of the SQL, per calc_compiler.go), then runs the cut node through
// HostRuntimeExecutor against fixture cash flows for two funds in the same
// tenant, verifying each fund's XIRR matches the Microsoft reference value
// independently and that RowSource only ever saw the requesting tenant.
func TestHostRuntimeExecutor_EndToEnd(t *testing.T) {
	graph := boresolver.NewCalcGraph()
	graph.AddNode(&boresolver.CalcNode{TermKey: "cashflow_amount", IsBaseField: true})
	graph.AddNode(&boresolver.CalcNode{TermKey: "cashflow_date", IsBaseField: true})
	graph.AddNode(&boresolver.CalcNode{
		TermKey:      "fund_xirr",
		Formula:      "xirr(${cashflow_amount}, ${cashflow_date})",
		Dependencies: []string{"cashflow_amount", "cashflow_date"},
	})

	layers, err := graph.ResolveExecutionLayers()
	assert.NoError(t, err)

	gen := &boresolver.BOSQLGenerator{}
	_, hostNodes, err := gen.CompileDeepCalculations(layers, "SELECT 1", []string{"fund_xirr"})
	assert.NoError(t, err)
	assert.Len(t, hostNodes, 1)

	rowsByEntity := map[string][]boresolver.CalcRow{
		"fund-alpha": {
			{"cashflow_amount": -10000.0, "cashflow_date": date("2008-01-01")},
			{"cashflow_amount": 2750.0, "cashflow_date": date("2008-03-01")},
			{"cashflow_amount": 4250.0, "cashflow_date": date("2008-10-30")},
			{"cashflow_amount": 3250.0, "cashflow_date": date("2009-02-15")},
			{"cashflow_amount": 2750.0, "cashflow_date": date("2009-04-01")},
		},
		"fund-beta": {
			// Same shape, different scale — proves per-entity grouping
			// (each fund gets its own independent XIRR).
			{"cashflow_amount": -20000.0, "cashflow_date": date("2008-01-01")},
			{"cashflow_amount": 5500.0, "cashflow_date": date("2008-03-01")},
			{"cashflow_amount": 8500.0, "cashflow_date": date("2008-10-30")},
			{"cashflow_amount": 6500.0, "cashflow_date": date("2009-02-15")},
			{"cashflow_amount": 5500.0, "cashflow_date": date("2009-04-01")},
		},
	}

	source := &fakeRowSource{
		tenantRows: map[string]map[string][]boresolver.CalcRow{
			"tenant-a": rowsByEntity,
			"tenant-b": {"fund-gamma": rowsByEntity["fund-alpha"]},
		},
	}
	executor := &boresolver.HostRuntimeExecutor{Rows: source}

	results, err := executor.Execute(context.Background(), "tenant-a", hostNodes)
	assert.NoError(t, err)
	assert.Len(t, results, 2)

	byEntity := map[string]boresolver.HostRuntimeResult{}
	for _, r := range results {
		assert.NoError(t, r.Err)
		assert.Equal(t, "fund_xirr", r.TermKey)
		byEntity[r.EntityID] = r
	}

	const expected = 0.373362535
	assert.InDelta(t, expected, byEntity["fund-alpha"].Value, 1e-6)
	assert.InDelta(t, expected, byEntity["fund-beta"].Value, 1e-6, "scaling every cash flow by 2x must not change XIRR")

	// Tenant isolation: only tenant-a's rows were requested, never tenant-b's.
	assert.Equal(t, []string{"tenant-a"}, source.calls)
}

func date(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}
