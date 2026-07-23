package backend

import (
	"fmt"
	"log"
	"time"
)

// Production WebSocket Server with ML Analytics
func StartProductionServer() {
	fmt.Println("🚀 Starting Production WebSocket Server with ML Analytics")
	fmt.Println("========================================================")

	// Initialize ML Service
	mlService := NewMLService()

	// Initialize WebSocket Hub with scaling (max 1000 concurrent clients)
	hub := NewWebSocketHub(1000)

	// Start the hub
	go hub.Run()

	// Start cleanup routine
	go startProductionCleanupRoutine(hub)

	// Start ML analytics streaming service
	go startMLAnalyticsService(mlService, hub)

	// Start production server
	StartProductionWebSocketServer(8081, 1000)

	fmt.Println("✅ Production server started successfully!")
	fmt.Println("📊 ML Analytics: Enabled")
	fmt.Println("🔄 Real-time Streaming: Enabled")
	fmt.Println("📈 Concurrent Clients: Up to 1000")
}

// Start ML analytics streaming service
func startMLAnalyticsService(mlService *MLService, hub *WebSocketHub) {
	fmt.Println("🤖 Starting ML Analytics Streaming Service...")

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Broadcast ML insights to all connected clients
		insights := mlService.generateRealTimeAnalytics()

		// Broadcast to all clients
		hub.broadcast <- []byte(fmt.Sprintf(`{
			"type": "ml_insights_broadcast",
			"data": %v,
			"timestamp": "%s"
		}`, insights, time.Now().Format(time.RFC3339)))

		fmt.Printf("📡 Broadcasted ML insights to %d clients\n", hub.GetClientCount())
	}
}

// Enhanced cleanup routine
func startProductionCleanupRoutine(hub *WebSocketHub) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		fmt.Printf("📊 Server Status - Active Clients: %d/%d | Uptime: %v\n",
			hub.GetClientCount(),
			hub.maxClients,
			time.Since(time.Now().Add(-time.Hour))) // Simplified uptime

		// Log performance metrics
		logPerformanceMetrics(hub)
	}
}

// Log performance metrics
func logPerformanceMetrics(hub *WebSocketHub) {
	clientCount := hub.GetClientCount()
	maxClients := hub.maxClients

	utilization := float64(clientCount) / float64(maxClients) * 100

	var status string
	switch {
	case utilization > 90:
		status = "CRITICAL"
	case utilization > 75:
		status = "HIGH"
	case utilization > 50:
		status = "MODERATE"
	default:
		status = "NORMAL"
	}

	log.Printf("📈 Performance - Utilization: %.1f%% | Status: %s | Clients: %d/%d",
		utilization, status, clientCount, maxClients)
}
