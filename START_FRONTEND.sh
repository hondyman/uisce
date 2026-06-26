#!/bin/bash

###############################################################################
#                   START FRONTEND SERVER ONLY                               #
###############################################################################

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
FRONTEND_DIR="$SCRIPT_DIR/frontend"
LOG_DIR="$SCRIPT_DIR/logs"
TIMESTAMP=$(date '+%Y%m%d_%H%M%S')

mkdir -p "$LOG_DIR"

echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Starting Frontend Server${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if port is in use
if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}ℹ️  Port 3000 is in use. Killing existing process...${NC}"
    lsof -ti:3000 | xargs kill -9 2>/dev/null || true
    sleep 2
fi

cd "$FRONTEND_DIR"

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}Installing dependencies...${NC}"
    npm install
fi

# Start dev server
echo -e "${YELLOW}Starting frontend dev server...${NC}"
npm run dev 2>&1 | tee "$LOG_DIR/frontend_${TIMESTAMP}.log" &
FRONTEND_PID=$!

sleep 3

if kill -0 $FRONTEND_PID 2>/dev/null; then
    echo -e "${GREEN}✅ Frontend server started${NC}"
    echo -e "${GREEN}   URL: http://localhost:3000${NC}"
    echo -e "${GREEN}   PID: $FRONTEND_PID${NC}"
    echo -e "${GREEN}   Logs: $LOG_DIR/frontend_${TIMESTAMP}.log${NC}"
    echo ""
    echo "Press Ctrl+C to stop"
    wait
else
    echo -e "${YELLOW}❌ Frontend server failed to start${NC}"
    cat "$LOG_DIR/frontend_${TIMESTAMP}.log" | tail -20
    exit 1
fi
