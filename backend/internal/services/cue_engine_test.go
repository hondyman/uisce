package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCueEngine_EvaluateValidation(t *testing.T) {
	engine := NewCueEngine()
	ctx := context.Background()

	tests := []struct {
		name      string
		script    string
		data      map[string]interface{}
		wantValid bool
		wantMsg   string
	}{
		{
			name: "Basic valid constraint",
			script: `
record: {
	age: >18
}
`,
			data:      map[string]interface{}{"age": 20},
			wantValid: true,
		},
		{
			name: "Basic invalid constraint",
			script: `
record: {
	age: >18
}
`,
			data:      map[string]interface{}{"age": 10},
			wantValid: false,
			wantMsg:   "validation failed",
		},
		{
			name: "With explicit result message (valid)",
			script: `
record: {
	age: int
}
result: {
	valid: record.age > 18
	message: "Age must be over 18"
}
`,
			data:      map[string]interface{}{"age": 20},
			wantValid: true,
		},
		{
			name: "With explicit result message (invalid)",
			script: `
record: {
	age: int
}
result: {
	valid: record.age > 18
	message: "Age must be over 18, got \(record.age)"
}
`,
			data:      map[string]interface{}{"age": 15},
			wantValid: false,
			wantMsg:   "Age must be over 18, got 15",
		},
		{
			name:      "Syntax error in script",
			script:    `record: { incomplete`,
			data:      map[string]interface{}{},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.EvaluateValidation(ctx, tt.script, tt.data)
			assert.NoError(t, err)
			assert.Equal(t, tt.wantValid, got.IsValid)
			if !tt.wantValid && tt.wantMsg != "" && tt.wantMsg != "validation failed" {
				assert.Contains(t, got.Message, tt.wantMsg)
			}
			if tt.name == "Syntax error in script" {
				assert.Contains(t, got.Message, "Script compilation failed")
			}
		})
	}
}

func TestCueEngine_CrossRecordValidation(t *testing.T) {
	engine := NewCueEngine()
	ctx := context.Background()

	tests := []struct {
		name      string
		script    string
		data      map[string]interface{}
		wantValid bool
	}{
		{
			name: "Batch Aggregation: Total Salary <= Budget",
			script: `
import "list"

record: {
    budget: number
    employees: [...{
        salary: number
    }]
}

// Validation logic
totalSalary: list.Sum([for e in record.employees { e.salary }])

// Constraint: valid must be true
valid: totalSalary <= record.budget
valid: true
`,
			data: map[string]interface{}{
				"budget": 100000,
				"employees": []map[string]interface{}{
					{"salary": 40000},
					{"salary": 50000},
				},
			},
			wantValid: true,
		},
		{
			name: "Batch Aggregation: Budget Exceeded",
			script: `
import "list"

record: {
    budget: number
    employees: [...{
        salary: number
    }]
}

totalSalary: list.Sum([for e in record.employees { e.salary }])
valid: totalSalary <= record.budget
valid: true
`,
			data: map[string]interface{}{
				"budget": 80000,
				"employees": []map[string]interface{}{
					{"salary": 40000},
					{"salary": 50000},
				},
			},
			wantValid: false,
		},
		{
			name: "Uniqueness Check: Duplicate IDs",
			script: `
record: {
    items: [...{ id: string }]
}

// Check for uniqueness
ids: { for i in record.items { (i.id): true } }

isUnique: len(ids) == len(record.items)
isUnique: true
`,
			data: map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": "A"},
					{"id": "B"},
					{"id": "A"}, // Duplicate
				},
			},
			wantValid: false,
		},
		{
			name: "Uniqueness Check: No Duplicates",
			script: `
record: {
    items: [...{ id: string }]
}
ids: { for i in record.items { (i.id): true } }

isUnique: len(ids) == len(record.items)
isUnique: true
`,
			data: map[string]interface{}{
				"items": []map[string]interface{}{
					{"id": "A"},
					{"id": "B"},
					{"id": "C"},
				},
			},
			wantValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.EvaluateValidation(ctx, tt.script, tt.data)
			assert.NoError(t, err)

			// With pure constraints (no 'result' struct), failure usually means 'validation failed'
			// EvaluateValidation returns valid=false if Unify fails
			if got.IsValid != tt.wantValid {
				t.Errorf("Mismatch for %s. Valid: %v, Want: %v. Msg: %s", tt.name, got.IsValid, tt.wantValid, got.Message)
			}
		})
	}
}
