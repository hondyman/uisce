package alternative_investment

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type AltInvSubtype string

const (
	SubtypePrivateEquity     AltInvSubtype = "private_equity"
	SubtypeVentureCapital    AltInvSubtype = "venture_capital"
	SubtypeRealEstate        AltInvSubtype = "real_estate"
	SubtypePrivateCredit     AltInvSubtype = "private_credit"
	SubtypeHedgeFund         AltInvSubtype = "hedge_fund"
	SubtypeInfrastructure    AltInvSubtype = "infrastructure"
)

var validAltInvSubtypes = map[AltInvSubtype]bool{
	SubtypePrivateEquity:  true,
	SubtypeVentureCapital: true,
	SubtypeRealEstate:    true,
	SubtypePrivateCredit: true,
	SubtypeHedgeFund:     true,
	SubtypeInfrastructure: true,
}

type AlternativeInvestmentRecord struct {
	ID              uuid.UUID  `db:"id" json:"id"`
	TenantID        uuid.UUID  `db:"tenant_id" json:"tenant_id"`
	InvestmentName  string     `db:"investment_name" json:"investment_name"`
	SponsorName     string     `db:"sponsor_name" json:"sponsor_name"`
	AssetClass      string     `db:"asset_class" json:"asset_class"`
	Status          string     `db:"status" json:"status"`
	SubtypeCode     string     `db:"subtype_code" json:"subtype_code"`

	VintageYear           *int             `db:"vintage_year" json:"vintage_year,omitempty"`
	CommittedCapital      *decimal.Decimal `db:"committed_capital" json:"committed_capital,omitempty"`
	CalledCapital         *decimal.Decimal `db:"called_capital" json:"called_capital,omitempty"`
	UnfundedCommitment    *decimal.Decimal `db:"unfunded_commitment" json:"unfunded_commitment,omitempty"`
	DPI                   *float64         `db:"dpi" json:"dpi,omitempty"`
	RVPI                  *float64         `db:"rvpi" json:"rvpi,omitempty"`
	RoundSeries           *string          `db:"round_series" json:"round_series,omitempty"`
	ProRataRightsFlag     *bool            `db:"pro_rata_rights_flag" json:"pro_rata_rights_flag,omitempty"`
	LeadInvestorName      *string          `db:"lead_investor_name" json:"lead_investor_name,omitempty"`
	PostMoneyValuation   *decimal.Decimal `db:"post_money_valuation" json:"post_money_valuation,omitempty"`
	PropertyType         *string          `db:"property_type" json:"property_type,omitempty"`
	OccupancyRatePct     *float64         `db:"occupancy_rate_pct" json:"occupancy_rate_pct,omitempty"`
	GrossAssetValue      *decimal.Decimal `db:"gross_asset_value" json:"gross_asset_value,omitempty"`
	LoanToValuePct       *float64         `db:"loan_to_value_pct" json:"loan_to_value_pct,omitempty"`
	SOFRSpreadBPS        *float64         `db:"sofr_spread_bps" json:"sofr_spread_bps,omitempty"`
	PIKInterestPct       *float64         `db:"pik_interest_pct" json:"pik_interest_pct,omitempty"`
	WarrantCoveragePct   *float64         `db:"warrant_coverage_pct" json:"warrant_coverage_pct,omitempty"`
	CovenantType         *string          `db:"covenant_type" json:"covenant_type,omitempty"`
	HurdleRatePct        *float64         `db:"hurdle_rate_pct" json:"hurdle_rate_pct,omitempty"`
	HighWaterMarkNAV     *decimal.Decimal `db:"high_water_mark_nav" json:"high_water_mark_nav,omitempty"`
	LockupPeriodMonths   *int             `db:"lockup_period_months" json:"lockup_period_months,omitempty"`
	RedemptionNoticeDays *int             `db:"redemption_notice_days" json:"redemption_notice_days,omitempty"`
	ProjectPhase         *string          `db:"project_phase" json:"project_phase,omitempty"`
	ConcessionExpiryYear *int             `db:"concession_expiry_year" json:"concession_expiry_year,omitempty"`
	ESGCarbonOffsetTons  *float64         `db:"esg_carbon_offset_tons" json:"esg_carbon_offset_tons,omitempty"`

	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	ValidFrom time.Time  `db:"valid_from" json:"valid_from"`
	ValidTo   *time.Time `db:"valid_to" json:"valid_to,omitempty"`
}

func (a AlternativeInvestmentRecord) Validate() error {
	if !validAltInvSubtypes[AltInvSubtype(a.SubtypeCode)] {
		return ErrInvalidSubtype
	}
	return nil
}
