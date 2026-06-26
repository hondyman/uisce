package business_process

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

// ActivityRegistry maps activity names to their function implementations.
var ActivityRegistry = map[string]interface{}{
	"ValidateOrder":       ValidateOrderActivity,
	"RunComplianceChecks": RunComplianceChecksActivity,
	"ApprovalWorkflow":    ApprovalWorkflowActivity,
	"RouteToExecution":    RouteToExecutionActivity,
	"ConfirmSettlement":   ConfirmSettlementActivity,
	"PostJournalEntries":  PostJournalEntriesActivity,
}

// DynamicWorkflowParams encapsulates arguments for the dynamic workflow
type DynamicWorkflowParams struct {
	WorkflowDefinitionID string
	ObjectType           string
	Event                string
	ProcessName          string
	TenantID             string
	BusinessObject       GenericBusinessObject
}

// DynamicProcessWorkflow executes a business process based on a graph template.
// It supports "Hot Reload" by fetching the latest definition from the DB if not provided directly.
func DynamicProcessWorkflow(ctx workflow.Context, params DynamicWorkflowParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting DynamicProcessWorkflow", "ObjectID", params.BusinessObject.ID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var template ProcessTemplate

	// Fetch Definition (Hot Reload)
	// We prioritize ID if present, otherwise lookup by Object/Event
	if params.WorkflowDefinitionID != "" {
		err := workflow.ExecuteActivity(ctx, FetchWorkflowDefinitionActivity, params.WorkflowDefinitionID).Get(ctx, &template)
		if err != nil {
			return fmt.Errorf("failed to fetch workflow definition: %v", err)
		}
	} else if params.ObjectType != "" && params.Event != "" {
		err := workflow.ExecuteActivity(ctx, FetchWorkflowDefinitionByEventActivity, params.ObjectType, params.Event).Get(ctx, &template)
		if err != nil {
			return fmt.Errorf("failed to fetch workflow definition by event: %v", err)
		}
	} else {
		return fmt.Errorf("invalid params: must provide WorkflowDefinitionID or ObjectType/Event")
	}

	obj := params.BusinessObject

	// Determine start step (first step in list for now, or find root)
	// In a real graph, we'd find the node with no incoming edges or a specific Start node.
	// For this schema, we assume the first step is the start.
	if len(template.Steps) == 0 {
		return fmt.Errorf("no steps in process")
	}
	currentStepID := template.Steps[0].ID
	if obj.State != "" {
		// If resuming, find the step matching the state
		// This assumes State == StepID, which is a simplification
		currentStepID = obj.State
	}

	for {
		// Find current step definition
		var currentStep *Step
		for i := range template.Steps {
			if template.Steps[i].ID == currentStepID {
				currentStep = &template.Steps[i]
				break
			}
		}

		if currentStep == nil {
			return fmt.Errorf("step not found: %s", currentStepID)
		}

		logger.Info("Executing Step", "StepID", currentStep.ID, "Type", currentStep.Type)

		// Execute based on Step Type
		switch currentStep.Type {
		case StepTypeActivity:
			if err := executeActivityStep(ctx, currentStep, &obj); err != nil {
				return err
			}
		case StepTypeApproval:
			if err := executeApprovalStep(ctx, currentStep, &obj); err != nil {
				return err
			}
		case StepTypeEvent:
			// Publish event (already handled implicitly by transitions, but could be explicit)
		case StepTypeDecision:
			// Logic handled in transitions
		}

		// Determine next step
		nextStepID, err := GetNextStep(template, currentStepID)
		if err != nil {
			return err
		}

		if nextStepID == "" {
			logger.Info("Process Completed", "FinalStep", currentStepID)
			break
		}

		// Transition
		logger.Info("Transitioning", "From", currentStepID, "To", nextStepID)
		currentStepID = nextStepID
		obj.State = currentStepID

		// TODO: Persist state change
	}

	return nil
}

func executeActivityStep(ctx workflow.Context, step *Step, obj *GenericBusinessObject) error {
	// Use string-based dispatch as per Interpreter Pattern requirements
	activityName := step.ActivityRef
	if activityName == "" {
		return fmt.Errorf("activity reference is empty for step %s", step.ID)
	}

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 10,
	}

	// Apply SLA if present
	if step.SLA != "" {
		// Parse ISO 8601 duration (simplified)
		if d, err := time.ParseDuration(step.SLA); err == nil {
			ao.StartToCloseTimeout = d
		}
	}

	ctx = workflow.WithActivityOptions(ctx, ao)

	// Execute activity by string name
	// Note: The worker must register the activity with this exact name.
	return workflow.ExecuteActivity(ctx, activityName, *obj).Get(ctx, obj)
}

func executeApprovalStep(ctx workflow.Context, step *Step, obj *GenericBusinessObject) error {
	logger := workflow.GetLogger(ctx)

	// Create Human Task Record
	// Activity: CreateHumanTask
	var taskID string
	err := workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	}), CreateHumanTaskActivity, step.ID, obj.ID, step.Roles).Get(ctx, &taskID)
	if err != nil {
		return err
	}

	// Wait for signal "Approve" or "Reject"
	var signalVal string
	signalChan := workflow.GetSignalChannel(ctx, "ApprovalSignal")

	// Use workflow.Await for cleaner syntax if just waiting for condition,
	// but Select is better for timeouts + signals.

	selector := workflow.NewSelector(ctx)
	selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, more bool) {
		c.Receive(ctx, &signalVal)
	})

	// Handle SLA timeout for approval
	if step.SLA != "" {
		// Simplified parsing
		d, _ := time.ParseDuration("24h") // Default
		if parsed, err := time.ParseDuration(step.SLA); err == nil {
			d = parsed
		}

		timer := workflow.NewTimer(ctx, d)
		selector.AddFuture(timer, func(f workflow.Future) {
			logger.Info("Approval SLA Breached")
			// Handle escalation
		})
	}

	// Wait for signal or timeout
	selector.Select(ctx)

	if signalVal == "Reject" {
		// Update task status to Rejected
		_ = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
			StartToCloseTimeout: time.Minute,
		}), UpdateHumanTaskStatusActivity, taskID, "Rejected").Get(ctx, nil)
		return fmt.Errorf("approval rejected")
	}

	// Update task status to Approved
	_ = workflow.ExecuteActivity(workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	}), UpdateHumanTaskStatusActivity, taskID, "Approved").Get(ctx, nil)

	obj.Data["approved"] = true
	return nil
}

// --- Activities ---

func CreateHumanTaskActivity(ctx context.Context, stepID, entityID string, roles []string) (string, error) {
	taskID := uuid.New().String()
	fmt.Printf("Creating Human Task: %s for Entity %s (Roles: %v)\n", taskID, entityID, roles)
	// INSERT INTO human_tasks ...
	return taskID, nil
}

func UpdateHumanTaskStatusActivity(ctx context.Context, taskID, status string) error {
	fmt.Printf("Updating Human Task %s to %s\n", taskID, status)
	// UPDATE human_tasks SET status = $2 WHERE id = $1
	return nil
}

// --- BPF Activities ---

func FetchWorkflowDefinitionActivity(ctx context.Context, definitionID string) (*ProcessTemplate, error) {
	// In a real implementation, this would query the 'workflow_definitions' table
	// SELECT steps_json FROM workflow_definitions WHERE id = $1
	fmt.Printf("Fetching Workflow Definition: %s\n", definitionID)
	return &ProcessTemplate{}, nil // Return mock or empty for now
}

func FetchWorkflowDefinitionByEventActivity(ctx context.Context, objectType, event string) (*ProcessTemplate, error) {
	// SELECT steps_json FROM workflow_definitions WHERE object_type = $1 AND event = $2 AND is_active = true
	fmt.Printf("Fetching Workflow Definition for %s - %s\n", objectType, event)
	return &ProcessTemplate{}, nil
}

// --- Mock Activities ---

func ValidateOrderActivity(ctx context.Context, obj GenericBusinessObject) (*GenericBusinessObject, error) {
	obj.Data["valid"] = true
	return &obj, nil
}

func RunComplianceChecksActivity(ctx context.Context, obj GenericBusinessObject) (*GenericBusinessObject, error) {
	return &obj, nil
}

func ApprovalWorkflowActivity(ctx context.Context, obj GenericBusinessObject) (*GenericBusinessObject, error) {
	obj.Data["approved"] = true
	return &obj, nil
}

func RouteToExecutionActivity(ctx context.Context, obj GenericBusinessObject) (*GenericBusinessObject, error) {
	return &obj, nil
}

func ConfirmSettlementActivity(ctx context.Context, obj GenericBusinessObject) (*GenericBusinessObject, error) {
	return &obj, nil
}

func PostJournalEntriesActivity(ctx context.Context, obj GenericBusinessObject) (*GenericBusinessObject, error) {
	return &obj, nil
}
