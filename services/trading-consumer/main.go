package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// CalendarEvent is the wire-format shape consumed from the
// calendar-updates topic. JSON-serialized for portability — the original
// calendar-service proto package has been removed (see eventsv1 stub
// note in services/calendar-service/pkg/proto/calendar/events/v1/).
//
// Field naming mirrors the producer's `internal/publisher.redpanda.go`
// JSON encoding so unmarshaling stays byte-compatible.
type CalendarEvent struct {
	EventId               string `json:"event_id"`
	EventType             string `json:"event_type"`
	TenantId              string `json:"tenant_id"`
	Region                string `json:"region"`
	Exchange              string `json:"exchange"`
	CalendarDate          string `json:"calendar_date"`
	IsBusinessDay         bool   `json:"is_business_day"`
	HolidayName           string `json:"holiday_name"`
	SourceSystem          string `json:"source_system"`
	ConfidenceScore       int32  `json:"confidence_score"`
	RuleApplied           string `json:"rule_applied"`
	SemanticTermVersion   string `json:"semantic_term_version"`
	BusinessObjectVersion string `json:"business_object_version"`
	Timestamp             int64  `json:"timestamp_ms"`
}

// IngestionEvent is the wire-format shape consumed from the
// ingestion-lifecycle topic.
type IngestionEvent struct {
	IngestionId        string   `json:"ingestion_id"`
	TenantId           string   `json:"tenant_id"`
	EventType          string   `json:"event_type"`
	Status             string   `json:"status"`
	Regions            []string `json:"regions"`
	TargetYear         int32    `json:"target_year"`
	ForceRefresh       bool     `json:"force_refresh"`
	SourcesQueried     int32    `json:"sources_queried"`
	StartedAt          int64    `json:"started_at_ms"`
	TriggeredBy        string   `json:"triggered_by"`
	WasmRulesVersion   string   `json:"wasm_rules_version"`
	RecordsIngested    int32    `json:"records_ingested"`
	RecordsCreated     int32    `json:"records_created"`
	RecordsUpdated     int32    `json:"records_updated"`
	RecordsDeleted     int32    `json:"records_deleted"`
	ConflictsDetected  int32    `json:"conflicts_detected"`
	ConflictsResolved  int32    `json:"conflicts_resolved"`
	ConflictsEscalated int32    `json:"conflicts_escalated"`
	SourcesSucceeded   int32    `json:"sources_succeeded"`
	SourcesFailed      int32    `json:"sources_failed"`
	ErrorMessages      []string `json:"error_messages"`
	CompletedAt        int64    `json:"completed_at_ms"`
	DurationMs         int64    `json:"duration_ms"`
}

// TradingCalendarCache simulates a trading platform's calendar cache.
// In production, this would update your actual trading system's calendar.
type TradingCalendarCache struct {
	mu               sync.RWMutex
	businessDays     map[string]bool   // date -> is_business_day
	holidays         map[string]string // date -> holiday_name
	eventCounts      map[string]int    // event_type -> count
	lastUpdate       time.Time
	lastEventPerType map[string]*CalendarEvent // keyed by EventType
}

func NewTradingCalendarCache() *TradingCalendarCache {
	return &TradingCalendarCache{
		businessDays:     make(map[string]bool),
		holidays:         make(map[string]string),
		eventCounts:      make(map[string]int),
		lastEventPerType: make(map[string]*CalendarEvent),
	}
}

func (c *TradingCalendarCache) UpdateDay(event *CalendarEvent) {
	c.mu.Lock()
	defer c.mu.Unlock()

	dateStr := event.CalendarDate
	c.businessDays[dateStr] = event.IsBusinessDay
	if event.HolidayName != "" {
		c.holidays[dateStr] = event.HolidayName
	}

	c.lastUpdate = time.Now()
	c.lastEventPerType[event.EventType] = event
	c.eventCounts[event.EventType]++

	status := "💼 Business Day"
	if !event.IsBusinessDay {
		status = "🎉 Holiday"
		if event.HolidayName != "" {
			status += " (" + event.HolidayName + ")"
		}
	}

	fmt.Printf("[CACHE] %s | %s | %s | Confidence: %d%% | Source: %s\n",
		dateStr,
		event.Region,
		status,
		event.ConfidenceScore,
		event.SourceSystem,
	)
}

//nolint:gocritic // body kept verbatim from upstream
func main() {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		brokers = "localhost:9092"
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers),
		kgo.ConsumerGroup("trading-consumer"),
		kgo.ConsumeTopics("calendar-updates", "ingestion-lifecycle"),
	)
	if err != nil {
		log.Fatalf("Failed to create Kafka client: %v", err)
	}
	defer client.Close()

	cache := NewTradingCalendarCache()

	fmt.Println("🗓️  Trading Calendar Consumer Started")
	fmt.Println("=====================================")
	fmt.Printf("   Brokers: %s\n", brokers)
	fmt.Println("   Topics:  calendar-updates, ingestion-lifecycle")
	fmt.Println()

	ctx := context.Background()

	for {
		fetches := client.PollFetches(ctx)
		errs := fetches.Errors()
		if len(errs) > 0 {
			for _, e := range errs {
				log.Printf("fetch error from topic %s: %v", e.Topic, e.Err)
			}
		}

		fetches.EachRecord(func(record *kgo.Record) {
			switch record.Topic {
			case "calendar-updates":
				processCalendarUpdate(record, cache)
			case "ingestion-lifecycle":
				processIngestionEvent(record, cache)
			default:
				fmt.Printf("Unknown topic: %s\n", record.Topic)
			}
		})
	}
}

func processCalendarUpdate(record *kgo.Record, cache *TradingCalendarCache) {
	// Deserialize JSON
	event := &CalendarEvent{}
	if err := json.Unmarshal(record.Value, event); err != nil {
		fmt.Printf("⚠️  Failed to unmarshal calendar event: %v\n", err)
		return
	}

	// Process event based on type
	switch event.EventType {
	case "CALENDAR_UPDATE":
		cache.UpdateDay(event)

		// Example: Trigger trading system action if holiday
		if !event.IsBusinessDay {
			fmt.Printf("  ↳ 🎪 ACTION REQUIRED: Holiday detected - may need to reschedule trades\n")
		}

	default:
		fmt.Printf("[%s] Received event: %s\n", event.EventType, event.EventId[:8])
	}
}

func processIngestionEvent(record *kgo.Record, cache *TradingCalendarCache) {
	// Deserialize JSON
	event := &IngestionEvent{}
	if err := json.Unmarshal(record.Value, event); err != nil {
		fmt.Printf("⚠️  Failed to unmarshal ingestion event: %v\n", err)
		return
	}

	switch event.EventType {
	case "STARTED":
		fmt.Printf("\n🔄 INGESTION STARTED\n")
		fmt.Printf("   Ingestion ID: %s\n", event.IngestionId[:8])
		fmt.Printf("   Regions: %v\n", event.Regions)
		fmt.Printf("   Year: %d\n\n", event.TargetYear)

	case "COMPLETED":
		fmt.Printf("\n✅ INGESTION COMPLETED\n")
		fmt.Printf("   Status: %s\n", event.Status)
		fmt.Printf("   Records ingested: %d (created: %d, updated: %d, deleted: %d)\n",
			event.RecordsIngested, event.RecordsCreated, event.RecordsUpdated, event.RecordsDeleted)
		fmt.Printf("   Conflicts: detected: %d, resolved: %d, escalated: %d\n",
			event.ConflictsDetected, event.ConflictsResolved, event.ConflictsEscalated)
		fmt.Printf("   Sources: queried: %d, succeeded: %d, failed: %d\n",
			event.SourcesQueried, event.SourcesSucceeded, event.SourcesFailed)
		fmt.Printf("   Duration: %dms\n\n", event.DurationMs)

		if len(event.ErrorMessages) > 0 {
			fmt.Println("   Errors:")
			for _, msg := range event.ErrorMessages {
				fmt.Printf("     - %s\n", msg)
			}
			fmt.Println()
		}
	}
}
