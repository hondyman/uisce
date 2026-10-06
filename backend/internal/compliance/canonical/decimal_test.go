package canonical

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"
)

func TestDecimal6Normalization(t *testing.T) {
	tests := []struct {
		name     string
		input    decimal.Decimal
		expected string
	}{
		{
			name:     "Integer to 6 decimals",
			input:    decimal.NewFromInt(100),
			expected: "100.000000",
		},
		{
			name:     "Single decimal place",
			input:    decimal.NewFromFloat(123.4),
			expected: "123.400000",
		},
		{
			name:     "Small fractional decimal at 6 places",
			input:    decimal.NewFromFloat(0.000001),
			expected: "0.000001",
		},
		{
			name:     "Trailing zeros beyond 6 are accepted and formatted to 6",
			input:    decimal.RequireFromString("123.45000000"),
			expected: "123.450000",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := FormatDecimal6(tt.input)
			if err != nil {
				t.Fatalf("FormatDecimal6() unexpected error: %v", err)
			}
			if got != tt.expected {
				t.Errorf("FormatDecimal6() mismatch: got %s, expected %s", got, tt.expected)
			}
		})
	}
}

func TestDecimal6ScaleRejection(t *testing.T) {
	// Values with non-zero fractional digits beyond 6 decimal places must be rejected!
	exceededValues := []string{
		"123.4567891",
		"0.0000001",
		"100.0000009",
	}

	for _, s := range exceededValues {
		d := decimal.RequireFromString(s)
		_, err := FormatDecimal6(d)
		if err == nil {
			t.Errorf("FormatDecimal6(%s) expected ErrDecimalScaleExceeded, got nil", s)
		}
		if !errors.Is(err, ErrDecimalScaleExceeded) {
			t.Errorf("FormatDecimal6(%s) expected ErrDecimalScaleExceeded, got %v", s, err)
		}
	}
}

func TestCanonicalDecimalMap(t *testing.T) {
	inputMap := map[string]interface{}{
		"limit":       decimal.NewFromFloat(0.05),
		"marketValue": 1250000.5,
		"symbol":      "AAPL",
		"nested": map[string]interface{}{
			"nav": decimal.NewFromInt(10000000),
		},
	}

	res, err := CanonicalDecimalMap(inputMap)
	if err != nil {
		t.Fatalf("CanonicalDecimalMap() error: %v", err)
	}
	raw, err := Marshal(res)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	expected := `{"limit":"0.050000","marketValue":"1250000.500000","nested":{"nav":"10000000.000000"},"symbol":"AAPL"}`
	if string(raw) != expected {
		t.Errorf("CanonicalDecimalMap() JSON mismatch:\n got:      %s\n expected: %s", string(raw), expected)
	}
}
