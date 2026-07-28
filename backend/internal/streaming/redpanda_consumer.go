package streaming

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"
)

// CDCEvent represents a Change Data Capture event from Redpanda/Kafka
type CDCEvent struct {
	Topic     string                 `json:"topic"`
	Partition int32                  `json:"partition"`
	Offset    int64                  `json:"offset"`
	Key       string                 `json:"key"`
	EventType string                 `json:"event_type"` // INSERT, UPDATE, DELETE
	TableName string                 `json:"table_name"`
	Before    map[string]interface{} `json:"before,omitempty"`
	After     map[string]interface{} `json:"after,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	TenantID  string                 `json:"tenant_id"`
}

// MarketDataTick represents real-time market data
type MarketDataTick struct {
	InstrumentID string    `json:"instrument_id"`
	Ticker       string    `json:"ticker"`
	Price        float64   `json:"price"`
	Bid          float64   `json:"bid"`
	Ask          float64   `json:"ask"`
	Volume       int64     `json:"volume"`
	Timestamp    time.Time `json:"timestamp"`
	Source       string    `json:"source"`
}

// TopicSubscription represents a client subscribed to specific BO/table events
type TopicSubscription struct {
	ClientID  string
	Topics    []string // table names or BO IDs
	TenantID  string
	MsgChan   chan []byte
	CreatedAt time.Time
}

// MultiTopicHub extends the basic Hub with topic-based routing
type MultiTopicHub struct {
	*Hub
	subscriptions map[string][]*TopicSubscription // keyed by topic (table name or BO ID)
	mu            sync.RWMutex
	globalSub     chan CDCEvent
}

// NewMultiTopicHub creates a topic-aware hub with CDC event routing
func NewMultiTopicHub() *MultiTopicHub {
	return &MultiTopicHub{
		Hub:           NewHub(),
		subscriptions: make(map[string][]*TopicSubscription),
		globalSub:     make(chan CDCEvent, 1024),
	}
}

// Subscribe registers a subscription for specific topics/BO IDs
func (h *MultiTopicHub) Subscribe(sub *TopicSubscription) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, topic := range sub.Topics {
		h.subscriptions[topic] = append(h.subscriptions[topic], sub)
	}
	log.Printf("[StreamHub] Client %s subscribed to topics: %v", sub.ClientID, sub.Topics)
}

// Unsubscribe removes a client subscription
func (h *MultiTopicHub) Unsubscribe(clientID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for topic, subs := range h.subscriptions {
		filtered := subs[:0]
		for _, s := range subs {
			if s.ClientID != clientID {
				filtered = append(filtered, s)
			}
		}
		h.subscriptions[topic] = filtered
	}
}

// RouteCDCEvent dispatches a CDC event to all subscribers of the affected table/BO
func (h *MultiTopicHub) RouteCDCEvent(event CDCEvent) {
	h.mu.RLock()
	subs, ok := h.subscriptions[event.TableName]
	h.mu.RUnlock()

	if !ok || len(subs) == 0 {
		// No topic subscribers — broadcast globally
		h.BroadcastBOEvent(event.TableName, event.EventType, event.After)
		return
	}

	msg := map[string]interface{}{
		"type":       "cdc_event",
		"table":      event.TableName,
		"event_type": event.EventType,
		"tenant_id":  event.TenantID,
		"before":     event.Before,
		"after":      event.After,
		"timestamp":  event.Timestamp,
	}
	bytes, _ := json.Marshal(msg)

	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, sub := range subs {
		// Only route to matching tenant
		if sub.TenantID == "" || sub.TenantID == event.TenantID {
			select {
			case sub.MsgChan <- bytes:
			default:
				log.Printf("[StreamHub] Dropping event for slow subscriber %s", sub.ClientID)
			}
		}
	}
}

// RouteMarketTick routes a market data tick to subscribers of the instrument
func (h *MultiTopicHub) RouteMarketTick(tick MarketDataTick) {
	msg := map[string]interface{}{
		"type":          "market_tick",
		"instrument_id": tick.InstrumentID,
		"ticker":        tick.Ticker,
		"price":         tick.Price,
		"bid":           tick.Bid,
		"ask":           tick.Ask,
		"volume":        tick.Volume,
		"timestamp":     tick.Timestamp,
	}
	bytes, _ := json.Marshal(msg)
	h.broadcast <- bytes
}

// KafkaConsumerConfig holds Redpanda/Kafka connection settings
type KafkaConsumerConfig struct {
	Brokers  []string `json:"brokers"`
	GroupID  string   `json:"group_id"`
	Topics   []string `json:"topics"`
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	SASL     bool     `json:"sasl"`
}

// RedpandaCDCConsumer consumes CDC events from Redpanda/Kafka and routes them to the hub
// In production this uses the Kafka Go client; here we implement the interface for integration
type RedpandaCDCConsumer struct {
	config  KafkaConsumerConfig
	hub     *MultiTopicHub
	running bool
	stopCh  chan struct{}
}

// NewRedpandaCDCConsumer creates a consumer bound to the streaming hub
func NewRedpandaCDCConsumer(cfg KafkaConsumerConfig, hub *MultiTopicHub) *RedpandaCDCConsumer {
	return &RedpandaCDCConsumer{
		config: cfg,
		hub:    hub,
		stopCh: make(chan struct{}),
	}
}

// Start begins consuming CDC events (simulation mode without actual Kafka dependency)
func (c *RedpandaCDCConsumer) Start(ctx context.Context) error {
	if c.running {
		return fmt.Errorf("consumer already running")
	}
	c.running = true
	log.Printf("[RedpandaCDC] Starting consumer for topics: %v on brokers: %v", c.config.Topics, c.config.Brokers)

	go c.consumeLoop(ctx)
	return nil
}

func (c *RedpandaCDCConsumer) consumeLoop(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	offset := int64(0)

	for {
		select {
		case <-ctx.Done():
			log.Println("[RedpandaCDC] Context cancelled, stopping consumer")
			c.running = false
			return
		case <-c.stopCh:
			log.Println("[RedpandaCDC] Stop signal received")
			c.running = false
			return
		case <-ticker.C:
			// In production: poll Kafka for new records
			// Here we simulate a heartbeat CDC event for integration testing
			event := CDCEvent{
				Topic:     "uisce.cdc.positions",
				Partition: 0,
				Offset:    offset,
				EventType: "UPDATE",
				TableName: "positions",
				After: map[string]interface{}{
					"account_id": "ACC001",
					"market_value": 1250000.0 + float64(offset)*100,
					"timestamp":    time.Now(),
				},
				Timestamp: time.Now(),
				TenantID:  "core",
			}
			c.hub.RouteCDCEvent(event)
			offset++
			log.Printf("[RedpandaCDC] Processed CDC event offset=%d", offset)
		}
	}
}

// Stop halts the consumer
func (c *RedpandaCDCConsumer) Stop() {
	if c.running {
		close(c.stopCh)
	}
}

// IngestMarketTick allows external market data services to push ticks into the hub
func IngestMarketTick(hub *MultiTopicHub, tick MarketDataTick) {
	hub.RouteMarketTick(tick)
}
