package activities

import (
	"context"
	"fmt"

	"github.com/hondyman/semlayer/backend/internal/logging"
	"github.com/hondyman/semlayer/backend/internal/security"
	"github.com/hondyman/semlayer/backend/workflows"
	"go.uber.org/zap"
)

// AccessRuleActivities encapsulates all activities for rule promotion.
type AccessRuleActivities struct {
	ruleRepo  *security.AccessRuleRepository
	validator *security.DslValidator
	analyzer  *security.ImpactAnalyzer
	// cacheClient would be added for cache invalidation
}

// NewAccessRuleActivities creates activities handler.
func NewAccessRuleActivities(
	ruleRepo *security.AccessRuleRepository,
	validator *security.DslValidator,
	analyzer *security.ImpactAnalyzer,
) *AccessRuleActivities {
	return &AccessRuleActivities{
		ruleRepo:  ruleRepo,
		validator: validator,
		analyzer:  analyzer,
	}
}

// LoadRuleActivity loads a rule from the database.
func (a *AccessRuleActivities) LoadRuleActivity(ctx context.Context, ruleID string) (*workflows.AccessRule, error) {
	rule, err := a.ruleRepo.Get(ctx, ruleID)
	if err != nil {
		return nil, fmt.Errorf("failed to load rule: %w", err)
	}

	workflowRule := &workflows.AccessRule{
		RuleID:           rule.RuleID,
		TenantID:         rule.TenantID,
		BusinessObjectID: rule.BusinessObjectID,
		GroupDn:          rule.GroupDn,
		AccessLevel:      rule.AccessLevel,
		Status:           rule.Status,
		RowFilterDsl:     rule.RowFilterDsl,
		AppliesToApis:    rule.AppliesToApis != nil && *rule.AppliesToApis,
		AppliesToBi:      rule.AppliesToBi != nil && *rule.AppliesToBi,
		AppliesToAi:      rule.AppliesToAi != nil && *rule.AppliesToAi,
	}

	for _, mask := range rule.ColumnMasks {
		workflowRule.ColumnMasks = append(workflowRule.ColumnMasks, workflows.ColumnMask{
			SemanticTermID: mask.SemanticTermID,
			MaskType:       mask.MaskType,
		})
	}

	return workflowRule, nil
}

// ValidateRuleSyntaxActivity validates the DSL syntax of a rule.
func (a *AccessRuleActivities) ValidateRuleSyntaxActivity(ctx context.Context, rule workflows.AccessRule) error {
	if rule.RowFilterDsl == "" {
		return nil
	}

	result, err := a.validator.Validate(ctx, rule.BusinessObjectID, rule.RowFilterDsl)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid {
		return fmt.Errorf("invalid DSL: %s", *result.ErrorMessage)
	}

	return nil
}

// ImpactAnalysisActivity computes the impact of a rule.
func (a *AccessRuleActivities) ImpactAnalysisActivity(ctx context.Context, rule workflows.AccessRule) (*workflows.ImpactReport, error) {
	impact, err := a.analyzer.ComputeImpact(ctx, rule.RuleID)
	if err != nil {
		return nil, fmt.Errorf("impact analysis failed: %w", err)
	}

	report := &workflows.ImpactReport{
		SemanticTerms: impact.SemanticTerms,
		Apis:          impact.Apis,
		BiArtifacts:   impact.BiArtifacts,
		AiArtifacts:   impact.AiArtifacts,
	}

	return report, nil
}

// RunSecurityTestsActivity runs integration tests for the rule in target environment.
func (a *AccessRuleActivities) RunSecurityTestsActivity(ctx context.Context, rule workflows.AccessRule, targetEnv string) error {
	// TODO: Implement actual security tests.
	return nil
}

// PromoteRuleActivity promotes a rule to the target environment.
func (a *AccessRuleActivities) PromoteRuleActivity(ctx context.Context, rule workflows.AccessRule, targetEnv string) error {
	// TODO: Implement environment-specific promotion.
	return nil
}

// EmitAuditAndInvalidateCacheActivity emits audit events and invalidates caches.
func (a *AccessRuleActivities) EmitAuditAndInvalidateCacheActivity(
	ctx context.Context,
	rule workflows.AccessRule,
	targetEnv string,
	requestedBy string,
) error {
	// TODO: Implement audit emission, cache invalidation, and bus publish.
	logger := logging.GetLogger()
	logger.Info(
		"EmitAuditAndInvalidateCacheActivity placeholder",
		zap.String("ruleId", rule.RuleID),
		zap.String("tenantId", rule.TenantID),
		zap.String("targetEnv", targetEnv),
		zap.String("requestedBy", requestedBy),
	)

	return nil
}
