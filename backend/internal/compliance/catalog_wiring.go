package compliance

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/catalog"
)

// Graph Entity Kinds for Compliance
const (
	KindComplianceRule    = "COMPLIANCE_RULE"
	KindComplianceRuleSet = "COMPLIANCE_RULESET"
)

// Graph Edge Types for Compliance
const (
	EdgeTypeExtendsCoreRule = "EXTENDS_CORE_RULE"
	EdgeTypeAppliedTo       = "IS_APPLIED_TO"
	EdgeTypeClassifiedAs    = "IS_CLASSIFIED_AS"
)

// InheritMode represents rule inheritance mode
type InheritMode string

const (
	Inherit  InheritMode = "inherit"
	Extend   InheritMode = "extend"
	Override InheritMode = "override"
	Custom   InheritMode = "custom"
)

// DriftStatus represents rule drift state
type DriftStatus string

const (
	DriftCurrent            DriftStatus = "CURRENT"
	DriftCoreVersionUpdated DriftStatus = "CORE_VERSION_UPDATED"
	DriftDetected           DriftStatus = "DRIFT_DETECTED"
	DriftReconciled         DriftStatus = "RECONCILED"
)

// ExtendsCoreRuleProps defines the strictly validated properties on an EXTENDS_CORE_RULE edge.
type ExtendsCoreRuleProps struct {
	PinnedCoreVersion int         `json:"pinned_core_version"`
	DriftStatus       DriftStatus `json:"drift_status"`
	ExtendedAt        time.Time   `json:"extended_at"`
	LastReviewedAt    *time.Time  `json:"last_reviewed_at,omitempty"`
	ReviewerID        string      `json:"reviewer_id,omitempty"`
}

// ValidateExtendsCoreRuleProps validates edge properties for EXTENDS_CORE_RULE edges.
func ValidateExtendsCoreRuleProps(props map[string]interface{}) (*ExtendsCoreRuleProps, error) {
	if props == nil {
		return nil, errors.New("EXTENDS_CORE_RULE edge properties cannot be nil")
	}

	raw, err := json.Marshal(props)
	if err != nil {
		return nil, fmt.Errorf("marshal edge props: %w", err)
	}

	var parsed ExtendsCoreRuleProps
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal edge props: %w", err)
	}

	if parsed.PinnedCoreVersion <= 0 {
		return nil, fmt.Errorf("pinned_core_version must be a positive integer, got %d", parsed.PinnedCoreVersion)
	}

	switch parsed.DriftStatus {
	case DriftCurrent, DriftCoreVersionUpdated, DriftDetected, DriftReconciled:
		// valid
	default:
		return nil, fmt.Errorf("invalid drift_status: %q", parsed.DriftStatus)
	}

	return &parsed, nil
}

// ComplianceRuleRecord represents a compliance rule from the compliance.compliance_rule table.
type ComplianceRuleRecord struct {
	ID                  uuid.UUID              `json:"id"`
	TenantID            uuid.UUID              `json:"tenantId"`
	CoreRuleID          *uuid.UUID             `json:"coreRuleId,omitempty"`
	InheritMode         InheritMode            `json:"inheritMode"`
	PinnedCoreVersion   int                    `json:"pinnedCoreVersion"`
	DriftStatus         DriftStatus            `json:"driftStatus"`
	RuleCode            string                 `json:"ruleCode"`
	Name                string                 `json:"name"`
	Description         string                 `json:"description"`
	RulePhase           string                 `json:"rulePhase"`
	Severity            string                 `json:"severity"`
	ASTCondition        map[string]interface{} `json:"astCondition"`
	ParameterThresholds map[string]interface{} `json:"parameterThresholds"`
	CompiledBytecode    []byte                 `json:"-"` // Kept separate from graph JSON payload
	Priority            int                    `json:"priority"`
	IsActive            bool                   `json:"isActive"`
	CreatedAt           time.Time              `json:"createdAt"`
	UpdatedAt           time.Time              `json:"updatedAt"`
}

// ToCatalogNode converts a ComplianceRuleRecord to a clean CatalogNode for the semantic graph.
// NOTE: Bytecode is strictly omitted from node.Properties to prevent 50KB payload bloat in graph traversals.
func (r *ComplianceRuleRecord) ToCatalogNode() catalog.CatalogNode {
	props := map[string]interface{}{
		"rule_code":             r.RuleCode,
		"rule_phase":            r.RulePhase,
		"severity":              r.Severity,
		"inherit_mode":          string(r.InheritMode),
		"pinned_core_version":   r.PinnedCoreVersion,
		"drift_status":          string(r.DriftStatus),
		"priority":              r.Priority,
		"is_active":             r.IsActive,
		"has_compiled_bytecode": len(r.CompiledBytecode) > 0,
	}

	if r.CoreRuleID != nil {
		props["core_rule_id"] = r.CoreRuleID.String()
	}

	return catalog.CatalogNode{
		ID:            r.ID.String(),
		TenantID:      r.TenantID.String(),
		NodeType:      KindComplianceRule,
		Kind:          KindComplianceRule,
		Name:          r.Name,
		QualifiedPath: fmt.Sprintf("compliance.rule/%s/%s", r.TenantID.String(), r.RuleCode),
		Description:   r.Description,
		Properties:    props,
		CreatedAt:     r.CreatedAt,
		UpdatedAt:     r.UpdatedAt,
	}
}

// MultiTenantRuleLoader loads and merges Core and Tenant compliance rules.
type MultiTenantRuleLoader struct {
	db *sql.DB
}

// NewMultiTenantRuleLoader creates a new loader.
func NewMultiTenantRuleLoader(db *sql.DB) *MultiTenantRuleLoader {
	return &MultiTenantRuleLoader{db: db}
}

// LoadTenantActiveRules loads all active compliance rules for a tenant, resolving inheritance from Core.
func (l *MultiTenantRuleLoader) LoadTenantActiveRules(ctx context.Context, tenantID uuid.UUID, isGoldCopy bool) ([]ComplianceRuleRecord, error) {
	// 1. If this tenant is gold_copy, return its direct core rules
	if isGoldCopy {
		return l.loadDirectRules(ctx, tenantID)
	}

	// 2. Otherwise:
	// a) Fetch core rules from the gold copy tenant
	goldTenantID, err := l.getGoldCopyTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get gold copy tenant: %w", err)
	}

	coreRules, err := l.loadDirectRules(ctx, goldTenantID)
	if err != nil {
		return nil, fmt.Errorf("load core rules: %w", err)
	}

	// b) Fetch tenant-specific custom / extended / overridden rules
	tenantRules, err := l.loadDirectRules(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load tenant rules: %w", err)
	}

	// c) Merge: Index tenant rules by CoreRuleID and RuleCode
	tenantByCoreID := make(map[uuid.UUID]ComplianceRuleRecord)
	tenantByCode := make(map[string]ComplianceRuleRecord)
	customRules := make([]ComplianceRuleRecord, 0)

	for _, tr := range tenantRules {
		if tr.CoreRuleID != nil {
			tenantByCoreID[*tr.CoreRuleID] = tr
		} else {
			tenantByCode[tr.RuleCode] = tr
			if tr.InheritMode == Custom {
				customRules = append(customRules, tr)
			}
		}
	}

	merged := make([]ComplianceRuleRecord, 0, len(coreRules)+len(customRules))

	// Resolve Core rules
	for _, cr := range coreRules {
		if override, exists := tenantByCoreID[cr.ID]; exists {
			switch override.InheritMode {
			case Extend:
				// Tenant extends Core: uses tenant thresholds with pinned core AST
				effective := override
				if effective.ASTCondition == nil || len(effective.ASTCondition) == 0 {
					effective.ASTCondition = cr.ASTCondition
				}
				merged = append(merged, effective)
			case Override:
				// Full replacement
				merged = append(merged, override)
			case Inherit:
				// Passthrough core rule
				merged = append(merged, cr)
			}
		} else if directByCode, codeExists := tenantByCode[cr.RuleCode]; codeExists {
			merged = append(merged, directByCode)
		} else {
			// Default: inherit core rule directly
			inherited := cr
			inherited.TenantID = tenantID
			inherited.InheritMode = Inherit
			merged = append(merged, inherited)
		}
	}

	// Add brand new custom tenant rules
	merged = append(merged, customRules...)

	return merged, nil
}

func (l *MultiTenantRuleLoader) loadDirectRules(ctx context.Context, tenantID uuid.UUID) ([]ComplianceRuleRecord, error) {
	query := `
		SELECT id, tenant_id, core_rule_id, inherit_mode, pinned_core_version, drift_status,
		       rule_code, name, coalesce(description, ''), rule_phase, severity,
		       ast_condition, parameter_thresholds, compiled_bytecode, priority, is_active,
		       created_at, updated_at
		FROM compliance.compliance_rule
		WHERE tenant_id = $1 AND is_active = true AND valid_to IS NULL
		ORDER BY priority ASC, rule_code ASC
	`
	rows, err := l.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []ComplianceRuleRecord
	for rows.Next() {
		var r ComplianceRuleRecord
		var astBytes, paramBytes []byte
		var inheritModeStr, driftStatusStr string

		err := rows.Scan(
			&r.ID, &r.TenantID, &r.CoreRuleID, &inheritModeStr, &r.PinnedCoreVersion, &driftStatusStr,
			&r.RuleCode, &r.Name, &r.Description, &r.RulePhase, &r.Severity,
			&astBytes, &paramBytes, &r.CompiledBytecode, &r.Priority, &r.IsActive,
			&r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}

		r.InheritMode = InheritMode(inheritModeStr)
		r.DriftStatus = DriftStatus(driftStatusStr)

		if len(astBytes) > 0 {
			_ = json.Unmarshal(astBytes, &r.ASTCondition)
		}
		if len(paramBytes) > 0 {
			_ = json.Unmarshal(paramBytes, &r.ParameterThresholds)
		}

		rules = append(rules, r)
	}

	return rules, rows.Err()
}

func (l *MultiTenantRuleLoader) getGoldCopyTenantID(ctx context.Context) (uuid.UUID, error) {
	var id uuid.UUID
	err := l.db.QueryRowContext(ctx, "SELECT uisce_gold_copy_tenant_id()").Scan(&id)
	if err != nil {
		// Fallback query if stored in tenants table
		err = l.db.QueryRowContext(ctx, "SELECT id FROM tenants WHERE gold_copy = true LIMIT 1").Scan(&id)
		if err != nil {
			return uuid.Nil, fmt.Errorf("gold copy tenant not found: %w", err)
		}
	}
	return id, nil
}
