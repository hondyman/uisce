package handlers

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var driftUpgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type DriftCount struct {
	TenantID    string `json:"tenant_id"`
	Count       int    `json:"count"`
	LastSeen    string `json:"last_seen"`
	JobID       string `json:"job_id,omitempty"`
	Severity    string `json:"severity,omitempty"`
	Description string `json:"description,omitempty"`
}

type DriftWSHub struct {
	mu      sync.RWMutex
	subs    map[string]map[*DriftSubscriber]struct{}
	register chan *DriftSubscriber
	unregister chan *DriftSubscriber
}

type DriftSubscriber struct {
	TenantID  string
	EventChan chan *DriftCount
	done      chan struct{}
}

func NewDriftWSHub() *DriftWSHub {
	hub := &DriftWSHub{
		subs:       make(map[string]map[*DriftSubscriber]struct{}),
		register:   make(chan *DriftSubscriber),
		unregister: make(chan *DriftSubscriber),
	}
	go hub.run()
	return hub
}

func (h *DriftWSHub) run() {
	for {
		select {
		case sub := <-h.register:
			h.mu.Lock()
			if h.subs[sub.TenantID] == nil {
				h.subs[sub.TenantID] = make(map[*DriftSubscriber]struct{})
			}
			h.subs[sub.TenantID][sub] = struct{}{}
			h.mu.Unlock()

		case sub := <-h.unregister:
			h.mu.Lock()
			if set, ok := h.subs[sub.TenantID]; ok {
				delete(set, sub)
				close(sub.EventChan)
			}
			h.mu.Unlock()
		}
	}
}

func (h *DriftWSHub) Subscribe(tenantID string) *DriftSubscriber {
	sub := &DriftSubscriber{
		TenantID:  tenantID,
		EventChan: make(chan *DriftCount, 100),
		done:      make(chan struct{}),
	}
	h.register <- sub
	return sub
}

func (h *DriftWSHub) Unsubscribe(sub *DriftSubscriber) {
	h.unregister <- sub
}

func (h *DriftWSHub) Broadcast(drift *DriftCount) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for sub := range h.subs[drift.TenantID] {
		select {
		case sub.EventChan <- drift:
		default:
			go func(s *DriftSubscriber) { h.unregister <- s }(sub)
		}
	}
}

type DriftWSHandler struct {
	hub *DriftWSHub
}

func NewDriftWSHandler(hub *DriftWSHub) *DriftWSHandler {
	return &DriftWSHandler{hub: hub}
}

func (h *DriftWSHandler) RegisterMuxRoutes(r *mux.Router) {
	r.Handle("/api/v1/ws/drift", h).Methods("GET")
}

func (h *DriftWSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		http.Error(w, "Missing tenant_id parameter", http.StatusBadRequest)
		return
	}

	ws, err := driftUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Drift WS upgrade error: %v", err)
		return
	}
	defer ws.Close()

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	sub := h.hub.Subscribe(tenantID)
	defer h.hub.Unsubscribe(sub)

	ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	ws.SetPongHandler(func(string) error {
		ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		select {
		case drift, ok := <-sub.EventChan:
			if !ok {
				ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
				return
			}
			data, err := json.Marshal(drift)
			if err != nil {
				continue
			}
			if err := ws.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ctx.Done():
			ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			return
		}
	}
}

func BroadcastDriftAlert(hub *DriftWSHub, tenantID, jobID, jobName, severity, description string, issueCount int) {
	hub.Broadcast(&DriftCount{
		TenantID:  tenantID,
		Count:     issueCount,
		LastSeen:  time.Now().Format(time.RFC3339),
		JobID:     jobID,
		Severity:  severity,
		Description: description,
	})
}
