package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/websocket"

	"github.com/hondyman/semlayer/backend/internal/events"
)

// ============================================================================
// Phase 3.4: WebSocket HTTP Handlers
// Serves real-time event streams to connected clients
// ============================================================================

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, validate origin properly
		return true
	},
}

// WebSocketEventHandler handles WebSocket connections
type WebSocketEventHandler struct {
	broker *events.EventStreamBroker
}

// NewWebSocketEventHandler creates a new WebSocket handler
func NewWebSocketEventHandler(broker *events.EventStreamBroker) *WebSocketEventHandler {
	return &WebSocketEventHandler{
		broker: broker,
	}
}

// ServeHTTP handles WebSocket upgrade and streaming
func (h *WebSocketEventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Extract tenant ID and regions from query parameters
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "Missing tenant_id parameter", http.StatusBadRequest)
		return
	}

	// Parse regions parameter (comma-separated)
	regionsParam := r.URL.Query().Get("regions")
	var regions []string
	if regionsParam != "" {
		regions = strings.Split(regionsParam, ",")
		for i, r := range regions {
			regions[i] = strings.TrimSpace(r)
		}
	}

	// Upgrade connection to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer ws.Close()

	// Create subscriber
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	subscriber, err := h.broker.Subscribe(ctx, tenantID, regions)
	if err != nil {
		log.Printf("Subscribe error: %v", err)
		ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseInternalServerErr, err.Error()))
		return
	}

	// Handle ping/pong for keep-alive
	ws.SetReadDeadline(makeWriteDeadline())
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(makeWriteDeadline())
		return nil
	})

	// Stream events to WebSocket
	for {
		select {
		case event, ok := <-subscriber.EventChan:
			if !ok {
				ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}

			// Marshal event to JSON
			data, err := json.Marshal(event)
			if err != nil {
				log.Printf("JSON marshal error: %v", err)
				continue
			}

			// Send to WebSocket
			err = ws.WriteMessage(websocket.TextMessage, data)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}
				h.broker.Unsubscribe(subscriber.ID)
				return
			}

		case <-ctx.Done():
			h.broker.Unsubscribe(subscriber.ID)
			ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
	}
}

// HealthCheckEventHandler serves event broker health status
type HealthCheckEventHandler struct {
	broker *events.EventStreamBroker
}

// NewHealthCheckEventHandler creates a health check handler
func NewHealthCheckEventHandler(broker *events.EventStreamBroker) *HealthCheckEventHandler {
	return &HealthCheckEventHandler{
		broker: broker,
	}
}

// ServeHTTP returns event broker health status
func (h *HealthCheckEventHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	health := map[string]interface{}{
		"status":  "healthy",
		"service": "event-broker",
	}

	json.NewEncoder(w).Encode(health)
}

// makeWriteDeadline creates a write deadline for keep-alive
func makeWriteDeadline() time.Time {
	return time.Now().Add(60 * time.Second)
}

// EventStreamingMiddleware is HTTP middleware for event-aware operations
type EventStreamingMiddleware struct {
	broker *events.EventStreamBroker
}

// NewEventStreamingMiddleware creates middleware
func NewEventStreamingMiddleware(broker *events.EventStreamBroker) *EventStreamingMiddleware {
	return &EventStreamingMiddleware{
		broker: broker,
	}
}

// Wrap wraps an HTTP handler to stream events
func (m *EventStreamingMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add broker to request context
		ctx := context.WithValue(r.Context(), "eventBroker", m.broker)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetBrokerFromContext retrieves broker from request context
func GetBrokerFromContext(ctx context.Context) *events.EventStreamBroker {
	broker, ok := ctx.Value("eventBroker").(*events.EventStreamBroker)
	if !ok {
		return nil
	}
	return broker
}
