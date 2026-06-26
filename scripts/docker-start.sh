#!/usr/bin/env bash
set -euo pipefail

# Helper to start the backend compose stack using .env values
# Usage: ./scripts/docker-start.sh [up|down|logs]

CMD=${1:-up}
COMPOSE_FILE="docker-compose.backend.yml"

if [ ! -f .env ]; then
  echo ".env not found. Copy .env.example -> .env and edit credentials before starting."
  exit 1
fi

case "$CMD" in
  up)
    # Start full stack (all services defined in compose)
    docker compose -f "$COMPOSE_FILE" up -d
    ;;
  up-minimal)
    # Start only the minimal set required for local backend dev (Hasura, RabbitMQ, Backend)
    docker compose -f "$COMPOSE_FILE" up -d hasura rabbitmq backend
    ;;
  down)
    docker compose -f "$COMPOSE_FILE" down
    ;;
  logs)
    docker compose -f "$COMPOSE_FILE" logs -f
    ;;
  *)
    echo "Usage: $0 [up|up-minimal|down|logs]"
    exit 2
    ;;
esac
