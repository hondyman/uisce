#!/bin/bash

echo "🚀 Real-Time WebSocket Integration Demo"
echo "========================================"
echo ""

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go first."
    exit 1
fi

echo "✅ Go is installed"

# Navigate to backend directory
cd /Users/eganpj/GitHub/semlayer/backend

echo "📁 Working directory: $(pwd)"
echo ""

echo "🔧 Building WebSocket server..."
go build -o websocket_server websocket_server.go test_websocket_integration.go

if [ $? -ne 0 ]; then
    echo "❌ Failed to build WebSocket server"
    exit 1
fi

echo "✅ WebSocket server built successfully"

echo ""
echo "🌐 Starting WebSocket server on port 8081..."
echo "   WebSocket endpoint: ws://localhost:8081/ws"
echo "   HTTP trigger endpoint: http://localhost:8081/trigger"
echo ""

# Start server in background
./websocket_server &
SERVER_PID=$!

# Wait a moment for server to start
sleep 2

echo "🎯 Starting WebSocket client..."
echo "   Choose calculations from the interactive menu"
echo ""

# Build and run client
go run websocket_client.go test_websocket_integration.go

# Cleanup
echo ""
echo "🧹 Cleaning up..."
kill $SERVER_PID 2>/dev/null
rm -f websocket_server

echo "✅ Demo completed!"
