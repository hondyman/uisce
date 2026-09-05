package datapipeline

import (
	"context"
	"fmt"
	"log"

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
	TriggerID  string                 `json:"trigger_id,omitempty"`
	Record     map[string]interface{} `json:"record"`
}

// OutboxPublisher implements validation.PipelineOutboxPublisher by writing
// "Pipeline.Trigger" rows to the shared outbox table. A background poller
// (events.ProcessOutbox, wired in cmd/worker) later routes those rows to
// PipelineEngine.ExecuteRunAsWorkflow via NewPipelineTriggerOutboxHandler.
//
// Transactional guarantee: PublishPipelineTriggerTx accepts a *sqlx.Tx so the
// outbox row is committed atomically with the originating BO write. The legacy
// PublishPipelineTrigger opens its own transaction and is NOT atomic with the BO
// write — callers should migrate to PublishPipelineTriggerTx.
type OutboxPublisher struct {
	db *sqlx.DB
}

// NewOutboxPublisher creates an OutboxPublisher backed by db.
func NewOutboxPublisher(db *sqlx.DB) *OutboxPublisher {
	return &OutboxPublisher{db: db}
}

// PublishPipelineTrigger is the legacy method. Prefer PublishPipelineTriggerTx.
// Kept for backward compatibility with callers that haven't migrated to pass
// the caller's transaction.
func (p *OutboxPublisher) PublishPipelineTrigger(ctx context.Context, tenantID uuid.UUID, pipelineID uuid.UUID, triggerID uuid.UUID, record map[string]interface{}) error {
	log.Printf("[WARN] OutboxPublisher.PublishPipelineTrigger (legacy) called — migrate to PublishPipelineTriggerTx for transactional atomicity")
	return p.PublishPipelineTriggerTx(ctx, nil, tenantID, pipelineID, triggerID, record)
}

// PublishPipelineTriggerTx writes the outbox row inside the caller's transaction
// (if tx is non-nil) or its own short-lived transaction (if tx is nil, compat path).
func (p *OutboxPublisher) PublishPipelineTriggerTx(ctx context.Context, tx *sqlx.Tx, tenantID uuid.UUID, pipelineID uuid.UUID, triggerID uuid.UUID, record map[string]interface{}) error {
	if p.db == nil {
		return fmt.Errorf("outbox publisher has no database configured")
	}
	ownsTx := false
	if tx == nil {
		var err error
		tx, err = p.db.BeginTxx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin outbox tx: %w", err)
		}
		ownsTx = true
		defer tx.Rollback() //nolint:errcheck
	}

	payload := pipelineTriggerPayload{
		TenantID:   tenantID.String(),
		PipelineID: pipelineID.String(),
		TriggerID:  triggerID.String(),
		Record:     record,
	}
	if err := events.PublishEvent(ctx, tx, PipelineTriggerEventType, payload); err != nil {
		return err
	}
	if ownsTx {
		return tx.Commit()
	}
	return nil
}

// NewPipelineTriggerOutboxHandler returns an events.EventHandlerFunc that
// routes "Pipeline.Trigger" outbox events to engine.ExecuteRunAsWorkflow.
// Register it with events.ProcessOutbox's handlers map, keyed by
// PipelineTriggerEventType (see cmd/worker/main.go's outbox-polling loop).
func NewPipelineTriggerOutboxHandler(engine *PipelineEngine) events.EventHandlerFunc {
	return func(ctx context.Context, payload map[string]interface{}) error {
		tenantIDStr, _ := payload["tenant_id"].(string)
		pipelineIDStr, _ := payload["pipeline_id"].(string)
		triggerIDStr, _ := payload["trigger_id"].(string)
		record, _ := payload["record"].(map[string]interface{})

		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			return fmt.Errorf("invalid tenant_id in Pipeline.Trigger event: %w", err)
		}
		pipelineID, err := uuid.Parse(pipelineIDStr)
		if err != nil {
			return fmt.Errorf("invalid pipeline_id in Pipeline.Trigger event: %w", err)
		}

		var triggerID *uuid.UUID
		if triggerIDStr != "" {
			tid, err := uuid.Parse(triggerIDStr)
			if err == nil {
				triggerID = &tid
			}
		}

		def, err := engine.loadPipelineDefinition(ctx, tenantID, pipelineID)
		if err != nil {
			return err
		}

		if engine.temporalClient != nil {
			_, _, err := engine.ExecuteRunAsWorkflow(ctx, tenantID, *def, []PipelineRecord{record}, triggerID)
			return err
		}
		// No Temporal client configured (e.g. dev/test worker): fall back
		// to synchronous in-process execution so the trigger still runs.
		_, err = engine.ExecuteRun(ctx, tenantID, *def, []PipelineRecord{record}, false, triggerID)
		return err
	}
}
