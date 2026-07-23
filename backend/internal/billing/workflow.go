package billing

import (
	"context"
	"time"

	"go.temporal.io/sdk/workflow"
)

// Workflow Definitions
func MonthlyBillingWorkflow(ctx workflow.Context, billingMonth time.Time) error {
	logger := workflow.GetLogger(ctx)
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 10,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1. Get Billable Clients (Mocked list for now)
	// In real implementation, this would be an activity
	clientIDs := []string{"mock-client-1", "mock-client-2"}

	logger.Info("Starting billing cycle", "Month", billingMonth, "ClientCount", len(clientIDs))

	// 2. Calculate Fees Parallel
	futures := []workflow.Future{}
	for _, clientID := range clientIDs {
		future := workflow.ExecuteActivity(ctx, CalculateFeeActivity, clientID, billingMonth)
		futures = append(futures, future)
	}

	// 3. Wait for results
	results := make([]FeeCalculation, 0, len(clientIDs))
	for _, f := range futures {
		var calc FeeCalculation
		if err := f.Get(ctx, &calc); err != nil {
			logger.Error("Failed to calculate fee", "Error", err)
			continue
		}
		results = append(results, calc)
	}

	// 4. Generate Invoices (Mocked)
	// workflow.ExecuteActivity(ctx, GenerateInvoicesActivity, results)

	logger.Info("Billing cycle completed", "Processed", len(results))
	return nil
}

// Activity Definitions
func CalculateFeeActivity(ctx context.Context, clientID string, month time.Time) (*FeeCalculation, error) {
	// In a real app, we'd inject the Service into the Activity struct
	// For now, we'll just return a mock or error since we can't easily inject the DB here without more setup
	// This is a placeholder to satisfy the workflow definition
	return &FeeCalculation{
		Status:   "DRAFT",
		TotalFee: 1000.0,
	}, nil
}
