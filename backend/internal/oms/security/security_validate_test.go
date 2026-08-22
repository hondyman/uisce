package security

import (
	"testing"
)

func TestSecurityRecord_Validate_ValidSubtypes(t *testing.T) {
	validSubtypes := []string{"equity", "sovereign_debt", "corporate_debt", "structured_abs_mbs", "etd_derivative", "otc_derivative"}
	for _, s := range validSubtypes {
		rec := SecurityRecord{SubtypeCode: s}
		if err := rec.Validate(); err != nil {
			t.Errorf("expected subtype %q to be valid, got %v", s, err)
		}
	}
}

func TestSecurityRecord_Validate_InvalidSubtype(t *testing.T) {
	rec := SecurityRecord{SubtypeCode: "invalid_type"}
	if err := rec.Validate(); err != ErrInvalidSubtype {
		t.Errorf("expected ErrInvalidSubtype, got %v", err)
	}
}
