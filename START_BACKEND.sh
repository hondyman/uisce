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

echo -e "${YELLOW}Preparing server binary...${NC}"
SERVER_BINARY="${HOME}/uisce-server-fixed"
if [ -f "$SERVER_BINARY" ]; then
    # A binary built for the host (scripts/start-backends.sh always builds from
    # source instead, so this only applies to a deliberate cross-build).
    echo -e "${YELLOW}   using pre-built binary $SERVER_BINARY${NC}"
    cp "$SERVER_BINARY" ./server
    chmod +x ./server
else
    # No cross-built binary on this machine - build locally rather than stopping.
    echo -e "${YELLOW}   no pre-built binary, building from source...${NC}"
    if ! go build -o server ./cmd/server; then
        echo -e "${RED}❌ Build failed (see above). Fix the build, or cross-build and place it at $SERVER_BINARY${NC}"
        exit 1
    fi
    echo -e "${YELLOW}   built ./server${NC}"
fi

if [ -f "$SCRIPT_DIR/scripts/infisical-bootstrap.sh" ] && command -v infisical &>/dev/null; then
    echo -e "${YELLOW}Bootstrapping secrets from Infisical...${NC}"
    INFISICAL_TOKEN="${INFISICAL_TOKEN:-}" "$SCRIPT_DIR/scripts/infisical-bootstrap.sh" -e dev \
      || echo -e "${RED}⚠️  Infisical bootstrap FAILED — serving from stale .env files${NC}"
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

export POSTGRES_DSN="${POSTGRES_DSN:-${DATABASE_URL:-postgresql://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable}}"
export DATABASE_URL="${DATABASE_URL:-$POSTGRES_DSN}"
: "${JWT_SECRET:?JWT_SECRET not set — refusing to start with the test-secret default that is in git history}"
export JWT_SECRET
export PORT="${PORT:-8080}"
export TEMPORAL_HOST="${TEMPORAL_HOST:-100.84.50.65:7233}"
export TEMPORAL_RETRY_ATTEMPTS="${TEMPORAL_RETRY_ATTEMPTS:-2}"
export ENVIRONMENT="${ENVIRONMENT:-}"
: "${API_TOKEN_ENCRYPTION_KEY:?API_TOKEN_ENCRYPTION_KEY not set — refusing to fall back to a value that is in git history (origin/main:de336a41af)}"

# Data pipelines. Without these the server starts but logs "file sources and
# exports are disabled" and "staging loads are disabled" (api wiring), so the
# MDM pipeline appears to run and silently writes nothing. The staging data
# plane is the crims database; the control plane is alpha.
export DATAPIPELINE_ENGINE_URL="${DATAPIPELINE_ENGINE_URL:-http://100.84.50.65:8091}"
export DATAPIPELINE_STAGING_DSN="${DATAPIPELINE_STAGING_DSN:-$(printf '%s' "$POSTGRES_DSN" | sed -E 's#/alpha([?]|$)#/crims\1#')}"
[ -n "${DATAPIPELINE_ENGINE_TOKEN:-}" ] && echo -e "${YELLOW}   DATAPIPELINE_ENGINE_URL: $DATAPIPELINE_ENGINE_URL${NC}" \
  && echo -e "${YELLOW}   DATAPIPELINE_ENGINE_TOKEN: set${NC}" \
  || echo -e "${RED}⚠️  DATAPIPELINE_ENGINE_TOKEN not set — pipelines cannot reach the file engine${NC}"

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
