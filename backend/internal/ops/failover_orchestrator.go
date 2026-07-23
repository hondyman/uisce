package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// FailoverOrchestrator manages automated regional failover based on policies
type FailoverOrchestrator struct {
	store          Store
	actionExecutor *ActionExecutor
	regionRouter   *InMemoryRegionRegistry
}

// NewFailoverOrchestrator creates a new failover orchestration engine
func NewFailoverOrchestrator(store Store, actionExecutor *ActionExecutor, regionRouter *InMemoryRegionRegistry) *FailoverOrchestrator {
	return &FailoverOrchestrator{
		store:          store,
		actionExecutor: actionExecutor,
		regionRouter:   regionRouter,
	}
}

// FailoverConditionMet represents the result of condition evaluation
type FailoverConditionMet struct {
	Met          bool
	TriggerType  string // "health_score", "error_rate", "latency"
	TriggerValue float64
	TargetRegion string
}

// CheckFailoverConditions evaluates if a failover policy should trigger
func (f *FailoverOrchestrator) CheckFailoverConditions(ctx context.Context, policy *FailoverPolicy) (*FailoverConditionMet, error) {
	// Get current metrics for source region
	metrics, err := f.store.GetRegionalMetrics(ctx, policy.SourceRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to get regional metrics: %w", err)
	}

	if metrics == nil {
		// No metrics data yet, don't trigger failover
		return &FailoverConditionMet{Met: false}, nil
	}

	// Get health status to make decision
	health, err := f.store.GetRegionalHealth(ctx, policy.SourceRegion)
	if err != nil {
		return nil, fmt.Errorf("failed to get regional health: %w", err)
	}

	// Parse target regions from JSON
	var targetRegions []string
	if err := json.Unmarshal([]byte(policy.TargetRegions), &targetRegions); err != nil {
		return &FailoverConditionMet{Met: false}, nil
	}

	if len(targetRegions) == 0 {
		return &FailoverConditionMet{Met: false}, nil
	}

	// Check error rate threshold
	if policy.TriggerErrorRate != nil && metrics.ErrorRate > *policy.TriggerErrorRate {
		targetRegion := f.selectBestTargetRegion(ctx, targetRegions)
		return &FailoverConditionMet{
			Met:          true,
			TriggerType:  "error_rate",
			TriggerValue: metrics.ErrorRate,
			TargetRegion: targetRegion,
		}, nil
	}

	// Check latency threshold (p99 latency in ms)
	if policy.TriggerLatency != nil && metrics.P99Latency > *policy.TriggerLatency {
		targetRegion := f.selectBestTargetRegion(ctx, targetRegions)
		return &FailoverConditionMet{
			Met:          true,
			TriggerType:  "latency",
			TriggerValue: float64(metrics.P99Latency),
			TargetRegion: targetRegion,
		}, nil
	}

	// Check health score threshold
	if policy.TriggerHealthScore != nil && health != nil && health.Score < *policy.TriggerHealthScore {
		targetRegion := f.selectBestTargetRegion(ctx, targetRegions)
		return &FailoverConditionMet{
			Met:          true,
			TriggerType:  "health_score",
			TriggerValue: float64(health.Score),
			TargetRegion: targetRegion,
		}, nil
	}

	return &FailoverConditionMet{Met: false}, nil
}

// selectBestTargetRegion chooses the healthiest target region
func (f *FailoverOrchestrator) selectBestTargetRegion(ctx context.Context, targetRegions []string) string {
	if len(targetRegions) == 0 {
		return ""
	}

	// For now, use first target region; future: evaluate health of all targets
	return targetRegions[0]
}

// ExecuteFailover initiates a failover from source to target region
func (f *FailoverOrchestrator) ExecuteFailover(ctx context.Context, policy *FailoverPolicy, condition *FailoverConditionMet, incidentID *uuid.UUID) (*FailoverEvent, error) {
	// Create failover event (IncidentID can be nil)
	var incident uuid.UUID
	if incidentID != nil {
		incident = *incidentID
	}

	event := &FailoverEvent{
		PolicyID:      policy.ID,
		TenantID:      policy.TenantID,
		SourceRegion:  policy.SourceRegion,
		TargetRegion:  condition.TargetRegion,
		TriggerReason: condition.TriggerType,
		TriggerValue:  condition.TriggerValue,
		Status:        "in_progress",
		TriggeredAt:   time.Now().UTC(),
		IncidentID:    incident,
	}

	// Store failover event
	if err := f.store.InsertFailoverEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("failed to insert failover event: %w", err)
	}

	// Execute the failover action
	startTime := time.Now()
	var actionErr error

	// For automatic failover, execute region switch action
	if policy.IsAutomatic {
		// Create failover action parameters
		actionParams := map[string]interface{}{
			"action_type":    "region_switch",
			"source_region":  policy.SourceRegion,
			"target_region":  condition.TargetRegion,
			"trigger_reason": condition.TriggerType,
		}

		// Execute through action executor
		_, actionErr = f.actionExecutor.ExecuteRegionSwitch(ctx, actionParams)

		if actionErr != nil {
			// Log failure and update event
			event.Status = "failed"
			errMsg := actionErr.Error()
			event.ErrorMsg = &errMsg

			completedAt := time.Now().UTC()
			_ = f.store.UpdateFailoverEvent(ctx, event.ID, "failed", event.ErrorMsg, &completedAt)

			return event, fmt.Errorf("failover execution failed: %w", actionErr)
		}
	}

	// Update failover event to success
	event.Status = "success"
	completedAt := time.Now().UTC()
	_ = f.store.UpdateFailoverEvent(ctx, event.ID, "success", nil, &completedAt)

	// Update failover metrics
	f.updateFailoverMetrics(ctx, policy, true, time.Since(startTime))

	// Update region router to mark failover state
	_ = f.regionRouter.MarkRegionDown(ctx, policy.SourceRegion)
	_ = f.regionRouter.MarkRegionUp(ctx, condition.TargetRegion)

	return event, nil
}

// RollbackFailover initiates a rollback from target region back to source
func (f *FailoverOrchestrator) RollbackFailover(ctx context.Context, failoverEvent *FailoverEvent, policy *FailoverPolicy) error {
	now := time.Now().UTC()
	var rollbackErr error

	// Execute rollback action
	rollbackParams := map[string]interface{}{
		"action_type":    "region_switch",
		"source_region":  failoverEvent.TargetRegion,
		"target_region":  failoverEvent.SourceRegion,
		"trigger_reason": "failover_rollback",
	}

	_, err := f.actionExecutor.ExecuteRegionSwitch(ctx, rollbackParams)
	if err != nil {
		rollbackErr = err
		failoverEvent.Status = "rollback_failed"
	} else {
		failoverEvent.Status = "rolled_back"
		// Mark source region as healthy again
		_ = f.regionRouter.MarkRegionUp(ctx, failoverEvent.SourceRegion)
	}

	// Update failover event
	if rollbackErr != nil {
		msg := rollbackErr.Error()
		_ = f.store.UpdateFailoverEvent(ctx, failoverEvent.ID, "rollback_failed", &msg, &now)
	} else {
		_ = f.store.UpdateFailoverEvent(ctx, failoverEvent.ID, "rolled_back", nil, &now)
	}

	return rollbackErr
}

// updateFailoverMetrics tracks failover success rates
func (f *FailoverOrchestrator) updateFailoverMetrics(ctx context.Context, policy *FailoverPolicy, success bool, duration time.Duration) error {
	// Get or initialize metrics
	metrics, err := f.store.GetFailoverMetrics(ctx, policy.ID)
	if err != nil {
		return err
	}

	if metrics == nil {
		metrics = &FailoverMetrics{
			PolicyID:        policy.ID,
			TotalFailovers:  1,
			SuccessfulCount: 0,
			FailedCount:     0,
			AvgDurationMs:   int64(duration.Milliseconds()),
		}

		if success {
			metrics.SuccessfulCount = 1
		} else {
			metrics.FailedCount = 1
		}
	} else {
		metrics.TotalFailovers++
		if success {
			metrics.SuccessfulCount++
		} else {
			metrics.FailedCount++
		}

		// Update average duration
		totalDurationMs := int64(metrics.AvgDurationMs) * int64(metrics.TotalFailovers-1)
		totalDurationMs += int64(duration.Milliseconds())
		metrics.AvgDurationMs = totalDurationMs / int64(metrics.TotalFailovers)
	}

	// Calculate success rate
	if metrics.TotalFailovers > 0 {
		metrics.SuccessRatePct = (float64(metrics.SuccessfulCount) / float64(metrics.TotalFailovers)) * 100
	}

	now := time.Now().UTC()
	metrics.LastFailoverAt = &now

	return f.store.UpsertFailoverMetrics(ctx, metrics)
}

// EvaluateAllPolicies runs periodic evaluation of all active failover policies
// Returns a map of policy IDs to failover conditions that should trigger
func (f *FailoverOrchestrator) EvaluateAllPolicies(ctx context.Context, tenantID uuid.UUID) (map[uuid.UUID]*FailoverConditionMet, error) {
	// Get all active policies for tenant
	policies, err := f.store.ListFailoverPolicies(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("failed to list failover policies: %w", err)
	}

	result := make(map[uuid.UUID]*FailoverConditionMet)

	for _, policy := range policies {
		// Skip if not automatic
		if !policy.IsAutomatic {
			continue
		}

		// Check cooldown: don't evaluate policies that recently executed
		recentEvents, err := f.store.ListFailoverEvents(ctx, policy.ID, 1)
		if err == nil && len(recentEvents) > 0 {
			lastEvent := recentEvents[0]
			cooldownDuration := time.Duration(policy.CooldownMinutes) * time.Minute
			if time.Since(lastEvent.TriggeredAt) < cooldownDuration {
				continue // Skip this policy due to cooldown
			}
		}

		// Evaluate conditions
		condition, err := f.CheckFailoverConditions(ctx, &policy)
		if err != nil {
			// Log but continue with other policies
			continue
		}

		if condition.Met {
			result[policy.ID] = condition
		}
	}

	return result, nil
}
