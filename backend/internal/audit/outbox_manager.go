package audit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// OutboxEvent represents an atomic governance audit record in the outbox ledger.
type OutboxEvent struct {
	EventID        uuid.UUID       `db:"event_id" json:"event_id"`
	TenantID       uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	AggregateType  string          `db:"aggregate_type" json:"aggregate_type"`
	AggregateID    uuid.UUID       `db:"aggregate_id" json:"aggregate_id"`
	EventType      string          `db:"event_type" json:"event_type"`
	ActorID        string          `db:"actor_id" json:"actor_id"`
	ActorRole      string          `db:"actor_role" json:"actor_role"`
	IdempotencyKey string          `db:"idempotency_key" json:"idempotency_key"`
	Payload        json.RawMessage `db:"payload" json:"payload"`
	PayloadHash    string          `db:"payload_hash" json:"payload_hash"`
	ChainSeal      sql.NullString  `db:"chain_seal" json:"chain_seal"`
	RetryCount     int             `db:"retry_count" json:"retry_count"`
	CreatedAt      time.Time       `db:"created_at" json:"created_at"`
	PublishedAt    *time.Time      `db:"published_at" json:"published_at,omitempty"`
	LastError      sql.NullString  `db:"last_error" json:"last_error,omitempty"`
}

// EventPublisher abstracts streaming event publishing (e.g. Redpanda, Kafka, or Mock).
type EventPublisher interface {
	Publish(ctx context.Context, topic, key string, payload []byte) error
}

// MemoryEventPublisher is an in-memory publisher for testing and standalone dev mode.
type MemoryEventPublisher struct {
	PublishedMessages []struct {
		Topic   string
		Key     string
		Payload []byte
	}
}

func NewMemoryEventPublisher() *MemoryEventPublisher {
	return &MemoryEventPublisher{
		PublishedMessages: make([]struct {
			Topic   string
			Key     string
			Payload []byte
		}, 0),
	}
}

func (p *MemoryEventPublisher) Publish(ctx context.Context, topic, key string, payload []byte) error {
	p.PublishedMessages = append(p.PublishedMessages, struct {
		Topic   string
		Key     string
		Payload []byte
	}{Topic: topic, Key: key, Payload: payload})
	return nil
}

// TransactionalOutboxManager coordinates atomic staging and asynchronous outbox relay.
type TransactionalOutboxManager struct {
	db        *sqlx.DB
	publisher EventPublisher
	hmacKey   []byte
}

// NewTransactionalOutboxManager creates a new TransactionalOutboxManager instance.
func NewTransactionalOutboxManager(db *sqlx.DB, publisher EventPublisher, hmacKey []byte) *TransactionalOutboxManager {
	if len(hmacKey) == 0 {
		hmacKey = []byte("uisce-default-sec-rule-17a4-hmac-key")
	}
	return &TransactionalOutboxManager{
		db:        db,
		publisher: publisher,
		hmacKey:   hmacKey,
	}
}

// CalculatePayloadHash computes the cryptographic SHA-256 hash of the event payload.
func CalculatePayloadHash(payloadBytes []byte) string {
	hash := sha256.Sum256(payloadBytes)
	return hex.EncodeToString(hash[:])
}

// CalculateChainSeal generates a tamper-evident HMAC-SHA256 seal for an audit event.
func CalculateChainSeal(hmacKey []byte, tenantID uuid.UUID, eventType, payloadHash string, timestamp time.Time) string {
	mac := hmac.New(sha256.New, hmacKey)
	mac.Write([]byte(fmt.Sprintf("%s:%s:%s:%s", tenantID, eventType, payloadHash, timestamp.UTC().Format(time.RFC3339))))
	return hex.EncodeToString(mac.Sum(nil))
}

// StageOutboxEventAtomic writes the audit record within the caller's active database transaction.
func (m *TransactionalOutboxManager) StageOutboxEventAtomic(
	ctx context.Context,
	tx *sqlx.Tx,
	tenantID, aggregateID uuid.UUID,
	aggregateType, eventType, actorID, actorRole string,
	payload interface{},
) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed marshaling outbox payload: %w", err)
	}

	payloadHash := CalculatePayloadHash(payloadBytes)
	idempotencyKey := fmt.Sprintf("%s:%s:%s", aggregateID, eventType, payloadHash[:16])
	chainSeal := CalculateChainSeal(m.hmacKey, tenantID, eventType, payloadHash, time.Now().UTC())

	query := `
		INSERT INTO public.catalog_outbox_events (
			tenant_id, aggregate_type, aggregate_id, event_type,
			actor_id, actor_role, idempotency_key, payload,
			payload_hash, chain_seal, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		ON CONFLICT (tenant_id, idempotency_key) DO NOTHING;
	`
	_, err = tx.ExecContext(ctx, query,
		tenantID, aggregateType, aggregateID, eventType,
		actorID, actorRole, idempotencyKey, payloadBytes,
		payloadHash, chainSeal,
	)
	return err
}

// RelayPendingEvents drains the outbox table concurrently using FOR UPDATE SKIP LOCKED.
func (m *TransactionalOutboxManager) RelayPendingEvents(ctx context.Context, batchSize int) (int, error) {
	if m.db == nil || m.publisher == nil {
		return 0, nil
	}

	tx, err := m.db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	query := `
		SELECT event_id, tenant_id, aggregate_type, aggregate_id, event_type,
		       actor_id, actor_role, idempotency_key, payload, payload_hash,
		       chain_seal, retry_count, created_at
		FROM public.catalog_outbox_events
		WHERE published_at IS NULL
		ORDER BY created_at ASC
		LIMIT $1
		FOR UPDATE SKIP LOCKED;
	`
	var events []OutboxEvent
	if err := tx.SelectContext(ctx, &events, query, batchSize); err != nil {
		return 0, fmt.Errorf("failed fetching pending outbox events: %w", err)
	}

	if len(events) == 0 {
		return 0, nil
	}

	for _, evt := range events {
		topic := "uisce.catalog.mutations.v1"
		if evt.AggregateType == "BUSINESS_OBJECT" || evt.EventType == "MAPPING_REJECTED" {
			topic = "uisce.audit.governance.v1"
		}

		eventEnvelope, _ := json.Marshal(evt)
		pubErr := m.publisher.Publish(ctx, topic, evt.TenantID.String(), eventEnvelope)

		if pubErr != nil {
			_, _ = tx.ExecContext(ctx, `
				UPDATE public.catalog_outbox_events
				SET retry_count = retry_count + 1, last_error = $1
				WHERE event_id = $2;
			`, pubErr.Error(), evt.EventID)
			continue
		}

		_, err = tx.ExecContext(ctx, `
			UPDATE public.catalog_outbox_events
			SET published_at = NOW(), last_error = NULL
			WHERE event_id = $1;
		`, evt.EventID)
		if err != nil {
			return 0, err
		}
	}

	return len(events), tx.Commit()
}
