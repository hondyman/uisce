package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// SecurityAuditEvent represents an access rule change event.
type SecurityAuditEvent struct {
	EventID          string                 `json:"event_id"`
	EventType        string                 `json:"event_type"` // "rule.created", "rule.updated", "rule.deleted", "rule.promoted"
	TenantID         string                 `json:"tenant_id"`
	RuleID           string                 `json:"rule_id"`
	BusinessObjectID string                 `json:"business_object_id"`
	GroupDN          string                 `json:"group_dn"`
	AccessLevel      string                 `json:"access_level"`
	ActorID          string                 `json:"actor_id"`
	Timestamp        time.Time              `json:"timestamp"`
	OldValue         map[string]interface{} `json:"old_value,omitempty"`
	NewValue         map[string]interface{} `json:"new_value"`
	Environment      string                 `json:"environment"` // "dev", "staging", "prod"
	IPAddress        string                 `json:"ip_address,omitempty"`
	UserAgent        string                 `json:"user_agent,omitempty"`
}

// SecuritySnapshotEvent represents a full snapshot of an access rule for Iceberg.
type SecuritySnapshotEvent struct {
	SnapshotID       string                 `json:"snapshot_id"`
	TenantID         string                 `json:"tenant_id"`
	RuleID           string                 `json:"rule_id"`
	BusinessObjectID string                 `json:"business_object_id"`
	GroupDN          string                 `json:"group_dn"`
	AccessLevel      string                 `json:"access_level"`
	Status           string                 `json:"status"`
	RowFilterDsl     string                 `json:"row_filter_dsl"`
	ColumnMasks      []ColumnMaskSnapshot   `json:"column_masks"`
	AppliesToApis    bool                   `json:"applies_to_apis"`
	AppliesToBi      bool                   `json:"applies_to_bi"`
	AppliesToAi      bool                   `json:"applies_to_ai"`
	CreatedBy        string                 `json:"created_by"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedBy        string                 `json:"updated_by"`
	UpdatedAt        time.Time              `json:"updated_at"`
	SnapshotTime     time.Time              `json:"snapshot_time"`
	Version          int                    `json:"version"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

type ColumnMaskSnapshot struct {
	SemanticTermID string `json:"semantic_term_id"`
	MaskType       string `json:"mask_type"`
}

// PublishSecurityAuditEvent publishes an audit event to the outbox for async processing.
func PublishSecurityAuditEvent(ctx context.Context, tx *sqlx.Tx, event SecurityAuditEvent) error {
	if event.EventID == "" {
		event.EventID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal audit event: %w", err)
	}

	q := `
		INSERT INTO outbox (event_type, payload, created_at)
		VALUES ($1, $2, $3)
	`
	_, err = tx.ExecContext(ctx, q, "security.audit."+event.EventType, payload, time.Now())
	if err != nil {
		return fmt.Errorf("insert security audit to outbox: %w", err)
	}

	return nil
}

// PublishSecuritySnapshotEvent publishes a snapshot event to the outbox for Iceberg.
func PublishSecuritySnapshotEvent(ctx context.Context, tx *sqlx.Tx, event SecuritySnapshotEvent) error {
	if event.SnapshotID == "" {
		event.SnapshotID = uuid.New().String()
	}
	if event.SnapshotTime.IsZero() {
		event.SnapshotTime = time.Now()
	}

	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal snapshot event: %w", err)
	}

	q := `
		INSERT INTO outbox (event_type, payload, created_at)
		VALUES ($1, $2, $3)
	`
	_, err = tx.ExecContext(ctx, q, "security.snapshot", payload, time.Now())
	if err != nil {
		return fmt.Errorf("insert security snapshot to outbox: %w", err)
	}

	return nil
}

// KafkaSecurityPublisher publishes security events to Kafka for Trino/Iceberg ingestion.
type KafkaSecurityPublisher struct {
	kafkaPublisher *KafkaPublisher
}

// NewKafkaSecurityPublisher creates a new security event publisher.
func NewKafkaSecurityPublisher(brokers string) *KafkaSecurityPublisher {
	return &KafkaSecurityPublisher{
		kafkaPublisher: NewKafkaPublisher(brokers),
	}
}

// PublishAuditToKafka publishes audit events to the security.audit topic.
func (p *KafkaSecurityPublisher) PublishAuditToKafka(ctx context.Context, event SecurityAuditEvent) error {
	return p.kafkaPublisher.PublishToTopic(ctx, "security.audit", event.EventID, event)
}

// PublishSnapshotToKafka publishes snapshot events to the security.snapshot topic.
func (p *KafkaSecurityPublisher) PublishSnapshotToKafka(ctx context.Context, event SecuritySnapshotEvent) error {
	return p.kafkaPublisher.PublishToTopic(ctx, "security.snapshot", event.SnapshotID, event)
}

// ProcessSecurityOutbox processes security events from the outbox and publishes to Kafka.
// This runs as a background worker to decouple main API processing.
func ProcessSecurityOutbox(ctx context.Context, db *sqlx.DB, publisher *KafkaSecurityPublisher) error {
	q := `
		SELECT id, event_type, payload, created_at
		FROM outbox
		WHERE published = FALSE
		  AND event_type LIKE 'security.%'
		ORDER BY created_at ASC
		LIMIT 100
		FOR UPDATE SKIP LOCKED
	`

	rows, err := db.QueryxContext(ctx, q)
	if err != nil {
		return fmt.Errorf("query security outbox: %w", err)
	}
	defer rows.Close()

	successIDs := []uuid.UUID{}

	for rows.Next() {
		var evt struct {
			ID        uuid.UUID       `db:"id"`
			EventType string          `db:"event_type"`
			Payload   json.RawMessage `db:"payload"`
			CreatedAt time.Time       `db:"created_at"`
		}

		if err := rows.StructScan(&evt); err != nil {
			continue
		}

		// Route to appropriate handler
		var publishErr error
		switch {
		case evt.EventType == "security.snapshot":
			var snapshot SecuritySnapshotEvent
			if err := json.Unmarshal(evt.Payload, &snapshot); err != nil {
				continue
			}
			publishErr = publisher.PublishSnapshotToKafka(ctx, snapshot)

		case evt.EventType[:15] == "security.audit.":
			var audit SecurityAuditEvent
			if err := json.Unmarshal(evt.Payload, &audit); err != nil {
				continue
			}
			publishErr = publisher.PublishAuditToKafka(ctx, audit)
		}

		if publishErr == nil {
			successIDs = append(successIDs, evt.ID)
		}
	}

	// Mark published
	if len(successIDs) > 0 {
		query, args, err := sqlx.In(`
			UPDATE outbox
			SET published = TRUE, published_at = NOW()
			WHERE id IN (?)
		`, successIDs)
		if err != nil {
			return fmt.Errorf("build update query: %w", err)
		}

		query = db.Rebind(query)
		if _, err := db.ExecContext(ctx, query, args...); err != nil {
			return fmt.Errorf("mark outbox published: %w", err)
		}
	}

	return nil
}
