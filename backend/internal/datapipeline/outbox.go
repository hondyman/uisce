package datapipeline

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/hondyman/uisce/backend/internal/events"
)

// PipelineTriggerEventType is the outbox event type written by "async"
// DispatchMode validation triggers (internal/validation.TriggerValidationEngine)
// and consumed by ProcessPipelineTriggerOutboxEvent.
const PipelineTriggerEventType = "Pipeline.Trigger"

// pipelineTriggerPayload is the outbox payload shape for PipelineTriggerEventType.
type pipelineTriggerPayload struct {
	TenantID   string                 `json:"tenant_id"`
	PipelineID string                 `json:"pipeline_id"`
	Record     map[string]interface{} `json:"record"`
}

// OutboxPublisher implements validation.PipelineOutboxPublisher by writing
// "Pipeline.Trigger" rows to the shared outbox table. A background poller
// (events.ProcessOutbox, wired in cmd/worker) later routes those rows to
// PipelineEngine.ExecuteRunAsWorkflow via NewPipelineTriggerOutboxHandler.
//
// NOTE: this begins its own short-lived transaction rather than joining the
// caller's BO-write transaction — internal/validation.TriggerValidationEngine
// only holds a *sql.DB, not the transaction the BO write path
// (internal/metadata/businessobject_service.go,
// internal/services/business_object_service.go) uses, and threading that
// transaction handle through the validation engine's public API was out of
// scope for this pass. This is an at-least-once, not exactly-once dispatch:
// in the rare case the outer BO write later fails/rolls back after this
// commits, a pipeline trigger can fire for a write that didn't happen.
// Follow-up: accept an *sqlx.Tx on TriggerValidate/DispatchTrigger so this
// publish can join the caller's transaction like the existing
// "BusinessObject.CatalogSync" event does.
type OutboxPublisher struct {
	db *sqlx.DB
}

// NewOutboxPublisher creates an OutboxPublisher backed by db.
func NewOutboxPublisher(db *sqlx.DB) *OutboxPublisher {
	return &OutboxPublisher{db: db}
}

// PublishPipelineTrigger implements validation.PipelineOutboxPublisher.
func (p *OutboxPublisher) PublishPipelineTrigger(ctx context.Context, tenantID uuid.UUID, pipelineID uuid.UUID, record map[string]interface{}) error {
	if p.db == nil {
		return fmt.Errorf("outbox publisher has no database configured")
	}
	tx, err := p.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin outbox tx: %w", err)
	}
	defer tx.Rollback() //nolint:errcheck

	payload := pipelineTriggerPayload{
		TenantID:   tenantID.String(),
		PipelineID: pipelineID.String(),
		Record:     record,
	}
	if err := events.PublishEvent(ctx, tx, PipelineTriggerEventType, payload); err != nil {
		return err
	}
	return tx.Commit()
}

// NewPipelineTriggerOutboxHandler returns an events.EventHandlerFunc that
// routes "Pipeline.Trigger" outbox events to engine.ExecuteRunAsWorkflow.
// Register it with events.ProcessOutbox's handlers map, keyed by
// PipelineTriggerEventType (see cmd/worker/main.go's outbox-polling loop).
func NewPipelineTriggerOutboxHandler(engine *PipelineEngine) events.EventHandlerFunc {
	return func(ctx context.Context, payload map[string]interface{}) error {
		tenantIDStr, _ := payload["tenant_id"].(string)
		pipelineIDStr, _ := payload["pipeline_id"].(string)
		record, _ := payload["record"].(map[string]interface{})

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			return fmt.Errorf("invalid tenant_id in Pipeline.Trigger event: %w", err)
		}
		pipelineID, err := uuid.Parse(pipelineIDStr)
		if err != nil {
			return fmt.Errorf("invalid pipeline_id in Pipeline.Trigger event: %w", err)
		}

		def, err := engine.loadPipelineDefinition(ctx, tenantID, pipelineID)
		if err != nil {
			return err
		}

		if engine.temporalClient != nil {
			_, err := engine.ExecuteRunAsWorkflow(ctx, tenantID, *def, []PipelineRecord{record})
			return err
		}
		// No Temporal client configured (e.g. dev/test worker): fall back
		// to synchronous in-process execution so the trigger still runs.
		_, err = engine.ExecuteRun(ctx, tenantID, *def, []PipelineRecord{record}, false)
		return err
	}
}
