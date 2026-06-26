package rules

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// ComplianceRule represents a rule stored in the database.
// Deprecated: Moving specific rule types (Tenant/Core) below
type ComplianceRule struct {
	ID          uuid.UUID `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	RuleType    string    `db:"rule_type" json:"rule_type"`
	Expression  string    `db:"expression" json:"expression"`
	Severity    string    `db:"severity" json:"severity"`
	Enabled     bool      `db:"enabled" json:"enabled"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// CoreValidationRule represents a compiled core rule
type CoreValidationRule struct {
	CoreRuleID   string    `db:"core_rule_id" json:"coreRuleId"`
	RuleKey      string    `db:"rule_key" json:"ruleKey"`
	Version      int       `db:"version" json:"version"`
	ModuleName   string    `db:"module_name" json:"moduleName"`
	Entrypoint   string    `db:"entrypoint" json:"entrypoint"`
	ConditionSrc string    `db:"condition_src" json:"conditionSrc"`
	IsActive     bool      `db:"is_active" json:"isActive"`
	CreatedAt    time.Time `db:"created_at" json:"createdAt"`
}

// TenantValidationRule represents a tenant's customization of a rule
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

// RuleScenario represents a container for testing "what-if" changes
type RuleScenario struct {
	ID          string    `db:"id" json:"id"`
	TenantID    string    `db:"tenant_id" json:"tenantId"`
	BaseRuleID  *string   `db:"base_rule_id" json:"baseRuleId,omitempty"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`
	Status      string    `db:"status" json:"status"` // draft, running, completed, archived
	CreatedBy   string    `db:"created_by" json:"createdBy"`
	CreatedAt   time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

// RuleScenarioVersion represents an immutable snapshot of a rule for simulation
type RuleScenarioVersion struct {
	ID           string          `db:"id" json:"id"`
	ScenarioID   string          `db:"scenario_id" json:"scenarioId"`
	Version      int             `db:"version" json:"version"`
	RuleSnapshot json.RawMessage `db:"rule_snapshot" json:"ruleSnapshot"`
	CreatedAt    time.Time       `db:"created_at" json:"createdAt"`
	CreatedBy    string          `db:"created_by" json:"createdBy"`
}

// RuleTestRun represents an execution of a rule (scenario or standard) against a sample
type RuleTestRun struct {
	ID                string     `db:"id" json:"id"`
	TenantID          string     `db:"tenant_id" json:"tenantId"`
	ScenarioVersionID *string    `db:"scenario_version_id" json:"scenarioVersionId,omitempty"`
	Status            string     `db:"status" json:"status"` // pending, running, completed, failed
	SampleSize        int        `db:"sample_size" json:"sampleSize"`
	FailureCount      int        `db:"failure_count" json:"failureCount"`
	StartedAt         time.Time  `db:"started_at" json:"startedAt"`
	CompletedAt       *time.Time `db:"completed_at" json:"completedAt,omitempty"`
}
