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

# Load secrets from Infisical if available, otherwise use defaults
if [ -f "$SCRIPT_DIR/.env.infisical" ]; then
    echo -e "${YELLOW}Loading secrets from .env.infisical...${NC}"
    set -a
    source "$SCRIPT_DIR/.env.infisical"
    set +a

    # Fetch DATABASE_URL from Infisical if not set
    if [ -z "${DATABASE_URL:-}" ] && [ -n "${INFISICAL_TOKEN:-}" ] && [ -n "${INFISICAL_PROJECT_ID:-}" ]; then
        echo -e "${YELLOW}Fetching DATABASE_URL from Infisical...${NC}"
        export DATABASE_URL=$(curl -s -X GET \
            "$INFISICAL_DOMAIN/api/v3/secrets/raw?projectId=$INFISICAL_PROJECT_ID&environment=$INFISICAL_ENVIRONMENT&path=/core" \
            -H "Authorization: Bearer $INFISICAL_TOKEN" 2>/dev/null | \
            python3 -c "import sys,json; print(next(s['secretValue'] for s in json.load(sys.stdin)['secrets'] if s['secretKey']=='DATABASE_URL'))" 2>/dev/null || echo "")
    fi

    if [ -z "${JWT_SECRET:-}" ] && [ -n "${INFISICAL_TOKEN:-}" ] && [ -n "${INFISICAL_PROJECT_ID:-}" ]; then
        export JWT_SECRET=$(curl -s -X GET \
            "$INFISICAL_DOMAIN/api/v3/secrets/raw?projectId=$INFISICAL_PROJECT_ID&environment=$INFISICAL_ENVIRONMENT&path=/core" \
            -H "Authorization: Bearer $INFISICAL_TOKEN" 2>/dev/null | \
            python3 -c "import sys,json; print(next(s['secretValue'] for s in json.load(sys.stdin)['secrets'] if s['secretKey']=='JWT_SECRET'))" 2>/dev/null || echo "")
    fi
fi

# Set defaults if not loaded
export DATABASE_URL="${DATABASE_URL:-postgresql://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable}"
export JWT_SECRET="${JWT_SECRET:-TND5KO7xY/Fz1ifgTR5QMm9T+R5/aPxxavmMzp+hURJxRWTm2Pns+RC+q9NKMxMB3F/R2KAWXnwo7r8N5JIACQ==}"

# Check if main binary exists
if [ ! -f "./main" ]; then
    echo -e "${RED}❌ Backend binary not found at $BACKEND_DIR/main${NC}"
    echo -e "${RED}   Please build the backend first or restore the binary${NC}"
    exit 1
fi

# Start server
echo -e "${YELLOW}Starting server...${NC}"
echo -e "${YELLOW}   DATABASE_URL: ${DATABASE_URL:0:50}...${NC}"
./main > "$LOG_DIR/backend_${TIMESTAMP}.log" 2>&1 &
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
