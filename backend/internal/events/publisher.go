package events

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	kafka "github.com/segmentio/kafka-go"
)

type AuditEventPublisher struct {
	writer *kafka.Writer
}

func NewAuditEventPublisher(brokers string) *AuditEventPublisher {
	w := &kafka.Writer{
		Addr:     kafka.TCP(strings.Split(brokers, ",")...),
		Balancer: &kafka.LeastBytes{},
	}
	return &AuditEventPublisher{writer: w}
}

type AuditEvent struct {
	ID         string                 `json:"id"`
	InstanceID string                 `json:"instance_id"`
	TenantID   string                 `json:"tenant_id"`
	BPKey      string                 `json:"bp_key"`
	EventType  string                 `json:"event_type"`
	StepKey    string                 `json:"step_key"`
	ActorID    string                 `json:"actor_id"`
	ActorRole  string                 `json:"actor_role"`
	OldValue   map[string]interface{} `json:"old_value"`
	NewValue   map[string]interface{} `json:"new_value"`
	Reason     string                 `json:"reason"`
	IPAddress  string                 `json:"ip_address"`
	UserAgent  string                 `json:"user_agent"`
	CreatedAt  string                 `json:"created_at"`
}

func (p *AuditEventPublisher) PublishAuditEvent(ctx context.Context, event AuditEvent) error {
	if p.writer == nil {
		return nil
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal audit event: %w", err)
	}

	msg := kafka.Message{
		Topic: "audit.events",
		Key:   []byte(event.ID),
		Value: body,
		Time:  time.Now(),
	}

	return p.writer.WriteMessages(ctx, msg)
}

func (p *AuditEventPublisher) PublishSemanticTermComplianceUpdatedEvent(ctx context.Context, event SemanticTermComplianceUpdatedEvent) error {
	if p.writer == nil {
		return nil
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal compliance event: %w", err)
	}

	msg := kafka.Message{
		Topic: "compliance.events",
		Key:   []byte(event.EventID),
		Value: body,
		Time:  time.Now(),
	}

	return p.writer.WriteMessages(ctx, msg)
}

func (p *AuditEventPublisher) PublishBusinessTermComplianceUpdatedEvent(ctx context.Context, event *BusinessTermComplianceUpdatedEvent) error {
	if p.writer == nil {
		return nil
	}

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal compliance event: %w", err)
	}

	msg := kafka.Message{
		Topic: "compliance.events",
		Key:   []byte(event.EventID),
		Value: body,
		Time:  time.Now(),
	}

	return p.writer.WriteMessages(ctx, msg)
}

func (p *AuditEventPublisher) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
