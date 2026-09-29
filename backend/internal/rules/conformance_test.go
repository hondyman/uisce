package rules

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

type ConformanceCase struct {
	ID          string                 `json:"id"`
	Description string                 `json:"description"`
	AST         json.RawMessage        `json:"ast"`
	Data        map[string]interface{} `json:"data"`
	Expected    bool                   `json:"expected"`
}

func TestConformanceSuite(t *testing.T) {
	data, err := os.ReadFile("testdata/conformance_cases.json")
	if err != nil {
		t.Fatalf("failed to read testdata/conformance_cases.json: %v", err)
	}

	var cases []ConformanceCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatalf("failed to unmarshal conformance cases: %v", err)
	}

	ae := vm.NewAdvancedEvaluator()

	for _, tc := range cases {
		t.Run(tc.ID, func(t *testing.T) {
			var node vm.RuleNode
			if err := json.Unmarshal(tc.AST, &node); err != nil {
				t.Fatalf("failed to unmarshal AST for case %s: %v", tc.ID, err)
			}

			result, err := ae.Evaluate(node, tc.Data)
			if err != nil {
				t.Fatalf("evaluation returned unexpected error for case %s: %v", tc.ID, err)
			}

			if result != tc.Expected {
				t.Errorf("case %q failed: expected %v, got %v (data=%+v)", tc.ID, tc.Expected, result, tc.Data)
			}
		})
	}
}
