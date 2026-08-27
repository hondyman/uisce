package api

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type StreamEventType string

const (
	EventIBORPositionTick StreamEventType = "IBOR_POSITION_TICK"
	EventABORJournalPost   StreamEventType = "ABOR_JOURNAL_POSTED"
	EventTaxLossHarvested StreamEventType = "TAX_LOSS_HARVESTED"
	EventZKProofGenerated StreamEventType = "ZK_PROOF_VERIFIED"
	EventReconBreakAlert  StreamEventType = "RECON_BREAK_DETECTED"
)

type StreamMessage struct {
	Type      StreamEventType `json:"type"`
	TenantID  uuid.UUID       `json:"tenant_id"`
	Timestamp time.Time       `json:"timestamp"`
	Payload   json.RawMessage `json:"payload"`
}

type ClientConnection struct {
	conn     *websocket.Conn
	tenantID uuid.UUID
	sendChan chan []byte
}

type InstitutionalWebSocketGateway struct {
	clients    map[*ClientConnection]bool
	broadcast  chan StreamMessage
	register   chan *ClientConnection
	unregister chan *ClientConnection
	mu         sync.RWMutex
}

func NewInstitutionalWebSocketGateway() *InstitutionalWebSocketGateway {
	return &InstitutionalWebSocketGateway{
		clients:    make(map[*ClientConnection]bool),
		broadcast:  make(chan StreamMessage, 2048),
		register:   make(chan *ClientConnection),
		unregister: make(chan *ClientConnection),
	}
}

func (gw *InstitutionalWebSocketGateway) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			gw.mu.Lock()
			for client := range gw.clients {
				close(client.sendChan)
				_ = client.conn.Close()
				delete(gw.clients, client)
			}
			gw.mu.Unlock()
			return

		case client := <-gw.register:
			gw.mu.Lock()
			gw.clients[client] = true
			gw.mu.Unlock()

		case client := <-gw.unregister:
			gw.mu.Lock()
			if _, ok := gw.clients[client]; ok {
				close(client.sendChan)
				_ = client.conn.Close()
				delete(gw.clients, client)
			}
			gw.mu.Unlock()

		case msg := <-gw.broadcast:
			payloadBytes, err := json.Marshal(msg)
			if err != nil {
				continue
			}

			gw.mu.RLock()
			for client := range gw.clients {
				if client.tenantID == msg.TenantID || client.tenantID == uuid.MustParse("00000000-0000-0000-0000-000000000000") {
					select {
					case client.sendChan <- payloadBytes:
					default:
						close(client.sendChan)
						delete(gw.clients, client)
					}
				}
			}
			gw.mu.RUnlock()
		}
	}
}

func (gw *InstitutionalWebSocketGateway) BroadcastEvent(tenantID uuid.UUID, eventType StreamEventType, payload interface{}) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}

	gw.broadcast <- StreamMessage{
		Type:      eventType,
		TenantID:  tenantID,
		Timestamp: time.Now().UTC(),
		Payload:   raw,
	}
}

func (gw *InstitutionalWebSocketGateway) ServeWebSocket(w http.ResponseWriter, r *http.Request) {
	tenantIDStr := r.URL.Query().Get("tenant_id")
	tenantID, err := uuid.Parse(tenantIDStr)
	if err != nil {
		http.Error(w, "Rule 7 violation: valid tenant_id required for stream handshake", http.StatusUnauthorized)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	client := &ClientConnection{
		conn:     conn,
		tenantID: tenantID,
		sendChan: make(chan []byte, 256),
	}

	gw.register <- client

	go gw.writePump(client)
	go gw.readPump(client)
}

func (gw *InstitutionalWebSocketGateway) readPump(c *ClientConnection) {
	defer func() {
		gw.unregister <- c
	}()

	c.conn.SetReadLimit(4096)
	_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := c.conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (gw *InstitutionalWebSocketGateway) writePump(c *ClientConnection) {
	ticker := time.NewTicker(25 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.sendChan:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(msg)
			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
