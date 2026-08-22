package account

import (
	"testing"

	"github.com/google/uuid"
)

func TestAccountRecord_Validate_ValidSubtypes(t *testing.T) {
	validSubtypes := []string{"retail_wealth", "sma", "trust_estate", "qualified_retirement", "corporate_treasury"}
	for _, s := range validSubtypes {
		rec := AccountRecord{SubtypeCode: s}
		if err := rec.Validate(); err != nil {
			t.Errorf("expected subtype %q to be valid, got %v", s, err)
		}
	}
}

func TestAccountRecord_Validate_InvalidSubtype(t *testing.T) {
	rec := AccountRecord{SubtypeCode: "invalid_type"}
	if err := rec.Validate(); err != ErrInvalidSubtype {
		t.Errorf("expected ErrInvalidSubtype, got %v", err)
	}
}

func TestAccountRecord_Validate_InstitutionalRequiresSponsor(t *testing.T) {
	rec := AccountRecord{SubtypeCode: "institutional"}
	if err := rec.Validate(); err != ErrRequiresSponsorID {
		t.Errorf("expected ErrRequiresSponsorID, got %v", err)
	}
}

func TestAccountRecord_Validate_InstitutionalWithSponsor(t *testing.T) {
	sponsorID := uuid.New()
	rec := AccountRecord{SubtypeCode: "institutional", SponsorID: &sponsorID}
	if err := rec.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestAccountRecord_Validate_QualifiedRetirementWithERISARequiresPlanType(t *testing.T) {
	erisaFlag := true
	rec := AccountRecord{SubtypeCode: "qualified_retirement", ErisaFlag: &erisaFlag}
	if err := rec.Validate(); err != ErrRequiresPlanType {
		t.Errorf("expected ErrRequiresPlanType, got %v", err)
	}
}

func TestAccountRecord_Validate_QualifiedRetirementWithERISAAndPlanType(t *testing.T) {
	erisaFlag := true
	planType := "401k"
	rec := AccountRecord{SubtypeCode: "qualified_retirement", ErisaFlag: &erisaFlag, PlanType: &planType}
	if err := rec.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestAccountRecord_Validate_QualifiedRetirementWithoutERISA(t *testing.T) {
	rec := AccountRecord{SubtypeCode: "qualified_retirement"}
	if err := rec.Validate(); err != nil {
		t.Errorf("expected valid without ERISA flag, got %v", err)
	}
}
