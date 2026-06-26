#!/usr/bin/env bash
set -euo pipefail

# Start backend services via docker-compose.backend.yml
# Uses local Postgres running on host. Do not start Postgres container.
# Run from repo root: ./scripts/start-backend.sh

COMPOSE_FILE="docker-compose.backend.yml"

cd "$(dirname "${BASH_SOURCE[0]}")/.."


echo "Bringing up backend services using ${COMPOSE_FILE}..."

# By default prefer linux/amd64 to avoid platform mismatch when images are only published for amd64
# Users can override by exporting DOCKER_DEFAULT_PLATFORM before running this script.
if [ -z "${DOCKER_DEFAULT_PLATFORM-}" ]; then
  export DOCKER_DEFAULT_PLATFORM="linux/amd64"
fi
arch="$(uname -m)"
echo "Using DOCKER_DEFAULT_PLATFORM=${DOCKER_DEFAULT_PLATFORM} (detected arch: ${arch})"

# Remove any existing containers that would conflict with compose container names
# (e.g. left-over semlayer-backend) to avoid 'name already in use' errors.
existing_containers=$(docker ps -a -q --filter "name=semlayer-backend" || true)
if [ -n "${existing_containers}" ]; then
  echo "Found existing containers matching 'semlayer-backend'. Removing to avoid name conflicts..."
  docker rm -f ${existing_containers} || true
fi

# -----------------------------------------------------------------------------
# Pre-flight: verify local Postgres is reachable
# The compose file is configured to use Postgres on the Docker host (host.docker.internal)
# so we ensure the host Postgres is up before bringing up services. Users running a
# remote DB or Docker-hosted Postgres can set SKIP_PG_CHECK=1 to skip this probe.
# -----------------------------------------------------------------------------

LOCAL_PG_HOST="${LOCAL_PG_HOST:-host.docker.internal}"
LOCAL_PG_PORT="${LOCAL_PG_PORT:-5432}"

if [ -z "${SKIP_PG_CHECK-}" ]; then
  echo "🔎 Checking for Postgres at ${LOCAL_PG_HOST}:${LOCAL_PG_PORT} (this may take a few seconds)..."
  pg_ok=0
  for i in {1..30}; do
    # Prefer nc if available (portable), fall back to /dev/tcp
    if command -v nc >/dev/null 2>&1; then
      if nc -z ${LOCAL_PG_HOST} ${LOCAL_PG_PORT} >/dev/null 2>&1; then
        pg_ok=1
        break
      fi
    else
      # bash TCP probe (may not be available in some shells)
      if timeout 1 bash -c "</dev/tcp/${LOCAL_PG_HOST}/${LOCAL_PG_PORT}" >/dev/null 2>&1; then
        pg_ok=1
        break
      fi
    fi
    sleep 1
  done

  if [ "${pg_ok}" -ne 1 ]; then
    echo "❌ Could not reach Postgres at ${LOCAL_PG_HOST}:${LOCAL_PG_PORT}."
    echo "   Please ensure Postgres is running on the host (localhost) and listening on that port."
    echo "   To skip this check and continue, run: SKIP_PG_CHECK=1 ./scripts/start-backend.sh"
    exit 1
  fi
  echo "✅ Local Postgres appears reachable."
else
  echo "⚠️  SKIP_PG_CHECK is set; skipping Postgres availability probe."
fi

docker compose -f ${COMPOSE_FILE} up --build -d

echo "Waiting for RabbitMQ management API to respond..."
# Wait for RabbitMQ to be healthy
for i in {1..30}; do
  if curl -s http://localhost:15672/api/whoami -u guest:guest >/dev/null 2>&1; then
    echo "RabbitMQ is up"
    break
  fi
  sleep 1
done

echo "All backend services started. Use 'docker compose -f ${COMPOSE_FILE} ps' to view status." 

echo "To follow logs: docker compose -f ${COMPOSE_FILE} logs -f backend event-router rabbitmq"