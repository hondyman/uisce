#!/bin/bash

# Semlayer Services Stop Script
# This script stops all services and cleans up ports

set -e

echo "🛑 Stopping all Semlayer services..."

# Define required ports
FRONTEND_PORT=5173
DEV_PROXY_PORT=5175
BACKEND_DOCKER_PORT=8080
BACKEND_DEV_PORT=9090
API_GATEWAY_PORT=8001
HASURA_PORT=8081
SWAGGER_PORT=8082

# Stop Docker services
echo "🐳 Stopping Docker containers..."
docker-compose down --remove-orphans 2>/dev/null || true

# Kill any remaining processes on our ports
PORTS=($FRONTEND_PORT $DEV_PROXY_PORT $BACKEND_DOCKER_PORT $BACKEND_DEV_PORT $API_GATEWAY_PORT $HASURA_PORT $SWAGGER_PORT)

for port in "${PORTS[@]}"; do
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "🔍 Killing processes on port $port..."
        lsof -ti:$port | xargs kill -9 2>/dev/null || true
    fi
done

# Clean up any node/go processes that might be hanging
echo "🧹 Cleaning up remaining processes..."
pkill -f "dev-proxy" 2>/dev/null || true
pkill -f "npm run dev" 2>/dev/null || true
pkill -f "go run cmd/server/main.go" 2>/dev/null || true

echo "✅ All services stopped and ports cleaned up"

# Verify ports are free
echo "🔍 Verifying ports are free..."
for port in "${PORTS[@]}"; do
    if lsof -Pi :$port -sTCP:LISTEN -t >/dev/null 2>&1; then
        echo "⚠️  Port $port is still in use"
    else
        echo "✅ Port $port is free"
    fi
done

echo "🎯 All services stopped successfully!"
