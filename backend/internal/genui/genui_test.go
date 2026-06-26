package genui

import (
	"context"
	"testing"
)

func TestIntentClassifier_Classify(t *testing.T) {
	classifier := NewIntentClassifier()
	ctx := context.Background()

	tests := []struct {
		name           string
		query          string
		expectedType   string
		expectedObject string
	}{
		{
			name:           "portfolio performance chart",
			query:          "Show portfolio performance over time",
			expectedType:   "chart",
			expectedObject: "Portfolio",
		},
		{
			name:           "holdings table",
			query:          "Display holdings in a table",
			expectedType:   "grid",
			expectedObject: "Position",
		},
		{
			name:           "compare accounts",
			query:          "Compare account values",
			expectedType:   "chart",
			expectedObject: "Account",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			intent, err := classifier.Classify(ctx, tt.query)
			if err != nil {
				t.Fatalf("Classify error: %v", err)
			}

			if intent.Type != tt.expectedType {
				t.Errorf("Expected type %s, got %s", tt.expectedType, intent.Type)
			}

			found := false
			for _, obj := range intent.Objects {
				if obj == tt.expectedObject {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected object %s not found in %v", tt.expectedObject, intent.Objects)
			}
		})
	}
}

func TestLayoutBuilder_BuildChart(t *testing.T) {
	builder := NewLayoutBuilder()
	ctx := context.Background()

	intent := &Intent{
		Type:    "chart",
		Objects: []string{"Portfolio"},
		Metrics: []string{"nav"},
	}

	layout, err := builder.Build(ctx, intent)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if len(layout.Components) == 0 {
		t.Fatal("Expected at least one component")
	}

	if layout.Components[0].Type != "chart" {
		t.Errorf("Expected chart component, got %s", layout.Components[0].Type)
	}
}

func TestLayoutBuilder_BuildDashboard(t *testing.T) {
	builder := NewLayoutBuilder()
	ctx := context.Background()

	intent := &Intent{
		Type:    "dashboard",
		Objects: []string{"Portfolio"},
		Metrics: []string{"nav", "return_pct"},
	}

	layout, err := builder.Build(ctx, intent)
	if err != nil {
		t.Fatalf("Build error: %v", err)
	}

	if len(layout.Components) < 2 {
		t.Errorf("Expected multiple components, got %d", len(layout.Components))
	}
}
