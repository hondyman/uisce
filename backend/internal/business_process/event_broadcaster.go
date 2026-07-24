package business_process

import (
	"context"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
)

// EventBroadcaster broadcasts workflow events to WebSocket clients
type EventBroadcaster struct {
	db            *sqlx.DB
	broadcastFunc func(ProcessEvent)
}

// ProcessEvent represents a workflow event
type ProcessEvent struct {
	Type         string                 `json:"type"`
	WorkflowID   string                 `json:"workflow_id"`
	WorkflowType string                 `json:"workflow_type"`
	StepName     string                 `json:"step_name,omitempty"`
	Status       string                 `json:"status"`
	Timestamp    time.Time              `json:"timestamp"`
	TenantID     string                 `json:"tenant_id"`
	DatasourceID string                 `json:"datasource_id"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// NewEventBroadcaster creates a new event broadcaster
func NewEventBroadcaster(db *sqlx.DB, broadcastFunc func(ProcessEvent)) *EventBroadcaster {
	return &EventBroadcaster{
		db:            db,
		broadcastFunc: broadcastFunc,
	}
}

// BroadcastStepStart broadcasts a step start event
func (eb *EventBroadcaster) BroadcastStepStart(ctx context.Context, workflowID, workflowType, stepName, tenantID, datasourceID string, metadata map[string]interface{}) {
	if eb.broadcastFunc == nil {
		return
	}

	event := ProcessEvent{
		Type:         "step_started",
		WorkflowID:   workflowID,
		WorkflowType: workflowType,
		StepName:     stepName,
		Status:       "running",
		Timestamp:    time.Now(),
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		Metadata:     metadata,
	}

	go eb.broadcastFunc(event)
}

// BroadcastStepComplete broadcasts a step completion event
func (eb *EventBroadcaster) BroadcastStepComplete(ctx context.Context, workflowID, workflowType, stepName, tenantID, datasourceID string, duration time.Duration, metadata map[string]interface{}) {
	if eb.broadcastFunc == nil {
		return
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["duration_seconds"] = duration.Seconds()

	event := ProcessEvent{
		Type:         "step_completed",
		WorkflowID:   workflowID,
		WorkflowType: workflowType,
		StepName:     stepName,
		Status:       "completed",
		Timestamp:    time.Now(),
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		Metadata:     metadata,
	}

	go eb.broadcastFunc(event)
}

// BroadcastStepFailed broadcasts a step failure event
func (eb *EventBroadcaster) BroadcastStepFailed(ctx context.Context, workflowID, workflowType, stepName, tenantID, datasourceID, errorMsg string, metadata map[string]interface{}) {
	if eb.broadcastFunc == nil {
		return
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["error"] = errorMsg

	event := ProcessEvent{
		Type:         "step_failed",
		WorkflowID:   workflowID,
		WorkflowType: workflowType,
		StepName:     stepName,
		Status:       "failed",
		Timestamp:    time.Now(),
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		Metadata:     metadata,
	}

	go eb.broadcastFunc(event)
}

// BroadcastWorkflowStart broadcasts a workflow start event
func (eb *EventBroadcaster) BroadcastWorkflowStart(ctx context.Context, workflowID, workflowType, tenantID, datasourceID string, metadata map[string]interface{}) {
	if eb.broadcastFunc == nil {
		return
	}

	event := ProcessEvent{
		Type:         "workflow_started",
		WorkflowID:   workflowID,
		WorkflowType: workflowType,
		Status:       "running",
		Timestamp:    time.Now(),
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		Metadata:     metadata,
	}

	go eb.broadcastFunc(event)
	log.Printf("Broadcast workflow start: %s (%s)", workflowID, workflowType)
}

// BroadcastWorkflowComplete broadcasts a workflow completion event
func (eb *EventBroadcaster) BroadcastWorkflowComplete(ctx context.Context, workflowID, workflowType, tenantID, datasourceID string, duration time.Duration, metadata map[string]interface{}) {
	if eb.broadcastFunc == nil {
		return
	}

	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	metadata["duration_seconds"] = duration.Seconds()

	event := ProcessEvent{
		Type:         "workflow_completed",
		WorkflowID:   workflowID,
		WorkflowType: workflowType,
		Status:       "completed",
		Timestamp:    time.Now(),
		TenantID:     tenantID,
		DatasourceID: datasourceID,
		Metadata:     metadata,
	}

	go eb.broadcastFunc(event)
	log.Printf("Broadcast workflow complete: %s (%s) in %.2fs", workflowID, workflowType, duration.Seconds())
}
