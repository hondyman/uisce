package sales_ledger

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestSalesLedgerRecord_Validate_ValidSubtypes(t *testing.T) {
	validSubtypes := []string{"aum_management_fee", "trading_commission", "performance_fee", "platform_subscription"}
	for _, s := range validSubtypes {
		rec := SalesLedgerRecord{SubtypeCode: s}
		if err := rec.Validate(); err != nil {
			t.Errorf("expected subtype %q to be valid, got %v", s, err)
		}
	}
}

func TestSalesLedgerRecord_Validate_InvalidSubtype(t *testing.T) {
	rec := SalesLedgerRecord{SubtypeCode: "invalid_type"}
	if err := rec.Validate(); err != ErrInvalidSubtype {
		t.Errorf("expected ErrInvalidSubtype, got %v", err)
	}
}

func TestSalesLedgerRecord_Validate_AUMManagementFee(t *testing.T) {
	rec := SalesLedgerRecord{
		SubtypeCode:      "aum_management_fee",
		InvoiceNumber:    "INV-001",
		ClientID:         uuid.New(),
		BillingPeriodEnd: time.Now(),
		InvoiceStatus:    "pending",
	}
	if err := rec.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}

func TestSalesLedgerRecord_Validate_WithOptionalFields(t *testing.T) {
	aumBasis := decimal.NewFromFloat(1000000)
	bps := float64(25)
	rec := SalesLedgerRecord{
		SubtypeCode:       "aum_management_fee",
		InvoiceNumber:     "INV-002",
		ClientID:          uuid.New(),
		BillingPeriodEnd: time.Now(),
		InvoiceStatus:     "pending",
		AUMBasisAmount:   &aumBasis,
		EffectiveFeeBPS:   &bps,
	}
	if err := rec.Validate(); err != nil {
		t.Errorf("expected valid, got %v", err)
	}
}
