package models

import (
	"time"

	"github.com/google/uuid"
)

// CompliancePolicy represents a versioned validation rule in the database
type CompliancePolicy struct {
	PolicyID           int        `db:"policy_id" json:"policyId"`
	VersionTag         string     `db:"version_tag" json:"versionTag"`
	EffectiveStartDate time.Time  `db:"effective_start_date" json:"effectiveStartDate"`
	EffectiveEndDate   *time.Time `db:"effective_end_date" json:"effectiveEndDate,omitempty"`
	RuleType           string     `db:"rule_type" json:"ruleType"` // PRE_TRADE or POST_TRADE
	CueContent         string     `db:"cue_content" json:"cueContent"`
	CreatedAt          time.Time  `db:"created_at" json:"createdAt"`
	CreatedBy          string     `db:"created_by" json:"createdBy"`
}

// TradeRequest represents an incoming trade validation request
type TradeRequest struct {
	ID         string   `json:"id"`
	TradeDate  string   `json:"tradeDate"`
	Amount     float64  `json:"amount"`
	Currency   string   `json:"currency"`
	SecurityID string   `json:"securityId"`
	OrderType  string   `json:"orderType"`
	LimitPrice *float64 `json:"limitPrice,omitempty"`

	// For post-trade checks that require approval
	ApprovalStatus *string `json:"approvalStatus,omitempty"`
}

// ComplianceEvent represents a validation result event
type ComplianceEvent struct {
	EventID      uuid.UUID              `db:"event_id" json:"eventId"`
	TraceID      string                 `db:"trace_id" json:"traceId"`
	EventType    string                 `db:"event_type" json:"eventType"` // PRE_TRADE or POST_TRADE
	Status       string                 `db:"status" json:"status"`        // PASS or FAIL
	RuleVersion  string                 `db:"rule_version" json:"ruleVersion"`
	TradeData    map[string]interface{} `db:"trade_data" json:"tradeData"`
	ErrorDetails map[string]interface{} `db:"error_details" json:"errorDetails,omitempty"`
	CreatedAt    time.Time              `db:"created_at" json:"createdAt"`
}

// ComplianceAuditLog represents a comprehensive audit entry
type ComplianceAuditLog struct {
	AuditID    uuid.UUID              `db:"audit_id" json:"auditId"`
	EventID    uuid.UUID              `db:"event_id" json:"eventId"`
	TenantID   *uuid.UUID             `db:"tenant_id" json:"tenantId,omitempty"`
	UserID     *uuid.UUID             `db:"user_id" json:"userId,omitempty"`
	Action     string                 `db:"action" json:"action"`
	EntityType string                 `db:"entity_type" json:"entityType"`
	EntityID   string                 `db:"entity_id" json:"entityId"`
	Metadata   map[string]interface{} `db:"metadata" json:"metadata,omitempty"`
	CreatedAt  time.Time              `db:"created_at" json:"createdAt"`
}

// ValidationResult is the response from validation
type ValidationResult struct {
	Status      string                 `json:"status"` // APPROVED or REJECTED
	TraceID     string                 `json:"traceId"`
	RuleVersion string                 `json:"ruleVersion"`
	Errors      []string               `json:"errors,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ValidatedAt time.Time              `json:"validatedAt"`
}
