package workflows

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/activity"
)

// ============================================================================
// External System Activities
// ============================================================================

// ExternalTaskConfig defines input for creating an external task
type ExternalTaskConfig struct {
	System         string                 `json:"system"`          // Salesforce, ServiceNow, Jira
	Action         IntegrationAction      `json:"action"`          // create_case, create_incident, etc.
	Payload        map[string]interface{} `json:"payload"`         // Business data
	RoutingResult  *RoutingResult         `json:"_routing_result"` // Injected by interpreter
	TimeoutSeconds int                    `json:"timeout_seconds"`
}

// ExternalTaskResult defines output from an external task
type ExternalTaskResult struct {
	ExternalID     string                 `json:"external_id"`
	Status         string                 `json:"status"`
	SystemResponse map[string]interface{} `json:"system_response"`
	TaskURL        string                 `json:"task_url"`
}

// ActivityCreateExternalTask contacts the Unified Integration Microservice
func ActivityCreateExternalTask(ctx context.Context, config ExternalTaskConfig, state map[string]interface{}) (*ExternalTaskResult, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Creating external task", "system", config.System, "action", config.Action)

	// In a real implementation, this would make an HTTP/gRPC call to the integration service.
	// We will simulate the interaction and return a deterministic mock response.

	// Simulate latency
	time.Sleep(100 * time.Millisecond)

	// CHECK: Global Dry Run Flag
	if dryRun, ok := state["dry_run"].(bool); ok && dryRun {
		logger.Info("SIMULATION MODE: Skipping actual external task creation")
		return &ExternalTaskResult{
			ExternalID: "SIMULATED-ID-" + config.System,
			Status:     "simulated",
			SystemResponse: map[string]interface{}{
				"message": "Simulated creation in " + config.System,
				"mode":    "dry_run",
			},
			TaskURL: "#simulated",
		}, nil
	}

	// Generate a simulated ID based on system
	mockID := fmt.Sprintf("MOCK-%s-%d", config.System, time.Now().Unix()%10000)

	// If routing result provided, use it for queue assignment details in payload logging
	if config.RoutingResult != nil && len(config.RoutingResult.Assignees) > 0 {
		logger.Info("Routed to external queue", "queue", config.RoutingResult.Assignees[0].ID)
	}

	result := &ExternalTaskResult{
		ExternalID: mockID,
		Status:     "created",
		SystemResponse: map[string]interface{}{
			"message": "Successfully created task in " + config.System,
			"code":    201,
		},
		TaskURL: fmt.Sprintf("https://%s.com/view/%s", config.System, mockID),
	}

	return result, nil
}

// ActivityUpdateExternalTask updates an existing external task
func ActivityUpdateExternalTask(ctx context.Context, taskID string, updates map[string]interface{}) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Updating external task", "taskID", taskID)
	return nil
}

// ActivityWaitForExternalCallback waits for a webhook/signal from the external system
func ActivityWaitForExternalCallback(ctx context.Context, workflowID, stepID string) (string, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("Waiting for external callback...", "workflowID", workflowID, "stepID", stepID)

	// In a real scenario, this might involve polling or setting up a signal channel listener.
	// For simulation: return "approved" immediately or after short sleep.
	time.Sleep(500 * time.Millisecond)

	return "approved", nil
}

// ActivityCloseExternalTask closes a task in the external system
func ActivityCloseExternalTask(ctx context.Context, taskID string) error {
	logger := activity.GetLogger(ctx)
	logger.Info("Closing external task", "taskID", taskID)
	return nil
}
