#!/bin/bash

###############################################################################
#                   START BACKEND SERVER ONLY                                #
###############################################################################

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$SCRIPT_DIR/backend"
LOG_DIR="$SCRIPT_DIR/logs"
TIMESTAMP=$(date '+%Y%m%d_%H%M%S')

mkdir -p "$LOG_DIR"

echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  Starting Backend Server${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if port is in use
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}ℹ️  Port 8080 is in use. Killing existing process...${NC}"
    lsof -ti:8080 | xargs kill -9 2>/dev/null || true
    sleep 2
fi

cd "$BACKEND_DIR"

# Ensure cmd/server/main.go exists
if [ ! -f "cmd/server/main.go" ]; then
    echo -e "${RED}❌ Missing cmd/server/main.go - creating...${NC}"
    mkdir -p cmd/server
    cat > cmd/server/main.go << 'EOF'
package main

import (
	"github.com/hondyman/uisce/backend/internal/api"
)

func main() {
	api.StartServer()
}
EOF
    echo -e "${GREEN}✅ Created cmd/server/main.go${NC}"
fi

# Build the backend
echo -e "${YELLOW}Building backend...${NC}"
go build -o server ./cmd/server/main.go

# Load secrets from Infisical if available
if [ -f "$SCRIPT_DIR/.env.infisical" ]; then
    echo -e "${YELLOW}Loading secrets from .env.infisical...${NC}"
    set -a
    source "$SCRIPT_DIR/.env.infisical"
    set +a
fi

# Set defaults if not loaded
# POSTGRES_DSN is used by internal/api/server.go
if [[ -z "${POSTGRES_DSN:-}" ]]; then
  echo "ERROR: POSTGRES_DSN environment variable is not set" >&2
  exit 1
fi
if [[ -z "${JWT_SECRET:-}" ]]; then
  echo "ERROR: JWT_SECRET environment variable is not set" >&2
  exit 1
fi
export TEMPORAL_HOST="${TEMPORAL_HOST:-100.84.50.65:7233}"
export TEMPORAL_RETRY_ATTEMPTS="${TEMPORAL_RETRY_ATTEMPTS:-2}"

echo -e "${YELLOW}Starting server...${NC}"
echo -e "${YELLOW}   POSTGRES_DSN: ${POSTGRES_DSN:0:50}...${NC}"
echo -e "${YELLOW}   TEMPORAL_HOST: $TEMPORAL_HOST${NC}"

# Start server
./server > "$LOG_DIR/backend_${TIMESTAMP}.log" 2>&1 &
BACKEND_PID=$!

sleep 3

if kill -0 $BACKEND_PID 2>/dev/null; then
    echo -e "${GREEN}✅ Backend server started${NC}"
    echo -e "${GREEN}   URL: http://localhost:8080${NC}"
    echo -e "${GREEN}   PID: $BACKEND_PID${NC}"
    echo -e "${GREEN}   Logs: $LOG_DIR/backend_${TIMESTAMP}.log${NC}"
    echo ""
    echo "Press Ctrl+C to stop"
    wait
else
    echo -e "${YELLOW}❌ Backend server failed to start${NC}"
    cat "$LOG_DIR/backend_${TIMESTAMP}.log" | tail -30
    exit 1
fi
