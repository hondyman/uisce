package document

import (
	"testing"
)

func TestParseFinancialNumber(t *testing.T) {
	tests := []struct {
		input    string
		expected *float64
	}{
		{"$1,250.50", floatPtr(1250.50)},
		{"(500.00)", floatPtr(-500.00)},
		{"—", nil},
		{"-", nil},
		{"$450,000,000", floatPtr(450000000.0)},
	}

	for _, tt := range tests {
		got := ParseFinancialNumber(tt.input)
		if tt.expected == nil {
			if got != nil {
				t.Errorf("expected nil for %s, got %v", tt.input, *got)
			}
		} else {
			if got == nil || *got != *tt.expected {
				t.Errorf("for input %s, expected %v, got %v", tt.input, *tt.expected, got)
			}
		}
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
