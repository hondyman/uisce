package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Trade represents a single trade record
type Trade struct {
	ID          uuid.UUID       `json:"id" db:"id"`
	PortfolioID uuid.UUID       `json:"portfolio_id" db:"portfolio_id"`
	Symbol      string          `json:"symbol" db:"symbol"`
	Action      string          `json:"action" db:"action"` // buy, sell
	Shares      float64         `json:"shares" db:"shares"`
	Price       float64         `json:"price" db:"price"`
	TradeDate   time.Time       `json:"trade_date" db:"trade_date"`
	SettleDate  time.Time       `json:"settle_date" db:"settle_date"`
	Custodian   string          `json:"custodian" db:"custodian"`
	Status      string          `json:"status" db:"status"` // pending, confirmed, discrepancy
	CreatedAt   time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" db:"updated_at"`
	Metadata    json.RawMessage `json:"metadata" db:"metadata"`
}

// TradeConfirm represents a confirmation from custodian/broker
type TradeConfirm struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	Source     string          `json:"source" db:"source"` // email, sftp, api, manual
	RawData    json.RawMessage `json:"raw_data" db:"raw_data"`
	Parsed     json.RawMessage `json:"parsed" db:"parsed"`
	ReceivedAt time.Time       `json:"received_at" db:"received_at"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
}

// ReconciliationResult holds the output of an AI reconciliation run
type ReconciliationResult struct {
	ID              uuid.UUID       `json:"id" db:"id"`
	RunDate         time.Time       `json:"run_date" db:"run_date"`
	MatchRate       float64         `json:"match_rate" db:"match_rate"`
	MatchedCount    int             `json:"matched_count" db:"matched_count"`
	UnmatchedCount  int             `json:"unmatched_count" db:"unmatched_count"`
	DiscrepancyJSON json.RawMessage `json:"discrepancies" db:"discrepancies"`
	ModelVersion    int             `json:"model_version" db:"model_version"`
	Status          string          `json:"status" db:"status"` // in_progress, completed, failed
	ErrorMessage    *string         `json:"error_message" db:"error_message"`
	CreatedAt       time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at" db:"updated_at"`
}

// Unmarshal the discrepancies from JSON
func (r *ReconciliationResult) GetDiscrepancies() ([]Discrepancy, error) {
	var discs []Discrepancy
	if err := json.Unmarshal(r.DiscrepancyJSON, &discs); err != nil {
		return nil, err
	}
	return discs, nil
}

// TradeMatch represents a successful match between trade and confirm
type TradeMatch struct {
	TradeID     string    `json:"trade_id"`
	ConfirmID   string    `json:"confirm_id"`
	Confidence  float64   `json:"confidence"`
	MatchFields []string  `json:"match_fields"` // which fields matched
	MatchedAt   time.Time `json:"matched_at"`
}

// Discrepancy represents a mismatch or unmatched item
type Discrepancy struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	ResultID     uuid.UUID       `json:"result_id" db:"result_id"`
	TradeID      *string         `json:"trade_id" db:"trade_id"`
	ConfirmID    *string         `json:"confirm_id" db:"confirm_id"`
	DiscrepType  string          `json:"discrepancy_type" db:"discrepancy_type"` // unmatched_trade, unmatched_confirm, mismatch
	Field        *string         `json:"field" db:"field"`                       // for mismatches
	TradeValue   json.RawMessage `json:"trade_value" db:"trade_value"`
	ConfirmValue json.RawMessage `json:"confirm_value" db:"confirm_value"`
	Severity     string          `json:"severity" db:"severity"` // low, medium, high
	SuggestedFix *string         `json:"suggested_fix" db:"suggested_fix"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
}

// ReconciliationTask represents action items for ops team
type ReconciliationTask struct {
	ID            uuid.UUID  `json:"id" db:"id"`
	ResultID      uuid.UUID  `json:"result_id" db:"result_id"`
	DiscrepancyID uuid.UUID  `json:"discrepancy_id" db:"discrepancy_id"`
	Status        string     `json:"status" db:"status"` // open, in_progress, resolved, escalated
	AssignedTo    *uuid.UUID `json:"assigned_to" db:"assigned_to"`
	Priority      string     `json:"priority" db:"priority"` // low, medium, high
	Notes         string     `json:"notes" db:"notes"`
	ResolvedAt    *time.Time `json:"resolved_at" db:"resolved_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
}

// ReconciliationRule represents a low-code matching tolerance rule
type ReconciliationRule struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	RuleType    string    `json:"rule_type" db:"rule_type"` // share_tolerance, price_tolerance, date_tolerance
	Enabled     bool      `json:"enabled" db:"enabled"`
	RuleExpr    string    `json:"rule_expr" db:"rule_expr"` // JSONata expression
	Version     int       `json:"version" db:"version"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// ReconciliationAuditLog tracks all reconciliation operations
type ReconciliationAuditLog struct {
	ID        uuid.UUID       `json:"id" db:"id"`
	ResultID  uuid.UUID       `json:"result_id" db:"result_id"`
	Action    string          `json:"action" db:"action"` // reconciliation_started, ai_matched, rule_applied, task_created, task_resolved
	Actor     *uuid.UUID      `json:"actor" db:"actor"`
	Details   json.RawMessage `json:"details" db:"details"`
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
}

// JSONB type for postgres JSONB columns
type JSONB json.RawMessage

func (j JSONB) Value() (driver.Value, error) {
	return json.RawMessage(j), nil
}

func (j *JSONB) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("type assertion to []byte failed: %v", value)
	}
	*j = JSONB(bytes)
	return nil
}
