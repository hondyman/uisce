#!/usr/bin/env bash
set -euo pipefail

# Simple Redpanda smoke test for local use and CI
# - Starts a temporary Redpanda container (if none exists)
# - Creates a topic
# - Produces a JSON message
# - Consumes the message and verifies it contains the key

IMAGE="docker.redpanda.com/redpandadata/redpanda:latest"
CONTAINER_NAME="${REDPANDA_CONTAINER:-semlayer-redpanda-ci}"
# If REDPANDA_CONTAINER is not supplied we will start and clean up our own container
OWN_CONTAINER=1
if [ -n "${REDPANDA_CONTAINER:-}" ]; then
  OWN_CONTAINER=0
fi
TOPIC="semlayer-smoke-$(date +%s)"
# If we are going to start our own container, default to binding host ports so Pandaproxy is available
if [ "$OWN_CONTAINER" -eq 1 ] && [ -z "${BIND_HOST_PORTS+x}" ]; then
  BIND_HOST_PORTS=1
fi

cleanup() {
  if [ "$OWN_CONTAINER" -eq 1 ]; then
    if docker ps -a --format '{{.Names}}' | grep -q "${CONTAINER_NAME}"; then
      echo "Stopping and removing ${CONTAINER_NAME}..."
      docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
    fi
  else
    echo "Not cleaning up user-provided container ${CONTAINER_NAME}"
  fi
}
trap cleanup EXIT

# Pull image
echo "Pulling Redpanda image ${IMAGE}..."
docker pull "${IMAGE}"

# Start container without binding host ports (avoids conflicts when ports already in use)
# To bind host ports for manual testing, set BIND_HOST_PORTS=1 in the environment
if [ "$OWN_CONTAINER" -eq 1 ]; then
  if [ "${BIND_HOST_PORTS:-0}" -eq 1 ]; then
    echo "Starting Redpanda container with host port binding (may fail if ports are busy)..."
    if ! docker run -d --name "${CONTAINER_NAME}" --rm -p 9092:9092 -p 8082:8082 "${IMAGE}" \
      redpanda start --overprovisioned --smp 1 --memory 1G --reserve-memory 0M --check=false >/dev/null; then
      echo "Host ports are already allocated — retrying without host port bindings"
      docker run -d --name "${CONTAINER_NAME}" --rm "${IMAGE}" \
        redpanda start --overprovisioned --smp 1 --memory 1G --reserve-memory 0M --check=false
    fi
  else
    docker run -d --name "${CONTAINER_NAME}" --rm "${IMAGE}" \
      redpanda start --overprovisioned --smp 1 --memory 1G --reserve-memory 0M --check=false
  fi
else
  echo "Using existing Redpanda container: ${CONTAINER_NAME}"
fi

# Wait for cluster to be healthy
echo "Waiting for Redpanda to be ready..."
ATTEMPTS=0
MAX=60
until docker exec "${CONTAINER_NAME}" rpk cluster info >/dev/null 2>&1; do
  ATTEMPTS=$((ATTEMPTS+1))
  if [ "$ATTEMPTS" -ge "$MAX" ]; then
    echo "Redpanda did not become ready in time" >&2
    docker logs "${CONTAINER_NAME}" --tail 100 || true
    exit 1
  fi
  sleep 1
done

# Create topic
echo "Creating topic ${TOPIC}..."
docker exec "${CONTAINER_NAME}" rpk topic create "${TOPIC}"

# Produce and consume test message using Pandaproxy (HTTP) if available
# This avoids broker leader/partition placement issues when using rpk directly.
if docker exec "${CONTAINER_NAME}" sh -c 'command -v curl >/dev/null 2>&1'; then
  echo "Pandaproxy (curl) detected in container — using HTTP producer/consumer"

  echo "Producing test message via Pandaproxy..."
  RESP=$(docker exec -i "${CONTAINER_NAME}" sh -lc "curl -s -w '::%{http_code}' -X POST -H 'Content-Type: application/vnd.kafka.json.v2+json' --data-binary @- 'http://localhost:8082/v1/topics/${TOPIC}'" <<'JSON'
{"records":[{"key":"smoke","value":{"msg":"hello from smoke test"}}]}
JSON
)
  HTTP_CODE=${RESP##*::}
  BODY=${RESP%::*}
  if [ "${HTTP_CODE}" != "200" ] && [ "${HTTP_CODE}" != "202" ]; then
    echo "Pandaproxy produce failed (HTTP ${HTTP_CODE}): ${BODY}"

    # Try to detect Pandaproxy on other running containers (console/pandaproxy container)
    echo "Scanning other containers for Pandaproxy..."
    PANDAPROXY_CONTAINER=""
    for cand in $(docker ps --format '{{.Names}}' | grep -E 'redpanda|console|pandaproxy' || true); do
      echo "Checking $cand..."
      RESP2=$(docker exec "$cand" sh -lc "curl -s -w '::%{http_code}' -X POST -H 'Content-Type: application/vnd.kafka.json.v2+json' --data-binary '{\"records\":[{\"key\":\"probe\",\"value\":{}}]}' 'http://localhost:8082/v1/topics/${TOPIC}'" 2>/dev/null || true)
      CODE2=${RESP2##*::}
      BODY2=${RESP2%::*}
      if [ "$CODE2" = "200" ] || [ "$CODE2" = "202" ]; then
        echo "Found Pandaproxy on container: $cand (HTTP $CODE2)"
        PANDAPROXY_CONTAINER="$cand"
        break
      fi
    done

    if [ -n "$PANDAPROXY_CONTAINER" ]; then
      echo "Will use Pandaproxy on $PANDAPROXY_CONTAINER for produce/consume"
      PANDAPROXY_HOST_CONTAINER="$PANDAPROXY_CONTAINER"
      USE_RPK_FALLBACK=0
    else
      echo "Pandaproxy not found on other containers — falling back to rpk CLI"
      USE_RPK_FALLBACK=1
    fi
  else
    USE_RPK_FALLBACK=0
  fi

  # Create a temporary consumer instance
  GROUP="smoke-test-group-$(date +%s)"
  INSTANCE="smoke"

  if [ "${USE_RPK_FALLBACK:-0}" -eq 1 ]; then
    echo "Using rpk fallback path"
  else
    echo "Creating Pandaproxy consumer group ${GROUP} (instance ${INSTANCE})..."
    docker exec "${CONTAINER_NAME}" sh -lc "curl -s -X POST -H 'Content-Type: application/vnd.kafka.v2+json' --data '{\"name\":\"${INSTANCE}\",\"format\":\"json\",\"auto.offset.reset\":\"earliest\"}' 'http://localhost:8082/consumers/${GROUP}'" >/dev/null || true

    echo "Subscribing consumer to topic ${TOPIC}..."
    docker exec "${CONTAINER_NAME}" sh -lc "curl -s -X POST -H 'Content-Type: application/vnd.kafka.v2+json' --data '{\"topics\":[\"${TOPIC}\"]}' 'http://localhost:8082/consumers/${GROUP}/instances/${INSTANCE}/subscription'" >/dev/null || true

    echo "Consuming message (via Pandaproxy records API)..."
    set +e
    FOUND=0
    RECV=""
    # If Pandaproxy is on a different container, use that container for records fetch
    RECORDS_CONTAINER="${PANDAPROXY_HOST_CONTAINER:-${CONTAINER_NAME}}"
    for i in $(seq 1 12); do
      RECV=$(docker exec "${RECORDS_CONTAINER}" sh -lc "curl -s -X GET -H 'Accept: application/vnd.kafka.json.v2+json' 'http://localhost:8082/consumers/${GROUP}/instances/${INSTANCE}/records'") || true
      if [ -n "${RECV}" ] && echo "${RECV}" | grep -q 'hello from smoke test'; then
        FOUND=1
        break
      fi
      echo "Attempt $i: no records yet, retrying..."
      sleep 1
    done
    set -e

    # Cleanup consumer instance (using the container we created the consumer on)
    docker exec "${RECORDS_CONTAINER}" sh -lc "curl -s -X DELETE 'http://localhost:8082/consumers/${GROUP}/instances/${INSTANCE}'" >/dev/null || true

    if [ "$FOUND" -eq 1 ]; then
      echo "Received record: ${RECV}"
      echo "Smoke test PASSED"
      exit 0
    else
      echo "Failed to consume message via Pandaproxy after retries" >&2
      docker exec "${CONTAINER_NAME}" rpk topic describe "${TOPIC}" || true
      echo "Falling back to rpk CLI for produce/consume"
      USE_RPK_FALLBACK=1
    fi
  fi

  if [ "${USE_RPK_FALLBACK:-0}" -eq 1 ]; then
    echo "Producing test message via rpk..."
    docker exec -i "${CONTAINER_NAME}" rpk topic produce "${TOPIC}" -k smoke -f '%v{json}\n' <<'EOF'
{"msg":"hello from smoke test"}
EOF

    echo "Consuming message..."
    set +e
    OUT=""
    RC=1
    for i in $(seq 1 6); do
      OUT=$(docker exec "${CONTAINER_NAME}" rpk topic consume "${TOPIC}" -o start -n 1 --fetch-max-wait 5s -f '%k %v\n' 2>/dev/null)
      RC=$?
      if [ $RC -eq 0 ] && [ -n "${OUT}" ]; then
        break
      fi
      echo "Attempt $i: message not found yet, retrying..."
      sleep 1
    done
    set -e

    if [ $RC -ne 0 ] || [ -z "${OUT}" ]; then
      echo "Failed to consume message after retries" >&2
      docker exec "${CONTAINER_NAME}" rpk topic describe "${TOPIC}" || true
      exit 1
    fi

    echo "Consume output: ${OUT}"
    if echo "${OUT}" | grep -q 'smoke'; then
      echo "Smoke test PASSED"
      exit 0
    else
      echo "Smoke test FAILED: unexpected output" >&2
      exit 1
    fi
  fi
else
  echo "Pandaproxy curl not found inside container — falling back to rpk CLI"

  echo "Producing test message via rpk..."
  docker exec -i "${CONTAINER_NAME}" rpk topic produce "${TOPIC}" -k smoke -f '%v{json}\n' <<'EOF'
{"msg":"hello from smoke test"}
EOF

  echo "Consuming message..."
  set +e
  OUT=""
  RC=1
  CONSUMED_FROM=""
  for i in $(seq 1 6); do
    # Try consuming from the target container first; if that fails, try brokers' IPs directly
    for c in $(docker ps --format '{{.Names}}' | grep -E 'semlayer-redpanda' || true); do
      ip=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' "$c" 2>/dev/null || true)
      if [ -z "$ip" ]; then
        continue
      fi
      OUT=$(docker exec "${CONTAINER_NAME}" rpk topic consume "${TOPIC}" -b "${ip}:9092" -o start -n 1 --fetch-max-wait 5s -f '%k %v\n' 2>/dev/null) || true
      if [ -n "${OUT}" ]; then
        RC=0
        CONSUMED_FROM=${c}
        break 2
      fi
    done
    echo "Attempt $i: message not found yet from any broker, retrying..."
    sleep 1
  done
  set -e

  if [ $RC -ne 0 ] || [ -z "${OUT}" ]; then
    echo "Failed to consume message after retries" >&2
    docker exec "${CONTAINER_NAME}" rpk topic describe "${TOPIC}" || true
    exit 1
  fi

  echo "Consume output (from ${CONSUMED_FROM}): ${OUT}"
  if echo "${OUT}" | grep -q 'smoke'; then
    echo "Smoke test PASSED"
    exit 0
  else
    echo "Smoke test FAILED: unexpected output" >&2
    exit 1
  fi
fi
