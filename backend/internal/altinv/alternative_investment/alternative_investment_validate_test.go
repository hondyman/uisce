package alternative_investment

import (
	"errors"
	"testing"
)

func TestAlternativeInvestmentRecord_Validate_ValidSubtypes(t *testing.T) {
	validSubtypes := []string{"PRIVATE_EQUITY", "VENTURE_CAPITAL", "HEDGE_FUND", "REAL_ESTATE", "DIRECT_INVESTMENT", "INFRASTRUCTURE", "PRIVATE_DEBT"}
	for _, s := range validSubtypes {
		rec := AlternativeInvestmentRecord{SubtypeCode: s}
		if err := rec.Validate(); err != nil {
			t.Errorf("expected subtype %q to be valid, got %v", s, err)
		}
	}
}

func TestAlternativeInvestmentRecord_Validate_InvalidSubtype(t *testing.T) {
	rec := AlternativeInvestmentRecord{SubtypeCode: "invalid_type"}
	if err := rec.Validate(); err == nil || !errors.Is(err, ErrInvalidSubtype) {
		t.Errorf("expected ErrInvalidSubtype, got %v", err)
	}
}
