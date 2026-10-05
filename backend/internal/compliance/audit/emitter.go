package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/compliance/canonical"
)

// EvaluationEventPayload represents the wire format of an evaluation event sent to Redpanda
type EvaluationEventPayload struct {
	ID              uuid.UUID              `json:"id"`
	LineageID       uuid.UUID              `json:"lineageId"`
	TenantID        uuid.UUID              `json:"tenantId"`
	OrderID         *uuid.UUID             `json:"orderId,omitempty"`
	RuleID          uuid.UUID              `json:"ruleId"`
	RuleVersion     int                    `json:"ruleVersion"`
	RuleContentHash string                 `json:"ruleContentHash"`
	Passed          bool                   `json:"passed"`
	ActionTaken     string                 `json:"actionTaken"`
	LatencyMicros   int64                  `json:"latencyMicros"`
	EvaluationHash  string                 `json:"evaluationHash"`
	InputParams     map[string]interface{} `json:"inputParams"`
	MetricSnapshots map[string]interface{} `json:"metricSnapshots"`
	EvaluatedAt     time.Time              `json:"evaluatedAt"`
}

// MessageBroker defines the interface for durable Redpanda/Kafka message producing & consuming
type MessageBroker interface {
	Publish(ctx context.Context, topic string, key string, payload []byte) error
	Subscribe(ctx context.Context, topic string, groupID string, handler func(key string, payload []byte) error) error
}

// DurableAuditEmitter publishes compliance evaluations durably to Redpanda with acks=1
type DurableAuditEmitter struct {
	broker MessageBroker
	topic  string
}

// NewDurableAuditEmitter creates a new durable emitter
func NewDurableAuditEmitter(broker MessageBroker, topic string) *DurableAuditEmitter {
	if topic == "" {
		topic = "compliance.evaluations"
	}
	return &DurableAuditEmitter{
		broker: broker,
		topic:  topic,
	}
}

// EmitEvaluation sends the evaluation event to the Redpanda durable log
func (e *DurableAuditEmitter) EmitEvaluation(ctx context.Context, event EvaluationEventPayload) error {
	if event.LineageID == uuid.Nil {
		return errors.New("audit: lineageId cannot be nil")
	}

	payloadBytes, err := canonical.Marshal(event)
	if err != nil {
		return fmt.Errorf("audit marshal: %w", err)
	}

	key := event.LineageID.String()
	return e.broker.Publish(ctx, e.topic, key, payloadBytes)
}

// HotTierAuditConsumer ingests evaluations from Redpanda and commits them to PostgreSQL
type HotTierAuditConsumer struct {
	db     *sql.DB
	broker MessageBroker
	topic  string
}

// NewHotTierAuditConsumer creates a new Hot Tier consumer
func NewHotTierAuditConsumer(db *sql.DB, broker MessageBroker, topic string) *HotTierAuditConsumer {
	if topic == "" {
		topic = "compliance.evaluations"
	}
	return &HotTierAuditConsumer{
		db:     db,
		broker: broker,
		topic:  topic,
	}
}

// ProcessBatch inserts a slice of events into PostgreSQL inside a single transaction
func (c *HotTierAuditConsumer) ProcessBatch(ctx context.Context, events []EvaluationEventPayload) error {
	if len(events) == 0 {
		return nil
	}

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO compliance.compliance_evaluation_event (
			id, lineage_id, tenant_id, order_id, rule_id, rule_version, rule_content_hash, passed, action_taken,
			latency_micros, evaluation_hash, input_params, metric_snapshots, evaluated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		) ON CONFLICT (lineage_id, evaluated_at) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("prepare stmt: %w", err)
	}
	defer stmt.Close()

	for _, ev := range events {
		inputBytes, _ := json.Marshal(ev.InputParams)
		metricBytes, _ := json.Marshal(ev.MetricSnapshots)

		if ev.ID == uuid.Nil {
			ev.ID = uuid.New()
		}
		if ev.EvaluatedAt.IsZero() {
			ev.EvaluatedAt = time.Now().UTC()
		}

		_, err := stmt.ExecContext(ctx,
			ev.ID, ev.LineageID, ev.TenantID, ev.OrderID, ev.RuleID, ev.RuleVersion, ev.RuleContentHash, ev.Passed, ev.ActionTaken,
			ev.LatencyMicros, ev.EvaluationHash, string(inputBytes), string(metricBytes), ev.EvaluatedAt,
		)
		if err != nil {
			return fmt.Errorf("insert eval event %s: %w", ev.LineageID, err)
		}
	}

	return tx.Commit()
}
