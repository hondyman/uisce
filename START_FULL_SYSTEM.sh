#!/bin/bash

###############################################################################
#                                                                             #
#                    START FULL SEMLAYER SYSTEM                              #
#                                                                             #
#  This script starts all components of the Semlayer system:                 #
#  - PostgreSQL Database (assumes already running)                           #
#  - Backend API Server (Go)                                                 #
#  - Frontend Development Server (React + TypeScript)                        #
#                                                                             #
#  Usage: bash START_FULL_SYSTEM.sh                                          #
#                                                                             #
###############################################################################

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend"
FRONTEND_DIR="$SCRIPT_DIR/frontend"
LOG_DIR="$SCRIPT_DIR/logs"
TIMESTAMP=$(date '+%Y%m%d_%H%M%S')
RULE_ENGINE_PORT=8084

# Create logs directory
mkdir -p "$LOG_DIR"

# Function to print section headers
print_header() {
    echo ""
    echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║  $1${NC}"
    echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
}

# Function to print success messages
print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

# Function to print error messages
print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Function to print info messages
print_info() {
    echo -e "${YELLOW}ℹ️  $1${NC}"
}

# Function to check if a port is in use
is_port_in_use() {
    lsof -Pi :$1 -sTCP:LISTEN -t >/dev/null 2>&1
    return $?
}

# Function to cleanup on exit
cleanup() {
    echo ""
    echo -e "${YELLOW}Cleaning up...${NC}"
    # Kill background jobs started by this script
    jobs -p | xargs -r kill 2>/dev/null || true
    # Kill any recorded PIDs for started services
    [ -f "$BACKEND_DIR/.backend.pid" ] && xargs -r kill 2>/dev/null < "$BACKEND_DIR/.backend.pid" || true
    [ -f "$BACKEND_DIR/.rule-engine.pid" ] && xargs -r kill 2>/dev/null < "$BACKEND_DIR/.rule-engine.pid" || true
    [ -f "$FRONTEND_DIR/.frontend.pid" ] && xargs -r kill 2>/dev/null < "$FRONTEND_DIR/.frontend.pid" || true
    print_info "System shutdown complete"
}

trap cleanup EXIT

###############################################################################
#                              START CHECKS
###############################################################################

print_header "SYSTEM STARTUP - Checking Prerequisites"

# Check if we're in the right directory
if [ ! -f "$SCRIPT_DIR/config.yaml" ]; then
    print_error "config.yaml not found. Are you in the semlayer root directory?"
    exit 1
fi
print_success "Found project root directory"

# Check if backend exists
if [ ! -d "$BACKEND_DIR" ]; then
    print_error "Backend directory not found at $BACKEND_DIR"
    exit 1
fi
print_success "Found backend directory"

# Check if frontend exists
if [ ! -d "$FRONTEND_DIR" ]; then
    print_error "Frontend directory not found at $FRONTEND_DIR"
    exit 1
fi
print_success "Found frontend directory"

# Check Go installation
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.20+"
    exit 1
fi
GO_VERSION=$(go version | awk '{print $3}')
print_success "Found Go: $GO_VERSION"

# Check Node.js installation
if ! command -v node &> /dev/null; then
    print_error "Node.js is not installed. Please install Node.js 16+"
    exit 1
fi
NODE_VERSION=$(node --version)
print_success "Found Node.js: $NODE_VERSION"

# Check npm installation
if ! command -v npm &> /dev/null; then
    print_error "npm is not installed. Please install npm"
    exit 1
fi
NPM_VERSION=$(npm --version)
print_success "Found npm: $NPM_VERSION"

# Check PostgreSQL
if ! command -v psql &> /dev/null; then
    print_error "PostgreSQL is not installed. Please install PostgreSQL 12+"
    exit 1
fi
PG_VERSION=$(psql --version | awk '{print $3}')
print_success "Found PostgreSQL: $PG_VERSION"

# Check if PostgreSQL is running
if ! psql -U postgres -d postgres -c "SELECT 1" >/dev/null 2>&1; then
    print_error "PostgreSQL is not running. Please start PostgreSQL first"
    exit 1
fi
print_success "PostgreSQL is running and accessible"

###############################################################################
#                         CHECK PORT AVAILABILITY
###############################################################################

print_header "Checking Port Availability"

# Check port 8080 (Backend)
if is_port_in_use 8080; then
    print_info "Port 8080 (Backend) is in use. Killing existing process..."
    lsof -ti:8080 | xargs kill -9 2>/dev/null || true
    sleep 2
fi
print_success "Port 8080 (Backend) is available"

# Check rule-engine port (used by lightweight rule service)
if is_port_in_use $RULE_ENGINE_PORT; then
    print_info "Port $RULE_ENGINE_PORT (rule-engine-service) is in use. Killing existing process..."
    lsof -ti:$RULE_ENGINE_PORT | xargs kill -9 2>/dev/null || true
    sleep 2
fi
print_success "Port $RULE_ENGINE_PORT (rule-engine-service) is available"

# Check port 5173 (Frontend)
if is_port_in_use 5173; then
    print_info "Port 5173 (Frontend) is in use. Killing existing process..."
    lsof -ti:5173 | xargs kill -9 2>/dev/null || true
    sleep 2
fi
print_success "Port 5173 (Frontend) is available"

###############################################################################
#                         DOCKER / HASURA SETUP
###############################################################################

print_header "Docker & Hasura"

# Check Docker
if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker Desktop or the Docker engine."
        exit 1
fi
print_success "Found Docker"

if ! docker info >/dev/null 2>&1; then
        print_error "Docker daemon does not appear to be running. Start Docker and retry."
        exit 1
fi
print_success "Docker daemon running"

print_info "Starting required Docker services: graphql-engine, rabbitmq, api-gateway (event-router optional)"
docker compose up -d graphql-engine rabbitmq api-gateway || true
# event-router is optional; it may fail if port 8081 is already in use (e.g., by temporal-ui)
docker compose up -d event-router 2>&1 | grep -i "failed\|error" || print_info "event-router started or skipped (port conflict)"

print_info "Waiting for Hasura (http://localhost:8083/healthz) to be healthy..."
HASURA_HEALTH_URL="http://localhost:8083/healthz"
HASURA_WAIT_TIMEOUT=${HASURA_WAIT_TIMEOUT:-60}
end=$((SECONDS + HASURA_WAIT_TIMEOUT))
while :; do
    if curl -fsS "$HASURA_HEALTH_URL" >/dev/null 2>&1; then
        print_success "Hasura is healthy"
        break
    fi
    if [ $SECONDS -ge $end ]; then
        print_error "Timed out waiting for Hasura after ${HASURA_WAIT_TIMEOUT}s"
        break
    fi
    sleep 1
done

# Apply Hasura metadata/migrations if hasura CLI is available and hasura folder exists
# This is optional; if it fails, the system will still work but Hasura may not have tracked tables
if command -v hasura >/dev/null 2>&1 && [ -d "$SCRIPT_DIR/hasura" ]; then
  print_info "Attempting to apply Hasura metadata and migrations using hasura CLI (optional step)"
  (cd "$SCRIPT_DIR/hasura" && \
    HASURA_GRAPHQL_ENDPOINT=http://localhost:8083 \
    HASURA_GRAPHQL_ADMIN_SECRET=${HASURA_ADMIN_SECRET:-newadminsecretkey} \
    hasura metadata apply --endpoint http://localhost:8083 --admin-secret ${HASURA_ADMIN_SECRET:-newadminsecretkey} 2>&1 | tail -5) || print_info "Hasura metadata apply skipped or failed (optional; system will continue)"
  (cd "$SCRIPT_DIR/hasura" && \
    HASURA_GRAPHQL_ENDPOINT=http://localhost:8083 \
    HASURA_GRAPHQL_ADMIN_SECRET=${HASURA_ADMIN_SECRET:-newadminsecretkey} \
    hasura migrate apply --all-databases --endpoint http://localhost:8083 --admin-secret ${HASURA_ADMIN_SECRET:-newadminsecretkey} 2>&1 | tail -5) || print_info "Hasura migrations skipped or failed (optional; system will continue)"
else
  print_info "hasura CLI not found or hasura dir missing; skipping metadata/migrations (optional step). Run install_hasura_cli.sh if needed."
fi
###############################################################################
#                         BUILD BACKEND
###############################################################################

print_header "Building Backend"

cd "$BACKEND_DIR"

if [ ! -f "go.mod" ]; then
    print_error "go.mod not found in backend directory"
    exit 1
fi

print_info "Building server executable..."
if go build -o server cmd/server/main.go 2>&1 | tee "$LOG_DIR/backend_build_${TIMESTAMP}.log"; then
    print_success "Backend built successfully"
else
    print_error "Backend build failed. Check logs: $LOG_DIR/backend_build_${TIMESTAMP}.log"
    exit 1
fi

if [ ! -f "server" ]; then
    print_error "Server executable not found after build"
    exit 1
fi
print_success "Server executable ready"

###############################################################################
#                         BUILD RULE-ENGINE SERVICE
###############################################################################

print_header "Building Rule Engine Service"

if go build -o rule-engine-service cmd/rule-engine-service/main.go 2>&1 | tee "$LOG_DIR/rule_engine_build_${TIMESTAMP}.log"; then
    print_success "rule-engine-service built successfully"
else
    print_error "rule-engine-service build failed. Check logs: $LOG_DIR/rule_engine_build_${TIMESTAMP}.log"
    exit 1
fi


###############################################################################
#                         INSTALL FRONTEND DEPENDENCIES
###############################################################################

print_header "Preparing Frontend"

cd "$FRONTEND_DIR"

if [ ! -f "package.json" ]; then
    print_error "package.json not found in frontend directory"
    exit 1
fi

if [ ! -d "node_modules" ]; then
    print_info "Installing frontend dependencies (this may take a few minutes)..."
    if npm install 2>&1 | tee "$LOG_DIR/frontend_install_${TIMESTAMP}.log"; then
        print_success "Frontend dependencies installed"
    else
        print_error "Frontend dependency installation failed"
        exit 1
    fi
else
    print_info "Frontend dependencies already installed"
fi

###############################################################################
#                         START BACKEND SERVER
###############################################################################

print_header "Starting Backend Server"

cd "$BACKEND_DIR"

# Ensure any stale backend process is cleaned up before starting a new one.
# Prefer killing a recorded PID file, then fallback to killing whatever is
# listening on the configured backend port (8080) to avoid bind conflicts.
if [ -f ".backend.pid" ]; then
    OLD_PID=$(cat ".backend.pid" 2>/dev/null || echo "")
    if [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" >/dev/null 2>&1; then
        print_info "Killing stale backend process (PID: $OLD_PID)"
        kill -9 "$OLD_PID" >/dev/null 2>&1 || true
        sleep 1
    fi
    rm -f ".backend.pid" >/dev/null 2>&1 || true
fi

# If any process is listening on port 8080, kill it to avoid bind errors.
if lsof -ti:8080 >/dev/null 2>&1; then
    print_info "Killing any process currently listening on port 8080"
    lsof -ti:8080 | xargs -r kill -9 2>/dev/null || true
    sleep 1
fi

print_info "Starting API server on port 8080..."
BACKEND_HELPER="$SCRIPT_DIR/scripts/start-backend-local.sh"

if [ -x "$BACKEND_HELPER" ]; then
    print_info "Starting backend via helper script: $BACKEND_HELPER"
    # helper writes its own logs under logs/ and the pid to .backend.pid
    PORT=8080 "$BACKEND_HELPER" > "$LOG_DIR/backend_start_helper_${TIMESTAMP}.log" 2>&1 &
    # Wait for helper to create .backend.pid (timeout after 15s)
    for i in $(seq 1 15); do
        if [ -f ".backend.pid" ]; then
            break
        fi
        sleep 1
    done
    BACKEND_PID=$(cat ".backend.pid" 2>/dev/null || echo "")
    
    # Use health check instead of just PID check - backend might still be initializing
    # Wait up to 20 seconds for backend to be responsive
    HEALTH_CHECK_ATTEMPTS=0
    MAX_HEALTH_ATTEMPTS=20
    while [ $HEALTH_CHECK_ATTEMPTS -lt $MAX_HEALTH_ATTEMPTS ]; do
        if curl -s "http://localhost:8080/swagger/index.html" > /dev/null 2>&1; then
            print_success "Backend server started (PID: $BACKEND_PID)"
            print_info "Backend URL: http://localhost:8080"
            break
        fi
        HEALTH_CHECK_ATTEMPTS=$((HEALTH_CHECK_ATTEMPTS + 1))
        if [ $HEALTH_CHECK_ATTEMPTS -lt $MAX_HEALTH_ATTEMPTS ]; then
            sleep 1
        fi
    done
    
    if [ $HEALTH_CHECK_ATTEMPTS -eq $MAX_HEALTH_ATTEMPTS ]; then
        print_error "Backend health check failed (no response after 20s). Check logs: $LOG_DIR/backend_start_helper_${TIMESTAMP}.log"
        if [ -f "$LOG_DIR/backend_start_helper_${TIMESTAMP}.log" ]; then
            print_error "Helper script logs:"
            tail -n 40 "$LOG_DIR/backend_start_helper_${TIMESTAMP}.log"
        fi
        if [ -n "$BACKEND_PID" ]; then
            # Check if actual backend log exists
            BACKEND_LOG=$(ls -t "$LOG_DIR"/backend_*.log 2>/dev/null | head -1)
            if [ -n "$BACKEND_LOG" ]; then
                print_error "Backend logs:"
                tail -n 40 "$BACKEND_LOG"
            fi
        fi
        exit 1
    fi
else
    print_info "Starting API server on port 8080..."
    ./server > "$LOG_DIR/backend_${TIMESTAMP}.log" 2>&1 &
    BACKEND_PID=$!
    echo $BACKEND_PID > ".backend.pid"

    # Wait for backend to start and become responsive
    HEALTH_CHECK_ATTEMPTS=0
    MAX_HEALTH_ATTEMPTS=20
    while [ $HEALTH_CHECK_ATTEMPTS -lt $MAX_HEALTH_ATTEMPTS ]; do
        if curl -s "http://localhost:8080/swagger/index.html" > /dev/null 2>&1; then
            print_success "Backend server started (PID: $BACKEND_PID)"
            print_info "Backend URL: http://localhost:8080"
            break
        fi
        HEALTH_CHECK_ATTEMPTS=$((HEALTH_CHECK_ATTEMPTS + 1))
        if [ $HEALTH_CHECK_ATTEMPTS -lt $MAX_HEALTH_ATTEMPTS ]; then
            sleep 1
        fi
    done
    
    if [ $HEALTH_CHECK_ATTEMPTS -eq $MAX_HEALTH_ATTEMPTS ]; then
        print_error "Backend server failed to become responsive. Check logs: $LOG_DIR/backend_${TIMESTAMP}.log"
        cat "$LOG_DIR/backend_${TIMESTAMP}.log" | tail -20
        exit 1
    fi
fi

###############################################################################
#                         START RULE-ENGINE SERVICE
###############################################################################

print_header "Starting Rule-Engine Service"

cd "$BACKEND_DIR"

# Ensure any stale rule-engine process is cleaned up
if [ -f ".rule-engine.pid" ]; then
    OLD_PID=$(cat ".rule-engine.pid" 2>/dev/null || echo "")
    if [ -n "$OLD_PID" ] && kill -0 "$OLD_PID" >/dev/null 2>&1; then
        print_info "Killing stale rule-engine process (PID: $OLD_PID)"
        kill -9 "$OLD_PID" >/dev/null 2>&1 || true
        sleep 1
    fi
    rm -f ".rule-engine.pid" >/dev/null 2>&1 || true
fi

# Start rule-engine-service on reserved port
print_info "Starting rule-engine-service on port $RULE_ENGINE_PORT..."
PORT=$RULE_ENGINE_PORT ./rule-engine-service > "$LOG_DIR/rule_engine_${TIMESTAMP}.log" 2>&1 &
RULE_ENGINE_PID=$!
echo $RULE_ENGINE_PID > ".rule-engine.pid"

# Wait for rule-engine to be healthy
HEALTH_CHECK_ATTEMPTS=0
MAX_HEALTH_ATTEMPTS=15
while [ $HEALTH_CHECK_ATTEMPTS -lt $MAX_HEALTH_ATTEMPTS ]; do
    if curl -s "http://localhost:$RULE_ENGINE_PORT/health" > /dev/null 2>&1; then
        print_success "rule-engine-service started (PID: $RULE_ENGINE_PID)"
        print_info "rule-engine URL: http://localhost:$RULE_ENGINE_PORT"
        break
    fi
    HEALTH_CHECK_ATTEMPTS=$((HEALTH_CHECK_ATTEMPTS + 1))
    sleep 1
done

if [ $HEALTH_CHECK_ATTEMPTS -eq $MAX_HEALTH_ATTEMPTS ]; then
    print_error "rule-engine-service failed to become responsive. Check logs: $LOG_DIR/rule_engine_${TIMESTAMP}.log"
    tail -n 40 "$LOG_DIR/rule_engine_${TIMESTAMP}.log" || true
    # Not fatal — continue, but warn
    print_info "Continuing startup despite rule-engine health check failure"
fi


###############################################################################
#                         START FRONTEND SERVER
###############################################################################

print_header "Starting Frontend Server"

cd "$FRONTEND_DIR"

print_info "Starting development server on port 5173..."
"$SCRIPT_DIR/scripts/start-frontend.sh" > "$LOG_DIR/frontend_${TIMESTAMP}.log" 2>&1 &
FRONTEND_PID=$!
echo $FRONTEND_PID > ".frontend.pid"

# Wait for frontend to start
sleep 5

if kill -0 $FRONTEND_PID 2>/dev/null; then
    print_success "Frontend server started (PID: $FRONTEND_PID)"
    print_info "Frontend URL: http://localhost:5173"
else
    print_error "Frontend server failed to start. Check logs: $LOG_DIR/frontend_${TIMESTAMP}.log"
    cat "$LOG_DIR/frontend_${TIMESTAMP}.log" | tail -20
    exit 1
fi

###############################################################################
#                         SYSTEM RUNNING
###############################################################################

print_header "🎉 SYSTEM RUNNING - ALL SERVICES ACTIVE"

echo ""
echo -e "${GREEN}Backend API:${NC}"
echo "  URL: http://localhost:8080"
echo "  PID: $BACKEND_PID"
echo "  Logs: $LOG_DIR/backend_${TIMESTAMP}.log"
echo ""

echo -e "${GREEN}Frontend UI:${NC}"
echo "  URL: http://localhost:5173"
echo "  PID: $FRONTEND_PID"
echo "  Logs: $LOG_DIR/frontend_${TIMESTAMP}.log"
echo ""

echo -e "${GREEN}Database:${NC}"
echo "  Host: localhost:5432"
echo "  Database: alpha"
echo "  Connection: postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable"
echo ""

echo -e "${YELLOW}Quick Start:${NC}"
echo "  1. Open browser: http://localhost:5173"
echo "  2. Navigate to: Config → Dynamic UI Generator"
echo "  3. Fill employee form and click Save"
echo "  4. Check DevTools Network tab for 201 response"
echo ""

echo -e "${BLUE}To stop the system:${NC}"
echo "  Press Ctrl+C to shutdown all services"
echo ""

print_header "Waiting for services..."

# Wait for both processes to stay alive
wait

