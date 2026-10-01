package bp

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TriggerSubscription represents a row in public.bp_trigger_subscriptions
type TriggerSubscription struct {
	TenantID    uuid.UUID       `db:"tenant_id" json:"tenant_id"`
	ProcessID   string          `db:"process_id" json:"process_id"`
	TriggerType string          `db:"trigger_type" json:"trigger_type"` // 'event'
	Topic       string          `db:"topic" json:"topic"`
	EventType   *string         `db:"event_type" json:"event_type,omitempty"`
	Ops         json.RawMessage `db:"ops" json:"ops,omitempty"`
	Version     int             `db:"version" json:"version"`
	UpdatedAt   time.Time       `db:"updated_at" json:"updated_at"`
}

// TriggerSubscriptionStore provides methods to manage and match trigger subscriptions
type TriggerSubscriptionStore struct {
	db *sql.DB
}

// NewTriggerSubscriptionStore creates a new TriggerSubscriptionStore
func NewTriggerSubscriptionStore(db *sql.DB) *TriggerSubscriptionStore {
	return &TriggerSubscriptionStore{db: db}
}

// UpsertSubscription inserts or updates a trigger subscription (called by compiler at compile/publish time)
func (s *TriggerSubscriptionStore) UpsertSubscription(ctx context.Context, sub TriggerSubscription) error {
	if s.db == nil {
		return fmt.Errorf("trigger subscription store: db is nil")
	}
	if sub.TenantID == uuid.Nil || sub.ProcessID == "" || sub.Topic == "" {
		return fmt.Errorf("tenant_id, process_id, and topic are required")
	}

	query := `
		INSERT INTO public.bp_trigger_subscriptions (
			tenant_id, process_id, trigger_type, topic, event_type, ops, version, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		ON CONFLICT (tenant_id, process_id, topic) DO UPDATE SET
			trigger_type = EXCLUDED.trigger_type,
			event_type = EXCLUDED.event_type,
			ops = EXCLUDED.ops,
			version = EXCLUDED.version,
			updated_at = NOW()
	`

	var opsBytes []byte
	if len(sub.Ops) > 0 {
		opsBytes = sub.Ops
	}

	_, err := s.db.ExecContext(ctx, query,
		sub.TenantID,
		sub.ProcessID,
		sub.TriggerType,
		sub.Topic,
		sub.EventType,
		opsBytes,
		sub.Version,
	)
	return err
}

// DeleteSubscription removes a trigger subscription (called on trigger type change or definition archive)
func (s *TriggerSubscriptionStore) DeleteSubscription(ctx context.Context, tenantID uuid.UUID, processID, topic string) error {
	if s.db == nil {
		return fmt.Errorf("trigger subscription store: db is nil")
	}
	query := `
		DELETE FROM public.bp_trigger_subscriptions
		WHERE tenant_id = $1 AND process_id = $2 AND topic = $3
	`
	_, err := s.db.ExecContext(ctx, query, tenantID, processID, topic)
	return err
}

// MatchSubscriptions finds all process subscriptions matching an incoming topic, event_type, and optional CDC op.
// Used by bpbridge to dispatch events across tenants without RLS bypass.
func (s *TriggerSubscriptionStore) MatchSubscriptions(ctx context.Context, topic string, eventType string, op string) ([]TriggerSubscription, error) {
	if s.db == nil {
		return nil, fmt.Errorf("trigger subscription store: db is nil")
	}

	query := `
		SELECT tenant_id, process_id, trigger_type, topic, event_type, ops, version, updated_at
		FROM public.bp_trigger_subscriptions
		WHERE topic = $1
		  AND (event_type IS NULL OR event_type = '' OR event_type = $2)
	`
	rows, err := s.db.QueryContext(ctx, query, topic, eventType)
	if err != nil {
		return nil, fmt.Errorf("query trigger subscriptions: %w", err)
	}
	defer rows.Close()

	var results []TriggerSubscription
	for rows.Next() {
		var sub TriggerSubscription
		var eventTypeNull sql.NullString
		var opsRaw []byte
		if err := rows.Scan(
			&sub.TenantID,
			&sub.ProcessID,
			&sub.TriggerType,
			&sub.Topic,
			&eventTypeNull,
			&opsRaw,
			&sub.Version,
			&sub.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan trigger subscription: %w", err)
		}
		if eventTypeNull.Valid {
			sub.EventType = &eventTypeNull.String
		}
		if len(opsRaw) > 0 {
			sub.Ops = opsRaw
			// If an op filter is defined (e.g. ["c", "u"]), check if op matches
			if op != "" {
				var allowedOps []string
				if err := json.Unmarshal(opsRaw, &allowedOps); err == nil && len(allowedOps) > 0 {
					matched := false
					for _, allowed := range allowedOps {
						if allowed == op {
							matched = true
							break
						}
					}
					if !matched {
						continue // skip non-matching op
					}
				}
			}
		}
		results = append(results, sub)
	}
	return results, nil
}
