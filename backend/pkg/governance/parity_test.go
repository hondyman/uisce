package governance

import (
	"context"
	"reflect"
	"sort"
	"testing"
)

// Parity table for the governance policies, pinned when they moved from
// OPA/Rego to internal/rules/vm.
//
// Oracle: the semantic policy's expectations were captured from the running
// OPA engine. The trade and pipeline policies never evaluated under OPA on
// main (pipeline_validation.rego failed to parse - `deny contains` without
// importing `contains`; the trade query also asked for the undefined
// data.tenant.rules, so every call returned "no results"), so their
// expectations are what the Rego source states.

type parityCase struct {
	name    string
	input   map[string]interface{}
	allowed bool
	reasons []string
}

var tradeCases = []parityCase{
	{"empty payload", map[string]interface{}{}, true, nil},
	{"safe trade", map[string]interface{}{"symbol": "AAPL", "amount": 500000, "exposure": 100000}, true, nil},
	{"high value unapproved", map[string]interface{}{"amount": 1000000}, false, []string{"High-value trade (>1M) requires pre-approval"}},
	{"high value approved false", map[string]interface{}{"amount": 1500000, "compliance_approved": false, "symbol": "GOOGL"}, false, []string{"High-value trade (>1M) requires pre-approval"}},
	{"high value approved", map[string]interface{}{"amount": 2000000, "compliance_approved": true}, true, nil},
	{"just under threshold", map[string]interface{}{"amount": 999999.99}, true, nil},
	{"restricted symbol", map[string]interface{}{"symbol": "BAD", "amount": 100}, false, []string{"Symbol BAD is on the restricted list"}},
	{"exposure over limit", map[string]interface{}{"exposure": 6000000}, false, []string{"Portfolio exposure limit exceeded (5M)"}},
	{"exposure at limit", map[string]interface{}{"exposure": 5000000}, true, nil},
	{"everything wrong", map[string]interface{}{"amount": 5000000, "symbol": "RSTR", "exposure": 9000000}, false, []string{
		"High-value trade (>1M) requires pre-approval",
		"Portfolio exposure limit exceeded (5M)",
		"Symbol RSTR is on the restricted list",
	}},
}

var semanticCases = []parityCase{
	{"valid term", map[string]interface{}{"node_name": "Total Revenue", "description": "Sum of revenue"}, true, nil},
	{"short description", map[string]interface{}{"node_name": "x", "description": "abc"}, false, []string{"Description must be at least 5 characters long if provided"}},
	{"empty description allowed", map[string]interface{}{"node_name": "x", "description": ""}, true, nil},
	{"invalid name chars", map[string]interface{}{"node_name": "Bad;Name"}, false, []string{"Node name contains invalid characters"}},
	{"missing name", map[string]interface{}{}, true, nil}, // OPA: regex over an undefined value is undefined, so no deny
	{"name punctuation allowed", map[string]interface{}{"node_name": "t(1).v-2_x y"}, true, nil},
	{"complexity too high", map[string]interface{}{"node_name": "t", "complexity_score": 7}, false, []string{"Complexity score 7 exceeds limit of 5"}},
	{"complexity at limit", map[string]interface{}{"node_name": "t", "complexity_score": 5}, true, nil},
	{"restricted column", map[string]interface{}{"node_name": "t", "referenced_columns": []interface{}{"salary", "amount"}}, false, []string{"Access to restricted column 'salary' denied"}},
	{"restricted column with PII access", map[string]interface{}{"node_name": "t", "referenced_columns": []interface{}{"salary"}, "user_has_pii_access": true}, true, nil},
	{"two restricted columns", map[string]interface{}{"node_name": "t", "referenced_columns": []interface{}{"ssn", "salary"}}, false, []string{
		"Access to restricted column 'salary' denied",
		"Access to restricted column 'ssn' denied",
	}},
}

var pipelineCases = []parityCase{
	{"no nodes key", map[string]interface{}{}, true, nil},
	{"empty nodes", map[string]interface{}{"nodes": []interface{}{}}, false, []string{
		"Pipeline must have a Start node",
		"Pipeline must have at least one node",
	}},
	{"start node", map[string]interface{}{"nodes": []interface{}{map[string]interface{}{"id": "1", "type": "start"}}}, true, nil},
	{"no start node", map[string]interface{}{"nodes": []interface{}{map[string]interface{}{"id": "1", "type": "task"}}}, false, []string{"Pipeline must have a Start node"}},
	{"start among others", map[string]interface{}{"nodes": []interface{}{
		map[string]interface{}{"type": "task"}, map[string]interface{}{"type": "start"},
	}, "edges": []interface{}{}}, true, nil},
}

func sorted(xs []string) []string {
	out := append([]string(nil), xs...)
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func runParity(t *testing.T, cases []parityCase, eval func(map[string]interface{}) (*ValidationResult, error)) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			res, err := eval(c.input)
			if err != nil {
				t.Fatalf("error: %v", err)
			}
			if res.Allowed != c.allowed || !reflect.DeepEqual(sorted(res.Reasons), sorted(c.reasons)) {
				t.Errorf("got allowed=%v reasons=%q; want allowed=%v reasons=%q", res.Allowed, sorted(res.Reasons), c.allowed, sorted(c.reasons))
			}
		})
	}
}

func TestParity_TradeCompliance(t *testing.T) {
	e := NewGovernanceEngine(nil)
	runParity(t, tradeCases, func(in map[string]interface{}) (*ValidationResult, error) {
		return e.ValidateTransaction(context.Background(), "", in)
	})
}

func TestParity_SemanticValidation(t *testing.T) {
	e := NewGovernanceEngine(nil)
	runParity(t, semanticCases, func(in map[string]interface{}) (*ValidationResult, error) {
		return e.ValidateSemanticTerm(context.Background(), in)
	})
}

func TestParity_PipelineValidation(t *testing.T) {
	e := NewGovernanceEngine(nil)
	runParity(t, pipelineCases, func(in map[string]interface{}) (*ValidationResult, error) {
		return e.ValidatePipeline(context.Background(), "", in)
	})
}
