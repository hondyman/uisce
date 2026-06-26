package cash

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// CashBalanceRecord aligns with edm.cash_balance_master
// Aligns with Semantic Design §5: Business Objects as Structured Entities
type CashBalanceRecord struct {
	CashBalanceID   uuid.UUID        `json:"cash_balance_id" db:"cash_balance_id"`
	PortfolioID     uuid.UUID        `json:"portfolio_id" db:"portfolio_id"`
	CashAccountID   *string          `json:"cash_account_id,omitempty" db:"cash_account_id"`
	Currency        string           `json:"currency" db:"currency"`
	ValuationDate   time.Time        `json:"valuation_date" db:"valuation_date"`
	OpeningBalance  *decimal.Decimal `json:"opening_balance,omitempty" db:"opening_balance"`
	CashInflows     *decimal.Decimal `json:"cash_inflows,omitempty" db:"cash_inflows"`
	CashOutflows    *decimal.Decimal `json:"cash_outflows,omitempty" db:"cash_outflows"`
	InterestAccrual *decimal.Decimal `json:"interest_accrual,omitempty" db:"interest_accrual"`
	FXEffect        *decimal.Decimal `json:"fx_effect,omitempty" db:"fx_effect"`
	ClosingBalance  *decimal.Decimal `json:"closing_balance,omitempty" db:"closing_balance"`
	SourceSystem    string           `json:"source_system" db:"source_system"`
	IsClosed        bool             `json:"is_closed" db:"is_closed"`
	ValidFrom       time.Time        `json:"valid_from" db:"valid_from"`
	ValidTo         time.Time        `json:"valid_to" db:"valid_to"`
	TenantID        uuid.UUID        `json:"tenant_id" db:"tenant_id"`
	CoreID          *uuid.UUID       `json:"core_id,omitempty" db:"core_id"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" db:"updated_at"`
}

// CashLedgerEntryRecord aligns with edm.cash_ledger
type CashLedgerEntryRecord struct {
	CashLedgerID      uuid.UUID       `json:"cash_ledger_id" db:"cash_ledger_id"`
	PortfolioID       uuid.UUID       `json:"portfolio_id" db:"portfolio_id"`
	CashAccountID     *string         `json:"cash_account_id,omitempty" db:"cash_account_id"`
	Currency          string          `json:"currency" db:"currency"`
	ValueDate         time.Time       `json:"value_date" db:"value_date"`
	BookingDate       *time.Time      `json:"booking_date,omitempty" db:"booking_date"`
	CashEventType     string          `json:"cash_event_type" db:"cash_event_type"`
	CashEventSubtype  *string         `json:"cash_event_subtype,omitempty" db:"cash_event_subtype"`
	Amount            decimal.Decimal `json:"amount" db:"amount"`
	AmountSign        *string         `json:"amount_sign,omitempty" db:"amount_sign"`
	TransactionID     *uuid.UUID      `json:"transaction_id,omitempty" db:"transaction_id"`
	SecurityID        *uuid.UUID      `json:"security_id,omitempty" db:"security_id"`
	CounterpartyID    *string         `json:"counterparty_id,omitempty" db:"counterparty_id"`
	Status            string          `json:"status" db:"status"`
	SourceSystem      string          `json:"source_system" db:"source_system"`
	ExternalReference *string         `json:"external_reference,omitempty" db:"external_reference"`
	ValidFrom         time.Time       `json:"valid_from" db:"valid_from"`
	ValidTo           time.Time       `json:"valid_to" db:"valid_to"`
	TenantID          uuid.UUID       `json:"tenant_id" db:"tenant_id"`
	CoreID            *uuid.UUID      `json:"core_id,omitempty" db:"core_id"`
	CreatedAt         time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at" db:"updated_at"`
}

// CashFlowTrace links Ledger → Balance (Lineage - Whitepaper §9)
type CashFlowTrace struct {
	TraceID            uuid.UUID       `json:"trace_id" db:"trace_id"`
	CashBalanceID      uuid.UUID       `json:"cash_balance_id" db:"cash_balance_id"`
	CashLedgerID       *uuid.UUID      `json:"cash_ledger_id,omitempty" db:"cash_ledger_id"`
	ContributionAmount decimal.Decimal `json:"contribution_amount" db:"contribution_amount"`
	ContributionType   string          `json:"contribution_type" db:"contribution_type"`
	ProcessedAt        time.Time       `json:"processed_at" db:"processed_at"`
	TenantID           uuid.UUID       `json:"tenant_id" db:"tenant_id"`
}

// TransactionCashMapping links Transaction → Cash Ledger
type TransactionCashMapping struct {
	MappingID     uuid.UUID       `json:"mapping_id" db:"mapping_id"`
	TransactionID uuid.UUID       `json:"transaction_id" db:"transaction_id"`
	CashLedgerID  *uuid.UUID      `json:"cash_ledger_id,omitempty" db:"cash_ledger_id"`
	MappingType   string          `json:"mapping_type" db:"mapping_type"`
	Amount        decimal.Decimal `json:"amount" db:"amount"`
	Currency      string          `json:"currency" db:"currency"`
	ValueDate     time.Time       `json:"value_date" db:"value_date"`
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	TenantID      uuid.UUID       `json:"tenant_id" db:"tenant_id"`
}

// FieldSurvivorshipDecision tracks survivorship rules
type FieldSurvivorshipDecision struct {
	FieldName        string      `json:"field_name"`
	SelectedSource   string      `json:"selected_source"`
	SelectedValue    interface{} `json:"selected_value"`
	RuleApplied      string      `json:"rule_applied"`
	CompetingSources []string    `json:"competing_sources"`
}

// CashGoldCopyRunResult captures survivorship execution
type CashGoldCopyRunResult struct {
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
