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
	// Domain distinguishes which rule-authoring surface produced this
	// rule - "validation" (the original BO-scoped surface), "mdm", or
	// "compliance" (the domain values the rulefabric consolidation adds;
	// see docs/unified-rule-engine-handoff.md, "Rulefabric consolidation").
	// Empty/omitted (every rule written before this field existed) is
	// read as ValidationRuleDomainDefault ("validation") by GetByID/
	// ListByBO's descriptorFromNode - the service defaults new writes to
	// it explicitly rather than leaving new rows blank too, so "domain"
	// is unambiguous for anything written from this point forward.
	Domain string `json:"domain,omitempty"`
	// BindingIDs scopes the rule to specific bindings of the BO
	// (business_object_binding.bo_binding_id). Empty means the rule applies to every
	// binding, which is what every rule written before this field existed
	// means, so they are unchanged. The rule's field references stay
	// semantic terms either way; scoping only decides whether the rule runs
	// for a write that arrived through a given binding.
	BindingIDs []string `json:"binding_ids,omitempty"`
}

const (
	ValidationRuleSeverityBlock = "BLOCK"
	ValidationRuleSeverityWarn  = "WARN"

	ValidationRuleTimingPreWrite  = "pre_write"
	ValidationRuleTimingReconcile = "reconcile"

	ValidationRuleDomainDefault    = "validation"
	ValidationRuleDomainMDM        = "mdm"
	ValidationRuleDomainCompliance = "compliance"

	// Origin of a rule as seen by a tenant. A rule authored in the gold-copy tenant is "core": every
	// tenant inherits it read-only. A rule authored in the tenant itself is "custom" and applies to
	// that tenant only.
	ValidationRuleOriginCore   = "core"
	ValidationRuleOriginCustom = "custom"
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
	// Domain: "mdm" or "compliance" for the rulefabric-consolidation
	// domain values; empty defaults to ValidationRuleDomainDefault
	// ("validation") in the service layer.
	Domain  string          `json:"domain,omitempty"`
	RuleAST json.RawMessage `json:"rule_ast"`
	// BindingIDs optionally scopes the rule to specific bindings of the BO;
	// see ValidationRuleProperties.BindingIDs. Each must be a binding of
	// BOName in this tenant.
	BindingIDs []string `json:"binding_ids,omitempty"`
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
	Domain           string          `json:"domain"`
	BindingIDs       []string        `json:"binding_ids,omitempty"`
	Origin           string          `json:"origin,omitempty"` // "core" | "custom", relative to the requesting tenant
	RuleAST          json.RawMessage `json:"rule_ast"`
	GovernanceStatus string          `json:"governance_status"`
	IsActive         bool            `json:"is_active"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
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
