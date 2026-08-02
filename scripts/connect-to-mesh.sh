#!/usr/bin/env bash
# =============================================================================
# Uisce — Connect MacBook to Remote Docker Mesh
# =============================================================================
# Configures the local backend to connect to a docker-compose mesh running
# on a remote Linux host (not localhost).
#
# Usage:
#   source scripts/connect-to-mesh.sh          # just export vars into current shell
#   ./scripts/connect-to-mesh.sh               # print the commands without running
#   ./scripts/connect-to-mesh.sh --start     # also start the backend
#   ./scripts/connect-to-mesh.sh --verify    # verify connectivity only
#
# Configuration:
#   Set MESH_HOST in .env.mesh or environment before sourcing.
#   Example:  export MESH_HOST=192.168.1.100
# =============================================================================

set -euo pipefail

# --- Resolve script directory ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# --- Load mesh host from .env.mesh if it exists ---
ENV_MESH="$PROJECT_ROOT/.env.mesh"
if [[ -f "$ENV_MESH" ]] && grep -q "MESH_HOST=" "$ENV_MESH" 2>/dev/null; then
  # shellcheck disable=SC1090
  set -a
  source "$ENV_MESH"
  set +a
fi

# --- Defaults ---
: "${MESH_HOST:=${MESH_HOST:-}}"
: "${MESH_USER:=${MESH_USER:-$(whoami)}}"
: "${MESH_PASSWORD:=${MESH_PASSWORD:-uisce_admin_password_localdev}}"
: "${MESH_DB:=${MESH_DB:-uisce_control_plane}}"
: "${MESH_DB_PORT:=${MESH_DB_PORT:-5432}}"
: "${MESH_REDIS_PORT:=${MESH_REDIS_PORT:-6379}}"
: "${MESH_KAFKA_PORT:=${MESH_KAFKA_PORT:-9092}}"
: "${MESH_STARROCKS_PORT:=${MESH_STARROCKS_PORT:-9030}}"
: "${MESH_BACKEND_PORT:=${MESH_BACKEND_PORT:-8080}}"
: "${BACKEND_DIR:=${PROJECT_ROOT}/backend}"

# --- Colour codes ---
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log_info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_fail()  { echo -e "${RED}[FAIL]${NC}  $*"; }

# =============================================================================
# Usage
# =============================================================================
usage() {
  cat <<EOF
Usage: source scripts/connect-to-mesh.sh  [options]

  Connect the local MacBook backend to a docker-compose mesh running on a
  remote Linux host. Run WITHOUT --start to just export variables into
  your current shell:

    source scripts/connect-to-mesh.sh

  Or pipe into a new shell:

    eval "\$(scripts/connect-to-mesh.sh)"

Options:
  --start       Start the backend after configuring
  --verify      Verify connectivity to all mesh services and exit
  --stop        Stop a running backend
  --env FILE    Read MESH_HOST from FILE (default: .env.mesh in project root)
  -h, --help    Show this help

Environment variables (set before sourcing, or in .env.mesh):
  MESH_HOST       IP address of the Linux mesh host (required)
  MESH_USER       SSH user for mesh host (default: current user)
  MESH_PASSWORD   Postgres password (default: uisce_admin_password_localdev)
  MESH_DB         Database name (default: uisce_control_plane)
  MESH_DB_PORT    Postgres port (default: 5432)
  MESH_REDIS_PORT Redis port (default: 6379)
  MESH_KAFKA_PORT Kafka/Redpanda port (default: 9092)
  MESH_STARROCKS_PORT StarRocks port (default: 9030)
  BACKEND_DIR     Path to backend (default: \$PROJECT_ROOT/backend)

Examples:
  # Set MESH_HOST and connect
  export MESH_HOST=192.168.1.100
  source scripts/connect-to-mesh.sh

  # Or put it in .env.mesh
  echo "MESH_HOST=192.168.1.100" > .env.mesh
  source scripts/connect-to-mesh.sh

  # Verify connectivity without starting
  source scripts/connect-to-mesh.sh --verify

  # Start backend pointing at mesh
  source scripts/connect-to-mesh.sh --start
EOF
}

# =============================================================================
# Parse arguments
# =============================================================================
ACTION=""   # start | verify | stop

while [[ $# -gt 0 ]]; do
  case "$1" in
    --start)    ACTION="start";    shift ;;
    --verify)   ACTION="verify";   shift ;;
    --stop)     ACTION="stop";     shift ;;
    --env)      ENV_MESH="$2"; shift 2 ;;
    -h|--help)  usage; exit 0 ;;
    *)          echo "Unknown option: $1"; usage; exit 1 ;;
  esac
done

# =============================================================================
# Guard: MESH_HOST required
# =============================================================================
if [[ -z "$MESH_HOST" ]]; then
  log_fail "MESH_HOST is not set."
  echo ""
  log_info "Set it via:"
  echo "    export MESH_HOST=192.168.1.100"
  echo "    echo 'MESH_HOST=192.168.1.100' > .env.mesh"
  echo ""
  log_info "Then re-run: source scripts/connect-to-mesh.sh"
  echo ""
  usage
  exit 1
fi

# =============================================================================
# Compute derived values
# =============================================================================
export POSTGRES_DSN="postgres://uisce_admin:${MESH_PASSWORD}@${MESH_HOST}:${MESH_DB_PORT}/${MESH_DB}?sslmode=disable"
export REDIS_ADDR="${MESH_HOST}:${MESH_REDIS_PORT}"
export KAFKA_BROKERS="${MESH_HOST}:${MESH_KAFKA_PORT}"
export STARROCKS_HOST="${MESH_HOST}:${MESH_STARROCKS_PORT}"

# Backend runs on localhost MacBook — tell it to listen on 8080
export PORT="${MESH_BACKEND_PORT}"

# eBPF always off on MacBook
export EBPF_ENABLED=false

# =============================================================================
# Print/export functions
# =============================================================================

# Used when sourced — just print the export statements
print_exports() {
  cat <<EOF
# =============================================================================
# Uisce — MacBook → Mesh connection vars
# MESH_HOST=$MESH_HOST
# =============================================================================
export POSTGRES_DSN="$POSTGRES_DSN"
export REDIS_ADDR="$REDIS_ADDR"
export KAFKA_BROKERS="$KAFKA_BROKERS"
export STARROCKS_HOST="$STARROCKS_HOST"
export PORT="$PORT"
export EBPF_ENABLED="$EBPF_ENABLED"

log_info "Connected to mesh at $MESH_HOST"
log_info "POSTGRES_DSN : ${POSTGRES_DSN%@*}@..."
log_info "REDIS_ADDR   : $REDIS_ADDR"
log_info "KAFKA_BROKERS: $KAFKA_BROKERS"
log_info "STARROCKS_HOST: $STARROCKS_HOST"
log_info "Backend port : $PORT"
EOF
}

# =============================================================================
# Verify connectivity
# =============================================================================
verify() {
  echo ""
  echo "=============================================================="
  echo "   Verifying connectivity to mesh at $MESH_HOST"
  echo "=============================================================="

  local failures=0

  # Postgres
  log_info "Testing PostgreSQL :${MESH_DB_PORT}..."
  if command -v psql &>/dev/null; then
    if PGPASSWORD="$MESH_PASSWORD" psql -h "$MESH_HOST" -p "${MESH_DB_PORT}" -U uisce_admin -d "$MESH_DB" -c "SELECT 1;" -q &>/dev/null; then
      log_ok "PostgreSQL :${MESH_DB_PORT} — reachable"
    else
      log_fail "PostgreSQL :${MESH_DB_PORT} — NOT reachable (check credentials, pg_hba.conf, firewall)"
      failures=$((failures + 1))
    fi
  else
    log_warn "psql not installed — skipping PostgreSQL check (brew install postgresql)"
  fi

  # Redis
  log_info "Testing Redis :${MESH_REDIS_PORT}..."
  if command -v redis-cli &>/dev/null; then
    if redis-cli -h "$MESH_HOST" -p "${MESH_REDIS_PORT}" ping &>/dev/null; then
      log_ok "Redis :${MESH_REDIS_PORT} — reachable"
    else
      log_fail "Redis :${MESH_REDIS_PORT} — NOT reachable (check firewall, Redis binding)"
      failures=$((failures + 1))
    fi
  else
    log_warn "redis-cli not installed — skipping Redis check (brew install redis)"
  fi

  # Redpanda / Kafka
  log_info "Testing Redpanda :${MESH_KAFKA_PORT}..."
  if command -v nc &>/dev/null; then
    if nc -z -w3 "$MESH_HOST" "${MESH_KAFKA_PORT}" 2>/dev/null; then
      log_ok "Redpanda :${MESH_KAFKA_PORT} — reachable"
    else
      log_fail "Redpanda :${MESH_KAFKA_PORT} — NOT reachable (check firewall)"
      failures=$((failures + 1))
    fi
  else
    log_warn "nc not installed — skipping Redpanda check"
  fi

  # Backend HTTP (if already running locally)
  log_info "Testing local backend :${MESH_BACKEND_PORT}..."
  if curl -sf --max-time 3 "http://localhost:${MESH_BACKEND_PORT}/health" &>/dev/null; then
    log_ok "Local backend :${MESH_BACKEND_PORT} — running"
  else
    log_warn "Local backend :${MESH_BACKEND_PORT} — not running (start with: go run ./backend/cmd/server)"
  fi

  echo ""
  if [[ $failures -gt 0 ]]; then
    log_fail "$failures service(s) unreachable — fix before proceeding"
    echo ""
    log_info "Common fixes:"
    echo "  Linux box firewall:"
    echo "    sudo ufw allow from \$MACBOOK_SUBNET to any port 5432,6379,9092"
    echo "  Postgres pg_hba.conf — add:"
    echo "    host  all  all  \$MACBOOK_IP/32  md5"
    echo ""
    return 1
  else
    log_ok "All reachable services — mesh is ready"
  fi
}

# =============================================================================
# Stop backend
# =============================================================================
stop_backend() {
  log_info "Stopping any running backend on port $MESH_BACKEND_PORT..."
  local pid
  pid=$(lsof -ti ":${MESH_BACKEND_PORT}" 2>/dev/null || true)
  if [[ -n "$pid" ]]; then
    kill "$pid" 2>/dev/null && log_ok "Stopped backend (PID $pid)" || log_warn "Could not kill PID $pid"
  else
    log_info "No backend running on port $MESH_BACKEND_PORT"
  fi
}

# =============================================================================
# Start backend
# =============================================================================
start_backend() {
  echo ""
  log_info "Starting backend with mesh at $MESH_HOST..."
  log_info "Backend will listen on :$PORT"
  log_info "POSTGRES_DSN : ${POSTGRES_DSN%@*}@..."
  log_info "REDIS_ADDR   : $REDIS_ADDR"
  log_info "KAFKA_BROKERS: $KAFKA_BROKERS"
  echo ""
  log_info "Starting: cd $BACKEND_DIR && go run ./cmd/server"
  echo ""
  cd "$BACKEND_DIR" && go run ./cmd/server
}

# =============================================================================
# Dispatch
# =============================================================================

if [[ -z "${BASH_SOURCE[0]}" || "${BASH_SOURCE[0]}" != "${0}" ]]; then
  # Sourced — print exports and wrap with a startup guard
  print_exports
  echo ""

  if [[ -n "$ACTION" ]]; then
    case "$ACTION" in
      verify)
        verify
        ;;
      stop)
        stop_backend
        ;;
      start)
        start_backend
        ;;
    esac
  else
    echo "To also start the backend, run: source scripts/connect-to-mesh.sh --start"
    echo "To verify connectivity, run:  source scripts/connect-to-mesh.sh --verify"
    echo ""
  fi
else
  # Executed directly — behave like a normal script
  case "$ACTION" in
    verify)
      verify
      ;;
    stop)
      stop_backend
      ;;
    start)
      start_backend
      ;;
    *)
      print_exports
      ;;
  esac
fi
