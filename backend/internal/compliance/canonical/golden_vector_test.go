package canonical

import (
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestEvaluationHashGoldenVector(t *testing.T) {
	// Fixed, immutable golden input
	lineageID := uuid.MustParse("018f2d5e-7a42-7000-8000-000000000001")
	tenantID := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	ruleID := uuid.MustParse("b0000000-0000-0000-0000-000000000001")

	input := EvaluationHashInput{
		LineageID:   lineageID,
		TenantID:    tenantID,
		RuleID:      ruleID,
		RuleVersion: 1,
		ActionTaken: "APPROVED",
		Passed:      true,
		InputParams: map[string]interface{}{
			"accountId":   "acc-9921",
			"securityId":  "sec-aapl",
			"orderQty":    decimal.NewFromInt(500),
			"orderPrice":  decimal.RequireFromString("182.50"),
			"orderSide":   "BUY",
			"orderSource": "INTERNAL_OMS",
		},
		MetricSnapshots: map[string]interface{}{
			"accountNav":        decimal.NewFromInt(10000000),
			"currentPositionVal": decimal.NewFromInt(400000),
			"proposedPositionVal": decimal.RequireFromString("491250.00"),
			"proposedWeight":    decimal.RequireFromString("0.049125"),
			"maxAllowedWeight":  decimal.RequireFromString("0.050000"),
		},
	}

	hash, err := ComputeEvaluationHash(input)
	if err != nil {
		t.Fatalf("ComputeEvaluationHash failed: %v", err)
	}

	// Verify against the deterministic reference hash
	t.Logf("Computed Golden EvaluationHash: %s", hash)

	// Ensure re-evaluation yields the exact same hash (100% deterministic)
	hash2, err := ComputeEvaluationHash(input)
	if err != nil {
		t.Fatalf("ComputeEvaluationHash second run failed: %v", err)
	}
	if hash != hash2 {
		t.Fatalf("Nondeterministic hash! Run 1: %s, Run 2: %s", hash, hash2)
	}

	// Verify against golden constant
	expectedGolden := "7e9fee6e5078f24064230c69c35b908469f0b4e6d3e74c83e00fbd84a03f65b1"
	if hash != expectedGolden {
		t.Errorf("Golden vector mismatch:\n got:      %s\n expected: %s", hash, expectedGolden)
	}
}
