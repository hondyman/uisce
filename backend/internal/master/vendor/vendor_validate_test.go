package vendor

import (
	"testing"
)

func TestVendorRecord_Validate_ValidSubtypes(t *testing.T) {
	validSubtypes := []string{"custodian_prime_broker", "market_data", "fund_admin", "cloud_tech"}
	for _, s := range validSubtypes {
		rec := VendorRecord{SubtypeCode: s}
		if err := rec.Validate(); err != nil {
			t.Errorf("expected subtype %q to be valid, got %v", s, err)
		}
	}
}

func TestVendorRecord_Validate_InvalidSubtype(t *testing.T) {
	rec := VendorRecord{SubtypeCode: "invalid_type"}
	if err := rec.Validate(); err != ErrInvalidSubtype {
		t.Errorf("expected ErrInvalidSubtype, got %v", err)
	}
}
