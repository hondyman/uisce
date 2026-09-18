package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"os"
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

	// subIndex is the fan-out reverse index for streaming subscriptions:
	// (channel, key) -> the subscriptions listening on it. See
	// streaming_channels.go for HandleSubscribe/HandleUnsubscribe/FanOut.
	subMu    sync.Mutex
	subIndex map[channelKey]map[SubscriptionID]*subscription
}

type WebSocketClient struct {
	conn     *websocket.Conn
	send     chan []byte
	userID   string
	tenantID string
	audience string
	hub      *WebSocketHub

	// connectedAt anchors the max-connection-lifetime enforcement: the ticket
	// that authenticated the upgrade only proves identity at that instant, so
	// a long-lived connection is force-closed after maxConnectionLifetime and
	// must re-authenticate via a fresh ticket to reconnect.
	connectedAt time.Time

	// revokedMu/revoked back the Client.revoke() extension point: a stub for
	// a future session-revocation hook to force-close a live connection
	// before its natural max-lifetime expiry. Not wired to anything yet.
	revokedMu sync.Mutex
	revoked   bool

	// subsMu/subscriptions track this client's live streaming subscriptions
	// so a disconnect can tear them all down via the hub's reverse index
	// (see streaming_channels.go) instead of leaking them.
	subsMu        sync.Mutex
	subscriptions map[SubscriptionID]*subscription
}

// maxConnectionLifetime bounds how long a single WebSocket connection may
// stay open before it must reconnect with a freshly issued ticket. See the
// connectedAt field comment for why this exists.
const maxConnectionLifetime = 4 * time.Hour

// revoke marks the client as revoked and force-closes its connection. It is
// currently only reachable from the max-lifetime timer, but is named/shaped
// so a future session-revocation hook can call it directly.
func (c *WebSocketClient) revoke() {
	c.revokedMu.Lock()
	alreadyRevoked := c.revoked
	c.revoked = true
	c.revokedMu.Unlock()
	if alreadyRevoked {
		return
	}
	_ = c.conn.Close()
}

// isRevoked reports whether the client has been revoked (lifetime expiry or
// a future revocation hook), which readPump/writePump treat as terminal.
func (c *WebSocketClient) isRevoked() bool {
	c.revokedMu.Lock()
	defer c.revokedMu.Unlock()
	return c.revoked
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
			// Enforce same-origin: parse the Origin header and compare its host
			// exactly, rather than substring-matching, which a hostname like
			// "https://api.example.com.evil.tld" would pass against r.Host
			// "api.example.com".
			originURL, err := url.Parse(origin)
			if err != nil {
				return false
			}
			return originURL.Host == r.Host
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
		subIndex:   make(map[channelKey]map[SubscriptionID]*subscription),
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
			// Tear down every subscription this connection held so the
			// reverse index never fans out to a dead socket and doesn't
			// leak entries for connections that come and go.
			h.teardownSubscriptions(client)

		case message := <-h.broadcast:
			// Write lock: this branch can delete from h.clients on a full
			// send buffer, so a read lock here would race with that delete.
			h.mutex.Lock()
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
			h.mutex.Unlock()
		}
	}
}

func (h *WebSocketHub) broadcastToAudience(audience string, message []byte) {
	// Write lock: the default branch below deletes from h.clients.
	h.mutex.Lock()
	defer h.mutex.Unlock()

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
	// Write lock: the default branch below deletes from h.clients.
	h.mutex.Lock()
	defer h.mutex.Unlock()

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

// readPump reads messages from the WebSocket connection. Streaming
// subscribe/unsubscribe requests (see streaming_channels.go) are handled
// here; any other message shape is ignored, preserving the prior
// discard-unknown-messages behavior for non-streaming connections.
func (c *WebSocketClient) readPump() {
	lifetimeTimer := time.AfterFunc(maxConnectionLifetime, c.revoke)
	defer func() {
		lifetimeTimer.Stop()
		c.hub.unregister <- c
		c.conn.Close()
	}()

	// 64KB: large enough for a subscribe message carrying several hundred
	// instrument/order keys (see SubscribeRequest.Keys) without falling back
	// to the old 512-byte limit, which would fail any real subscribe payload
	// with a read-limit error and silently kill the connection.
	c.conn.SetReadLimit(64 * 1024)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
		c.handleIncomingMessage(raw)
	}
}

// handleIncomingMessage parses a single client message as a streaming
// action; malformed or unrecognized payloads are ignored rather than
// closing the connection, matching the prior discard-everything behavior
// for non-streaming clients that also share this hub.
func (c *WebSocketClient) handleIncomingMessage(raw []byte) {
	var req streamActionRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return
	}
	switch req.Action {
	case streamActionSubscribe:
		resp := c.hub.HandleSubscribe(c, req)
		c.sendJSON(resp)
	case streamActionUnsubscribe:
		resp := c.hub.HandleUnsubscribe(c, req)
		c.sendJSON(resp)
	default:
		// Unknown action: ignore, do not close the connection.
	}
}

// sendJSON marshals v and pushes it onto the client's send channel, going
// through the same channel writePump reads from so it remains the single
// writer on the underlying connection (gorilla/websocket requires this).
// A full channel drops the message (select/default) rather than blocking or
// growing unboundedly.
func (c *WebSocketClient) sendJSON(v interface{}) {
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	select {
	case c.send <- b:
	default:
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
