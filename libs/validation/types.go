package validation

import (
	"context"
	"time"
)

type InheritMode string

const (
	Inherit  InheritMode = "inherit"
	Extend   InheritMode = "extend"
	Override InheritMode = "override"
	Custom   InheritMode = "custom"
)

type TenantValidationRule struct {
	TenantID        string      `db:"tenant_id" json:"tenantId"`
	RuleID          string      `db:"rule_id" json:"ruleId"`
	CoreRuleID      *string     `db:"core_rule_id" json:"coreRuleId,omitempty"`
	InheritMode     InheritMode `db:"inherit_mode" json:"inheritMode"`
	CreatedFromVers *int        `db:"created_from_vers" json:"createdFromVers,omitempty"`
	ConditionSrc    string      `db:"condition_src" json:"conditionSrc"`
	CreatedAt       time.Time   `db:"created_at" json:"createdAt"`
	UpdatedAt       time.Time   `db:"updated_at" json:"updatedAt"`
}

type RuleEngine interface {
	EvaluateTenantRule(ctx context.Context, rule *TenantValidationRule, boCtx map[string]map[string]interface{}) (bool, error)
}
