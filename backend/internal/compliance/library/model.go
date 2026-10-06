package library

import (
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance"
)

// CoreRuleSummary represents a concise summary of a core library rule
type CoreRuleSummary struct {
	ID             uuid.UUID `json:"id"`
	RuleCode       string    `json:"rule_code"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	RulePhase      string    `json:"rule_phase"`
	Severity       string    `json:"severity"`
	Priority       int       `json:"priority"`
	Jurisdictions  []string  `json:"jurisdictions"`
	Citation       string    `json:"citation"`
	CurrentVersion int       `json:"current_version"`
	ContentHash    string    `json:"content_hash"`
	LibraryStatus  string    `json:"library_status"`
	EffectiveFrom  time.Time `json:"effective_from"`
	EffectiveTo    *time.Time `json:"effective_to,omitempty"`
	RulesetCodes   []string  `json:"ruleset_codes"`
	Domain         string    `json:"domain"`
}

// RuleVersionSummary represents a historical snapshot of a rule
type RuleVersionSummary struct {
	Version      int       `json:"version"`
	EffectiveFrom time.Time `json:"effective_from"`
	EffectiveTo  *time.Time `json:"effective_to,omitempty"`
	Citation     string    `json:"citation"`
	ContentHash  string    `json:"content_hash"`
	CreatedBy    string    `json:"created_by"`
	CreatedAt    time.Time `json:"created_at"`
}

// RuleDetails represents the complete metadata and AST definition of a compliance rule
type RuleDetails struct {
	CoreRuleSummary
	ASTCondition         map[string]interface{}    `json:"ast_condition"`
	ParameterThresholds  map[string]string         `json:"parameter_thresholds"`
	VersionHistory       []RuleVersionSummary      `json:"version_history"`
	SampleTestCases      []Scenario                `json:"sample_test_cases,omitempty"`
}

// RulesetSummary represents a licensable packaging of compliance rules
type RulesetSummary struct {
	RulesetCode string      `json:"ruleset_code"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	PlanTier    string      `json:"plan_tier"`
	TotalRules  int         `json:"total_rules"`
	RuleIDs     []uuid.UUID `json:"rule_ids"`
}

// TenantRulesetStatus represents a tenant's activation state for a specific ruleset
type TenantRulesetStatus struct {
	RulesetCode string `json:"ruleset_code"`
	Name        string `json:"name"`
	Description string `json:"description"`
	PlanTier    string `json:"plan_tier"`
	IsLicensed  bool   `json:"is_licensed"`
	IsActive    bool   `json:"is_active"`
	ActiveRules int    `json:"active_rules"`
	TotalRules  int    `json:"total_rules"`
}

// TenantActivationItem represents a single rule's activation and inheritance configuration for a tenant
type TenantActivationItem struct {
	RuleID              uuid.UUID              `json:"rule_id"`
	CoreRuleID          *uuid.UUID             `json:"core_rule_id,omitempty"`
	RuleCode            string                 `json:"rule_code"`
	RuleName            string                 `json:"rule_name"`
	RulePhase           string                 `json:"rule_phase"`
	Severity            string                 `json:"severity"`
	Citation            string                 `json:"citation"`
	Domain              string                 `json:"domain"`
	Jurisdictions       []string               `json:"jurisdictions"`
	RulesetCodes        []string               `json:"ruleset_codes"`
	CurrentCoreVersion  int                    `json:"current_core_version"`
	PinnedCoreVersion   int                    `json:"pinned_core_version"`
	Enabled             bool                   `json:"enabled"`
	InheritMode         compliance.InheritMode `json:"inherit_mode"` // "inherit", "extend", "custom"
	DriftStatus         compliance.DriftStatus `json:"drift_status"` // "CURRENT", "CORE_VERSION_UPDATED", "DRIFT_DETECTED", "RECONCILED"
	CoreThresholds      map[string]string      `json:"core_thresholds"`
	TenantThresholds    map[string]string      `json:"tenant_thresholds"`
	CoreContentHash     string                 `json:"core_content_hash"`
	TenantContentHash   string                 `json:"tenant_content_hash,omitempty"`
	ActivatedBy         string                 `json:"activated_by,omitempty"`
	ActivatedAt         *time.Time             `json:"activated_at,omitempty"`
}

// TenantActivationMatrix represents the full compliance configuration for a tenant
type TenantActivationMatrix struct {
	TenantID     uuid.UUID              `json:"tenant_id"`
	TenantName   string                 `json:"tenant_name"`
	Plan         string                 `json:"plan"`
	GoldCopy     bool                   `json:"gold_copy"`
	Rulesets     []TenantRulesetStatus  `json:"rulesets"`
	Rules        []TenantActivationItem `json:"rules"`
	TotalActive  int                    `json:"total_active"`
	TotalRules   int                    `json:"total_rules"`
	DriftCount   int                    `json:"drift_count"`
}

// UpdateRuleActivationRequest contains parameters for updating a tenant's rule configuration
type UpdateRuleActivationRequest struct {
	Enabled            bool                   `json:"enabled"`
	InheritMode        compliance.InheritMode `json:"inherit_mode"` // "inherit", "extend", "custom"
	ParameterOverrides map[string]string      `json:"parameter_overrides,omitempty"`
	ActorID            string                 `json:"actor_id"`
}

// RepinRuleRequest contains parameters for repinning an extended rule to a new core version
type RepinRuleRequest struct {
	TargetVersion int    `json:"target_version"`
	ActorID       string `json:"actor_id"`
	StewardNotes  string `json:"steward_notes,omitempty"`
}

// ToggleRulesetRequest contains parameters for toggling all rules in a ruleset
type ToggleRulesetRequest struct {
	Enabled bool   `json:"enabled"`
	ActorID string `json:"actor_id"`
}

// ListRulesFilter represents query parameters for filtering rules in the library
type ListRulesFilter struct {
	Domain        string `json:"domain,omitempty"`
	RulesetCode   string `json:"ruleset_code,omitempty"`
	LibraryStatus string `json:"library_status,omitempty"`
	RulePhase     string `json:"rule_phase,omitempty"`
	Severity      string `json:"severity,omitempty"`
	Jurisdiction  string `json:"jurisdiction,omitempty"`
	SearchQuery   string `json:"search_query,omitempty"`
}
