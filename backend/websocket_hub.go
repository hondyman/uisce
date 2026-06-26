package backend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	httpapi "github.com/hondyman/semlayer/backend/internal/api"

	"github.com/hondyman/semlayer/backend/internal/logging"
)

// Production WebSocket Hub for managing multiple concurrent connections
type WebSocketHub struct {
	clients    map[*WebSocketClient]bool
	broadcast  chan []byte
	register   chan *WebSocketClient
	unregister chan *WebSocketClient
	mutex      sync.RWMutex
	maxClients int
}

type WebSocketClient struct {
	conn       *websocket.Conn
	send       chan []byte
	hub        *WebSocketHub
	userID     string
	sessionID  string
	lastActive time.Time
	mutex      sync.Mutex
}

type MLRequest struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	UserID    string                 `json:"user_id,omitempty"`
	SessionID string                 `json:"session_id,omitempty"`
	Priority  int                    `json:"priority,omitempty"` // 1=low, 2=normal, 3=high
}

type MLPrediction struct {
	Type       string                 `json:"type"`
	Prediction map[string]interface{} `json:"prediction"`
	Confidence float64                `json:"confidence"`
	Timestamp  time.Time              `json:"timestamp"`
	Features   map[string]interface{} `json:"features_used"`
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper CORS validation
		return true
	},
}

// NewWebSocketHub creates a new hub for managing WebSocket connections
func NewWebSocketHub(maxClients int) *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*WebSocketClient]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *WebSocketClient),
		unregister: make(chan *WebSocketClient),
		maxClients: maxClients,
	}
}

// Run starts the hub's main loop
func (h *WebSocketHub) Run() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case client := <-h.register:
			h.mutex.Lock()
			if len(h.clients) < h.maxClients {
				h.clients[client] = true
				logging.GetLogger().Sugar().Infow("Client connected", "clients", len(h.clients), "max", h.maxClients)
			} else {
				logging.GetLogger().Sugar().Warnw("Connection rejected: max clients reached", "max", h.maxClients)
				client.conn.Close()
			}
			h.mutex.Unlock()

		case client := <-h.unregister:
			h.mutex.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				logging.GetLogger().Sugar().Infow("Client disconnected", "clients", len(h.clients), "max", h.maxClients)
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

		case <-ticker.C:
			// Clean up inactive connections
			h.cleanupInactiveConnections()
		}
	}
}

// cleanupInactiveConnections removes connections that haven't been active
func (h *WebSocketHub) cleanupInactiveConnections() {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	now := time.Now()
	inactiveCount := 0

	for client := range h.clients {
		client.mutex.Lock()
		if now.Sub(client.lastActive) > 5*time.Minute {
			delete(h.clients, client)
			close(client.send)
			inactiveCount++
		}
		client.mutex.Unlock()
	}

	if inactiveCount > 0 {
		logging.GetLogger().Sugar().Infow("Cleaned up inactive connections", "removed", inactiveCount, "clients", len(h.clients), "max", h.maxClients)
	}
}

// BroadcastToUser sends a message to a specific user
func (h *WebSocketHub) BroadcastToUser(userID string, message []byte) {
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

// GetClientCount returns the current number of connected clients
func (h *WebSocketHub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// NewWebSocketClient creates a new client instance
func NewWebSocketClient(hub *WebSocketHub, conn *websocket.Conn, userID, sessionID string) *WebSocketClient {
	return &WebSocketClient{
		conn:       conn,
		send:       make(chan []byte, 256),
		hub:        hub,
		userID:     userID,
		sessionID:  sessionID,
		lastActive: time.Now(),
	}
}

// WritePump handles sending messages to the client
func (c *WebSocketClient) WritePump() {
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

// ReadPump handles reading messages from the client
func (c *WebSocketClient) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(4096)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		c.mutex.Lock()
		c.lastActive = time.Now()
		c.mutex.Unlock()
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logging.GetLogger().Sugar().Errorw("WebSocket error", "error", err)
			}
			break
		}

		c.mutex.Lock()
		c.lastActive = time.Now()
		c.mutex.Unlock()

		// Process the message
		go c.processMessage(message)
	}
}

// processMessage handles incoming messages from clients
func (c *WebSocketClient) processMessage(message []byte) {
	var req MLRequest
	if err := json.Unmarshal(message, &req); err != nil {
		c.sendError("invalid_request", "Invalid JSON format")
		return
	}

	// Set user/session info if not provided
	if req.UserID == "" {
		req.UserID = c.userID
	}
	if req.SessionID == "" {
		req.SessionID = c.sessionID
	}

	logging.GetLogger().Sugar().Infow("Processing request", "type", req.Type, "user", req.UserID)

	// Route to appropriate handler
	switch req.Type {
	case "calculation":
		c.handleCalculationRequest(req)
	case "ml_prediction":
		c.handleMLPredictionRequest(req)
	case "analytics_stream":
		c.handleAnalyticsStreamRequest(req)
	case "portfolio_analysis":
		c.handlePortfolioAnalysisRequest(req)
	default:
		c.sendError("unknown_type", fmt.Sprintf("Unknown request type: %s", req.Type))
	}
}

// Handle calculation requests
func (c *WebSocketClient) handleCalculationRequest(req MLRequest) {
	calcType, ok := req.Data["type"].(string)
	if !ok {
		c.sendError("calculation", "Missing calculation type")
		return
	}

	var calc httpapi.FinancialCalc
	var result map[string]interface{}

	switch calcType {
	case "markowitz":
		calc = httpapi.FinancialCalc{
			Type: "markowitz",
			Mu:   []float64{0.08, 0.12, 0.10},
			Covariance: [][]float64{
				{0.04, 0.006, 0.004},
				{0.006, 0.09, 0.008},
				{0.004, 0.008, 0.0625},
			},
			LongOnly:     true,
			RiskFreeRate: 0.02,
		}

	case "gbm":
		calc = httpapi.FinancialCalc{
			Type:          "gbm",
			InitialValues: []float64{100},
			DriftRates:    []float64{0.05},
			Volatilities:  []float64{0.2},
			TimeHorizon:   1.0,
			NumSteps:      10,
		}

	case "efficient_frontier":
		calc = httpapi.FinancialCalc{
			Type: "efficient_frontier",
			Mu:   []float64{0.08, 0.12, 0.10},
			Covariance: [][]float64{
				{0.04, 0.006, 0.004},
				{0.006, 0.09, 0.008},
				{0.004, 0.008, 0.0625},
			},
			LongOnly:     true,
			RiskFreeRate: 0.02,
			Points:       5,
		}

	case "excel_formula":
		calc = httpapi.FinancialCalc{
			Type:    "excel_formula",
			Formula: "=XIRR({cash_flows}, {dates})",
			Arguments: map[string]interface{}{
				"cash_flows": []interface{}{-1000.0, 200.0, 300.0, 400.0, 500.0},
				"dates":      []interface{}{1.0, 2.0, 3.0, 4.0, 5.0},
			},
		}

	case "excel_formula_vectorized":
		// Example: Calculate XIRR for multiple portfolios in one batch
		calc = httpapi.FinancialCalc{
			Type:    "excel_formula",
			Formula: "=XIRR({cash_flows}, {dates})",
			Arguments: map[string]interface{}{
				"cash_flows": []interface{}{
					[]interface{}{-1000.0, 200.0, 300.0, 400.0, 500.0},  // Portfolio 1
					[]interface{}{-2000.0, 400.0, 600.0, 800.0, 1000.0}, // Portfolio 2
					[]interface{}{-500.0, 100.0, 150.0, 200.0, 250.0},   // Portfolio 3
				},
				"dates": []interface{}{
					[]interface{}{1.0, 2.0, 3.0, 4.0, 5.0}, // Dates for all portfolios
					[]interface{}{1.0, 2.0, 3.0, 4.0, 5.0},
					[]interface{}{1.0, 2.0, 3.0, 4.0, 5.0},
				},
			},
		}

	default:
		c.sendError("calculation", fmt.Sprintf("Unknown calculation type: %s", calcType))
		return
	}

	resultRaw, err := httpapi.Dispatch(calc, nil)
	if err != nil {
		c.sendError("calculation", err.Error())
		return
	}
	result = resultRaw.(map[string]interface{})

	// Send result
	resultMsg := httpapi.RealTimeMessage{
		Type:      "calculation_result",
		Data:      map[string]interface{}{"type": calcType, "result": result},
		Timestamp: time.Now(),
	}

	resultBytes, _ := json.Marshal(resultMsg)
	c.hub.BroadcastToUser(c.userID, resultBytes)
}

// Handle ML prediction requests
func (c *WebSocketClient) handleMLPredictionRequest(req MLRequest) {
	// Simulate ML prediction (in production, this would call actual ML models)
	prediction := MLPrediction{
		Type: "portfolio_return",
		Prediction: map[string]interface{}{
			"expected_return": 0.12,
			"volatility":      0.18,
			"sharpe_ratio":    0.67,
		},
		Confidence: 0.85,
		Timestamp:  time.Now(),
		Features: map[string]interface{}{
			"market_trend":        "bullish",
			"volatility_index":    0.15,
			"economic_indicators": "positive",
		},
	}

	resultMsg := httpapi.RealTimeMessage{
		Type:      "ml_prediction",
		Data:      prediction,
		Timestamp: time.Now(),
	}

	resultBytes, _ := json.Marshal(resultMsg)
	c.hub.BroadcastToUser(c.userID, resultBytes)
}

// Handle analytics streaming requests
func (c *WebSocketClient) handleAnalyticsStreamRequest(req MLRequest) {
	// Start streaming analytics data
	go c.streamAnalyticsData()
}

// Handle portfolio analysis requests
func (c *WebSocketClient) handlePortfolioAnalysisRequest(req MLRequest) {
	// Simulate comprehensive portfolio analysis
	analysis := map[string]interface{}{
		"risk_metrics": map[string]interface{}{
			"var_95":       0.12,
			"cvar_95":      0.18,
			"max_drawdown": 0.15,
		},
		"performance": map[string]interface{}{
			"total_return":      0.25,
			"annualized_return": 0.08,
			"alpha":             0.03,
			"beta":              1.1,
		},
		"recommendations": []string{
			"Increase allocation to technology sector",
			"Reduce exposure to high-volatility assets",
			"Consider hedging strategies",
		},
	}

	resultMsg := httpapi.RealTimeMessage{
		Type:      "portfolio_analysis",
		Data:      analysis,
		Timestamp: time.Now(),
	}

	resultBytes, _ := json.Marshal(resultMsg)
	c.hub.BroadcastToUser(c.userID, resultBytes)
}

// Stream real-time analytics data
func (c *WebSocketClient) streamAnalyticsData() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for i := 0; i < 30; i++ { // Stream for 60 seconds
		<-ticker.C
		analytics := map[string]interface{}{
			"market_data": map[string]interface{}{
				"sp500":      4200 + float64(i)*10,
				"volatility": 0.15 + float64(i)*0.01,
				"timestamp":  time.Now(),
			},
			"portfolio_metrics": map[string]interface{}{
				"value":     100000 + float64(i)*500,
				"daily_pnl": 250 + float64(i)*25,
				"total_pnl": 15000 + float64(i)*500,
			},
		}

		streamMsg := httpapi.RealTimeMessage{
			Type:      "analytics_update",
			Data:      analytics,
			Timestamp: time.Now(),
		}

		resultBytes, _ := json.Marshal(streamMsg)
		c.hub.BroadcastToUser(c.userID, resultBytes)
	}

	// Send stream end message
	endMsg := httpapi.RealTimeMessage{
		Type:      "analytics_stream_end",
		Data:      map[string]interface{}{"message": "Analytics stream completed"},
		Timestamp: time.Now(),
	}

	endBytes, _ := json.Marshal(endMsg)
	c.hub.BroadcastToUser(c.userID, endBytes)
}

// Send error message to client
func (c *WebSocketClient) sendError(requestType, errorMsg string) {
	errorMessage := httpapi.RealTimeMessage{
		Type:      "error",
		Data:      map[string]interface{}{"error": errorMsg, "type": requestType},
		Timestamp: time.Now(),
	}

	errorBytes, _ := json.Marshal(errorMessage)
	c.hub.BroadcastToUser(c.userID, errorBytes)
}
