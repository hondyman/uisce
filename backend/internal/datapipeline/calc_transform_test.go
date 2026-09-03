package datapipeline

import (
	"context"
	"testing"

	"github.com/hondyman/uisce/backend/internal/boresolver"
)

// TestHostRuntimeCalcTransformer_MaterializesXIRR proves the pipeline path
// this transformer exists for: source rows (irregular cash flows already
// fetched by an earlier pipeline step, one record per cash flow) go in,
// and ONE materialized record per entity carrying its computed XIRR comes
// out — ready for a loader step to write into an ordinary table, which is
// what lets a LATER pushdown query reference the result as a normal
// column instead of hitting the "can't reference host-runtime output"
// wall.
func TestHostRuntimeCalcTransformer_MaterializesXIRR(t *testing.T) {
	transformer := &HostRuntimeCalcTransformer{
		Nodes: []*boresolver.CalcNode{
			{
				TermKey:      "fund_xirr",
				Formula:      "xirr(${cashflow_amount}, ${cashflow_date})",
				Dependencies: []string{"cashflow_amount", "cashflow_date"},
			},
		},
		EntityField: "fund_id",
		TenantID:    "tenant-a",
	}

	input := []PipelineRecord{
		{"fund_id": "fund-1", "cashflow_amount": -10000.0, "cashflow_date": "2008-01-01"},
		{"fund_id": "fund-1", "cashflow_amount": 2750.0, "cashflow_date": "2008-03-01"},
		{"fund_id": "fund-1", "cashflow_amount": 4250.0, "cashflow_date": "2008-10-30"},
		{"fund_id": "fund-1", "cashflow_amount": 3250.0, "cashflow_date": "2009-02-15"},
		{"fund_id": "fund-1", "cashflow_amount": 2750.0, "cashflow_date": "2009-04-01"},
	}

	output, errs, err := transformer.Transform(context.Background(), input)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got: %v", errs)
	}
	if len(output) != 1 {
		t.Fatalf("expected 1 materialized record (one per entity), got %d", len(output))
	}

	rec := output[0]
	if rec["fund_id"] != "fund-1" {
		t.Errorf("expected fund_id=fund-1, got %v", rec["fund_id"])
	}
	value, ok := rec["fund_xirr"].(float64)
	if !ok {
		t.Fatalf("expected fund_xirr to be a float64, got %T (%v)", rec["fund_xirr"], rec["fund_xirr"])
	}
	const expected = 0.373362535
	if diff := value - expected; diff > 1e-6 || diff < -1e-6 {
		t.Errorf("fund_xirr = %.9f, want %.9f", value, expected)
	}
}

// TestHostRuntimeCalcTransformer_ComposesCalcInCalc proves the pipeline
// tile composes multiple nodes the same way HostRuntimeExecutor does
// standalone: net_fund_xirr references fund_xirr's already-computed
// result plus a plain per-entity field (hurdle_rate), without re-running
// the XIRR solve.
func TestHostRuntimeCalcTransformer_ComposesCalcInCalc(t *testing.T) {
	transformer := &HostRuntimeCalcTransformer{
		Nodes: []*boresolver.CalcNode{
			{
				TermKey:      "fund_xirr",
				Formula:      "xirr(${cashflow_amount}, ${cashflow_date})",
				Dependencies: []string{"cashflow_amount", "cashflow_date"},
			},
			{
				TermKey:      "net_fund_xirr",
				Formula:      "${fund_xirr} - ${hurdle_rate}",
				Dependencies: []string{"fund_xirr", "hurdle_rate"},
			},
		},
		EntityField: "fund_id",
		TenantID:    "tenant-a",
	}

	input := []PipelineRecord{
		{"fund_id": "fund-1", "cashflow_amount": -10000.0, "cashflow_date": "2008-01-01", "hurdle_rate": 0.05},
		{"fund_id": "fund-1", "cashflow_amount": 2750.0, "cashflow_date": "2008-03-01", "hurdle_rate": 0.05},
		{"fund_id": "fund-1", "cashflow_amount": 4250.0, "cashflow_date": "2008-10-30", "hurdle_rate": 0.05},
		{"fund_id": "fund-1", "cashflow_amount": 3250.0, "cashflow_date": "2009-02-15", "hurdle_rate": 0.05},
		{"fund_id": "fund-1", "cashflow_amount": 2750.0, "cashflow_date": "2009-04-01", "hurdle_rate": 0.05},
	}

	output, errs, err := transformer.Transform(context.Background(), input)
	if err != nil {
		t.Fatalf("Transform failed: %v", err)
	}
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got: %v", errs)
	}
	if len(output) != 1 {
		t.Fatalf("expected 1 materialized record, got %d", len(output))
	}

	rec := output[0]
	xirr, _ := rec["fund_xirr"].(float64)
	net, _ := rec["net_fund_xirr"].(float64)
	const expectedNet = 0.373362535 - 0.05
	if diff := net - expectedNet; diff > 1e-6 || diff < -1e-6 {
		t.Errorf("net_fund_xirr = %.9f, want %.9f (fund_xirr=%.9f)", net, expectedNet, xirr)
	}
}

func TestHostRuntimeCalcTransformer_RequiresEntityFieldAndTenant(t *testing.T) {
	ctx := context.Background()

	if _, _, err := (&HostRuntimeCalcTransformer{TenantID: "t"}).Transform(ctx, nil); err == nil {
		t.Error("expected error when EntityField is empty")
	}
	if _, _, err := (&HostRuntimeCalcTransformer{EntityField: "id"}).Transform(ctx, nil); err == nil {
		t.Error("expected error when TenantID is empty")
	}
}
