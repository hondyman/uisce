package sales_ledger

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SalesLedgerSubtype string

const (
	SubtypeAUMManagementFee   SalesLedgerSubtype = "aum_management_fee"
	SubtypeTradingCommission  SalesLedgerSubtype = "trading_commission"
	SubtypePerformanceFee     SalesLedgerSubtype = "performance_fee"
	SubtypePlatformSubscription SalesLedgerSubtype = "platform_subscription"
)

var validSalesLedgerSubtypes = map[SalesLedgerSubtype]bool{
	SubtypeAUMManagementFee:    true,
	SubtypeTradingCommission:  true,
	SubtypePerformanceFee:     true,
	SubtypePlatformSubscription: true,
}

type SalesLedgerRecord struct {
	ID               uuid.UUID        `db:"id" json:"id"`
	TenantID         uuid.UUID        `db:"tenant_id" json:"tenant_id"`
	InvoiceNumber    string           `db:"invoice_number" json:"invoice_number"`
	ClientID         uuid.UUID        `db:"client_id" json:"client_id"`
	SubtypeCode      string           `db:"subtype_code" json:"subtype_code"`
	BillingPeriodEnd time.Time        `db:"billing_period_end" json:"billing_period_end"`
	AUMBasisAmount  *decimal.Decimal `db:"aum_basis_amount" json:"aum_basis_amount,omitempty"`
	EffectiveFeeBPS  *float64         `db:"effective_fee_bps" json:"effective_fee_bps,omitempty"`
	HWMBenchmarkNAV *decimal.Decimal `db:"hwm_benchmark_nav" json:"hwm_benchmark_nav,omitempty"`
	InvoiceStatus   string           `db:"invoice_status" json:"invoice_status"`

	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	ValidFrom time.Time  `db:"valid_from" json:"valid_from"`
	ValidTo   *time.Time `db:"valid_to" json:"valid_to,omitempty"`
}

func (s SalesLedgerRecord) Validate() error {
	if !validSalesLedgerSubtypes[SalesLedgerSubtype(s.SubtypeCode)] {
		return ErrInvalidSubtype
	}
	return nil
}
