package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ValidationRuleProperties is stored in catalog_node.properties, mirroring
// PreAggProperties' role for pre-aggregation nodes and term_type/
// return_type's role for calculated semantic terms - node metadata that
// isn't the AST itself.
type ValidationRuleProperties struct {
	BOName           string `json:"bo_name"`
	TenantID         string `json:"tenant_id"`
	Severity         string `json:"severity"` // "BLOCK" | "WARN"
	Timing           string `json:"timing"`   // "pre_write" | "reconcile"
	Category         string `json:"category,omitempty"`
	GovernanceStatus string `json:"governance_status,omitempty"` // "draft", "review", "published", "deprecated"
}

const (
	ValidationRuleSeverityBlock = "BLOCK"
	ValidationRuleSeverityWarn  = "WARN"

	ValidationRuleTimingPreWrite  = "pre_write"
	ValidationRuleTimingReconcile = "reconcile"
)

// ValidationRuleConfig is stored in catalog_node.config. RuleAST is a
// vm.RuleNode-shaped json.RawMessage (internal/rules/vm) - kept as
// json.RawMessage here rather than a typed vm.RuleNode so this package
// doesn't need to import internal/rules/vm (models is a low-level,
// widely-imported package; the AST is parsed by whoever evaluates it).
type ValidationRuleConfig struct {
	RuleAST json.RawMessage `json:"rule_ast"`
}

// UpsertValidationRuleRequest is the API request shape for creating or
// updating a validation rule.
type UpsertValidationRuleRequest struct {
	TenantID    string          `json:"tenant_id"`
	BOName      string          `json:"bo_name"`
	Name        string          `json:"name"` // catalog_node.node_name
	Description string          `json:"description,omitempty"`
	Severity    string          `json:"severity"`
	Timing      string          `json:"timing"`
	Category    string          `json:"category,omitempty"`
	RuleAST     json.RawMessage `json:"rule_ast"`
}

// ValidationRuleDescriptor is the API response shape.
type ValidationRuleDescriptor struct {
	ID               uuid.UUID       `json:"id"`
	TenantID         string          `json:"tenant_id"`
	BOName           string          `json:"bo_name"`
	Name             string          `json:"name"`
	Description      string          `json:"description,omitempty"`
	Severity         string          `json:"severity"`
	Timing           string          `json:"timing"`
	Category         string          `json:"category,omitempty"`
	RuleAST          json.RawMessage `json:"rule_ast"`
	GovernanceStatus string          `json:"governance_status"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	IsActive         bool            `json:"is_active"`
}

// ParseValidationRuleProperties unmarshals ValidationRuleProperties from
// catalog_node.properties.
func ParseValidationRuleProperties(raw json.RawMessage) (*ValidationRuleProperties, error) {
	var props ValidationRuleProperties
	if err := json.Unmarshal(raw, &props); err != nil {
		return nil, err
	}
	return &props, nil
}

// ParseValidationRuleConfig unmarshals ValidationRuleConfig from
// catalog_node.config.
func ParseValidationRuleConfig(raw json.RawMessage) (*ValidationRuleConfig, error) {
	var cfg ValidationRuleConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
