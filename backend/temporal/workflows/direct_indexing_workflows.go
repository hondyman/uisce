package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// DailyHarvestScanInput defines the workflow input
type DailyHarvestScanInput struct {
	TenantID string
}

// DailyHarvestScanResult contains scan results
type DailyHarvestScanResult struct {
	AccountsScanned       int     `json:"accounts_scanned"`
	OpportunitiesDetected int     `json:"opportunities_detected"`
	TotalPotentialSavings float64 `json:"total_potential_savings"`
}

// DailyTaxLossHarvestScanWorkflow scans all accounts for tax-loss harvest opportunities
// Scheduled to run daily at 4 PM ET (market close)
func DailyTaxLossHarvestScanWorkflow(ctx workflow.Context, input DailyHarvestScanInput) (*DailyHarvestScanResult, error) {
	logger := workflow.GetLogger(ctx)

	logger.Info("Starting daily tax-loss harvest scan", "tenant_id", input.TenantID)

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute, // Give enough time for large scans
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
			InitialInterval: time.Second,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Run Python harvest engine scan
	var scanResult DailyHarvestScanResult
	err := workflow.ExecuteActivity(ctx, "RunDailyHarvestScan", input.TenantID).Get(ctx, &scanResult)
	if err != nil {
		logger.Error("Harvest scan failed", "error", err)
		return nil, err
	}

	logger.Info("Harvest scan completed",
		"accounts", scanResult.AccountsScanned,
		"opportunities", scanResult.OpportunitiesDetected,
		"potential_savings", scanResult.TotalPotentialSavings,
	)

	// Step 2: Send notifications for high-value opportunities (>$5K tax savings)
	if scanResult.TotalPotentialSavings > 5000 {
		err = workflow.ExecuteActivity(ctx, "SendHarvestNotifications", map[string]interface{}{
			"tenant_id":         input.TenantID,
			"total_savings":     scanResult.TotalPotentialSavings,
			"opportunity_count": scanResult.OpportunitiesDetected,
		}).Get(ctx, nil)

		if err != nil {
			logger.Warn("Failed to send notifications", "error", err)
			// Don't fail workflow if notifications fail
		}
	}

	return &scanResult, nil
}

// ScheduleDailyHarvestScans sets up schedules for all tenants
// Called once during system initialization
func ScheduleDailyHarvestScans(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)

	// Get all active tenants
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var tenantIDs []string
	err := workflow.ExecuteActivity(ctx, "GetActiveTenants").Get(ctx, &tenantIDs)
	if err != nil {
		return err
	}

	logger.Info("Scheduling daily harvest scans", "tenant_count", len(tenantIDs))

	// Schedule a workflow per tenant at 4 PM ET daily
	for _, tenantID := range tenantIDs {
		scheduleID := fmt.Sprintf("daily-harvest-scan-%s", tenantID)

		// Create schedule (Temporal Schedules feature)
		err = workflow.ExecuteActivity(ctx, "CreateSchedule", map[string]interface{}{
			"schedule_id": scheduleID,
			"workflow":    "DailyTaxLossHarvestScanWorkflow",
			"input": DailyHarvestScanInput{
				TenantID: tenantID,
			},
			"cron":     "0 16 * * MON-FRI", // 4 PM ET, weekdays only
			"timezone": "America/New_York",
		}).Get(ctx, nil)

		if err != nil {
			logger.Error("Failed to create schedule", "tenant_id", tenantID, "error", err)
			continue
		}

		logger.Info("Created harvest scan schedule", "tenant_id", tenantID)
	}

	return nil
}

// OnDemandHarvestScanWorkflow allows manual triggering of harvest scan for a specific account
func OnDemandHarvestScanWorkflow(ctx workflow.Context, accountID string) (*DailyHarvestScanResult, error) {
	logger := workflow.GetLogger(ctx)

	logger.Info("Starting on-demand harvest scan", "account_id", accountID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var result DailyHarvestScanResult
	err := workflow.ExecuteActivity(ctx, "RunAccountHarvestScan", accountID).Get(ctx, &result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
