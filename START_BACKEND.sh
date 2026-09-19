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

# =============================================================================
# Platform detection: macOS gets a native build + local run; Linux uses the
# pre-built server binary (existing deploy path below).
# =============================================================================
if [[ "$(uname -s)" == "Darwin" ]]; then
    ###############################################################################
    #                macOS: native build, foreground run, no deploy                #
    ###############################################################################
    # Local laptop run. NEVER touches the host's port 8080 — that may belong to
    # another project, a Vite proxy, or a Docker container. Use a separate port
    # by default, overrideable via PORT in .env.
    : "${PORT:=8081}"

    # 1. Source user env FIRST — all checks below must see sourced values.
    #    Otherwise a forbidden value sitting in .env would defeat the rejection
    #    check that runs against the empty inherited env.
    for env_file in "$SCRIPT_DIR/.env" "$BACKEND_DIR/.env"; do
        if [ -f "$env_file" ]; then
            echo -e "${YELLOW}Loading environment from $(basename "$env_file")...${NC}"
            set -a; source "$env_file"; set +a
        fi
    done

    # 2. NOW reject forbidden values — effective against .env contents too.
    if [[ "${JWT_SECRET:-}" == "test-secret" ]]; then
        echo -e "${RED}❌ JWT_SECRET=test-secret is forbidden (in git history, was the JWT compromise vector).${NC}"
        echo -e "${RED}   Unset it and let the script generate an ephemeral secret per run.${NC}"
        exit 1
    fi

    # 3. Ephemeral random fallbacks — never reuse defaults from git history.
    #    Local dev tokens don't need to survive restarts; this is strictly safer
    #    than any default and removes the "careless operator copies the weak
    #    default" failure mode entirely.
    : "${JWT_SECRET:=$(openssl rand -hex 32)}"
    export JWT_SECRET
    export API_TOKEN_ENCRYPTION_KEY="${API_TOKEN_ENCRYPTION_KEY:-$(openssl rand -hex 32)}"

    # 4. Connectivity defaults. .env may have overridden any of these above;
    #    these :="..." only fire if still unset. localhost for DATABASE_URL is
    #    a sensible fresh-clone fallback; in practice your Mac's .env points
    #    at the server DB via ~/.uisce/certs/, so this default rarely fires.
    : "${DATABASE_URL:=postgresql://postgres:postgres@localhost:5432/alpha?sslmode=disable}"
    : "${POSTGRES_DSN:=$DATABASE_URL}"
    : "${TEMPORAL_HOST:=100.84.50.65:7233}"
    : "${TEMPORAL_RETRY_ATTEMPTS:=2}"   # local runs shouldn't hang ~120s on unreachable Temporal

    cd "$BACKEND_DIR"
    echo -e "${YELLOW}Local macOS run: building native binary in /tmp/uisce-server-local...${NC}"
    go build -buildvcs=false -o /tmp/uisce-server-local ./cmd/server

    echo -e "${YELLOW}Starting server on port ${PORT} (foreground; Ctrl+C to stop)...${NC}"
    echo -e "${YELLOW}   DATABASE_URL: ${DATABASE_URL:0:50}...${NC}"
    echo -e "${YELLOW}   TEMPORAL_HOST: $TEMPORAL_HOST${NC}"
    exec /tmp/uisce-server-local
fi

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
