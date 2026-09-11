package vm

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdvancedEvaluator_Evaluate(t *testing.T) {
	evaluator := NewAdvancedEvaluator()

	tests := []struct {
		name     string
		ruleJSON string
		data     map[string]interface{}
		want     bool
		wantErr  bool
	}{
		{
			name: "Simple AND - Pass",
			ruleJSON: `
			{
				"type": "group",
				"id": "g1",
				"operator": "AND",
				"conditions": [
					{
						"type": "condition",
						"id": "c1", 
						"field": "age", 
						"operator": ">", 
						"value": 18
					},
					{
						"type": "condition",
						"id": "c2", 
						"field": "status", 
						"operator": "equals", 
						"value": "active"
					}
				]
			}`,
			data: map[string]interface{}{"age": 25, "status": "active"},
			want: true,
		},
		{
			name: "Simple AND - Fail",
			ruleJSON: `
			{
				"type": "group",
				"id": "g1",
				"operator": "AND",
				"conditions": [
					{
						"type": "condition",
						"id": "c1", 
						"field": "age", 
						"operator": ">", 
						"value": 18
					},
					{
						"type": "condition",
						"id": "c2", 
						"field": "status", 
						"operator": "equals", 
						"value": "active"
					}
				]
			}`,
			data: map[string]interface{}{"age": 25, "status": "inactive"},
			want: false,
		},
		{
			name: "Nested OR inside AND",
			ruleJSON: `
			{
				"type": "group",
				"id": "root",
				"operator": "AND",
				"conditions": [
					{
						"type": "condition",
						"id": "c1", 
						"field": "role", 
						"operator": "equals", 
						"value": "admin"
					},
					{
						"type": "group",
						"id": "g2",
						"operator": "OR",
						"conditions": [
							{
								"type": "condition",
								"id": "c2", 
								"field": "department", 
								"operator": "equals", 
								"value": "IT"
							},
							{
								"type": "condition",
								"id": "c3", 
								"field": "department", 
								"operator": "equals", 
								"value": "Security"
							}
						]
					}
				]
			}`,
			data: map[string]interface{}{"role": "admin", "department": "Security"},
			want: true,
		},
		{
			name: "Nested OR inside AND - Fail",
			ruleJSON: `
			{
				"type": "group",
				"id": "root",
				"operator": "AND",
				"conditions": [
					{
						"type": "condition",
						"id": "c1", 
						"field": "role", 
						"operator": "equals", 
						"value": "admin"
					},
					{
						"type": "group",
						"id": "g2",
						"operator": "OR",
						"conditions": [
							{
								"type": "condition",
								"id": "c2", 
								"field": "department", 
								"operator": "equals", 
								"value": "IT"
							},
							{
								"type": "condition",
								"id": "c3", 
								"field": "department", 
								"operator": "equals", 
								"value": "Security"
							}
						]
					}
				]
			}`,
			data: map[string]interface{}{"role": "admin", "department": "HR"},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var node RuleNode
			err := json.Unmarshal([]byte(tt.ruleJSON), &node)
			assert.NoError(t, err, "Failed to unmarshal JSON")

			got, err := evaluator.Evaluate(node, tt.data)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

// TestAdvancedEvaluator_CollectionAggregation covers the collection-aggregation
// feature: SUM(field) over a collection-typed BO relation field (e.g.
// OrderAllocations.target_qty) resolves through the dotted FieldRef path
// via ResolveFieldPathArray, producing a []any which requireFloatSlice
// already knows how to flatten. The four cases below correspond to the four
// design decisions settled in the Slice 1 PR.
//
// Design decisions:
//   - Decision 1: missing field in a row → loud rule error
//   - Decision 2: bare collection used outside aggregate → specific error
//   - Decision 3: legacy scalar keys coexist (regression below)
func TestAdvancedEvaluator_CollectionAggregation(t *testing.T) {
	evaluator := NewAdvancedEvaluator()

	// Rule: SUM(OrderAllocations.target_qty) == TargetQuantity
	// Context: OrderAllocations is a []map[string] of allocation rows;
	// TargetQuantity is the expected total on the parent order.
	sumRule := map[string]any{
		"type": "expression",
		"root": map[string]any{
			"op": "==",
			"left": map[string]any{
				"func": "SUM",
				"args": []any{
					map[string]any{"path": "OrderAllocations.target_qty"},
				},
			},
			"right": map[string]any{"path": "TargetQuantity"},
		},
	}

	t.Run("SUM over collection resolves correctly", func(t *testing.T) {
		node := parseRuleNode(t, sumRule)
		data := map[string]any{
			"OrderAllocations": []any{
				map[string]any{"target_qty": 30.0},
				map[string]any{"target_qty": 70.0},
			},
			"TargetQuantity": 100.0,
		}
		got, err := evaluator.Evaluate(node, data)
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("SUM over collection fails when one row is missing the field — loud error", func(t *testing.T) {
		node := parseRuleNode(t, sumRule)
		data := map[string]any{
			"OrderAllocations": []any{
				map[string]any{"target_qty": 30.0},
				map[string]any{}, // target_qty absent
			},
			"TargetQuantity": 100.0,
		}
		_, err := evaluator.Evaluate(node, data)
		require.Error(t, err)
		errMsg := err.Error()
		assert.True(t, strings.Contains(errMsg, "field not found in one or more rows"), "expected loud error about missing field, got: %s", errMsg)
		assert.True(t, strings.Contains(errMsg, "OrderAllocations.target_qty"), "error should mention the path: %s", errMsg)
	})

	t.Run("bare collection used outside aggregate — specific error", func(t *testing.T) {
		// Rule: OrderAllocations > 5  (bare collection, not inside SUM)
		node := parseRuleNode(t, map[string]any{
			"type": "expression",
			"root": map[string]any{
				"op": ">",
				"left":  map[string]any{"path": "OrderAllocations"},
				"right": map[string]any{"value": 5.0},
			},
		})
		data := map[string]any{
			"OrderAllocations": []any{
				map[string]any{"target_qty": 30.0},
				map[string]any{"target_qty": 70.0},
			},
		}
		_, err := evaluator.Evaluate(node, data)
		require.Error(t, err)
		errMsg := err.Error()
		assert.True(t, strings.Contains(errMsg, "collection used in comparison"), "bare collection should error specifically, got: %s", errMsg)
	})

	t.Run("legacy scalar rules still work — regression", func(t *testing.T) {
		// Rule: TargetQuantity > 0  (flat field, no dots — existing path)
		node := parseRuleNode(t, map[string]any{
			"type": "expression",
			"root": map[string]any{
				"op": ">",
				"left":  map[string]any{"path": "TargetQuantity"},
				"right": map[string]any{"value": 0.0},
			},
		})
		data := map[string]any{
			"TargetQuantity": 100.0,
		}
		got, err := evaluator.Evaluate(node, data)
		require.NoError(t, err)
		assert.True(t, got)
	})

	t.Run("dotted path through single related object — scalar comparison must work", func(t *testing.T) {
		// Rule: parent.quantity > 0  (dotted path through a single related object,
		// not a collection — cross-entity fields like parent.quantity, client.risk_score.
		// ResolveFieldPathArray returns [100.0] for a non-array chain; must unwrap
		// to scalar so the comparison succeeds without hitting the Decision-2 guard.)
		node := parseRuleNode(t, map[string]any{
			"type": "expression",
			"root": map[string]any{
				"op": ">",
				"left":  map[string]any{"path": "parent.quantity"},
				"right": map[string]any{"value": 0.0},
			},
		})
		data := map[string]any{
			"parent": map[string]any{"quantity": 100.0},
		}
		got, err := evaluator.Evaluate(node, data)
		require.NoError(t, err)
		assert.True(t, got, "parent.quantity > 0 should pass for parent={quantity: 100}")
	})

	t.Run("empty collection evaluates to 0 — SUM over zero rows is 0, not an error", func(t *testing.T) {
		// An order with no allocations yet: OrderAllocations = [].
		// SUM(OrderAllocations.target_qty) = 0, and 0 != TargetQuantity(100)
		// should produce a violation — not a rule error.
		node := parseRuleNode(t, sumRule)
		data := map[string]any{
			"OrderAllocations": []any{},
			"TargetQuantity":    100.0,
		}
		got, err := evaluator.Evaluate(node, data)
		require.NoError(t, err)
		assert.False(t, got, "empty collection should evaluate: SUM([]) = 0, and 0 != 100 is a correct violation")
	})

}

func parseRuleNode(t *testing.T, nodeMap map[string]any) RuleNode {
	t.Helper()
	blob, err := json.Marshal(nodeMap)
	require.NoError(t, err)
	var node RuleNode
	err = json.Unmarshal(blob, &node)
	require.NoError(t, err, "failed to unmarshal rule node from %s", string(blob))
	return node
}
