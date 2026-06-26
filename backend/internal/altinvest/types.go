package altinvest

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
)

type InvestmentType string

const (
	TypePrivateEquity    InvestmentType = "PRIVATE_EQUITY"
	TypeVentureCapital   InvestmentType = "VENTURE_CAPITAL"
	TypeHedgeFund        InvestmentType = "HEDGE_FUND"
	TypeRealEstate       InvestmentType = "REAL_ESTATE"
	TypeDirectInvestment InvestmentType = "DIRECT_INVESTMENT"
)

type AlternativeInvestment struct {
	InvestmentID   uuid.UUID      `db:"investment_id" json:"investment_id"`
	ClientID       uuid.UUID      `db:"client_id" json:"client_id"`
	InvestmentType InvestmentType `db:"investment_type" json:"investment_type"`
	FundName       string         `db:"fund_name" json:"fund_name"`
	GeneralPartner *string        `db:"general_partner" json:"general_partner,omitempty"`
	VintageYear    *int           `db:"vintage_year" json:"vintage_year,omitempty"`

	// Capital
	TotalCommitment    float64 `db:"total_commitment_amount" json:"total_commitment_amount"`
	UnfundedCommitment float64 `db:"unfunded_commitment" json:"unfunded_commitment"`
	TotalCapitalCalled float64 `db:"total_capital_called" json:"total_capital_called"`
	TotalDistributions float64 `db:"total_distributions" json:"total_distributions"`

	// Valuations
	CurrentNAV      float64    `db:"current_nav" json:"current_nav"`
	NAVDate         *time.Time `db:"nav_date" json:"nav_date,omitempty"`
	ValuationSource *string    `db:"valuation_source" json:"valuation_source,omitempty"`

	// Performance
	IRR  *float64 `db:"irr_since_inception" json:"irr_since_inception,omitempty"`
	TVPI *float64 `db:"tvpi" json:"tvpi,omitempty"`
	DPI  *float64 `db:"dpi" json:"dpi,omitempty"`
	RVPI *float64 `db:"rvpi" json:"rvpi,omitempty"`
	MOIC *float64 `db:"moic" json:"moic,omitempty"`

	// Liquidity
	LockUpEndDate       *time.Time `db:"lock_up_end_date" json:"lock_up_end_date,omitempty"`
	RedemptionNotice    *int       `db:"redemption_notice_days" json:"redemption_notice_days,omitempty"`
	RedemptionFrequency *string    `db:"redemption_frequency" json:"redemption_frequency,omitempty"`

	Metadata types.JSONText `db:"metadata" json:"metadata,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type CapitalCall struct {
	CallID               uuid.UUID  `db:"call_id" json:"call_id"`
	InvestmentID         uuid.UUID  `db:"investment_id" json:"investment_id"`
	NoticeDate           time.Time  `db:"notice_date" json:"notice_date"`
	DueDate              time.Time  `db:"due_date" json:"due_date"`
	AmountRequested      float64    `db:"amount_requested" json:"amount_requested"`
	AmountFunded         float64    `db:"amount_funded" json:"amount_funded"`
	Status               string     `db:"status" json:"status"` // PENDING, FUNDED, OVERDUE
	FundingSourceAccount *uuid.UUID `db:"funding_source_account" json:"funding_source_account,omitempty"`
	LiquidityCheckPassed *bool      `db:"liquidity_check_passed" json:"liquidity_check_passed,omitempty"`
	AlertSentAt          *time.Time `db:"alert_sent_at" json:"alert_sent_at,omitempty"`
	CreatedAt            time.Time  `db:"created_at" json:"created_at"`
}

type Distribution struct {
	DistributionID   uuid.UUID `db:"distribution_id" json:"distribution_id"`
	InvestmentID     uuid.UUID `db:"investment_id" json:"investment_id"`
	DistributionDate time.Time `db:"distribution_date" json:"distribution_date"`
	Amount           float64   `db:"amount" json:"amount"`
	DistributionType string    `db:"distribution_type" json:"distribution_type"` // INCOME, RETURN_OF_CAPITAL, CAPITAL_GAIN
	Reinvested       bool      `db:"reinvested" json:"reinvested"`
	CreatedAt        time.Time `db:"created_at" json:"created_at"`
}
