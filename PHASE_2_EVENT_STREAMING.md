# Phase 2: Real-Time Event Streaming with Redpanda

**Timeline:** 4-6 hours | **Start After:** Phase 1 validation passes

## Objective
Connect Redpanda message broker so downstream systems (trading platforms, analytics, dashboards) receive real-time calendar updates as they happen.

---

## What You'll Build

```
Semantic Engine (Phase 1)
    ↓ (On each calendar update)
Redpanda Event Publisher (NEW - This Phase)
    ↓ (publishes to topics)
Redpanda Message Broker
    ├─→ Trading Platform Consumer (example)
    ├─→ Analytics System (example)
    └─→ React Frontend (real-time subscriptions)
```

---

## Prerequisites

✅ Phase 1 completed (docker-compose running with 9 services)  
✅ Redpanda broker available at `redpanda:9092`  
✅ Go 1.21+ installed locally  
✅ Docker images built

---

## Step 1: Event Schema Definition (30 min)

### 1.1 Create Protobuf Schema

```bash
mkdir -p proto/calendar/events
```

Create `proto/calendar/events/v1/calendar_events.proto`:

```protobuf
syntax = "proto3";

package calendar.events.v1;

option go_package = "github.com/usice/calendar-service/pkg/proto/calendar/events/v1";

// CalendarEvent represents a change to calendar data
message CalendarEvent {
  string event_id = 1;                    // UUID unique to this event
  string event_type = 2;                  // CALENDAR_UPDATE, HOLIDAY_ADDED, CONFLICT_DETECTED
  string tenant_id = 3;                   // Multi-tenant isolation
  
  string region = 4;                      // Region affected (US, GB, etc)
  string exchange = 5;                    // Exchange code (optional)
  string calendar_date = 6;               // ISO 8601 date string
  
  bool is_business_day = 7;               // True if market/office open
  string holiday_name = 8;                // Human-readable holiday name
  
  string source_system = 9;               // Where data came from (Workalendar, TradingHours, etc)
  int32 confidence_score = 10;            // 0-100 confidence in this data
  
  string operation = 11;                  // CREATE, UPDATE, DELETE
  string rule_applied = 12;               // Which survivorship rule won
  
  string semantic_term_version = 13;      // Schema version for semantic terms
  string business_object_version = 14;    // Schema version for business objects
  
  int64 timestamp = 15;                   // Unix mills when event created
  string wasm_execution_id = 16;          // Traceable to rules engine execution
}

// CalendarConflict represents a discrepancy between sources
message CalendarConflict {
  string conflict_id = 1;
  string tenant_id = 2;
  string region = 3;
  string calendar_date = 4;
  
  string field_name = 5;                  // Which field had conflict
  repeated string conflicting_values = 6; // Values that disagreed
  repeated string source_systems = 7;     // Sources that disagreed
  
  int32 severity = 8;                     // 1=LOW, 2=MEDIUM, 3=HIGH, 4=CRITICAL
  string reason = 9;                      // Why this conflict occurred
  bool resolved = 10;                     // Was it resolved?
  
  int64 created_at = 11;
  int64 resolved_at = 12;
}
```

### 1.2 Compile Protobuf

```bash
# Install protoc if not present
brew install protobuf

# Generate Go code
protoc --go_out=. --go_opt=paths=source_relative proto/calendar/events/v1/*.proto

# Verify generated file exists
ls -la pkg/proto/calendar/events/v1/
```

---

## Step 2: Event Publisher Implementation (2 hours)

Create `internal/publisher/redpanda.go`:

```go
package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/protobuf/proto"

	eventspb "github.com/usice/calendar-service/pkg/proto/calendar/events/v1"
)

type CalendarEventPublisher struct {
	client *kgo.Client
	logger *logrus.Entry
}

// EventType defines the type of calendar event
type EventType string

const (
	EventTypeCalendarUpdate    EventType = "CALENDAR_UPDATE"
	EventTypeHolidayAdded      EventType = "HOLIDAY_ADDED"
	EventTypeHolidayRemoved    EventType = "HOLIDAY_REMOVED"
	EventTypeConflictDetected  EventType = "CONFLICT_DETECTED"
	EventTypeIngestionStarted  EventType = "INGESTION_STARTED"
	EventTypeIngestionComplete EventType = "INGESTION_COMPLETED"
)

// NewCalendarEventPublisher creates a new publisher
func NewCalendarEventPublisher(brokers []string, logger *logrus.Entry) (*CalendarEventPublisher, error) {
	opts := []kgo.Opt{
		kgo.SeedBrokers(brokers...),
		kgo.ProducerBatchCompression(kgo.Snappy()),
		kgo.RecordPartitioner(kgo.StickyKeyPartitioner()),
		kgo.DisableIdempotentWrites(false), // Enable idempotent writes for exactly-once semantics
		kgo.ProducerLinger(100 * time.Millisecond), // Batch small messages
	}

	client, err := kgo.NewClient(opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kafka client: %w", err)
	}

	// Test connectivity
	err = client.Ping(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redpanda: %w", err)
	}

	logger.Info("Connected to Redpanda broker")

	return &CalendarEventPublisher{
		client: client,
		logger: logger,
	}, nil
}

// PublishCalendarUpdate publishes a calendar day update event
func (p *CalendarEventPublisher) PublishCalendarUpdate(
	ctx context.Context,
	tenantID uuid.UUID,
	region, exchange, date string,
	isBusinessDay bool,
	holidayName, sourceSystem string,
	confidenceScore int,
	ruleApplied string,
) error {
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
		Operation:             "UPDATE",
		RuleApplied:           ruleApplied,
		SemanticTermVersion:   "1.0",
		BusinessObjectVersion: "1.0",
		Timestamp:             time.Now().UnixMilli(),
	}

	return p.publishEvent(ctx, "calendar-updates", event, tenantID.String())
}

// PublishConflictDetected publishes a conflict detection event
func (p *CalendarEventPublisher) PublishConflictDetected(
	ctx context.Context,
	tenantID uuid.UUID,
	region, date string,
	conflictingValues []string,
	sourceSystems []string,
) error {
	event := &eventspb.CalendarEvent{
		EventId:     uuid.New().String(),
		EventType:   string(EventTypeConflictDetected),
		TenantId:    tenantID.String(),
		Region:      region,
		CalendarDate: date,
		SourceSystem: "conflict-detection",
		Timestamp:   time.Now().UnixMilli(),
	}

	return p.publishEvent(ctx, "calendar-conflicts", event, tenantID.String())
}

// PublishIngestionStarted publishes when ingestion begins
func (p *CalendarEventPublisher) PublishIngestionStarted(
	ctx context.Context,
	tenantID uuid.UUID,
	regions []string,
	year int,
) error {
	event := &eventspb.CalendarEvent{
		EventId:   uuid.New().String(),
		EventType: string(EventTypeIngestionStarted),
		TenantId:  tenantID.String(),
		Region:    fmt.Sprintf("%d regions for %d", len(regions), year),
		Timestamp: time.Now().UnixMilli(),
	}

	return p.publishEvent(ctx, "ingestion-lifecycle", event, tenantID.String())
}

// PublishIngestionCompleted publishes when ingestion finishes
func (p *CalendarEventPublisher) PublishIngestionCompleted(
	ctx context.Context,
	tenantID uuid.UUID,
	recordsIngested, conflictsDetected int,
	duration time.Duration,
) error {
	event := &eventspb.CalendarEvent{
		EventId:         uuid.New().String(),
		EventType:       string(EventTypeIngestionComplete),
		TenantId:        tenantID.String(),
		ConfidenceScore: int32(recordsIngested), // Reuse field for count
		Timestamp:       time.Now().UnixMilli(),
	}

	return p.publishEvent(ctx, "ingestion-lifecycle", event, tenantID.String())
}

// publishEvent is the internal method that serializes and publishes events
func (p *CalendarEventPublisher) publishEvent(
	ctx context.Context,
	topic string,
	event *eventspb.CalendarEvent,
	partitionKey string,
) error {
	// Serialize to Protobuf
	data, err := proto.Marshal(event)
	if err != nil {
		p.logger.WithError(err).Error("Failed to marshal event")
		return fmt.Errorf("marshal error: %w", err)
	}

	// Create Kafka record
	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(partitionKey), // Partition by tenant for order guarantee
		Value: data,
		Timestamp: time.Now(),
	}

	// Publish synchronously
	results := p.client.ProduceSync(ctx, record)
	if err := results.FirstErr(); err != nil {
		p.logger.WithFields(logrus.Fields{
			"topic": topic,
			"error": err,
		}).Error("Failed to publish event")
		return fmt.Errorf("publish error: %w", err)
	}

	p.logger.WithFields(logrus.Fields{
		"event_id": event.EventId,
		"topic":    topic,
		"tenant":   event.TenantId[:8], // First 8 chars of UUID for logging
	}).Debug("Event published successfully")

	return nil
}

// PublishJSON publishes a JSON-serialized event (for debugging)
func (p *CalendarEventPublisher) PublishJSON(ctx context.Context, topic string, event interface{}, partitionKey string) error {
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

// Close gracefully shuts down the publisher
func (p *CalendarEventPublisher) Close() error {
	p.client.LeaveGroup()
	p.client.Flush(context.Background())
	return nil
}

// Health returns the current health status
func (p *CalendarEventPublisher) Health(ctx context.Context) error {
	return p.client.Ping(ctx)
}
```

---

## Step 3: Integrate Publisher into Orchestrator (30 min)

Update `internal/mdm/orchestrator.go`:

```go
// Add to struct
type IngestionOrchestrator struct {
	db               *sql.DB
	httpClient       *http.Client
	logger           *logrus.Entry
	sourceCache      map[string]SourceConfig
	eventPublisher   *publisher.CalendarEventPublisher  // NEW
}

// Add to constructor
func NewIngestionOrchestrator(
	db *sql.DB,
	eventPublisher *publisher.CalendarEventPublisher,  // NEW param
	logger *logrus.Entry,
) *IngestionOrchestrator {
	return &IngestionOrchestrator{
		db:             db,
		eventPublisher: eventPublisher,  // NEW
		logger:         logger,
		sourceCache:    make(map[string]SourceConfig),
	}
}

// Add to RunIngestionCycle - publish start event
func (o *IngestionOrchestrator) RunIngestionCycle(...) error {
	jobID := uuid.New()
	
	// NEW: Publish ingestion started
	o.eventPublisher.PublishIngestionStarted(ctx, tenantID, regions, year)
	
	// ... existing ingestion logic ...
	
	// NEW: Publish ingestion completed
	o.eventPublisher.PublishIngestionCompleted(
		ctx,
		tenantID,
		job.RecordsIngested,
		job.ConflictsDetected,
		time.Since(job.StartedAt),
	)
	
	return nil
}

// Add to upsertGoldenRecord - publish update events
func (o *IngestionOrchestrator) upsertGoldenRecord(
	ctx context.Context,
	tenantID uuid.UUID,
	result *SurvivorshipResult,
) error {
	// ... existing upsert logic ...
	
	// NEW: Publish event for each updated day
	for _, change := range result.Changes {
		err := o.eventPublisher.PublishCalendarUpdate(
			ctx,
			tenantID,
			change.RegionCode,
			change.ExchangeCode,
			change.Date.Format("2006-01-02"),
			change.IsBusinessDay,
			change.HolidayName,
			change.SourceSystem,
			change.ConfidenceScore,
			change.RuleApplied,
		)
		if err != nil {
			o.logger.WithError(err).Warn("Failed to publish calendar event")
			// Don't fail ingestion if event publishing fails
		}
	}
	
	return nil
}
```

---

## Step 4: Example Consumer (Trading Platform) (1 hour)

Create `services/trading-consumer/main.go`:

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/protobuf/proto"

	eventspb "github.com/usice/calendar-service/pkg/proto/calendar/events/v1"
)

// TradingCalendarCache simulates a trading platform's calendar cache
type TradingCalendarCache struct {
	data map[string]bool // date -> is_business_day
}

func NewTradingCalendarCache() *TradingCalendarCache {
	return &TradingCalendarCache{
		data: make(map[string]bool),
	}
}

func (c *TradingCalendarCache) UpdateDay(date string, isBusinessDay bool) {
	c.data[date] = isBusinessDay
	fmt.Printf("[CACHE] Updated %s: isBusinessDay=%v\n", date, isBusinessDay)
}

func (c *TradingCalendarCache) IsBusinessDay(date string) bool {
	if val, exists := c.data[date]; exists {
		return val
	}
	// Default: assume business day if not in cache
	return true
}

func main() {
	brokers := []string{os.Getenv("REDPANDA_BROKERS")}
	if len(brokers) == 0 || brokers[0] == "" {
		brokers = []string{"redpanda:9092"}
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumeTopics("calendar-updates"),
		kgo.GroupID("trading-platform-consumer"),
		kgo.SessionTimeout(30 * time.Second),
		kgo.RebalanceStrategies(kgo.RoundRobinAssignor()),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	cache := NewTradingCalendarCache()

	fmt.Println("🚀 Trading Platform Consumer Started")
	fmt.Println("Listening for calendar updates on topic: calendar-updates")
	fmt.Println("")

	ctx := context.Background()
	for {
		fetches := client.PollFetches(ctx)

		if errs := fetches.Errors(); len(errs) > 0 {
			fmt.Printf("Errors: %v\n", errs)
			continue
		}

		fetches.EachRecord(func(r *kgo.Record) {
			// Deserialize Protobuf
			event := &eventspb.CalendarEvent{}
			if err := proto.Unmarshal(r.Value, event); err != nil {
				fmt.Printf("⚠️  Failed to unmarshal event: %v\n", err)
				return
			}

			// Process event
			fmt.Printf("[EVENT] Type: %s | Date: %s | Region: %s | IsBusinessDay: %v | Confidence: %d%%\n",
				event.EventType,
				event.CalendarDate,
				event.Region,
				event.IsBusinessDay,
				event.ConfidenceScore,
			)

			// Update cache
			if event.EventType == "CALENDAR_UPDATE" {
				cache.UpdateDay(event.CalendarDate, event.IsBusinessDay)

				// Example: Reschedule trades if holiday changed
				if !event.IsBusinessDay {
					fmt.Printf("  ↳ Holiday detected: %s\n", event.HolidayName)
				}
			}

			// Commit offset after processing
			client.CommitUncommittedOffsets(ctx)
		})
	}
}
```

Create `services/trading-consumer/Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o consumer main.go

FROM alpine:3.18
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/consumer /usr/local/bin/

EXPOSE 8080
CMD ["consumer"]
```

---

## Step 5: React Subscription Hook (1 hour)

Create `frontend/src/hooks/useCalendarSubscription.ts`:

```typescript
import { useState, useEffect, useCallback } from 'react';
import { useApolloClient } from '@apollo/client';
import { gql } from '@apollo/client/core';

export const CALENDAR_UPDATES_SUBSCRIPTION = gql`
  subscription OnCalendarUpdate {
    calendarUpdate {
      eventId
      eventType
      calendarDate
      isBusinessDay
      holidayName
      region
      confidenceScore
      sourceSystem
      timestamp
    }
  }
`;

interface CalendarEvent {
  eventId: string;
  eventType: string;
  calendarDate: string;
  isBusinessDay: boolean;
  holidayName?: string;
  region: string;
  confidenceScore: number;
  sourceSystem: string;
  timestamp: number;
}

interface UseCalendarSubscriptionResult {
  events: CalendarEvent[];
  isConnected: boolean;
  error?: Error;
  lastUpdate?: CalendarEvent;
}

export function useCalendarSubscription(): UseCalendarSubscriptionResult {
  const client = useApolloClient();
  const [events, setEvents] = useState<CalendarEvent[]>([]);
  const [isConnected, setIsConnected] = useState(false);
  const [error, setError] = useState<Error>();
  const [lastUpdate, setLastUpdate] = useState<CalendarEvent>();

  useEffect(() => {
    // Subscribe to calendar updates
    const subscription = client
      .subscribe({
        query: CALENDAR_UPDATES_SUBSCRIPTION,
      })
      .subscribe({
        next: (data: any) => {
          const event: CalendarEvent = data.data.calendarUpdate;
          setEvents((prev) => [event, ...prev].slice(0, 100)); // Keep last 100
          setLastUpdate(event);
        },
        error: (err: Error) => {
          setError(err);
          setIsConnected(false);
        },
        complete: () => {
          setIsConnected(false);
        },
      });

    setIsConnected(true);

    return () => {
      subscription.unsubscribe();
    };
  }, [client]);

  const clearEvents = useCallback(() => {
    setEvents([]);
  }, []);

  return {
    events,
    isConnected,
    error,
    lastUpdate,
  };
}
```

Create `frontend/src/components/LiveCalendarUpdates.tsx`:

```typescript
import React from 'react';
import { useCalendarSubscription } from '../hooks/useCalendarSubscription';

export const LiveCalendarUpdates: React.FC = () => {
  const { events, isConnected, lastUpdate } = useCalendarSubscription();

  return (
    <div className="p-4 bg-white rounded-lg border">
      <div className="flex items-center justify-between mb-4">
        <h2 className="text-lg font-bold">📡 Real-Time Calendar Updates</h2>
        <div className="flex items-center gap-2">
          <div
            className={`w-3 h-3 rounded-full ${
              isConnected ? 'bg-green-500' : 'bg-gray-300'
            }`}
          />
          <span className="text-sm text-gray-600">
            {isConnected ? 'Connected' : 'Disconnected'}
          </span>
        </div>
      </div>

      {lastUpdate && (
        <div className="mb-4 p-3 bg-blue-50 rounded border-l-4 border-blue-500">
          <p className="text-sm font-medium">Latest Update</p>
          <p className="text-sm mt-1">
            <strong>{lastUpdate.calendarDate}</strong> ({lastUpdate.region}):{' '}
            {lastUpdate.isBusinessDay ? '💼 Business Day' : '🎉 Holiday'}
            {lastUpdate.holidayName && ` - ${lastUpdate.holidayName}`}
          </p>
          <p className="text-xs text-gray-500 mt-1">
            Confidence: {lastUpdate.confidenceScore}% | From: {lastUpdate.sourceSystem}
          </p>
        </div>
      )}

      <div className="space-y-2 max-h-96 overflow-y-auto">
        {events.length === 0 ? (
          <p className="text-sm text-gray-500 text-center py-8">
            No updates received yet. Waiting for calendar changes...
          </p>
        ) : (
          events.map((event) => (
            <div key={event.eventId} className="p-2 bg-gray-50 rounded text-xs">
              <div className="flex justify-between">
                <span className="font-medium">{event.calendarDate}</span>
                <span className="text-gray-500">
                  {event.isBusinessDay ? '💼' : '🎉'}
                </span>
              </div>
              <p className="text-gray-600">
                {event.region} • {event.sourceSystem} • {event.confidenceScore}%
              </p>
              {event.holidayName && (
                <p className="text-gray-700 font-medium">{event.holidayName}</p>
              )}
            </div>
          ))
        )}
      </div>

      <div className="text-xs text-gray-500 mt-4 pt-2 border-t">
        {events.length} recent updates
      </div>
    </div>
  );
};
```

---

## Step 6: Update Docker Compose (15 min)

Update `docker-compose.mdm.yml` to add trading consumer:

```yaml
services:
  # ... existing services ...

  trading-consumer:
    build:
      context: ./services/trading-consumer
      dockerfile: Dockerfile
    container_name: trading-consumer
    environment:
      - REDPANDA_BROKERS=redpanda:9092
    depends_on:
      - redpanda
    networks:
      - usice-network
    logging:
      driver: "json-file"
      options:
        max-size: "10m"
        max-file: "3"

  # Redpanda Console (UI for viewing topics/messages)
  redpanda-console:
    image: docker.redpanda.com/redpandadata/console:latest
    container_name: redpanda-console
    environment:
      - KAFKA_BROKERS=redpanda:9092
    ports:
      - "8888:8080"
    depends_on:
      - redpanda
    networks:
      - usice-network
```

---

## Step 7: Deployment (15 min)

```bash
# 1. Build new consumer image
docker build -t semlayer/trading-consumer:latest services/trading-consumer

# 2. Restart Docker Compose
docker-compose -f docker-compose.mdm.yml down
docker-compose -f docker-compose.mdm.yml up -d

# 3. Wait for startup
sleep 30

# 4. Verify all services running
docker-compose -f docker-compose.mdm.yml ps

# 5. View trading consumer logs
docker-compose -f docker-compose.mdm.yml logs -f trading-consumer

# 6. View Redpanda Console
open http://localhost:8888
# Navigate to Topics → calendar-updates → View messages
```

---

## Step 8: Test Event Publishing (20 min)

```bash
# 1. Trigger ingestion (will publish events)
curl -X POST http://localhost:8080/api/v1/mdm/calendar/ingest \
  -H "Content-Type: application/json" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000001" \
  -d '{
    "regions": ["US"],
    "year": 2026,
    "force_refresh": true
  }'

# 2. Watch trading consumer process events
docker-compose -f docker-compose.mdm.yml logs -f trading-consumer | head -50

# Expected output:
# [EVENT] Type: CALENDAR_UPDATE | Date: 2026-01-02 | Region: US | IsBusinessDay: true | Confidence: 85%
# [EVENT] Type: CALENDAR_UPDATE | Date: 2026-01-12 | Region: US | IsBusinessDay: false | Confidence: 90%
# ...

# 3. View events in Redpanda Console
open http://localhost:8888
# Click: Topics → calendar-updates
# Should show many records arriving in real-time
```

---

## Success Criteria ✅

Phase 2 is complete when:

```bash
# 1. Events published to Redpanda
docker-compose exec redpanda rpk topic list | grep calendar
# Should show: calendar-updates, calendar-conflicts, ingestion-lifecycle

# 2. Consumer receiving events
docker-compose logs trading-consumer | grep "EVENT" | wc -l
# Should show: hundreds of lines

# 3. Redpanda Console accessible
curl http://localhost:8888/health
# Should return: HTTP 200

# 4. Event schema valid
protoc --version
# Should show: version 3

# 5. React component builds without errors
cd frontend && npm run build
# Should complete successfully
```

---

## Troubleshooting Phase 2

| Issue | Solution |
|-------|----------|
| "Connection refused" on Redpanda | Verify redpanda service running: `docker-compose ps` |
| Trading consumer not receiving events | Check ingestion is triggered and logs show publishing |
| Protobuf compilation fails | Ensure protoc installed: `brew install protobuf` |
| React component errors | Verify Apollo Client v4+ and GraphQL subscriptions enabled |
| Events not visible in Redpanda Console | Check network connectivity and Redpanda logs |

---

## Next Steps

Once Phase 2 validates:

```bash
# 1. Verify all event types working
docker-compose logs semantic-engine | grep "event_published"

# 2. Add custom consumers (your own systems)
# Use trading-consumer as template

# 3. Proceed to Phase 3 (commercial sources)
cat COMPLETE_MDM_ROADMAP.md | grep "PHASE 3"
```

---

**Phase 2 Complete!** 🎉

Next: Phase 3 (Production Hardening - TradingHours, EODHD, failover, etc.)
