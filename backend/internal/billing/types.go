package billing

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
)

type FeeType string

const (
	FeeTypeAUMTiered    FeeType = "AUM_TIERED"
	FeeTypeFlatAnnual   FeeType = "FLAT_ANNUAL"
	FeeTypePerformance  FeeType = "PERFORMANCE"
	FeeTypeSubscription FeeType = "SUBSCRIPTION"
	FeeTypeHybrid       FeeType = "HYBRID"
)

type FeeSchedule struct {
	ScheduleID    uuid.UUID      `db:"schedule_id" json:"schedule_id"`
	ScheduleName  string         `db:"schedule_name" json:"schedule_name"`
	FeeType       FeeType        `db:"fee_type" json:"fee_type"`
	TierStructure types.JSONText `db:"tier_structure" json:"tier_structure,omitempty"` // [{"min": 0, "max": 1000000, "rate": 0.01}]

	// Performance Fee
	HurdleRate      *float64 `db:"performance_hurdle_rate" json:"performance_hurdle_rate,omitempty"`
	PerformanceRate *float64 `db:"performance_fee_rate" json:"performance_fee_rate,omitempty"`
	HighWaterMark   bool     `db:"high_water_mark_enabled" json:"high_water_mark_enabled"`

	BillingFrequency string `db:"billing_frequency" json:"billing_frequency"`                   // MONTHLY, QUARTERLY, ANNUAL
	BillingTiming    string `db:"billing_advance_or_arrears" json:"billing_advance_or_arrears"` // ADVANCE, ARREARS

	MinQuarterlyFee *float64 `db:"minimum_quarterly_fee" json:"minimum_quarterly_fee,omitempty"`
	MinAnnualFee    *float64 `db:"minimum_annual_fee" json:"minimum_annual_fee,omitempty"`

	ExcludeCash bool `db:"exclude_cash_from_aum" json:"exclude_cash_from_aum"`
	ExcludeAlts bool `db:"exclude_alternatives_from_aum" json:"exclude_alternatives_from_aum"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type ClientFeeAssignment struct {
	AssignmentID  uuid.UUID  `db:"assignment_id" json:"assignment_id"`
	ClientID      uuid.UUID  `db:"client_id" json:"client_id"`
	AccountID     *uuid.UUID `db:"account_id" json:"account_id,omitempty"`
	ScheduleID    uuid.UUID  `db:"schedule_id" json:"schedule_id"`
	EffectiveDate time.Time  `db:"effective_date" json:"effective_date"`
	EndDate       *time.Time `db:"end_date" json:"end_date,omitempty"`

	CustomDiscountPct *float64 `db:"custom_discount_pct" json:"custom_discount_pct,omitempty"`
	CustomMinFee      *float64 `db:"custom_minimum_fee" json:"custom_minimum_fee,omitempty"`

	InvoiceEmail  *string `db:"invoice_contact_email" json:"invoice_contact_email,omitempty"`
	PaymentMethod *string `db:"payment_method" json:"payment_method,omitempty"`
	BillingDay    *int    `db:"billing_day_of_month" json:"billing_day_of_month,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type FeeCalculation struct {
	CalculationID uuid.UUID `db:"calculation_id" json:"calculation_id"`
	ClientID      uuid.UUID `db:"client_id" json:"client_id"`
	PeriodStart   time.Time `db:"billing_period_start" json:"billing_period_start"`
	PeriodEnd     time.Time `db:"billing_period_end" json:"billing_period_end"`

	AvgDailyBalance float64 `db:"average_daily_balance" json:"average_daily_balance"`
	AUMFee          float64 `db:"aum_based_fee" json:"aum_based_fee"`

	PortfolioReturn *float64 `db:"portfolio_return_pct" json:"portfolio_return_pct,omitempty"`
	HurdleReturn    *float64 `db:"hurdle_return_pct" json:"hurdle_return_pct,omitempty"`
	ExcessReturn    *float64 `db:"excess_return" json:"excess_return,omitempty"`
	PerformanceFee  float64  `db:"performance_fee" json:"performance_fee"`

	PriorAdjustment float64 `db:"prior_period_adjustment" json:"prior_period_adjustment"`
	DiscountAmount  float64 `db:"discount_amount" json:"discount_amount"`
	MinFeeAdj       float64 `db:"minimum_fee_adjustment" json:"minimum_fee_adjustment"`

	TotalFee float64 `db:"total_fee" json:"total_fee"`

	Status     string     `db:"calculation_status" json:"calculation_status"` // DRAFT, APPROVED, INVOICED, PAID
	ApprovedBy *uuid.UUID `db:"approved_by" json:"approved_by,omitempty"`
	ApprovedAt *time.Time `db:"approved_at" json:"approved_at,omitempty"`

	InvoiceID *uuid.UUID `db:"invoice_id" json:"invoice_id,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
}
