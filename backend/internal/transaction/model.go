package transaction

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TransactionMasterRecord aligns with edm.transaction_master
type TransactionMasterRecord struct {
	TransactionID       uuid.UUID        `json:"transaction_id" db:"transaction_id"`
	PortfolioID         uuid.UUID        `json:"portfolio_id" db:"portfolio_id"`
	SecurityID          *uuid.UUID       `json:"security_id,omitempty" db:"security_id"`
	TradeDate           time.Time        `json:"trade_date" db:"trade_date"`
	SettlementDate      *time.Time       `json:"settlement_date,omitempty" db:"settlement_date"`
	BookingDate         *time.Time       `json:"booking_date,omitempty" db:"booking_date"`
	TransactionType     string           `json:"transaction_type" db:"transaction_type"`
	TransactionSubtype  *string          `json:"transaction_subtype,omitempty" db:"transaction_subtype"`
	Quantity            *decimal.Decimal `json:"quantity,omitempty" db:"quantity"`
	Price               *decimal.Decimal `json:"price,omitempty" db:"price"`
	GrossAmount         *decimal.Decimal `json:"gross_amount,omitempty" db:"gross_amount"`
	NetAmount           *decimal.Decimal `json:"net_amount,omitempty" db:"net_amount"`
	Commission          *decimal.Decimal `json:"commission,omitempty" db:"commission"`
	Fees                *decimal.Decimal `json:"fees,omitempty" db:"fees"`
	Taxes               *decimal.Decimal `json:"taxes,omitempty" db:"taxes"`
	AccruedInterest     *decimal.Decimal `json:"accrued_interest,omitempty" db:"accrued_interest"`
	TransactionCurrency string           `json:"transaction_currency" db:"transaction_currency"`
	SettlementCurrency  *string          `json:"settlement_currency,omitempty" db:"settlement_currency"`
	FXRate              *decimal.Decimal `json:"fx_rate,omitempty" db:"fx_rate"`
	CounterpartyID      *string          `json:"counterparty_id,omitempty" db:"counterparty_id"`
	BrokerID            *string          `json:"broker_id,omitempty" db:"broker_id"`
	CustodyAccountID    *string          `json:"custody_account_id,omitempty" db:"custody_account_id"`
	CorporateActionID   *string          `json:"corporate_action_id,omitempty" db:"corporate_action_id"`
	Status              string           `json:"status" db:"status"`
	SourceSystem        string           `json:"source_system" db:"source_system"`
	ExternalReference   *string          `json:"external_reference,omitempty" db:"external_reference"`
	ValidFrom           time.Time        `json:"valid_from" db:"valid_from"`
	ValidTo             time.Time        `json:"valid_to" db:"valid_to"`
	TenantID            uuid.UUID        `json:"tenant_id" db:"tenant_id"`
	CoreID              *uuid.UUID       `json:"core_id,omitempty" db:"core_id"`
	CreatedAt           time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at" db:"updated_at"`
}

// TransactionFlowTrace links Transaction → Position Impact
type TransactionFlowTrace struct {
	TraceID        uuid.UUID        `json:"trace_id" db:"trace_id"`
	TransactionID  uuid.UUID        `json:"transaction_id" db:"transaction_id"`
	PositionID     *uuid.UUID       `json:"position_id,omitempty" db:"position_id"`
	ImpactType     string           `json:"impact_type" db:"impact_type"` // OPEN, CLOSE, INCREASE, DECREASE
	QuantityDelta  decimal.Decimal  `json:"quantity_delta" db:"quantity_delta"`
	CostBasisDelta *decimal.Decimal `json:"cost_basis_delta,omitempty" db:"cost_basis_delta"`
	RealizedPL     *decimal.Decimal `json:"realized_pl,omitempty" db:"realized_pl"`
	ProcessedAt    time.Time        `json:"processed_at" db:"processed_at"`
	TenantID       uuid.UUID        `json:"tenant_id" db:"tenant_id"`
}

// FieldSurvivorshipDecision tracks survivorship rules for transaction trace
type FieldSurvivorshipDecision struct {
	FieldName        string      `json:"field_name"`
	SelectedSource   string      `json:"selected_source"`
	SelectedValue    interface{} `json:"selected_value"`
	RuleApplied      string      `json:"rule_applied"`
	CompetingSources []string    `json:"competing_sources"`
}

// TransactionGoldCopyRunResult captures survivorship execution
type TransactionGoldCopyRunResult struct {
	RunID             uuid.UUID                   `json:"run_id"`
	ClusterKey        string                      `json:"cluster_key"`
	RecordsProcessed  int                         `json:"records_processed"`
	RecordsSelected   int                         `json:"records_selected"`
	ConflictsResolved int                         `json:"conflicts_resolved"`
	FieldDecisions    []FieldSurvivorshipDecision `json:"field_decisions"`
	ExecutionTimeMs   int64                       `json:"execution_time_ms"`
	StartedAt         time.Time                   `json:"started_at"`
	CompletedAt       time.Time                   `json:"completed_at"`
	TenantID          uuid.UUID                   `json:"tenant_id"`
}
