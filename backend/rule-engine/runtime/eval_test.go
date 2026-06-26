package runtime

import (
	"testing"
)

func TestEvaluateRule(t *testing.T) {
	tests := []struct {
		name     string
		rule     RuleNode
		context  map[string]interface{}
		expected bool
	}{
		{
			name: "simple condition equals",
			rule: RuleNode{
				Type: "Condition",
				Condition: &Condition{
					Field:    "status",
					Operator: "equals",
					Value:    "active",
				},
			},
			context: map[string]interface{}{
				"status": "active",
			},
			expected: true,
		},
		{
			name: "simple condition not equals",
			rule: RuleNode{
				Type: "Condition",
				Condition: &Condition{
					Field:    "status",
					Operator: "equals",
					Value:    "active",
				},
			},
			context: map[string]interface{}{
				"status": "inactive",
			},
			expected: false,
		},
		{
			name: "AND group both true",
			rule: RuleNode{
				Type: "Group",
				Group: &Group{
					Operator: "AND",
					Children: []RuleNode{
						{
							Type: "Condition",
							Condition: &Condition{
								Field:    "status",
								Operator: "equals",
								Value:    "active",
							},
						},
						{
							Type: "Condition",
							Condition: &Condition{
								Field:    "type",
								Operator: "equals",
								Value:    "premium",
							},
						},
					},
				},
			},
			context: map[string]interface{}{
				"status": "active",
				"type":   "premium",
			},
			expected: true,
		},
		{
			name: "AND group one false",
			rule: RuleNode{
				Type: "Group",
				Group: &Group{
					Operator: "AND",
					Children: []RuleNode{
						{
							Type: "Condition",
							Condition: &Condition{
								Field:    "status",
								Operator: "equals",
								Value:    "active",
							},
						},
						{
							Type: "Condition",
							Condition: &Condition{
								Field:    "type",
								Operator: "equals",
								Value:    "premium",
							},
						},
					},
				},
			},
			context: map[string]interface{}{
				"status": "active",
				"type":   "basic",
			},
			expected: false,
		},
		{
			name: "OR group one true",
			rule: RuleNode{
				Type: "Group",
				Group: &Group{
					Operator: "OR",
					Children: []RuleNode{
						{
							Type: "Condition",
							Condition: &Condition{
								Field:    "status",
								Operator: "equals",
								Value:    "active",
							},
						},
						{
							Type: "Condition",
							Condition: &Condition{
								Field:    "type",
								Operator: "equals",
								Value:    "premium",
							},
						},
					},
				},
			},
			context: map[string]interface{}{
				"status": "inactive",
				"type":   "premium",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EvaluateRule(tt.rule, tt.context)
			if result != tt.expected {
				t.Errorf("EvaluateRule() = %v, want %v", result, tt.expected)
			}
		})
	}
}
