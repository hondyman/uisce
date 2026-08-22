package settlement

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type SettlementSubtype string

const (
	SubtypeDividend          SettlementSubtype = "dividend"
	SubtypeCouponFixedIncome SettlementSubtype = "coupon_fixed_income"
	SubtypeCapitalCall       SettlementSubtype = "capital_call"
	SubtypeLPDistribution    SettlementSubtype = "lp_distribution"
	SubtypeCorporateAction   SettlementSubtype = "corporate_action"
	SubtypeExpenseFee        SettlementSubtype = "expense_fee"
)

var validSettlementSubtypes = map[SettlementSubtype]bool{
	SubtypeDividend:          true,
	SubtypeCouponFixedIncome: true,
	SubtypeCapitalCall:       true,
	SubtypeLPDistribution:    true,
	SubtypeCorporateAction:   true,
	SubtypeExpenseFee:        true,
}

type SettlementRecord struct {
	ID              uuid.UUID        `db:"id" json:"id"`
	TenantID        uuid.UUID        `db:"tenant_id" json:"tenant_id"`
	AccountID       uuid.UUID        `db:"account_id" json:"account_id"`
	Amount          decimal.Decimal  `db:"amount" json:"amount"`
	Currency        string           `db:"currency" json:"currency"`
	SettlementDate  time.Time        `db:"settlement_date" json:"settlement_date"`
	SettlementStatus string          `db:"settlement_status" json:"settlement_status"`
	SubtypeCode     string           `db:"subtype_code" json:"subtype_code"`

	ExDate                 *time.Time       `db:"ex_date" json:"ex_date,omitempty"`
	RecordDate             *time.Time       `db:"record_date" json:"record_date,omitempty"`
	DripReinvestFlag       *bool            `db:"drip_reinvest_flag" json:"drip_reinvest_flag,omitempty"`
	TaxWithholdingAmount   *decimal.Decimal `db:"tax_withholding_amount" json:"tax_withholding_amount,omitempty"`
	CouponPeriodStart      *time.Time       `db:"coupon_period_start" json:"coupon_period_start,omitempty"`
	AccruedInterest        *decimal.Decimal `db:"accrued_interest" json:"accrued_interest,omitempty"`
	PaymentFrequency       *string          `db:"payment_frequency" json:"payment_frequency,omitempty"`
	CallNoticeID           *string          `db:"call_notice_id" json:"call_notice_id,omitempty"`
	DueDate                *time.Time       `db:"due_date" json:"due_date,omitempty"`
	ManagementFeePortion    *decimal.Decimal `db:"management_fee_portion" json:"management_fee_portion,omitempty"`
	InvestmentPortion      *decimal.Decimal `db:"investment_portion" json:"investment_portion,omitempty"`
	ReturnOfCapital        *decimal.Decimal `db:"return_of_capital" json:"return_of_capital,omitempty"`
	PreferredReturn        *decimal.Decimal `db:"preferred_return" json:"preferred_return,omitempty"`
	CarriedInterestRetained *decimal.Decimal `db:"carried_interest_retained" json:"carried_interest_retained,omitempty"`
	ActionTypeCode         *string          `db:"action_type_code" json:"action_type_code,omitempty"`
	CashInLieuAmount       *decimal.Decimal `db:"cash_in_lieu_amount" json:"cash_in_lieu_amount,omitempty"`
	MandatoryFlag          *bool            `db:"mandatory_flag" json:"mandatory_flag,omitempty"`
	FeeCategory            *string          `db:"fee_category" json:"fee_category,omitempty"`
	InvoiceReferenceID     *string          `db:"invoice_reference_id" json:"invoice_reference_id,omitempty"`
	VATAmount              *decimal.Decimal `db:"vat_amount" json:"vat_amount,omitempty"`

	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	ValidFrom time.Time  `db:"valid_from" json:"valid_from"`
	ValidTo   *time.Time `db:"valid_to" json:"valid_to,omitempty"`
}

func (s SettlementRecord) Validate() error {
	if !validSettlementSubtypes[SettlementSubtype(s.SubtypeCode)] {
		return ErrInvalidSubtype
	}
	return nil
}
