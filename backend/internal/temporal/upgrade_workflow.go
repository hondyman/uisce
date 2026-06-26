package temporal

import (
	"time"

	"go.temporal.io/sdk/workflow"

	"github.com/hondyman/semlayer/backend/internal/metadata"
)

// UpgradeRequest defines the input for the upgrade workflow
type UpgradeRequest struct {
	NewCoreVersion string   `json:"new_core_version"`
	TargetTenants  []string `json:"target_tenants"` // Empty for all
}

// UpgradeResult defines the output of the upgrade workflow
type UpgradeResult struct {
	SuccessCount int      `json:"success_count"`
	FailureCount int      `json:"failure_count"`
	ReportID     string   `json:"report_id"`
}

// UpgradePipeline orchestrates the core metadata upgrade process
func UpgradePipeline(ctx workflow.Context, req UpgradeRequest) (*UpgradeResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting Metadata Upgrade Pipeline", "Version", req.NewCoreVersion)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1. Load New Core Metadata (Activity)
	var newCoreBOs []metadata.BusinessObject
	err := workflow.ExecuteActivity(ctx, "LoadCoreMetadataActivity", req.NewCoreVersion).Get(ctx, &newCoreBOs)
	if err != nil {
		return nil, err
	}

	// 2. Load Old Core Metadata (Activity)
	var oldCoreBOs []metadata.BusinessObject
	err = workflow.ExecuteActivity(ctx, "LoadCoreMetadataActivity", "CURRENT").Get(ctx, &oldCoreBOs)
	if err != nil {
		return nil, err
	}

	// 3. Iterate over Tenants
	successCount := 0
	failureCount := 0

	// In a real system, we'd fetch the list of tenants
	tenants := req.TargetTenants
	if len(tenants) == 0 {
		tenants = []string{"tenant-1", "tenant-2"} // Mock list
	}

	for _, tenantID := range tenants {
		// Execute Child Workflow per Tenant for isolation
		cwo := workflow.ChildWorkflowOptions{
			WorkflowID: "UpgradeTenant_" + tenantID + "_" + req.NewCoreVersion,
		}
		ctxChild := workflow.WithChildOptions(ctx, cwo)

		var result string
		err := workflow.ExecuteChildWorkflow(ctxChild, TenantUpgradeWorkflow, tenantID, oldCoreBOs, newCoreBOs).Get(ctx, &result)
		
		if err != nil {
			logger.Error("Tenant Upgrade Failed", "Tenant", tenantID, "Error", err)
			failureCount++
		} else {
			logger.Info("Tenant Upgrade Success", "Tenant", tenantID)
			successCount++
		}
	}

	return &UpgradeResult{
		SuccessCount: successCount,
		FailureCount: failureCount,
		ReportID:     "report-" + req.NewCoreVersion,
	}, nil
}

// TenantUpgradeWorkflow handles the upgrade for a single tenant
func TenantUpgradeWorkflow(ctx workflow.Context, tenantID string, oldCore, newCore []metadata.BusinessObject) (string, error) {
	logger := workflow.GetLogger(ctx)
	
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1. Load Tenant Overlay
	// Mock: Load overlay for first BO
	// In reality, loop through all objects
	
	// 2. Rebase Overlay (Activity)
	var rebaseResult metadata.RebaseResult
	err := workflow.ExecuteActivity(ctx, "RebaseOverlayActivity", tenantID, oldCore[0], newCore[0]).Get(ctx, &rebaseResult)
	if err != nil {
		return "", err
	}

	if !rebaseResult.Success {
		// 3a. Handle Conflicts (Human Task)
		logger.Warn("Conflicts Detected", "Tenant", tenantID, "Conflicts", len(rebaseResult.Conflicts))
		// Create Human Task / Notification
		// Wait for approval signal...
		return "Conflicts Flagged", nil
	}

	// 3b. Deploy to Sandbox (Activity)
	err = workflow.ExecuteActivity(ctx, "DeployToSandboxActivity", tenantID, rebaseResult.RebasedBO).Get(ctx, nil)
	if err != nil {
		return "", err
	}

	// 4. Run Regression Tests (Activity)
	err = workflow.ExecuteActivity(ctx, "RunRegressionTestsActivity", tenantID).Get(ctx, nil)
	if err != nil {
		return "", err
	}

	// 5. Publish to Production (Activity)
	// In reality, wait for Maker/Checker signal here
	err = workflow.ExecuteActivity(ctx, "PublishOverlayActivity", tenantID, rebaseResult.RebasedBO).Get(ctx, nil)
	if err != nil {
		return "", err
	}

	// 6. Log Audit (Activity)
	_ = workflow.ExecuteActivity(ctx, "LogAuditActivity", tenantID, "UpgradeSuccess").Get(ctx, nil)

	return "Success", nil
}
