package api

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket Hub for managing real-time connections
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mutex      sync.RWMutex
}

type WebSocketClient struct {
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	audience string
	hub      *WebSocketHub
}

type RealTimeMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	UserID    string      `json:"user_id,omitempty"`
}

type FundUpdateMessage struct {
	FundID    string                 `json:"fund_id"`
	Metrics   map[string]interface{} `json:"metrics"`
	UpdatedAt time.Time              `json:"updated_at"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Restrict origin in production
		env := os.Getenv("ENVIRONMENT")
		if env == "production" || env == "prod" {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true // Direct connection (e.g. mobile app, backend service)
			}
			// Enforce same-origin policy or specific allowed domains
			// For now, ensuring the origin matches the host is a reasonable baseline
			return strings.Contains(origin, r.Host)
		}
		// Allow any origin in development
		return true
	},
}

// Note: CheckOrigin signature differs in gorilla versions; the upgrader in api.go used
// an http.Request-based CheckOrigin. We'll set a permissive default in the hub and the
// handler will supply the proper upgrader options at runtime if needed.

func newWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
	}
}

func (h *WebSocketHub) run() {
	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			h.clients[client] = true
			h.mutex.Unlock()

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
			h.mutex.Unlock()

		case message := <-h.broadcast:
			h.mutex.RLock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mutex.RUnlock()
		}
	}
}

func (h *WebSocketHub) broadcastToAudience(audience string, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		if client.audience == audience || client.audience == "" {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

//lint:ignore U1000 retained for compatibility with older hub implementations
func (h *WebSocketHub) broadcastToUser(userID string, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// readPump reads messages from the WebSocket connection. It currently discards incoming messages.
func (c *WebSocketClient) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		// incoming messages are ignored in this implementation
	}
}

// writePump writes messages from the send channel to the WebSocket connection and
// periodically sends pings.
func (c *WebSocketClient) writePump() {
	ticker := time.NewTicker(54 * time.Second)
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
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
