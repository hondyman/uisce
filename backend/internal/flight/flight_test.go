package flight_test

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/hondyman/uisce/backend/internal/flight"
)

func TestVectorMasker_PartialMask(t *testing.T) {
	mem := memory.NewGoAllocator()

	schema := flight.BuildPortfolioSchema()
	builder := flight.NewPortfolioRecordBuilder(mem, schema)
	defer builder.Release()

	builder.Append("99e99e99-99e9-49e9-89e9-99e99e99e999", "PT-001", "1234567890123456", 2100000.0, 0.21)
	rec := builder.NewRecord()
	defer rec.Release()

	if rec.NumCols() != 5 {
		t.Fatalf("expected 5 columns, got %d", rec.NumCols())
	}
}

func TestMaskPII(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1234567890123456", "1234****3456"},
		{"12345678", "********"},
		{"ABCD1234EFGH5678", "ABCD****5678"},
		{"short", "*****"},
	}

	for _, tt := range tests {
		result := flight.MaskPII(tt.input)
		if result != tt.expected {
			t.Errorf("MaskPII(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestBuildPortfolioSchema(t *testing.T) {
	schema := flight.BuildPortfolioSchema()
	if schema.NumFields() != 5 {
		t.Errorf("expected 5 fields, got %d", schema.NumFields())
	}

	expectedFields := []string{"tenant_id", "portfolio_id", "security_isin", "market_value", "effective_exposure_pct"}
	for i, name := range expectedFields {
		if schema.Field(i).Name != name {
			t.Errorf("field %d: expected %q, got %q", i, name, schema.Field(i).Name)
		}
	}
}
