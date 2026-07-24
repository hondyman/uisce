package workflows

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
)

// ============================================================================
// System Activities
// ============================================================================

// ActivityServiceCall performs a generic service call
func ActivityServiceCall(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Executing generic ServiceCall", "config", config)

	// Check if this is a mapped External Integration (from interpreter routing)
	if externalID, ok := config["external_integration"].(string); ok {
		logger.Info("Delegating ServiceCall to External Integration", "integration_queue", externalID)

		// In a real system, we'd map this generic call to ActivityCreateExternalTask
		// specific logic. For now, we simulate success.
		return map[string]interface{}{
			"service_call_status": "success",
			"external_ref":        externalID,
			"timestamp":           time.Now(),
		}, nil
	}

	return map[string]interface{}{
		"service_call_status": "success",
		"executed_at":         time.Now(),
	}, nil
}

// ActivitySemanticRollup refreshes semantic layer aggregates
func ActivitySemanticRollup(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Refreshing semantic rollup")

	// Simulate query cost
	time.Sleep(50 * time.Millisecond)

	return map[string]interface{}{
		"rollup_status":   "refreshed",
		"items_processed": 42,
	}, nil
}

// ActivityDataValidation runs policy checks
func ActivityDataValidation(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Running data validation checks")

	// Simulate validation logic
	// In reality this would call a Policy Engine (CUE/OPA)

	return map[string]interface{}{
		"validation_status": "passed",
		"policy_checks":     []string{"Compliance", "Risk", "Suitability"},
	}, nil
}

// ActivityGenerateReport generates a downloadable report document
func ActivityGenerateReport(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Generating report document")

	reportID := fmt.Sprintf("RPT-%d", time.Now().Unix())
	reportURL := fmt.Sprintf("https://internal-reports.semlayer.com/%s.pdf", reportID)

	return map[string]interface{}{
		"report_id":    reportID,
		"report_url":   reportURL,
		"generated_at": time.Now(),
	}, nil
}

// ActivityNotification sends a notification
func ActivityNotification(ctx context.Context, config map[string]interface{}, state map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Sending notification")

	// Check for recipients from routing
	if recipients, ok := config["recipients"]; ok {
		logger.Info("Targeting recipients", "count", recipients)
	}

	return map[string]interface{}{
		"notification_status": "sent",
		"channel":             "email",
	}, nil
}
