package mdm

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// GoldenPublishRecord represents a materialized golden record to publish
type GoldenPublishRecord struct {
	EntityKey  string                 `json:"entity_key"`
	Payload    map[string]interface{} `json:"payload"`
	Provenance map[string]interface{} `json:"provenance,omitempty"`
}

// PublishGoldenRecordsRequest is the activity input
type PublishGoldenRecordsRequest struct {
	TenantID   string                `json:"tenant_id"`
	EntityType string                `json:"entity_type"`
	BatchID    uuid.UUID             `json:"batch_id"`
	Topic      string                `json:"topic"`
	Records    []GoldenPublishRecord `json:"records"`
}

// GoldenRecordEvent represents a formatted event payload sent to the publish topic
type GoldenRecordEvent struct {
	EventID    string                 `json:"event_id"` // gold.<tenant>.<type>.<key>.<hash>
	TenantID   string                 `json:"tenant_id"`
	EntityType string                 `json:"entity_type"`
	EntityKey  string                 `json:"entity_key"`
	BatchID    string                 `json:"batch_id"`
	Payload    map[string]interface{} `json:"payload"`
	Provenance map[string]interface{} `json:"provenance,omitempty"`
	EmittedAt  time.Time              `json:"emitted_at"`
}

// PublishGoldenRecordsResult is the activity output summary
type PublishGoldenRecordsResult struct {
	TotalPublished int      `json:"total_published"`
	EventIDs       []string `json:"event_ids"`
	Topic          string   `json:"topic"`
	EmittedAt      time.Time `json:"emitted_at"`
}

// GoldenEventPublisher interface abstracts event emission (e.g. Redpanda/Kafka producer)
type GoldenEventPublisher interface {
	PublishEvent(ctx context.Context, topic string, key string, event GoldenRecordEvent) error
}

// InMemoryEventPublisher is an in-memory test publisher
type InMemoryEventPublisher struct {
	Events []GoldenRecordEvent
}

func (p *InMemoryEventPublisher) PublishEvent(ctx context.Context, topic string, key string, event GoldenRecordEvent) error {
	p.Events = append(p.Events, event)
	return nil
}

// GoldenPublishingActivity provides Temporal activity methods for emitting golden events
type GoldenPublishingActivity struct {
	publisher GoldenEventPublisher
}

// NewGoldenPublishingActivity creates a new GoldenPublishingActivity
func NewGoldenPublishingActivity(publisher GoldenEventPublisher) *GoldenPublishingActivity {
	if publisher == nil {
		publisher = &InMemoryEventPublisher{}
	}
	return &GoldenPublishingActivity{publisher: publisher}
}

// PublishGoldenRecordsActivity formats and emits deduplicated golden record events for the batch
func (a *GoldenPublishingActivity) PublishGoldenRecordsActivity(
	ctx context.Context,
	req PublishGoldenRecordsRequest,
) (*PublishGoldenRecordsResult, error) {
	if req.TenantID == "" || req.EntityType == "" || req.BatchID == uuid.Nil {
		return nil, fmt.Errorf("tenant_id, entity_type, and batch_id are required")
	}

	topic := req.Topic
	if topic == "" {
		topic = fmt.Sprintf("gold.%s.%s", req.TenantID, req.EntityType)
	}

	result := &PublishGoldenRecordsResult{
		Topic:     topic,
		EmittedAt: time.Now().UTC(),
		EventIDs:  make([]string, 0, len(req.Records)),
	}

	for _, rec := range req.Records {
		hash, err := ComputePayloadHash(rec.Payload)
		if err != nil {
			return nil, fmt.Errorf("compute payload hash for key %s: %w", rec.EntityKey, err)
		}

		// Deduplicated event ID: gold.<tenant>.<type>.<key>.<hash>
		eventID := fmt.Sprintf("gold.%s.%s.%s.%s", req.TenantID, req.EntityType, rec.EntityKey, hash)

		evt := GoldenRecordEvent{
			EventID:    eventID,
			TenantID:   req.TenantID,
			EntityType: req.EntityType,
			EntityKey:  rec.EntityKey,
			BatchID:    req.BatchID.String(),
			Payload:    rec.Payload,
			Provenance: rec.Provenance,
			EmittedAt:  result.EmittedAt,
		}

		if err := a.publisher.PublishEvent(ctx, topic, rec.EntityKey, evt); err != nil {
			return nil, fmt.Errorf("publish golden event %s: %w", eventID, err)
		}

		result.EventIDs = append(result.EventIDs, eventID)
		result.TotalPublished++
	}

	return result, nil
}
