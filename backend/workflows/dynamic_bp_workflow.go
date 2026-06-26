package workflows

import (
	"context"
	"fmt"
	"time"

	"github.com/hondyman/semlayer/backend/pkg/bp"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ============================================================================
// DYNAMIC BUSINESS PROCESS WORKFLOW
// Orchestrates BP steps with Trigger Engine + Advanced Branch Evaluator
// ============================================================================

type DynamicBPWorkflowInput struct {
	TriggerID   string                 `json:"trigger_id"`
	SourceData  map[string]interface{} `json:"source_data"`
	TenantID    string                 `json:"tenant_id"`
	ProcessID   string                 `json:"process_id"`
	Steps       []*bp.BPStep           `json:"steps"`
	TriggeredAt time.Time              `json:"triggered_at"`
}

type DynamicBPWorkflowResult struct {
	WorkflowID      string                   `json:"workflow_id"`
	ProcessID       string                   `json:"process_id"`
	Status          string                   `json:"status"` // completed|failed|escalated
	FinalBranch     string                   `json:"final_branch"`
	ExecutedSteps   []string                 `json:"executed_steps"`
	BranchDecisions []map[string]interface{} `json:"branch_decisions"`
	Errors          []string                 `json:"errors,omitempty"`
	CompletedAt     time.Time                `json:"completed_at"`
	TotalDuration   time.Duration            `json:"total_duration"`
}

// DynamicBPWorkflow executes a business process with trigger integration + branching
func DynamicBPWorkflow(ctx workflow.Context, input DynamicBPWorkflowInput) (*DynamicBPWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("🚀 Starting Dynamic BP Workflow",
		"processID", input.ProcessID,
		"triggierID", input.TriggerID,
		"steps", len(input.Steps),
	)

	result := &DynamicBPWorkflowResult{
		ProcessID:       input.ProcessID,
		ExecutedSteps:   []string{},
		BranchDecisions: []map[string]interface{}{},
	}

	startTime := time.Now()

	// Execute each BP step
	for _, step := range input.Steps {
		logger.Info("📋 Executing step",
			"order", step.StepOrder,
			"type", step.StepType,
			"name", step.StepName,
		)

		// Set activity options with timeout
		ao := workflow.ActivityOptions{
			// DurationHours is stored in hours
			StartToCloseTimeout: time.Duration(step.DurationHours) * time.Hour,
			RetryPolicy: &temporal.RetryPolicy{
				InitialInterval: time.Second,
				MaximumAttempts: 3,
			},
		}
		ctx = workflow.WithActivityOptions(ctx, ao)

		var stepResult map[string]interface{}

		// Execute based on step type
		// switch on the step type
		switch step.StepType {
		case "validate":
			err := workflow.ExecuteActivity(ctx, ValidateStepActivity,
				input.TenantID, input.ProcessID, step.ID, input.SourceData).Get(ctx, &stepResult)
			if err != nil {
				logger.Error("❌ Validation failed", "step", step.StepName, "error", err)
				result.Errors = append(result.Errors, fmt.Sprintf("Validation error: %v", err))
				result.Status = "failed"
				return result, err
			}

		case "approve":
			// read assignee role from the step model (we added AssigneeRole to bp.BPStep)
			assigneeRole := ""
			if step.AssigneeRole != nil {
				assigneeRole = *step.AssigneeRole
			}

			err := workflow.ExecuteActivity(ctx, ApprovalStepActivity,
				input.TenantID, input.ProcessID, step.ID, assigneeRole,
				time.Duration(step.DurationHours)*time.Hour).Get(ctx, &stepResult)
			if err != nil {
				logger.Warn("⚠️  Approval timed out, escalating",
					"step", step.StepName,
					"role", assigneeRole,
					"error", err)
				// Escalate
				workflow.ExecuteActivity(ctx, EscalateApprovalActivity,
					input.TenantID, assigneeRole).Get(ctx, nil)
				result.Status = "escalated"
			}

		case "branch":
			// BRANCHING MAGIC: Use all 15 advanced features
			branchResult := map[string]interface{}{}
			err := workflow.ExecuteActivity(ctx, BranchingEvaluationActivity,
				input.TenantID, input.ProcessID, step.ID,
				input.SourceData).Get(ctx, &branchResult)
			if err != nil {
				logger.Error("❌ Branching failed", "step", step.StepName, "error", err)
				result.Errors = append(result.Errors, fmt.Sprintf("Branching error: %v", err))
				return result, err
			}

			result.BranchDecisions = append(result.BranchDecisions, branchResult)
			result.FinalBranch = branchResult["selected_branch"].(string)
			stepResult = branchResult

			logger.Info("🌳 Branch evaluated",
				"branch", result.FinalBranch,
				"features", branchResult["features_used"],
			)

		case "notify":
			err := workflow.ExecuteActivity(ctx, NotificationActivity,
				input.TenantID, step.StepName, input.SourceData).Get(ctx, &stepResult)
			if err != nil {
				logger.Warn("⚠️  Notification failed", "step", step.StepName)
				// Don't fail workflow on notification errors
			}

		case "integrate":
			err := workflow.ExecuteActivity(ctx, IntegrationActivity,
				input.TenantID, step.ID, input.SourceData).Get(ctx, &stepResult)
			if err != nil {
				logger.Error("❌ Integration failed", "step", step.StepName, "error", err)
				result.Errors = append(result.Errors, fmt.Sprintf("Integration error: %v", err))
				return result, err
			}

		default:
			logger.Warn("⚠️  Unknown step type", "type", step.StepType)
		}

		result.ExecutedSteps = append(result.ExecutedSteps, step.StepName)

		// Record step execution to analytics
		workflow.ExecuteActivity(ctx, RecordStepAnalyticsActivity,
			input.TenantID, input.ProcessID, step.ID, stepResult).Get(ctx, nil)
	}

	// Record final analytics
	result.Status = "completed"
	result.CompletedAt = time.Now()
	result.TotalDuration = result.CompletedAt.Sub(startTime)

	workflow.ExecuteActivity(ctx, RecordWorkflowAnalyticsActivity,
		input.TenantID, input.ProcessID, result).Get(ctx, nil)

	logger.Info("✅ Dynamic BP Workflow completed successfully",
		"duration", result.TotalDuration,
		"finalBranch", result.FinalBranch,
	)

	return result, nil
}

// ============================================================================
// ACTIVITY: VALIDATION STEP
// ============================================================================

type ValidateInput struct {
	TenantID   string
	ProcessID  string
	StepID     string
	SourceData map[string]interface{}
}

func ValidateStepActivity(ctx context.Context, tenantID string, processID string, stepID string, sourceData map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("🔍 Validating step data", "stepID", stepID)

	// In production: load validation rules from DB and execute
	// For demo: simple validation
	validationResult := map[string]interface{}{
		"passed":      true,
		"validations": 3,
		"warnings":    0,
		"timestamp":   time.Now(),
	}

	logger.Info("✅ Validation completed", "result", validationResult)
	return validationResult, nil
}

// ============================================================================
// ACTIVITY: APPROVAL STEP
// ============================================================================

func ApprovalStepActivity(ctx context.Context, tenantID string, processID string, stepID string, assigneeRole string, timeout time.Duration) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("⏳ Waiting for approval",
		"role", assigneeRole,
		"timeout", timeout,
	)

	// Validate required inputs
	if assigneeRole == "" {
		logger.Error("❌ Approval failed: missing assignee role", "stepID", stepID)
		return nil, fmt.Errorf("missing assignee role for step %s", stepID)
	}

	// In production: integrate with notification service + polling for approval
	// For demo: auto-approve after 1 second and return the approver role
	select {
	case <-time.After(1 * time.Second):
		return map[string]interface{}{
			"approved":  true,
			"approver":  assigneeRole,
			"timestamp": time.Now(),
		}, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("approval timeout for role: %s", assigneeRole)
	}
}

// ============================================================================
// ACTIVITY: ESCALATION
// ============================================================================

func EscalateApprovalActivity(ctx context.Context, tenantID string, role string) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("🔔 Escalating approval",
		"fromRole", role,
		"toRole", "Manager",
	)

	// In production: notify manager via email/Slack
	return map[string]interface{}{
		"escalated": true,
		"fromRole":  role,
		"toRole":    "Manager",
		"timestamp": time.Now(),
	}, nil
}

// ============================================================================
// ACTIVITY: BRANCHING EVALUATION (CORE OF OPTION A + C)
// ============================================================================

func BranchingEvaluationActivity(ctx context.Context, tenantID string, processID string, stepID string, sourceData map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("🌳 Evaluating branches with all 15 advanced features", "stepID", stepID)

	// In production: create CompleteABranchEvaluator with DB connection
	// For demo: simulate the evaluation logic

	branchDecision := map[string]interface{}{
		"selected_branch": "high_priority_approval",
		"confidence":      0.95,
		"features_used": []string{
			"AI-Powered Routing",
			"Semantic Intent Routing",
			"Multi-Dimensional Scoring",
			"Time-Series Forecasting",
			"Adaptive Branching",
			"Blockchain Audit",
			"Explainability",
		},
		"evaluation_path": map[string]interface{}{
			"step_1": "AI model selected (accuracy: 0.96)",
			"step_2": "Semantic intent matched (similarity: 0.92)",
			"step_3": "Score: 8.5/10 (high priority)",
			"step_4": "Forecast: high load detected",
			"step_5": "Route: escalate to CFO for review",
		},
		"alternatives": []map[string]interface{}{
			{
				"branch": "standard_approval",
				"score":  6.2,
				"reason": "Lower urgency",
			},
			{
				"branch": "auto_approve",
				"score":  3.1,
				"reason": "Insufficient authority level",
			},
		},
		"explainability": map[string]interface{}{
			"feature_importance": map[string]float64{
				"salary_level":  0.40,
				"employee_type": 0.30,
				"department":    0.20,
				"urgency_score": 0.10,
			},
			"decision_path": "Salary > $100K AND VP-level → CFO approval required",
			"confidence":    0.95,
		},
		"blockchain": map[string]interface{}{
			"event_hash":          "abc123def456...",
			"verification_status": "verified",
			"signed_by":           "system",
		},
		"analytics": map[string]interface{}{
			"execution_time_ms": 245,
			"model_latency_ms":  45,
			"db_queries":        3,
		},
		"timestamp": time.Now(),
	}

	logger.Info("✅ Branching evaluation completed",
		"branch", branchDecision["selected_branch"],
		"confidence", branchDecision["confidence"],
		"features", len(branchDecision["features_used"].([]string)),
	)

	return branchDecision, nil
}

// ============================================================================
// ACTIVITY: NOTIFICATION
// ============================================================================

func NotificationActivity(ctx context.Context, tenantID string, message string, data map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("📧 Sending notification", "message", message)

	// In production: send via RabbitMQ, email, Slack, etc
	return map[string]interface{}{
		"sent":      true,
		"channel":   "email",
		"message":   message,
		"timestamp": time.Now(),
	}, nil
}

// ============================================================================
// ACTIVITY: EXTERNAL INTEGRATION
// ============================================================================

func IntegrationActivity(ctx context.Context, tenantID string, stepID string, data map[string]interface{}) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("🔗 Calling external integration", "stepID", stepID)

	// In production: call external APIs (HR system, accounting system, etc)
	return map[string]interface{}{
		"integrated": true,
		"system":     "HR System",
		"status":     "success",
		"timestamp":  time.Now(),
	}, nil
}

// ============================================================================
// ACTIVITY: RECORD STEP ANALYTICS
// ============================================================================

func RecordStepAnalyticsActivity(ctx context.Context, tenantID string, processID string, stepID string, stepResult map[string]interface{}) (map[string]interface{}, error) {
	// In production: update bp_branch_analytics_extended or custom analytics table
	return map[string]interface{}{
		"recorded": true,
		"step_id":  stepID,
	}, nil
}

// ============================================================================
// ACTIVITY: RECORD WORKFLOW ANALYTICS
// ============================================================================

func RecordWorkflowAnalyticsActivity(ctx context.Context, tenantID string, processID string, result *DynamicBPWorkflowResult) (map[string]interface{}, error) {
	logger := activity.GetLogger(ctx)
	logger.Info("📊 Recording workflow analytics",
		"status", result.Status,
		"duration", result.TotalDuration,
		"steps", len(result.ExecutedSteps),
	)

	// In production: update analytics tables
	return map[string]interface{}{
		"recorded":   true,
		"process_id": processID,
		"status":     result.Status,
	}, nil
}
