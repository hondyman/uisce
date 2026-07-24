package bp

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// ExecuteBlueprintWorkflow interprets and executes the designer blueprint
func ExecuteBlueprintWorkflow(ctx workflow.Context, bpDefID string, boCtx map[string]map[string]interface{}) error {
	logger := workflow.GetLogger(ctx)

	// Audit integration (using activity for database access)
	// In a real scenario, we'd inject this via context or activity
	// For now, we will perform audit logging via activities to be safe with Temporal determinism

	// Activity Options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumAttempts:    3,
		},
	}
	ctxA := workflow.WithActivityOptions(ctx, ao)

	// 1. Load Blueprint (via Activity to ensure determinism/isolation)
	var blueprint *WorkflowBlueprint
	// Note: You must register "LoadBlueprintActivity"
	err := workflow.ExecuteActivity(ctxA, "LoadBlueprintActivity", bpDefID).Get(ctx, &blueprint)
	if err != nil {
		logger.Error("Failed to load blueprint", "error", err)
		return err
	}

	ec := &BPExecutionContext{
		Blueprint:     blueprint,
		CurrentNodeID: blueprint.StartID,
		Values:        make(map[string]interface{}),
		BOCtx:         boCtx,
	}

	// Loop until end
	for ec.CurrentNodeID != "" {
		node, ok := blueprint.Nodes[ec.CurrentNodeID]
		if !ok {
			return fmt.Errorf("node %s not found", ec.CurrentNodeID)
		}

		logger.Info("Executing node", "StepKey", node.StepKey, "Type", node.Type)

		var nextNodeID string
		var err error

		switch node.Type {
		case "task":
			nextNodeID, err = executeTask(ctx, ec, node)
		case "approval":
			nextNodeID, err = executeApproval(ctx, ec, node)
		case "branch":
			nextNodeID, err = executeBranch(ctx, ec, node)
		case "delay":
			nextNodeID, err = executeDelay(ctx, ec, node)
		case "signal":
			nextNodeID, err = executeSignal(ctx, ec, node)
		default:
			// Fallback: treated as pass-through task or error
			// If it has NextNodes, just go there
			if len(node.NextNodes) > 0 {
				nextNodeID = node.NextNodes[0]
			} else {
				// If no next node and unknown type, maybe it's the end?
				if ec.CurrentNodeID == blueprint.EndID {
					nextNodeID = "" // Done
				} else {
					return fmt.Errorf("unknown node type: %s and no next node", node.Type)
				}
			}
		}

		if err != nil {
			logger.Error("Node execution failed", "StepKey", node.StepKey, "error", err)
			return err
		}

		// Check if we reached the explicit EndID
		if ec.CurrentNodeID == blueprint.EndID && nextNodeID == "" {
			break
		}

		ec.CurrentNodeID = nextNodeID
	}

	logger.Info("Workflow completed successfully")
	return nil
}

func executeTask(ctx workflow.Context, ec *BPExecutionContext, node *CompiledNode) (string, error) {
	// Execute the specific generic activity named in the node
	// We allow the user to specify activity name in designer.
	if node.ActivityName == "" {
		// Reference pass-through
		if len(node.NextNodes) > 0 {
			return node.NextNodes[0], nil
		}
		return "", nil
	}

	activityAO := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
	}
	ctxA := workflow.WithActivityOptions(ctx, activityAO)

	var result interface{}
	// Dynamic activity invocation
	err := workflow.ExecuteActivity(ctxA, node.ActivityName, ec.BOCtx).Get(ctx, &result)
	if err != nil {
		return "", err
	}
	ec.Values[node.StepKey] = result

	if len(node.NextNodes) > 0 {
		return node.NextNodes[0], nil
	}
	return "", nil
}

func executeApproval(ctx workflow.Context, ec *BPExecutionContext, node *CompiledNode) (string, error) {
	logger := workflow.GetLogger(ctx)

	activityAO := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	}
	ctxA := workflow.WithActivityOptions(ctx, activityAO)

	// 1. Resolve initial approver
	var initialApprover string
	err := workflow.ExecuteActivity(
		ctxA,
		"ResolveApproverRoleActivity",
		node.ApprovalChain, ec.BOCtx,
	).Get(ctx, &initialApprover)
	if err != nil {
		return "", err
	}
	logger.Info("Initial approver step", "role", initialApprover)

	// 2. Resolve overall SLA if present
	var overallSLA time.Duration
	if node.SLAExpr != "" {
		var d time.Duration
		err := workflow.ExecuteActivity(
			ctxA,
			"EvaluateDurationActivity",
			node.SLAExpr, ec.BOCtx,
		).Get(ctx, &d)
		if err == nil {
			overallSLA = d
		} else {
			logger.Warn("Failed to evaluate SLA expr", "error", err)
		}
	}

	// 3. Resolve escalation delays
	escalationDelays := make([]time.Duration, len(node.Escalations))
	for i, esc := range node.Escalations {
		var d time.Duration
		err := workflow.ExecuteActivity(
			ctxA,
			"EvaluateDurationActivity",
			esc.DelayAfterPreviousExpr,
			ec.BOCtx,
		).Get(ctx, &d)
		if err == nil {
			escalationDelays[i] = d
		} else {
			// Default to something safe or error?
			// For now log and default to 24h
			logger.Warn("Failed to evaluate escalation delay", "step", i, "error", err)
			escalationDelays[i] = 24 * time.Hour
		}
	}

	// 4. Run approval loop with escalations
	currentApprover := initialApprover
	currentEscalationIdx := 0
	approvalChan := workflow.GetSignalChannel(ctx, "approval_"+node.StepKey)

	// Track time elapsed for overall SLA
	startTime := workflow.Now(ctx)

	for {
		// Prepare context for activities
		ctxLoop := workflow.WithActivityOptions(ctx, activityAO)

		// Notify current approver
		// We use a specific activity for this now
		_ = workflow.ExecuteActivity(
			ctxLoop,
			"NotifyApproverActivity",
			currentApprover,
			node.StepKey,
			fmt.Sprintf("%d", currentEscalationIdx),
		).Get(ctx, nil)

		// Calculate time remaining for Overall SLA
		var slaTimerFuture workflow.Future
		var slaTimerCanceled workflow.CancelFunc
		if overallSLA > 0 {
			elapsed := workflow.Now(ctx).Sub(startTime)
			remaining := overallSLA - elapsed
			if remaining <= 0 {
				// SLA already exceeded?
				logger.Info("Overall SLA exceeded immediately")
				_ = workflow.ExecuteActivity(ctxLoop, "FinalEscalationActivity", node.StepKey, currentApprover).Get(ctx, nil)
				return "", fmt.Errorf("approval sla exceeded")
			}
			// Create timer (we might need to cancel it if we loop, or just let selector handle it)
			// Using NewTimer is fine, selector will just ignore if we break out of Select
			// But strictly speaking, if we loop, we create a new timer.
			// It's cleaner to have the timer fire relative to start or track remaining.
			// Simpler: Just set a timer for 'remaining'.
			ctxTimer, cancel := workflow.WithCancel(ctx)
			slaTimerCanceled = cancel
			slaTimerFuture = workflow.NewTimer(ctxTimer, remaining)
		}

		// Calculate time for Next Escalation
		var escalationTimerFuture workflow.Future
		var escalationTimerCanceled workflow.CancelFunc
		if currentEscalationIdx < len(node.Escalations) {
			delay := escalationDelays[currentEscalationIdx]
			ctxEsc, cancel := workflow.WithCancel(ctx)
			escalationTimerCanceled = cancel
			escalationTimerFuture = workflow.NewTimer(ctxEsc, delay)
			logger.Info("Waiting for approval", "role", currentApprover, "escalation_in", delay)
		} else {
			logger.Info("Waiting for approval (final level)", "role", currentApprover)
		}

		// Create selector
		selector := workflow.NewSelector(ctx)

		var signalReceived bool
		var approved bool
		selector.AddReceive(approvalChan, func(c workflow.ReceiveChannel, more bool) {
			c.Receive(ctx, &approved)
			signalReceived = true
		})

		var escalationFired bool
		if escalationTimerFuture != nil {
			selector.AddFuture(escalationTimerFuture, func(f workflow.Future) {
				escalationFired = true
			})
		}

		var slaExceeded bool
		if slaTimerFuture != nil {
			selector.AddFuture(slaTimerFuture, func(f workflow.Future) {
				slaExceeded = true
			})
		}

		selector.Select(ctx)

		// Clean up timers to avoid leaks or ghost firings?
		// Temporal Go SDK timers are persistent. Canceling context is good practice.
		if slaTimerCanceled != nil {
			slaTimerCanceled()
		}
		if escalationTimerCanceled != nil {
			escalationTimerCanceled()
		}

		if signalReceived {
			if approved {
				logger.Info("Approval granted")
				ec.Values[node.StepKey] = map[string]interface{}{
					"approved":        true,
					"approver":        currentApprover,
					"escalationLevel": currentEscalationIdx,
				}
				if len(node.NextNodes) > 0 {
					return node.NextNodes[0], nil
				}
				return "", nil
			} else {
				logger.Info("Approval rejected")
				return "", fmt.Errorf("approval rejected")
			}
		}

		if slaExceeded {
			logger.Info("Overall SLA exceeded, final escalation")
			_ = workflow.ExecuteActivity(
				ctxLoop,
				"FinalEscalationActivity",
				node.StepKey,
				currentApprover,
			).Get(ctx, nil)
			return "", fmt.Errorf("approval sla exceeded")
		}

		if escalationFired {
			// Move to next escalation step
			if currentEscalationIdx < len(node.Escalations) {
				nextEsc := node.Escalations[currentEscalationIdx]
				currentApprover = nextEsc.TargetActorRole
				// Optional: Check condition?
				logger.Info("Escalating approval", "new_role", currentApprover)
				currentEscalationIdx++
			}
			// Loop continues with new approver and next escalation delay
		}
	}
}

func executeBranch(ctx workflow.Context, ec *BPExecutionContext, node *CompiledNode) (string, error) {
	routingAO := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	}
	ctxA := workflow.WithActivityOptions(ctx, routingAO)

	var targetID string // ID or Key
	err := workflow.ExecuteActivity(
		ctxA,
		"ResolveBranchActivity",
		node.RoutingRules, ec.BOCtx,
	).Get(ctx, &targetID)
	if err != nil {
		return "", err
	}

	// If targetID is returned, verify it exists in blueprint
	if targetID != "" {
		// Check if it matches an ID directly
		if _, ok := ec.Blueprint.Nodes[targetID]; ok {
			return targetID, nil
		}
		// Check if it matches a StepKey (fallback lookup)
		for id, n := range ec.Blueprint.Nodes {
			if n.StepKey == targetID {
				return id, nil
			}
		}
	}

	// Default path if no rule matched
	if len(node.NextNodes) > 0 {
		return node.NextNodes[0], nil
	}

	return "", nil
}

func executeDelay(ctx workflow.Context, ec *BPExecutionContext, node *CompiledNode) (string, error) {
	delayAO := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	}
	ctxA := workflow.WithActivityOptions(ctx, delayAO)

	var d time.Duration
	err := workflow.ExecuteActivity(
		ctxA,
		"EvaluateDurationActivity",
		node.DelayExpr, ec.BOCtx,
	).Get(ctx, &d)
	if err != nil {
		return "", err
	}

	if d > 0 {
		if err := workflow.Sleep(ctx, d); err != nil {
			return "", err
		}
	}

	if len(node.NextNodes) > 0 {
		return node.NextNodes[0], nil
	}
	return "", nil
}

func executeSignal(ctx workflow.Context, ec *BPExecutionContext, node *CompiledNode) (string, error) {
	signalChan := workflow.GetSignalChannel(ctx, node.SignalName)
	var signalValue interface{}
	signalChan.Receive(ctx, &signalValue)

	ec.Values[node.StepKey] = signalValue

	if len(node.NextNodes) > 0 {
		return node.NextNodes[0], nil
	}
	return "", nil
}
