#!/bin/bash

echo "🚀 Production WebSocket Server with ML Analytics Demo"
echo "===================================================="
echo ""

# Build all production components
echo "📦 Building production components..."
go build websocket_hub.go
go build ml_service.go
go build production_websocket_server.go
go build production_server.go
echo "✅ Build complete"
echo ""

# Start the production server in background
echo "🔧 Starting Production WebSocket Server..."
./production_server &
SERVER_PID=$!
echo "✅ Production server started (PID: $SERVER_PID)"
echo ""

# Wait for server to start
sleep 3

echo "🎯 Production Server Features:"
echo "  📊 ML Analytics: Real-time predictions and insights"
echo "  🔄 Streaming: Live market data and analytics"
echo "  📈 Scaling: Up to 1000 concurrent WebSocket clients"
echo "  🏥 Health: Health check endpoint at /health"
echo "  📋 Metrics: Performance metrics at /metrics"
echo ""

echo "🌐 WebSocket Endpoints:"
echo "  ws://localhost:8081/ws - Main WebSocket connection"
echo "  http://localhost:8081/health - Health check"
echo "  http://localhost:8081/metrics - Server metrics"
echo ""

echo "🤖 ML Models Available:"
echo "  - Portfolio Return Predictor"
echo "  - Risk Assessment Model"
echo "  - Market Sentiment Analyzer"
echo "  - Volatility Forecasting Model"
echo ""

echo "📡 Real-time Features:"
echo "  - Live calculation requests/responses"
echo "  - ML-powered predictions"
echo "  - Analytics streaming"
echo "  - Portfolio analysis"
echo ""

# Test health endpoint
echo "🏥 Testing health endpoint..."
curl -s http://localhost:8081/health | head -10
echo ""
echo ""

# Test metrics endpoint
echo "📊 Testing metrics endpoint..."
curl -s http://localhost:8081/metrics | head -15
echo ""
echo ""

echo "🎮 To test the WebSocket connection:"
echo "1. Open another terminal"
echo "2. Run: go run websocket_client.go"
echo "3. Request calculations and see real-time ML predictions!"
echo ""

echo "⏹️  Press Ctrl+C to stop the server"
echo ""

# Wait for user interrupt
trap "echo ''; echo '🧹 Cleaning up...'; kill $SERVER_PID 2>/dev/null; echo '✅ Demo complete!'" INT
wait $SERVER_PID
