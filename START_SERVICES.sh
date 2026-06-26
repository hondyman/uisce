#!/bin/bash

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  SemLayer Full Stack Startup${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Kill any existing processes
echo -e "${YELLOW}Cleaning up existing processes...${NC}"
killall -9 server vite node 2>/dev/null || true
sleep 2

# Start backend
echo -e "${YELLOW}Starting Backend (port 8080)...${NC}"
cd "$SCRIPT_DIR/backend"
PORT=8080 ./server > "$SCRIPT_DIR/logs/backend.log" 2>&1 &
BACKEND_PID=$!
echo -e "${GREEN}✅ Backend started (PID: $BACKEND_PID)${NC}"
sleep 3

# Start frontend
echo -e "${YELLOW}Starting Frontend (port 5173)...${NC}"
cd "$SCRIPT_DIR/frontend"
npx vite --host 0.0.0.0 > "$SCRIPT_DIR/logs/frontend.log" 2>&1 &
FRONTEND_PID=$!
echo -e "${GREEN}✅ Frontend started (PID: $FRONTEND_PID)${NC}"
sleep 4

echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✅ All services started successfully!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════════${NC}"
echo ""
echo "📱 Frontend:  http://localhost:5173"
echo "🔌 Backend:   http://localhost:8080"
echo ""
echo "📋 Logs:"
echo "   Backend:  $SCRIPT_DIR/logs/backend.log"
echo "   Frontend: $SCRIPT_DIR/logs/frontend.log"
echo ""
echo "To stop services: killall backend vite"
echo ""

# Keep script running
wait
