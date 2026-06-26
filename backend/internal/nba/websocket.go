package nba

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hondyman/semlayer/libs/jwt-middleware"
)

// WebSocketHub manages WebSocket connections for real-time NBA updates
type WebSocketHub struct {
	// Registered clients by advisor ID
	clients map[string]map[*WebSocketClient]bool

	// Broadcast channel for new recommendations
	broadcast chan *NBABroadcast

	// Register requests from clients
	register chan *WebSocketClient

	// Unregister requests from clients
	unregister chan *WebSocketClient

	mu sync.RWMutex
}

// WebSocketClient represents a connected advisor client
type WebSocketClient struct {
	Hub       *WebSocketHub
	Conn      *websocket.Conn
	Send      chan []byte
	AdvisorID string
	TenantID  string
}

// NBABroadcast contains a recommendation to broadcast
type NBABroadcast struct {
	AdvisorID       string           `json:"advisor_id"`
	TenantID        string           `json:"tenant_id"`
	Recommendations []NextBestAction `json:"recommendations"`
	Type            string           `json:"type"` // "NEW_RECOMMENDATION", "SIGNAL_DETECTED", "ACTION_COMPLETED"
	Timestamp       time.Time        `json:"timestamp"`
}

// WebSocket configuration
const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512 * 1024 // 512KB
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In production, validate origin properly
		return true
	},
}

// NewWebSocketHub creates a new WebSocket hub
func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[string]map[*WebSocketClient]bool),
		broadcast:  make(chan *NBABroadcast, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
}

// Run starts the hub's main loop
func (h *WebSocketHub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Shutdown: close all connections
			h.mu.Lock()
			for _, clients := range h.clients {
				for client := range clients {
					close(client.Send)
				}
			}
			h.clients = make(map[string]map[*WebSocketClient]bool)
			h.mu.Unlock()
			return

		case client := <-h.register:
			h.mu.Lock()
			if h.clients[client.AdvisorID] == nil {
				h.clients[client.AdvisorID] = make(map[*WebSocketClient]bool)
			}
			h.clients[client.AdvisorID][client] = true
			h.mu.Unlock()
			log.Printf("NBA WebSocket: Advisor %s connected", client.AdvisorID)

		case client := <-h.unregister:
			h.mu.Lock()
			if clients, ok := h.clients[client.AdvisorID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.clients, client.AdvisorID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("NBA WebSocket: Advisor %s disconnected", client.AdvisorID)

		case broadcast := <-h.broadcast:
			h.broadcastToAdvisor(broadcast)
		}
	}
}

// broadcastToAdvisor sends a message to all connections for an advisor
func (h *WebSocketHub) broadcastToAdvisor(broadcast *NBABroadcast) {
	message, err := json.Marshal(broadcast)
	if err != nil {
		log.Printf("NBA WebSocket: Failed to marshal broadcast: %v", err)
		return
	}

	h.mu.RLock()
	clients, ok := h.clients[broadcast.AdvisorID]
	h.mu.RUnlock()

	if !ok {
		return
	}

	for client := range clients {
		select {
		case client.Send <- message:
		default:
			// Client buffer full, disconnect
			h.unregister <- client
		}
	}
}

// BroadcastNewRecommendation sends a new recommendation to an advisor
func (h *WebSocketHub) BroadcastNewRecommendation(advisorID, tenantID string, recommendations []NextBestAction) {
	h.broadcast <- &NBABroadcast{
		AdvisorID:       advisorID,
		TenantID:        tenantID,
		Recommendations: recommendations,
		Type:            "NEW_RECOMMENDATION",
		Timestamp:       time.Now(),
	}
}

// BroadcastSignalDetected notifies advisor of a new signal
func (h *WebSocketHub) BroadcastSignalDetected(advisorID, tenantID string, signal DetectedSignal) {
	// Convert signal to a minimal recommendation for display
	h.broadcast <- &NBABroadcast{
		AdvisorID: advisorID,
		TenantID:  tenantID,
		Recommendations: []NextBestAction{{
			ActionID:      uuid.New(),
			ClientID:      signal.ClientID,
			ActionType:    "SIGNAL_ALERT",
			ActionName:    signal.SignalType,
			TriggerSignal: signal.SignalType,
			UrgencyScore:  signal.Strength,
		}},
		Type:      "SIGNAL_DETECTED",
		Timestamp: time.Now(),
	}
}

// ServeWs handles WebSocket connection requests
func (h *WebSocketHub) ServeWs(w http.ResponseWriter, r *http.Request) {
	// Get advisor ID from query or header
	advisorID := r.URL.Query().Get("advisor_id")
	if advisorID == "" {
		advisorID = r.Header.Get("X-Advisor-ID")
	}
	if advisorID == "" {
		http.Error(w, "advisor_id required", http.StatusBadRequest)
		return
	}

	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = jwtmiddleware.GetClaimsFromContext(r).TenantID
	}

	// Upgrade connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("NBA WebSocket: Upgrade failed: %v", err)
		return
	}

	client := &WebSocketClient{
		Hub:       h,
		Conn:      conn,
		Send:      make(chan []byte, 256),
		AdvisorID: advisorID,
		TenantID:  tenantID,
	}

	h.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// readPump handles incoming messages from the client
func (c *WebSocketClient) readPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("NBA WebSocket: Read error: %v", err)
			}
			break
		}

		// Handle incoming messages (e.g., action updates from client)
		c.handleMessage(message)
	}
}

// writePump handles sending messages to the client
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub closed the channel
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Batch any queued messages
			n := len(c.Send)
			for i := 0; i < n; i++ {
				w.Write([]byte{'\n'})
				w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from the client
func (c *WebSocketClient) handleMessage(message []byte) {
	var msg struct {
		Type     string `json:"type"`
		ActionID string `json:"action_id,omitempty"`
		Status   string `json:"status,omitempty"`
	}

	if err := json.Unmarshal(message, &msg); err != nil {
		log.Printf("NBA WebSocket: Failed to parse message: %v", err)
		return
	}

	switch msg.Type {
	case "ACTION_STARTED":
		log.Printf("NBA WebSocket: Advisor %s started action %s", c.AdvisorID, msg.ActionID)
	case "ACTION_COMPLETED":
		log.Printf("NBA WebSocket: Advisor %s completed action %s with status %s", c.AdvisorID, msg.ActionID, msg.Status)
	case "PING":
		// Respond with pong
		response, _ := json.Marshal(map[string]string{"type": "PONG"})
		c.Send <- response
	}
}

// Global hub instance
var globalHub *WebSocketHub
var hubOnce sync.Once

// GetWebSocketHub returns the global WebSocket hub instance
func GetWebSocketHub() *WebSocketHub {
	hubOnce.Do(func() {
		globalHub = NewWebSocketHub()
	})
	return globalHub
}

// NBAWebSocketHandler is an HTTP handler for the NBA WebSocket endpoint
func NBAWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	hub := GetWebSocketHub()
	hub.ServeWs(w, r)
}
