package rules

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
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
