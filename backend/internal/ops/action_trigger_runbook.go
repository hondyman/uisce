package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TriggerRunbookAction executes a workflow runbook
type TriggerRunbookAction struct {
	store Store
}

// RunbookExecutionParams are parameters for triggering a runbook
type RunbookExecutionParams struct {
	RunbookID string                 `json:"runbook_id"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// RunbookExecutionResult tracks runbook execution
type RunbookExecutionResult struct {
	RunbookID      string                 `json:"runbook_id"`
	ExecutionID    string                 `json:"execution_id"`
	Status         string                 `json:"status"` // running, completed, failed
	Duration       int                    `json:"duration_ms"`
	StepsCompleted int                    `json:"steps_completed"`
	StepsFailed    int                    `json:"steps_failed"`
	Output         map[string]interface{} `json:"output,omitempty"`
	Error          string                 `json:"error,omitempty"`
}

// NewTriggerRunbookAction creates a new runbook trigger action
func NewTriggerRunbookAction(store Store) *TriggerRunbookAction {
	return &TriggerRunbookAction{store: store}
}

// ID returns the action type identifier
func (a *TriggerRunbookAction) ID() string {
	return "trigger_runbook"
}

// Name returns the human-readable action name
func (a *TriggerRunbookAction) Name() string {
	return "Trigger Runbook"
}

// Validate checks if runbook parameters are valid
func (a *TriggerRunbookAction) Validate(ctx context.Context, params json.RawMessage) error {
	var p RunbookExecutionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return fmt.Errorf("invalid runbook parameters: %w", err)
	}

	if p.RunbookID == "" {
		return fmt.Errorf("runbook_id is required")
	}

	// In production, validate against known runbooks
	// For now, any non-empty runbook_id passes validation
	return nil
}

// Execute triggers the runbook execution
// In production, this would call a Temporal/Airflow/etc workflow engine
func (a *TriggerRunbookAction) Execute(ctx context.Context, params json.RawMessage) (map[string]interface{}, error) {
	var p RunbookExecutionParams
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("invalid parameters: %w", err)
	}

	// Generate execution ID
	executionID := uuid.New().String()

	// Simulate runbook execution
	// In production: Call Temporal workflow client or Airflow API
	startTime := time.Now()

	// Simulate execution steps
	stepsCompleted := 3
	stepsFailed := 0
	duration := 2500 // ms

	// Simulate some work
	time.Sleep(time.Duration(duration) * time.Millisecond)

	result := RunbookExecutionResult{
		RunbookID:      p.RunbookID,
		ExecutionID:    executionID,
		Status:         "completed",
		Duration:       int(time.Since(startTime).Milliseconds()),
		StepsCompleted: stepsCompleted,
		StepsFailed:    stepsFailed,
		Output: map[string]interface{}{
			"status":       "success",
			"runbook":      p.RunbookID,
			"execution_id": executionID,
			"timestamp":    startTime.Format(time.RFC3339),
		},
	}

	return map[string]interface{}{
		"runbook_id":      result.RunbookID,
		"execution_id":    result.ExecutionID,
		"status":          result.Status,
		"duration_ms":     result.Duration,
		"steps_completed": result.StepsCompleted,
		"steps_failed":    result.StepsFailed,
		"output":          result.Output,
	}, nil
}

// Rollback cancels a running runbook execution
// In production, this would call workflow engine to cancel
func (a *TriggerRunbookAction) Rollback(ctx context.Context, store Store, historyID uuid.UUID) error {
	// In production: Call Temporal/Airflow to cancel execution
	// For now, just return success
	return nil
}
