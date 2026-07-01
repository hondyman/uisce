package main

import (
	"bytes"
	"context"
	"database/sql"
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
	_ "github.com/lib/pq"
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
	CreatedAt  time.Time       `json:"created_at"`
}

// RawEvent is the inbound event payload
type RawEvent struct {
	TenantID  uuid.UUID   `json:"tenant_id"`
	BOType    string      `json:"bo_type"`
	EventType string      `json:"event_type"`
	FieldName string      `json:"field_name"`
	OldValue  interface{} `json:"old_value"`
	NewValue  interface{} `json:"new_value"`
	Metadata  interface{} `json:"metadata"`
}

// RoutedEvent is the outbound event payload published to Kafka
type RoutedEvent struct {
	*RawEvent
	ConfigID   uuid.UUID `json:"config_id"`
	RoutedAt   time.Time `json:"routed_at"`
	RouteQueue string    `json:"route_queue"`
}

var (
	db          *sql.DB
	kafkaWriter *kafka.Writer
	configCache = make(map[string][]EventConfig)
	cacheMu     sync.RWMutex
)

func main() {
	// Initialize PostgreSQL connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable"
	}

	var err error
	db, err = sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connection established")

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
	go refreshConfigCache()

	// Gin router
	r := gin.Default()

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	// Event processing endpoint
	r.POST("/events", processEventHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8083"
	}
	log.Printf("Event Router starting on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// refreshConfigCache periodically fetches configurations from the database and updates the in-memory cache
func refreshConfigCache() {
	tick := time.NewTicker(5 * time.Minute)
	defer tick.Stop()

	// Initial load
	fetchAndCacheConfigs()

	for range tick.C {
		fetchAndCacheConfigs()
	}
}

// fetchAndCacheConfigs executes a SQL query to fetch all event configs directly from PostgreSQL
func fetchAndCacheConfigs() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, event_type, bo_type, field_name,
		       COALESCE(filter_json, '{}')::text, route_queue, created_at
		FROM event_configs
		ORDER BY created_at
	`)
	if err != nil {
		log.Printf("Config fetch failed: %v, will retry", err)
		return
	}
	defer rows.Close()

	newCache := make(map[string][]EventConfig)
	totalCount := 0
	for rows.Next() {
		var cfg EventConfig
		var idStr, tenantIDStr, filterStr string
		if err := rows.Scan(&idStr, &tenantIDStr, &cfg.EventType, &cfg.BOType,
			&cfg.FieldName, &filterStr, &cfg.RouteQueue, &cfg.CreatedAt); err != nil {
			log.Printf("Row scan failed: %v", err)
			continue
		}
		cfg.ID = uuid.MustParse(idStr)
		cfg.TenantID = uuid.MustParse(tenantIDStr)
		cfg.FilterJSON = json.RawMessage(filterStr)

		key := fmt.Sprintf("%s_%s", cfg.BOType, cfg.EventType)
		newCache[key] = append(newCache[key], cfg)
		totalCount++
	}
	if err := rows.Err(); err != nil {
		log.Printf("Row iteration error: %v", err)
		return
	}

	cacheMu.Lock()
	configCache = newCache
	cacheMu.Unlock()

	log.Printf("Config cache updated: %d total configs", totalCount)
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
