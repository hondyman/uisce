package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/machinebox/graphql"
	kafka "github.com/segmentio/kafka-go"
)

// EventConfig mirrors the PostgreSQL event_configs table
type EventConfig struct {
	ID         uuid.UUID       `json:"id"`
	TenantID   uuid.UUID       `json:"tenant_id"`
	EventType  string          `json:"event_type"`
	BOType     string          `json:"bo_type"`
	FieldName  string          `json:"field_name"`
	FilterJSON json.RawMessage `json:"filter_json"`
	RouteQueue string          `json:"route_queue"`
	CreatedAt  string          `json:"created_at"`
}

// RawEvent represents an incoming event from the core application
type RawEvent struct {
	TenantID   uuid.UUID       `json:"tenant_id"`
	BOType     string          `json:"bo_type"`
	BOID       string          `json:"bo_id"`
	EventType  string          `json:"event_type"`
	FieldName  string          `json:"field_name"`
	OldValue   interface{}     `json:"old_value"`
	NewValue   interface{}     `json:"new_value"`
	ChangedBy  string          `json:"changed_by"`
	CustomData json.RawMessage `json:"custom_data"`
}

// RoutedEvent is the enriched event sent to Redpanda/Kafka (topic)
type RoutedEvent struct {
	*RawEvent
	ConfigID   uuid.UUID `json:"config_id"`
	RoutedAt   time.Time `json:"routed_at"`
	RouteQueue string    `json:"route_queue"`
}

var (
	hasuraClient *graphql.Client
	kafkaWriter  *kafka.Writer
	configCache  = make(map[string][]EventConfig)
	cacheMu      sync.RWMutex
)

func main() {
	// Initialize Hasura client
	hasuraURL := os.Getenv("HASURA_URL")
	if hasuraURL == "" {
		hasuraURL = "http://localhost:8080/v1/graphql"
	}
	hasuraSecret := os.Getenv("HASURA_ADMIN_SECRET")
	if hasuraSecret == "" {
		hasuraSecret = "your-secret-key"
	}

	hasuraClient = graphql.NewClient(hasuraURL)

	// Initialize Kafka writer (Redpanda)
	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		kafkaBrokers = "localhost:9092"
	}
	brokers := strings.Split(kafkaBrokers, ",")

	kafkaWriter = &kafka.Writer{
		Addr:     kafka.TCP(brokers...),
		Balancer: &kafka.LeastBytes{},
	}
	defer kafkaWriter.Close()

	// Start config cache refresher (every 5 minutes)
	go refreshConfigCache(hasuraSecret)

	// Gin router
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Event processing endpoint
	r.POST("/events", processEventHandler)

	log.Println("Event-router service starting on :8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// refreshConfigCache periodically fetches configurations from Hasura and updates the in-memory cache
func refreshConfigCache(hasuraSecret string) {
	tick := time.NewTicker(5 * time.Minute)
	defer tick.Stop()

	// Initial load
	fetchAndCacheConfigs(hasuraSecret)

	for range tick.C {
		fetchAndCacheConfigs(hasuraSecret)
	}
}

// fetchAndCacheConfigs executes a GraphQL query against Hasura to fetch all event configs
func fetchAndCacheConfigs(hasuraSecret string) {
	req := graphql.NewRequest(`
		query FetchConfigs {
			event_configs {
				id
				tenant_id
				event_type
				bo_type
				field_name
				filter_json
				route_queue
				created_at
			}
		}
	`)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var resp struct {
		EventConfigs []EventConfig `json:"event_configs"`
	}

	if err := hasuraClient.Run(ctx, req, &resp); err != nil {
		log.Printf("Config fetch failed: %v, will retry", err)
		return
	}

	cacheMu.Lock()
	configCache = make(map[string][]EventConfig)
	for _, config := range resp.EventConfigs {
		key := fmt.Sprintf("%s_%s", config.BOType, config.EventType)
		configCache[key] = append(configCache[key], config)
	}
	cacheMu.Unlock()

	log.Printf("Config cache updated: %d total configs", len(resp.EventConfigs))
}

// processEventHandler handles incoming events and routes them to Redpanda/Kafka (publishes to configured topic)
func processEventHandler(c *gin.Context) {
	var rawEvent RawEvent
	if err := c.BindJSON(&rawEvent); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body", "details": err.Error()})
		return
	}

	// Validate required fields
	if rawEvent.TenantID == uuid.Nil || rawEvent.BOType == "" || rawEvent.EventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id, bo_type, and event_type are required"})
		return
	}

	log.Printf("[Event] Received: tenant=%s, bo_type=%s, event_type=%s, field=%s",
		rawEvent.TenantID, rawEvent.BOType, rawEvent.EventType, rawEvent.FieldName)

	// Look up matching configs
	cacheKey := fmt.Sprintf("%s_%s", rawEvent.BOType, rawEvent.EventType)
	cacheMu.RLock()
	matchedConfigs, exists := configCache[cacheKey]
	cacheMu.RUnlock()

	if !exists || len(matchedConfigs) == 0 {
		log.Printf("[Event] No matching configs for %s, ignoring", cacheKey)
		c.JSON(http.StatusOK, gin.H{"status": "no matching configs"})
		return
	}

	routedCount := 0
	for _, config := range matchedConfigs {
		// Tenant-scoped filter
		if rawEvent.TenantID != config.TenantID {
			continue
		}

		// Apply filter logic (expand as needed)
		if !applyFilter(config.FilterJSON, &rawEvent) {
			log.Printf("[Event] Event filtered out by config %s", config.ID)
			continue
		}

		// Create routed event
		routedEvent := RoutedEvent{
			RawEvent:   &rawEvent,
			ConfigID:   config.ID,
			RoutedAt:   time.Now(),
			RouteQueue: config.RouteQueue,
		}

		// Publish to Kafka topic (use config.RouteQueue as topic name)
		eventBody, _ := json.Marshal(routedEvent)
		msg := kafka.Message{
			Topic: config.RouteQueue,
			Key:   []byte(rawEvent.TenantID.String()),
			Value: eventBody,
			Time:  time.Now(),
		}
		if err := kafkaWriter.WriteMessages(context.Background(), msg); err != nil {
			log.Printf("[Event] Publish to topic %s failed: %v", config.RouteQueue, err)
			continue
		}

		log.Printf("[Event] Routed to queue=%s, config=%s", config.RouteQueue, config.ID)
		routedCount++
	}

	if routedCount == 0 {
		log.Printf("[Event] Event matched no valid configs after filtering")
		c.JSON(http.StatusOK, gin.H{"status": "no matching configs after filter"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":       "routed",
		"routed_count": routedCount,
		"config_count": len(matchedConfigs),
	})
}

// applyFilter evaluates a JSON filter against an event's new_value
// Supports filters like {"min_value": 100, "max_value": 1000} for numeric comparisons
// Extend this function to support more complex filtering logic
func applyFilter(filterJSON json.RawMessage, event *RawEvent) bool {
	if len(filterJSON) == 0 || string(filterJSON) == "{}" {
		return true
	}

	var filter map[string]interface{}
	if err := json.Unmarshal(filterJSON, &filter); err != nil {
		log.Printf("Filter unmarshal failed: %v", err)
		return false
	}

	// Example: numeric min/max filters
	if minVal, ok := filter["min_value"].(float64); ok {
		if newVal, ok := event.NewValue.(float64); ok {
			if newVal < minVal {
				return false
			}
		}
	}

	if maxVal, ok := filter["max_value"].(float64); ok {
		if newVal, ok := event.NewValue.(float64); ok {
			if newVal > maxVal {
				return false
			}
		}
	}

	// Example: string match filter
	if contains, ok := filter["contains"].(string); ok {
		if newStr, ok := event.NewValue.(string); ok {
			if !bytes.Contains([]byte(newStr), []byte(contains)) {
				return false
			}
		}
	}

	return true
}
