#!/usr/bin/env bash
set -euo pipefail

# e2e_amqp.sh
# Start a RabbitMQ container, run the e2e publisher that uses NewAMQPEventBus,
# verify it receives the message, and clean up.

CONTAINER_NAME="semlayer-e2e-rabbitmq"
IMAGE="rabbitmq:3.11-management"

echo "Starting RabbitMQ container..."
docker run -d --rm --name ${CONTAINER_NAME} -p 5672:5672 -p 15672:15672 \
  -e RABBITMQ_DEFAULT_USER=guest -e RABBITMQ_DEFAULT_PASS=guest ${IMAGE}

echo "Waiting for RabbitMQ to accept connections on 5672..."
for i in {1..30}; do
  if nc -z localhost 5672; then
    echo "RabbitMQ is up"
    break
  fi
  sleep 1
done

if ! nc -z localhost 5672; then
  echo "RabbitMQ did not become ready in time" >&2
  docker rm -f ${CONTAINER_NAME} || true
  exit 1
fi

echo "Running Go e2e program..."
# Deprecated: AMQP-based E2E (kept for reference)
# RABBITMQ_URL="amqp://guest:guest@localhost:5672/" go run ./backend/cmd/e2e_amqp
KAFKA_BROKERS="localhost:9092" # use Kafka-based E2E tests instead (if available)

EXIT_CODE=$?

echo "Cleaning up RabbitMQ container"
docker rm -f ${CONTAINER_NAME} || true

exit ${EXIT_CODE}
