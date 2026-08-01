package api

import (
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/rules"
)

type SeverityLevel string

const (
	SeverityHardBlock SeverityLevel = "HARD_BLOCK"
	SeveritySoftWarn  SeverityLevel = "SOFT_WARNING"
	SeverityInfo      SeverityLevel = "INFORMATIONAL"
)

type ExternalEvaluateRequest struct {
	SystemIdentifier string         `json:"system_identifier"`
	PortfolioID     string         `json:"portfolio_id"`
	ProposedTrade   map[string]any `json:"proposed_trade"`
	RuleChainID    string         `json:"rule_chain_id"`
}

type ExternalBatchEvaluateRequest struct {
	SystemIdentifier string              `json:"system_identifier"`
	BatchID         string              `json:"batch_id"`
	Trades          []ExternalTradeItem `json:"trades"`
	OverrideReason  string              `json:"override_reason,omitempty"`
}

type ExternalTradeItem struct {
	ExternalOrderID string         `json:"external_order_id"`
	PortfolioID    string         `json:"portfolio_id"`
	TradeData      map[string]any `json:"trade_data"`
}

type ExternalEvaluateResponse struct {
	Approved          bool                  `json:"approved"`
	CanOverride      bool                  `json:"can_override"`
	HighestSeverity  SeverityLevel         `json:"highest_severity"`
	EvaluatedVM     bool                  `json:"evaluated_vm"`
	ExecutionTimeNs int64                 `json:"execution_time_ns"`
	TraceRevision   uint64                `json:"trace_revision"`
	Violations      []rules.RuleViolation `json:"violations,omitempty"`
	Timestamp       time.Time             `json:"timestamp"`
}

type ExternalBatchEvaluateResponse struct {
	BatchID         string                  `json:"batch_id"`
	AllApproved     bool                    `json:"all_approved"`
	HighestSeverity SeverityLevel           `json:"highest_severity"`
	ExecutionTimeNs int64                   `json:"execution_time_ns"`
	Results         []ExternalTradeResult   `json:"results"`
	Timestamp       time.Time              `json:"timestamp"`
}

type ExternalTradeResult struct {
	ExternalOrderID string                `json:"external_order_id"`
	PortfolioID    string                `json:"portfolio_id"`
	Approved       bool                  `json:"approved"`
	CanOverride   bool                  `json:"can_override"`
	Violations     []rules.RuleViolation `json:"violations,omitempty"`
}

type OverrideRecord struct {
	TenantID            uuid.UUID             `json:"tenant_id"`
	TradeID            string               `json:"trade_id"`
	OverrideByUser     string               `json:"override_by_user"`
	JustificationReason string               `json:"justification_reason"`
	ViolationsBypassed []rules.RuleViolation `json:"violations_bypassed"`
	Timestamp          time.Time            `json:"timestamp"`
}
