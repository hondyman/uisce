package taxplan

import (
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx/types"
)

type OpportunityType string

const (
	OpportunityTaxLossHarvest OpportunityType = "TAX_LOSS_HARVEST"
	OpportunityRothConversion OpportunityType = "ROTH_CONVERSION"
	OpportunityCharitable     OpportunityType = "CHARITABLE_DONATION"
	OpportunityAssetLocation  OpportunityType = "ASSET_LOCATION"
)

type TaxOpportunity struct {
	OpportunityID   uuid.UUID       `db:"opportunity_id" json:"opportunity_id"`
	ClientID        uuid.UUID       `db:"client_id" json:"client_id"`
	OpportunityType OpportunityType `db:"opportunity_type" json:"opportunity_type"`
	DetectedDate    time.Time       `db:"detected_date" json:"detected_date"`

	EstimatedSavings         float64 `db:"estimated_tax_savings" json:"estimated_tax_savings"`
	ImplementationComplexity string  `db:"implementation_complexity" json:"implementation_complexity"` // LOW, MEDIUM, HIGH
	TimeSensitivity          string  `db:"time_sensitivity" json:"time_sensitivity"`

	RecommendedActions types.JSONText `db:"recommended_actions" json:"recommended_actions,omitempty"`
	PositionsAffected  types.JSONText `db:"positions_affected" json:"positions_affected,omitempty"`
	BracketImpact      *float64       `db:"projected_bracket_impact" json:"projected_bracket_impact,omitempty"`

	Status         string  `db:"status" json:"status"` // IDENTIFIED, PRESENTED_TO_CLIENT, APPROVED, IMPLEMENTED, DECLINED
	AdvisorNotes   *string `db:"advisor_notes" json:"advisor_notes,omitempty"`
	ClientResponse *string `db:"client_response" json:"client_response,omitempty"`

	ImplementedDate *time.Time `db:"implemented_date" json:"implemented_date,omitempty"`
	ActualSavings   *float64   `db:"actual_tax_savings" json:"actual_tax_savings,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type TaxLot struct {
	LotID             uuid.UUID  `db:"lot_id" json:"lot_id"`
	ClientID          uuid.UUID  `db:"client_id" json:"client_id"`
	AccountID         *uuid.UUID `db:"account_id" json:"account_id,omitempty"`
	Ticker            string     `db:"ticker" json:"ticker"`
	PurchaseDate      time.Time  `db:"purchase_date" json:"purchase_date"`
	Quantity          float64    `db:"quantity" json:"quantity"`
	CostBasis         float64    `db:"cost_basis" json:"cost_basis"`
	CurrentValue      float64    `db:"current_value" json:"current_value"`
	UnrealizedGL      float64    `db:"unrealized_gain_loss" json:"unrealized_gain_loss"`
	HoldingPeriodDays int        `db:"holding_period_days" json:"holding_period_days"`
	IsLongTerm        bool       `db:"is_long_term" json:"is_long_term"`
	IsWashSale        bool       `db:"is_wash_sale" json:"is_wash_sale"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type ClientTaxProfile struct {
	ProfileID uuid.UUID `db:"profile_id" json:"profile_id"`
	ClientID  uuid.UUID `db:"client_id" json:"client_id"`

	CurrentYearIncome float64 `db:"current_year_income" json:"current_year_income"`
	EstimatedBracket  float64 `db:"estimated_tax_bracket" json:"estimated_tax_bracket"`
	FilingStatus      string  `db:"filing_status" json:"filing_status"`

	AvgAnnualIncome float64 `db:"average_annual_income" json:"average_annual_income"`
	AvgTaxBracket   float64 `db:"average_tax_bracket" json:"average_tax_bracket"`

	HasTraditionalIRA    bool `db:"has_traditional_ira" json:"has_traditional_ira"`
	HasRothIRA           bool `db:"has_roth_ira" json:"has_roth_ira"`
	CharitableIntent     bool `db:"charitable_intent" json:"charitable_intent"`
	EstatePlanningNeeded bool `db:"estate_planning_needed" json:"estate_planning_needed"`

	Age          int  `db:"age" json:"age"`
	RMDStartYear *int `db:"rmd_start_year" json:"rmd_start_year,omitempty"`

	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}
