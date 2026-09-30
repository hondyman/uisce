package mdm

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// ResolutionTier indicates where a resolved rule originated
type ResolutionTier string

const (
	ResolutionTierTenantOverride   ResolutionTier = "TENANT_OVERRIDE"
	ResolutionTierCoreInherited    ResolutionTier = "CORE_INHERITED"
	ResolutionTierDefaultHierarchy ResolutionTier = "DEFAULT_HIERARCHY"
	ResolutionTierSystemFallback   ResolutionTier = "SYSTEM_FALLBACK"
)

// ResolvedFieldRule represents the complete survivorship configuration for a single attribute
type ResolvedFieldRule struct {
	ID              uuid.UUID      `json:"id"`
	AttributeName   string         `json:"attribute_name"`
	SemanticTermID  uuid.UUID      `json:"semantic_term_id"`
	EntityType      string         `json:"entity_type"`
	Strategy        string         `json:"strategy"` // SOURCE_PRIORITY, MOST_RECENT, CONSERVATIVE_MIN, CONSERVATIVE_MAX, WEIGHTED_CONFIDENCE, MOST_FREQUENT
	PriorityOrder   []string       `json:"priority_order"`
	MaxStaleSeconds int            `json:"max_stale_seconds"`
	SelectionRuleID *string        `json:"selection_rule_id,omitempty"`
	SelectionMode   string         `json:"selection_mode,omitempty"` // ENFORCE, FLAG
	ResolutionTier  ResolutionTier `json:"resolution_tier"`          // TENANT_OVERRIDE, CORE_INHERITED, DEFAULT_HIERARCHY, SYSTEM_FALLBACK
	IsActive        bool           `json:"is_active"`
}

// rawRuleRow represents a row from semantic_survivorship_rules with join to semantic term/attribute
type rawRuleRow struct {
	ID              uuid.UUID      `db:"id"`
	TenantID        uuid.UUID      `db:"tenant_id"`
	EntityType      string         `db:"entity_type"`
	SemanticTermID  uuid.UUID      `db:"semantic_term_id"`
	AttributeName   string         `db:"attribute_name"`
	Strategy        string         `db:"strategy"`
	PriorityOrder   pq.StringArray `db:"priority_order"`
	MaxStaleSeconds int            `db:"max_stale_seconds"`
	IsActive        bool           `db:"is_active"`
}

// RuleResolver resolves effective survivorship rules across the Core-vs-Delta boundary
type RuleResolver struct {
	db *sql.DB
}

// NewRuleResolver creates a new resolver instance
func NewRuleResolver(db *sql.DB) *RuleResolver {
	return &RuleResolver{db: db}
}

// ResolveEntityRules computes the effective, merged survivorship rule set for an entity type
func (r *RuleResolver) ResolveEntityRules(ctx context.Context, tenantID uuid.UUID, entityType string) (map[string]ResolvedFieldRule, error) {
	if r.db == nil {
		return nil, errors.New("database connection required")
	}
	if tenantID == uuid.Nil {
		return nil, errors.New("tenant_id is required")
	}
	entityType = strings.ToUpper(strings.TrimSpace(entityType))
	if entityType == "" {
		return nil, errors.New("entity_type is required")
	}

	// 1. Identify Core Tenant ID
	var coreTenantID uuid.UUID
	coreQuery := `SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1`
	err := r.db.QueryRowContext(ctx, coreQuery).Scan(&coreTenantID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("lookup core tenant: %w", err)
	}

	// Core tenant self-resolution short-circuit:
	// When tenantID is the core tenant, own rows are the core baseline
	if tenantID == coreTenantID {
		return r.resolveCoreRulesOnly(ctx, coreTenantID, entityType)
	}

	// 2. Query rules for both the requesting client tenant and the core tenant
	query := `
		SELECT 
			r.id,
			r.tenant_id,
			r.entity_type,
			r.semantic_term_id,
			COALESCE(t.node_name, t.qualified_path, r.semantic_term_id::text) AS attribute_name,
			r.strategy,
			r.priority_order,
			r.max_stale_seconds,
			r.is_active
		FROM public.semantic_survivorship_rules r
		LEFT JOIN public.catalog_node t ON t.id = r.semantic_term_id
		WHERE r.entity_type = $1 
		  AND r.is_active = true
		  AND (r.tenant_id = $2 OR r.tenant_id = $3)
	`

	dbx := sqlx.NewDb(r.db, "postgres")
	var rows []rawRuleRow
	err = dbx.SelectContext(ctx, &rows, query, entityType, tenantID, coreTenantID)
	if err != nil {
		return nil, fmt.Errorf("query survivorship rules: %w", err)
	}

	// 3. Separate tenant delta rules from core baseline rules
	tenantRules := make(map[string]rawRuleRow)
	coreRules := make(map[string]rawRuleRow)

	for _, row := range rows {
		key := strings.ToLower(row.AttributeName)
		if row.TenantID == tenantID {
			tenantRules[key] = row
		} else if row.TenantID == coreTenantID {
			coreRules[key] = row
		}
	}

	// 4. Perform atomic per-attribute merge (Tenant delta overrides Core baseline)
	resolved := make(map[string]ResolvedFieldRule)

	// A) First populate from Core baseline
	for attr, coreRow := range coreRules {
		resolved[attr] = ResolvedFieldRule{
			ID:              coreRow.ID,
			AttributeName:   coreRow.AttributeName,
			SemanticTermID:  coreRow.SemanticTermID,
			EntityType:      coreRow.EntityType,
			Strategy:        coreRow.Strategy,
			PriorityOrder:   coreRow.PriorityOrder,
			MaxStaleSeconds: coreRow.MaxStaleSeconds,
			ResolutionTier:  ResolutionTierCoreInherited,
			IsActive:        coreRow.IsActive,
		}
	}

	// B) Apply Tenant Delta overrides
	for attr, deltaRow := range tenantRules {
		resolved[attr] = ResolvedFieldRule{
			ID:              deltaRow.ID,
			AttributeName:   deltaRow.AttributeName,
			SemanticTermID:  deltaRow.SemanticTermID,
			EntityType:      deltaRow.EntityType,
			Strategy:        deltaRow.Strategy,
			PriorityOrder:   deltaRow.PriorityOrder,
			MaxStaleSeconds: deltaRow.MaxStaleSeconds,
			ResolutionTier:  ResolutionTierTenantOverride,
			IsActive:        deltaRow.IsActive,
		}
	}

	return resolved, nil
}

// resolveCoreRulesOnly handles the short-circuit when requesting tenant is Core
func (r *RuleResolver) resolveCoreRulesOnly(ctx context.Context, coreTenantID uuid.UUID, entityType string) (map[string]ResolvedFieldRule, error) {
	query := `
		SELECT 
			r.id,
			r.tenant_id,
			r.entity_type,
			r.semantic_term_id,
			COALESCE(t.node_name, t.qualified_path, r.semantic_term_id::text) AS attribute_name,
			r.strategy,
			r.priority_order,
			r.max_stale_seconds,
			r.is_active
		FROM public.semantic_survivorship_rules r
		LEFT JOIN public.catalog_node t ON t.id = r.semantic_term_id
		WHERE r.entity_type = $1 
		  AND r.is_active = true
		  AND r.tenant_id = $2
	`
	dbx := sqlx.NewDb(r.db, "postgres")
	var rows []rawRuleRow
	err := dbx.SelectContext(ctx, &rows, query, entityType, coreTenantID)
	if err != nil {
		return nil, fmt.Errorf("query core survivorship rules: %w", err)
	}

	resolved := make(map[string]ResolvedFieldRule)
	for _, row := range rows {
		key := strings.ToLower(row.AttributeName)
		resolved[key] = ResolvedFieldRule{
			ID:              row.ID,
			AttributeName:   row.AttributeName,
			SemanticTermID:  row.SemanticTermID,
			EntityType:      row.EntityType,
			Strategy:        row.Strategy,
			PriorityOrder:   row.PriorityOrder,
			MaxStaleSeconds: row.MaxStaleSeconds,
			ResolutionTier:  ResolutionTierCoreInherited,
			IsActive:        row.IsActive,
		}
	}
	return resolved, nil
}
