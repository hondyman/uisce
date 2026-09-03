package events

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// OutboxEvent represents a row in the outbox table.
type OutboxEvent struct {
	ID          uuid.UUID       `db:"id"`
	EventType   string          `db:"event_type"`
	Payload     json.RawMessage `db:"payload"`
	Published   bool            `db:"published"`
	CreatedAt   time.Time       `db:"created_at"`
	PublishedAt sql.NullTime    `db:"published_at"`
}

// PublishEvent writes an event to the outbox table within a transaction.
func PublishEvent(ctx context.Context, tx *sqlx.Tx, eventType string, payload interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	q := `
		INSERT INTO outbox (event_type, payload)
		VALUES ($1, $2)
	`
	_, err = tx.ExecContext(ctx, q, eventType, payloadBytes)
	if err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}
	return nil
}

// EventHandlerFunc processes a single outbox event's decoded payload. It is
// used for event types that need application-level routing (e.g.
// "Pipeline.Trigger" invoking the data pipeline engine) instead of the
// default Kafka publish path.
type EventHandlerFunc func(ctx context.Context, payload map[string]interface{}) error

// ProcessOutbox reads unpublished events and dispatches them.
//
// For each event, if handlers contains an entry keyed by the event's
// EventType, that handler is invoked instead of the default Kafka publish
// path — this is how "Pipeline.Trigger" events (written by
// internal/validation's async trigger dispatch) get routed to
// datapipeline.PipelineEngine.ExecuteRunAsWorkflow without this package
// needing to import internal/datapipeline. Event types with no matching
// handler keep going through publisher.publishEvent, preserving existing
// behavior. handlers may be nil, in which case every event goes through the
// publisher as before.
//
// This is called by a background worker on a ticker (see cmd/worker's
// outbox-polling goroutine).
func ProcessOutbox(ctx context.Context, db *sqlx.DB, publisher *KafkaPublisher, handlers map[string]EventHandlerFunc) error {
	// Lock rows to prevent concurrent processing (SKIP LOCKED)
	q := `
		SELECT id, event_type, payload, published, created_at
		FROM outbox
		WHERE published = FALSE
		ORDER BY created_at ASC
		LIMIT 50
		FOR UPDATE SKIP LOCKED
	`

	rows, err := db.QueryxContext(ctx, q)
	if err != nil {
		return fmt.Errorf("query outbox: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var evt OutboxEvent
		if err := rows.StructScan(&evt); err != nil {
			continue
		}

		// Publish to broker (prefer Kafka/Redpanda)
		// Note: We are using a generic map for now; using strict typed event structs is preferred
		var payloadMap map[string]interface{}
		if err := json.Unmarshal(evt.Payload, &payloadMap); err != nil {
			// Mark as failed or log error; for now, skip
			continue
		}

		if handler, ok := handlers[evt.EventType]; ok {
			err = handler(ctx, payloadMap)
		} else if publisher != nil {
			// The publisher should implement a generic publish method (e.g., `PublishToTopic(ctx, topic, key, payload)`)
			// Prefer `KafkaPublisher` which exposes `PublishToTopic` for generic payloads. If using legacy `RabbitMQPublisher`,
			// provide an adapter or wrapper to maintain parity with Kafka publishing semantics.
			// Publish via configured publisher implementation
			err = publisher.publishEvent(ctx, EventType(evt.EventType), payloadMap)
		}
		if err != nil {
			// Leave unpublished for retry on the next tick.
			continue
		}

		// Mark as published
		updateQ := `UPDATE outbox SET published = TRUE, published_at = NOW() WHERE id = $1`
		_, err = db.ExecContext(ctx, updateQ, evt.ID)
		if err != nil {
			return fmt.Errorf("update outbox: %w", err)
		}
	}

	return nil
}
