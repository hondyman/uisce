#!/bin/bash

###############################################################################
#                   START BACKEND SERVER ONLY                                #
###############################################################################

set -e

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

if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null 2>&1; then
    echo -e "${YELLOW}ℹ️  Port 8080 is in use. Killing existing process...${NC}"
    lsof -ti:8080 | xargs kill -9 2>/dev/null || true
    sleep 2
fi

cd "$BACKEND_DIR"

echo -e "${YELLOW}Deploying pre-built server binary...${NC}"
SERVER_BINARY="${HOME}/uisce-server-fixed"
if [ ! -f "$SERVER_BINARY" ]; then
    echo -e "${RED}❌ Server binary not found at $SERVER_BINARY${NC}"
    echo -e "${RED}   Build on Mac: GOOS=linux GOARCH=amd64 go build -o uisce-server ./cmd/server/main.go${NC}"
    echo -e "${RED}   Then copy to server: scp uisce-server eganpj@100.84.50.65:~/uisce-server-fixed${NC}"
    exit 1
fi
cp "$SERVER_BINARY" ./server
chmod +x ./server

if [ -f "$SCRIPT_DIR/scripts/infisical-bootstrap.sh" ] && command -v infisical &>/dev/null; then
    echo -e "${YELLOW}Bootstrapping secrets from Infisical...${NC}"
    INFISICAL_TOKEN="${INFISICAL_TOKEN:-}" "$SCRIPT_DIR/scripts/infisical-bootstrap.sh" -e dev || true
fi

if [ -f "$SCRIPT_DIR/.env" ]; then
    echo -e "${YELLOW}Loading environment from .env...${NC}"
    set -a
    source "$SCRIPT_DIR/.env"
    set +a
fi
if [ -f "$BACKEND_DIR/.env" ]; then
    echo -e "${YELLOW}Loading environment from backend/.env...${NC}"
    set -a
    source "$BACKEND_DIR/.env"
    set +a
fi
if [ -f "$SCRIPT_DIR/.env.infisical" ]; then
    echo -e "${YELLOW}Loading secrets from .env.infisical...${NC}"
    set -a
    source "$SCRIPT_DIR/.env.infisical"
    set +a
fi

# Declare environment explicitly. This script is local-dev only — production
# deploys invoke the binary directly. Fail-closed production config assertion
# (backend/internal/api/helpers.go::AssertProductionConfig) treats unset
# ENVIRONMENT as production and rejects dev-only flags; defaulting to
# "development" keeps the safe-set behavior aligned with the script's actual
# use case. Operators can still override by setting ENVIRONMENT=local/test in
# their shell.
export ENVIRONMENT="${ENVIRONMENT:-development}"

# Set defaults if not loaded
export POSTGRES_DSN="${POSTGRES_DSN:-${DATABASE_URL:-postgresql://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable}}"
export DATABASE_URL="${DATABASE_URL:-$POSTGRES_DSN}"
: "${JWT_SECRET:?JWT_SECRET not set — refusing to start with the test-secret default that is in git history}"
export JWT_SECRET
export PORT="${PORT:-8080}"
export TEMPORAL_HOST="${TEMPORAL_HOST:-100.84.50.65:7233}"
export TEMPORAL_RETRY_ATTEMPTS="${TEMPORAL_RETRY_ATTEMPTS:-2}"
export FIX_ENABLE="${FIX_ENABLE:-true}"
export FIX_DEMO_AGENT="${FIX_DEMO_AGENT:-true}"
export FIX_ADMIN_ADDR="${FIX_ADMIN_ADDR:-127.0.0.1:8981}"
export FIX_ADMIN_TOKEN="${FIX_ADMIN_TOKEN:-dev-fix-admin}"
export FIX_ACCEPTOR_PORT="${FIX_ACCEPTOR_PORT:-8980}"
export API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK="${API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK:-true}"
: "${API_TOKEN_ENCRYPTION_KEY:?API_TOKEN_ENCRYPTION_KEY not set — refusing to fall back to a value that is in git history (origin/main:de336a41af)}"

echo -e "${YELLOW}Starting server...${NC}"
echo -e "${YELLOW}   POSTGRES_DSN: ${POSTGRES_DSN:0:50}...${NC}"
echo -e "${YELLOW}   TEMPORAL_HOST: $TEMPORAL_HOST${NC}"

nohup ./server > "$LOG_DIR/backend_${TIMESTAMP}.log" 2>&1 & disown
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
