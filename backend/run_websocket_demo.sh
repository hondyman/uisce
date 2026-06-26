#!/bin/bash

echo "🚀 Real-Time WebSocket Calculation Demo"
echo "========================================"
echo ""

# Build all components
echo "📦 Building WebSocket components..."
go build websocket_server.go
go build websocket_client.go
go build test_websocket_integration.go
echo "✅ Build complete"
echo ""

# Start the WebSocket server in background
echo "🔧 Starting WebSocket server..."
./websocket_server &
SERVER_PID=$!
echo "✅ Server started (PID: $SERVER_PID)"
echo ""

# Wait a moment for server to start
sleep 2

# Run integration tests
echo "🧪 Running WebSocket integration tests..."
./test_websocket_integration
echo ""

# Start the interactive client
echo "🎯 Starting interactive WebSocket client..."
echo "Use the client to request real-time calculations"
echo "Press Ctrl+C to exit"
echo ""

./websocket_client

# Cleanup
echo ""
echo "🧹 Cleaning up..."
kill $SERVER_PID 2>/dev/null
echo "✅ Demo complete!"
