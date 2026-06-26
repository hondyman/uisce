package review

import (
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/internal/lineage"
	"github.com/hondyman/semlayer/backend/internal/semantic"
	"go.temporal.io/sdk/workflow"
)

// ChangeReviewWorkflow orchestrates the review analysis
func ChangeReviewWorkflow(ctx workflow.Context, changeSetID uuid.UUID) (*ChangeReview, error) {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute, // Adjust as needed
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var activities *ReviewActivities // Used for name resolution

	// 1. Compute Semantic Diff
	var diffs map[string]semantic.SemanticDiffDTO
	err := workflow.ExecuteActivity(ctx, activities.ComputeSemanticDiffActivity, changeSetID).Get(ctx, &diffs)
	if err != nil {
		return nil, err
	}

	// 2. Compute Lineage Impact
	var impacts map[string]*lineage.ImpactReport
	err = workflow.ExecuteActivity(ctx, activities.ComputeLineageImpactActivity, changeSetID).Get(ctx, &impacts)
	if err != nil {
		return nil, err
	}

	// 3. Run Semantic Tests
	var testResults []semantic.TestResult
	err = workflow.ExecuteActivity(ctx, activities.RunSemanticTestsActivity, impacts).Get(ctx, &testResults)
	if err != nil {
		return nil, err
	}

	// 4. Save Review
	var review *ChangeReview
	err = workflow.ExecuteActivity(ctx, activities.SaveChangeReviewActivity, changeSetID, diffs, impacts, testResults).Get(ctx, &review)
	if err != nil {
		return nil, err
	}

	return review, nil
}

// PromoteChangeSetWorkflow orchestrates the promotion process
func PromoteChangeSetWorkflow(ctx workflow.Context, changeSetID uuid.UUID) error {
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var activities *ReviewActivities

	// 1. Apply Change Set
	err := workflow.ExecuteActivity(ctx, activities.ApplyChangeSetActivity, changeSetID).Get(ctx, nil)
	if err != nil {
		return err
	}

	// 2. Rebuild Lineage
	err = workflow.ExecuteActivity(ctx, activities.RebuildLineageForChangeSetActivity, changeSetID).Get(ctx, nil)
	if err != nil {
		return err
	}

	// 3. Invalidate ASO
	err = workflow.ExecuteActivity(ctx, activities.InvalidateASOActivity, changeSetID).Get(ctx, nil)
	if err != nil {
		return err
	}

	return nil
}
