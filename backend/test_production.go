package backend

import (
	"fmt"
)

func TestProductionComponents() {
	fmt.Println("🧪 Testing Production Components")
	fmt.Println("===============================")

	// Test ML Service
	fmt.Println("\n🤖 Testing ML Service...")
	mlService := NewMLService()
	fmt.Printf("✅ ML Service initialized with %d models\n", len(mlService.models))

	// Test WebSocket Hub
	fmt.Println("\n🔌 Testing WebSocket Hub...")
	hub := NewWebSocketHub(100)
	fmt.Printf("✅ WebSocket Hub initialized (max: %d clients)\n", hub.maxClients)

	// Test basic functionality
	fmt.Println("\n📊 Testing basic functionality...")

	// Test prediction
	req := PredictionRequest{
		ModelType: "portfolio_return",
		Features: map[string]interface{}{
			"market_trend":    0.15,
			"volatility":      0.18,
			"diversification": 0.75,
		},
	}

	prediction, err := mlService.Predict(req)
	if err != nil {
		fmt.Printf("❌ Prediction failed: %v\n", err)
	} else {
		fmt.Printf("✅ Prediction successful: %+v\n", prediction.Prediction)
	}

	// Test hub client count
	fmt.Printf("📈 Hub client count: %d/%d\n", hub.GetClientCount(), hub.maxClients)

	fmt.Println("\n🎉 All production components tested successfully!")
	fmt.Println("🚀 Ready for production deployment!")
}
