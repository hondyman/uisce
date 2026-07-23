package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// BPStep represents a single step in a business process (workflow-local view)
type BPStep struct {
	StepID        string                 `json:"step_id"`
	StepName      string                 `json:"step_name"`
	StepType      string                 `json:"step_type"`
	StepOrder     int                    `json:"step_order"`
	DurationHours int                    `json:"duration_hours"`
	AssigneeRole  string                 `json:"assignee_role"`
	Config        map[string]interface{} `json:"config"`
}

// BPWorkflowInput is the input to DynamicBPWorkflow
type BPWorkflowInput struct {
	ProcessID   string                 `json:"process_id"`
	TenantID    string                 `json:"tenant_id"`
	TriggerName string                 `json:"trigger_name"`
	EventData   map[string]interface{} `json:"event_data"`
	Entity      string                 `json:"entity"`
	EntityID    string                 `json:"entity_id"`
}

// DynamicBPWorkflow executes a business process by orchestrating steps through Temporal activities
func DynamicBPWorkflow(ctx workflow.Context, input BPWorkflowInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("🎬 Starting DynamicBPWorkflow", "ProcessID", input.ProcessID, "Trigger", input.TriggerName)

	// Set activity options
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Step 1: Load BP definition and steps from database
	var steps []BPStep
	err := workflow.ExecuteActivity(ctx, (*Activities).LoadBPStepsActivity, input.ProcessID, input.TenantID).Get(ctx, &steps)
	if err != nil {
		logger.Error("Failed to load BP steps", "Error", err)
		return fmt.Errorf("load BP steps failed: %w", err)
	}

	if len(steps) == 0 {
		logger.Warn("No steps found for process", "ProcessID", input.ProcessID)
		return nil
	}

	logger.Info("📋 Loaded BP steps", "Count", len(steps))

	// Create escalation signal channel
	escalationCh := workflow.GetSignalChannel(ctx, "escalate")

	// Step 2: Execute each step in sequence
	for i, step := range steps {
		logger.Info(fmt.Sprintf("▶️  Executing Step %d/%d: %s (%s)", i+1, len(steps), step.StepName, step.StepType))

		// Create cancellation context for this step (in case of escalation)
		stepCtx, cancel := workflow.WithCancel(ctx)
		stepResult := make(map[string]interface{})

		// Execute the appropriate activity based on step type
		var stepErr error
		switch step.StepType {
		case "data_entry":
			stepErr = workflow.ExecuteActivity(stepCtx, (*Activities).DataEntryActivity, step, input.EventData).Get(stepCtx, &stepResult)

		case "validate":
			stepErr = workflow.ExecuteActivity(stepCtx, (*Activities).ValidationActivity, step, input.EventData).Get(stepCtx, &stepResult)

		case "approve":
			stepErr = workflow.ExecuteActivity(stepCtx, (*Activities).ApprovalActivity, step, input.EventData).Get(stepCtx, &stepResult)

		case "notify_email":
			stepErr = workflow.ExecuteActivity(stepCtx, (*Activities).EmailNotificationActivity, step, input.EventData).Get(stepCtx, &stepResult)

		case "notify_slack":
			stepErr = workflow.ExecuteActivity(stepCtx, (*Activities).SlackNotificationActivity, step, input.EventData).Get(stepCtx, &stepResult)

		default:
			stepErr = workflow.ExecuteActivity(stepCtx, (*Activities).GenericStepActivity, step, input.EventData).Get(stepCtx, &stepResult)
		}

		if stepErr != nil {
			logger.Error("Step execution failed", "Step", step.StepName, "Error", stepErr)
			cancel()
			// Continue to next step rather than failing entire workflow
			// In production, you might want to fail or trigger escalation
			continue
		}

		logger.Info("✅ Step completed", "Step", step.StepName)

		// Merge step results back into event data for next steps
		for k, v := range stepResult {
			input.EventData[k] = v
		}

		// Apply step duration (how long this step is expected to take)
		if step.DurationHours > 0 {
			logger.Info(fmt.Sprintf("⏳ Step duration: %d hour(s), creating timer...", step.DurationHours))

			// Create a selector to handle both duration timeout and escalation signal
			durationTimer := workflow.NewTimer(stepCtx, time.Duration(step.DurationHours)*time.Hour)
			selector := workflow.NewSelector(stepCtx)
			escalated := false

			// Listen for escalation signal
			selector.AddReceive(escalationCh, func(c workflow.ReceiveChannel, more bool) {
				var escSignal map[string]interface{}
				c.Receive(stepCtx, &escSignal)
				logger.Warn("Escalation signal received", "Step", step.StepName)

				// Execute escalation activity
				_ = workflow.ExecuteActivity(stepCtx, (*Activities).EscalateStepActivity, step, escSignal).Get(stepCtx, nil)
				escalated = true
				cancel() // Cancel duration timer if escalation happens
			})

			// Timer for automatic escalation if duration exceeded
			selector.AddFuture(durationTimer, func(f workflow.Future) {
				if !escalated {
					logger.Info("⏰ Duration timeout, executing auto-escalation", "Step", step.StepName)
					_ = workflow.ExecuteActivity(stepCtx, (*Activities).AutoEscalateActivity, step, input.EventData).Get(stepCtx, nil)
				}
			})

			selector.Select(stepCtx)
		}

		cancel()
	}

	logger.Info("🎉 DynamicBPWorkflow completed successfully", "ProcessID", input.ProcessID)
	return nil
}
