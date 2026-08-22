package alternative_investment

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type AlternativeInvestmentRecord struct {
	ID uuid.UUID `db:"id"`

	TenantID uuid.UUID `db:"tenant_id"`

	InvestmentID   uuid.UUID `db:"investment_id"`
	ClientID       uuid.UUID `db:"client_id"`
	InvestmentType string    `db:"investment_type"`
	FundName       string    `db:"fund_name"`
	GeneralPartner *string   `db:"general_partner"`
	VintageYear    *int      `db:"vintage_year"`

	TotalCommitmentAmount float64 `db:"total_commitment_amount"`
	UnfundedCommitment    float64 `db:"unfunded_commitment"`
	TotalCapitalCalled    float64 `db:"total_capital_called"`
	TotalDistributions    float64 `db:"total_distributions"`

	CurrentNAV      *float64    `db:"current_nav"`
	NAVDate         *time.Time  `db:"nav_date"`
	ValuationSource *string     `db:"valuation_source"`

	IRRSinceInception *float64 `db:"irr_since_inception"`
	TVPI              *float64 `db:"tvpi"`
	DPI               *float64 `db:"dpi"`
	RVPI              *float64 `db:"rvpi"`
	MOIC              *float64 `db:"moic"`

	LockUpEndDate        *time.Time `db:"lock_up_end_date"`
	RedemptionNoticeDays *int       `db:"redemption_notice_days"`
	RedemptionFrequency  *string    `db:"redemption_frequency"`

	LastCapitalCallDate  *time.Time `db:"last_capital_call_date"`
	LastDistributionDate *time.Time `db:"last_distribution_date"`
	LastK1ReceivedDate   *time.Time `db:"last_k1_received_date"`

	SubtypeCode string `db:"subtype_code"`

	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	ValidFrom time.Time  `db:"valid_from"`
	ValidTo   *time.Time `db:"valid_to"`
}

var validSubtypeCodes = map[string]bool{
	"PRIVATE_EQUITY":    true,
	"VENTURE_CAPITAL":   true,
	"HEDGE_FUND":        true,
	"REAL_ESTATE":       true,
	"DIRECT_INVESTMENT": true,
	"INFRASTRUCTURE":    true,
	"PRIVATE_DEBT":      true,
}

func (r *AlternativeInvestmentRecord) Validate() error {
	if !validSubtypeCodes[r.SubtypeCode] {
		return fmt.Errorf("%w: %s", ErrInvalidSubtype, r.SubtypeCode)
	}
	return nil
}