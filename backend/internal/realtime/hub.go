package realtime

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Event represents a real-time analytics or operational event
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	TenantID  string                 `json:"tenant_id"`
	Region    string                 `json:"region"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
}

// Client represents a WebSocket client connection
type Client struct {
	conn     *websocket.Conn
	send     chan []byte
	tenantID string
	regions  map[string]bool
	mu       sync.RWMutex
	closed   bool
}

// Hub manages all WebSocket connections and broadcasts events
type Hub struct {
	clients         map[*Client]bool
	mu              sync.RWMutex
	register        chan *Client
	unregister      chan *Client
	broadcast       chan *Event
	tenantClients   map[string][]*Client
	tenantClientsMu sync.RWMutex
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// NewHub creates and initializes a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:       make(map[*Client]bool),
		register:      make(chan *Client, 256),
		unregister:    make(chan *Client, 256),
		broadcast:     make(chan *Event, 1024),
		tenantClients: make(map[string][]*Client),
	}
}

// Run starts the hub's event loop (should run in a goroutine)
func (h *Hub) Run(ctx context.Context) {
	log.Println("WebSocket hub started")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("WebSocket hub shutting down")
			return

		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			h.tenantClientsMu.Lock()
			h.tenantClients[client.tenantID] = append(h.tenantClients[client.tenantID], client)
			h.tenantClientsMu.Unlock()

			log.Printf("Client registered: tenant=%s, total=%d", client.tenantID, len(h.clients))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mu.Unlock()

			h.tenantClientsMu.Lock()
			if clients, ok := h.tenantClients[client.tenantID]; ok {
				for i, c := range clients {
					if c == client {
						h.tenantClients[client.tenantID] = append(clients[:i], clients[i+1:]...)
						break
					}
				}
			}
			h.tenantClientsMu.Unlock()

			log.Printf("Client unregistered: tenant=%s, remaining=%d", client.tenantID, len(h.clients))

		case event := <-h.broadcast:
			h.broadcastEvent(event)

		case <-ticker.C:
			// Periodic cleanup of disconnected clients
			h.mu.Lock()
			for client := range h.clients {
				select {
				case client.send <- []byte(`{"type":"ping"}`):
				default:
					go func(c *Client) { h.unregister <- c }(client)
				}
			}
			h.mu.Unlock()
		}
	}
}

// broadcastEvent sends an event to all subscribed clients
func (h *Hub) broadcastEvent(event *Event) {
	h.tenantClientsMu.RLock()
	defer h.tenantClientsMu.RUnlock()

	if clients, ok := h.tenantClients[event.TenantID]; ok {
		payload, _ := json.Marshal(event)
		for _, client := range clients {
			if event.Region != "" && event.Region != "global" {
				client.mu.RLock()
				subscribed := client.regions[event.Region]
				client.mu.RUnlock()
				if !subscribed {
					continue
				}
			}

			select {
			case client.send <- payload:
			default:
				log.Printf("Dropping event for slow client: tenant=%s", client.tenantID)
			}
		}
	}
}

// Publish publishes an event to all subscribed clients
func (h *Hub) Publish(event *Event) {
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}

	select {
	case h.broadcast <- event:
	default:
		log.Println("Broadcast channel full, dropping event")
	}
}

// ServeWS handles WebSocket upgrades and client initialization
func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "tenant_id required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	client := &Client{
		conn:     conn,
		send:     make(chan []byte, 256),
		tenantID: tenantID,
		regions:  make(map[string]bool),
	}

	h.register <- client
	go client.readPump(h)
	go client.writePump()
}

// readPump reads messages from the client and handles subscriptions
func (c *Client) readPump(h *Hub) {
	defer func() {
		c.conn.Close()
		h.unregister <- c
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket error: %v", err)
			}
			return
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("Failed to parse message: %v", err)
			continue
		}

		if action, ok := msg["action"].(string); ok {
			c.mu.Lock()
			switch action {
			case "subscribe":
				if region, ok := msg["region"].(string); ok {
					c.regions[region] = true
					log.Printf("Client subscribed to region: %s", region)
				}
			case "unsubscribe":
				if region, ok := msg["region"].(string); ok {
					delete(c.regions, region)
					log.Printf("Client unsubscribed from region: %s", region)
				}
			}
			c.mu.Unlock()
		}
	}
}

// writePump writes messages to the client
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, []byte{}); err != nil {
				return
			}
		}
	}
}

// PublishEventHandler is an HTTP handler for external services to publish events
func (h *Hub) PublishEventHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, "invalid event payload", http.StatusBadRequest)
		return
	}

	if event.TenantID == "" || event.Type == "" {
		http.Error(w, "tenant_id and type required", http.StatusBadRequest)
		return
	}

	h.Publish(&event)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "published", "event_id": event.ID})
}

// StatsHandler returns hub statistics
func (h *Hub) StatsHandler(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	totalClients := len(h.clients)
	h.mu.RUnlock()

	h.tenantClientsMu.RLock()
	tenantCount := len(h.tenantClients)
	h.tenantClientsMu.RUnlock()

	stats := map[string]interface{}{
		"total_clients":  totalClients,
		"unique_tenants": tenantCount,
		"timestamp":      time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}
