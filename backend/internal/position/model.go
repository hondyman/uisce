package position

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ─────────────────────────────────────────────────────────────────────────────
// Position Master (Root Holdings)
// ─────────────────────────────────────────────────────────────────────────────

// PositionMasterRecord is the gold-copy holding of a security in a portfolio.
// Uses decimal.Decimal for all financial amounts to avoid floating-point rounding.
type PositionMasterRecord struct {
	ID                 uuid.UUID        `db:"id"                   json:"id"`
	TenantID           uuid.UUID        `db:"tenant_id"            json:"tenant_id"`
	CoreID             *uuid.UUID       `db:"core_id"              json:"core_id,omitempty"`
	PortfolioID        uuid.UUID        `db:"portfolio_id"         json:"portfolio_id"`
	SecurityID         uuid.UUID        `db:"security_id"          json:"security_id"`
	PositionDate       time.Time        `db:"position_date"        json:"position_date"`
	PositionQuantity   decimal.Decimal  `db:"position_quantity"    json:"position_quantity"`
	PositionSide       string           `db:"position_side"        json:"position_side"`
	PositionCurrency   string           `db:"position_currency"    json:"position_currency"`
	PriceID            *uuid.UUID       `db:"price_id"             json:"price_id,omitempty"`
	MarketValueLocal   *decimal.Decimal `db:"market_value_local"   json:"market_value_local,omitempty"`
	MarketValueBase    *decimal.Decimal `db:"market_value_base"    json:"market_value_base,omitempty"`
	ValuationFXRate    *decimal.Decimal `db:"valuation_fx_rate"    json:"valuation_fx_rate,omitempty"`
	CostBasisLocal     *decimal.Decimal `db:"cost_basis_local"     json:"cost_basis_local,omitempty"`
	UnrealizedPLLocal  *decimal.Decimal `db:"unrealized_pl_local"  json:"unrealized_pl_local,omitempty"`
	UnrealizedPLPct    *decimal.Decimal `db:"unrealized_pl_pct"    json:"unrealized_pl_pct,omitempty"`
	PositionWeightPct  *decimal.Decimal `db:"position_weight_pct"  json:"position_weight_pct,omitempty"`
	PositionSource     string           `db:"position_source"      json:"position_source"`
	PositionConfidence int              `db:"position_confidence"  json:"position_confidence"`
	IsReconciled       bool             `db:"is_reconciled"        json:"is_reconciled"`
	ReconciliationDiff *decimal.Decimal `db:"reconciliation_diff"  json:"reconciliation_diff,omitempty"`
	SourceSystems      json.RawMessage  `db:"source_systems"       json:"source_systems"`
	CreatedAt          time.Time        `db:"created_at"           json:"created_at"`
	UpdatedAt          time.Time        `db:"updated_at"           json:"updated_at"`
	ValidFrom          time.Time        `db:"valid_from"           json:"valid_from"`
	ValidTo            *time.Time       `db:"valid_to"             json:"valid_to,omitempty"`
}

// RawPositionRecord is an incoming record from a custodian or accounting system.
// All numeric fields are strings to allow safe parsing and survivorship comparison.
type RawPositionRecord struct {
	PortfolioID      string `json:"portfolio_id"`
	SecurityID       string `json:"security_id"`
	PositionDate     string `json:"position_date"` // YYYY-MM-DD
	Quantity         string `json:"position_quantity"`
	Side             string `json:"position_side,omitempty"` // Long, Short
	Currency         string `json:"position_currency"`
	MarketValueLocal string `json:"market_value_local,omitempty"`
	MarketValueBase  string `json:"market_value_base,omitempty"`
	ValuationFXRate  string `json:"valuation_fx_rate,omitempty"`
	CostBasisLocal   string `json:"cost_basis_local,omitempty"`
	Source           string `json:"position_source"`
	PriceID          string `json:"price_id,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Position Lot (Tax Lot)
// ─────────────────────────────────────────────────────────────────────────────

// PositionLotRecord is a single tax lot linked to a position.
type PositionLotRecord struct {
	ID              uuid.UUID        `db:"id"               json:"id"`
	TenantID        uuid.UUID        `db:"tenant_id"        json:"tenant_id"`
	PositionID      uuid.UUID        `db:"position_id"      json:"position_id"`
	LotReference    *string          `db:"lot_reference"    json:"lot_reference,omitempty"`
	AcquisitionDate time.Time        `db:"acquisition_date" json:"acquisition_date"`
	SettlementDate  *time.Time       `db:"settlement_date"  json:"settlement_date,omitempty"`
	LotQuantity     decimal.Decimal  `db:"lot_quantity"     json:"lot_quantity"`
	CostPerUnit     decimal.Decimal  `db:"cost_per_unit"    json:"cost_per_unit"`
	TotalCostBasis  decimal.Decimal  `db:"total_cost_basis" json:"total_cost_basis"`
	LotMethod       string           `db:"lot_method"       json:"lot_method"`
	IsClosed        bool             `db:"is_closed"        json:"is_closed"`
	ClosedDate      *time.Time       `db:"closed_date"      json:"closed_date,omitempty"`
	RealizedPL      *decimal.Decimal `db:"realized_pl"      json:"realized_pl,omitempty"`
	CreatedAt       time.Time        `db:"created_at"       json:"created_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Cash Position
// ─────────────────────────────────────────────────────────────────────────────

// CashPositionRecord is a cash balance for a portfolio in one currency.
type CashPositionRecord struct {
	ID                 uuid.UUID        `db:"id"                    json:"id"`
	TenantID           uuid.UUID        `db:"tenant_id"             json:"tenant_id"`
	PortfolioID        uuid.UUID        `db:"portfolio_id"          json:"portfolio_id"`
	CashCurrency       string           `db:"cash_currency"         json:"cash_currency"`
	AccountID          *uuid.UUID       `db:"account_id"            json:"account_id,omitempty"`
	ValueDate          time.Time        `db:"value_date"            json:"value_date"`
	BalanceAmount      decimal.Decimal  `db:"balance_amount"        json:"balance_amount"`
	AvailableBalance   *decimal.Decimal `db:"available_balance"     json:"available_balance,omitempty"`
	PendingSettlements json.RawMessage  `db:"pending_settlements"   json:"pending_settlements"`
	InterestAccrued    decimal.Decimal  `db:"interest_accrued"      json:"interest_accrued"`
	CashSource         string           `db:"cash_source"           json:"cash_source"`
	SourceSystems      json.RawMessage  `db:"source_systems"        json:"source_systems"`
	CreatedAt          time.Time        `db:"created_at"            json:"created_at"`
	UpdatedAt          time.Time        `db:"updated_at"            json:"updated_at"`
	ValidFrom          time.Time        `db:"valid_from"            json:"valid_from"`
	ValidTo            *time.Time       `db:"valid_to"              json:"valid_to,omitempty"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Position Snapshot (Append-only time-series)
// ─────────────────────────────────────────────────────────────────────────────

// PositionSnapshotRecord is an immutable, append-only historical snapshot.
type PositionSnapshotRecord struct {
	ID                   uuid.UUID        `db:"id"                     json:"id"`
	TenantID             uuid.UUID        `db:"tenant_id"              json:"tenant_id"`
	PositionID           uuid.UUID        `db:"position_id"            json:"position_id"`
	SnapshotDate         time.Time        `db:"snapshot_date"          json:"snapshot_date"`
	SnapshotQuantity     *decimal.Decimal `db:"snapshot_quantity"      json:"snapshot_quantity,omitempty"`
	SnapshotMarketValue  *decimal.Decimal `db:"snapshot_market_value"  json:"snapshot_market_value,omitempty"`
	SnapshotPriceUsed    *decimal.Decimal `db:"snapshot_price_used"    json:"snapshot_price_used,omitempty"`
	SnapshotFXRate       *decimal.Decimal `db:"snapshot_fx_rate"       json:"snapshot_fx_rate,omitempty"`
	PortfolioComposition json.RawMessage  `db:"portfolio_composition"  json:"portfolio_composition,omitempty"`
	SnapshotSource       string           `db:"snapshot_source"        json:"snapshot_source"`
	CreatedAt            time.Time        `db:"created_at"             json:"created_at"`
}

// ─────────────────────────────────────────────────────────────────────────────
// Survivorship / Run Result
// ─────────────────────────────────────────────────────────────────────────────

// FieldSurvivorshipDecision records which source won for one field during a run.
type FieldSurvivorshipDecision struct {
	FieldName        string   `json:"field_name"`
	SelectedSource   string   `json:"selected_source"`
	SelectedValue    string   `json:"selected_value"`
	AlternateSources []string `json:"alternate_sources,omitempty"`
	SelectionRule    string   `json:"selection_rule"`
}

// PositionGoldCopyRunResult summarises a completed position survivorship run.
type PositionGoldCopyRunResult struct {
	RunID              uuid.UUID                   `json:"run_id"`
	TenantID           uuid.UUID                   `json:"tenant_id"`
	PositionsProcessed int                         `json:"positions_processed"`
	CashProcessed      int                         `json:"cash_processed"`
	SnapshotsCreated   int                         `json:"snapshots_created"`
	DQPassCount        int                         `json:"dq_pass_count"`
	DQFailCount        int                         `json:"dq_fail_count"`
	DQFailureDetails   []string                    `json:"dq_failure_details"`
	UnreconciledCount  int                         `json:"unreconciled_count"`
	FieldDecisions     []FieldSurvivorshipDecision `json:"field_decisions,omitempty"`
	StartedAt          time.Time                   `json:"started_at"`
	CompletedAt        time.Time                   `json:"completed_at"`
}
