package rules

import (
	"context"
	"database/sql"
	"fmt"
	"sort"

	"github.com/google/uuid"
)

// ValidationRuleRepository handles validation rules for business objects
type ValidationRuleRepository struct {
	db             *sql.DB
	catalogService CatalogService
}

// CatalogService provides catalog graph queries
type CatalogService interface {
	GetTermsForRule(ctx context.Context, ruleNodeID uuid.UUID, tenantID uuid.UUID, datasourceID *uuid.UUID) ([]SemanticTerm, error)
	GetFieldsForBusinessObject(ctx context.Context, boID uuid.UUID, tenantID uuid.UUID, datasourceID *uuid.UUID) ([]FieldDefinition, error)
	GetRuleNodeID(ctx context.Context, ruleID uuid.UUID, tenantID uuid.UUID) (uuid.UUID, error)
	GetImpactGraph(ctx context.Context, ruleNodeID uuid.UUID, tenantID uuid.UUID) ([]ImpactNode, error)
}

// NewValidationRuleRepository creates a new validation rule repository
func NewValidationRuleRepository(db *sql.DB, catalogService CatalogService) *ValidationRuleRepository {
	return &ValidationRuleRepository{
		db:             db,
		catalogService: catalogService,
	}
}

// ResolveRulesForBusinessObject resolves the final list of rules for a business object
// considering core rules, tenant overrides, and enrichment via catalog
func (r *ValidationRuleRepository) ResolveRulesForBusinessObject(
	ctx context.Context,
	tenantID uuid.UUID,
	boID uuid.UUID,
	datasourceID *uuid.UUID,
) ([]ResolvedRule, error) {

	// 1. Load tenant rules
	tenantRules, err := r.loadRules(ctx, tenantID, boID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load tenant rules: %w", err)
	}

	// 2. Load core rules (Gold Copy tenant)
	// TODO: Get GoldCopyTenantID from configuration
	goldCopyID := uuid.MustParse("00000000-0000-0000-0000-000000000000") // placeholder
	coreRules, err := r.loadRules(ctx, goldCopyID, boID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to load core rules: %w", err)
	}

	// 3. Apply tenant overrides
	resolved := r.applyOverrides(coreRules, tenantRules)

	// 4. Enrich via catalog graph (semantic terms)
	resolved, err = r.enrichWithSemanticTerms(ctx, resolved, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to enrich rules: %w", err)
	}

	// 5. Sort by evaluation order
	sort.Slice(resolved, func(i, j int) bool {
		return resolved[i].EvaluationOrder < resolved[j].EvaluationOrder
	})

	return resolved, nil
}

// loadRules loads raw rule records from the database
func (r *ValidationRuleRepository) loadRules(
	ctx context.Context,
	tenantID uuid.UUID,
	boID uuid.UUID,
	datasourceID *uuid.UUID,
) ([]RuleRecord, error) {

	query := `
		SELECT 
			id, tenant_id, target_entity_id, name, description, rule_type,
			compiled_sql, compiled_wasm, compiled_cue, execute_server_side, execute_client_side,
			run_on_submit, severity, remediation_hint, evaluation_order, is_active,
			core_rule_id, datasource_id
		FROM validation_rules
		WHERE tenant_id = $1
		  AND target_entity_id = $2
		  AND is_active = true
	`

	args := []interface{}{tenantID, boID}

	if datasourceID != nil {
		query += " AND (datasource_id IS NULL OR datasource_id = $3)"
		args = append(args, *datasourceID)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []RuleRecord
	for rows.Next() {
		var rec RuleRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.TargetEntityID, &rec.Name, &rec.Description,
			&rec.RuleType, &rec.CompiledSQL, &rec.CompiledWASM, &rec.CompiledCUE,
			&rec.ExecuteServerSide, &rec.ExecuteClientSide, &rec.RunOnSubmit, &rec.Severity,
			&rec.RemediationHint, &rec.EvaluationOrder, &rec.IsActive,
			&rec.CoreRuleID, &rec.DatasourceID,
		); err != nil {
			return nil, err
		}
		rules = append(rules, rec)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return rules, nil
}

// applyOverrides applies tenant overrides to core rules
func (r *ValidationRuleRepository) applyOverrides(coreRules, tenantRules []RuleRecord) []ResolvedRule {
	coreMap := make(map[uuid.UUID]RuleRecord)
	for _, c := range coreRules {
		coreMap[c.ID] = c
	}

	// Remove core rules that are overridden
	for _, t := range tenantRules {
		if t.CoreRuleID != nil {
			delete(coreMap, *t.CoreRuleID)
		}
	}

	// Combine core + tenant
	resolved := []ResolvedRule{}
	for _, c := range coreMap {
		resolved = append(resolved, r.toResolved(c))
	}
	for _, t := range tenantRules {
		resolved = append(resolved, r.toResolved(t))
	}

	return resolved
}

// toResolved converts a RuleRecord to a ResolvedRule
func (r *ValidationRuleRepository) toResolved(rec RuleRecord) ResolvedRule {
	var compiledSQL *string
	if rec.CompiledSQL.Valid {
		compiledSQL = &rec.CompiledSQL.String
	}

	var remediationHint *string
	if rec.RemediationHint.Valid {
		remediationHint = &rec.RemediationHint.String
	}

	return ResolvedRule{
		ID:                rec.ID,
		Name:              rec.Name,
		Description:       rec.Description.String,
		RuleType:          rec.RuleType,
		CompiledSQL:       compiledSQL,
		CompiledWASM:      rec.CompiledWASM,
		CompiledCUE:       rec.CompiledCUE.String,
		ExecuteServerSide: rec.ExecuteServerSide,
		ExecuteClientSide: rec.ExecuteClientSide,
		RunOnSubmit:       rec.RunOnSubmit,
		Severity:          rec.Severity,
		RemediationHint:   remediationHint,
		EvaluationOrder:   rec.EvaluationOrder,
		IsActive:          rec.IsActive,
	}
}

// enrichWithSemanticTerms enriches rules with semantic terms from catalog
func (r *ValidationRuleRepository) enrichWithSemanticTerms(
	ctx context.Context,
	rules []ResolvedRule,
	tenantID uuid.UUID,
	datasourceID *uuid.UUID,
) ([]ResolvedRule, error) {

	for i := range rules {
		// Get rule node ID from catalog
		nodeID, err := r.catalogService.GetRuleNodeID(ctx, rules[i].ID, tenantID)
		if err != nil {
			// If rule node doesn't exist in catalog, continue without enrichment
			continue
		}

		// Get terms linked to this rule
		terms, err := r.catalogService.GetTermsForRule(ctx, nodeID, tenantID, datasourceID)
		if err != nil {
			// Continue without enrichment if catalog lookup fails
			continue
		}

		// Store term IDs for client-side resolution
		for _, t := range terms {
			rules[i].SemanticTerms = append(rules[i].SemanticTerms, t.ID)
		}
	}

	return rules, nil
}

// GetValidationRuleSchema returns the schema for building validation rules
func (r *ValidationRuleRepository) GetValidationRuleSchema(
	ctx context.Context,
	tenantID uuid.UUID,
	boID uuid.UUID,
	datasourceID *uuid.UUID,
	locale string,
) (*RuleSchema, error) {

	// 1. Load BO fields
	fields, err := r.catalogService.GetFieldsForBusinessObject(ctx, boID, tenantID, datasourceID)
	if err != nil {
		return nil, fmt.Errorf("failed to load fields: %w", err)
	}

	// 2. Load semantic terms (for this BO)
	// TODO: Implement GetSemanticTermsForBO in catalog service
	terms := []SemanticTerm{}

	// 3. Build schema
	schema := &RuleSchema{
		Fields: fields,
		Terms:  terms,
		Locale: locale,
	}

	return schema, nil
}

// GetRuleImpactGraph returns the impact graph for a rule
func (r *ValidationRuleRepository) GetRuleImpactGraph(
	ctx context.Context,
	ruleID uuid.UUID,
	tenantID uuid.UUID,
) ([]ImpactNode, error) {

	nodeID, err := r.catalogService.GetRuleNodeID(ctx, ruleID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("rule not found in catalog: %w", err)
	}

	impacts, err := r.catalogService.GetImpactGraph(ctx, nodeID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to load impact graph: %w", err)
	}

	return impacts, nil
}

// New types for validation rule resolution
type ValidationRuleRecord struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	TargetEntityID    uuid.UUID
	Name              string
	Description       sql.NullString
	RuleType          string
	CompiledSQL       sql.NullString
	CompiledWASM      sql.RawBytes
	CompiledCUE       sql.NullString
	ExecuteServerSide bool
	ExecuteClientSide bool
	RunOnSubmit       bool
	Severity          string
	RemediationHint   sql.NullString
	EvaluationOrder   int
	IsActive          bool
	CoreRuleID        *uuid.UUID
	DatasourceID      *uuid.UUID
	CreatedAt         sql.NullTime
	UpdatedAt         sql.NullTime
}
