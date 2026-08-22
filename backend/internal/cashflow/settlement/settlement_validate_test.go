package settlement

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestSettlementRecord_Validate_ValidSubtypes(t *testing.T) {
	validSubtypes := []string{"dividend", "coupon_fixed_income", "capital_call", "lp_distribution", "corporate_action", "expense_fee"}
	for _, s := range validSubtypes {
		rec := SettlementRecord{SubtypeCode: s}
		if err := rec.Validate(); err != nil {
			t.Errorf("expected subtype %q to be valid, got %v", s, err)
		}
	}
}

func TestSettlementRecord_Validate_InvalidSubtype(t *testing.T) {
	rec := SettlementRecord{SubtypeCode: "invalid_type"}
	if err := rec.Validate(); err != ErrInvalidSubtype {
		t.Errorf("expected ErrInvalidSubtype, got %v", err)
	}
}

func TestSettlementRecord_Validate_Dividend(t *testing.T) {
	rec := SettlementRecord{
		SubtypeCode:    "dividend",
		AccountID:      uuid.New(),
		Amount:         decimal.NewFromFloat(100),
		Currency:       "USD",
		SettlementDate: time.Now(),
		SettlementStatus: "pending",
	}
	if err := rec.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}
