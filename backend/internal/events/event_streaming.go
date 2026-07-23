package events

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// Phase 3.4: WebSocket Event Streaming
// Real-time incident, RCA, and action updates to frontend clients
// ============================================================================

// StreamedEvent represents a single event sent over WebSocket
type StreamedEvent struct {
	ID         string                 `json:"id"`
	Type       EventType              `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	TenantID   string                 `json:"tenant_id"`
	IncidentID string                 `json:"incident_id,omitempty"`
	Region     string                 `json:"region,omitempty"`
	Severity   string                 `json:"severity,omitempty"`
	Payload    map[string]interface{} `json:"payload"`
}

// EventSubscriber represents a client subscription
type EventSubscriber struct {
	ID        string
	TenantID  string
	Regions   []string // Subscribe to specific regions or all
	EventChan chan *StreamedEvent
	Done      chan struct{}
}

// EventStreamBroker manages real-time event streaming to multiple subscribers
type EventStreamBroker struct {
	subscribers map[string]*EventSubscriber
	subMutex    sync.RWMutex

	// Event channels
	inbound chan *StreamedEvent

	// Buffering for backpressure
	bufferSize int

	// Retention for late subscribers (last N events)
	eventBuffer chan *StreamedEvent
	bufferMutex sync.RWMutex

	// Shutdown control
	done chan struct{}
	once sync.Once
}

// NewEventStreamBroker creates a new event broker
func NewEventStreamBroker(bufferSize int) *EventStreamBroker {
	broker := &EventStreamBroker{
		subscribers: make(map[string]*EventSubscriber),
		inbound:     make(chan *StreamedEvent, bufferSize),
		bufferSize:  bufferSize,
		eventBuffer: make(chan *StreamedEvent, bufferSize),
		done:        make(chan struct{}),
	}

	// Start broker event loop
	go broker.eventLoop()

	return broker
}

// Subscribe registers a new event subscriber
func (b *EventStreamBroker) Subscribe(ctx context.Context, tenantID string, regions []string) (*EventSubscriber, error) {
	subscriber := &EventSubscriber{
		ID:        uuid.New().String(),
		TenantID:  tenantID,
		Regions:   regions,
		EventChan: make(chan *StreamedEvent, 100), // Buffer individual subscriber
		Done:      make(chan struct{}),
	}

	b.subMutex.Lock()
	b.subscribers[subscriber.ID] = subscriber
	b.subMutex.Unlock()

	return subscriber, nil
}

// Unsubscribe removes an event subscriber
func (b *EventStreamBroker) Unsubscribe(subscriberID string) error {
	b.subMutex.Lock()
	subscriber, exists := b.subscribers[subscriberID]
	if exists {
		delete(b.subscribers, subscriberID)
	}
	b.subMutex.Unlock()

	if !exists {
		return fmt.Errorf("subscriber not found: %s", subscriberID)
	}

	close(subscriber.Done)
	close(subscriber.EventChan)

	return nil
}

// PublishEvent publishes an event to all matching subscribers
func (b *EventStreamBroker) PublishEvent(ctx context.Context, event *StreamedEvent) (err error) {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Prevent panic if inbound channel has been closed concurrently
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("publish panic: %v", r)
		}
	}()

	select {
	case b.inbound <- event:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	default:
		return fmt.Errorf("event buffer full")
	}
}

// eventLoop processes events and distributes to subscribers
func (b *EventStreamBroker) eventLoop() {
	for event := range b.inbound {
		// Buffer event for late subscribers
		select {
		case b.eventBuffer <- event:
		default:
			// Buffer full, discard oldest
			select {
			case <-b.eventBuffer:
				b.eventBuffer <- event
			default:
			}
		}

		// Distribute to matching subscribers
		b.subMutex.RLock()
		subscribers := make([]*EventSubscriber, 0, len(b.subscribers))
		for _, sub := range b.subscribers {
			subscribers = append(subscribers, sub)
		}
		b.subMutex.RUnlock()

		for _, subscriber := range subscribers {
			// Check if event matches subscriber filters
			if !b.matchesSubscriber(event, subscriber) {
				continue
			}

			// Send with timeout to avoid blocking on slow subscribers
			select {
			case subscriber.EventChan <- event:
			case <-time.After(5 * time.Second):
				// Subscriber is slow, log and skip
				fmt.Printf("Slow subscriber %s, skipping event\n", subscriber.ID)
			case <-subscriber.Done:
				// Subscriber disconnected
			}
		}
	}
}

// matchesSubscriber checks if event should be sent to subscriber
func (b *EventStreamBroker) matchesSubscriber(event *StreamedEvent, subscriber *EventSubscriber) bool {
	// Tenant must match
	if event.TenantID != subscriber.TenantID {
		return false
	}

	// If subscriber has region filters, event region must match
	if len(subscriber.Regions) > 0 && event.Region != "" {
		found := false
		for _, region := range subscriber.Regions {
			if region == event.Region {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// Stop gracefully shuts down the event broker and disconnects all subscribers
func (b *EventStreamBroker) Stop() error {
	var err error
	b.once.Do(func() {
		// Close inbound to signal event loop to stop
		close(b.inbound)

		// Wait for event loop to finish processing
		time.Sleep(100 * time.Millisecond)

		// Disconnect all subscribers
		b.subMutex.Lock()
		subscribers := make([]*EventSubscriber, 0, len(b.subscribers))
		for _, sub := range b.subscribers {
			subscribers = append(subscribers, sub)
		}
		b.subscribers = make(map[string]*EventSubscriber)
		b.subMutex.Unlock()

		for _, sub := range subscribers {
			close(sub.Done)
			close(sub.EventChan)
		}

		close(b.done)
	})
	return err
}

// GetSubscribers returns current subscriber map (for testing)
func (b *EventStreamBroker) GetSubscribers() map[string]*EventSubscriber {
	b.subMutex.RLock()
	defer b.subMutex.RUnlock()
	result := make(map[string]*EventSubscriber)
	for k, v := range b.subscribers {
		result[k] = v
	}
	return result
}

// IncidentEventFactory creates typed incident events
type IncidentEventFactory struct {
	broker *EventStreamBroker
}

// NewIncidentEventFactory creates an event factory
func NewIncidentEventFactory(broker *EventStreamBroker) *IncidentEventFactory {
	return &IncidentEventFactory{
		broker: broker,
	}
}

// NewIncidentDetected publishes incident.detected event
func (f *IncidentEventFactory) NewIncidentDetected(ctx context.Context, tenantID string, incidentID string, title string, severity string, region string) error {
	event := &StreamedEvent{
		Type:       EventTypeIncidentDetected,
		TenantID:   tenantID,
		IncidentID: incidentID,
		Region:     region,
		Severity:   severity,
		Payload: map[string]interface{}{
			"title":       title,
			"incident_id": incidentID,
		},
	}
	return f.broker.PublishEvent(ctx, event)
}

// RCAStarted publishes rca.started event
func (f *IncidentEventFactory) RCAStarted(ctx context.Context, tenantID string, incidentID string, region string) error {
	event := &StreamedEvent{
		Type:       EventTypeRCAStarted,
		TenantID:   tenantID,
		IncidentID: incidentID,
		Region:     region,
		Payload: map[string]interface{}{
			"status": "analyzing",
		},
	}
	return f.broker.PublishEvent(ctx, event)
}

// RCACompleted publishes rca.results event with results
func (f *IncidentEventFactory) RCACompleted(ctx context.Context, tenantID string, incidentID string, region string, rcaResults map[string]interface{}) error {
	event := &StreamedEvent{
		Type:       EventTypeRCAResultsAvailable,
		TenantID:   tenantID,
		IncidentID: incidentID,
		Region:     region,
		Payload:    rcaResults,
	}
	return f.broker.PublishEvent(ctx, event)
}

// ActionStarted publishes action.started event
func (f *IncidentEventFactory) ActionStarted(ctx context.Context, tenantID string, incidentID string, actionID string, actionType string, region string) error {
	event := &StreamedEvent{
		Type:       EventTypeActionStarted,
		TenantID:   tenantID,
		IncidentID: incidentID,
		Region:     region,
		Payload: map[string]interface{}{
			"action_id":   actionID,
			"action_type": actionType,
			"status":      "executing",
		},
	}
	return f.broker.PublishEvent(ctx, event)
}

// ActionCompleted publishes action.completed event
func (f *IncidentEventFactory) ActionCompleted(ctx context.Context, tenantID string, incidentID string, actionID string, actionType string, region string, result map[string]interface{}) error {
	event := &StreamedEvent{
		Type:       EventTypeActionCompleted,
		TenantID:   tenantID,
		IncidentID: incidentID,
		Region:     region,
		Payload: map[string]interface{}{
			"action_id":   actionID,
			"action_type": actionType,
			"status":      "completed",
			"result":      result,
		},
	}
	return f.broker.PublishEvent(ctx, event)
}

// PropagationDetected publishes propagation.detected event
func (f *IncidentEventFactory) PropagationDetected(ctx context.Context, tenantID string, incidentID string, fromRegion string, toRegions []string, likelihood float64) error {
	event := &StreamedEvent{
		Type:       EventTypePropagationDetected,
		TenantID:   tenantID,
		IncidentID: incidentID,
		Region:     fromRegion,
		Payload: map[string]interface{}{
			"from_region":      fromRegion,
			"to_regions":       toRegions,
			"likelihood_score": likelihood,
		},
	}
	return f.broker.PublishEvent(ctx, event)
}

// RegionFailover publishes region.failover event
func (f *IncidentEventFactory) RegionFailover(ctx context.Context, tenantID string, fromRegion string, toRegion string) error {
	event := &StreamedEvent{
		Type:     EventTypeRegionFailover,
		TenantID: tenantID,
		Region:   fromRegion,
		Payload: map[string]interface{}{
			"from_region": fromRegion,
			"to_region":   toRegion,
			"status":      "completed",
		},
	}
	return f.broker.PublishEvent(ctx, event)
}

// IncidentResolved publishes incident.resolved event
func (f *IncidentEventFactory) IncidentResolved(ctx context.Context, tenantID string, incidentID string, region string, resolution map[string]interface{}) error {
	event := &StreamedEvent{
		Type:       EventTypeIncidentResolved,
		TenantID:   tenantID,
		IncidentID: incidentID,
		Region:     region,
		Payload:    resolution,
	}
	return f.broker.PublishEvent(ctx, event)
}

// EventAggregator batches events for the frontend
type EventAggregator struct {
	broker     *EventStreamBroker
	batchSize  int
	flushTime  time.Duration
	aggregated chan []*StreamedEvent
}

// NewEventAggregator creates an event aggregator
func NewEventAggregator(broker *EventStreamBroker, batchSize int, flushTime time.Duration) *EventAggregator {
	return &EventAggregator{
		broker:     broker,
		batchSize:  batchSize,
		flushTime:  flushTime,
		aggregated: make(chan []*StreamedEvent, batchSize),
	}
}

// Subscribe returns a channel of aggregated event batches
func (ea *EventAggregator) Subscribe(ctx context.Context, tenantID string, regions []string) (chan []*StreamedEvent, error) {
	subscriber, err := ea.broker.Subscribe(ctx, tenantID, regions)
	if err != nil {
		return nil, err
	}

	output := make(chan []*StreamedEvent, 10)

	// Batch events from subscriber
	go func() {
		defer close(output)
		batch := make([]*StreamedEvent, 0, ea.batchSize)
		ticker := time.NewTicker(ea.flushTime)
		defer ticker.Stop()

		for {
			select {
			case event, ok := <-subscriber.EventChan:
				if !ok {
					if len(batch) > 0 {
						select {
						case output <- batch:
						case <-ctx.Done():
							return
						}
					}
					return
				}

				batch = append(batch, event)

				if len(batch) >= ea.batchSize {
					select {
					case output <- batch:
						batch = make([]*StreamedEvent, 0, ea.batchSize)
					case <-ctx.Done():
						return
					}
				}

			case <-ticker.C:
				if len(batch) > 0 {
					select {
					case output <- batch:
						batch = make([]*StreamedEvent, 0, ea.batchSize)
					case <-ctx.Done():
						return
					}
				}

			case <-ctx.Done():
				return

			case <-subscriber.Done:
				return
			}
		}
	}()

	return output, nil
}
