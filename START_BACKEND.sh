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

# Build the backend (cmd/server)
echo -e "${YELLOW}Building backend...${NC}"
go build -o server ./cmd/server/main.go

# Bootstrap secrets from Infisical if bootstrap script exists
if [ -f "$SCRIPT_DIR/scripts/infisical-bootstrap.sh" ] && command -v infisical &>/dev/null; then
    echo -e "${YELLOW}Bootstrapping secrets from Infisical...${NC}"
    INFISICAL_TOKEN="${INFISICAL_TOKEN:-}" "$SCRIPT_DIR/scripts/infisical-bootstrap.sh" -e dev || true
fi

# Load secrets from root .env, backend/.env, or .env.infisical if available
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
export JWT_SECRET="${JWT_SECRET:-test-secret}"
export PORT="${PORT:-8080}"
export TEMPORAL_HOST="${TEMPORAL_HOST:-100.84.50.65:7233}"
export TEMPORAL_RETRY_ATTEMPTS="${TEMPORAL_RETRY_ATTEMPTS:-2}"
export FIX_ENABLE="${FIX_ENABLE:-true}"
export FIX_DEMO_AGENT="${FIX_DEMO_AGENT:-true}"
export FIX_ADMIN_ADDR="${FIX_ADMIN_ADDR:-127.0.0.1:8981}"
export FIX_ADMIN_TOKEN="${FIX_ADMIN_TOKEN:-dev-fix-admin}"
export FIX_ACCEPTOR_PORT="${FIX_ACCEPTOR_PORT:-8980}"
# Defaulting DEV_FALLBACK to false — when a real API_TOKEN_ENCRYPTION_KEY is
# present in backend/.env (the typical case post-rotation), the fallback path
# inside buildApiDispatcherEncryptor is never reached and the flag is dead
# weight. Defaulting to true re-introduces a latent foot-gun: anyone running
# the binary outside this script with ENVIRONMENT unset would trip the
# production assertion in a way that's hard to debug. Opt in explicitly by
# setting API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK=true when you actually want
# the random process-lifetime key behavior.
export API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK="${API_TOKEN_ENCRYPTION_KEY_DEV_FALLBACK:-false}"
: "${API_TOKEN_ENCRYPTION_KEY:?API_TOKEN_ENCRYPTION_KEY not set — refusing to fall back to a value that is in git history (origin/main:de336a41af)}"

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
