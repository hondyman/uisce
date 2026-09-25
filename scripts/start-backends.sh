#!/usr/bin/env bash
# Starts everything the Uisce backend needs, from a Mac, against the shared
# Docker host (100.84.50.65):
#   1. an SSH tunnel to the host's Docker (for the file engine),
#   2. checks the host's services (Postgres, Temporal, Keycloak, Redis,
#      Redpanda, Infisical) are reachable,
#   3. makes sure the DataFusion file engine container is running with its
#      token from Infisical,
#   4. pulls secrets from Infisical into the .env files,
#   5. builds the server from source (never a stale binary) and starts it,
#      then waits until it answers.
#
# The pipeline's Temporal worker runs inside the server; nothing else to start.
#
# Usage:
#   scripts/start-backends.sh                       # this checkout, port 8080
#   scripts/start-backends.sh ../uisce-datapipeline 8090   # another checkout/port
#
# Secrets come from Infisical (log in once: infisical login
# --domain=http://100.84.50.65:8085/api --interactive). Nothing is printed.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"   # holds the .env files
SRC_DIR="$(cd "${1:-$REPO_ROOT}" && pwd)"                      # checkout to build
PORT="${2:-8080}"

HOST=100.84.50.65
DOCKER_SOCK=/tmp/rdocker.sock
INFISICAL_DOMAIN="http://$HOST:8085/api"
INFISICAL_PROJECT=860e3163-8e2d-410b-a9e5-c7dd44d1e343
INFISICAL_ENV=dev
ENGINE=uisce-datafusion-engine
LOG_DIR="$REPO_ROOT/logs"
mkdir -p "$LOG_DIR"

ok()   { printf '  \033[32m✔\033[0m %s\n' "$*"; }
warn() { printf '  \033[33m!\033[0m %s\n' "$*"; }
die()  { printf '  \033[31m✘\033[0m %s\n' "$*"; exit 1; }
step() { printf '\n\033[1m%s\033[0m\n' "$*"; }

step "1. Docker tunnel to $HOST"
if DOCKER_HOST=unix://$DOCKER_SOCK docker info >/dev/null 2>&1; then
  ok "tunnel is up"
else
  rm -f "$DOCKER_SOCK"
  ssh -fNT -o ExitOnForwardFailure=yes -L "$DOCKER_SOCK:/var/run/docker.sock" "eganpj@$HOST" </dev/null >/dev/null 2>&1 \
    || die "could not open the SSH tunnel (check: ssh eganpj@$HOST)"
  sleep 1
  DOCKER_HOST=unix://$DOCKER_SOCK docker info >/dev/null 2>&1 && ok "tunnel opened" || die "tunnel opened but Docker does not answer"
fi
export DOCKER_HOST=unix://$DOCKER_SOCK

step "2. Services on $HOST"
required_down=0
for svc in "Postgres:5432:required" "Temporal:7233:required" "Keycloak:8443:required" \
           "Redis:6379:optional" "Redpanda:9092:optional" "Infisical:8085:optional" "File engine:8091:optional"; do
  IFS=: read -r name port need <<<"$svc"
  if nc -z -w 3 "$HOST" "$port" >/dev/null 2>&1; then
    ok "$name ($port)"
  elif [ "$need" = required ]; then
    warn "$name ($port) is DOWN"; required_down=1
  else
    warn "$name ($port) is down (optional)"
  fi
done
[ "$required_down" = 0 ] || die "a required service is down - start it on $HOST (docker ps -a) and rerun"

step "3. File engine"
engine_token() {
  infisical secrets get FILE_ENGINE_TOKEN --domain="$INFISICAL_DOMAIN" \
    --projectId="$INFISICAL_PROJECT" --env="$INFISICAL_ENV" --plain --silent </dev/null 2>/dev/null || true
}
state=$(docker inspect -f '{{.State.Running}}' "$ENGINE" 2>/dev/null || echo missing)
if [ "$state" = true ]; then
  ok "$ENGINE is running"
elif [ "$state" = false ]; then
  docker start "$ENGINE" >/dev/null && ok "$ENGINE started"
else
  T=$(engine_token)
  [ -n "$T" ] || die "FILE_ENGINE_TOKEN not found in Infisical (log in, or check the secret exists)"
  docker run -d --name "$ENGINE" --restart unless-stopped -p 8091:8091 \
    -v uisce-pipeline-files:/data/files -e PORT=8091 -e DATAFUSION_FILE_ROOT=/data/files \
    -e FILE_ENGINE_TOKEN="$T" uisce-datafusion-engine:local >/dev/null
  unset T
  ok "$ENGINE created"
fi
code=$(curl -s -o /dev/null -w '%{http_code}' -X POST "http://$HOST:8091/files/list" \
  -H 'content-type: application/json' -d '{"prefix":"x"}' || true)
case "$code" in
  401) ok "engine requires its token" ;;
  503) die "engine has no token configured - remove it (docker rm -f $ENGINE) and rerun" ;;
  *)   warn "engine answered $code" ;;
esac

step "4. Secrets from Infisical"
if command -v infisical >/dev/null 2>&1 && [ -x "$REPO_ROOT/scripts/infisical-bootstrap.sh" ]; then
  if "$REPO_ROOT/scripts/infisical-bootstrap.sh" -e "$INFISICAL_ENV" >"$LOG_DIR/infisical-bootstrap.log" 2>&1; then
    ok ".env files refreshed"
  else
    warn "Infisical bootstrap failed (see logs/infisical-bootstrap.log) - using the existing .env files"
  fi
else
  warn "infisical CLI not found - using the existing .env files"
fi
set -a
for f in "$REPO_ROOT/.env" "$REPO_ROOT/backend/.env"; do
  # shellcheck disable=SC1090
  [ -f "$f" ] && . "$f"
done
set +a
[ -n "${POSTGRES_DSN:-${DATABASE_URL:-}}" ] || die "POSTGRES_DSN is not set - check Infisical"
export POSTGRES_DSN="${POSTGRES_DSN:-$DATABASE_URL}"
export TEMPORAL_HOST="${TEMPORAL_HOST:-$HOST:7233}"
export PORT
# Data pipelines: the file engine, and staging tables in the crims database.
export DATAPIPELINE_ENGINE_URL="${DATAPIPELINE_ENGINE_URL:-http://$HOST:8091}"
export DATAPIPELINE_STAGING_DSN="${DATAPIPELINE_STAGING_DSN:-$(printf '%s' "$POSTGRES_DSN" | sed -E 's#/alpha([?]|$)#/crims\1#')}"
[ -n "${DATAPIPELINE_ENGINE_TOKEN:-}" ] && ok "pipeline engine token loaded" || warn "DATAPIPELINE_ENGINE_TOKEN missing - pipelines cannot reach the file engine"

step "5. Backend on :$PORT from $SRC_DIR ($(git -C "$SRC_DIR" branch --show-current 2>/dev/null || echo '?'))"
if pid=$(lsof -tiTCP:"$PORT" -sTCP:LISTEN 2>/dev/null); then
  cmd=$(ps -p "$pid" -o comm= 2>/dev/null || true)
  case "$cmd" in
    *server*) kill "$pid"; sleep 2; ok "stopped the previous server (pid $pid)" ;;
    *) die "port $PORT is used by '$cmd' (pid $pid) - stop it or pick another port" ;;
  esac
fi
( cd "$SRC_DIR/backend" && go build -o server ./cmd/server ) >"$LOG_DIR/backend-build.log" 2>&1 \
  || { tail -20 "$LOG_DIR/backend-build.log"; die "build failed (logs/backend-build.log)"; }
git -C "$SRC_DIR" checkout -q -- go.work.sum 2>/dev/null || true
ok "built from source"

LOG="$LOG_DIR/backend_${PORT}_$(date +%Y%m%d_%H%M%S).log"
# Fully detached: no stream of this script is inherited, so the script (and
# anything reading its output) finishes while the server keeps running.
cd "$SRC_DIR/backend"
nohup ./server </dev/null >"$LOG" 2>&1 &
PID=$!
disown "$PID" 2>/dev/null || true
cd "$REPO_ROOT"
echo "$PID" >"$LOG_DIR/backend_$PORT.pid"
for _ in $(seq 1 90); do
  if ! kill -0 "$PID" 2>/dev/null; then
    tail -25 "$LOG"; die "server exited - see $LOG"
  fi
  if [ "$(curl -s -o /dev/null -w '%{http_code}' "http://localhost:$PORT/api/message-catalog/languages")" != 000 ]; then
    ok "server is answering"
    printf '\n  URL:  http://localhost:%s\n  PID:  %s   (stop: kill %s)\n  Log:  %s\n' "$PORT" "$PID" "$PID" "$LOG"
    exit 0
  fi
  sleep 2
done
die "server did not answer within 3 minutes - see $LOG"
