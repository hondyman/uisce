#!/usr/bin/env bash
# =============================================================================
# Uisce Semantic OS — Production Bootstrap
# =============================================================================
# Handles database readiness verification, schema migration execution,
# and fail-fast bootstrapping of the Uisce Core binary.
#
# Usage:
#   ./scripts/bootstrap-production.sh          # local dev credentials
#   infisical run -- ./scripts/bootstrap-production.sh  # production via Infisical
#
# Required env vars:
#   POSTGRES_DSN       PostgreSQL connection string
#   JWT_SECRET         JWT signing key
# Optional:
#   INFISICAL_TOKEN    Infisical service token (activates secret injection)
#   SKIP_MIGRATIONS    Set to "1" to skip migration execution
# =============================================================================

set -euo pipefail

# --- Colour codes ---
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'
NC='\033[0m' # No Color

log_info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_fatal() { echo -e "${RED}[FATAL]${NC} $*" >&2; exit 1; }

# --- Guard required env vars ---
if [[ -z "${POSTGRES_DSN:-}" ]]; then
  # Attempt default from compose env
  export POSTGRES_DSN="postgres://uisce_admin:uisce_admin_password_localdev@localhost:5432/uisce_control_plane?sslmode=disable"
  log_warn "POSTGRES_DSN not set — defaulting to local compose default."
fi

if [[ -z "${JWT_SECRET:-}" ]]; then
  log_fatal "JWT_SECRET environment variable is not set"
fi

if [[ -z "${INFISICAL_TOKEN:-}" ]]; then
  log_info "INFISICAL_TOKEN not set — skipping Infisical secret injection."
fi

# --- Resolve backend directory ---
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/../backend" && pwd)"
MIGRATIONS_DIR="$(cd "$SCRIPT_DIR/../backend/db/migrations" && pwd)"

echo ""
echo "=============================================================="
echo "   Uisce Semantic OS — G-SIFI Production Bootstrap"
echo "=============================================================="
log_info "Backend dir  : $BACKEND_DIR"
log_info "Migrations   : $MIGRATIONS_DIR"
log_info "POSTGRES_DSN : ${POSTGRES_DSN%@*}@..."  # redact password in log

# --- 1. Fail-Fast: Database Readiness ---
echo ""
log_info "[1/4] Verifying database connectivity..."
if ! command -v psql &>/dev/null; then
  log_fatal "psql not found. Install postgresql-client: brew install postgresql"
fi

# Support both postgres:// URLs (Go) and postgresql:// URLs (psql) via pg_isready first
for i in $(seq 1 30); do
  if pg_isready -d "$POSTGRES_DSN" -q 2>/dev/null; then
    log_ok "Database is accepting connections."
    break
  fi
  if [[ $i -eq 30 ]]; then
    log_fatal "Database did not become ready after 30 attempts (60s). Exiting."
  fi
  echo "   Waiting for PostgreSQL... ($i/30)"
  sleep 2
done

# --- 2. Execute Structural Migrations ---
echo ""
if [[ "${SKIP_MIGRATIONS:-}" == "1" ]]; then
  log_warn "SKIP_MIGRATIONS=1 — skipping migration execution."
else
  log_info "[2/4] Executing structural database migrations via backend/cmd/migrate..."

  # This step previously ran its own psql -f loop against a schema_migrations
  # table, entirely separate from the migration tracking (oms.migration_log)
  # that backend/internal/migrations.ApplyMigrations uses at server boot
  # (see step 3/4 below — cmd/server calls ApplyMigrations on every start).
  # That meant every production bootstrap applied migrations twice, through
  # two different mechanisms, with no transaction wrapping in this one and
  # no checksum tracking at all. Replaced with the canonical runner via the
  # standalone CLI so there is exactly one migration mechanism, one
  # bookkeeping table, and one place that fails loudly on drift.
  cd "$BACKEND_DIR"
  log_info "  Building migrate CLI..."
  if ! go build -o bin/migrate ./cmd/migrate 2>&1; then
    log_fatal "go build ./cmd/migrate failed. Check compilation errors above."
  fi

  log_info "  Verifying no drift in already-applied migrations..."
  if ! DATABASE_URL="$POSTGRES_DSN" ./bin/migrate verify; then
    log_fatal "Migration drift detected — an applied migration file's content no longer matches what was recorded. Resolve before bootstrapping. See output above."
  fi

  log_info "  Applying pending migrations..."
  if ! DATABASE_URL="$POSTGRES_DSN" ./bin/migrate up; then
    log_fatal "Migration failed. See output above."
  fi
  log_ok "Migrations complete."
fi

# --- 3. Build Uisce Core Binary ---
echo ""
log_info "[3/4] Building Uisce Core binary..."
cd "$BACKEND_DIR"

# Ensure go.mod is tidy (may fail due to pre-existing jwt-middleware conflict — skip)
if [[ -z "${SKIP_GO_MOD_TIDY:-}" ]]; then
  if go mod tidy 2>&1 | grep -v "libs/jwt-middleware"; then
    log_warn "go mod tidy produced warnings (ignored if related to jwt-middleware)."
  fi
fi

log_info "Compiling ./cmd/server..."
if ! go build -o bin/uisce-core ./cmd/server 2>&1; then
  log_fatal "go build failed. Check compilation errors above."
fi

BINARY_SHA256=$(shasum -a 256 "$BACKEND_DIR/bin/uisce-core" | cut -d' ' -f1)
log_ok "Build complete. SHA256: ${BINARY_SHA256:0:16}..."

# --- 4. Launch Uisce Core ---
echo ""
log_info "[4/4] Launching Uisce Core..."
log_info "API Gateway  : http://localhost:8080"
log_info "Flight SQL   : http://localhost:8090 (aspirational stub)"
log_info "FIX Acceptor : localhost:8980 (aspirational stub)"

export PORT=8080

exec "$BACKEND_DIR/bin/uisce-core"
