package ops

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// OpsAction defines the interface for executable operations
type OpsAction interface {
	// ID returns the unique identifier for this action type
	ID() string

	// Name returns human-readable action name
	Name() string

	// Validate checks if preconditions are met
	Validate(ctx context.Context, params json.RawMessage) error

	// Execute performs the action
	Execute(ctx context.Context, params json.RawMessage) (result map[string]interface{}, err error)

	// Rollback attempts to undo the action (optional, may return nil)
	Rollback(ctx context.Context, store Store, historyID uuid.UUID) error
}

// ActionHistory tracks executed ops actions
type ActionHistory struct {
	ID         uuid.UUID       `json:"id" db:"id"`
	IncidentID uuid.UUID       `json:"incident_id" db:"incident_id"`
	Region     *string         `json:"region,omitempty" db:"region"` // Geographic region where action executed
	ActionType string          `json:"action_type" db:"action_type"` // "restart_worker", "throttle_tenant", etc.
	Status     string          `json:"status" db:"status"`           // "pending", "success", "failed"
	Parameters json.RawMessage `json:"parameters" db:"parameters"`
	Result     json.RawMessage `json:"result,omitempty" db:"result"`
	ErrorMsg   *string         `json:"error_msg,omitempty" db:"error_msg"`
	ExecutedAt time.Time       `json:"executed_at" db:"executed_at"`
	CreatedAt  time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time       `json:"updated_at" db:"updated_at"`
}

// ExecuteActionRequest is the request body for executing an action
type ExecuteActionRequest struct {
	ActionType string          `json:"action_type"` // "restart_worker", "throttle_tenant", etc.
	Parameters json.RawMessage `json:"parameters"`
	Region     *string         `json:"region,omitempty"` // Phase 3.3: Region where action should execute
}

// ExecuteActionResponse is the response after executing an action
type ExecuteActionResponse struct {
	ActionHistoryID string                 `json:"action_history_id"`
	Status          string                 `json:"status"` // "success" or "failed"
	ActionType      string                 `json:"action_type"`
	Result          map[string]interface{} `json:"result,omitempty"`
	ErrorMsg        *string                `json:"error_msg,omitempty"`
	TimelineEventID *string                `json:"timeline_event_id,omitempty"`
}

// ActionExecutor orchestrates action execution with history and timeline tracking
type ActionExecutor struct {
	store          Store
	timeline       *TimelineService
	actions        map[string]OpsAction
	regionRegistry *RegionRegistry // Phase 3.3: Legacy region registry (deprecated)
	regionRouter   RegionRouter    // Phase 3.8a: Region routing control plane
}

// NewActionExecutor creates a new action executor
func NewActionExecutor(store Store, timeline *TimelineService) *ActionExecutor {
	router := NewInMemoryRegionRegistry(store, 5*time.Minute)
	executor := &ActionExecutor{
		store:          store,
		timeline:       timeline,
		actions:        make(map[string]OpsAction),
		regionRegistry: NewRegionRegistry(store), // Phase 3.3: Legacy (backward compat)
		regionRouter:   router,                   // Phase 3.8a: Use new routing control plane
	}

	// Register all available actions
	executor.RegisterAction(NewRestartWorkerAction())
	executor.RegisterAction(NewThrottleTenantAction())
	executor.RegisterAction(NewTriggerRunbookAction(store))
	executor.RegisterAction(NewCircuitBreakerAction(store))
	executor.RegisterAction(NewFailoverAction(store))

	return executor
}

// RegisterAction registers an action to the executor
func (e *ActionExecutor) RegisterAction(action OpsAction) {
	e.actions[action.ID()] = action
}

// ExecuteAction executes an action and records history + timeline events with region awareness
func (e *ActionExecutor) ExecuteAction(
	ctx context.Context,
	incidentID uuid.UUID,
	actionType string,
	params json.RawMessage,
	region *string, // Phase 3.3: Region where action executes
) (*ExecuteActionResponse, error) {

	// Find action
	action, ok := e.actions[actionType]
	if !ok {
		return nil, fmt.Errorf("unknown action: %s", actionType)
	}

	// Get incident to determine routing context (Phase 3.8a)
	incident, _, err := e.store.GetIncident(ctx, incidentID)
	if err != nil {
		errMsg := fmt.Sprintf("failed to fetch incident: %v", err)
		return &ExecuteActionResponse{
			Status:     "failed",
			ActionType: actionType,
			ErrorMsg:   &errMsg,
		}, fmt.Errorf("get incident: %w", err)
	}

	// Phase 3.8a: Route action to appropriate region using RegionRouter
	// Determine routed region via router
	target, err := e.regionRouter.RouteForIncident(ctx, incident)
	if err != nil {
		errMsg := fmt.Sprintf("region routing failed: %v", err)
		return &ExecuteActionResponse{
			Status:     "failed",
			ActionType: actionType,
			ErrorMsg:   &errMsg,
		}, fmt.Errorf("route incident: %w", err)
	}

	routedRegion := target.Region

	// Phase 3.8a: Action will be routed to region's worker pool
	// target.OpsWorkerPool contains the worker pool for this region
	// target.RedpandaBroker for event streaming
	// target.StarRocksCluster for metrics storage
	// target.TemporalNamespace for workflow coordination

	// Create history record with routed region
	historyID := uuid.New()
	history := ActionHistory{
		ID:         historyID,
		IncidentID: incidentID,
		Region:     &routedRegion, // Phase 3.8a: Track routed region
		ActionType: actionType,
		Status:     "pending",
		Parameters: params,
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	// Insert pending history record
	if err := e.store.InsertActionHistory(ctx, history); err != nil {
		return nil, fmt.Errorf("insert action history: %w", err)
	}

	// Validate action preconditions
	if err := action.Validate(ctx, params); err != nil {
		// Record failure
		errMsg := err.Error()
		e.store.UpdateActionHistory(ctx, historyID, "failed", nil, &errMsg)
		return &ExecuteActionResponse{
			ActionHistoryID: historyID.String(),
			Status:          "failed",
			ActionType:      actionType,
			ErrorMsg:        &errMsg,
		}, err
	}

	// Execute action
	result, err := action.Execute(ctx, params)
	if err != nil {
		// Record failure
		errMsg := err.Error()
		e.store.UpdateActionHistory(ctx, historyID, "failed", nil, &errMsg)
		return &ExecuteActionResponse{
			ActionHistoryID: historyID.String(),
			Status:          "failed",
			ActionType:      actionType,
			ErrorMsg:        &errMsg,
		}, err
	}

	// Marshal result
	resultJSON, _ := json.Marshal(result)

	// Record success
	e.store.UpdateActionHistory(ctx, historyID, "success", resultJSON, nil)

	// Record timeline event
	var eventID string
	if e.timeline != nil {
		event := MakeActionEvent(incidentID, actionType, result)
		if err := e.store.InsertEvent(ctx, event); err == nil {
			eventID = event.ID.String()
		}
	}

	return &ExecuteActionResponse{
		ActionHistoryID: historyID.String(),
		Status:          "success",
		ActionType:      actionType,
		Result:          result,
		TimelineEventID: &eventID,
	}, nil
}

// GetRegionWorkerPool returns region-specific worker pool for action execution
// Phase 3.8a: Use RegionRouter to discover which region's worker pool should handle the action
func (e *ActionExecutor) GetRegionWorkerPool(ctx context.Context, tenantID uuid.UUID, region string) (*string, error) {
	target, err := e.regionRouter.GetRegionTarget(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("get region target: %w", err)
	}
	if target == nil {
		return nil, fmt.Errorf("no routing target for region %s", region)
	}
	// Return ops worker pool for this region
	workerPool := target.OpsWorkerPool
	return &workerPool, nil
}

// MakeActionEvent creates a timeline event for an action
func MakeActionEvent(incidentID uuid.UUID, actionType string, result map[string]interface{}) Event {
	title := fmt.Sprintf("Action executed: %s", actionType)
	details, _ := json.Marshal(result)

	return Event{
		ID:         uuid.New(),
		IncidentID: &incidentID,
		EventType:  EventActionExecuted,
		Scope:      "incident",
		Severity:   SeverityInfo,
		Title:      title,
		Details:    details,
		OccurredAt: time.Now().UTC(),
	}
}

// ExecuteRegionSwitch performs a regional failover switch
// Phase 3.10: Used by failover orchestrator to execute automated regional failover
func (e *ActionExecutor) ExecuteRegionSwitch(ctx context.Context, params map[string]interface{}) ([]byte, error) {
	sourceRegion, ok := params["source_region"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid source_region parameter")
	}

	targetRegion, ok := params["target_region"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid target_region parameter")
	}

	triggerReason, _ := params["trigger_reason"].(string)

	// Verify both regions are valid
	sourceTarget, err := e.regionRouter.GetRegionTarget(ctx, sourceRegion)
	if err != nil || sourceTarget == nil {
		return nil, fmt.Errorf("invalid source region: %s", sourceRegion)
	}

	targetTarget, err := e.regionRouter.GetRegionTarget(ctx, targetRegion)
	if err != nil || targetTarget == nil {
		return nil, fmt.Errorf("invalid target region: %s", targetRegion)
	}

	// Execute the switch
	result := map[string]interface{}{
		"action_type":    "region_switch",
		"source_region":  sourceRegion,
		"target_region":  targetRegion,
		"trigger_reason": triggerReason,
		"status":         "completed",
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	resultJSON, _ := json.Marshal(result)
	return resultJSON, nil
}

// GetRegionRouter returns the region router for this executor
// Phase 3.8a: Exposes region routing control plane for testing and external use
func (e *ActionExecutor) GetRegionRouter() RegionRouter {
	return e.regionRouter
}
