package trade_order

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type TradeOrderSubtype string

const (
	SubtypeBlockParent   TradeOrderSubtype = "block_parent"
	SubtypeDMAExecution  TradeOrderSubtype = "dma_execution"
	SubtypeOTCBilateral  TradeOrderSubtype = "otc_bilateral"
	SubtypeFXSpotForward TradeOrderSubtype = "fx_spot_forward"
	SubtypePrimaryAuction TradeOrderSubtype = "primary_auction"
)

var validTradeOrderSubtypes = map[TradeOrderSubtype]bool{
	SubtypeBlockParent:   true,
	SubtypeDMAExecution:  true,
	SubtypeOTCBilateral:  true,
	SubtypeFXSpotForward: true,
	SubtypePrimaryAuction: true,
}

type TradeOrderRecord struct {
	ID             uuid.UUID        `db:"id" json:"id"`
	TenantID       uuid.UUID        `db:"tenant_id" json:"tenant_id"`
	AccountID      uuid.UUID        `db:"account_id" json:"account_id"`
	SecurityID     uuid.UUID        `db:"security_id" json:"security_id"`
	OrderSide      string           `db:"order_side" json:"order_side"`
	OrderedQuantity decimal.Decimal  `db:"ordered_quantity" json:"ordered_quantity"`
	ExecutionPrice *decimal.Decimal `db:"execution_price" json:"execution_price,omitempty"`
	OrderStatus    string           `db:"order_status" json:"order_status"`
	SubtypeCode    string           `db:"subtype_code" json:"subtype_code"`

	AllocationProfileID     *uuid.UUID       `db:"allocation_profile_id" json:"allocation_profile_id,omitempty"`
	TotalRequestedQuantity *decimal.Decimal `db:"total_requested_quantity" json:"total_requested_quantity,omitempty"`
	AveragePrice           *decimal.Decimal `db:"average_price" json:"average_price,omitempty"`
	ExecutionAlgoID        *string          `db:"execution_algo_id" json:"execution_algo_id,omitempty"`
	VenueID                *string          `db:"venue_id" json:"venue_id,omitempty"`
	LiquidityFlag          *string          `db:"liquidity_flag" json:"liquidity_flag,omitempty"`
	RouteTimeMicros        *int64           `db:"route_time_micros" json:"route_time_micros,omitempty"`
	CounterpartyDealerID   *uuid.UUID       `db:"counterparty_dealer_id" json:"counterparty_dealer_id,omitempty"`
	ConfirmationStatus     *string          `db:"confirmation_status" json:"confirmation_status,omitempty"`
	ISDAScheduleVersion    *string          `db:"isda_schedule_version" json:"isda_schedule_version,omitempty"`
	BaseCurrency           *string          `db:"base_currency" json:"base_currency,omitempty"`
	QuoteCurrency          *string          `db:"quote_currency" json:"quote_currency,omitempty"`
	FXRate                 *float64         `db:"fx_rate" json:"fx_rate,omitempty"`
	ValueDate              *time.Time       `db:"value_date" json:"value_date,omitempty"`
	SyndicateManagerID     *uuid.UUID       `db:"syndicate_manager_id" json:"syndicate_manager_id,omitempty"`
	ConcessionAmount       *float64         `db:"concession_amount" json:"concession_amount,omitempty"`
	AllotmentShares        *decimal.Decimal `db:"allotment_shares" json:"allotment_shares,omitempty"`

	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt time.Time  `db:"updated_at" json:"updated_at"`
	ValidFrom time.Time  `db:"valid_from" json:"valid_from"`
	ValidTo   *time.Time `db:"valid_to" json:"valid_to,omitempty"`
}

func (t TradeOrderRecord) Validate() error {
	if !validTradeOrderSubtypes[TradeOrderSubtype(t.SubtypeCode)] {
		return ErrInvalidSubtype
	}
	return nil
}
