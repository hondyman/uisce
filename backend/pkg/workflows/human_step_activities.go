package workflows

import (
	"encoding/json"
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// Human Step Activities - Workday-Inspired Human Tasks
// ============================================================================

// HumanStepConfig defines configuration for human workflow steps
type HumanStepConfig struct {
	StepType       string      `json:"step_type"`       // Approval, Review, ToDo, Acknowledgment
	Title          string      `json:"title"`           // Task title
	Description    string      `json:"description"`     // Task description
	Instructions   string      `json:"instructions"`    // Instructions for assignee
	RoutingRule    RoutingRule `json:"routing_rule"`    // How to assign
	SLADuration    string      `json:"sla_duration"`    // e.g., "24h", "4h"
	RequiredFields []string    `json:"required_fields"` // Fields user must complete
	AllowedActions []string    `json:"allowed_actions"` // e.g., ["approve", "reject", "request_info"]
	FormSchema     interface{} `json:"form_schema"`     // Optional form schema
	OutputVariable string      `json:"output_variable"` // Where to store result
}

// HumanStepResult holds the result of a human step
type HumanStepResult struct {
	StepType     string                 `json:"step_type"`
	Status       string                 `json:"status"` // pending, completed, expired, escalated
	Action       string                 `json:"action"` // approve, reject, acknowledge, etc.
	CompletedBy  *Assignee              `json:"completed_by"`
	CompletedAt  time.Time              `json:"completed_at"`
	FormData     map[string]interface{} `json:"form_data"` // User-entered data
	Comments     string                 `json:"comments"`
	RoutingTrace *RoutingTrace          `json:"routing_trace"`
	Assignees    []Assignee             `json:"assignees"`
	SLABreached  bool                   `json:"sla_breached"`
}

// ============================================================================
// Human Step Execution
// ============================================================================

// ExecuteHumanStep executes any human step type
func ExecuteHumanStep(
	ctx workflow.Context,
	config HumanStepConfig,
	currentState map[string]interface{},
) (*HumanStepResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing human step", "type", config.StepType, "title", config.Title)

	// Resolve routing
	routingResult, err := ResolveRouting(ctx, config.RoutingRule, currentState)
	if err != nil {
		logger.Error("Failed to resolve routing", "error", err)
		return nil, fmt.Errorf("routing failed: %w", err)
	}

	// Parse SLA
	slaDuration, _ := time.ParseDuration(config.SLADuration)
	if slaDuration == 0 {
		slaDuration = 24 * time.Hour // Default 24h SLA
	}

	// Create human task
	taskID := fmt.Sprintf("task_%s_%d", config.StepType, workflow.Now(ctx).UnixNano())

	result := &HumanStepResult{
		StepType:     config.StepType,
		Status:       "pending",
		RoutingTrace: routingResult.Trace,
		Assignees:    routingResult.Assignees,
	}

	// Register the task (via activity)
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: workflow.GetInfo(ctx).WorkflowExecutionTimeout / 10,
	}
	actx := workflow.WithActivityOptions(ctx, activityOptions)

	err = workflow.ExecuteActivity(actx, "ActivityCreateHumanTask", map[string]interface{}{
		"task_id":         taskID,
		"step_type":       config.StepType,
		"title":           config.Title,
		"description":     config.Description,
		"instructions":    config.Instructions,
		"assignees":       routingResult.Assignees,
		"required_fields": config.RequiredFields,
		"allowed_actions": config.AllowedActions,
		"form_schema":     config.FormSchema,
		"sla_deadline":    workflow.Now(ctx).Add(slaDuration),
		"workflow_id":     workflow.GetInfo(ctx).WorkflowExecution.ID,
	}).Get(ctx, nil)

	if err != nil {
		logger.Error("Failed to create human task", "error", err)
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// Wait for completion signal or SLA timeout
	completionChan := workflow.GetSignalChannel(ctx, "human_step_completed")
	timerCtx, cancelTimer := workflow.WithCancel(ctx)
	timerFuture := workflow.NewTimer(timerCtx, slaDuration)

	selector := workflow.NewSelector(ctx)

	// Handle completion signal
	selector.AddReceive(completionChan, func(c workflow.ReceiveChannel, more bool) {
		var signalData map[string]interface{}
		c.Receive(ctx, &signalData)

		result.Status = "completed"
		result.Action = getStringFromMap(signalData, "action")
		result.Comments = getStringFromMap(signalData, "comments")
		result.CompletedAt = workflow.Now(ctx)

		if formData, ok := signalData["form_data"].(map[string]interface{}); ok {
			result.FormData = formData
		}

		if completedBy, ok := signalData["completed_by"].(map[string]interface{}); ok {
			result.CompletedBy = &Assignee{
				Type:  "user",
				ID:    getStringFromMap(completedBy, "id"),
				Name:  getStringFromMap(completedBy, "name"),
				Email: getStringFromMap(completedBy, "email"),
			}
		}

		cancelTimer()
	})

	// Handle SLA timeout
	selector.AddFuture(timerFuture, func(f workflow.Future) {
		result.Status = "expired"
		result.SLABreached = true

		// Check for escalation
		shouldEscalate, escalationRule := CheckEscalation(ctx, config.RoutingRule, workflow.Now(ctx).Add(-slaDuration))
		if shouldEscalate && escalationRule != nil {
			result.Status = "escalated"
			// Re-route with escalation rule
			logger.Info("Escalating human step", "title", config.Title)
			// Could recursively call ExecuteHumanStep with new routing
		}
	})

	selector.Select(ctx)

	logger.Info("Human step completed", "status", result.Status, "action", result.Action)

	return result, nil
}

// ============================================================================
// Specific Human Step Types
// ============================================================================

// ExecuteApprovalStep executes an approval step
func ExecuteApprovalStep(
	ctx workflow.Context,
	config HumanStepConfig,
	currentState map[string]interface{},
) (*HumanStepResult, error) {
	config.StepType = "Approval"
	if len(config.AllowedActions) == 0 {
		config.AllowedActions = []string{"approve", "reject", "request_info"}
	}
	return ExecuteHumanStep(ctx, config, currentState)
}

// ExecuteReviewStep executes a review step
func ExecuteReviewStep(
	ctx workflow.Context,
	config HumanStepConfig,
	currentState map[string]interface{},
) (*HumanStepResult, error) {
	config.StepType = "Review"
	if len(config.AllowedActions) == 0 {
		config.AllowedActions = []string{"complete", "edit", "request_changes"}
	}
	return ExecuteHumanStep(ctx, config, currentState)
}

// ExecuteToDoStep executes a to-do step
func ExecuteToDoStep(
	ctx workflow.Context,
	config HumanStepConfig,
	currentState map[string]interface{},
) (*HumanStepResult, error) {
	config.StepType = "ToDo"
	if len(config.AllowedActions) == 0 {
		config.AllowedActions = []string{"complete"}
	}
	return ExecuteHumanStep(ctx, config, currentState)
}

// ExecuteAcknowledgmentStep executes an acknowledgment step
func ExecuteAcknowledgmentStep(
	ctx workflow.Context,
	config HumanStepConfig,
	currentState map[string]interface{},
) (*HumanStepResult, error) {
	config.StepType = "Acknowledgment"
	if len(config.AllowedActions) == 0 {
		config.AllowedActions = []string{"acknowledge"}
	}
	return ExecuteHumanStep(ctx, config, currentState)
}

// ============================================================================
// Parser
// ============================================================================

// ParseHumanStepConfig extracts config from node
func ParseHumanStepConfig(config map[string]interface{}) (*HumanStepConfig, error) {
	data, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var cfg HumanStepConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse human step config: %w", err)
	}

	if cfg.Title == "" {
		cfg.Title = "Task"
	}
	if cfg.SLADuration == "" {
		cfg.SLADuration = "24h"
	}

	return &cfg, nil
}

// ============================================================================
// Helpers
// ============================================================================

func getStringFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}
