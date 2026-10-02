#!/usr/bin/env bash
# End-to-end check: a Temporal workflow runs an activity that publishes to
# Kafka/Redpanda, and the event is observed on the "events" topic.
set -euo pipefail

NAME_RP="e2e-redpanda"
NAME_TMP="e2e-temporal"
PORT_KAFKA=9092
PORT_TMP=7233

cleanup() { docker rm -f "$NAME_RP" "$NAME_TMP" 2>/dev/null || true; }
trap cleanup EXIT

cleanup

echo "Starting Redpanda..."
docker run -d --name "$NAME_RP" -p "$PORT_KAFKA:$PORT_KAFKA" \
  docker.redpanda.com/redpandadata/redpanda:latest \
  redpanda start --overprovisioned --smp 1 --memory 512M --reserve-memory 0M --node-id 0 --check=false \
  --kafka-addr "PLAINTEXT://0.0.0.0:$PORT_KAFKA" --advertise-kafka-addr "PLAINTEXT://localhost:$PORT_KAFKA"

echo "Starting Temporal dev server (docker image) ..."
# 'temporalio/auto-setup' exits at startup unless a Cassandra/Postgres database is
# configured; the dev server runs standalone with in-memory persistence.
docker run -d --name "$NAME_TMP" -p "$PORT_TMP:$PORT_TMP" temporalio/temporal:1.5.1 \
  server start-dev --ip 0.0.0.0

echo "Waiting for services to become ready..."
# A published docker port accepts connections before the service behind it is
# ready, so wait on the services' own readiness instead of nc.
until docker exec "$NAME_RP" rpk cluster health 2>/dev/null | grep -q "Healthy:.*true"; do sleep 1; done
until docker exec "$NAME_TMP" temporal operator cluster health >/dev/null 2>&1; do sleep 1; done

# libs/temporal-client reads TEMPORAL_HOST (or TEMPORAL_ADDRESS / TEMPORAL_HOSTPORT);
# without it the client falls back to the compose hostname 'temporal:7233'.
EXIT_CODE=0
KAFKA_BROKERS="localhost:$PORT_KAFKA" TEMPORAL_HOST="localhost:$PORT_TMP" \
  go run ./backend/cmd/e2e_temporal || EXIT_CODE=$?

exit "$EXIT_CODE"
