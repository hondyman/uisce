#!/bin/bash
# SemLayer Platform - Start Complete System
# Usage: bash RUN_PLATFORM.sh

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║          SemLayer Platform - Complete Startup                  ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if running from correct directory
if [ ! -f "backend/config.yaml" ]; then
    echo -e "${YELLOW}❌ Must be run from semlayer root directory${NC}"
    exit 1
fi

echo -e "${YELLOW}📋 Platform Startup Checklist${NC}"
echo ""

# 1. Check PostgreSQL
echo -n "1. Checking PostgreSQL... "
if command -v psql &> /dev/null; then
    if psql -U postgres -h localhost -d alpha -c "SELECT 1;" &> /dev/null; then
        echo -e "${GREEN}✅${NC}"
    else
        echo -e "${YELLOW}⚠️  PostgreSQL not accessible${NC}"
        echo "   Make sure PostgreSQL is running with database 'alpha'"
        echo "   Try: brew services start postgresql"
        exit 1
    fi
else
    echo -e "${YELLOW}⚠️  psql not found${NC}"
    echo "   PostgreSQL should be running on localhost:5432"
fi

# 2. Check Docker (optional)
echo -n "2. Checking Docker (optional)... "
if command -v docker &> /dev/null && docker ps &> /dev/null; then
    echo -e "${GREEN}✅${NC}"
else
    echo -e "${YELLOW}⚠️  Docker not running (optional)${NC}"
fi

# 3. Check Node.js for frontend
echo -n "3. Checking Node.js... "
if command -v node &> /dev/null; then
    NODE_VERSION=$(node --version)
    echo -e "${GREEN}✅${NC} ($NODE_VERSION)"
else
    echo -e "${YELLOW}⚠️  Node.js not found${NC}"
    exit 1
fi

# 4. Check Go for backend
echo -n "4. Checking Go... "
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    echo -e "${GREEN}✅${NC} ($GO_VERSION)"
else
    echo -e "${YELLOW}⚠️  Go not found${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✅ All prerequisites met!${NC}"
echo ""
echo -e "${BLUE}🚀 Starting Platform Components${NC}"
echo ""

# Create logs directory
mkdir -p logs

# Start backend in background
echo "→ Starting Backend API on port 8080..."
bash START_BACKEND.sh &
BACKEND_PID=$!
sleep 3

# Check if backend started successfully
if ! kill -0 $BACKEND_PID 2>/dev/null; then
    echo -e "${YELLOW}❌ Backend failed to start${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✅ Backend started (PID: $BACKEND_PID)${NC}"
echo "   URL: http://localhost:8080"
echo "   Swagger UI: http://localhost:8080/swagger/index.html"
echo "   Health Check: curl http://localhost:8080/health"
echo ""

# Start frontend in background
echo "→ Starting Frontend Dev Server on port 5173..."
bash START_FRONTEND.sh &
FRONTEND_PID=$!
sleep 5

# Check if frontend started successfully
if ! kill -0 $FRONTEND_PID 2>/dev/null; then
    echo -e "${YELLOW}❌ Frontend failed to start${NC}"
    kill -9 $BACKEND_PID 2>/dev/null || true
    exit 1
fi

echo -e "${GREEN}✅ Frontend started (PID: $FRONTEND_PID)${NC}"
echo "   URL: http://localhost:5173"
echo ""

# Summary
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                   🎉 Platform Ready! 🎉                        ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${GREEN}Frontend:${NC}      http://localhost:5173"
echo -e "${GREEN}Backend API:${NC}   http://localhost:8080"
echo -e "${GREEN}Swagger UI:${NC}    http://localhost:8080/swagger/index.html"
echo ""
echo -e "${YELLOW}Platform PIDs:${NC}"
echo "  Backend:  $BACKEND_PID"
echo "  Frontend: $FRONTEND_PID"
echo ""
echo -e "${YELLOW}Logs:${NC}"
echo "  Backend:  logs/backend_*.log"
echo "  Frontend: logs/frontend_*.log"
echo ""
echo -e "${YELLOW}To stop the platform:${NC}"
echo "  kill -9 $BACKEND_PID  # Stop backend"
echo "  kill -9 $FRONTEND_PID # Stop frontend"
echo "  Or press Ctrl+C in each terminal"
echo ""
echo -e "${YELLOW}Documentation:${NC}"
echo "  Quick Start: PLATFORM_QUICK_START.md"
echo "  Fixes Applied: FIXES_APPLIED_SUMMARY.md"
echo ""

# Wait for Ctrl+C
trap "kill -9 $BACKEND_PID $FRONTEND_PID 2>/dev/null; echo 'Platform stopped'; exit 0" INT TERM

echo -e "${BLUE}Press Ctrl+C to stop the platform${NC}"
echo ""

# Keep script running
wait
