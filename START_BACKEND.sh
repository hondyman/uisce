#!/bin/bash

###############################################################################
#                   START FULL BACKEND ENVIRONMENT                            #
###############################################################################

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend"
LOG_DIR="$SCRIPT_DIR/logs"
TIMESTAMP=$(date '+%Y%m%d_%H%M%S')

# Load environment variables if .env exists
if [ -f "$SCRIPT_DIR/.env" ]; then
    echo -e "Loading .env file..."
    set -a
    source "$SCRIPT_DIR/.env"
    set +a
fi

mkdir -p "$LOG_DIR"

echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Starting Full Backend Environment                             ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if ports are in use
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}ℹ️  Port 8080 (Semantic Rules API) is in use. Killing existing process...${NC}"
    lsof -ti:8080 | xargs kill -9 2>/dev/null || true
    sleep 1
fi

if lsof -Pi :8083 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}ℹ️  Port 8083 (Platform Backend) is in use. Killing existing process...${NC}"
    lsof -ti:8083 | xargs kill -9 2>/dev/null || true
    sleep 1
fi

cd "$BACKEND_DIR"

echo -e "${YELLOW}Building backends...${NC}"
echo "Building Semantic Rules API..."
go build -o rules_server ./cmd/semantic-rules-api/main.go
echo "Building Platform Backend..."
go build -o platform_server ./cmd/server

echo -e "${YELLOW}Starting servers...${NC}"

# Start Semantic Rules API (8080)
./rules_server 2>&1 | tee "$LOG_DIR/rules_backend_${TIMESTAMP}.log" &
RULES_PID=$!

# Start Platform Backend (8083)
# Using default PORT 8083, which is standard for platform_server
PORT=8083 ./platform_server 2>&1 | tee "$LOG_DIR/platform_backend_${TIMESTAMP}.log" &
PLATFORM_PID=$!

sleep 3

if kill -0 $RULES_PID 2>/dev/null; then
    echo -e "${GREEN}✅ Semantic Rules API started${NC}"
    echo -e "${GREEN}   URL: http://localhost:8080${NC}"
    echo -e "${GREEN}   PID: $RULES_PID${NC}"
else
    echo -e "${YELLOW}❌ Semantic Rules API failed to start${NC}"
fi

if kill -0 $PLATFORM_PID 2>/dev/null; then
    echo -e "${GREEN}✅ Platform Backend started${NC}"
    echo -e "${GREEN}   URL: http://localhost:8083${NC}"
    echo -e "${GREEN}   PID: $PLATFORM_PID${NC}"
else
    echo -e "${YELLOW}❌ Platform Backend failed to start${NC}"
fi

echo ""
echo "Press Ctrl+C to stop both servers"

# Trap Ctrl+C to kill both
trap "kill -9 $RULES_PID $PLATFORM_PID 2>/dev/null; echo -e '\n${GREEN}Servers stopped${NC}'; exit 0" INT TERM

wait
