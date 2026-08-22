package personnel

import (
	"testing"
)

func TestPersonnelRecord_Validate_ValidSubtypes(t *testing.T) {
	validSubtypes := []string{"portfolio_manager", "trade_execution", "compliance_officer", "client_advisor"}
	for _, s := range validSubtypes {
		rec := PersonnelRecord{SubtypeCode: s}
		if err := rec.Validate(); err != nil {
			t.Errorf("expected subtype %q to be valid, got %v", s, err)
		}
	}
}

func TestPersonnelRecord_Validate_InvalidSubtype(t *testing.T) {
	rec := PersonnelRecord{SubtypeCode: "invalid_type"}
	if err := rec.Validate(); err != ErrInvalidSubtype {
		t.Errorf("expected ErrInvalidSubtype, got %v", err)
	}
}
