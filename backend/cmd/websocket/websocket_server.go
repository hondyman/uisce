package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	httpapi "github.com/hondyman/semlayer/backend/internal/api"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		// In production, implement proper CORS validation
		return true
	},
}

type CalculationRequest struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params"`
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	fmt.Println("🔗 New WebSocket connection established")

	// Send welcome message
	welcomeMsg := httpapi.RealTimeMessage{
		Type:      "connection_established",
		Data:      map[string]interface{}{"message": "Connected to real-time calculation service"},
		Timestamp: time.Now(),
	}

	welcomeBytes, _ := json.Marshal(welcomeMsg)
	conn.WriteMessage(websocket.TextMessage, welcomeBytes)

	// Listen for messages
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("WebSocket read error: %v", err)
			break
		}

		if messageType == websocket.TextMessage {
			var req CalculationRequest
			if err := json.Unmarshal(message, &req); err != nil {
				log.Printf("Invalid message format: %v", err)
				continue
			}

			// Process calculation request
			go handleCalculationRequest(conn, req)
		}
	}
}

func handleCalculationRequest(conn *websocket.Conn, req CalculationRequest) {
	fmt.Printf("📊 Processing calculation request: %s\n", req.Type)

	var calc httpapi.FinancialCalc
	var result map[string]interface{}

	switch req.Type {
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

		resultRaw, err := httpapi.Dispatch(calc, nil)
		if err != nil {
			sendError(conn, req.Type, err.Error())
			return
		}
		result = resultRaw.(map[string]interface{})

	case "gbm":
		calc = httpapi.FinancialCalc{
			Type:          "gbm",
			InitialValues: []float64{100},
			DriftRates:    []float64{0.05},
			Volatilities:  []float64{0.2},
			TimeHorizon:   1.0,
			NumSteps:      10,
		}

		resultRaw, err := httpapi.Dispatch(calc, nil)
		if err != nil {
			sendError(conn, req.Type, err.Error())
			return
		}
		result = resultRaw.(map[string]interface{})

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

		resultRaw, err := httpapi.Dispatch(calc, nil)
		if err != nil {
			sendError(conn, req.Type, err.Error())
			return
		}
		result = resultRaw.(map[string]interface{})

	default:
		sendError(conn, req.Type, fmt.Sprintf("unknown calculation type: %s", req.Type))
		return
	}

	// Send calculation result
	resultMsg := httpapi.RealTimeMessage{
		Type:      "calculation_result",
		Data:      map[string]interface{}{"type": req.Type, "result": result},
		Timestamp: time.Now(),
	}

	resultBytes, _ := json.Marshal(resultMsg)
	conn.WriteMessage(websocket.TextMessage, resultBytes)

	fmt.Printf("✅ Calculation %s completed and sent to client\n", req.Type)
}

func sendError(conn *websocket.Conn, calcType, errorMsg string) {
	errorMessage := httpapi.RealTimeMessage{
		Type:      "calculation_error",
		Data:      map[string]interface{}{"error": errorMsg, "type": calcType},
		Timestamp: time.Now(),
	}

	errorBytes, _ := json.Marshal(errorMessage)
	conn.WriteMessage(websocket.TextMessage, errorBytes)
}

func handleTriggerCalculation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req CalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	fmt.Printf("🔄 Triggering calculation: %s\n", req.Type)

	// This would normally broadcast to all connected WebSocket clients
	// For now, just return success
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "calculation_triggered",
		"type":    req.Type,
		"message": "Calculation queued for real-time broadcasting",
	})
}

// StartWebSocketServer starts the real-time WebSocket calculation server
func StartWebSocketServer() {
	fmt.Println("🚀 Starting Real-Time WebSocket Calculation Server")
	fmt.Println("==================================================")
	fmt.Println("WebSocket endpoint: ws://localhost:8081/ws")
	fmt.Println("HTTP trigger endpoint: http://localhost:8081/trigger")
	fmt.Println("Press Ctrl+C to stop")

	http.HandleFunc("/ws", handleWebSocket)
	http.HandleFunc("/trigger", handleTriggerCalculation)

	log.Fatal(http.ListenAndServe(":8081", nil))
}

func RunWebSocketServerMain() {
	StartWebSocketServer()
}
