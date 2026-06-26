package workflows

import (
	"context"
	"encoding/json"
	"time"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/workflow"
)

// WorkflowEvent represents a compliance event in the glass box
type WorkflowEvent struct {
	ID          string    `json:"id"`
	WorkflowID  string    `json:"workflow_id"`
	RunID       string    `json:"run_id"`
	EventType   string    `json:"event_type"` // DECISION, ACTION, INPUT, OUTPUT
	Timestamp   time.Time `json:"timestamp"`
	Description string    `json:"description"`
	Data        any       `json:"data"`
}

// ComplianceActivity logs events to the immutable audit trail
func ComplianceActivity(ctx context.Context, event WorkflowEvent) error {
	// In a real app, this would write to the audit service/DB
	logger := activity.GetLogger(ctx)
	dataJSON, _ := json.Marshal(event.Data)
	logger.Info("COMPLIANCE_EVENT",
		"type", event.EventType,
		"desc", event.Description,
		"data", string(dataJSON),
	)
	return nil
}

// RecordComplianceEvent is a helper to log events from workflows
func RecordComplianceEvent(ctx workflow.Context, eventType, desc string, data any) error {
	info := workflow.GetInfo(ctx)
	event := WorkflowEvent{
		WorkflowID:  info.WorkflowExecution.ID,
		RunID:       info.WorkflowExecution.RunID,
		EventType:   eventType,
		Timestamp:   workflow.Now(ctx),
		Description: desc,
		Data:        data,
	}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
	})

	return workflow.ExecuteActivity(ctx, ComplianceActivity, event).Get(ctx, nil)
}
