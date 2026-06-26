package resilience

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ConsistencyEvent represents an event in the eventual consistency model
type ConsistencyEvent struct {
	ID            string
	AggregateID   string
	EventType     string
	Payload       interface{}
	Version       int64
	Timestamp     time.Time
	Status        string // "published", "acknowledged", "processed"
	RetryCount    int
	MaxRetries    int
	NextRetryTime time.Time
}

// ConsistencyModel manages eventual consistency across services
type ConsistencyModel struct {
	events           map[string]*ConsistencyEvent
	mu               sync.RWMutex
	subscribers      map[string][]EventHandler
	subMutex         sync.RWMutex
	metrics          *ConsistencyMetrics
	retryTicker      *time.Ticker
	acknowledgeMap   map[string]bool
	acknowledgeMutex sync.RWMutex
}

// EventHandler is a function that handles consistency events
type EventHandler func(context.Context, *ConsistencyEvent) error

// ConsistencyMetrics tracks consistency metrics
type ConsistencyMetrics struct {
	TotalEvents          int64
	PublishedEvents      int64
	AcknowledgedEvents   int64
	ProcessedEvents      int64
	FailedEvents         int64
	RetryCount           int64
	AverageLatency       time.Duration
	MaxLatency           time.Duration
	CurrentPendingEvents int64
	mu                   sync.RWMutex
}

// NewConsistencyModel creates a new eventual consistency model
func NewConsistencyModel() *ConsistencyModel {
	cm := &ConsistencyModel{
		events:         make(map[string]*ConsistencyEvent),
		subscribers:    make(map[string][]EventHandler),
		metrics:        &ConsistencyMetrics{},
		retryTicker:    time.NewTicker(10 * time.Second),
		acknowledgeMap: make(map[string]bool),
	}

	// Start retry goroutine
	go cm.retryFailedEvents()

	return cm
}

// PublishEvent publishes an event for eventual consistency
func (cm *ConsistencyModel) PublishEvent(
	ctx context.Context,
	eventID string,
	aggregateID string,
	eventType string,
	payload interface{},
	maxRetries int,
) *ConsistencyEvent {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	event := &ConsistencyEvent{
		ID:            eventID,
		AggregateID:   aggregateID,
		EventType:     eventType,
		Payload:       payload,
		Version:       1,
		Timestamp:     time.Now(),
		Status:        "published",
		MaxRetries:    maxRetries,
		NextRetryTime: time.Now(),
	}

	cm.events[eventID] = event

	cm.metrics.mu.Lock()
	cm.metrics.TotalEvents++
	cm.metrics.PublishedEvents++
	cm.metrics.CurrentPendingEvents = int64(len(cm.events))
	cm.metrics.mu.Unlock()

	// Asynchronously deliver to subscribers
	go cm.deliverEvent(ctx, event)

	return event
}

// SubscribeToEventType subscribes a handler to an event type
func (cm *ConsistencyModel) SubscribeToEventType(
	eventType string,
	handler EventHandler,
) {
	cm.subMutex.Lock()
	defer cm.subMutex.Unlock()

	cm.subscribers[eventType] = append(cm.subscribers[eventType], handler)
}

// deliverEvent delivers an event to all subscribers
func (cm *ConsistencyModel) deliverEvent(ctx context.Context, event *ConsistencyEvent) {
	cm.subMutex.RLock()
	handlers, exists := cm.subscribers[event.EventType]
	cm.subMutex.RUnlock()

	if !exists || len(handlers) == 0 {
		return
	}

	for _, handler := range handlers {
		// Execute handler with timeout
		handlerCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		err := handler(handlerCtx, event)
		cancel()

		if err != nil {
			// Mark for retry on failure
			cm.mu.Lock()
			event.RetryCount++
			event.Status = "failed"
			event.NextRetryTime = time.Now().Add(time.Duration(1<<uint(event.RetryCount)) * time.Second)
			cm.mu.Unlock()

			cm.metrics.mu.Lock()
			cm.metrics.FailedEvents++
			cm.metrics.RetryCount++
			cm.metrics.mu.Unlock()

			return
		}
	}

	// Mark as processed
	cm.mu.Lock()
	event.Status = "processed"
	cm.mu.Unlock()

	cm.metrics.mu.Lock()
	cm.metrics.ProcessedEvents++
	cm.metrics.mu.Unlock()
}

// AcknowledgeEvent marks an event as acknowledged by a service
func (cm *ConsistencyModel) AcknowledgeEvent(
	ctx context.Context,
	eventID string,
	acknowledgedBy string,
) error {
	cm.mu.RLock()
	event, exists := cm.events[eventID]
	cm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("event %s not found", eventID)
	}

	cm.acknowledgeMutex.Lock()
	cm.acknowledgeMap[fmt.Sprintf("%s-%s", eventID, acknowledgedBy)] = true
	cm.acknowledgeMutex.Unlock()

	cm.mu.Lock()
	event.Status = "acknowledged"
	cm.mu.Unlock()

	cm.metrics.mu.Lock()
	cm.metrics.AcknowledgedEvents++
	cm.metrics.mu.Unlock()

	return nil
}

// retryFailedEvents retries events that failed to be delivered
func (cm *ConsistencyModel) retryFailedEvents() {
	for range cm.retryTicker.C {
		cm.mu.Lock()
		now := time.Now()
		var failedEvents []*ConsistencyEvent

		for _, event := range cm.events {
			if event.Status == "failed" && event.RetryCount < event.MaxRetries && now.After(event.NextRetryTime) {
				failedEvents = append(failedEvents, event)
			}
		}
		cm.mu.Unlock()

		// Retry failed events
		for _, event := range failedEvents {
			go cm.deliverEvent(context.Background(), event)
		}
	}
}

// GetEventStatus retrieves the status of an event
func (cm *ConsistencyModel) GetEventStatus(eventID string) (*ConsistencyEvent, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	event, exists := cm.events[eventID]
	if !exists {
		return nil, fmt.Errorf("event %s not found", eventID)
	}

	return event, nil
}

// GetAggregateEvents retrieves all events for an aggregate
func (cm *ConsistencyModel) GetAggregateEvents(aggregateID string) []*ConsistencyEvent {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var aggregateEvents []*ConsistencyEvent
	for _, event := range cm.events {
		if event.AggregateID == aggregateID {
			aggregateEvents = append(aggregateEvents, event)
		}
	}

	return aggregateEvents
}

// WaitForEventProcessing blocks until an event is processed or timeout occurs
func (cm *ConsistencyModel) WaitForEventProcessing(
	ctx context.Context,
	eventID string,
	timeout time.Duration,
) error {
	deadline := time.Now().Add(timeout)

	for {
		cm.mu.RLock()
		event, exists := cm.events[eventID]
		cm.mu.RUnlock()

		if !exists {
			return fmt.Errorf("event %s not found", eventID)
		}

		if event.Status == "processed" {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for event %s to be processed", eventID)
		}

		// Check context deadline
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(100 * time.Millisecond):
			// Continue checking
		}
	}
}

// GetMetrics returns consistency metrics
func (cm *ConsistencyModel) GetMetrics() *ConsistencyMetrics {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()

	metricsCopy := &ConsistencyMetrics{
		TotalEvents:          cm.metrics.TotalEvents,
		PublishedEvents:      cm.metrics.PublishedEvents,
		AcknowledgedEvents:   cm.metrics.AcknowledgedEvents,
		ProcessedEvents:      cm.metrics.ProcessedEvents,
		FailedEvents:         cm.metrics.FailedEvents,
		RetryCount:           cm.metrics.RetryCount,
		AverageLatency:       cm.metrics.AverageLatency,
		MaxLatency:           cm.metrics.MaxLatency,
		CurrentPendingEvents: cm.metrics.CurrentPendingEvents,
	}

	return metricsCopy
}

// ExportMetrics exports consistency metrics in Prometheus format
func (cm *ConsistencyModel) ExportMetrics() string {
	cm.metrics.mu.RLock()
	defer cm.metrics.mu.RUnlock()

	processedRate := 0.0
	if cm.metrics.PublishedEvents > 0 {
		processedRate = float64(cm.metrics.ProcessedEvents) / float64(cm.metrics.PublishedEvents)
	}

	failureRate := 0.0
	if cm.metrics.PublishedEvents > 0 {
		failureRate = float64(cm.metrics.FailedEvents) / float64(cm.metrics.PublishedEvents)
	}

	return fmt.Sprintf(`
# Eventual Consistency Metrics
consistency_total_events %d
consistency_published_events %d
consistency_acknowledged_events %d
consistency_processed_events %d
consistency_failed_events %d
consistency_retry_count %d
consistency_processed_rate %.4f
consistency_failure_rate %.4f
consistency_current_pending_events %d
consistency_avg_latency_ms %d
consistency_max_latency_ms %d
`,
		cm.metrics.TotalEvents,
		cm.metrics.PublishedEvents,
		cm.metrics.AcknowledgedEvents,
		cm.metrics.ProcessedEvents,
		cm.metrics.FailedEvents,
		cm.metrics.RetryCount,
		processedRate,
		failureRate,
		cm.metrics.CurrentPendingEvents,
		cm.metrics.AverageLatency.Milliseconds(),
		cm.metrics.MaxLatency.Milliseconds(),
	)
}

// CleanupProcessedEvents removes events older than specified duration
func (cm *ConsistencyModel) CleanupProcessedEvents(maxAge time.Duration) int {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cutoff := time.Now().Add(-maxAge)
	removed := 0

	for id, event := range cm.events {
		if event.Status == "processed" && event.Timestamp.Before(cutoff) {
			delete(cm.events, id)
			removed++
		}
	}

	cm.metrics.mu.Lock()
	cm.metrics.CurrentPendingEvents = int64(len(cm.events))
	cm.metrics.mu.Unlock()

	return removed
}
