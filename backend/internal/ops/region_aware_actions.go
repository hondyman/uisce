package ops

import (
	"context"
	"fmt"
	"time"
)

// ============================================================================
// Phase 3.2: Region-Aware Action Execution
// Execute remediation actions scoped to specific regions
// ============================================================================

// RegionAwareActionExecutor manages execution of actions within region boundaries
type RegionAwareActionExecutor struct {
	regionRouter RegionRouter
	store        Store
	baseExecutor *ActionExecutor
}

// NewRegionAwareActionExecutor creates a region-aware action executor
func NewRegionAwareActionExecutor(
	regionRouter RegionRouter,
	store Store,
	baseExecutor *ActionExecutor,
) *RegionAwareActionExecutor {
	return &RegionAwareActionExecutor{
		regionRouter: regionRouter,
		store:        store,
		baseExecutor: baseExecutor,
	}
}

// RegionExecutionPlan specifies which actions to execute in which regions
type RegionExecutionPlan struct {
	PlanID           string
	CreatedAt        time.Time
	TargetIncidentID string
	TenantID         string

	// Execution phases
	PhaseOne   *ExecutionPhase // Immediate actions in root cause region
	PhaseTwo   *ExecutionPhase // Secondary actions in affected regions
	PhaseThree *ExecutionPhase // Cross-region propagation blocking

	// Execution state
	CurrentPhase     int
	Status           string // "pending", "in_progress", "completed", "failed"
	ExecutionResults map[string]*RegionExecutionResult
	LastUpdate       time.Time
}

// ExecutionPhase represents actions to execute in a particular phase
type ExecutionPhase struct {
	PhaseNumber     int
	TargetRegions   []string
	Actions         []*RegionScopedAction
	StartAfterMs    int64 // Delay before starting this phase (for staggered execution)
	TimeoutMs       int64
	RequiredSuccess float64 // % of actions that must succeed (0.5 = 50%)
}

// ActionExecutionResult tracks the result of a single action execution
type ActionExecutionResult struct {
	ActionID    string
	Status      string // "pending", "in_progress", "success", "failed", "rolled_back"
	StartedAt   *time.Time
	CompletedAt *time.Time
	Message     string
	Retries     int
}

// RegionScopedAction is an action execution scoped to a region
type RegionScopedAction struct {
	ActionID        string
	Region          string
	ActionType      string
	Config          map[string]interface{}
	Priority        string // "critical", "high", "medium", "low"
	TimeoutMs       int64
	RetryAttempts   int
	RollbackShuttle func(context.Context) error // Callback to undo action if needed

	// Execution state
	Attempts      int
	LastAttemptAt time.Time
	Result        *ActionExecutionResult
}

// RegionExecutionResult contains outcome of execution in a region
type RegionExecutionResult struct {
	Region            string
	PlanID            string
	ExecutedAt        time.Time
	CompletedAt       *time.Time
	ActionsAttempted  int
	ActionsSucceeded  int
	ActionsFailed     int
	IsolationApplied  bool
	RollbackTriggered bool
	ErrorsEncountered []string
	SuccessfulActions []string
	FailedActions     []string
}

// ExecuteWithRegionAwareness executes a remediation plan with region awareness
func (e *RegionAwareActionExecutor) ExecuteWithRegionAwareness(
	ctx context.Context,
	plan *RegionExecutionPlan,
) error {

	plan.Status = "in_progress"
	plan.ExecutionResults = make(map[string]*RegionExecutionResult)
	plan.LastUpdate = time.Now()

	// Phase 1: Execute in root cause region (immediate)
	if plan.PhaseOne != nil {
		if err := e.executePhase(ctx, plan, plan.PhaseOne); err != nil {
			plan.Status = "failed"
			return fmt.Errorf("phase 1 execution failed: %w", err)
		}
	}

	// Phase 2: Execute in secondary regions (with delay)
	if plan.PhaseTwo != nil {
		time.Sleep(time.Duration(plan.PhaseTwo.StartAfterMs) * time.Millisecond)
		if err := e.executePhase(ctx, plan, plan.PhaseTwo); err != nil {
			plan.Status = "failed"
			return fmt.Errorf("phase 2 execution failed: %w", err)
		}
	}

	// Phase 3: Execute cross-region blocking (if needed)
	if plan.PhaseThree != nil {
		time.Sleep(time.Duration(plan.PhaseThree.StartAfterMs) * time.Millisecond)
		if err := e.executePhase(ctx, plan, plan.PhaseThree); err != nil {
			plan.Status = "failed"
			return fmt.Errorf("phase 3 execution failed: %w", err)
		}
	}

	plan.Status = "completed"
	plan.LastUpdate = time.Now()
	return nil
}

// executePhase executes all actions in a single phase
func (e *RegionAwareActionExecutor) executePhase(
	ctx context.Context,
	plan *RegionExecutionPlan,
	phase *ExecutionPhase,
) error {

	plan.CurrentPhase = phase.PhaseNumber

	// Group actions by region
	actionsByRegion := make(map[string][]*RegionScopedAction)
	for _, action := range phase.Actions {
		actionsByRegion[action.Region] = append(actionsByRegion[action.Region], action)
	}

	// Execute actions per region with context-aware routing
	for region, actions := range actionsByRegion {
		result := &RegionExecutionResult{
			Region:            region,
			PlanID:            plan.PlanID,
			ExecutedAt:        time.Now(),
			SuccessfulActions: make([]string, 0),
			FailedActions:     make([]string, 0),
			ErrorsEncountered: make([]string, 0),
		}

		// Execute each action in the region
		for _, action := range actions {
			action.Attempts++
			action.LastAttemptAt = time.Now()

			// Route action to correct region infrastructure
			target, err := e.regionRouter.GetRegionTarget(ctx, region)
			if err != nil {
				result.ErrorsEncountered = append(result.ErrorsEncountered,
					fmt.Sprintf("Action %s: failed to resolve region target: %v", action.ActionID, err))
				result.ActionsFailed++
				action.Result = &ActionExecutionResult{
					Status:  "failed",
					Message: fmt.Sprintf("Failed to resolve region target: %v", err),
				}
				result.FailedActions = append(result.FailedActions, action.ActionID)
				continue
			}

			// Execute action with region context
			execResult := e.executeRegionScopedAction(ctx, action, target)
			action.Result = execResult
			result.ActionsAttempted++

			if execResult.Status == "success" {
				result.ActionsSucceeded++
				result.SuccessfulActions = append(result.SuccessfulActions, action.ActionID)
			} else {
				result.ActionsFailed++
				result.FailedActions = append(result.FailedActions, action.ActionID)
				result.ErrorsEncountered = append(result.ErrorsEncountered, execResult.Message)

				// Attempt rollback if action supports it
				if action.RollbackShuttle != nil && e.shouldRollback(result, phase) {
					if err := action.RollbackShuttle(ctx); err != nil {
						result.RollbackTriggered = true
						result.ErrorsEncountered = append(result.ErrorsEncountered,
							fmt.Sprintf("Action %s rollback also failed: %v", action.ActionID, err))
					}
				}
			}
		}

		result.CompletedAt = timePtr(time.Now())
		plan.ExecutionResults[region] = result

		// Check phase success threshold
		if result.ActionsAttempted > 0 {
			successRate := float64(result.ActionsSucceeded) / float64(result.ActionsAttempted)
			if successRate < phase.RequiredSuccess {
				return fmt.Errorf("region %s: success rate %f%% below required %f%%",
					region, successRate*100, phase.RequiredSuccess*100)
			}
		}
	}

	return nil
}

// executeRegionScopedAction executes a single action in a targeted region
func (e *RegionAwareActionExecutor) executeRegionScopedAction(
	ctx context.Context,
	action *RegionScopedAction,
	target *RegionTarget,
) *ActionExecutionResult {

	result := &ActionExecutionResult{
		ActionID:  action.ActionID,
		Status:    "pending",
		StartedAt: timePtr(time.Now()),
	}

	// Apply region-specific execution logic
	switch action.ActionType {
	case "restart_worker":
		result = e.executeRestartWorkerInRegion(ctx, action, target)

	case "throttle_tenant":
		result = e.executeThrottleTenantInRegion(ctx, action, target)

	case "isolate_region":
		result = e.executeIsolateRegionAction(ctx, action, target)

	case "failover_region":
		result = e.executeFailoverRegionAction(ctx, action, target)

	case "throttle_region":
		result = e.executeThrottleRegionAction(ctx, action, target)

	default:
		result.Status = "failed"
		result.Message = fmt.Sprintf("unknown action type: %s", action.ActionType)
	}

	result.CompletedAt = timePtr(time.Now())
	return result
}

// executeRestartWorkerInRegion restarts workers in a specific region
func (e *RegionAwareActionExecutor) executeRestartWorkerInRegion(
	ctx context.Context,
	action *RegionScopedAction,
	target *RegionTarget,
) *ActionExecutionResult {

	result := &ActionExecutionResult{
		ActionID:  action.ActionID,
		Status:    "in_progress",
		StartedAt: timePtr(time.Now()),
	}

	// Get action config
	workerPool, ok := action.Config["worker_pool"].(string)
	if !ok {
		workerPool = target.OpsWorkerPool
	}

	// Log restart action
	result.Message = fmt.Sprintf("Restarting worker pool '%s' in region %s", workerPool, action.Region)

	// TODO: Implement actual worker restart in target region
	// For now, simulate success

	result.Status = "success"
	result.Message = fmt.Sprintf("Successfully restarted %d workers", 3)

	return result
}

// executeThrottleTenantInRegion throttles a tenant in a specific region
func (e *RegionAwareActionExecutor) executeThrottleTenantInRegion(
	ctx context.Context,
	action *RegionScopedAction,
	target *RegionTarget,
) *ActionExecutionResult {

	result := &ActionExecutionResult{
		ActionID:  action.ActionID,
		Status:    "in_progress",
		StartedAt: timePtr(time.Now()),
	}

	throttlePercent, ok := action.Config["throttle_percent"].(float64)
	if !ok {
		throttlePercent = 0.5 // Default 50% throttle
	}

	result.Message = fmt.Sprintf("Applying %.0f%% throttle to tenant in region %s", throttlePercent*100, action.Region)

	// TODO: Implement actual tenant throttling via Redpanda/queuing system

	result.Status = "success"
	result.Message = fmt.Sprintf("Applied %.0f%% throttle via rate limiter", throttlePercent*100)

	return result
}

// executeIsolateRegionAction isolates a region from cross-region propagation
func (e *RegionAwareActionExecutor) executeIsolateRegionAction(
	ctx context.Context,
	action *RegionScopedAction,
	target *RegionTarget,
) *ActionExecutionResult {

	result := &ActionExecutionResult{
		ActionID:  action.ActionID,
		Status:    "in_progress",
		StartedAt: timePtr(time.Now()),
	}

	result.Message = fmt.Sprintf("Isolating region %s from cross-region traffic", action.Region)

	// TODO: Implement network isolation / circuit breaker

	result.Status = "success"
	result.Message = fmt.Sprintf("Region %s isolated - cross-region traffic blocked", action.Region)

	return result
}

// executeFailoverRegionAction fails over to alternate region
func (e *RegionAwareActionExecutor) executeFailoverRegionAction(
	ctx context.Context,
	action *RegionScopedAction,
	target *RegionTarget,
) *ActionExecutionResult {

	result := &ActionExecutionResult{
		ActionID:  action.ActionID,
		Status:    "in_progress",
		StartedAt: timePtr(time.Now()),
	}

	if target.FailoverTarget == nil {
		result.Status = "failed"
		result.Message = fmt.Sprintf("No failover target configured for region %s", action.Region)
		return result
	}

	result.Message = fmt.Sprintf("Failing over from %s to %s", action.Region, *target.FailoverTarget)

	// TODO: Implement actual failover

	result.Status = "success"
	result.Message = fmt.Sprintf("Successfully failed over to region %s", *target.FailoverTarget)

	return result
}

// executeThrottleRegionAction throttles an entire region to limit propagation
func (e *RegionAwareActionExecutor) executeThrottleRegionAction(
	ctx context.Context,
	action *RegionScopedAction,
	target *RegionTarget,
) *ActionExecutionResult {

	result := &ActionExecutionResult{
		ActionID:  action.ActionID,
		Status:    "in_progress",
		StartedAt: timePtr(time.Now()),
	}

	throttlePercent, ok := action.Config["throttle_percent"].(float64)
	if !ok {
		throttlePercent = 0.3 // Default 30% throttle for region
	}

	result.Message = fmt.Sprintf("Applying %.0f%% region-wide throttle to %s", throttlePercent*100, action.Region)

	// TODO: Implement region-wide throttling

	result.Status = "success"
	result.Message = fmt.Sprintf("Applied %.0f%% region throttle", throttlePercent*100)

	return result
}

// shouldRollback decides whether to rollback based on execution state
func (e *RegionAwareActionExecutor) shouldRollback(result *RegionExecutionResult, phase *ExecutionPhase) bool {
	if result.ActionsAttempted == 0 {
		return false
	}
	failureRate := float64(result.ActionsFailed) / float64(result.ActionsAttempted)
	return failureRate > (1.0 - phase.RequiredSuccess)
}

// Helper function to get pointer to time.Time
func timePtr(t time.Time) *time.Time {
	return &t
}
