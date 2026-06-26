package services

import (
	"context"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Payload   interface{} `json:"payload"`
	Timestamp string      `json:"timestamp"`
}

// UpgradeStatusMessage represents a version status change
type UpgradeStatusMessage struct {
	CoreVersion string   `json:"coreVersion"`
	Status      string   `json:"status"`
	Warnings    []string `json:"warnings,omitempty"`
	Blockers    []string `json:"blockers,omitempty"`
}

// ConnectionState represents the state of a WebSocket connection
type ConnectionState int32

const (
	StateConnected ConnectionState = iota
	StateConnecting
	StateDisconnected
	StateFailed
)

// ClientInfo holds information about a connected client
type ClientInfo struct {
	conn         *websocket.Conn
	state        int32
	lastPing     time.Time
	connectTime  time.Time
	messageCount int64
	errorCount   int64
}

// WebSocketHub manages WebSocket connections and broadcasting with advanced error handling
type WebSocketHub struct {
	clients         map[*websocket.Conn]*ClientInfo
	broadcast       chan WebSocketMessage
	register        chan *websocket.Conn
	unregister      chan *websocket.Conn
	shutdown        chan struct{}
	wg              sync.WaitGroup
	mutex           sync.RWMutex
	upgrader        websocket.Upgrader
	messageQueue    chan WebSocketMessage
	queueSize       int
	healthCheckTick *time.Ticker
	metrics         *HubMetrics
}

// HubMetrics tracks WebSocket hub performance
type HubMetrics struct {
	messagesSent     int64
	messagesFailed   int64
	clientsConnected int64
	clientsFailed    int64
	uptime           time.Time
}

// NewWebSocketHub creates a new WebSocket hub with advanced features
func NewWebSocketHub() *WebSocketHub {
	hub := &WebSocketHub{
		clients:         make(map[*websocket.Conn]*ClientInfo),
		broadcast:       make(chan WebSocketMessage, 100), // Buffered channel
		register:        make(chan *websocket.Conn, 10),
		unregister:      make(chan *websocket.Conn, 10),
		shutdown:        make(chan struct{}),
		messageQueue:    make(chan WebSocketMessage, 1000), // Message queue for offline clients
		queueSize:       1000,
		healthCheckTick: time.NewTicker(30 * time.Second), // Health check every 30s
		metrics: &HubMetrics{
			uptime: time.Now(),
		},
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin in development
				// In production, you should restrict this to your domain
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
	}

	// Start background workers
	hub.wg.Add(3)
	go hub.run()
	go hub.processQueue()
	go hub.healthCheck()

	return hub
}

// Run starts the WebSocket hub with advanced error handling
func (h *WebSocketHub) Run() {
	h.wg.Add(1)
	go h.run()
}

// run is the main hub loop
func (h *WebSocketHub) run() {
	defer h.wg.Done()

	for {
		select {
		case <-h.shutdown:
			logging.GetLogger().Sugar().Info("WebSocket hub shutting down")
			return

		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// registerClient adds a new client with connection tracking
func (h *WebSocketHub) registerClient(client *websocket.Conn) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	clientInfo := &ClientInfo{
		conn:        client,
		state:       int32(StateConnected),
		lastPing:    time.Now(),
		connectTime: time.Now(),
	}

	h.clients[client] = clientInfo
	atomic.AddInt64(&h.metrics.clientsConnected, 1)

	logging.GetLogger().Sugar().Infow("WebSocket client connected", "clients", len(h.clients))
}

// unregisterClient removes a client and cleans up resources
func (h *WebSocketHub) unregisterClient(client *websocket.Conn) {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	if clientInfo, exists := h.clients[client]; exists {
		atomic.StoreInt32(&clientInfo.state, int32(StateDisconnected))
		delete(h.clients, client)
		client.Close()
		atomic.AddInt64(&h.metrics.clientsFailed, 1)
	}

	logging.GetLogger().Sugar().Infow("WebSocket client disconnected", "clients", len(h.clients))
}

// broadcastMessage sends a message to all connected clients with error handling
func (h *WebSocketHub) broadcastMessage(message WebSocketMessage) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	sentCount := int64(0)
	failedCount := int64(0)

	for client, clientInfo := range h.clients {
		if atomic.LoadInt32(&clientInfo.state) != int32(StateConnected) {
			continue
		}

		select {
		case h.messageQueue <- message:
			// Message queued for processing
		default:
			logging.GetLogger().Sugar().Warn("Message queue full, dropping message for client")
			atomic.AddInt64(&h.metrics.messagesFailed, 1)
			continue
		}

		// Send message with timeout and error handling
		go func(c *websocket.Conn, ci *ClientInfo, msg WebSocketMessage) {
			c.SetWriteDeadline(time.Now().Add(5 * time.Second))

			if err := c.WriteJSON(msg); err != nil {
				atomic.AddInt64(&ci.errorCount, 1)
				atomic.AddInt64(&h.metrics.messagesFailed, 1)
				logging.GetLogger().Sugar().Errorw("WebSocket write error for client", "error", err, "errors", atomic.LoadInt64(&ci.errorCount))

				// If too many errors, disconnect client
				if atomic.LoadInt64(&ci.errorCount) > 5 {
					h.unregister <- c
				}
			} else {
				atomic.AddInt64(&ci.messageCount, 1)
				atomic.AddInt64(&h.metrics.messagesSent, 1)
			}
		}(client, clientInfo, message)
	}

	logging.GetLogger().Sugar().Infow("Broadcast complete", "sent", sentCount, "failed", failedCount)
}

// processQueue processes queued messages for offline clients
func (h *WebSocketHub) processQueue() {
	defer h.wg.Done()

	for {
		select {
		case <-h.shutdown:
			return
		case message := <-h.messageQueue:
			// Process queued message
			h.broadcastMessage(message)
		}
	}
}

// healthCheck performs periodic health checks on connections
func (h *WebSocketHub) healthCheck() {
	defer h.wg.Done()

	for {
		select {
		case <-h.shutdown:
			return
		case <-h.healthCheckTick.C:
			h.performHealthCheck()
		}
	}
}

// performHealthCheck checks connection health and cleans up stale connections
func (h *WebSocketHub) performHealthCheck() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	now := time.Now()
	staleConnections := 0

	for client, clientInfo := range h.clients {
		// Check if connection is stale (no ping for 5 minutes)
		if now.Sub(clientInfo.lastPing) > 5*time.Minute {
			logging.GetLogger().Sugar().Infow("Removing stale WebSocket connection", "last_ping", clientInfo.lastPing)
			delete(h.clients, client)
			client.Close()
			staleConnections++
		}
	}

	if staleConnections > 0 {
		logging.GetLogger().Sugar().Infow("Health check removed stale connections", "count", staleConnections)
	}
}

// BroadcastUpgradeStatus broadcasts a version status change to all connected clients
func (h *WebSocketHub) BroadcastUpgradeStatus(coreVersion, status string, warnings, blockers []string) {
	message := WebSocketMessage{
		Type: "upgrade_status",
		Payload: UpgradeStatusMessage{
			CoreVersion: coreVersion,
			Status:      status,
			Warnings:    warnings,
			Blockers:    blockers,
		},
		Timestamp: time.Now().Format(time.RFC3339),
	}

	// Try to broadcast immediately, fallback to queue
	select {
	case h.broadcast <- message:
		// Message sent to broadcast channel
	default:
		// Broadcast channel full, try queue
		select {
		case h.messageQueue <- message:
			logging.GetLogger().Sugar().Warn("Message queued due to full broadcast channel")
		default:
			logging.GetLogger().Sugar().Warn("Both broadcast and queue channels full, dropping message")
			atomic.AddInt64(&h.metrics.messagesFailed, 1)
		}
	}
}

// RegisterRoutes registers the websocket handlers
func (h *WebSocketHub) RegisterRoutes(r chi.Router) {
	r.Get("/ws", h.HandleWebSocket)
}

// HandleWebSocket handles WebSocket connections with advanced error handling
func (h *WebSocketHub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logging.GetLogger().Sugar().Errorw("WebSocket upgrade error", "error", err)
		http.Error(w, "WebSocket upgrade failed", http.StatusBadRequest)
		return
	}

	// Set connection parameters
	conn.SetReadLimit(512 * 1024) // 512KB max message size
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// Register the client
	h.register <- conn

	// Start ping routine
	go h.pingClient(conn)

	// Handle client messages
	go func() {
		defer func() {
			h.unregister <- conn
		}()

		for {
			var msg WebSocketMessage
			err := conn.ReadJSON(&msg)
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					logging.GetLogger().Sugar().Errorw("WebSocket read error", "error", err)
				}
				break
			}

			// Handle ping messages
			if msg.Type == "ping" {
				response := WebSocketMessage{
					Type:      "pong",
					Payload:   map[string]interface{}{"timestamp": time.Now().Unix()},
					Timestamp: time.Now().Format(time.RFC3339),
				}

				if err := conn.WriteJSON(response); err != nil {
					logging.GetLogger().Sugar().Errorw("WebSocket pong write error", "error", err)
					break
				}
				continue
			}

			// Handle other client messages
			response := WebSocketMessage{
				Type:      "ack",
				Payload:   map[string]string{"received": msg.Type},
				Timestamp: msg.Timestamp,
			}

			if err := conn.WriteJSON(response); err != nil {
				logging.GetLogger().Sugar().Errorw("WebSocket ack write error", "error", err)
				break
			}
		}
	}()
}

// pingClient sends periodic pings to maintain connection health
func (h *WebSocketHub) pingClient(conn *websocket.Conn) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-h.shutdown:
			return
		case <-ticker.C:
			h.mutex.RLock()
			if clientInfo, exists := h.clients[conn]; exists {
				clientInfo.lastPing = time.Now()
			}
			h.mutex.RUnlock()

			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logging.GetLogger().Sugar().Errorw("WebSocket ping error", "error", err)
				return
			}
		}
	}
}

// GetClientCount returns the number of connected clients
func (h *WebSocketHub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// GetMetrics returns hub performance metrics
func (h *WebSocketHub) GetMetrics() *HubMetrics {
	return &HubMetrics{
		messagesSent:     atomic.LoadInt64(&h.metrics.messagesSent),
		messagesFailed:   atomic.LoadInt64(&h.metrics.messagesFailed),
		clientsConnected: atomic.LoadInt64(&h.metrics.clientsConnected),
		clientsFailed:    atomic.LoadInt64(&h.metrics.clientsFailed),
		uptime:           h.metrics.uptime,
	}
}

// Shutdown gracefully shuts down the WebSocket hub
func (h *WebSocketHub) Shutdown(ctx context.Context) error {
	logging.GetLogger().Sugar().Info("Initiating WebSocket hub shutdown")

	// Signal shutdown
	close(h.shutdown)

	// Stop health check ticker
	h.healthCheckTick.Stop()

	// Close all client connections
	h.mutex.Lock()
	for client := range h.clients {
		client.Close()
	}
	h.clients = nil
	h.mutex.Unlock()

	// Wait for all goroutines to finish
	done := make(chan struct{})
	go func() {
		h.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logging.GetLogger().Sugar().Info("WebSocket hub shutdown complete")
		return nil
	case <-ctx.Done():
		logging.GetLogger().Sugar().Warn("WebSocket hub shutdown timeout")
		return ctx.Err()
	}
}
