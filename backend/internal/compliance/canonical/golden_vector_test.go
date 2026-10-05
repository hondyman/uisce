package canonical

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestEvaluationHashGoldenVector(t *testing.T) {
	// Fixed, immutable golden input
	lineageID := uuid.MustParse("018f2d5e-7a42-7000-8000-000000000001")
	tenantID := uuid.MustParse("a0000000-0000-0000-0000-000000000001")
	ruleID := uuid.MustParse("b0000000-0000-0000-0000-000000000001")
	ruleContentHash := "c000000000000000000000000000000000000000000000000000000000000001"

	input := EvaluationHashInput{
		LineageID:       lineageID,
		TenantID:        tenantID,
		RuleID:          ruleID,
		RuleVersion:     1,
		RuleContentHash: ruleContentHash,
		ActionTaken:     "APPROVED",
		Passed:          true,
		InputParams: map[string]interface{}{
			"accountId":   "acc-9921",
			"securityId":  "sec-aapl",
			"orderQty":    decimal.NewFromInt(500),
			"orderPrice":  decimal.RequireFromString("182.50"),
			"orderSide":   "BUY",
			"orderSource": "INTERNAL_OMS",
		},
		MetricSnapshots: map[string]interface{}{
			"accountNav":         decimal.NewFromInt(10000000),
			"currentPositionVal": decimal.NewFromInt(400000),
			"proposedPositionVal": decimal.RequireFromString("491250.00"),
			"proposedWeight":     decimal.RequireFromString("0.049125"),
			"maxAllowedWeight":   decimal.RequireFromString("0.050000"),
		},
	}

	hash, err := ComputeEvaluationHash(input)
	if err != nil {
		t.Fatalf("ComputeEvaluationHash failed: %v", err)
	}

	t.Logf("Computed Golden EvaluationHash v2: %s", hash)

	// Ensure re-evaluation yields the exact same hash (100% deterministic)
	hash2, err := ComputeEvaluationHash(input)
	if err != nil {
		t.Fatalf("ComputeEvaluationHash second run failed: %v", err)
	}
	if hash != hash2 {
		t.Fatalf("Nondeterministic hash! Run 1: %s, Run 2: %s", hash, hash2)
	}

	// Verify against golden constant v2 (pre-computed and cross-checked)
	expectedGolden := "4629fea4eb92e26de3f49931f0e264a094cd819de3298ed9ea7ad144d4621328"
	if hash != expectedGolden {
		t.Errorf("Golden vector mismatch:\n got:      %s\n expected: %s", hash, expectedGolden)
	}

	// Independent cross-implementation verification with Python
	pyScript := `
import json, hashlib

lineage_id = "018f2d5e-7a42-7000-8000-000000000001"
tenant_id = "a0000000-0000-0000-0000-000000000001"
rule_id = "b0000000-0000-0000-0000-000000000001"
rule_version = 1
rule_content_hash = "c000000000000000000000000000000000000000000000000000000000000001"
action_taken = "APPROVED"

input_params = {
    "accountId": "acc-9921",
    "orderPrice": "182.500000",
    "orderQty": "500.000000",
    "orderSide": "BUY",
    "orderSource": "INTERNAL_OMS",
    "securityId": "sec-aapl"
}

metric_snapshots = {
    "accountNav": "10000000.000000",
    "currentPositionVal": "400000.000000",
    "maxAllowedWeight": "0.050000",
    "proposedPositionVal": "491250.000000",
    "proposedWeight": "0.049125"
}

jcs_input = json.dumps(input_params, separators=(',', ':'), sort_keys=True).encode('utf-8')
jcs_metrics = json.dumps(metric_snapshots, separators=(',', ':'), sort_keys=True).encode('utf-8')

preimage = f"v2|{lineage_id}|{tenant_id}|{rule_id}|{rule_version}|{rule_content_hash}|{action_taken}|true|".encode('utf-8') + jcs_input + b"|" + jcs_metrics
print(hashlib.sha256(preimage).hexdigest(), end="")
`
	cmd := exec.Command("python3", "-c", pyScript)
	pyOutput, pyErr := cmd.Output()
	if pyErr == nil {
		pyHash := strings.TrimSpace(string(pyOutput))
		if pyHash != hash {
			t.Fatalf("Cross-implementation discrepancy! Go: %s, Python: %s", hash, pyHash)
		}
		t.Logf("Cross-implementation verified: Go and Python agree 100%% on hash %s", pyHash)
	}
}
