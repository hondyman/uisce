#!/bin/bash

# Semlayer Services Startup Script
# This script ensures clean port usage and starts all services consistently

set -e

echo "🐳 Checking Docker daemon connection..."

# Function to check for Docker daemon
check_docker() {
    if ! docker info >/dev/null 2>&1; then
        echo "⚠️  Docker daemon is not reachable. Trying to set DOCKER_HOST for macOS..."
        # Common Docker Desktop for Mac socket paths
        DOCKER_SOCKET_PATHS=(
            "$HOME/.docker/run/docker.sock"
            "$HOME/Library/Containers/com.docker.docker/Data/backend.sock"
            "/var/run/docker.sock"
        )
        for socket_path in "${DOCKER_SOCKET_PATHS[@]}"; do
            if [ -S "$socket_path" ]; then
                echo "💡 Found Docker socket at $socket_path. Exporting DOCKER_HOST."
                export DOCKER_HOST="unix://$socket_path"
                # Re-check connection with the new DOCKER_HOST
                if docker info >/dev/null 2>&1; then
                    echo "✅ Docker daemon is now reachable."
                    return
                fi
            fi
        done
        echo "❌ Could not connect to Docker daemon. Please ensure Docker Desktop is running."
        exit 1
    fi
    echo "✅ Docker daemon is reachable."
}

# Run the check
check_docker

echo "🧹 Cleaning up ports and stopping existing services..."

# Define required ports
FRONTEND_PORT=5173
DEV_PROXY_PORT=5175
BACKEND_DOCKER_PORT=8080
BACKEND_DEV_PORT=9090
API_GATEWAY_PORT=8001
HASURA_PORT=8081
SWAGGER_PORT=8082

# Kill processes on required ports
PORTS=($FRONTEND_PORT $DEV_PROXY_PORT $BACKEND_DOCKER_PORT $BACKEND_DEV_PORT $API_GATEWAY_PORT $HASURA_PORT $SWAGGER_PORT)

for port in "${PORTS[@]}"; do
    echo "🔍 Checking port $port..."
    # Find listening PIDs for the port
    PIDS=$(lsof -ti:$port -sTCP:LISTEN -n 2>/dev/null || true)
    if [ -n "$PIDS" ]; then
        echo "⚠️  Port $port is in use. Inspecting processes before killing..."
        for pid in $PIDS; do
            # Read process info
            cmd=$(ps -p $pid -o comm= 2>/dev/null | tr -d '[:space:]' || true)
            user=$(ps -p $pid -o user= 2>/dev/null | tr -d '[:space:]' || true)

            # Skip killing Docker Desktop / dockerd related processes or root-owned system services
            if [[ "$cmd" =~ Docker ]] || [[ "$cmd" =~ com.docke ]] || [[ "$cmd" =~ dockerd ]] || [[ "$user" == "root" ]]; then
                echo "🔒 Skipping PID $pid (owner=$user, cmd=$cmd) — likely Docker/system process"
                continue
            fi

            echo "🗡️  Killing PID $pid (owner=$user, cmd=$cmd)"
            kill -9 $pid 2>/dev/null || true
        done
        # Give the system a moment to release the port
        sleep 1
    else
        echo "✅ Port $port is free"
    fi
done

echo "🐳 Stopping existing Docker containers..."
docker compose down --remove-orphans 2>/dev/null || true

# Wait for ports to be fully released
echo "⏱️  Waiting for ports to be released..."
sleep 3

echo "🚀 Starting services..."

echo "📦 Starting Docker services (Hasura, API Gateway, Backend, Swagger)..."
echo "📦 Building backend & gateway images (to reflect local Go changes)..."
# Try a compose build with pull; fall back to simple build if the first fails
docker compose build --pull backend api-gateway || docker compose build backend api-gateway || true

echo "📦 Starting Docker services (Hasura, API Gateway, Backend, Swagger)..."
docker compose up -d graphql-engine api-gateway backend swagger-ui

# Wait for Docker services to be ready
echo "⏱️  Waiting for Docker services to initialize..."
sleep 10

echo "🔍 Verifying Docker services..."
docker compose ps

# 2. Start dev-proxy (forwards frontend API calls to backend/gateway)
echo "🔄 Starting development proxy on port $DEV_PROXY_PORT..."
cd frontend/dev-tools
node dev-proxy.cjs &
DEV_PROXY_PID=$!
cd ../..

# Wait for dev-proxy to start
sleep 2

# 3. Start frontend development server
echo "🌐 Starting frontend development server on port $FRONTEND_PORT..."
cd frontend
npm run dev &
FRONTEND_PID=$!
cd ..

# Wait for frontend to start
sleep 5

echo "✅ All services started successfully!"
echo ""
echo "📋 Services Status:"
echo "🌐 Frontend:      http://localhost:$FRONTEND_PORT"
echo "🔄 Dev Proxy:     http://localhost:$DEV_PROXY_PORT"
echo "🔧 API Gateway:   http://localhost:$API_GATEWAY_PORT"
echo "🏗️  Backend:       http://localhost:$BACKEND_DOCKER_PORT"
echo "📊 Hasura:        http://localhost:$HASURA_PORT"
echo "📚 Swagger:       http://localhost:$SWAGGER_PORT"
echo ""
echo "🎯 Main Application: http://localhost:$FRONTEND_PORT"
echo ""
echo "💡 To stop all services, run: ./scripts/stop-services.sh"
echo "💡 Or press Ctrl+C to stop this script and run: docker compose down"

# Keep script running and handle cleanup on exit
cleanup() {
    echo ""
    echo "🛑 Stopping services..."
    kill $DEV_PROXY_PID 2>/dev/null || true
    kill $FRONTEND_PID 2>/dev/null || true
    docker compose down
    echo "✅ All services stopped"
}

trap cleanup EXIT

# Wait for user to stop
echo "⏸️  Press Ctrl+C to stop all services..."
wait
