package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gorilla/websocket"
	httpapi "github.com/hondyman/semlayer/backend/internal/api"
)

type ClientCalculationRequest struct {
	Type   string                 `json:"type"`
	Params map[string]interface{} `json:"params"`
}

func runWebSocketClient() {
	fmt.Println("🎯 Real-Time WebSocket Calculation Client")
	fmt.Println("==========================================")

	// Connect to WebSocket server
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial("ws://localhost:8081/ws", nil)
	if err != nil {
		log.Fatalf("❌ Failed to connect to WebSocket server: %v", err)
	}
	defer conn.Close()

	fmt.Println("✅ Connected to WebSocket server")

	// Start listening for messages
	go listenForMessages(conn)

	// Interactive menu
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Println("\n📋 Available Calculations:")
		fmt.Println("1. Markowitz Portfolio Optimization")
		fmt.Println("2. GBM Stock Price Simulation")
		fmt.Println("3. Efficient Frontier")
		fmt.Println("4. Exit")
		fmt.Print("Choose calculation (1-4): ")

		scanner.Scan()
		choice := strings.TrimSpace(scanner.Text())

		switch choice {
		case "1":
			requestCalculation(conn, "markowitz")
		case "2":
			requestCalculation(conn, "gbm")
		case "3":
			requestCalculation(conn, "efficient_frontier")
		case "4":
			fmt.Println("👋 Goodbye!")
			return
		default:
			fmt.Println("❌ Invalid choice. Please try again.")
		}
	}
}

func requestCalculation(conn *websocket.Conn, calcType string) {
	fmt.Printf("🔄 Requesting %s calculation...\n", calcType)

	req := ClientCalculationRequest{
		Type:   calcType,
		Params: map[string]interface{}{},
	}

	messageBytes, _ := json.Marshal(req)
	err := conn.WriteMessage(websocket.TextMessage, messageBytes)
	if err != nil {
		log.Printf("❌ Failed to send calculation request: %v", err)
		return
	}

	fmt.Printf("✅ %s calculation request sent\n", calcType)
}

func listenForMessages(conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("❌ WebSocket read error: %v", err)
			return
		}

		var msg httpapi.RealTimeMessage
		if err := json.Unmarshal(message, &msg); err != nil {
			log.Printf("❌ Failed to parse message: %v", err)
			continue
		}

		switch msg.Type {
		case "connection_established":
			fmt.Printf("🔗 %s\n", msg.Data.(map[string]interface{})["message"])
		case "calculation_result":
			data := msg.Data.(map[string]interface{})
			calcType := data["type"].(string)
			result := data["result"]
			fmt.Printf("📊 %s calculation result received:\n", calcType)
			fmt.Printf("   Result: %+v\n", result)
		case "calculation_error":
			data := msg.Data.(map[string]interface{})
			calcType := data["type"].(string)
			errorMsg := data["error"].(string)
			fmt.Printf("❌ %s calculation error: %s\n", calcType, errorMsg)
		default:
			fmt.Printf("📨 Unknown message type: %s\n", msg.Type)
		}
	}
}

// RunWebSocketClient starts the interactive WebSocket client
func RunWebSocketClient() {
	runWebSocketClient()
}

func RunWebSocketClientMain() {
	RunWebSocketClient()
}
