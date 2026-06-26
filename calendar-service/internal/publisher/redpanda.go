package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/twmb/franz-go/pkg/kgo"
)

// ============================================================================
// Event Models (Aligning with Usice Architecture Section 5: Semantic Execution)
// ============================================================================

type EventType string

const (
	EventTypeCalendarUpdate     EventType = "CALENDAR_UPDATE"
	EventTypeConflictDetected   EventType = "CONFLICT_DETECTED"
	EventTypeSourceActivated    EventType = "SOURCE_ACTIVATED"
	EventTypeIngestionStarted   EventType = "INGESTION_STARTED"
	EventTypeIngestionCompleted EventType = "INGESTION_COMPLETED"
)

// CalendarEvent represents a semantic calendar event for publication to Redpanda
type CalendarEvent struct {
	EventID               string    `json:"event_id"`
	EventType             EventType `json:"event_type"`
	TenantID              string    `json:"tenant_id"`
	Region                string    `json:"region"`
	Exchange              *string   `json:"exchange,omitempty"`
	CalendarDate          string    `json:"calendar_date"`
	IsBusinessDay         bool      `json:"is_business_day"`
	HolidayName           *string   `json:"holiday_name,omitempty"`
	SourceSystem          string    `json:"source_system"`
	ConfidenceScore       int       `json:"confidence_score"`
	Operation             string    `json:"operation"` // CREATE, UPDATE, DELETE
	SemanticTermVersion   string    `json:"semantic_term_version"`
	BusinessObjectVersion string    `json:"business_object_version"`
	Timestamp             int64     `json:"timestamp"`
	ConflictingSources    []string  `json:"conflicting_sources,omitempty"`
	Lineage               *Lineage  `json:"lineage,omitempty"`
}

type Lineage struct {
	RuleApplied      string                 `json:"rule_applied"`
	WasmExecutionID  string                 `json:"wasm_execution_id"`
	WinningSource    string                 `json:"winning_source"`
	CompetingSources []string               `json:"competing_sources"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}

// ============================================================================
// Redpanda Publisher (Usice Architecture Section 2.3: Semantic Engine)
// ============================================================================

type RedpandaPublisher struct {
	client *kgo.Client
	logger *logrus.Entry
	topic  string
}

func NewRedpandaPublisher(brokers []string, topic string, logger *logrus.Entry) (*RedpandaPublisher, error) {
	if logger == nil {
		logger = logrus.NewEntry(logrus.New())
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ProducerBatchCompression(kgo.SnappyCompression()),
		kgo.RecordPartitioner(kgo.StickyKeyPartitioner(nil)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create redpanda client: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"brokers": brokers,
		"topic":   topic,
	}).Info("Redpanda publisher initialized")

	return &RedpandaPublisher{
		client: client,
		logger: logger,
		topic:  topic,
	}, nil
}

// PublishCalendarUpdate publishes a calendar update event
func (p *RedpandaPublisher) PublishCalendarUpdate(
	ctx context.Context,
	tenantID uuid.UUID,
	region string,
	exchange *string,
	date string,
	isBusinessDay bool,
	holidayName *string,
	sourceSystem string,
	confidence int,
	conflictingSources []string,
) error {
	event := CalendarEvent{
		EventID:               uuid.New().String(),
		EventType:             EventTypeCalendarUpdate,
		TenantID:              tenantID.String(),
		Region:                region,
		Exchange:              exchange,
		CalendarDate:          date,
		IsBusinessDay:         isBusinessDay,
		HolidayName:           holidayName,
		SourceSystem:          sourceSystem,
		ConfidenceScore:       confidence,
		Operation:             "UPDATE",
		SemanticTermVersion:   "1.0",
		BusinessObjectVersion: "1.0",
		Timestamp:             time.Now().UnixMilli(),
		ConflictingSources:    conflictingSources,
	}

	return p.PublishEvent(ctx, event)
}

// PublishConflict publishes a conflict detection event
func (p *RedpandaPublisher) PublishConflict(
	ctx context.Context,
	tenantID uuid.UUID,
	region string,
	date string,
	conflictingSources []string,
	description string,
) error {
	event := CalendarEvent{
		EventID:               uuid.New().String(),
		EventType:             EventTypeConflictDetected,
		TenantID:              tenantID.String(),
		Region:                region,
		CalendarDate:          date,
		SourceSystem:          "USICE_MDM",
		ConfidenceScore:       0,
		Timestamp:             time.Now().UnixMilli(),
		ConflictingSources:    conflictingSources,
		SemanticTermVersion:   "1.0",
		BusinessObjectVersion: "1.0",
	}

	return p.PublishEvent(ctx, event)
}

// PublishSourceActivation publishes when a source is activated/deactivated
func (p *RedpandaPublisher) PublishSourceActivation(
	ctx context.Context,
	sourceName string,
	isActive bool,
) error {
	operation := "ACTIVATE"
	if !isActive {
		operation = "DEACTIVATE"
	}

	event := CalendarEvent{
		EventID:               uuid.New().String(),
		EventType:             EventTypeSourceActivated,
		TenantID:              "SYSTEM",
		SourceSystem:          sourceName,
		Operation:             operation,
		Timestamp:             time.Now().UnixMilli(),
		SemanticTermVersion:   "1.0",
		BusinessObjectVersion: "1.0",
	}

	return p.PublishEvent(ctx, event)
}

// PublishIngestionStarted publishes when an ingestion job starts
func (p *RedpandaPublisher) PublishIngestionStarted(
	ctx context.Context,
	tenantID uuid.UUID,
	regions []string,
	sources []string,
) error {
	event := CalendarEvent{
		EventID:               uuid.New().String(),
		EventType:             EventTypeIngestionStarted,
		TenantID:              tenantID.String(),
		Region:                fmt.Sprintf("MULTI[%v]", regions),
		SourceSystem:          fmt.Sprintf("MULTI[%v]", sources),
		Operation:             "START",
		Timestamp:             time.Now().UnixMilli(),
		SemanticTermVersion:   "1.0",
		BusinessObjectVersion: "1.0",
	}

	return p.PublishEvent(ctx, event)
}

// PublishIngestionCompleted publishes when an ingestion job completes
func (p *RedpandaPublisher) PublishIngestionCompleted(
	ctx context.Context,
	tenantID uuid.UUID,
	recordsProcessed int,
	durationMs int,
	success bool,
) error {
	operation := "SUCCESS"
	if !success {
		operation = "FAILED"
	}

	event := CalendarEvent{
		EventID:               uuid.New().String(),
		EventType:             EventTypeIngestionCompleted,
		TenantID:              tenantID.String(),
		SourceSystem:          "USICE_MDM",
		ConfidenceScore:       recordsProcessed,
		Operation:             operation,
		Timestamp:             time.Now().UnixMilli(),
		SemanticTermVersion:   "1.0",
		BusinessObjectVersion: "1.0",
	}

	return p.PublishEvent(ctx, event)
}

// PublishEvent publishes a calendar event to Redpanda
func (p *RedpandaPublisher) PublishEvent(ctx context.Context, event CalendarEvent) error {
	// Serialize event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		p.logger.WithError(err).Error("Failed to marshal event")
		return err
	}

	// Publish to Redpanda
	// Partition key is tenant_id to maintain order per tenant (Usice Architecture Section 1: Deterministic Tracing)
	record := &kgo.Record{
		Topic: p.topic,
		Key:   []byte(event.TenantID),
		Value: data,
		Headers: []kgo.RecordHeader{
			{
				Key:   "event_type",
				Value: []byte(string(event.EventType)),
			},
			{
				Key:   "eventId",
				Value: []byte(event.EventID),
			},
		},
	}

	err = p.client.ProduceSync(ctx, record).FirstErr()
	if err != nil {
		p.logger.WithError(err).WithFields(logrus.Fields{
			"event_id":   event.EventID,
			"event_type": event.EventType,
			"tenant":     event.TenantID,
		}).Error("Failed to publish event to Redpanda")
		return err
	}

	p.logger.WithFields(logrus.Fields{
		"event_id":   event.EventID,
		"event_type": event.EventType,
		"topic":      p.topic,
	}).Debug("Event published to Redpanda")

	return nil
}

// Close gracefully shuts down the Redpanda connection
func (p *RedpandaPublisher) Close() error {
	p.client.Close()
	return nil
}
