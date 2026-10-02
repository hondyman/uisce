#!/usr/bin/env bash
# Runs the Uisce backend natively on this Mac, against the shared Docker host
# (default 100.84.50.65) where every service already runs.
#
# This is the Mac-native sibling of scripts/start-backends.sh. The difference:
# no Docker tunnel and no container management, because nothing needs Docker
# locally - the services are on the host and reachable over the network.
#
#   1. checks every service the backend needs is reachable on the host,
#   2. loads secrets from the .env files (optionally refreshed from Infisical),
#   3. rewrites the Docker Compose service names to the host's address, so the
#      same .env works both inside compose and on a Mac,
#   4. builds the server from source (never a stale binary) and starts it,
#   5. waits until it answers.
#
# Usage:
#   scripts/start-backend-mac.sh                     # port 8080
#   scripts/start-backend-mac.sh 8090                # another port
#   scripts/start-backend-mac.sh ../uisce-datapipeline 8090
#
# Stop:  kill "$(cat logs/backend_<port>.pid)"
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"   # holds the .env files
SRC_DIR="${1:-$REPO_ROOT}"
PORT="${2:-8080}"

HOST="${Uisce_HOST:-100.84.50.65}"
LOG_DIR="$REPO_ROOT/logs"
mkdir -p "$LOG_DIR"

ok()   { printf '  \033[32m✔\033[0m %s\n' "$*"; }
warn() { printf '  \033[33m!\033[0m %s\n' "$*"; }
die()  { printf '  \033[31m✘\033[0m %s\n' "$*"; exit 1; }
step() { printf '\n\033[1m%s\033[0m\n' "$*"; }

# --- 1. the host's services ---------------------------------------------------
# name port pairs the backend actually dials. Docker Compose service names are
# NOT resolvable from a Mac, which is why step 3 rewrites the .env values.
step "1. Services on $HOST"
check() {  # check <required|optional> <name> <port>
  if nc -z -w 3 "$HOST" "$3" >/dev/null 2>&1; then
    ok "$2 ($3)"
  elif [ "$1" = required ]; then
    warn "$2 ($3) is DOWN"; down=1
  else
    warn "$2 ($3) is down (optional)"
  fi
}
down=0
check required Postgres 5432
check required Temporal 7233
check required Keycloak 8443
check required FileEngine 8091
check optional Redis 6379
check optional Redpanda 9092
check optional Infisical 8085
check optional MinIO 9000
check optional StarRocks 9030
[ "$down" = 0 ] || die "a required service is down on $HOST (ssh $HOST 'docker ps -a')"

# --- 2. secrets --------------------------------------------------------------
step "2. Secrets"
if command -v infisical >/dev/null 2>&1 && [ -x "$REPO_ROOT/scripts/infisical-bootstrap.sh" ]; then
  if "$REPO_ROOT/scripts/infisical-bootstrap.sh" -e dev >"$LOG_DIR/infisical-bootstrap.log" 2>&1; then
    ok ".env files refreshed from Infisical"
  else
    warn "Infisical bootstrap failed (see logs/infisical-bootstrap.log) - using the existing .env files"
  fi
else
  warn "infisical CLI not found - using the existing .env files"
fi
for f in "$REPO_ROOT/.env" "$REPO_ROOT/backend/.env"; do
  # shellcheck disable=SC1090
  if [ -f "$f" ]; then set -a; . "$f"; set +a; fi
done

# --- 3. point everything at the host ----------------------------------------
# The .env is shared with docker-compose, so it carries service names
# (minio:9000, starrocks-fe, temporal:7233, redpanda:9092) that only resolve
# inside the compose network. Left as-is, the backend starts and then fails
# quietly when it reaches for Iceberg, CDC or streams.
step "3. Pointing the process at $HOST"
export POSTGRES_HOST="${POSTGRES_HOST:-$HOST}"
if [ "$POSTGRES_HOST" = "host.docker.internal" ]; then export POSTGRES_HOST="$HOST"; fi
[ -n "${POSTGRES_DSN:-${DATABASE_URL:-}}" ] || die "POSTGRES_DSN/DATABASE_URL not set - check Infisical"
export POSTGRES_DSN="${POSTGRES_DSN:-$DATABASE_URL}"
export DATABASE_URL="${DATABASE_URL:-$POSTGRES_DSN}"

export TEMPORAL_HOST="${TEMPORAL_HOST:-$HOST:7233}"
export CDC_TEMPORAL_ADDRESS="${CDC_TEMPORAL_ADDRESS:-$HOST:7233}"
export REDIS_URL="${REDIS_URL:-redis://$HOST:6379}"
export KAFKA_BROKERS="${KAFKA_BROKERS:-$HOST:9092}"
export MINIO_ENDPOINT="${MINIO_ENDPOINT:-100.84.50.65:9000}"
case "$MINIO_ENDPOINT" in
  minio:*|lakekeeper:*) export MINIO_ENDPOINT="$HOST:${MINIO_ENDPOINT##*:}";;
esac
export STARROCKS_HOST="${STARROCKS_HOST:-$HOST}"
case "$STARROCKS_HOST" in
  starrocks-fe|starrocks-be) export STARROCKS_HOST="$HOST";;
esac
export KEYCLOAK_HOST="${KEYCLOAK_HOST:-$HOST}"
export PORT

# Data pipelines: the file engine, and staging tables in the crims database.
# Without DATAPIPELINE_STAGING_DSN the server logs "staging loads are disabled"
# and every pipeline run writes nothing while still reporting success.
export DATAPIPELINE_ENGINE_URL="${DATAPIPELINE_ENGINE_URL:-http://$HOST:8091}"
export DATAPIPELINE_STAGING_DSN="${DATAPIPELINE_STAGING_DSN:-$(printf '%s' "$POSTGRES_DSN" | sed -E 's#/alpha([?]|$)#/crims\1#')}"
export DATAPIPELINE_ENGINE_TOKEN="${DATAPIPELINE_ENGINE_TOKEN:-${FILE_ENGINE_TOKEN:-}}"

ok "Postgres    $POSTGRES_HOST"
ok "Temporal    $TEMPORAL_HOST"
ok "FileEngine  $DATAPIPELINE_ENGINE_URL"
ok "MinIO       $MINIO_ENDPOINT"
ok "StarRocks   $STARROCKS_HOST:$STARROCKS_PORT"
ok "StagingDB   $(printf '%s' "$DATAPIPELINE_STAGING_DSN" | sed -E 's#(//[^:]+:)[^@]*@#\1***@#' | cut -c1-70)"
[ -n "$DATAPIPELINE_ENGINE_TOKEN" ] && ok "engine token loaded" || warn "no engine token - pipelines cannot read or write files"

# --- 4. build ----------------------------------------------------------------
step "4. Build from source ($SRC_DIR)"
( cd "$SRC_DIR/backend" && go build -o server ./cmd/server ) >"$LOG_DIR/backend-build.log" 2>&1 \
  || { tail -20 "$LOG_DIR/backend-build.log"; die "build failed (logs/backend-build.log)"; }
ok "built backend/server"

# --- 5. start ----------------------------------------------------------------
step "5. Backend on :$PORT"
listener() { lsof -tiTCP:"$PORT" -sTCP:LISTEN 2>/dev/null | head -1; }
health()   { curl -s -o /dev/null -w '%{http_code}' "http://localhost:$PORT/api/message-catalog/languages" 2>/dev/null; }

# Every server started from THIS checkout, whatever port it took. The pid
# files are the record; a pgrep over the built binary catches instances whose
# pid file was lost. Scoped to $SRC_DIR so a second checkout, or the copy
# running on the host, is never touched.
#
# This runs under `set -euo pipefail`, so every branch here must succeed: a
# failed `a && b` list, or a grep that matches nothing, is a non-zero status
# and would kill the script. Hence the explicit `if`s and the guaranteed
# `return 0`.
our_pids() {
  local bin="$SRC_DIR/backend/server" f p pat
  local -a found=()
  # The path is a regex here, so escape its metacharacters: a checkout in
  # "my.checkout" or "a+b" would otherwise match unrelated processes.
  pat="^$(printf '%s' "$bin" | sed 's/[][\\.^$*+?(){}|]/\\&/g')\$"
  for f in "$LOG_DIR"/backend_*.pid; do
    [ -f "$f" ] || continue
    p=$(cat "$f" 2>/dev/null || true)
    case "$p" in ''|*[!0-9]*) continue ;; esac
    if kill -0 "$p" 2>/dev/null; then found+=("$p"); fi
  done
  if command -v pgrep >/dev/null 2>&1; then
    while read -r p; do
      if [ -n "$p" ]; then found+=("$p"); fi
    done < <(pgrep -f "$pat" 2>/dev/null || true)
  fi
  if [ ${#found[@]} -eq 0 ]; then return 0; fi
  printf '%s\n' "${found[@]}" | sort -un
  return 0
}

stop_one() {  # stop_one <pid> <label>
  local p=$1 label=$2
  kill "$p" 2>/dev/null || true
  for _ in $(seq 1 15); do kill -0 "$p" 2>/dev/null || { ok "stopped $label (pid $p)"; return 0; }; sleep 1; done
  kill -9 "$p" 2>/dev/null || true
  sleep 1
  kill -0 "$p" 2>/dev/null && { warn "could not stop $label (pid $p)"; return 1; }
  ok "stopped $label (pid $p)"
}

FOUND=""
for p in $(our_pids); do FOUND="$FOUND $p"; done
if [ -n "$FOUND" ]; then
  for p in $FOUND; do stop_one "$p" "a server from this checkout" || true; done
  # pid files for instances that are now gone are stale; clear them so the
  # next sweep doesn't keep listing dead pids.
  for f in "$LOG_DIR"/backend_*.pid; do
    [ -f "$f" ] || continue
    p=$(cat "$f" 2>/dev/null) || continue
    kill -0 "$p" 2>/dev/null || rm -f "$f"
  done
else
  ok "no existing instance of this checkout's server"
fi

# The target port must now be free. If something is still on it, it is not one
# of ours and must not be killed.
if pid=$(listener) && [ -n "$pid" ]; then
  comm=$(ps -p "$pid" -o comm= 2>/dev/null || true)
  die "port $PORT is still used by '${comm:-unknown}' (pid $pid) and is not this checkout's server - stop it or pick another port"
fi

LOG="$LOG_DIR/backend_${PORT}_$(date +%Y%m%d_%H%M%S).log"
# Start the server from the main shell, not a subshell: $! must be the server's
# own pid, because the readiness test below compares it against what lsof
# reports as holding the port. A subshell adds a process level, $! names the
# wrapper, and the loop then never matches and times out on a healthy server.
cd "$SRC_DIR/backend"
# Launch by absolute path, NOT ./server: the sweep above matches processes on
# "^$SRC_DIR/backend/server$". A relative launch puts argv[0]="./server", which
# that pattern can never match, so the next run would not find its own server.
nohup "$SRC_DIR/backend/server" </dev/null >"$LOG" 2>&1 &
PID=$!
disown "$PID" 2>/dev/null || true
cd "$REPO_ROOT"
echo "$PID" >"$LOG_DIR/backend_$PORT.pid"

for _ in $(seq 1 90); do
  if ! kill -0 "$PID" 2>/dev/null; then
    tail -25 "$LOG"; die "server exited - see $LOG"
  fi
  # Up means *this* process holds the port and answers - not some other one.
  if [ "$(listener)" = "$PID" ] && \
     [ "$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:$PORT/api/message-catalog/languages")" != 000 ]; then
    ok "server is answering"
    printf '\n  URL:  http://localhost:%s\n  PID:  %s   (stop: kill %s)\n  Log:  %s\n' "$PORT" "$PID" "$PID" "$LOG"
    exit 0
  fi
  sleep 2
done
die "server did not answer within 3 minutes - see $LOG"
