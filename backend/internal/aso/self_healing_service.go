package aso

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// ============================================================================
// Self-Healing Types
// ============================================================================

// HealingAction represents a self-healing action taken
type HealingAction struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	TargetType  TargetType        `json:"target_type" db:"target_type"`
	TargetID    uuid.UUID         `json:"target_id" db:"target_id"`
	TargetName  string            `json:"target_name" db:"target_name"`
	TenantID    *uuid.UUID        `json:"tenant_id,omitempty" db:"tenant_id"`
	Env         string            `json:"env" db:"env"`
	ActionType  HealingActionType `json:"action_type" db:"action_type"`
	Trigger     string            `json:"trigger" db:"trigger"` // What triggered this
	Status      HealingStatus     `json:"status" db:"status"`
	Details     json.RawMessage   `json:"details" db:"details"`
	StartedAt   time.Time         `json:"started_at" db:"started_at"`
	CompletedAt *time.Time        `json:"completed_at,omitempty" db:"completed_at"`
	Error       *string           `json:"error,omitempty" db:"error"`
	RetryCount  int               `json:"retry_count" db:"retry_count"`
}

// HealingActionType categorizes the healing action
type HealingActionType string

const (
	HealingRetryRefresh     HealingActionType = "retry_refresh"
	HealingRebuildPreAgg    HealingActionType = "rebuild_preagg"
	HealingUpdateDefinition HealingActionType = "update_definition"
	HealingAdjustInterval   HealingActionType = "adjust_interval"
	HealingExpandCoverage   HealingActionType = "expand_coverage"
	HealingDeprecateUnused  HealingActionType = "deprecate_unused"
)

// HealingStatus tracks the healing action progress
type HealingStatus string

const (
	HealingPending    HealingStatus = "pending"
	HealingInProgress HealingStatus = "in_progress"
	HealingSuccess    HealingStatus = "success"
	HealingFailed     HealingStatus = "failed"
	HealingSkipped    HealingStatus = "skipped"
)

// ============================================================================
// Self-Healing Service
// ============================================================================

// SelfHealingService provides automatic recovery for optimization issues
type SelfHealingService interface {
	// ProcessDriftSignal attempts to auto-heal a drift signal
	ProcessDriftSignal(ctx context.Context, signal *DriftSignal) (*HealingAction, error)

	// RetryFailedRefresh retries a failed pre-agg refresh
	RetryFailedRefresh(ctx context.Context, targetID uuid.UUID) (*HealingAction, error)

	// RebuildPreAgg triggers a full rebuild of a pre-agg
	RebuildPreAgg(ctx context.Context, targetID uuid.UUID) (*HealingAction, error)

	// UpdatePreAggDefinition updates a pre-agg definition based on new patterns
	UpdatePreAggDefinition(ctx context.Context, targetID uuid.UUID, newMeasures, newGrains []string) (*HealingAction, error)

	// AdjustRefreshInterval adjusts refresh interval based on staleness
	AdjustRefreshInterval(ctx context.Context, targetID uuid.UUID, newInterval time.Duration) (*HealingAction, error)

	// GetHealingHistory returns healing actions for a target
	GetHealingHistory(ctx context.Context, targetID uuid.UUID, limit int) ([]HealingAction, error)

	// GetPendingActions returns pending healing actions
	GetPendingActions(ctx context.Context, env string) ([]HealingAction, error)
}

// selfHealingService implements SelfHealingService
type selfHealingService struct {
	db             *sqlx.DB
	optRepo        ASOOptimizationRepository
	anomalyService AnomalyDetectionService
	maxRetries     int
	retryBackoff   time.Duration
}

// NewSelfHealingService creates a new self-healing service
func NewSelfHealingService(
	db *sqlx.DB,
	optRepo ASOOptimizationRepository,
	anomalyService AnomalyDetectionService,
) SelfHealingService {
	return &selfHealingService{
		db:             db,
		optRepo:        optRepo,
		anomalyService: anomalyService,
		maxRetries:     3,
		retryBackoff:   5 * time.Minute,
	}
}

// ProcessDriftSignal attempts to auto-heal a drift signal
func (s *selfHealingService) ProcessDriftSignal(ctx context.Context, signal *DriftSignal) (*HealingAction, error) {
	// Determine healing action based on signal type
	switch signal.SignalType {
	case DriftSignalRefreshFailure:
		return s.RetryFailedRefresh(ctx, signal.TargetID)

	case DriftSignalMissRateSpike:
		// Analyze miss patterns and expand coverage
		var evidence MissRateEvidence
		_ = json.Unmarshal(signal.Evidence, &evidence)
		if len(evidence.CommonMissPatterns) > 0 {
			return s.expandPreAggCoverage(ctx, signal.TargetID, evidence.CommonMissPatterns)
		}

	case DriftSignalLatencyRegression:
		// Try rebuilding the pre-agg
		return s.RebuildPreAgg(ctx, signal.TargetID)

	case DriftSignalStaleData:
		// Tighten refresh interval
		return s.AdjustRefreshInterval(ctx, signal.TargetID, 30*time.Minute)

	case DriftSignalUsageDecline:
		// Mark for deprecation review
		return s.markForDeprecation(ctx, signal.TargetID)
	}

	return nil, nil
}

// RetryFailedRefresh retries a failed pre-agg refresh
func (s *selfHealingService) RetryFailedRefresh(ctx context.Context, targetID uuid.UUID) (*HealingAction, error) {
	action := &HealingAction{
		ID:         uuid.New(),
		TargetType: TargetTypePreAgg,
		TargetID:   targetID,
		ActionType: HealingRetryRefresh,
		Trigger:    "refresh_failure_detected",
		Status:     HealingInProgress,
		StartedAt:  time.Now(),
	}

	// Persist the action
	if err := s.persistAction(ctx, action); err != nil {
		return nil, err
	}

	// Call pre-agg service to trigger refresh
	// This would integrate with your scheduler/pre-agg service
	err := s.triggerRefresh(ctx, targetID)

	now := time.Now()
	action.CompletedAt = &now

	if err != nil {
		errStr := err.Error()
		action.Error = &errStr
		action.Status = HealingFailed
		action.RetryCount++

		// Schedule retry if under max
		if action.RetryCount < s.maxRetries {
			s.scheduleRetry(ctx, action, s.retryBackoff*time.Duration(action.RetryCount))
		}
	} else {
		action.Status = HealingSuccess
		// Resolve the drift signal
		s.anomalyService.ResolveSignal(ctx, targetID, "self_healing", true)
	}

	s.updateAction(ctx, action)
	return action, nil
}

// RebuildPreAgg triggers a full rebuild of a pre-agg
func (s *selfHealingService) RebuildPreAgg(ctx context.Context, targetID uuid.UUID) (*HealingAction, error) {
	action := &HealingAction{
		ID:         uuid.New(),
		TargetType: TargetTypePreAgg,
		TargetID:   targetID,
		ActionType: HealingRebuildPreAgg,
		Trigger:    "latency_regression_detected",
		Status:     HealingInProgress,
		StartedAt:  time.Now(),
	}

	if err := s.persistAction(ctx, action); err != nil {
		return nil, err
	}

	// Trigger rebuild via pre-agg service
	// This would call your pre-agg service
	err := s.triggerRebuild(ctx, targetID)

	now := time.Now()
	action.CompletedAt = &now

	if err != nil {
		errStr := err.Error()
		action.Error = &errStr
		action.Status = HealingFailed
	} else {
		action.Status = HealingSuccess
	}

	s.updateAction(ctx, action)
	return action, nil
}

// UpdatePreAggDefinition updates a pre-agg definition based on new patterns
func (s *selfHealingService) UpdatePreAggDefinition(ctx context.Context, targetID uuid.UUID, newMeasures, newGrains []string) (*HealingAction, error) {
	details := map[string]interface{}{
		"new_measures": newMeasures,
		"new_grains":   newGrains,
	}
	detailsJSON, _ := json.Marshal(details)

	action := &HealingAction{
		ID:         uuid.New(),
		TargetType: TargetTypePreAgg,
		TargetID:   targetID,
		ActionType: HealingUpdateDefinition,
		Trigger:    "pattern_change_detected",
		Status:     HealingInProgress,
		Details:    detailsJSON,
		StartedAt:  time.Now(),
	}

	if err := s.persistAction(ctx, action); err != nil {
		return nil, err
	}

	// Update definition via pre-agg service
	// This would call your pre-agg service to update the definition
	err := s.updateDefinition(ctx, targetID, newMeasures, newGrains)

	now := time.Now()
	action.CompletedAt = &now

	if err != nil {
		errStr := err.Error()
		action.Error = &errStr
		action.Status = HealingFailed
	} else {
		action.Status = HealingSuccess
	}

	s.updateAction(ctx, action)
	return action, nil
}

// AdjustRefreshInterval adjusts refresh interval based on staleness
func (s *selfHealingService) AdjustRefreshInterval(ctx context.Context, targetID uuid.UUID, newInterval time.Duration) (*HealingAction, error) {
	details := map[string]interface{}{
		"new_interval": newInterval.String(),
	}
	detailsJSON, _ := json.Marshal(details)

	action := &HealingAction{
		ID:         uuid.New(),
		TargetType: TargetTypePreAgg,
		TargetID:   targetID,
		ActionType: HealingAdjustInterval,
		Trigger:    "staleness_detected",
		Status:     HealingInProgress,
		Details:    detailsJSON,
		StartedAt:  time.Now(),
	}

	if err := s.persistAction(ctx, action); err != nil {
		return nil, err
	}

	// Update interval via scheduler/pre-agg service
	err := s.setRefreshInterval(ctx, targetID, newInterval)

	now := time.Now()
	action.CompletedAt = &now

	if err != nil {
		errStr := err.Error()
		action.Error = &errStr
		action.Status = HealingFailed
	} else {
		action.Status = HealingSuccess
	}

	s.updateAction(ctx, action)
	return action, nil
}

// expandPreAggCoverage adds new patterns to a pre-agg
func (s *selfHealingService) expandPreAggCoverage(ctx context.Context, targetID uuid.UUID, patterns []string) (*HealingAction, error) {
	details := map[string]interface{}{
		"patterns_to_add": patterns,
	}
	detailsJSON, _ := json.Marshal(details)

	action := &HealingAction{
		ID:         uuid.New(),
		TargetType: TargetTypePreAgg,
		TargetID:   targetID,
		ActionType: HealingExpandCoverage,
		Trigger:    "miss_rate_spike",
		Status:     HealingInProgress,
		Details:    detailsJSON,
		StartedAt:  time.Now(),
	}

	if err := s.persistAction(ctx, action); err != nil {
		return nil, err
	}

	// Expand coverage via pre-agg service
	// This is a placeholder - would integrate with actual service

	action.Status = HealingSuccess
	now := time.Now()
	action.CompletedAt = &now

	s.updateAction(ctx, action)
	return action, nil
}

// markForDeprecation marks an unused asset for deprecation
func (s *selfHealingService) markForDeprecation(ctx context.Context, targetID uuid.UUID) (*HealingAction, error) {
	action := &HealingAction{
		ID:         uuid.New(),
		TargetType: TargetTypePreAgg,
		TargetID:   targetID,
		ActionType: HealingDeprecateUnused,
		Trigger:    "usage_decline",
		Status:     HealingSuccess,
		StartedAt:  time.Now(),
	}

	now := time.Now()
	action.CompletedAt = &now

	if err := s.persistAction(ctx, action); err != nil {
		return nil, err
	}

	return action, nil
}

// GetHealingHistory returns healing actions for a target
func (s *selfHealingService) GetHealingHistory(ctx context.Context, targetID uuid.UUID, limit int) ([]HealingAction, error) {
	var actions []HealingAction
	err := s.db.SelectContext(ctx, &actions, `
		SELECT * FROM semantic.healing_action
		WHERE target_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`, targetID, limit)
	return actions, err
}

// GetPendingActions returns pending healing actions
func (s *selfHealingService) GetPendingActions(ctx context.Context, env string) ([]HealingAction, error) {
	var actions []HealingAction
	err := s.db.SelectContext(ctx, &actions, `
		SELECT * FROM semantic.healing_action
		WHERE env = $1 AND status IN ('pending', 'in_progress')
		ORDER BY started_at ASC
	`, env)
	return actions, err
}

// ============================================================================
// Internal Methods
// ============================================================================

func (s *selfHealingService) persistAction(ctx context.Context, action *HealingAction) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO semantic.healing_action (
			id, target_type, target_id, target_name, tenant_id, env,
			action_type, trigger, status, details, started_at, retry_count
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`, action.ID, action.TargetType, action.TargetID, action.TargetName,
		action.TenantID, action.Env, action.ActionType, action.Trigger,
		action.Status, action.Details, action.StartedAt, action.RetryCount)
	return err
}

func (s *selfHealingService) updateAction(ctx context.Context, action *HealingAction) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE semantic.healing_action
		SET status = $2, completed_at = $3, error = $4, retry_count = $5
		WHERE id = $1
	`, action.ID, action.Status, action.CompletedAt, action.Error, action.RetryCount)
	return err
}

func (s *selfHealingService) triggerRefresh(ctx context.Context, targetID uuid.UUID) error {
	// Integration point with pre-agg/scheduler service
	return nil
}

func (s *selfHealingService) triggerRebuild(ctx context.Context, targetID uuid.UUID) error {
	// Integration point with pre-agg service
	return nil
}

func (s *selfHealingService) updateDefinition(ctx context.Context, targetID uuid.UUID, measures, grains []string) error {
	// Integration point with pre-agg service
	return nil
}

func (s *selfHealingService) setRefreshInterval(ctx context.Context, targetID uuid.UUID, interval time.Duration) error {
	// Integration point with scheduler service
	return nil
}

func (s *selfHealingService) scheduleRetry(ctx context.Context, action *HealingAction, delay time.Duration) {
	// Would use Temporal or job scheduler to retry after delay
	fmt.Printf("Scheduling retry for action %s in %s\n", action.ID, delay)
}
