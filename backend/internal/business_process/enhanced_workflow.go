package business_process

import (
	"context"
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// MetricsCollectorKey is the context key for the metrics collector
type contextKey string

const MetricsCollectorKey contextKey = "metrics_collector"

// ParallelStepGroup groups steps that should execute in parallel
type ParallelStepGroup struct {
	GroupID    string
	Steps      []*Step
	WaitForAll bool // true = all must complete, false = any can complete
}

// ExecuteParallelSteps executes a group of steps in parallel
func ExecuteParallelSteps(ctx workflow.Context, group ParallelStepGroup, obj *GenericBusinessObject) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Executing Parallel Step Group", "GroupID", group.GroupID, "StepCount", len(group.Steps))

	// Create a selector for parallel execution
	selector := workflow.NewSelector(ctx)
	futures := make([]workflow.Future, len(group.Steps))

	// Start all steps in parallel
	for i, step := range group.Steps {
		idx := i
		currentStep := step

		// Execute each step as a child workflow or activity
		futures[idx] = workflow.ExecuteActivity(
			ctx,
			ExecuteStepActivity,
			currentStep,
			obj,
		)

		// Add to selector
		selector.AddFuture(futures[idx], func(f workflow.Future) {
			var result interface{}
			if err := f.Get(ctx, &result); err != nil {
				logger.Error("Step execution failed", "StepID", currentStep.ID, "Error", err)
			} else {
				logger.Info("Step completed", "StepID", currentStep.ID)
			}
		})
	}

	if group.WaitForAll {
		// Wait for all steps to complete
		for i := range futures {
			var result interface{}
			if err := futures[i].Get(ctx, &result); err != nil {
				return fmt.Errorf("parallel step failed: %v", err)
			}
		}
	} else {
		// Wait for any one to complete
		selector.Select(ctx)
	}

	return nil
}

// EvaluateAdvancedCondition evaluates complex boolean conditions
func EvaluateAdvancedCondition(condition *AdvancedCondition, obj *GenericBusinessObject) (bool, error) {
	if condition == nil || len(condition.Conditions) == 0 {
		return true, nil
	}

	results := make([]bool, len(condition.Conditions))

	// Evaluate each individual condition
	for i, cond := range condition.Conditions {
		result, err := evaluateSingleCondition(&cond, obj)
		if err != nil {
			return false, err
		}
		results[i] = result
	}

	// Apply boolean operator
	switch condition.Operator {
	case "AND":
		for _, r := range results {
			if !r {
				return false, nil
			}
		}
		return true, nil

	case "OR":
		for _, r := range results {
			if r {
				return true, nil
			}
		}
		return false, nil

	case "NOT":
		// NOT negates the first condition result
		if len(results) > 0 {
			return !results[0], nil
		}
		return true, nil

	default:
		return false, fmt.Errorf("unknown operator: %s", condition.Operator)
	}
}

// evaluateSingleCondition evaluates a single condition
func evaluateSingleCondition(cond *Condition, obj *GenericBusinessObject) (bool, error) {
	// Get the field value from the business object
	fieldValue, exists := obj.Data[cond.Field]
	if !exists {
		return false, nil
	}

	// Evaluate based on operator
	switch cond.Operator {
	case "==":
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", cond.Value), nil
	case "!=":
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", cond.Value), nil
	case ">":
		fv, fvOk := fieldValue.(float64)
		cv, cvOk := cond.Value.(float64)
		if fvOk && cvOk {
			return fv > cv, nil
		}
		return false, fmt.Errorf("comparison requires numeric values")
	case "<":
		fv, fvOk := fieldValue.(float64)
		cv, cvOk := cond.Value.(float64)
		if fvOk && cvOk {
			return fv < cv, nil
		}
		return false, fmt.Errorf("comparison requires numeric values")
	case ">=":
		fv, fvOk := fieldValue.(float64)
		cv, cvOk := cond.Value.(float64)
		if fvOk && cvOk {
			return fv >= cv, nil
		}
		return false, fmt.Errorf("comparison requires numeric values")
	case "<=":
		fv, fvOk := fieldValue.(float64)
		cv, cvOk := cond.Value.(float64)
		if fvOk && cvOk {
			return fv <= cv, nil
		}
		return false, fmt.Errorf("comparison requires numeric values")
	case "in":
		// Check if value is in array
		if arr, ok := cond.Value.([]interface{}); ok {
			for _, item := range arr {
				if fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", item) {
					return true, nil
				}
			}
		}
		return false, nil
	case "contains":
		fvStr := fmt.Sprintf("%v", fieldValue)
		cvStr := fmt.Sprintf("%v", cond.Value)
		return contains(fvStr, cvStr), nil
	case "startsWith":
		fvStr := fmt.Sprintf("%v", fieldValue)
		cvStr := fmt.Sprintf("%v", cond.Value)
		return len(fvStr) >= len(cvStr) && fvStr[:len(cvStr)] == cvStr, nil
	case "endsWith":
		fvStr := fmt.Sprintf("%v", fieldValue)
		cvStr := fmt.Sprintf("%v", cond.Value)
		return len(fvStr) >= len(cvStr) && fvStr[len(fvStr)-len(cvStr):] == cvStr, nil
	default:
		return false, fmt.Errorf("unknown operator: %s", cond.Operator)
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// ResolveApprovalChain resolves the approval chain based on type
func ResolveApprovalChain(ctx workflow.Context, chain *ApprovalChain, obj *GenericBusinessObject) ([]string, error) {
	if chain == nil {
		return []string{}, nil
	}

	switch chain.Type {
	case "role":
		// Return single role
		if len(chain.Roles) > 0 {
			return []string{chain.Roles[0]}, nil
		}
		return []string{}, nil

	case "multi_role":
		// Return all roles
		return chain.Roles, nil

	case "org_hierarchy":
		// Resolve organization hierarchy (would query org chart)
		// For now, return a simulated chain
		levels := 1
		if chain.Levels != nil {
			levels = *chain.Levels
		}

		approvers := []string{}
		currentManager := getManagerID(obj)
		for i := 0; i < levels && currentManager != ""; i++ {
			approvers = append(approvers, currentManager)
			currentManager = getManagerID(&GenericBusinessObject{
				Data: map[string]interface{}{"user_id": currentManager},
			})
		}
		return approvers, nil

	case "custom":
		// Execute custom logic (could call activity)
		return []string{}, nil

	default:
		return []string{}, fmt.Errorf("unknown approval chain type: %s", chain.Type)
	}
}

// getManagerID retrieves the manager ID for a given user (simplified)
func getManagerID(obj *GenericBusinessObject) string {
	// In a real implementation, this would query the org chart
	// For now, return a placeholder
	if userID, ok := obj.Data["user_id"].(string); ok {
		return "manager_of_" + userID
	}
	return ""
}

// CheckStepDependencies verifies all dependency steps have completed
func CheckStepDependencies(ctx workflow.Context, dependsOn []string, completedSteps map[string]bool) (bool, error) {
	if len(dependsOn) == 0 {
		return true, nil
	}

	for _, depStepID := range dependsOn {
		if !completedSteps[depStepID] {
			return false, nil
		}
	}

	return true, nil
}

// ShouldSkipStep evaluates skip condition to determine if step should be skipped
func ShouldSkipStep(ctx workflow.Context, skipCondition *AdvancedCondition, obj *GenericBusinessObject) (bool, error) {
	if skipCondition == nil {
		return false, nil
	}

	return EvaluateAdvancedCondition(skipCondition, obj)
}

// Enhanced DynamicProcessWorkflow with parallel execution and advanced features
func EnhancedDynamicProcessWorkflow(ctx workflow.Context, params DynamicWorkflowParams) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting EnhancedDynamicProcessWorkflow", "ObjectID", params.BusinessObject.ID)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	var template EnhancedProcessTemplate

	// Fetch workflow definition
	if params.WorkflowDefinitionID != "" {
		err := workflow.ExecuteActivity(ctx, FetchEnhancedWorkflowDefinitionActivity, params.WorkflowDefinitionID).Get(ctx, &template)
		if err != nil {
			return fmt.Errorf("failed to fetch workflow definition: %v", err)
		}
	} else {
		return fmt.Errorf("WorkflowDefinitionID is required")
	}

	obj := params.BusinessObject
	completedSteps := make(map[string]bool)

	// Group steps by parallel group
	stepGroups := make(map[string]*ParallelStepGroup)
	sequentialSteps := []*EnhancedStep{}

	for i := range template.Steps {
		step := &template.Steps[i]
		if step.ExecutionMode == "parallel" && step.ParallelGroup != "" {
			if stepGroups[step.ParallelGroup] == nil {
				stepGroups[step.ParallelGroup] = &ParallelStepGroup{
					GroupID:    step.ParallelGroup,
					Steps:      []*Step{},
					WaitForAll: step.WaitForAll,
				}
			}
			// Convert to base Step type for execution
			baseStep := &Step{
				ID:          step.ID,
				Type:        step.Type,
				ActivityRef: step.ActivityRef,
				SLA:         step.SLA,
			}
			stepGroups[step.ParallelGroup].Steps = append(stepGroups[step.ParallelGroup].Steps, baseStep)
		} else {
			sequentialSteps = append(sequentialSteps, step)
		}
	}

	// Execute sequential steps
	for _, step := range sequentialSteps {
		// Check dependencies
		canExecute, err := CheckStepDependencies(ctx, step.DependsOn, completedSteps)
		if err != nil {
			return err
		}
		if !canExecute {
			logger.Info("Step dependencies not met, skipping", "StepID", step.ID)
			continue
		}

		// Check skip condition
		shouldSkip, err := ShouldSkipStep(ctx, step.SkipCondition, &obj)
		if err != nil {
			return err
		}
		if shouldSkip {
			logger.Info("Step skip condition met, skipping", "StepID", step.ID)
			completedSteps[step.ID] = true
			continue
		}

		logger.Info("Executing Step", "StepID", step.ID, "Type", step.Type)

		// Record step start time for metrics
		stepStartTime := workflow.Now(ctx)

		// Execute step based on type
		var stepErr error
		if step.Type == "approval" && step.ApprovalChain != nil {
			// Resolve approval chain
			approvers, err := ResolveApprovalChain(ctx, step.ApprovalChain, &obj)
			if err != nil {
				return err
			}
			logger.Info("Approval chain resolved", "Approvers", approvers, "Mode", step.ApprovalChain.ApprovalMode)

			// Execute approval workflow with resolved chain
			// Implementation would depend on your approval system
		}

		// Check for advanced condition branching
		if step.ConditionLogic != nil {
			result, err := EvaluateAdvancedCondition(step.ConditionLogic, &obj)
			if err != nil {
				return err
			}

			// Execute true or false branch based on result
			if result && len(step.ConditionLogic.TrueBranch) > 0 {
				logger.Info("Condition true, executing true branch", "StepID", step.ID)
				// Would recursively execute true branch steps
			} else if !result && len(step.ConditionLogic.FalseBranch) > 0 {
				logger.Info("Condition false, executing false branch", "StepID", step.ID)
				// Would recursively execute false branch steps
			}
		}

		// Record step completion metrics
		stepEndTime := workflow.Now(ctx)
		duration := stepEndTime.Sub(stepStartTime)
		status := "completed"
		var errorMsg *string
		if stepErr != nil {
			status = "failed"
			errStr := stepErr.Error()
			errorMsg = &errStr
		}

		// Execute activity to record metrics (non-blocking, fire-and-forget)
		workflow.ExecuteActivity(
			workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
				StartToCloseTimeout: 10 * time.Second,
			}),
			"RecordStepMetrics",
			map[string]interface{}{
				"workflow_id":   workflow.GetInfo(ctx).WorkflowExecution.ID,
				"workflow_type": params.ProcessName,
				"tenant_id":     params.TenantID,
				"step_name":     step.Name,
				"step_type":     step.Type,
				"start_time":    stepStartTime,
				"end_time":      stepEndTime,
				"duration":      duration.String(),
				"status":        status,
				"error_message": errorMsg,
				"metadata": map[string]interface{}{
					"execution_mode": step.ExecutionMode,
					"parallel_group": step.ParallelGroup,
				},
			},
		)

		completedSteps[step.ID] = true
	}

	// Execute parallel groups
	for groupID, group := range stepGroups {
		logger.Info("Executing parallel group", "GroupID", groupID)
		if err := ExecuteParallelSteps(ctx, *group, &obj); err != nil {
			return err
		}

		// Mark all steps in group as completed
		for _, step := range group.Steps {
			completedSteps[step.ID] = true
		}
	}

	logger.Info("Enhanced workflow completed", "CompletedSteps", len(completedSteps))
	return nil
}

// --- Enhanced Activities ---

func FetchEnhancedWorkflowDefinitionActivity(ctx context.Context, definitionID string) (*EnhancedProcessTemplate, error) {
	fmt.Printf("Fetching Enhanced Workflow Definition: %s\n", definitionID)
	// Return mock for now
	return &EnhancedProcessTemplate{
		ID:    definitionID,
		Name:  "Enhanced Process",
		Steps: []EnhancedStep{},
	}, nil
}

func ExecuteStepActivity(ctx context.Context, step *Step, obj *GenericBusinessObject) (*GenericBusinessObject, error) {
	// Execute based on type (simplified wrapper)
	// In real implementation, this would delegate to specific activities
	fmt.Printf("Executing Step Activity: %s\n", step.ID)
	return obj, nil
}
