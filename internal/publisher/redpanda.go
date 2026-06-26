package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/twmb/franz-go/pkg/kgo"
	"github.com/twmb/franz-go/pkg/kmsg"
	"google.golang.org/protobuf/proto"

	eventspb "calendar-service/pkg/proto/calendar/events/v1"
)

type EventType string

const (
	EventTypeCalendarUpdate    EventType = "CALENDAR_UPDATE"
	EventTypeHolidayAdded      EventType = "HOLIDAY_ADDED"
	EventTypeHolidayRemoved    EventType = "HOLIDAY_REMOVED"
	EventTypeConflictDetected  EventType = "CONFLICT_DETECTED"
	EventTypeIngestionStarted  EventType = "INGESTION_STARTED"
	EventTypeIngestionComplete EventType = "INGESTION_COMPLETED"
)

type PublisherConfig struct {
	Brokers               []string
	Enabled               bool
	CompressProto         bool
	BatchSizeBytes        int
	LingerTimeMS          int
	MaxConcurrentRequests int
}

type CalendarEventPublisher struct {
	client *kgo.Client
	logger *logrus.Entry
	config PublisherConfig
}

func NewCalendarEventPublisher(cfg PublisherConfig, logger *logrus.Entry) (*CalendarEventPublisher, error) {
	if !cfg.Enabled {
		logger.Info("Event publisher disabled in config")
		return &CalendarEventPublisher{
			logger: logger,
			config: cfg,
		}, nil
	}

	if len(cfg.Brokers) == 0 {
		cfg.Brokers = []string{"redpanda:9092"}
	}

	opts := []kgo.Opt{
		kgo.SeedBrokers(cfg.Brokers...),
		kgo.ProducerBatchCompression(kgo.SnappyCompression()),
		kgo.RecordPartitioner(kgo.StickyKeyPartitioner(nil)),
		kgo.ProducerBatchMaxBytes(int32(cfg.BatchSizeBytes)),
		kgo.ProducerLinger(time.Duration(cfg.LingerTimeMS) * time.Millisecond),
		kgo.WithLogger(kgo.BasicLogger(nil, kgo.LogLevelError, nil)),
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = client.Ping(ctx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to connect to Redpanda at %v: %w", cfg.Brokers, err)
	}

	logger.WithField("brokers", cfg.Brokers).Info("Connected to Redpanda broker")
	ensureTopics(ctx, client, logger)

	return &CalendarEventPublisher{
		client: client,
		logger: logger,
		config: cfg,
	}, nil
}

func ensureTopics(ctx context.Context, client *kgo.Client, logger *logrus.Entry) {
	topics := []struct {
		name       string
		partitions int32
		replicas   int16
	}{
		{"calendar-updates", 3, 1},
		{"calendar-conflicts", 3, 1},
		{"ingestion-lifecycle", 1, 1},
	}

	for _, t := range topics {
		req := kmsg.NewCreateTopicsRequest()
		topic := kmsg.NewCreateTopicsRequestTopic()
		topic.Topic = t.name
		topic.NumPartitions = t.partitions
		topic.ReplicationFactor = t.replicas
		req.Topics = append(req.Topics, topic)

		resp, err := req.RequestWith(context.Background(), client)
		if err != nil {
			logger.WithField("topic", t.name).WithError(err).Debug("Topic creation failed")
			continue
		}
		if len(resp.Topics) > 0 && resp.Topics[0].ErrorCode != 0 {
			logger.WithField("topic", t.name).WithField("code", resp.Topics[0].ErrorCode).Debug("Topic already exists or failed")
		} else {
			logger.WithField("topic", t.name).Info("Topic created successfully")
		}
	}
}

func (p *CalendarEventPublisher) PublishCalendarUpdate(ctx context.Context, tenantID uuid.UUID, region, exchange, date string, isBusinessDay bool, holidayName, sourceSystem string, confidenceScore int, ruleApplied string) error {
	if p.client == nil {
		return nil
	}

	event := &eventspb.CalendarEvent{
		EventId:               uuid.New().String(),
		EventType:             string(EventTypeCalendarUpdate),
		TenantId:              tenantID.String(),
		Region:                region,
		Exchange:              exchange,
		CalendarDate:          date,
		IsBusinessDay:         isBusinessDay,
		HolidayName:           holidayName,
		SourceSystem:          sourceSystem,
		ConfidenceScore:       int32(confidenceScore),
		RuleApplied:           ruleApplied,
		SemanticTermVersion:   "1.0",
		BusinessObjectVersion: "1.0",
		Timestamp:             time.Now().UnixMilli(),
	}

	return p.publishEvent(ctx, "calendar-updates", event, tenantID.String())
}

func (p *CalendarEventPublisher) PublishConflictDetected(ctx context.Context, tenantID uuid.UUID, region, date string, fieldName string, conflictingValues []string, sourceSystems []string, severity int, reason string) error {
	if p.client == nil {
		return nil
	}

	conflict := &eventspb.CalendarConflict{
		ConflictId:        uuid.New().String(),
		TenantId:          tenantID.String(),
		Region:            region,
		CalendarDate:      date,
		FieldName:         fieldName,
		ConflictingValues: conflictingValues,
		SourceSystems:     sourceSystems,
		Severity:          int32(severity),
		Reason:            reason,
		Resolved:          false,
		CreatedAt:         time.Now().UnixMilli(),
	}

	data, err := proto.Marshal(conflict)
	if err != nil {
		p.logger.WithError(err).Error("Failed to marshal conflict event")
		return fmt.Errorf("marshal error: %w", err)
	}

	record := &kgo.Record{
		Topic: "calendar-conflicts",
		Key:   []byte(tenantID.String()),
		Value: data,
	}

	results := p.client.ProduceSync(ctx, record)
	if err := results.FirstErr(); err != nil {
		p.logger.WithError(err).Warn("Failed to publish conflict event")
		return fmt.Errorf("publish error: %w", err)
	}

	return nil
}

func (p *CalendarEventPublisher) PublishIngestionStarted(ctx context.Context, ingestionID uuid.UUID, tenantID uuid.UUID, regions []string, year int, forceRefresh bool, triggeredBy string) error {
	if p.client == nil {
		return nil
	}

	event := &eventspb.IngestionEvent{
		IngestionId:      ingestionID.String(),
		TenantId:         tenantID.String(),
		EventType:        "STARTED",
		Status:           "RUNNING",
		Regions:          regions,
		TargetYear:       int32(year),
		ForceRefresh:     forceRefresh,
		SourcesQueried:   0,
		StartedAt:        time.Now().UnixMilli(),
		TriggeredBy:      triggeredBy,
		WasmRulesVersion: "1.0",
	}

	data, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	record := &kgo.Record{
		Topic: "ingestion-lifecycle",
		Key:   []byte(tenantID.String()),
		Value: data,
	}

	results := p.client.ProduceSync(ctx, record)
	return results.FirstErr()
}

func (p *CalendarEventPublisher) PublishIngestionCompleted(ctx context.Context, ingestionID uuid.UUID, tenantID uuid.UUID, recordsIngested, recordsCreated, recordsUpdated, recordsDeleted int, conflictsDetected, conflictsResolved, conflictsEscalated int, sourcesQueried, sourcesSucceeded, sourcesFailed int, success bool, errorMessages []string, duration time.Duration) error {
	if p.client == nil {
		return nil
	}

	status := "SUCCESS"
	if !success {
		status = "PARTIAL_SUCCESS"
		if sourcesFailed == sourcesQueried {
			status = "FAILURE"
		}
	}

	event := &eventspb.IngestionEvent{
		IngestionId:        ingestionID.String(),
		TenantId:           tenantID.String(),
		EventType:          "COMPLETED",
		Status:             status,
		RecordsIngested:    int32(recordsIngested),
		RecordsCreated:     int32(recordsCreated),
		RecordsUpdated:     int32(recordsUpdated),
		RecordsDeleted:     int32(recordsDeleted),
		ConflictsDetected:  int32(conflictsDetected),
		ConflictsResolved:  int32(conflictsResolved),
		ConflictsEscalated: int32(conflictsEscalated),
		SourcesQueried:     int32(sourcesQueried),
		SourcesSucceeded:   int32(sourcesSucceeded),
		SourcesFailed:      int32(sourcesFailed),
		ErrorMessages:      errorMessages,
		CompletedAt:        time.Now().UnixMilli(),
		DurationMs:         duration.Milliseconds(),
		WasmRulesVersion:   "1.0",
	}

	data, err := proto.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	record := &kgo.Record{
		Topic: "ingestion-lifecycle",
		Key:   []byte(tenantID.String()),
		Value: data,
	}

	results := p.client.ProduceSync(ctx, record)
	return results.FirstErr()
}

func (p *CalendarEventPublisher) publishEvent(ctx context.Context, topic string, event *eventspb.CalendarEvent, partitionKey string) error {
	data, err := proto.Marshal(event)
	if err != nil {
		p.logger.WithError(err).Error("Failed to marshal event")
		return fmt.Errorf("marshal error: %w", err)
	}

	record := &kgo.Record{
		Topic:     topic,
		Key:       []byte(partitionKey),
		Value:     data,
		Timestamp: time.Now(),
	}

	results := p.client.ProduceSync(ctx, record)
	if err := results.FirstErr(); err != nil {
		p.logger.WithFields(logrus.Fields{
			"topic": topic,
			"event": event.EventId[:8],
			"error": err,
		}).Error("Failed to publish event")
		return fmt.Errorf("publish error: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"event_id": event.EventId[:8],
		"topic":    topic,
		"tenant":   event.TenantId[:8],
		"type":     event.EventType,
	}).Debug("Event published successfully")

	return nil
}

func (p *CalendarEventPublisher) PublishJSON(ctx context.Context, topic string, event interface{}, partitionKey string) error {
	if p.client == nil {
		return nil
	}

	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("json marshal error: %w", err)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(partitionKey),
		Value: data,
	}

	results := p.client.ProduceSync(ctx, record)
	return results.FirstErr()
}

func (p *CalendarEventPublisher) Flush(ctx context.Context) error {
	if p.client == nil {
		return nil
	}
	return p.client.Flush(ctx)
}

func (p *CalendarEventPublisher) Close() error {
	if p.client == nil {
		return nil
	}
	p.client.Close()
	p.logger.Info("Event publisher closed")
	return nil
}

func (p *CalendarEventPublisher) Health(ctx context.Context) error {
	if p.client == nil {
		return fmt.Errorf("publisher not initialized")
	}
	return p.client.Ping(ctx)
}

func (p *CalendarEventPublisher) IsEnabled() bool {
	return p.client != nil && p.config.Enabled
}
