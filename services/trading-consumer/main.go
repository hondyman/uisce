package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
	"google.golang.org/protobuf/proto"

	eventspb "calendar-service/pkg/proto/calendar/events/v1"
)

// TradingCalendarCache simulates a trading platform's calendar cache
// In production, this would update your actual trading system's calendar
type TradingCalendarCache struct {
	mu               sync.RWMutex
	businessDays     map[string]bool   // date -> is_business_day
	holidays         map[string]string // date -> holiday_name
	eventCounts      map[string]int    // event_type -> count
	lastUpdate       time.Time
	lastEventPerType map[string]*eventspb.CalendarEvent
}

func NewTradingCalendarCache() *TradingCalendarCache {
	return &TradingCalendarCache{
		businessDays:     make(map[string]bool),
		holidays:         make(map[string]string),
		eventCounts:      make(map[string]int),
		lastEventPerType: make(map[string]*eventspb.CalendarEvent),
	}
}

func (c *TradingCalendarCache) UpdateDay(event *eventspb.CalendarEvent) {
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
		status,
		event.Region,
		event.ConfidenceScore,
		event.SourceSystem,
	)
}

func (c *TradingCalendarCache) IsBusinessDay(date string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if val, exists := c.businessDays[date]; exists {
		return val
	}
	// Default to business day if not in cache
	return true
}

func (c *TradingCalendarCache) Stats() {
	c.mu.RLock()
	defer c.mu.RUnlock()

	fmt.Println("\n📊 Cache Statistics:")
	fmt.Printf("  Total days cached: %d\n", len(c.businessDays))
	fmt.Printf("  Holidays found: %d\n", len(c.holidays))
	fmt.Printf("  Events received: %d\n", sumMap(c.eventCounts))

	for eventType, count := range c.eventCounts {
		fmt.Printf("    - %s: %d\n", eventType, count)
	}

	fmt.Printf("  Last update: %v\n\n", c.lastUpdate)
}

func sumMap(m map[string]int) int {
	sum := 0
	for _, v := range m {
		sum += v
	}
	return sum
}

func main() {
	// Configuration
	brokers := []string{os.Getenv("REDPANDA_BROKERS")}
	if len(brokers) == 0 || brokers[0] == "" {
		brokers = []string{"redpanda:9092"}
	}

	groupID := os.Getenv("CONSUMER_GROUP_ID")
	if groupID == "" {
		groupID = "trading-platform-consumer"
	}

	fmt.Println("╔════════════════════════════════════════╗")
	fmt.Println("║  🚀 Trading Platform Consumer          ║")
	fmt.Println("║     Real-Time Calendar Sync            ║")
	fmt.Println("╚════════════════════════════════════════╝")
	fmt.Printf("\nConnecting to Redpanda at %v\n", brokers)
	fmt.Printf("Consumer group: %s\n\n", groupID)

	// Create Kafka client
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumeTopics("calendar-updates", "ingestion-lifecycle"),
		kgo.ConsumerGroup(groupID),
		kgo.SessionTimeout(30*time.Second),
		kgo.Balancers(kgo.RoundRobinBalancer()),
	)
	if err != nil {
		log.Fatalf("❌ Failed to create Kafka client: %v", err)
	}
	defer client.Close()

	fmt.Println("✅ Connected to Redpanda\n")

	cache := NewTradingCalendarCache()

	// Setup stats ticker
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	go func() {
		for range ticker.C {
			cache.Stats()
		}
	}()

	// Setup graceful shutdown
	ctx := context.Background()
	messageCount := 0

	fmt.Println("📡 Listening for calendar updates...\n")
	fmt.Println("Events will appear as they arrive in real-time:")
	fmt.Println("")

	// Main consumption loop
	for {
		fetches := client.PollFetches(ctx)

		// Handle errors
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, err := range errs {
				fmt.Printf("⚠️  Error: %v\n", err)
			}
			continue
		}

		// Process records
		fetches.EachRecord(func(r *kgo.Record) {
			messageCount++

			processMessage(r, cache)

			// Commit offset after successful processing
			client.CommitUncommittedOffsets(ctx)
		})
	}
}

func processMessage(record *kgo.Record, cache *TradingCalendarCache) {
	// Determine message type by topic
	switch record.Topic {
	case "calendar-updates":
		processCalendarUpdate(record, cache)
	case "ingestion-lifecycle":
		processIngestionEvent(record, cache)
	default:
		fmt.Printf("⚠️  Unknown topic: %s\n", record.Topic)
	}
}

func processCalendarUpdate(record *kgo.Record, cache *TradingCalendarCache) {
	// Deserialize Protobuf
	event := &eventspb.CalendarEvent{}
	if err := proto.Unmarshal(record.Value, event); err != nil {
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
	// Deserialize Protobuf
	event := &eventspb.IngestionEvent{}
	if err := proto.Unmarshal(record.Value, event); err != nil {
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
