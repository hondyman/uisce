package compliance

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ComplianceRule represents edm.compliance_rule
type ComplianceRule struct {
	RuleID            uuid.UUID        `json:"rule_id" db:"rule_id"`
	RuleCode          string           `json:"rule_code" db:"rule_code"`
	RuleName          string           `json:"rule_name" db:"rule_name"`
	Description       *string          `json:"description" db:"description"`
	ScopeType         *string          `json:"scope_type" db:"scope_type"`
	ScopeValue        *string          `json:"scope_value" db:"scope_value"`
	Expression        string           `json:"expression" db:"expression"`
	ExpressionType    string           `json:"expression_type" db:"expression_type"`
	ThresholdValue    *decimal.Decimal `json:"threshold_value" db:"threshold_value"`
	ThresholdOperator *string          `json:"threshold_operator" db:"threshold_operator"`
	Severity          *string          `json:"severity" db:"severity"`
	Status            string           `json:"status" db:"status"`
	EffectiveFrom     time.Time        `json:"effective_from" db:"effective_from"`
	EffectiveTo       *time.Time       `json:"effective_to" db:"effective_to"`
	ValidFrom         time.Time        `json:"valid_from" db:"valid_from"`
	ValidTo           time.Time        `json:"valid_to" db:"valid_to"`
	SystemFrom        time.Time        `json:"system_from" db:"system_from"`
	SystemTo          time.Time        `json:"system_to" db:"system_to"`
	TenantID          uuid.UUID        `json:"tenant_id" db:"tenant_id"`
	CoreID            *uuid.UUID       `json:"core_id" db:"core_id"`
	CreatedBy         *uuid.UUID       `json:"created_by" db:"created_by"`
	UpdatedBy         *uuid.UUID       `json:"updated_by" db:"updated_by"`
	CreatedAt         time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at" db:"updated_at"`
}

// ComplianceEvaluation represents edm.compliance_evaluation
type ComplianceEvaluation struct {
	EvaluationID     uuid.UUID        `json:"evaluation_id" db:"evaluation_id"`
	RuleID           uuid.UUID        `json:"rule_id" db:"rule_id"`
	PortfolioID      uuid.UUID        `json:"portfolio_id" db:"portfolio_id"`
	ValuationDate    time.Time        `json:"valuation_date" db:"valuation_date"`
	MetricValue      *decimal.Decimal `json:"metric_value" db:"metric_value"`
	ThresholdValue   *decimal.Decimal `json:"threshold_value" db:"threshold_value"`
	Result           *string          `json:"result" db:"result"`
	Details          json.RawMessage  `json:"details" db:"details"`
	EvaluationTimeMs *int             `json:"evaluation_time_ms" db:"evaluation_time_ms"`
	EvaluatedAt      time.Time        `json:"evaluated_at" db:"evaluated_at"`
	TenantID         uuid.UUID        `json:"tenant_id" db:"tenant_id"`
}

// ComplianceBreach represents edm.compliance_breach
type ComplianceBreach struct {
	BreachID        uuid.UUID        `json:"breach_id" db:"breach_id"`
	EvaluationID    uuid.UUID        `json:"evaluation_id" db:"evaluation_id"`
	RuleID          uuid.UUID        `json:"rule_id" db:"rule_id"`
	PortfolioID     uuid.UUID        `json:"portfolio_id" db:"portfolio_id"`
	ValuationDate   time.Time        `json:"valuation_date" db:"valuation_date"`
	Severity        string           `json:"severity" db:"severity"`
	MetricValue     *decimal.Decimal `json:"metric_value" db:"metric_value"`
	ThresholdValue  *decimal.Decimal `json:"threshold_value" db:"threshold_value"`
	Deviation       *decimal.Decimal `json:"deviation" db:"deviation"`
	Message         *string          `json:"message" db:"message"`
	Status          string           `json:"status" db:"status"`
	Priority        string           `json:"priority" db:"priority"`
	AssignedTo      *uuid.UUID       `json:"assigned_to" db:"assigned_to"`
	ResolvedAt      *time.Time       `json:"resolved_at" db:"resolved_at"`
	ResolvedBy      *uuid.UUID       `json:"resolved_by" db:"resolved_by"`
	ResolutionNotes *string          `json:"resolution_notes" db:"resolution_notes"`
	WaiverExpiry    *time.Time       `json:"waiver_expiry" db:"waiver_expiry"`
	CreatedAt       time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at" db:"updated_at"`
	TenantID        uuid.UUID        `json:"tenant_id" db:"tenant_id"`
}

// ComplianceLineage represents edm.compliance_lineage
type ComplianceLineage struct {
	LineageID          uuid.UUID        `json:"lineage_id" db:"lineage_id"`
	EvaluationID       uuid.UUID        `json:"evaluation_id" db:"evaluation_id"`
	SourceDomain       string           `json:"source_domain" db:"source_domain"`
	SourceTable        string           `json:"source_table" db:"source_table"`
	SourceRecordID     *uuid.UUID       `json:"source_record_id" db:"source_record_id"`
	ContributionType   *string          `json:"contribution_type" db:"contribution_type"`
	ContributionAmount *decimal.Decimal `json:"contribution_amount" db:"contribution_amount"`
	ProcessedAt        time.Time        `json:"processed_at" db:"processed_at"`
	TenantID           uuid.UUID        `json:"tenant_id" db:"tenant_id"`
}
