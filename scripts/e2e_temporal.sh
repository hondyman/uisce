#!/usr/bin/env bash
set -euo pipefail

NAME_RMQ="e2e-rabbitmq"
NAME_TMP="e2e-temporal"
PORT_RMQ=5672
PORT_TMP=7233

cleanup() { docker rm -f "$NAME_RMQ" "$NAME_TMP" 2>/dev/null || true; }
trap cleanup EXIT

cleanup

echo "Starting RabbitMQ..."
docker run -d --name "$NAME_RMQ" -p "$PORT_RMQ:$PORT_RMQ" rabbitmq:3.11-management

echo "Starting Temporal server (docker image) ..."
# Use the official Temporal docker image; adjust tag if desired
docker run -d --name "$NAME_TMP" -p "$PORT_TMP:$PORT_TMP" temporalio/auto-setup:1.20

echo "Waiting for services to become ready..."
until nc -z localhost "$PORT_RMQ"; do sleep 1; done
until nc -z localhost "$PORT_TMP"; do sleep 1; done

RABBITMQ_URL="amqp://guest:guest@localhost:$PORT_RMQ/" TEMPORAL_URL="localhost:$PORT_TMP" \
  go run ./backend/cmd/e2e_temporal

EXIT_CODE=$?

cleanup
exit "$EXIT_CODE"
