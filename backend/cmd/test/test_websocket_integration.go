package main

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gorilla/websocket"
	httpapi "github.com/hondyman/semlayer/backend/internal/api"
)

func testWebSocketIntegration() {
	// Start a simple WebSocket client test
	fmt.Println("🚀 Starting Real-Time WebSocket Integration Test")
	fmt.Println("================================================")

	// Test 1: Financial Calculations with Real-Time Broadcasting
	fmt.Println("\n📊 Test 1: Financial Calculations")
	testFinancialCalculations()

	// Test 2: WebSocket Connection and Real-Time Updates
	fmt.Println("\n🔗 Test 2: WebSocket Real-Time Updates")
	testWebSocketConnection()

	fmt.Println("\n✅ All Real-Time Integration Tests Completed!")
}

// RunWebSocketIntegrationTest runs the WebSocket integration test
func RunWebSocketIntegrationTest() {
	testWebSocketIntegration()
}

func RunWebSocketTest() {
	RunWebSocketIntegrationTest()
}

func testFinancialCalculations() {
	fmt.Println("Running Markowitz Portfolio Optimization...")

	// Create a Markowitz optimization request
	markowitzCalc := httpapi.FinancialCalc{
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

	result, err := httpapi.Dispatch(markowitzCalc, nil)
	if err != nil {
		log.Printf("❌ Markowitz failed: %v", err)
		return
	}

	fmt.Printf("✅ Markowitz Result: %+v\n", result)

	// Test GBM simulation
	fmt.Println("Running GBM Stock Price Simulation...")
	gbmCalc := httpapi.FinancialCalc{
		Type:          "gbm",
		InitialValues: []float64{100},
		DriftRates:    []float64{0.05},
		Volatilities:  []float64{0.2},
		TimeHorizon:   1.0,
		NumSteps:      10,
	}

	result, err = httpapi.Dispatch(gbmCalc, nil)
	if err != nil {
		log.Printf("❌ GBM failed: %v", err)
		return
	}

	fmt.Printf("✅ GBM Result: %+v\n", result)
}

func testWebSocketConnection() {
	fmt.Println("Testing WebSocket connection...")

	// WebSocket dialer
	dialer := websocket.Dialer{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	// Connect to WebSocket endpoint
	conn, _, err := dialer.Dial("ws://localhost:8081/ws", nil)
	if err != nil {
		log.Printf("❌ WebSocket connection failed: %v", err)
		log.Println("💡 Make sure the backend server is running on port 8080")
		return
	}
	defer conn.Close()

	fmt.Println("✅ WebSocket connection established")

	// Send a test message
	testMessage := httpapi.RealTimeMessage{
		Type:      "fund_update",
		Data:      map[string]interface{}{"fund_id": "test-fund", "metric": "irr", "value": 0.15},
		Timestamp: time.Now(),
	}

	messageBytes, _ := json.Marshal(testMessage)
	err = conn.WriteMessage(websocket.TextMessage, messageBytes)
	if err != nil {
		log.Printf("❌ Failed to send message: %v", err)
		return
	}

	fmt.Println("✅ Test message sent")

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Try to read response
	_, response, err := conn.ReadMessage()
	if err != nil {
		log.Printf("❌ Failed to read response: %v", err)
		return
	}

	fmt.Printf("✅ Received response: %s\n", string(response))

	// Test real-time calculation broadcasting
	fmt.Println("Testing real-time calculation broadcasting...")
	testRealTimeBroadcasting(conn)
}

func testRealTimeBroadcasting(conn *websocket.Conn) {
	// Simulate real-time fund updates
	fundUpdates := []map[string]interface{}{
		{"fund_id": "fund-1", "irr": 0.18, "nav": 125.5},
		{"fund_id": "fund-2", "irr": 0.22, "nav": 98.3},
		{"fund_id": "fund-3", "irr": 0.15, "nav": 156.7},
	}

	for i, update := range fundUpdates {
		message := httpapi.RealTimeMessage{
			Type:      "fund_update",
			Data:      update,
			Timestamp: time.Now(),
		}

		messageBytes, _ := json.Marshal(message)
		err := conn.WriteMessage(websocket.TextMessage, messageBytes)
		if err != nil {
			log.Printf("❌ Failed to send fund update %d: %v", i+1, err)
			continue
		}

		fmt.Printf("📡 Broadcasted fund update %d: %s\n", i+1, update["fund_id"])
		time.Sleep(500 * time.Millisecond) // Simulate real-time delay
	}

	fmt.Println("✅ Real-time broadcasting test completed")
}

// simulateRealTimeCalculation demonstrates how calculations can be triggered and broadcasted
// simulateRealTimeCalculation removed; test helper was unused and has been
// replaced by a lightweight placeholder in the notifications test.
