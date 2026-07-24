package altinv

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// InvestmentType represents the type of alternative investment
type InvestmentType string

const (
	PrivateEquity    InvestmentType = "PRIVATE_EQUITY"
	VentureCapital   InvestmentType = "VENTURE_CAPITAL"
	HedgeFund        InvestmentType = "HEDGE_FUND"
	RealEstate       InvestmentType = "REAL_ESTATE"
	DirectInvestment InvestmentType = "DIRECT_INVESTMENT"
	Infrastructure   InvestmentType = "INFRASTRUCTURE"
	PrivateDebt      InvestmentType = "PRIVATE_DEBT"
)

// ValuationSource represents who reported the NAV
type ValuationSource string

const (
	GPReported       ValuationSource = "GP_REPORTED"
	ThirdParty       ValuationSource = "THIRD_PARTY"
	InternalEstimate ValuationSource = "INTERNAL_ESTIMATE"
)

// RedemptionFrequency represents how often redemptions are allowed
type RedemptionFrequency string

const (
	Quarterly    RedemptionFrequency = "QUARTERLY"
	Annual       RedemptionFrequency = "ANNUAL"
	ClosedEnd    RedemptionFrequency = "CLOSED_END"
	NoRedemption RedemptionFrequency = "NONE"
)

// AlternativeInvestment represents an alternative investment holding
type AlternativeInvestment struct {
	InvestmentID   uuid.UUID      `json:"investment_id" db:"investment_id"`
	ClientID       uuid.UUID      `json:"client_id" db:"client_id"`
	InvestmentType InvestmentType `json:"investment_type" db:"investment_type"`
	FundName       string         `json:"fund_name" db:"fund_name"`
	GeneralPartner *string        `json:"general_partner" db:"general_partner"`
	VintageYear    *int           `json:"vintage_year" db:"vintage_year"`

	// Capital commitments and cash flows
	TotalCommitmentAmount float64 `json:"total_commitment_amount" db:"total_commitment_amount"`
	UnfundedCommitment    float64 `json:"unfunded_commitment" db:"unfunded_commitment"`
	TotalCapitalCalled    float64 `json:"total_capital_called" db:"total_capital_called"`
	TotalDistributions    float64 `json:"total_distributions" db:"total_distributions"`

	// Valuations
	CurrentNAV      *float64         `json:"current_nav" db:"current_nav"`
	NAVDate         *time.Time       `json:"nav_date" db:"nav_date"`
	ValuationSource *ValuationSource `json:"valuation_source" db:"valuation_source"`

	// Performance metrics
	IRRSinceInception *float64 `json:"irr_since_inception" db:"irr_since_inception"`
	TVPI              *float64 `json:"tvpi" db:"tvpi"` // Total Value to Paid-In
	DPI               *float64 `json:"dpi" db:"dpi"`   // Distributions to Paid-In
	RVPI              *float64 `json:"rvpi" db:"rvpi"` // Residual Value to Paid-In
	MOIC              *float64 `json:"moic" db:"moic"` // Multiple on Invested Capital

	// Liquidity constraints
	LockUpEndDate        *time.Time           `json:"lock_up_end_date" db:"lock_up_end_date"`
	RedemptionNoticeDays *int                 `json:"redemption_notice_days" db:"redemption_notice_days"`
	RedemptionFrequency  *RedemptionFrequency `json:"redemption_frequency" db:"redemption_frequency"`

	// Document tracking
	LastCapitalCallDate  *time.Time `json:"last_capital_call_date" db:"last_capital_call_date"`
	LastDistributionDate *time.Time `json:"last_distribution_date" db:"last_distribution_date"`
	LastK1ReceivedDate   *time.Time `json:"last_k1_received_date" db:"last_k1_received_date"`

	// Metadata
	Metadata json.RawMessage `json:"metadata" db:"metadata"`

	// Audit fields
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy *uuid.UUID `json:"created_by" db:"created_by"`
	UpdatedBy *uuid.UUID `json:"updated_by" db:"updated_by"`
}

// CapitalCallStatus represents the status of a capital call
type CapitalCallStatus string

const (
	StatusPending         CapitalCallStatus = "PENDING"
	StatusFunded          CapitalCallStatus = "FUNDED"
	StatusPartiallyFunded CapitalCallStatus = "PARTIALLY_FUNDED"
	StatusOverdue         CapitalCallStatus = "OVERDUE"
	StatusCancelled       CapitalCallStatus = "CANCELLED"
)

// CapitalCall represents a capital call notice
type CapitalCall struct {
	CallID               uuid.UUID         `json:"call_id" db:"call_id"`
	InvestmentID         uuid.UUID         `json:"investment_id" db:"investment_id"`
	NoticeDate           time.Time         `json:"notice_date" db:"notice_date"`
	DueDate              time.Time         `json:"due_date" db:"due_date"`
	AmountRequested      float64           `json:"amount_requested" db:"amount_requested"`
	AmountFunded         float64           `json:"amount_funded" db:"amount_funded"`
	Status               CapitalCallStatus `json:"status" db:"status"`
	FundingSourceAccount *uuid.UUID        `json:"funding_source_account" db:"funding_source_account"`

	// Liquidity validation
	LiquidityCheckPassed    *bool      `json:"liquidity_check_passed" db:"liquidity_check_passed"`
	LiquidityCheckDate      *time.Time `json:"liquidity_check_date" db:"liquidity_check_date"`
	LiquidityShortageAmount *float64   `json:"liquidity_shortage_amount" db:"liquidity_shortage_amount"`

	// Notifications
	AlertSentAt    *time.Time `json:"alert_sent_at" db:"alert_sent_at"`
	ReminderSentAt *time.Time `json:"reminder_sent_at" db:"reminder_sent_at"`

	// Notes
	AdvisorNotes *string `json:"advisor_notes" db:"advisor_notes"`

	// Audit fields
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy *uuid.UUID `json:"created_by" db:"created_by"`
	UpdatedBy *uuid.UUID `json:"updated_by" db:"updated_by"`
}

// DistributionType represents the type of distribution
type DistributionType string

const (
	DistIncome          DistributionType = "INCOME"
	DistReturnOfCapital DistributionType = "RETURN_OF_CAPITAL"
	DistCapitalGain     DistributionType = "CAPITAL_GAIN"
	DistRecallable      DistributionType = "RECALLABLE"
)

// Distribution represents a distribution from an alternative investment
type Distribution struct {
	DistributionID   uuid.UUID        `json:"distribution_id" db:"distribution_id"`
	InvestmentID     uuid.UUID        `json:"investment_id" db:"investment_id"`
	DistributionDate time.Time        `json:"distribution_date" db:"distribution_date"`
	Amount           float64          `json:"amount" db:"amount"`
	DistributionType DistributionType `json:"distribution_type" db:"distribution_type"`

	// Reinvestment
	Reinvested          bool       `json:"reinvested" db:"reinvested"`
	ReinvestmentDate    *time.Time `json:"reinvestment_date" db:"reinvestment_date"`
	ReinvestmentAccount *uuid.UUID `json:"reinvestment_account" db:"reinvestment_account"`

	// Tax implications
	TaxYear       *int     `json:"tax_year" db:"tax_year"`
	TaxableAmount *float64 `json:"taxable_amount" db:"taxable_amount"`

	// Notes
	AdvisorNotes *string `json:"advisor_notes" db:"advisor_notes"`

	// Audit fields
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	CreatedBy *uuid.UUID `json:"created_by" db:"created_by"`
	UpdatedBy *uuid.UUID `json:"updated_by" db:"updated_by"`
}

// DocumentType represents the type of alternative investment document
type DocumentType string

const (
	DocK1                    DocumentType = "K1"
	DocCapitalStatement      DocumentType = "CAPITAL_STATEMENT"
	DocQuarterlyReport       DocumentType = "QUARTERLY_REPORT"
	DocAnnualReport          DocumentType = "ANNUAL_REPORT"
	DocSubscriptionAgreement DocumentType = "SUBSCRIPTION_AGREEMENT"
	DocOperatingAgreement    DocumentType = "OPERATING_AGREEMENT"
	DocSideLetter            DocumentType = "SIDE_LETTER"
	DocOther                 DocumentType = "OTHER"
)

// ExtractionStatus represents the status of AI document extraction
type ExtractionStatus string

const (
	ExtractPending              ExtractionStatus = "PENDING"
	ExtractInProgress           ExtractionStatus = "IN_PROGRESS"
	ExtractCompleted            ExtractionStatus = "COMPLETED"
	ExtractFailed               ExtractionStatus = "FAILED"
	ExtractManualReviewRequired ExtractionStatus = "MANUAL_REVIEW_REQUIRED"
)

// AltInvestmentDocument represents a document related to an alternative investment
type AltInvestmentDocument struct {
	DocumentID   uuid.UUID    `json:"document_id" db:"document_id"`
	InvestmentID uuid.UUID    `json:"investment_id" db:"investment_id"`
	DocumentType DocumentType `json:"document_type" db:"document_type"`
	DocumentDate *time.Time   `json:"document_date" db:"document_date"`
	TaxYear      *int         `json:"tax_year" db:"tax_year"`

	// File storage
	FileURL       string  `json:"file_url" db:"file_url"`
	FileName      *string `json:"file_name" db:"file_name"`
	FileSizeBytes *int    `json:"file_size_bytes" db:"file_size_bytes"`
	MimeType      *string `json:"mime_type" db:"mime_type"`

	// AI extraction
	ExtractedData        json.RawMessage   `json:"extracted_data" db:"extracted_data"`
	ExtractionStatus     *ExtractionStatus `json:"extraction_status" db:"extraction_status"`
	ExtractionConfidence *float64          `json:"extraction_confidence" db:"extraction_confidence"`

	// Processing
	ProcessedAt *time.Time `json:"processed_at" db:"processed_at"`
	ProcessedBy *string    `json:"processed_by" db:"processed_by"`

	// Audit fields
	UploadedAt time.Time  `json:"uploaded_at" db:"uploaded_at"`
	UploadedBy *uuid.UUID `json:"uploaded_by" db:"uploaded_by"`
}

// ExtractedQuarterlyData represents data extracted from a GP quarterly statement
type ExtractedQuarterlyData struct {
	NAV                *float64   `json:"nav"`
	NAVDate            *time.Time `json:"nav_date"`
	CapitalCalled      *float64   `json:"capital_called"`
	Distributions      *float64   `json:"distributions"`
	IRR                *float64   `json:"irr"`
	TVPI               *float64   `json:"tvpi"`
	UnfundedCommitment *float64   `json:"unfunded_commitment"`
}

// InvestmentPerformance represents performance summary for an investment
type InvestmentPerformance struct {
	InvestmentID          uuid.UUID      `json:"investment_id" db:"investment_id"`
	ClientID              uuid.UUID      `json:"client_id" db:"client_id"`
	FundName              string         `json:"fund_name" db:"fund_name"`
	InvestmentType        InvestmentType `json:"investment_type" db:"investment_type"`
	VintageYear           *int           `json:"vintage_year" db:"vintage_year"`
	TotalCommitmentAmount float64        `json:"total_commitment_amount" db:"total_commitment_amount"`
	UnfundedCommitment    float64        `json:"unfunded_commitment" db:"unfunded_commitment"`
	TotalCapitalCalled    float64        `json:"total_capital_called" db:"total_capital_called"`
	TotalDistributions    float64        `json:"total_distributions" db:"total_distributions"`
	CurrentNAV            *float64       `json:"current_nav" db:"current_nav"`
	NAVDate               *time.Time     `json:"nav_date" db:"nav_date"`

	// Performance metrics
	IRRSinceInception *float64 `json:"irr_since_inception" db:"irr_since_inception"`
	TVPI              *float64 `json:"tvpi" db:"tvpi"`
	DPI               *float64 `json:"dpi" db:"dpi"`
	RVPI              *float64 `json:"rvpi" db:"rvpi"`
	MOIC              *float64 `json:"moic" db:"moic"`

	// Calculated fields
	NetCashFlow        float64  `json:"net_cash_flow" db:"net_cash_flow"`
	TotalValueMultiple *float64 `json:"total_value_multiple" db:"total_value_multiple"`
	PctUnfunded        float64  `json:"pct_unfunded" db:"pct_unfunded"`
}

// UpcomingCapitalCall represents upcoming capital call information
type UpcomingCapitalCall struct {
	CallID               uuid.UUID         `json:"call_id" db:"call_id"`
	InvestmentID         uuid.UUID         `json:"investment_id" db:"investment_id"`
	ClientID             uuid.UUID         `json:"client_id" db:"client_id"`
	FundName             string            `json:"fund_name" db:"fund_name"`
	NoticeDate           time.Time         `json:"notice_date" db:"notice_date"`
	DueDate              time.Time         `json:"due_date" db:"due_date"`
	AmountRequested      float64           `json:"amount_requested" db:"amount_requested"`
	AmountFunded         float64           `json:"amount_funded" db:"amount_funded"`
	Status               CapitalCallStatus `json:"status" db:"status"`
	LiquidityCheckPassed *bool             `json:"liquidity_check_passed" db:"liquidity_check_passed"`
	FundingSourceAccount *uuid.UUID        `json:"funding_source_account" db:"funding_source_account"`
	DaysUntilDue         int               `json:"days_until_due" db:"days_until_due"`
}
