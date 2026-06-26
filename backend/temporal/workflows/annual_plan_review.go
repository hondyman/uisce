package workflows

import (
	"time"

	"github.com/hondyman/semlayer/backend/internal/wealth"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// AnnualPlanReviewWorkflow orchestrates annual estate plan reviews
func AnnualPlanReviewWorkflow(ctx workflow.Context, input AnnualPlanReviewInput) (*AnnualPlanReviewResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting annual plan review workflow", "familyID", input.FamilyID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 3 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	result := &AnnualPlanReviewResult{
		FamilyID: input.FamilyID,
		Changes:  []string{},
	}

	// Step 1: Get current family profile and existing scenarios
	var profile wealth.FamilyProfile
	err := workflow.ExecuteActivity(ctx, "GetFamilyProfileActivity", input.FamilyID).Get(ctx, &profile)
	if err != nil {
		return nil, err
	}

	var existingScenarios []wealth.EstatePlanScenario
	err = workflow.ExecuteActivity(ctx, "GetExistingScenariosActivity", input.FamilyID).Get(ctx, &existingScenarios)
	if err != nil {
		return nil, err
	}

	// Step 2: Check for tax law changes
	var taxChanges []wealth.TaxLawChange
	err = workflow.ExecuteActivity(ctx, "CheckTaxLawChangesActivity", wealth.CheckTaxLawChangesInput{
		SinceDate: input.LastReviewDate,
	}).Get(ctx, &taxChanges)
	if err != nil {
		return nil, err
	}

	if len(taxChanges) > 0 {
		logger.Info("Tax law changes detected", "count", len(taxChanges))
		result.TaxLawChanges = taxChanges
		result.RequiresRecalculation = true
		for _, change := range taxChanges {
			result.Changes = append(result.Changes, change.Description)
		}
	}

	// Step 3: Check for family changes (new members, deaths, marriages)
	var familyChanges []string
	err = workflow.ExecuteActivity(ctx, "DetectFamilyChangesActivity", wealth.DetectFamilyChangesInput{
		FamilyID:  input.FamilyID,
		SinceDate: input.LastReviewDate,
	}).Get(ctx, &familyChanges)
	if err != nil {
		return nil, err
	}

	if len(familyChanges) > 0 {
		logger.Info("Family changes detected", "count", len(familyChanges))
		result.RequiresRecalculation = true
		result.Changes = append(result.Changes, familyChanges...)
	}

	// Step 4: Check for asset value changes (>10% variance)
	var assetChanges interface{}
	err = workflow.ExecuteActivity(ctx, "DetectAssetChangesActivity", wealth.DetectAssetChangesInput{
		FamilyID:     input.FamilyID,
		ThresholdPct: 10.0,
	}).Get(ctx, &assetChanges)
	if err != nil {
		return nil, err
	}

	// Step 5: If changes detected, regenerate scenarios
	if result.RequiresRecalculation {
		logger.Info("Triggering plan regeneration due to changes")

		// Trigger child workflow for plan generation
		childWorkflowOptions := workflow.ChildWorkflowOptions{
			WorkflowID: "estate-plan-gen-" + input.FamilyID + "-" + time.Now().Format("20060102"),
		}
		childCtx := workflow.WithChildOptions(ctx, childWorkflowOptions)

		var planResult wealth.EstatePlanGenerationResult
		// Assuming EstatePlanGenerationWorkflow is defined elsewhere or using string
		err = workflow.ExecuteChildWorkflow(childCtx, "EstatePlanGenerationWorkflow", wealth.EstatePlanGenerationInput{
			FamilyID:           input.FamilyID,
			MaxScenarios:       15,
			GenerateNarratives: true,
		}).Get(ctx, &planResult)

		if err != nil {
			logger.Error("Plan regeneration failed", "error", err)
			return nil, err
		}

		result.ScenariosRegenerated = planResult.ScenariosGenerated
	} else {
		logger.Info("No significant changes detected - plan remains valid")
		result.Changes = append(result.Changes, "No significant changes detected")
	}

	// Step 6: Send review notification to advisor
	err = workflow.ExecuteActivity(ctx, "SendReviewNotificationActivity", wealth.SendReviewNotificationInput{
		FamilyID:             input.FamilyID,
		Changes:              result.Changes,
		RequiresAction:       result.RequiresRecalculation,
		ScenariosRegenerated: result.ScenariosRegenerated,
	}).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to send notification", "error", err)
	}

	// Step 7: Schedule next review (1 year from now)
	err = workflow.ExecuteActivity(ctx, "ScheduleNextReviewActivity", wealth.ScheduleNextReviewInput{
		FamilyID:   input.FamilyID,
		ReviewDate: time.Now().AddDate(1, 0, 0),
	}).Get(ctx, nil)
	if err != nil {
		logger.Warn("Failed to schedule next review", "error", err)
	}

	result.Message = "Annual review complete"
	logger.Info("Annual plan review workflow complete", "requiresRecalculation", result.RequiresRecalculation)

	return result, nil
}

// AnnualPlanReviewInput is the workflow input
type AnnualPlanReviewInput struct {
	FamilyID       string    `json:"family_id"`
	LastReviewDate time.Time `json:"last_review_date"`
}

// AnnualPlanReviewResult is the workflow result
type AnnualPlanReviewResult struct {
	FamilyID              string                `json:"family_id"`
	RequiresRecalculation bool                  `json:"requires_recalculation"`
	Changes               []string              `json:"changes"`
	TaxLawChanges         []wealth.TaxLawChange `json:"tax_law_changes,omitempty"`
	ScenariosRegenerated  int                   `json:"scenarios_regenerated"`
	Message               string                `json:"message"`
}
