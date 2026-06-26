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

// SemanticPromotionInput is the input for semantic promotion workflow
type SemanticPromotionInput struct {
	ChangeSetID string `json:"changeset_id"`
	Actor       string `json:"actor"`
}

// SemanticPromotionResult is the result of a promotion
type SemanticPromotionResult struct {
	Success       bool                     `json:"success"`
	ChangeSetID   string                   `json:"changeset_id"`
	ASOValidation *aso.ASOValidationResult `json:"aso_validation,omitempty"`
	Errors        []string                 `json:"errors,omitempty"`
}

// ============================================================================
// Workflow Activity Names
// ============================================================================

const (
	ActivityLoadChangeSet           = "LoadChangeSet"
	ActivityRunASOValidation        = "RunASOValidation"
	ActivityApplyChangeSet          = "ApplyChangeSet"
	ActivityMarkChangeSetApplied    = "MarkChangeSetApplied"
	ActivityMarkChangeSetFailed     = "MarkChangeSetFailed"
	ActivityRunASOPostPromotion     = "RunASOPostPromotion"
	ActivityNotifyPromotionComplete = "NotifyPromotionComplete"
)

// ============================================================================
// Semantic Promotion Workflow
// ============================================================================

// SemanticPromotionWorkflow orchestrates the promotion of semantic changes
func SemanticPromotionWorkflow(ctx workflow.Context, input SemanticPromotionInput) (*SemanticPromotionResult, error) {
	logger := workflow.GetLogger(ctx)
	result := &SemanticPromotionResult{
		ChangeSetID: input.ChangeSetID,
	}

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1. Load ChangeSet
	logger.Info("Loading changeset", "changeSetID", input.ChangeSetID)
	var cs aso.SemanticChangeSet
	err := workflow.ExecuteActivity(ctx, ActivityLoadChangeSet, input.ChangeSetID).Get(ctx, &cs)
	if err != nil {
		result.Errors = append(result.Errors, "Failed to load changeset: "+err.Error())
		return result, err
	}

	// 2. Run ASO Validation
	logger.Info("Running ASO validation")
	var asoResult aso.ASOValidationResult
	err = workflow.ExecuteActivity(ctx, ActivityRunASOValidation, input.ChangeSetID).Get(ctx, &asoResult)
	if err != nil {
		result.Errors = append(result.Errors, "ASO validation failed: "+err.Error())
		workflow.ExecuteActivity(ctx, ActivityMarkChangeSetFailed, input.ChangeSetID, "Validation failed")
		return result, err
	}
	result.ASOValidation = &asoResult

	// Check for blocking errors
	if len(asoResult.Errors) > 0 {
		for _, e := range asoResult.Errors {
			result.Errors = append(result.Errors, e.Message)
		}
		logger.Warn("ASO validation found errors - changeset requires manual approval")
		// Let the workflow continue but require approval in the system
	}

	// 3. If approved, apply the changeset
	if cs.Status == aso.ChangeSetApproved {
		logger.Info("Applying changeset")
		err = workflow.ExecuteActivity(ctx, ActivityApplyChangeSet, input.ChangeSetID, input.Actor).Get(ctx, nil)
		if err != nil {
			result.Errors = append(result.Errors, "Failed to apply changeset: "+err.Error())
			workflow.ExecuteActivity(ctx, ActivityMarkChangeSetFailed, input.ChangeSetID, err.Error())
			return result, err
		}

		// 4. Mark as applied
		err = workflow.ExecuteActivity(ctx, ActivityMarkChangeSetApplied, input.ChangeSetID, input.Actor).Get(ctx, nil)
		if err != nil {
			logger.Error("Failed to mark changeset as applied", "error", err)
		}

		// 5. Run ASO post-promotion optimization
		logger.Info("Running ASO post-promotion optimization")
		tenantID := ""
		if cs.TenantID != nil {
			tenantID = cs.TenantID.String()
		}
		_ = workflow.ExecuteActivity(ctx, ActivityRunASOPostPromotion, cs.TargetEnv, tenantID).Get(ctx, nil)

		result.Success = true
	} else {
		logger.Info("Changeset not approved - awaiting approval", "status", cs.Status)
	}

	// 6. Notify completion
	workflow.ExecuteActivity(ctx, ActivityNotifyPromotionComplete, result)

	return result, nil
}

// ============================================================================
// Promotion Activities
// ============================================================================

// PromotionActivities contains promotion-related activities
type PromotionActivities struct {
	promotionService aso.PromotionService
	asoEngine        aso.ASOEngine
}

// NewPromotionActivities creates promotion activities
func NewPromotionActivities(
	promotionService aso.PromotionService,
	asoEngine aso.ASOEngine,
) *PromotionActivities {
	return &PromotionActivities{
		promotionService: promotionService,
		asoEngine:        asoEngine,
	}
}

// LoadChangeSetActivity loads a changeset
func (a *PromotionActivities) LoadChangeSetActivity(ctx context.Context, csID string) (*aso.SemanticChangeSet, error) {
	activity.RecordHeartbeat(ctx, "Loading changeset: "+csID)
	id, err := uuid.Parse(csID)
	if err != nil {
		return nil, err
	}
	return a.promotionService.GetChangeSet(ctx, id)
}

// RunASOValidationActivity runs ASO validation on a changeset
func (a *PromotionActivities) RunASOValidationActivity(ctx context.Context, csID string) (*aso.ASOValidationResult, error) {
	activity.RecordHeartbeat(ctx, "Running ASO validation: "+csID)
	id, err := uuid.Parse(csID)
	if err != nil {
		return nil, err
	}
	return a.promotionService.ValidateChangeSet(ctx, id)
}

// ApplyChangeSetActivity applies a changeset
func (a *PromotionActivities) ApplyChangeSetActivity(ctx context.Context, csID, actor string) error {
	activity.RecordHeartbeat(ctx, "Applying changeset: "+csID)
	id, err := uuid.Parse(csID)
	if err != nil {
		return err
	}
	return a.promotionService.ApplyChangeSet(ctx, id, actor)
}

// MarkChangeSetAppliedActivity marks changeset as applied
func (a *PromotionActivities) MarkChangeSetAppliedActivity(ctx context.Context, csID, actor string) error {
	// This is handled by ApplyChangeSet internally
	return nil
}

// MarkChangeSetFailedActivity marks changeset as failed
func (a *PromotionActivities) MarkChangeSetFailedActivity(ctx context.Context, csID, reason string) error {
	activity.RecordHeartbeat(ctx, "Marking failed: "+csID)
	// Update changeset status to failed
	return nil
}

// RunASOPostPromotionActivity runs ASO post-promotion optimization
func (a *PromotionActivities) RunASOPostPromotionActivity(ctx context.Context, env, tenantID string) error {
	activity.RecordHeartbeat(ctx, "Running post-promotion ASO: "+tenantID)
	_, err := a.asoEngine.EvaluateTenant(ctx, env, tenantID)
	return err
}

// NotifyPromotionCompleteActivity sends notification about promotion result
func (a *PromotionActivities) NotifyPromotionCompleteActivity(ctx context.Context, result *SemanticPromotionResult) error {
	activity.RecordHeartbeat(ctx, "Sending notification")
	// Integrate with notification service
	return nil
}
