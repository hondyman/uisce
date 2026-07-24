package feebilling

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// FeeType represents different billing models
type FeeType string

const (
	FeeTypeAUMTiered    FeeType = "AUM_TIERED"
	FeeTypeAUMFlat      FeeType = "AUM_FLAT"
	FeeTypePerformance  FeeType = "PERFORMANCE"
	FeeTypeSubscription FeeType = "SUBSCRIPTION"
	FeeTypeHybrid       FeeType = "HYBRID"
	FeeTypeRetainer     FeeType = "RETAINER"
	FeeTypeHourly       FeeType = "HOURLY"
)

// BillingFrequency represents how often fees are billed
type BillingFrequency string

const (
	Monthly    BillingFrequency = "MONTHLY"
	Quarterly  BillingFrequency = "QUARTERLY"
	SemiAnnual BillingFrequency = "SEMI_ANNUAL"
	Annual     BillingFrequency = "ANNUAL"
)

// BillingTiming represents when fees are billed
type BillingTiming string

const (
	Advance BillingTiming = "ADVANCE"
	Arrears BillingTiming = "ARREARS"
)

// CalculationStatus represents the state of a fee calculation
type CalculationStatus string

const (
	StatusDraft         CalculationStatus = "DRAFT"
	StatusPendingReview CalculationStatus = "PENDING_REVIEW"
	StatusApproved      CalculationStatus = "APPROVED"
	StatusInvoiced      CalculationStatus = "INVOICED"
	StatusPaid          CalculationStatus = "PAID"
	StatusPartiallyPaid CalculationStatus = "PARTIALLY_PAID"
	StatusWrittenOff    CalculationStatus = "WRITTEN_OFF"
)

// PaymentMethod represents payment types
type PaymentMethod string

const (
	PaymentDebitFromAccount PaymentMethod = "DEBIT_FROM_ACCOUNT"
	PaymentWire             PaymentMethod = "WIRE"
	PaymentCheck            PaymentMethod = "CHECK"
	PaymentACH              PaymentMethod = "ACH"
	PaymentCreditCard       PaymentMethod = "CREDIT_CARD"
)

// FeeSchedule represents a fee billing template
type FeeSchedule struct {
	ScheduleID   uuid.UUID `json:"schedule_id" db:"schedule_id"`
	ScheduleName string    `json:"schedule_name" db:"schedule_name"`
	Description  *string   `json:"description" db:"description"`
	FeeType      FeeType   `json:"fee_type" db:"fee_type"`

	// Tiered structure
	TierStructure json.RawMessage `json:"tier_structure" db:"tier_structure"` // Array of {min, max, rate}

	// Flat rate
	FlatAUMRate *float64 `json:"flat_aum_rate" db:"flat_aum_rate"`

	// Performance fees
	PerformanceHurdleRate *float64 `json:"performance_hurdle_rate" db:"performance_hurdle_rate"`
	PerformanceFeeRate    *float64 `json:"performance_fee_rate" db:"performance_fee_rate"`
	HighWaterMarkEnabled  bool     `json:"high_water_mark_enabled" db:"high_water_mark_enabled"`

	// Billing config
	BillingFrequency        BillingFrequency `json:"billing_frequency" db:"billing_frequency"`
	BillingAdvanceOrArrears BillingTiming    `json:"billing_advance_or_arrears" db:"billing_advance_or_arrears"`

	// Minimums
	MinimumQuarterlyFee *float64 `json:"minimum_quarterly_fee" db:"minimum_quarterly_fee"`
	MinimumAnnualFee    *float64 `json:"minimum_annual_fee" db:"minimum_annual_fee"`

	// Exclusions
	ExcludeCashFromAUM         bool `json:"exclude_cash_from_aum" db:"exclude_cash_from_aum"`
	ExcludeAlternativesFromAUM bool `json:"exclude_alternatives_from_aum" db:"exclude_alternatives_from_aum"`
	ExcludeHeldAwayFromAUM     bool `json:"exclude_held_away_from_aum" db:"exclude_held_away_from_aum"`

	// Preferences
	BillingDayOfMonth *int `json:"billing_day_of_month" db:"billing_day_of_month"`

	// Status
	IsActive   bool `json:"is_active" db:"is_active"`
	IsTemplate bool `json:"is_template" db:"is_template"`

	// Audit
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy *uuid.UUID `json:"created_by" db:"created_by"`
	UpdatedBy *uuid.UUID `json:"updated_by" db:"updated_by"`
}

// FeeTier represents a single tier in tiered AUM billing
type FeeTier struct {
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Rate float64 `json:"rate"` // As decimal, e.g., 0.01 = 1%
}

// ClientFeeAssignment assigns a fee schedule to a client
type ClientFeeAssignment struct {
	AssignmentID uuid.UUID  `json:"assignment_id" db:"assignment_id"`
	ClientID     uuid.UUID  `json:"client_id" db:"client_id"`
	AccountID    *uuid.UUID `json:"account_id" db:"account_id"`
	ScheduleID   uuid.UUID  `json:"schedule_id" db:"schedule_id"`

	// Date range
	EffectiveDate time.Time  `json:"effective_date" db:"effective_date"`
	EndDate       *time.Time `json:"end_date" db:"end_date"`

	// Custom overrides
	CustomDiscountPct       *float64        `json:"custom_discount_pct" db:"custom_discount_pct"`
	CustomMinimumFee        *float64        `json:"custom_minimum_fee" db:"custom_minimum_fee"`
	CustomTierStructure     json.RawMessage `json:"custom_tier_structure" db:"custom_tier_structure"`
	CustomPerformanceHurdle *float64        `json:"custom_performance_hurdle" db:"custom_performance_hurdle"`

	// Billing preferences
	InvoiceContactEmail *string        `json:"invoice_contact_email" db:"invoice_contact_email"`
	InvoiceContactName  *string        `json:"invoice_contact_name" db:"invoice_contact_name"`
	PaymentMethod       *PaymentMethod `json:"payment_method" db:"payment_method"`
	DebitAccountID      *uuid.UUID     `json:"debit_account_id" db:"debit_account_id"`
	BillingDayOfMonth   *int           `json:"billing_day_of_month" db:"billing_day_of_month"`

	IsActive bool `json:"is_active" db:"is_active"`

	// Audit
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy *uuid.UUID `json:"created_by" db:"created_by"`
	UpdatedBy *uuid.UUID `json:"updated_by" db:"updated_by"`
}

// FeeCalculation represents a single fee billing calculation
type FeeCalculation struct {
	CalculationID uuid.UUID  `json:"calculation_id" db:"calculation_id"`
	ClientID      uuid.UUID  `json:"client_id" db:"client_id"`
	AssignmentID  *uuid.UUID `json:"assignment_id" db:"assignment_id"`

	// Period
	BillingPeriodStart time.Time        `json:"billing_period_start" db:"billing_period_start"`
	BillingPeriodEnd   time.Time        `json:"billing_period_end" db:"billing_period_end"`
	BillingFrequency   BillingFrequency `json:"billing_frequency" db:"billing_frequency"`

	// AUM calculation
	AverageDailyBalance  float64 `json:"average_daily_balance" db:"average_daily_balance"`
	BeginningBalance     float64 `json:"beginning_balance" db:"beginning_balance"`
	EndingBalance        float64 `json:"ending_balance" db:"ending_balance"`
	AUMBasedFee          float64 `json:"aum_based_fee" db:"aum_based_fee"`
	AUMCalculationMethod string  `json:"aum_calculation_method" db:"aum_calculation_method"`

	// Performance calculation
	PortfolioReturnPct   *float64 `json:"portfolio_return_pct" db:"portfolio_return_pct"`
	HurdleReturnPct      *float64 `json:"hurdle_return_pct" db:"hurdle_return_pct"`
	ExcessReturn         *float64 `json:"excess_return" db:"excess_return"`
	PerformanceFee       float64  `json:"performance_fee" db:"performance_fee"`
	HighWaterMark        *float64 `json:"high_water_mark" db:"high_water_mark"`
	CurrentHighWaterMark *float64 `json:"current_high_water_mark" db:"current_high_water_mark"`

	// Other fees
	PlanningFee float64 `json:"planning_fee" db:"planning_fee"`
	HourlyFees  float64 `json:"hourly_fees" db:"hourly_fees"`
	OtherFees   float64 `json:"other_fees" db:"other_fees"`

	// Adjustments
	PriorPeriodAdjustment float64 `json:"prior_period_adjustment" db:"prior_period_adjustment"`
	DiscountAmount        float64 `json:"discount_amount" db:"discount_amount"`
	MinimumFeeAdjustment  float64 `json:"minimum_fee_adjustment" db:"minimum_fee_adjustment"`

	// Totals
	GrossFee      float64 `json:"gross_fee" db:"gross_fee"`
	NetFee        float64 `json:"net_fee" db:"net_fee"`
	TaxableAmount float64 `json:"taxable_amount" db:"taxable_amount"`

	// Workflow
	CalculationStatus    CalculationStatus `json:"calculation_status" db:"calculation_status"`
	RequiresManualReview bool              `json:"requires_manual_review" db:"requires_manual_review"`
	ReviewNotes          *string           `json:"review_notes" db:"review_notes"`

	// Approval
	ApprovedBy *uuid.UUID `json:"approved_by" db:"approved_by"`
	ApprovedAt *time.Time `json:"approved_at" db:"approved_at"`

	// Invoice
	InvoiceID     *uuid.UUID `json:"invoice_id" db:"invoice_id"`
	InvoiceNumber *string    `json:"invoice_number" db:"invoice_number"`
	InvoiceSentAt *time.Time `json:"invoice_sent_at" db:"invoice_sent_at"`

	// Payment
	PaymentReceivedAt *time.Time     `json:"payment_received_at" db:"payment_received_at"`
	PaymentAmount     *float64       `json:"payment_amount" db:"payment_amount"`
	PaymentMethod     *PaymentMethod `json:"payment_method" db:"payment_method"`

	// Audit
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy *uuid.UUID `json:"created_by" db:"created_by"`
	UpdatedBy *uuid.UUID `json:"updated_by" db:"updated_by"`
}

// HighWaterMark tracks performance fee high water marks
type HighWaterMark struct {
	HWMID                 uuid.UUID  `json:"hwm_id" db:"hwm_id"`
	ClientID              uuid.UUID  `json:"client_id" db:"client_id"`
	AccountID             *uuid.UUID `json:"account_id" db:"account_id"`
	CurrentHighWaterMark  float64    `json:"current_high_water_mark" db:"current_high_water_mark"`
	PreviousHighWaterMark *float64   `json:"previous_high_water_mark" db:"previous_high_water_mark"`
	HWMDate               time.Time  `json:"hwm_date" db:"hwm_date"`
	LastResetDate         *time.Time `json:"last_reset_date" db:"last_reset_date"`
	ResetReason           *string    `json:"reset_reason" db:"reset_reason"`
	CreatedAt             time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at" db:"updated_at"`
}

// RevenueRecognitionSchedule tracks when revenue should be recognized
type RevenueRecognitionSchedule struct {
	ScheduleID      uuid.UUID  `json:"schedule_id" db:"schedule_id"`
	CalculationID   uuid.UUID  `json:"calculation_id" db:"calculation_id"`
	RecognitionDate time.Time  `json:"recognition_date" db:"recognition_date"`
	Amount          float64    `json:"amount" db:"amount"`
	Recognized      bool       `json:"recognized" db:"recognized"`
	RecognizedAt    *time.Time `json:"recognized_at" db:"recognized_at"`
	JournalEntryID  *uuid.UUID `json:"journal_entry_id" db:"journal_entry_id"`
	LedgerAccount   *string    `json:"ledger_account" db:"ledger_account"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
}
