package models

import (
	"encoding/json"
	"time"
)

const (
	RuleBundleVersion = "1.0"

	// ValidationRuleGovernanceDraft is the default status for exported rules
	// that predate governance tracking.
	ValidationRuleGovernanceDraft              = "draft"
	ValidationRuleGovernancePublished          = "published"
	ValidationRuleGovernanceSubmittedForReview = "submitted_for_review"
)

// Overwrite policies for RuleImportRequest.OverwritePolicy.
const (
	ImportOverwriteFail      = "fail_on_conflict"
	ImportOverwriteSkip      = "skip_existing"
	ImportOverwriteOverwrite = "overwrite"
)

// Import error codes (machine-readable, actionable).
const (
	ImportErrCoreShadow     = "ERR_CORE_SHADOW"
	ImportErrUnresolvedTerm = "ERR_UNRESOLVED_SEMANTIC_TERM"
	ImportErrMissingBO      = "ERR_MISSING_BO"
	ImportErrDuplicateKey   = "ERR_DUPLICATE_RULE_KEY"
	ImportErrMissingDep     = "ERR_MISSING_DEPENDENCY"
	ImportErrDepCycle       = "ERR_DEPENDENCY_CYCLE"
	ImportErrDeleteMissing  = "ERR_DELETE_MISSING"
	ImportErrInvalidField   = "ERR_INVALID_FIELD"
	ImportErrConflict       = "ERR_CONFLICT"
	ImportErrGovernance     = "ERR_GOVERNANCE"
)

// RuleBundle represents a portable export of validation rules.
// The canonical interchange format is JSON; YAML is a read-only convenience
// converted to JSON before any checksumming occurs.
type RuleBundle struct {
	BundleVersion string             `json:"bundle_version"` // e.g. "1.0"
	CreatedAt     time.Time          `json:"created_at"`
	ExportedFrom  string             `json:"exported_from"`
	TenantScope   string             `json:"tenant_scope"` // "core" or tenant ID
	Origin        string             `json:"origin"`       // "core" | "custom"
	Checksum      string             `json:"checksum"`     // SHA-256 over CanonicalBytes()
	Rules         []PortableRuleSpec `json:"rules"`
}

// PortableRuleSpec defines a single rule stripped of environment-specific UUIDs.
type PortableRuleSpec struct {
	RuleKey          string          `json:"rule_key"`
	Name             string          `json:"name"`
	BOName           string          `json:"bo_name"`
	Description      string          `json:"description,omitempty"`
	Domain           string          `json:"domain"`
	Severity         string          `json:"severity"`
	Timing           string          `json:"timing"`
	Category         string          `json:"category,omitempty"`
	GovernanceStatus string          `json:"governance_status"`
	BindingScope     []string        `json:"binding_scope,omitempty"`
	DependsOn        []string        `json:"depends_on,omitempty"`
	RuleAST          json.RawMessage `json:"rule_ast,omitempty"` // canonical vm.RuleNode; empty when Deleted
	Deleted          bool            `json:"deleted,omitempty"`  // tombstone; checksum-relevant
}

// RuleImportRequest specifies import options.
type RuleImportRequest struct {
	Bundle          RuleBundle `json:"bundle"`
	TargetTenantID  string     `json:"target_tenant_id"`
	DryRun          bool       `json:"dry_run"`
	OverwritePolicy string     `json:"overwrite_policy"` // "fail_on_conflict" | "skip_existing" | "overwrite"
	PreserveStatus  bool       `json:"preserve_governance_status"`
	Prune           bool       `json:"prune"`                     // delete tenant-scoped custom rules absent from bundle (opt-in)
	IdempotencyKey  string     `json:"idempotency_key,omitempty"` // or via X-Idempotency-Key header
}

// RuleImportReport returns granular results of the import.
type RuleImportReport struct {
	DryRun      bool                    `json:"dry_run"`
	TotalRules  int                     `json:"total_rules"`
	Created     []string                `json:"created"`
	Updated     []string                `json:"updated"`
	Skipped     []string                `json:"skipped"`
	Pruned      []string                `json:"pruned,omitempty"`
	Errors      []RuleImportErrorDetail `json:"errors"`
	DiffSummary []RuleDiffSummary       `json:"diff_summary,omitempty"`
}

type RuleImportErrorDetail struct {
	Code    string `json:"code"`
	RuleKey string `json:"rule_key"`
	Name    string `json:"name"`
	Reason  string `json:"reason"`
}

type RuleDiffSummary struct {
	RuleKey string   `json:"rule_key"`
	Action  string   `json:"action"` // "CREATE" | "UPDATE" | "NO_OP" | "DELETE"
	Changes []string `json:"changes,omitempty"`
}
