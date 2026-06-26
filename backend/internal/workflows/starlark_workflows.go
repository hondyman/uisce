package workflows

import (
	"context"
	"time"

	"github.com/hondyman/semlayer/backend/internal/services"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// STARLARK EXPRESSION ACTIVITIES
// Temporal activities for dynamic rule evaluation
// ============================================================================

// StarlarkActivities provides Temporal activities for expression evaluation
type StarlarkActivities struct {
	engine *services.StarlarkEngine
}

// NewStarlarkActivities creates new Starlark activities
func NewStarlarkActivities(engine *services.StarlarkEngine) *StarlarkActivities {
	return &StarlarkActivities{engine: engine}
}

// EvaluateConditionInput input for condition evaluation
type EvaluateConditionInput struct {
	Script string                 `json:"script"`
	Data   map[string]interface{} `json:"data"`
}

// EvaluateConditionOutput output from condition evaluation
type EvaluateConditionOutput struct {
	Action string `json:"action"`
}

// EvaluateConditionActivity evaluates a Starlark condition and returns action
func (a *StarlarkActivities) EvaluateConditionActivity(ctx context.Context, input EvaluateConditionInput) (*EvaluateConditionOutput, error) {
	action, err := a.engine.EvaluateCondition(ctx, input.Script, input.Data)
	if err != nil {
		return nil, err
	}
	return &EvaluateConditionOutput{Action: action}, nil
}

// ValidateDataInput input for validation
type ValidateDataInput struct {
	Script string                 `json:"script"`
	Data   map[string]interface{} `json:"data"`
}

// ValidateDataOutput output from validation
type ValidateDataOutput struct {
	IsValid  bool   `json:"is_valid"`
	Message  string `json:"message,omitempty"`
	Severity string `json:"severity"`
}

// ValidateDataActivity validates data using a Starlark expression
func (a *StarlarkActivities) ValidateDataActivity(ctx context.Context, input ValidateDataInput) (*ValidateDataOutput, error) {
	result, err := a.engine.EvaluateValidation(ctx, input.Script, input.Data)
	if err != nil {
		return &ValidateDataOutput{IsValid: false, Message: err.Error(), Severity: "error"}, nil
	}
	return &ValidateDataOutput{
		IsValid:  result.IsValid,
		Message:  result.Message,
		Severity: result.Severity,
	}, nil
}

// CalculateFieldInput input for calculation
type CalculateFieldInput struct {
	Script string                 `json:"script"`
	Data   map[string]interface{} `json:"data"`
}

// CalculateFieldOutput output from calculation
type CalculateFieldOutput struct {
	Result interface{} `json:"result"`
}

// CalculateFieldActivity calculates a field value using Starlark
func (a *StarlarkActivities) CalculateFieldActivity(ctx context.Context, input CalculateFieldInput) (*CalculateFieldOutput, error) {
	result, err := a.engine.EvaluateCalculation(ctx, input.Script, input.Data)
	if err != nil {
		return nil, err
	}
	return &CalculateFieldOutput{Result: result}, nil
}

// ============================================================================
// DSL INTERPRETER WORKFLOW
// Generic workflow that executes business logic from Starlark scripts
// ============================================================================

// DSLWorkflowInput input for DSL interpreter workflow
type DSLWorkflowInput struct {
	ProcessID   string                 `json:"process_id"`
	Steps       []DSLStep              `json:"steps"`
	InitialData map[string]interface{} `json:"initial_data"`
	TenantID    string                 `json:"tenant_id"`
}

// DSLStep represents a step in the DSL workflow
type DSLStep struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"` // condition, validation, calculation, activity
	Script       string `json:"script,omitempty"`
	ActivityName string `json:"activity_name,omitempty"`
	OnSuccess    string `json:"on_success,omitempty"` // next step ID
	OnFailure    string `json:"on_failure,omitempty"` // step ID on failure
}

// DSLWorkflowOutput output from DSL workflow
type DSLWorkflowOutput struct {
	ProcessID string                 `json:"process_id"`
	Status    string                 `json:"status"`
	Results   map[string]interface{} `json:"results"`
}

// DSLInterpreterWorkflow executes a series of Starlark-defined steps
func DSLInterpreterWorkflow(ctx workflow.Context, input DSLWorkflowInput) (*DSLWorkflowOutput, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("DSL Interpreter Workflow started", "process_id", input.ProcessID)

	// Activity options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Track results
	results := make(map[string]interface{})
	currentData := input.InitialData

	// Step map for routing
	stepMap := make(map[string]DSLStep)
	for _, step := range input.Steps {
		stepMap[step.ID] = step
	}

	// Execute steps
	currentStepID := input.Steps[0].ID
	for currentStepID != "" {
		step, exists := stepMap[currentStepID]
		if !exists {
			break
		}

		logger.Info("Executing step", "step_id", step.ID, "step_name", step.Name, "type", step.Type)

		var nextStep string
		var stepResult interface{}

		switch step.Type {
		case "condition":
			var output EvaluateConditionOutput
			err := workflow.ExecuteActivity(ctx, "EvaluateConditionActivity", EvaluateConditionInput{
				Script: step.Script,
				Data:   currentData,
			}).Get(ctx, &output)
			if err != nil {
				return nil, err
			}
			stepResult = output.Action
			// Use action as routing
			if output.Action == "success" || output.Action == "true" {
				nextStep = step.OnSuccess
			} else {
				nextStep = step.OnFailure
			}

		case "validation":
			var output ValidateDataOutput
			err := workflow.ExecuteActivity(ctx, "ValidateDataActivity", ValidateDataInput{
				Script: step.Script,
				Data:   currentData,
			}).Get(ctx, &output)
			if err != nil {
				return nil, err
			}
			stepResult = output
			if output.IsValid {
				nextStep = step.OnSuccess
			} else {
				nextStep = step.OnFailure
			}

		case "calculation":
			var output CalculateFieldOutput
			err := workflow.ExecuteActivity(ctx, "CalculateFieldActivity", CalculateFieldInput{
				Script: step.Script,
				Data:   currentData,
			}).Get(ctx, &output)
			if err != nil {
				return nil, err
			}
			stepResult = output.Result
			// Add result to data for next steps
			currentData[step.ID] = output.Result
			nextStep = step.OnSuccess

		case "activity":
			// Execute named activity
			var result interface{}
			err := workflow.ExecuteActivity(ctx, step.ActivityName, currentData).Get(ctx, &result)
			if err != nil {
				if step.OnFailure != "" {
					nextStep = step.OnFailure
				} else {
					return nil, err
				}
			} else {
				stepResult = result
				nextStep = step.OnSuccess
			}

		default:
			nextStep = step.OnSuccess
		}

		results[step.ID] = stepResult
		currentStepID = nextStep
	}

	return &DSLWorkflowOutput{
		ProcessID: input.ProcessID,
		Status:    "completed",
		Results:   results,
	}, nil
}
