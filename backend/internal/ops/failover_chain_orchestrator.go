package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// FailoverChainOrchestrator manages cascading failover chains
type FailoverChainOrchestrator struct {
	store              Store
	actionExecutor     *ActionExecutor
	regionRouter       *InMemoryRegionRegistry
	simpleOrchestrator *FailoverOrchestrator
}

// NewFailoverChainOrchestrator creates a new chain orchestration engine
func NewFailoverChainOrchestrator(
	store Store,
	actionExecutor *ActionExecutor,
	regionRouter *InMemoryRegionRegistry,
	simpleOrchestrator *FailoverOrchestrator,
) *FailoverChainOrchestrator {
	return &FailoverChainOrchestrator{
		store:              store,
		actionExecutor:     actionExecutor,
		regionRouter:       regionRouter,
		simpleOrchestrator: simpleOrchestrator,
	}
}

// ChainStepResult represents the result of a single chain step
type ChainStepResult struct {
	Step       int
	Target     string
	Success    bool
	Error      string
	Duration   time.Duration
	ResolvedAt time.Time
}

// CheckChainConditions evaluates if a failover chain should trigger
func (f *FailoverChainOrchestrator) CheckChainConditions(ctx context.Context, chain *FailoverChain) (*FailoverConditionMet, error) {
	// Get current metrics for source region
	metrics, err := f.store.GetRegionalMetrics(ctx, chain.SourceRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to get regional metrics: %w", err)
	}

	if metrics == nil {
		// No metrics data yet, don't trigger chain
		return &FailoverConditionMet{Met: false}, nil
	}

	// Parse chain targets
	var chainTargets []string
	if err := json.Unmarshal([]byte(chain.ChainTargets), &chainTargets); err != nil {
		return &FailoverConditionMet{Met: false}, nil
	}

	if len(chainTargets) == 0 {
		return &FailoverConditionMet{Met: false}, nil
	}

	// Check error rate threshold
	if chain.TriggerErrorRate != nil && metrics.ErrorRate > *chain.TriggerErrorRate {
		return &FailoverConditionMet{
			Met:          true,
			TriggerType:  "error_rate",
			TriggerValue: metrics.ErrorRate,
			TargetRegion: chainTargets[0], // Start with first target
		}, nil
	}

	// Check latency threshold (p99 latency in ms)
	if chain.TriggerLatency != nil && metrics.P99Latency > *chain.TriggerLatency {
		return &FailoverConditionMet{
			Met:          true,
			TriggerType:  "latency",
			TriggerValue: float64(metrics.P99Latency),
			TargetRegion: chainTargets[0],
		}, nil
	}

	// Check health score threshold
	health, err := f.store.GetRegionalHealth(ctx, chain.SourceRegion)
	if err == nil && health != nil && chain.TriggerHealthScore != nil && health.Score < *chain.TriggerHealthScore {
		return &FailoverConditionMet{
			Met:          true,
			TriggerType:  "health_score",
			TriggerValue: float64(health.Score),
			TargetRegion: chainTargets[0],
		}, nil
	}

	return &FailoverConditionMet{Met: false}, nil
}

// ExecuteChain initiates a failover chain from source through multiple targets
func (f *FailoverChainOrchestrator) ExecuteChain(ctx context.Context, chain *FailoverChain, condition *FailoverConditionMet, incidentID *uuid.UUID) (*FailoverChainExecution, error) {
	// Parse chain targets
	var chainTargets []string
	if err := json.Unmarshal([]byte(chain.ChainTargets), &chainTargets); err != nil {
		return nil, fmt.Errorf("invalid chain targets JSON: %w", err)
	}

	if len(chainTargets) == 0 {
		return nil, fmt.Errorf("no targets in chain")
	}

	// Create chain execution record
	var incident uuid.UUID
	if incidentID != nil {
		incident = *incidentID
	}

	execution := &FailoverChainExecution{
		ChainID:        chain.ID,
		TenantID:       chain.TenantID,
		SourceRegion:   chain.SourceRegion,
		CurrentStep:    0,
		CurrentTarget:  chainTargets[0],
		Status:         "in_progress",
		TriggeredAt:    time.Now().UTC(),
		IncidentID:     incident,
		StepsExecuted:  []string{},
		FailureReasons: []string{},
	}

	if err := f.store.InsertFailoverChainExecution(ctx, execution); err != nil {
		return nil, fmt.Errorf("failed to insert chain execution: %w", err)
	}

	startTime := time.Now()
	stepResults := []ChainStepResult{}

	// Execute chain: try each target in sequence
	for step := 0; step < len(chainTargets) && step < chain.MaxChainDepth; step++ {
		target := chainTargets[step]
		stepStartTime := time.Now()

		// Try failover to this target
		policy := &FailoverPolicy{
			ID:            uuid.New(),
			SourceRegion:  chain.SourceRegion,
			TargetRegions: fmt.Sprintf(`["%s"]`, target),
			IsAutomatic:   true,
		}

		stepCondition := &FailoverConditionMet{
			Met:          true,
			TriggerType:  condition.TriggerType,
			TriggerValue: condition.TriggerValue,
			TargetRegion: target,
		}

		_, err := f.simpleOrchestrator.ExecuteFailover(ctx, policy, stepCondition, incidentID)
		stepDuration := time.Since(stepStartTime)

		execution.StepsExecuted = append(execution.StepsExecuted, target)
		execution.CurrentStep = step
		execution.CurrentTarget = target

		if err == nil {
			// Success! Chain resolved at this step
			execution.Status = "success"
			if step == 0 {
				// First step succeeded
				execution.Status = "success"
			} else {
				// Fallback target succeeded
				execution.Status = "success"
			}

			stepResults = append(stepResults, ChainStepResult{
				Step:       step,
				Target:     target,
				Success:    true,
				Duration:   stepDuration,
				ResolvedAt: time.Now().UTC(),
			})

			// Update execution
			completedAt := time.Now().UTC()
			f.store.UpdateFailoverChainExecution(
				ctx, execution.ID, execution.Status,
				execution.StepsExecuted, execution.FailureReasons, &completedAt,
			)

			// Update metrics
			f.updateChainMetrics(ctx, chain, step+1, true, time.Since(startTime))

			return execution, nil
		}

		// This step failed, record and continue
		execution.FailureReasons = append(execution.FailureReasons, fmt.Sprintf("Step %d (%s): %v", step, target, err))
		execution.PreviousTarget = &target

		stepResults = append(stepResults, ChainStepResult{
			Step:     step,
			Target:   target,
			Success:  false,
			Error:    err.Error(),
			Duration: stepDuration,
		})

		// Check cooldown before next step
		if step < len(chainTargets)-1 {
			cooldownDuration := time.Duration(chain.CooldownMinutes) * time.Minute
			time.Sleep(cooldownDuration)
		}
	}

	// All steps exhausted - chain failed
	execution.Status = "exhausted"
	completedAt := time.Now().UTC()
	f.store.UpdateFailoverChainExecution(
		ctx, execution.ID, execution.Status,
		execution.StepsExecuted, execution.FailureReasons, &completedAt,
	)

	// Update metrics
	f.updateChainMetrics(ctx, chain, len(chainTargets), false, time.Since(startTime))

	return execution, fmt.Errorf("all chain targets exhausted, chain failed")
}

// updateChainMetrics tracks chain execution statistics
func (f *FailoverChainOrchestrator) updateChainMetrics(ctx context.Context, chain *FailoverChain, stepsUsed int, success bool, duration time.Duration) error {
	metrics, err := f.store.GetFailoverChainMetrics(ctx, chain.ID)
	if err != nil {
		return err
	}

	if metrics == nil {
		metrics = &FailoverChainMetrics{
			ChainID:             chain.ID,
			TotalExecutions:     1,
			SuccessfulCount:     0,
			PartialSuccessCount: 0,
			FailedCount:         0,
			AvgStepsNeeded:      float64(stepsUsed),
			AvgDurationMs:       int64(duration.Milliseconds()),
		}

		if success {
			if stepsUsed == 1 {
				metrics.SuccessfulCount = 1
			} else {
				metrics.PartialSuccessCount = 1
			}
		} else {
			metrics.FailedCount = 1
		}
	} else {
		metrics.TotalExecutions++

		// Update average steps
		totalSteps := metrics.AvgStepsNeeded * float64(metrics.TotalExecutions-1)
		totalSteps += float64(stepsUsed)
		metrics.AvgStepsNeeded = totalSteps / float64(metrics.TotalExecutions)

		// Update average duration
		totalDurationMs := int64(metrics.AvgDurationMs) * int64(metrics.TotalExecutions-1)
		totalDurationMs += int64(duration.Milliseconds())
		metrics.AvgDurationMs = totalDurationMs / int64(metrics.TotalExecutions)

		if success {
			if stepsUsed == 1 {
				metrics.SuccessfulCount++
			} else {
				metrics.PartialSuccessCount++
			}
		} else {
			metrics.FailedCount++
		}
	}

	// Calculate success rate
	if metrics.TotalExecutions > 0 {
		totalResolved := metrics.SuccessfulCount + metrics.PartialSuccessCount
		metrics.SuccessRatePct = (float64(totalResolved) / float64(metrics.TotalExecutions)) * 100
	}

	now := time.Now().UTC()
	metrics.LastExecutionAt = &now

	return f.store.UpsertFailoverChainMetrics(ctx, metrics)
}

// EvaluateAllChains runs periodic evaluation of all active failover chains
// Returns a map of chain IDs to failover conditions that should trigger
func (f *FailoverChainOrchestrator) EvaluateAllChains(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]*FailoverConditionMet, error) {
	// Get all active chains for tenant
	chains, err := f.store.ListFailoverChains(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list failover chains: %w", err)
	}

	result := make(map[uuid.UUID]*FailoverConditionMet)

	for _, chain := range chains {
		// Check cooldown: don't evaluate chains that recently executed
		recentExecutions, err := f.store.ListFailoverChainExecutions(ctx, chain.ID, 1)
		if err == nil && len(recentExecutions) > 0 {
			lastExecution := recentExecutions[0]
			cooldownDuration := time.Duration(chain.CooldownMinutes) * time.Minute
			if time.Since(lastExecution.TriggeredAt) < cooldownDuration {
				continue // Skip this chain due to cooldown
			}
		}

		// Evaluate conditions
		condition, err := f.CheckChainConditions(ctx, &chain)
		if err != nil {
			// Log but continue with other chains
			continue
		}

		if condition.Met {
			result[chain.ID] = condition
		}
	}

	return result, nil
}
