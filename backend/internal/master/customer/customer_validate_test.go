package customer

import (
	"testing"
)

func TestCustomerRecord_Validate_ValidSubtypes(t *testing.T) {
	validSubtypes := []string{"institutional_client", "private_wealth", "broker_dealer", "corporate_treasury"}
	for _, s := range validSubtypes {
		rec := CustomerRecord{SubtypeCode: s}
		if err := rec.Validate(); err != nil {
			t.Errorf("expected subtype %q to be valid, got %v", s, err)
		}
	}
}

func TestCustomerRecord_Validate_InvalidSubtype(t *testing.T) {
	rec := CustomerRecord{SubtypeCode: "invalid_type"}
	if err := rec.Validate(); err != ErrInvalidSubtype {
		t.Errorf("expected ErrInvalidSubtype, got %v", err)
	}
}
