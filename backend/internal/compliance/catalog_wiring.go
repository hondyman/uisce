package compliance

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
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
	EdgeTypeMemberOf        = "MEMBER_OF"
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
	EffectiveFrom       time.Time              `json:"effectiveFrom"`
	EffectiveTo         *time.Time             `json:"effectiveTo,omitempty"`
	Citation            string                 `json:"citation,omitempty"`
	Jurisdictions       []string               `json:"jurisdictions,omitempty"`
	SourceVersion       string                 `json:"sourceVersion,omitempty"`
	LibraryStatus       string                 `json:"libraryStatus,omitempty"`
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
		"effective_from":        r.EffectiveFrom.Format(time.RFC3339),
		"citation":              r.Citation,
		"jurisdictions":         r.Jurisdictions,
		"source_version":        r.SourceVersion,
		"library_status":        r.LibraryStatus,
	}

	if r.EffectiveTo != nil {
		props["effective_to"] = r.EffectiveTo.Format(time.RFC3339)
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

// MultiTenantRuleLoader loads and merges Core and Tenant compliance rules with point-in-time effective dating.
type MultiTenantRuleLoader struct {
	db *sql.DB
}

// NewMultiTenantRuleLoader creates a new loader.
func NewMultiTenantRuleLoader(db *sql.DB) *MultiTenantRuleLoader {
	return &MultiTenantRuleLoader{db: db}
}

// LoadTenantActiveRules loads active compliance rules for a tenant as of now.
func (l *MultiTenantRuleLoader) LoadTenantActiveRules(ctx context.Context, tenantID uuid.UUID) ([]ComplianceRuleRecord, error) {
	return l.LoadTenantActiveRulesAsOf(ctx, tenantID, time.Now().UTC())
}

// LoadTenantActiveRulesAsOf loads active compliance rules for a tenant as of a specific evaluation timestamp.
// Dynamically derives whether the tenant is gold_copy from the database.
func (l *MultiTenantRuleLoader) LoadTenantActiveRulesAsOf(ctx context.Context, tenantID uuid.UUID, asOf time.Time) ([]ComplianceRuleRecord, error) {
	isGoldCopy, err := l.IsGoldCopyTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("check gold copy tenant: %w", err)
	}

	// 1. If this tenant is gold_copy, return its direct core rules in effect
	if isGoldCopy {
		return l.loadDirectRulesAsOf(ctx, tenantID, asOf)
	}

	// 2. Otherwise:
	// a) Fetch core rules from the gold copy tenant in effect as of asOf
	goldTenantID, err := l.GetGoldCopyTenantID(ctx)
	if err != nil {
		return nil, fmt.Errorf("get gold copy tenant: %w", err)
	}

	coreRules, err := l.loadDirectRulesAsOf(ctx, goldTenantID, asOf)
	if err != nil {
		return nil, fmt.Errorf("load core rules: %w", err)
	}

	// b) Fetch tenant-specific custom / extended / overridden rules
	tenantRules, err := l.loadDirectRulesAsOf(ctx, tenantID, asOf)
	if err != nil {
		return nil, fmt.Errorf("load tenant rules: %w", err)
	}

	// c) Fetch tenant activation matrix
	activations, err := l.loadTenantActivations(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("load tenant activations: %w", err)
	}

	// d) Merge: Index tenant rules by CoreRuleID and RuleCode
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

	// Resolve Core rules against tenant activation matrix (opt-in by default)
	for _, cr := range coreRules {
		act, active := activations[cr.ID]
		if !active || !act.Enabled {
			continue // Opt-in: core rules are inactive by default until tenant activates them
		}

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

func (l *MultiTenantRuleLoader) loadTenantActivations(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]TenantRuleActivation, error) {
	query := `
		SELECT rule_id, enabled, inherit_mode, activated_by, activated_at, audit_required, created_at, updated_at
		FROM compliance.tenant_rule_activation
		WHERE tenant_id = $1
	`
	rows, err := l.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		// If table does not exist or empty, return empty map
		return map[uuid.UUID]TenantRuleActivation{}, nil
	}
	defer rows.Close()

	activations := make(map[uuid.UUID]TenantRuleActivation)
	for rows.Next() {
		var a TenantRuleActivation
		a.TenantID = tenantID
		err := rows.Scan(
			&a.RuleID, &a.Enabled, &a.InheritMode, &a.ActivatedBy, &a.ActivatedAt,
			&a.AuditRequired, &a.CreatedAt, &a.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan tenant activation: %w", err)
		}
		activations[a.RuleID] = a
	}
	return activations, rows.Err()
}

func (l *MultiTenantRuleLoader) loadDirectRulesAsOf(ctx context.Context, tenantID uuid.UUID, asOf time.Time) ([]ComplianceRuleRecord, error) {
	query := `
		SELECT id, tenant_id, core_rule_id, inherit_mode, pinned_core_version, drift_status,
		       rule_code, name, coalesce(description, ''), rule_phase, severity,
		       ast_condition, parameter_thresholds, compiled_bytecode, priority, is_active,
		       effective_from, effective_to, coalesce(citation, ''), jurisdictions,
		       coalesce(source_version, ''), coalesce(library_status, 'ACTIVE'),
		       created_at, updated_at
		FROM compliance.compliance_rule
		WHERE tenant_id = $1
		  AND is_active = true
		  AND valid_to IS NULL
		  AND effective_from <= $2
		  AND (effective_to IS NULL OR $2 < effective_to)
		ORDER BY priority ASC, rule_code ASC
	`
	rows, err := l.db.QueryContext(ctx, query, tenantID, asOf)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []ComplianceRuleRecord
	for rows.Next() {
		var r ComplianceRuleRecord
		var astBytes, paramBytes []byte
		var inheritModeStr, driftStatusStr string
		var jurisdictions []string

		err := rows.Scan(
			&r.ID, &r.TenantID, &r.CoreRuleID, &inheritModeStr, &r.PinnedCoreVersion, &driftStatusStr,
			&r.RuleCode, &r.Name, &r.Description, &r.RulePhase, &r.Severity,
			&astBytes, &paramBytes, &r.CompiledBytecode, &r.Priority, &r.IsActive,
			&r.EffectiveFrom, &r.EffectiveTo, &r.Citation, pq.Array(&jurisdictions),
			&r.SourceVersion, &r.LibraryStatus,
			&r.CreatedAt, &r.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan rule: %w", err)
		}

		r.InheritMode = InheritMode(inheritModeStr)
		r.DriftStatus = DriftStatus(driftStatusStr)
		r.Jurisdictions = jurisdictions

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

// IsGoldCopyTenant checks whether a given tenant is the master gold-copy tenant.
// It queries public.tenants.gold_copy and compares with public.uisce_gold_copy_tenant_id().
func (l *MultiTenantRuleLoader) IsGoldCopyTenant(ctx context.Context, tenantID uuid.UUID) (bool, error) {
	var isGold bool
	err := l.db.QueryRowContext(ctx, "SELECT COALESCE(gold_copy, false) FROM public.tenants WHERE id = $1", tenantID).Scan(&isGold)
	if err == nil {
		return isGold, nil
	}

	goldID, err := l.GetGoldCopyTenantID(ctx)
	if err == nil && goldID != uuid.Nil {
		return goldID == tenantID, nil
	}
	return false, nil
}

// GetGoldCopyTenantID resolves the master gold-copy tenant UUID from the system.
func (l *MultiTenantRuleLoader) GetGoldCopyTenantID(ctx context.Context) (uuid.UUID, error) {
	var id uuid.UUID
	// 1. Check SQL function uisce_gold_copy_tenant_id()
	err := l.db.QueryRowContext(ctx, "SELECT public.uisce_gold_copy_tenant_id()").Scan(&id)
	if err == nil && id != uuid.Nil {
		return id, nil
	}

	// 2. Query public.tenants for gold_copy = true
	err = l.db.QueryRowContext(ctx, "SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1").Scan(&id)
	if err == nil && id != uuid.Nil {
		return id, nil
	}

	// 3. Check if compliance rules exist under core library source_version
	err = l.db.QueryRowContext(ctx, "SELECT tenant_id FROM compliance.compliance_rule WHERE source_version = 'CORE_LIB_V1' LIMIT 1").Scan(&id)
	if err == nil && id != uuid.Nil {
		return id, nil
	}

	return uuid.Nil, errors.New("no gold copy master tenant configured")
}
