package position

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type PositionSubtype string

const (
	SubtypeSettledLong        PositionSubtype = "settled_long"
	SubtypeShortBorrowed      PositionSubtype = "short_borrowed"
	SubtypeDerivativeExposure PositionSubtype = "derivative_exposure"
	SubtypePledgedCollateral  PositionSubtype = "pledged_collateral"
	SubtypeUnsettledPipeline  PositionSubtype = "unsettled_pipeline"
)

var validPositionSubtypes = map[PositionSubtype]bool{
	SubtypeSettledLong:        true,
	SubtypeShortBorrowed:      true,
	SubtypeDerivativeExposure: true,
	SubtypePledgedCollateral:  true,
	SubtypeUnsettledPipeline:  true,
}

type PositionRecord struct {
	ID         uuid.UUID       `db:"id" json:"id"`
	TenantID   uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	AccountID  uuid.UUID       `db:"account_id" json:"account_id"`
	SecurityID uuid.UUID       `db:"security_id" json:"security_id"`
	Quantity   decimal.Decimal `db:"quantity" json:"quantity"`
	MarketValue decimal.Decimal `db:"market_value" json:"market_value"`
	Currency   string         `db:"currency" json:"currency"`
	SubtypeCode string         `db:"subtype_code" json:"subtype_code"`

	CustodyAccountID    *uuid.UUID       `db:"custody_account_id" json:"custody_account_id,omitempty"`
	SettledShares       *decimal.Decimal `db:"settled_shares" json:"settled_shares,omitempty"`
	CostBasisMethod     *string          `db:"cost_basis_method" json:"cost_basis_method,omitempty"`
	HeldToMaturityFlag *bool            `db:"held_to_maturity_flag" json:"held_to_maturity_flag,omitempty"`
	PrimeBrokerID      *uuid.UUID       `db:"prime_broker_id" json:"prime_broker_id,omitempty"`
	BorrowRateBPS      *float64         `db:"borrow_rate_bps" json:"borrow_rate_bps,omitempty"`
	LocateID            *string          `db:"locate_id" json:"locate_id,omitempty"`
	HardToBorrowFlag   *bool            `db:"hard_to_borrow_flag" json:"hard_to_borrow_flag,omitempty"`
	UnderlyingSecurityID *uuid.UUID     `db:"underlying_security_id" json:"underlying_security_id,omitempty"`
	NotionalAmount     *decimal.Decimal `db:"notional_amount" json:"notional_amount,omitempty"`
	UnrealizedPnL      *decimal.Decimal `db:"unrealized_pnl" json:"unrealized_pnl,omitempty"`
	ExpirationDate      *time.Time      `db:"expiration_date" json:"expiration_date,omitempty"`
	PledgedToParty      *string          `db:"pledged_to_party" json:"pledged_to_party,omitempty"`
	HaircutPct         *float64         `db:"haircut_pct" json:"haircut_pct,omitempty"`
	RehypothecationAllowed *bool        `db:"rehypothecation_allowed_flag" json:"rehypothecation_allowed_flag,omitempty"`
	TradeDateShares    *decimal.Decimal `db:"trade_date_shares" json:"trade_date_shares,omitempty"`
	PendingSettlementCash *decimal.Decimal `db:"pending_settlement_cash" json:"pending_settlement_cash,omitempty"`
	FailsToDeliverFlag *bool            `db:"fails_to_deliver_flag" json:"fails_to_deliver_flag,omitempty"`

	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	ValidFrom time.Time  `db:"valid_from" json:"valid_from"`
	ValidTo   *time.Time `db:"valid_to" json:"valid_to,omitempty"`
}

func (p PositionRecord) Validate() error {
	if !validPositionSubtypes[PositionSubtype(p.SubtypeCode)] {
		return ErrInvalidSubtype
	}
	if p.SubtypeCode == string(SubtypeShortBorrowed) && p.PrimeBrokerID == nil {
		return ErrRequiresPrimeBroker
	}
	return nil
}
