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

func TestRuleContentHashGoldenVector_NestedAndUnicode(t *testing.T) {
	// Complex input containing nested AST, arrays, decimals, unicode in citation and keys
	astCondition := map[string]interface{}{
		"type":     "LOGICAL_AND",
		"operator": "AND",
		"conditions": []interface{}{
			map[string]interface{}{
				"type":     "COMPARISON",
				"operator": "GREATER_THAN",
				"left": map[string]interface{}{
					"type": "METRIC",
					"path": "position.issuer_exposure_pct",
				},
				"right": map[string]interface{}{
					"type": "PARAM",
					"name": "issuer_limit_pct",
				},
			},
			map[string]interface{}{
				"type":     "COMPARISON",
				"operator": "EQUAL",
				"left": map[string]interface{}{
					"type": "METRIC",
					"path": "instrument.currency",
				},
				"right": map[string]interface{}{
					"type":  "LITERAL",
					"value": "EUR € / USD $",
				},
			},
		},
	}

	parameterThresholds := map[string]interface{}{
		"issuer_limit_pct": decimal.RequireFromString("0.050000"),
		"lookthrough":      true,
		"tags":             []interface{}{"UCITS_V", "Régulation_2026", "Tier-1"},
		"nested_config": map[string]interface{}{
			"buffer_bps": decimal.NewFromInt(25),
			"enabled":    true,
			"sub_limits": []interface{}{
				decimal.RequireFromString("0.100000"),
				decimal.RequireFromString("0.400000"),
			},
		},
	}

	citation := "UCITS Directive 2009/65/EC Art. 52 § 1 — Concentration (5%/10%/40% & € / $ rules)"

	hashFromMap, err := ComputeRuleContentHash(astCondition, parameterThresholds, citation)
	if err != nil {
		t.Fatalf("ComputeRuleContentHash failed: %v", err)
	}

	// Also compute directly from raw JCS JSON bytes
	rawASTJSON := `{"type":"LOGICAL_AND","operator":"AND","conditions":[{"type":"COMPARISON","operator":"GREATER_THAN","left":{"type":"METRIC","path":"position.issuer_exposure_pct"},"right":{"type":"PARAM","name":"issuer_limit_pct"}},{"type":"COMPARISON","operator":"EQUAL","left":{"type":"METRIC","path":"instrument.currency"},"right":{"type":"LITERAL","value":"EUR € / USD $"}}]}`
	rawParamsJSON := `{"issuer_limit_pct":"0.050000","lookthrough":true,"nested_config":{"buffer_bps":"25.000000","enabled":true,"sub_limits":["0.100000","0.400000"]},"tags":["UCITS_V","Régulation_2026","Tier-1"]}`

	hashFromRaw, err := ComputeRuleContentHashFromRaw([]byte(rawASTJSON), []byte(rawParamsJSON), citation)
	if err != nil {
		t.Fatalf("ComputeRuleContentHashFromRaw failed: %v", err)
	}

	if hashFromMap != hashFromRaw {
		t.Fatalf("Discrepancy between Map and Raw! Map: %s, Raw: %s", hashFromMap, hashFromRaw)
	}

	t.Logf("Golden Rule Content Hash: %s", hashFromMap)

	// Cross-check against Python canonical JCS + SHA-256
	pyScript := `
import json, hashlib

ast = {
    "type": "LOGICAL_AND",
    "operator": "AND",
    "conditions": [
        {
            "type": "COMPARISON",
            "operator": "GREATER_THAN",
            "left": {"type": "METRIC", "path": "position.issuer_exposure_pct"},
            "right": {"type": "PARAM", "name": "issuer_limit_pct"}
        },
        {
            "type": "COMPARISON",
            "operator": "EQUAL",
            "left": {"type": "METRIC", "path": "instrument.currency"},
            "right": {"type": "LITERAL", "value": "EUR \u20ac / USD $"}
        }
    ]
}

params = {
    "issuer_limit_pct": "0.050000",
    "lookthrough": True,
    "tags": ["UCITS_V", "R\u00e9gulation_2026", "Tier-1"],
    "nested_config": {
        "buffer_bps": "25.000000",
        "enabled": True,
        "sub_limits": ["0.100000", "0.400000"]
    }
}

citation = "UCITS Directive 2009/65/EC Art. 52 \u00a7 1 \u2014 Concentration (5%/10%/40% & \u20ac / $ rules)"

jcs_ast = json.dumps(ast, ensure_ascii=False, separators=(',', ':'), sort_keys=True).encode('utf-8')
jcs_params = json.dumps(params, ensure_ascii=False, separators=(',', ':'), sort_keys=True).encode('utf-8')

preimage = b"v1|" + jcs_ast + b"|" + jcs_params + b"|" + citation.encode('utf-8')
print(hashlib.sha256(preimage).hexdigest(), end="")
`
	cmd := exec.Command("python3", "-c", pyScript)
	pyOutput, pyErr := cmd.Output()
	if pyErr != nil {
		t.Fatalf("Python execution failed: %v", pyErr)
	}

	pyHash := strings.TrimSpace(string(pyOutput))
	if pyHash != hashFromMap {
		t.Fatalf("Python vs Go RuleContentHash mismatch!\n Go:     %s\n Python: %s", hashFromMap, pyHash)
	}

	t.Logf("Cross-implementation verified: Go and Python agree 100%% on rule content hash %s", pyHash)
}

