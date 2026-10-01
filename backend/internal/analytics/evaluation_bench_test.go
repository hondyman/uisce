package analytics

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/hondyman/uisce/backend/internal/rules/vm"
)

func benchSnapshot(b *testing.B, n int) *RuleSnapshot {
	b.Helper()
	rules := make([]EvaluableRule, n)
	for i := 0; i < n; i++ {
		ast := vm.RuleNode{
			Type: vm.NodeTypeGroup,
			Group: &vm.RuleGroup{
				ID:       "g",
				Operator: "AND",
				Conditions: []vm.RuleNode{
					{
						Type: vm.NodeTypeCondition,
						Condition: &vm.RuleCondition{
							ID:       "a",
							Field:    "TargetQuantity",
							Operator: "greater_than",
							Value:    100.0,
						},
					},
					{
						Type: vm.NodeTypeCondition,
						Condition: &vm.RuleCondition{
							ID:       "b",
							Field:    "LimitPrice",
							Operator: "less_than",
							Value:    200.0,
						},
					},
				},
			},
		}
		raw, _ := json.Marshal(ast)
		var parsed vm.RuleNode
		_ = json.Unmarshal(raw, &parsed)
		rules[i] = EvaluableRule{
			RuleID:      "r" + strconv.Itoa(i),
			RuleKey:     "rule_" + strconv.Itoa(i),
			RuleName:    "Rule " + strconv.Itoa(i),
			BOName:      "order",
			Severity:    "BLOCK",
			RuleVersion: "1",
			AST:         &parsed,
			FieldRefs:   vm.FieldRefs(parsed),
		}
	}
	return &RuleSnapshot{
		TenantID:   "t",
		BOName:     "order",
		SnapshotID: "bench",
		Rules:      rules,
	}
}

func BenchmarkEvaluateRecord(b *testing.B) {
	for _, numRules := range []int{1, 5, 20} {
		b.Run(strconv.Itoa(numRules)+"_rules", func(b *testing.B) {
			snap := benchSnapshot(b, numRules)
			data := map[string]any{"TargetQuantity": 150.0, "LimitPrice": 105.5}
			ctx := context.Background()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				d := make(map[string]any, len(data))
				for k, v := range data {
					d[k] = v
				}
				if _, err := EvaluateRecord(ctx, snap, d, nil); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
