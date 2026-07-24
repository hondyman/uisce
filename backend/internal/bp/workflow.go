package bp

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/workflow"
)

// BPWorkflow is the main interpreter loop.
func BPWorkflow(ctx workflow.Context, wfCtx WorkflowContext) error {
	logger := workflow.GetLogger(ctx)
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute, // Basic timeout
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// 1. Load Definition
	var loadRes LoadDefinitionResult
	// We use string name for Activity to allow loose coupling or struct method if registered
	err := workflow.ExecuteActivity(ctx, "LoadDefinitionActivity", wfCtx.TenantID, wfCtx.BpKey, wfCtx.BpVersion).Get(ctx, &loadRes)
	if err != nil {
		logger.Error("Failed to load BP definition", "Error", err)
		return err
	}

	logger.Info("Starting BP Execution", "Name", loadRes.Def.Name, "Steps", len(loadRes.Steps))

	// 2. Initial Context (Simulated)
	boCtx := map[string]map[string]interface{}{
		"init": wfCtx.InputData,
	}

	// 3. Iterate Steps
	workflowRunID := workflow.GetInfo(ctx).WorkflowExecution.RunID
	// Log Process Start
	workflow.ExecuteActivity(ctx, "RecordProcessExecutionActivity", BPExecution{BPRunID: workflowRunID, Status: "running"})

	for _, step := range loadRes.Steps {
		logger.Info("Processing Step", "Seq", step.Seq, "Key", step.StepKey, "Type", step.Type)

		// 3a. Delay (Starlark)
		if step.DelayExpr != "" {
			var delay time.Duration
			delayKey := fmt.Sprintf("delay-%s", step.StepKey)
			err := workflow.ExecuteActivity(ctx, "EvaluateDurationActivity", DurationEvalInput{
				ExprType: step.DelayExprType,
				Expr:     step.DelayExpr,
				BOCtx:    boCtx,
			}).Get(ctx, &delay)
			if err != nil {
				return err
			}
			if delay > 0 {
				logger.Info("Step Delay Started", "Key", delayKey, "Duration", delay)
				workflow.Sleep(ctx, delay)
			}
		}

		// 3b. Execution Log Start
		stepExec := BPStepExecution{
			StepKey:  step.StepKey,
			Status:   "running",
			BPExecID: "mock-exec-id", // In real app, derived from Process Record
		}
		workflow.ExecuteActivity(ctx, "RecordStepExecutionActivity", stepExec)

		// 3c. Pre-Validations
		if len(step.PreValidationRuleIDs) > 0 {
			var valResult bool
			err := workflow.ExecuteActivity(ctx, "EvaluateRuleSetActivity", step.PreValidationRuleIDs, boCtx).Get(ctx, &valResult)
			if err != nil {
				return err
			}
			if !valResult {
				return fmt.Errorf("pre-validation failed for step %s", step.StepKey)
			}
		}

		// 3d. Condition (Starlark)
		if step.ConditionExpr != "" {
			var condResult bool
			err := workflow.ExecuteActivity(ctx, "EvaluateConditionActivity", ConditionEvalInput{
				ExprType: step.ConditionExprType,
				Expr:     step.ConditionExpr,
				BOCtx:    boCtx,
			}).Get(ctx, &condResult)
			if err != nil {
				return err
			}
			if !condResult {
				logger.Info("Skipping step due to condition", "Step", step.StepKey)
				// Log skip?
				continue
			}
		}

		// 3e. Routing (Advanced)
		var assignees []string
		if step.RoutingRules != nil {
			var role string
			err := workflow.ExecuteActivity(ctx, "EvaluateRoutingRulesActivity", step.RoutingRules, boCtx).Get(ctx, &role)
			if err != nil {
				return err
			}
			assignees = []string{role}
		}

		// 3f. Dynamic Participants (Fallback if no routing rules)
		if len(assignees) == 0 && (step.Type == "task" || step.Type == "approval") {
			err := workflow.ExecuteActivity(ctx, "ResolveParticipantsActivity", step, boCtx).Get(ctx, &assignees)
			if err != nil {
				return err
			}
		}

		// 3g. Step Execution Switch
		switch step.Type {
		case "task":
			// Create User Task & Wait
			var taskID string
			if err := workflow.ExecuteActivity(ctx, "CreateUserTaskActivity", step, workflowRunID, assignees).Get(ctx, &taskID); err != nil {
				return err
			}

			// SLA Logic (Parallel Monitor)
			taskCompleted := false

			if step.SLAExpr != "" {
				workflow.Go(ctx, func(ctx workflow.Context) {
					var slaDuration time.Duration
					err := workflow.ExecuteActivity(ctx, "EvaluateDurationActivity", DurationEvalInput{
						ExprType: step.SLAExprType,
						Expr:     step.SLAExpr,
						BOCtx:    boCtx,
					}).Get(ctx, &slaDuration)
					if err == nil && slaDuration > 0 {
						// Wait for SLA time
						_ = workflow.Sleep(ctx, slaDuration)

						if !taskCompleted {
							logger.Warn("SLA Breached - Escalating", "Step", step.StepKey)
							// Trigger Escalation (e.g. Notification)
							// Note: In real system, we might re-route or auto-complete here
							_ = workflow.ExecuteActivity(ctx, "NotificationActivity", step, boCtx).Get(ctx, nil)
						}
					}
				})
			}

			selector := workflow.NewSelector(ctx)
			signalName := "complete_task_" + step.StepKey
			var signalPayload interface{}

			// Task Completion Signal
			selector.AddReceive(workflow.GetSignalChannel(ctx, signalName), func(c workflow.ReceiveChannel, more bool) {
				c.Receive(ctx, &signalPayload)
				taskCompleted = true
			})

			selector.Select(ctx)

		case "approval":
			// Support Multi-Level Chain
			if step.ApprovalChain != nil {
				for i, level := range step.ApprovalChain.Levels {
					// Check Entry/Stop
					var res ApprovalLevelResult
					if err := workflow.ExecuteActivity(ctx, "EvaluateApprovalLevelActivity", level, boCtx).Get(ctx, &res); err != nil {
						return err
					}

					if res.ShouldStop {
						break
					}
					if !res.ShouldEnter {
						continue
					}

					// Execute Approval Level (User Task)
					subSignal := fmt.Sprintf("complete_task_%s_lvl%d", step.StepKey, i)

					logger.Info("Waiting for Approval Level", "Level", level.Name)
					workflow.GetSignalChannel(ctx, subSignal).Receive(ctx, nil)
				}
			} else {
				// Standard Single Approval
				workflow.GetSignalChannel(ctx, "complete_task_"+step.StepKey).Receive(ctx, nil)
			}

		case "integration":
			if step.IntegrationConfig != nil {
				if err := workflow.ExecuteActivity(ctx, "IntegrationActivity", step.IntegrationConfig, boCtx).Get(ctx, nil); err != nil {
					return err
				}
			}

		case "notification":
			if err := workflow.ExecuteActivity(ctx, "NotificationActivity", step, boCtx).Get(ctx, nil); err != nil {
				return err
			}
		}

		// 3h. Post-Validations
		if len(step.PostValidationRuleIDs) > 0 {
			var valResult bool
			err := workflow.ExecuteActivity(ctx, "EvaluateRuleSetActivity", step.PostValidationRuleIDs, boCtx).Get(ctx, &valResult)
			if err != nil {
				return err
			}
			if !valResult {
				return fmt.Errorf("post-validation failed for step %s", step.StepKey)
			}
		}

		// Log Complete
		stepExec.Status = "completed"
		workflow.ExecuteActivity(ctx, "RecordStepExecutionActivity", stepExec)
	}

	workflow.ExecuteActivity(ctx, "RecordProcessExecutionActivity", BPExecution{BPRunID: workflowRunID, Status: "completed"})
	return nil
}
