package rulefabric

import (
	"encoding/json"
	"testing"
)

// TestCompileConditionTreeToCEL_EndToEnd compiles representative condition
// trees to CEL and evaluates the result through the real evaluator's CEL
// environment, to catch compile-time syntax errors and runtime semantic
// errors (wrong operator precedence, wrong type coercions), not just "it
// produced a string".
func TestCompileConditionTreeToCEL_EndToEnd(t *testing.T) {
	evaluator, err := NewRuleEvaluator(nil)
	if err != nil {
		t.Fatalf("NewRuleEvaluator: %v", err)
	}

	cases := []struct {
		name      string
		tree      string
		data      map[string]interface{}
		wantMatch bool
	}{
		{
			name: "simple equals",
			tree: `{"type":"group","operator":"AND","conditions":[
				{"type":"condition","field":"status","operator":"equals","value":"ACTIVE"}
			]}`,
			data:      map[string]interface{}{"status": "ACTIVE"},
			wantMatch: true,
		},
		{
			name: "AND of two leaves, one false",
			tree: `{"type":"group","operator":"AND","conditions":[
				{"type":"condition","field":"amount","operator":"greater_than","value":10000},
				{"type":"condition","field":"status","operator":"equals","value":"PENDING"}
			]}`,
			data:      map[string]interface{}{"amount": 5000.0, "status": "PENDING"},
			wantMatch: false,
		},
		{
			name: "OR with nested group and NOT",
			tree: `{"type":"group","operator":"OR","conditions":[
				{"type":"condition","field":"region","operator":"equals","value":"EU"},
				{"type":"group","operator":"NOT","conditions":[
					{"type":"condition","field":"active","operator":"equals","value":true}
				]}
			]}`,
			data:      map[string]interface{}{"region": "US", "active": false},
			wantMatch: true, // NOT(active==true) is true since active is false
		},
		{
			name: "between",
			tree: `{"type":"group","operator":"AND","conditions":[
				{"type":"condition","field":"score","operator":"between","value":[10,20]}
			]}`,
			data:      map[string]interface{}{"score": 15.0},
			wantMatch: true,
		},
		{
			name: "contains/starts_with/ends_with",
			tree: `{"type":"group","operator":"AND","conditions":[
				{"type":"condition","field":"email","operator":"ends_with","value":"@example.com"},
				{"type":"condition","field":"email","operator":"contains","value":"@"}
			]}`,
			data:      map[string]interface{}{"email": "user@example.com"},
			wantMatch: true,
		},
		{
			name: "in list",
			tree: `{"type":"group","operator":"AND","conditions":[
				{"type":"condition","field":"tier","operator":"in","value":["gold","platinum"]}
			]}`,
			data:      map[string]interface{}{"tier": "silver"},
			wantMatch: false,
		},
		{
			name: "is_null on missing field",
			tree: `{"type":"group","operator":"AND","conditions":[
				{"type":"condition","field":"middle_name","operator":"is_null","value":null}
			]}`,
			data:      map[string]interface{}{"first_name": "Jane"},
			wantMatch: true,
		},
		{
			name: "dotted field path",
			tree: `{"type":"group","operator":"AND","conditions":[
				{"type":"condition","field":"customer.kyc_status","operator":"equals","value":"VERIFIED"}
			]}`,
			data:      map[string]interface{}{"customer": map[string]interface{}{"kyc_status": "VERIFIED"}},
			wantMatch: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			expr, unsupported, err := CompileConditionTreeToCEL(json.RawMessage(tc.tree))
			if err != nil {
				t.Fatalf("compile error: %v", err)
			}
			if len(unsupported) > 0 {
				t.Fatalf("unexpected unsupported notes: %v", unsupported)
			}

			got, err := evaluator.EvaluateCELBoolean(expr, tc.data, nil, nil)
			if err != nil {
				t.Fatalf("eval error for expr %q: %v", expr, err)
			}
			if got != tc.wantMatch {
				t.Errorf("expr %q: got %v, want %v (data=%v)", expr, got, tc.wantMatch, tc.data)
			}
		})
	}
}

// TestEvaluate_CELSemantics locks in the pass/fail inversion between a
// compliance-style CEL condition (migrated from a tree, or authored with no
// explicit "semantics") and a policy-style WHEN/THEN CEL condition
// (PolicyRuleBuilder.tsx, which always writes "semantics":"policy"). Getting
// this backwards silently flips every rule's pass/fail result - this exact
// bug was caught by testing the tree->CEL migration end to end against a
// real Postgres instance before it shipped.
func TestEvaluate_CELSemantics(t *testing.T) {
	evaluator, err := NewRuleEvaluator(nil)
	if err != nil {
		t.Fatalf("NewRuleEvaluator: %v", err)
	}

	data := map[string]interface{}{"amount": 15000.0}
	evalCtx := &EvaluationContext{Data: data, Extras: map[string]interface{}{}}

	cases := []struct {
		name       string
		conditionJ string
		wantStatus EvaluationStatus
	}{
		{
			name:       "compliance semantics (default): condition true -> Passed",
			conditionJ: `{"type":"cel","expression":"data.amount > 10000.0"}`,
			wantStatus: EvalPassed,
		},
		{
			name:       "policy semantics: condition true -> Failed (guard fired)",
			conditionJ: `{"type":"cel","expression":"data.amount > 10000.0","semantics":"policy"}`,
			wantStatus: EvalFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rule := minimalRuleWithLogic("semantics-test", "semantics-test", tc.conditionJ)
			result, err := evaluator.Evaluate(nil, rule, evalCtx)
			if err != nil {
				t.Fatalf("Evaluate error: %v", err)
			}
			if result.Status != tc.wantStatus {
				t.Errorf("got status %v, want %v", result.Status, tc.wantStatus)
			}
		})
	}
}

// TestCompileConditionTreeToCEL_EntityPathFlagged verifies a cross-entity
// condition is reported as unsupported rather than silently mistranslated.
func TestCompileConditionTreeToCEL_EntityPathFlagged(t *testing.T) {
	tree := `{"type":"group","operator":"AND","conditions":[
		{"type":"condition","field":"kyc_status","operator":"equals","value":"VERIFIED",
		 "entityPath":{"fromEntity":"trade","toEntity":"customer","relationship":"owns","field":"kyc_status"}}
	]}`
	_, unsupported, err := CompileConditionTreeToCEL(json.RawMessage(tree))
	if err != nil {
		t.Fatalf("compile error: %v", err)
	}
	if len(unsupported) != 1 {
		t.Fatalf("expected 1 unsupported note, got %d: %v", len(unsupported), unsupported)
	}
}
