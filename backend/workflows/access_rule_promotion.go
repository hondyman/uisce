package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// PromoteRuleParams defines parameters for rule promotion workflow.
type PromoteRuleParams struct {
	RuleID      string
	TargetEnv   string // "staging" | "prod"
	RequestedBy string
}

// PromoteAccessRuleWorkflow orchestrates the promotion of an access rule across environments.
func PromoteAccessRuleWorkflow(ctx workflow.Context, params PromoteRuleParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("PromoteAccessRuleWorkflow started", "ruleId", params.RuleID, "targetEnv", params.TargetEnv)

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1. Load rule from source environment
	var rule AccessRule
	err := workflow.ExecuteActivity(ctx, "LoadRuleActivity", params.RuleID).Get(ctx, &rule)
	if err != nil {
		logger.Error("Failed to load rule", "error", err)
		return fmt.Errorf("load rule failed: %w", err)
	}
	logger.Info("Rule loaded", "ruleId", rule.RuleID, "businessObjectId", rule.BusinessObjectID)

	// 2. Validate DSL syntax
	err = workflow.ExecuteActivity(ctx, "ValidateRuleSyntaxActivity", rule).Get(ctx, nil)
	if err != nil {
		logger.Error("DSL validation failed", "error", err)
		return fmt.Errorf("DSL validation failed: %w", err)
	}
	logger.Info("DSL validation passed")

	// 3. Impact analysis
	var impact ImpactReport
	err = workflow.ExecuteActivity(ctx, "ImpactAnalysisActivity", rule).Get(ctx, &impact)
	if err != nil {
		logger.Error("Impact analysis failed", "error", err)
		return fmt.Errorf("impact analysis failed: %w", err)
	}
	logger.Info("Impact analysis complete", "termsAffected", len(impact.SemanticTerms), "apisAffected", len(impact.Apis))

	// 4. Run security tests in target environment
	err = workflow.ExecuteActivity(ctx, "RunSecurityTestsActivity", rule, params.TargetEnv).Get(ctx, nil)
	if err != nil {
		logger.Error("Security tests failed", "error", err)
		return fmt.Errorf("security tests failed: %w", err)
	}
	logger.Info("Security tests passed")

	// 5. Wait for approval (via signal)
	var approved bool
	signalChannel := workflow.GetSignalChannel(ctx, "approval-decision")

	// Set timeout for approval
	approvalTimeout := 7 * 24 * time.Hour // 7 days
	selector := workflow.NewSelector(ctx)

	selector.AddReceive(signalChannel, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &approved)
	})

	timerFuture := workflow.NewTimer(ctx, approvalTimeout)
	selector.AddFuture(timerFuture, func(f workflow.Future) {
		logger.Warn("Approval timeout reached")
		approved = false
	})

	selector.Select(ctx)

	if !approved {
		logger.Warn("Rule promotion not approved or timed out")
		return fmt.Errorf("rule %s not approved for promotion to %s", rule.RuleID, params.TargetEnv)
	}
	logger.Info("Rule promotion approved")

	// 6. Promote rule to target environment
	err = workflow.ExecuteActivity(ctx, "PromoteRuleActivity", rule, params.TargetEnv).Get(ctx, nil)
	if err != nil {
		logger.Error("Rule promotion failed", "error", err)
		return fmt.Errorf("rule promotion failed: %w", err)
	}
	logger.Info("Rule promoted successfully")

	// 7. Emit audit event and invalidate caches
	err = workflow.ExecuteActivity(ctx, "EmitAuditAndInvalidateCacheActivity", rule, params.TargetEnv, params.RequestedBy).Get(ctx, nil)
	if err != nil {
		logger.Error("Post-promotion cleanup failed", "error", err)
		return fmt.Errorf("post-promotion cleanup failed: %w", err)
	}
	logger.Info("Audit emitted and caches invalidated")

	logger.Info("PromoteAccessRuleWorkflow completed successfully", "ruleId", params.RuleID, "targetEnv", params.TargetEnv)
	return nil
}

// AccessRule represents an access rule for workflow processing.
type AccessRule struct {
	RuleID           string
	TenantID         string
	BusinessObjectID string
	GroupDn          string
	AccessLevel      string
	Status           string
	RowFilterDsl     string
	ColumnMasks      []ColumnMask
	AppliesToApis    bool
	AppliesToBi      bool
	AppliesToAi      bool
}

// ColumnMask represents a field-level masking rule.
type ColumnMask struct {
	SemanticTermID string
	MaskType       string // "HIDE" | "MASK" | "NONE"
}

// ImpactReport captures downstream impact of a rule.
type ImpactReport struct {
	SemanticTerms []string
	Apis          []string
	BiArtifacts   []string
	AiArtifacts   []string
}
