package temporal

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/aso"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// Workflow Inputs
// ============================================================================

// ASOEvaluationInput is the input for ASO evaluation workflow
type ASOEvaluationInput struct {
	Env      string `json:"env"`
	TenantID string `json:"tenant_id,omitempty"` // Empty = all tenants
}

// ASOApplyInput is the input for optimization application workflow
type ASOApplyInput struct {
	OptimizationID string `json:"optimization_id"`
	Actor          string `json:"actor"`
}

// ASOValidationInput is the input for changeset validation
type ASOValidationInput struct {
	ChangeSetID string `json:"changeset_id"`
	Env         string `json:"env"`
	TenantID    string `json:"tenant_id,omitempty"`
}

// ============================================================================
// Workflow Results
// ============================================================================

// ASOEvaluationResult is the result of ASO evaluation
type ASOEvaluationResult struct {
	OptimizationsFound   int      `json:"optimizations_found"`
	OptimizationsApplied int      `json:"optimizations_applied"`
	OptimizationIDs      []string `json:"optimization_ids"`
	TenantsEvaluated     int      `json:"tenants_evaluated"`
	Errors               []string `json:"errors,omitempty"`
}

// ============================================================================
// Workflows
// ============================================================================

// Activity names for registration
const (
	ActivityEvaluateTenantASO         = "EvaluateTenantASO"
	ActivityEvaluateCoreASO           = "EvaluateCoreASO"
	ActivityGetTenantIDs              = "GetTenantIDs"
	ActivityLoadOptimization          = "LoadOptimization"
	ActivityCheckPolicy               = "CheckPolicy"
	ActivityUpdateOptimizationStatus  = "UpdateOptimizationStatus"
	ActivityMarkOptimizationApplied   = "MarkOptimizationApplied"
	ActivityApplyTuneRefresh          = "ApplyTuneRefresh"
	ActivityApplyCreatePreAgg         = "ApplyCreatePreAgg"
	ActivityApplyRetireAsset          = "ApplyRetireAsset"
	ActivityApplyPrewarm              = "ApplyPrewarm"
	ActivityValidateChangeSet         = "ValidateChangeSet"
	ActivityPostPromotionOptimization = "PostPromotionOptimization"
)

// ASOEvaluationWorkflow runs periodic ASO evaluation for an environment
func ASOEvaluationWorkflow(ctx workflow.Context, input ASOEvaluationInput) (*ASOEvaluationResult, error) {
	result := &ASOEvaluationResult{}

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	if input.TenantID != "" {
		// Evaluate single tenant
		var tenantResult TenantEvaluationResult
		err := workflow.ExecuteActivity(ctx, ActivityEvaluateTenantASO, input.Env, input.TenantID).Get(ctx, &tenantResult)
		if err != nil {
			result.Errors = append(result.Errors, err.Error())
		} else {
			result.OptimizationsFound = tenantResult.OptimizationsFound
			result.OptimizationsApplied = tenantResult.OptimizationsApplied
			result.OptimizationIDs = tenantResult.OptimizationIDs
			result.TenantsEvaluated = 1
		}
	} else {
		// Get all tenants
		var tenantIDs []string
		err := workflow.ExecuteActivity(ctx, ActivityGetTenantIDs, input.Env).Get(ctx, &tenantIDs)
		if err != nil {
			return nil, err
		}

		// Evaluate each tenant (consider using child workflows for parallelism)
		for _, tenantID := range tenantIDs {
			var tenantResult TenantEvaluationResult
			err := workflow.ExecuteActivity(ctx, ActivityEvaluateTenantASO, input.Env, tenantID).Get(ctx, &tenantResult)
			if err != nil {
				result.Errors = append(result.Errors, tenantID+": "+err.Error())
				continue
			}

			result.OptimizationsFound += tenantResult.OptimizationsFound
			result.OptimizationsApplied += tenantResult.OptimizationsApplied
			result.OptimizationIDs = append(result.OptimizationIDs, tenantResult.OptimizationIDs...)
			result.TenantsEvaluated++
		}

		// Also evaluate core
		var coreResult TenantEvaluationResult
		if err := workflow.ExecuteActivity(ctx, ActivityEvaluateCoreASO, input.Env).Get(ctx, &coreResult); err == nil {
			result.OptimizationsFound += coreResult.OptimizationsFound
			result.OptimizationIDs = append(result.OptimizationIDs, coreResult.OptimizationIDs...)
		}
	}

	return result, nil
}

// ASOApplyOptimizationWorkflow safely applies an optimization
func ASOApplyOptimizationWorkflow(ctx workflow.Context, input ASOApplyInput) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 2,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1. Load and validate optimization
	var opt aso.ASOOptimization
	err := workflow.ExecuteActivity(ctx, ActivityLoadOptimization, input.OptimizationID).Get(ctx, &opt)
	if err != nil {
		return err
	}

	// 2. Check policy and governance
	var canApply bool
	err = workflow.ExecuteActivity(ctx, ActivityCheckPolicy, opt.Env, opt.TenantID, opt.OptimizationType).Get(ctx, &canApply)
	if err != nil || !canApply {
		return workflow.ExecuteActivity(ctx, ActivityUpdateOptimizationStatus,
			input.OptimizationID, "rejected", input.Actor, "Policy check failed").Get(ctx, nil)
	}

	// 3. Apply based on type
	switch opt.OptimizationType {
	case aso.OptTypeTuneRefresh:
		err = workflow.ExecuteActivity(ctx, ActivityApplyTuneRefresh, opt).Get(ctx, nil)
	case aso.OptTypeCreatePreAgg:
		err = workflow.ExecuteActivity(ctx, ActivityApplyCreatePreAgg, opt).Get(ctx, nil)
	case aso.OptTypeRetireAsset:
		err = workflow.ExecuteActivity(ctx, ActivityApplyRetireAsset, opt).Get(ctx, nil)
	case aso.OptTypePrewarm:
		err = workflow.ExecuteActivity(ctx, ActivityApplyPrewarm, opt).Get(ctx, nil)
	}

	if err != nil {
		workflow.ExecuteActivity(ctx, ActivityUpdateOptimizationStatus,
			input.OptimizationID, "failed", input.Actor, err.Error()).Get(ctx, nil)
		return err
	}

	// 4. Mark as applied
	return workflow.ExecuteActivity(ctx, ActivityMarkOptimizationApplied,
		input.OptimizationID, input.Actor).Get(ctx, nil)
}

// ASOValidationWorkflow validates a changeset for performance impacts
func ASOValidationWorkflow(ctx workflow.Context, input ASOValidationInput) (*aso.ASOValidationResult, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result aso.ASOValidationResult
	err := workflow.ExecuteActivity(ctx, ActivityValidateChangeSet, input.ChangeSetID, input.Env, input.TenantID).Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

// ASOPostPromotionWorkflow re-evaluates after a promotion
func ASOPostPromotionWorkflow(ctx workflow.Context, env, tenantID string) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Re-evaluate workload and update optimizations
	return workflow.ExecuteActivity(ctx, ActivityPostPromotionOptimization, env, tenantID).Get(ctx, nil)
}

// ============================================================================
// Activity Results
// ============================================================================

// TenantEvaluationResult is the result of evaluating a single tenant
type TenantEvaluationResult struct {
	TenantID             string   `json:"tenant_id"`
	OptimizationsFound   int      `json:"optimizations_found"`
	OptimizationsApplied int      `json:"optimizations_applied"`
	OptimizationIDs      []string `json:"optimization_ids"`
}

// ============================================================================
// Activities
// ============================================================================

// ASOActivities contains all ASO-related activities
type ASOActivities struct {
	engine      aso.ASOEngine
	policyStore aso.ASOPolicyStore
	optRepo     aso.ASOOptimizationRepository
}

// NewASOActivities creates ASO activities
func NewASOActivities(
	engine aso.ASOEngine,
	policyStore aso.ASOPolicyStore,
	optRepo aso.ASOOptimizationRepository,
) *ASOActivities {
	return &ASOActivities{
		engine:      engine,
		policyStore: policyStore,
		optRepo:     optRepo,
	}
}

// EvaluateTenantASOActivity evaluates ASO for a single tenant
func (a *ASOActivities) EvaluateTenantASOActivity(ctx context.Context, env, tenantID string) (*TenantEvaluationResult, error) {
	activity.RecordHeartbeat(ctx, "Evaluating tenant: "+tenantID)

	opts, err := a.engine.EvaluateTenant(ctx, env, tenantID)
	if err != nil {
		return nil, err
	}

	result := &TenantEvaluationResult{
		TenantID:           tenantID,
		OptimizationsFound: len(opts),
	}

	for _, opt := range opts {
		result.OptimizationIDs = append(result.OptimizationIDs, opt.ID.String())
		if opt.Status == aso.OptStatusApplied {
			result.OptimizationsApplied++
		}
	}

	return result, nil
}

// EvaluateCoreASOActivity evaluates ASO for core assets
func (a *ASOActivities) EvaluateCoreASOActivity(ctx context.Context, env string) (*TenantEvaluationResult, error) {
	activity.RecordHeartbeat(ctx, "Evaluating core: "+env)

	opts, err := a.engine.EvaluateCore(ctx, env)
	if err != nil {
		return nil, err
	}

	result := &TenantEvaluationResult{
		TenantID:           "core",
		OptimizationsFound: len(opts),
	}

	for _, opt := range opts {
		result.OptimizationIDs = append(result.OptimizationIDs, opt.ID.String())
	}

	return result, nil
}

// GetTenantIDsActivity returns all tenant IDs
func (a *ASOActivities) GetTenantIDsActivity(ctx context.Context, env string) ([]string, error) {
	// This would query your tenants table
	return []string{}, nil
}

// LoadOptimizationActivity loads an optimization by ID
func (a *ASOActivities) LoadOptimizationActivity(ctx context.Context, optID string) (*aso.ASOOptimization, error) {
	id, err := uuid.Parse(optID)
	if err != nil {
		return nil, err
	}
	return a.optRepo.GetByID(ctx, id)
}

// CheckPolicyActivity checks if an optimization can be applied
func (a *ASOActivities) CheckPolicyActivity(ctx context.Context, env string, tenantID *uuid.UUID, optType aso.OptimizationType) (bool, error) {
	policy, err := a.policyStore.GetPolicy(ctx, env, tenantID)
	if err != nil {
		return false, err
	}

	if !policy.Enabled {
		return false, nil
	}

	switch policy.Mode {
	case aso.ASOModeAutoApply:
		return true, nil
	case aso.ASOModeAutoTune:
		return optType == aso.OptTypeTuneRefresh || optType == aso.OptTypePrewarm, nil
	default:
		return false, nil
	}
}

// UpdateOptimizationStatusActivity updates optimization status
func (a *ASOActivities) UpdateOptimizationStatusActivity(ctx context.Context, optID string, status string, actor string, reason string) error {
	id, err := uuid.Parse(optID)
	if err != nil {
		return err
	}
	return a.optRepo.UpdateStatus(ctx, id, aso.OptimizationStatus(status), actor, reason)
}

// MarkOptimizationAppliedActivity marks optimization as applied
func (a *ASOActivities) MarkOptimizationAppliedActivity(ctx context.Context, optID string, actor string) error {
	id, err := uuid.Parse(optID)
	if err != nil {
		return err
	}
	return a.optRepo.MarkApplied(ctx, id, actor, nil)
}

// ApplyTuneRefreshActivity applies a refresh interval tune
func (a *ASOActivities) ApplyTuneRefreshActivity(ctx context.Context, opt aso.ASOOptimization) error {
	// Call your pre-agg service to update refresh interval
	activity.RecordHeartbeat(ctx, "Tuning refresh for: "+opt.TargetName)
	return nil
}

// ApplyCreatePreAggActivity creates a new pre-agg
func (a *ASOActivities) ApplyCreatePreAggActivity(ctx context.Context, opt aso.ASOOptimization) error {
	// Call your pre-agg service to create new pre-agg in draft state
	activity.RecordHeartbeat(ctx, "Creating pre-agg for: "+opt.TargetName)
	return nil
}

// ApplyRetireAssetActivity retires an asset
func (a *ASOActivities) ApplyRetireAssetActivity(ctx context.Context, opt aso.ASOOptimization) error {
	// Mark asset as deprecated, stop scheduling
	activity.RecordHeartbeat(ctx, "Retiring: "+opt.TargetName)
	return nil
}

// ApplyPrewarmActivity sets up pre-warm schedule
func (a *ASOActivities) ApplyPrewarmActivity(ctx context.Context, opt aso.ASOOptimization) error {
	// Update pre-agg with pre-warm schedule
	activity.RecordHeartbeat(ctx, "Setting prewarm for: "+opt.TargetName)
	return nil
}

// ValidateChangeSetActivity validates a changeset for performance impacts
func (a *ASOActivities) ValidateChangeSetActivity(ctx context.Context, changeSetID, env, tenantID string) (*aso.ASOValidationResult, error) {
	csID, err := uuid.Parse(changeSetID)
	if err != nil {
		return nil, err
	}
	return a.engine.ValidateChangeSet(ctx, csID)
}

// PostPromotionOptimizationActivity re-evaluates after promotion
func (a *ASOActivities) PostPromotionOptimizationActivity(ctx context.Context, env, tenantID string) error {
	activity.RecordHeartbeat(ctx, "Post-promotion optimization: "+tenantID)
	_, err := a.engine.EvaluateTenant(ctx, env, tenantID)
	return err
}
