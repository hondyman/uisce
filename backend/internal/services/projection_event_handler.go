package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hondyman/semlayer/backend/internal/models"
)

// ProjectionEventHandler connects the event bus to projection updates
// Phase 4b: Async event-driven projection updates for consistent read models
type ProjectionEventHandler interface {
	// Start listening to events and updating projections
	Start(ctx context.Context) error
	Stop() error

	// Handle specific event types
	OnBOEvent(ctx context.Context, event *models.Event) error
	OnInstanceEvent(ctx context.Context, event *models.Event) error

	// Get health metrics
	GetMetrics() ProjectionMetrics
}

// ProjectionEventHandlerImpl implements ProjectionEventHandler
type ProjectionEventHandlerImpl struct {
	projectionUpdater ProjectionUpdater
	eventConsumer     EventConsumer
	boQueue           chan *models.Event
	instanceQueue     chan *models.Event
	mu                sync.RWMutex
	isRunning         bool
	stopCh            chan struct{}
	metrics           ProjectionMetrics
}

// ProjectionMetrics tracks projection update performance
type ProjectionMetrics struct {
	TotalEventsProcessed     int64
	SuccessfulUpdates        int64
	FailedUpdates            int64
	AverageProcessingTime    time.Duration
	LastUpdateTime           time.Time
	ProjectionLag            time.Duration
	BOProjectionsCount       int64
	InstanceProjectionsCount int64
}

// NewProjectionEventHandler creates a new handler
func NewProjectionEventHandler(
	projectionUpdater ProjectionUpdater,
	eventConsumer EventConsumer,
) ProjectionEventHandler {
	return &ProjectionEventHandlerImpl{
		projectionUpdater: projectionUpdater,
		eventConsumer:     eventConsumer,
		boQueue:           make(chan *models.Event, 100),
		instanceQueue:     make(chan *models.Event, 100),
		stopCh:            make(chan struct{}),
	}
}

// Start begins listening for events
func (h *ProjectionEventHandlerImpl) Start(ctx context.Context) error {
	h.mu.Lock()
	if h.isRunning {
		h.mu.Unlock()
		return fmt.Errorf("projection event handler already running")
	}
	h.isRunning = true
	h.mu.Unlock()

	log.Println("[ProjectionEventHandler] Starting projection event listener")

	// Start BO event processor
	go h.processBoEvents(ctx)

	// Start Instance event processor
	go h.processInstanceEvents(ctx)

	// Start event listener (subscribes to RabbitMQ)
	go h.listenToEvents(ctx)

	return nil
}

// Stop stops the handler
func (h *ProjectionEventHandlerImpl) Stop() error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.isRunning {
		return fmt.Errorf("projection event handler not running")
	}

	close(h.stopCh)
	h.isRunning = false

	log.Println("[ProjectionEventHandler] Stopped projection event handler")
	return nil
}

// ============================================================================
// Event Listening
// ============================================================================

// listenToEvents subscribes to the event bus and routes events
func (h *ProjectionEventHandlerImpl) listenToEvents(ctx context.Context) {
	// Subscribe to semlayer.events exchange with a dedicated queue
	queueName := "projection-updates-" + generateCorrelationID()

	// This would be wired to the EventConsumer which provides the actual subscription
	// For now, this is a placeholder for the integration point

	log.Printf("[ProjectionEventHandler] Listening for events on queue: %s", queueName)

	for {
		select {
		case <-h.stopCh:
			return
		case <-ctx.Done():
			return
		default:
			// Events would come from EventConsumer subscription here
			// This is where we'd poll or receive events from RabbitMQ
			time.Sleep(100 * time.Millisecond)
		}
	}
}

// routeEvent sends event to appropriate queue based on type
func (h *ProjectionEventHandlerImpl) routeEvent(event *models.Event) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if !h.isRunning {
		log.Printf("[ProjectionEventHandler] Handler not running, dropping event: %s", event.ID)
		return
	}

	switch event.EventType {
	case "BusinessObjectCreated", "BusinessObjectUpdated", "BusinessObjectDeleted", "BusinessObjectCloned":
		select {
		case h.boQueue <- event:
		default:
			log.Printf("[ProjectionEventHandler] Warning: BO queue full, event may be dropped: %s", event.ID)
		}

	case "InstanceCreated", "InstanceUpdated", "InstanceDeleted":
		select {
		case h.instanceQueue <- event:
		default:
			log.Printf("[ProjectionEventHandler] Warning: Instance queue full, event may be dropped: %s", event.ID)
		}

	default:
		log.Printf("[ProjectionEventHandler] Unknown event type, skipping: %s", event.EventType)
	}
}

// ============================================================================
// BO Event Processing
// ============================================================================

// processBoEvents handles BO events
func (h *ProjectionEventHandlerImpl) processBoEvents(ctx context.Context) {
	for {
		select {
		case <-h.stopCh:
			return
		case <-ctx.Done():
			return
		case event := <-h.boQueue:
			err := h.OnBOEvent(ctx, event)
			h.recordMetric(err)

			if err != nil {
				log.Printf("[ProjectionEventHandler] Error processing BO event: %v", err)
			}
		}
	}
}

// OnBOEvent handles business object events
func (h *ProjectionEventHandlerImpl) OnBOEvent(ctx context.Context, event *models.Event) error {
	start := time.Now()

	var err error
	switch event.EventType {
	case "BusinessObjectCreated":
		err = h.projectionUpdater.HandleBOCreatedEvent(ctx, event)

	case "BusinessObjectUpdated":
		err = h.projectionUpdater.HandleBOUpdatedEvent(ctx, event)

	case "BusinessObjectDeleted":
		err = h.projectionUpdater.HandleBODeletedEvent(ctx, event)

	case "BusinessObjectCloned":
		err = h.projectionUpdater.HandleBOClonedEvent(ctx, event)

	default:
		err = fmt.Errorf("unknown BO event type: %s", event.EventType)
	}

	if err != nil {
		log.Printf("[ProjectionEventHandler] Failed to handle %s event %s: %v", event.EventType, event.ID, err)
	} else {
		log.Printf("[ProjectionEventHandler] Successfully processed %s event in %dms", event.EventType, time.Since(start).Milliseconds())
	}

	return err
}

// ============================================================================
// Instance Event Processing
// ============================================================================

// processInstanceEvents handles instance events
func (h *ProjectionEventHandlerImpl) processInstanceEvents(ctx context.Context) {
	for {
		select {
		case <-h.stopCh:
			return
		case <-ctx.Done():
			return
		case event := <-h.instanceQueue:
			err := h.OnInstanceEvent(ctx, event)
			h.recordMetric(err)

			if err != nil {
				log.Printf("[ProjectionEventHandler] Error processing Instance event: %v", err)
			}
		}
	}
}

// OnInstanceEvent handles instance events
func (h *ProjectionEventHandlerImpl) OnInstanceEvent(ctx context.Context, event *models.Event) error {
	start := time.Now()

	var err error
	switch event.EventType {
	case "InstanceCreated":
		err = h.projectionUpdater.HandleInstanceCreatedEvent(ctx, event)

	case "InstanceUpdated":
		err = h.projectionUpdater.HandleInstanceUpdatedEvent(ctx, event)

	case "InstanceDeleted":
		err = h.projectionUpdater.HandleInstanceDeletedEvent(ctx, event)

	default:
		err = fmt.Errorf("unknown Instance event type: %s", event.EventType)
	}

	if err != nil {
		log.Printf("[ProjectionEventHandler] Failed to handle %s event %s: %v", event.EventType, event.ID, err)
	} else {
		log.Printf("[ProjectionEventHandler] Successfully processed %s event in %dms", event.EventType, time.Since(start).Milliseconds())
	}

	return err
}

// ============================================================================
// Metrics & Health
// ============================================================================

// GetMetrics returns current metrics
func (h *ProjectionEventHandlerImpl) GetMetrics() ProjectionMetrics {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return h.metrics
}

// recordMetric updates metrics
func (h *ProjectionEventHandlerImpl) recordMetric(err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.metrics.TotalEventsProcessed++
	h.metrics.LastUpdateTime = time.Now()

	if err != nil {
		h.metrics.FailedUpdates++
	} else {
		h.metrics.SuccessfulUpdates++
	}
}

// ============================================================================
// Batch Processing (Optional - for recovery/catch-up)
// ============================================================================

// ProcessEventBatch processes multiple events as a batch
// Useful for recovery or initialization
func (h *ProjectionEventHandlerImpl) ProcessEventBatch(ctx context.Context, events []*models.Event) error {
	log.Printf("[ProjectionEventHandler] Processing batch of %d events", len(events))

	for _, event := range events {
		var err error

		// Route to appropriate handler
		switch event.EventType {
		case "BusinessObjectCreated", "BusinessObjectUpdated", "BusinessObjectDeleted", "BusinessObjectCloned":
			err = h.OnBOEvent(ctx, event)
		case "InstanceCreated", "InstanceUpdated", "InstanceDeleted":
			err = h.OnInstanceEvent(ctx, event)
		default:
			err = fmt.Errorf("unknown event type: %s", event.EventType)
		}

		if err != nil {
			log.Printf("[ProjectionEventHandler] Error processing event %s: %v", event.ID, err)
			// Continue processing despite errors
		}

		h.recordMetric(err)
	}

	return nil
}

// ============================================================================
// Query Projection State (Useful for Testing/Debugging)
// ============================================================================

// GetProjectionState returns current projection data
func (h *ProjectionEventHandlerImpl) GetProjectionState(ctx context.Context, boID string) (map[string]interface{}, error) {
	// This would query the projection tables and return the denormalized data
	// For now, it's a placeholder

	return map[string]interface{}{
		"bo_id":  boID,
		"status": "projected",
	}, nil
}

// ============================================================================
// Event Buffer Management (Handling Backpressure)
// ============================================================================

// GetQueueDepth returns how many events are pending
func (h *ProjectionEventHandlerImpl) GetQueueDepth() (boQueue int, instanceQueue int) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.boQueue), len(h.instanceQueue)
}

// HandleBackpressure pauses event processing if queues are full
func (h *ProjectionEventHandlerImpl) HandleBackpressure(ctx context.Context) {
	boDepth, instDepth := h.GetQueueDepth()

	if boDepth > 80 || instDepth > 80 {
		log.Printf("[ProjectionEventHandler] Backpressure detected - BO: %d, Instance: %d", boDepth, instDepth)
		// Could signal to event producer to slow down
	}
}

// ============================================================================
// Debugging & Tracing
// ============================================================================

// TraceEvent logs detailed event information for debugging
func (h *ProjectionEventHandlerImpl) TraceEvent(event *models.Event) {
	var payload map[string]interface{}
	_ = json.Unmarshal(event.Payload, &payload)

	log.Printf(
		"[ProjectionEventHandler] TRACE: EventType=%s, EventID=%s, CorrelationID=%s, Timestamp=%s",
		event.EventType,
		event.ID,
		event.CorrelationID,
		event.CreatedAt,
	)

	for key, val := range payload {
		log.Printf("  %s: %v", key, val)
	}
}

// ============================================================================
// Helper Functions
// ============================================================================

// generateCorrelationID is a placeholder - should come from util package
func generateCorrelationID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
