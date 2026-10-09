#!/usr/bin/env bash
# Deploys the Kafka Connect Iceberg bronze sink to a remote host.
# Idempotent: connector registration uses PUT (update-or-create).
#
# Usage:
#   deploy-bronze.sh <host> <user> <path> <port> <image_digest>
#
# Required secrets (GitHub Actions):
#   REMOTE_SSH_HOST, REMOTE_SSH_USER, REMOTE_SSH_PORT, REMOTE_SSH_PATH,
#   REMOTE_SSH_PRIVATE_KEY (loaded into ssh-agent by the workflow)
#   GHCR_TOKEN (environment, for the image pull; the workflow passes GITHUB_TOKEN)
#
# What this script does NOT need sudo for:
#   - docker build / docker compose up   → docker group membership (not root)
#   - writing to the deploy directory    → owned by deploy user
#   - CA cert placement                  → owned by deploy user, mounted into container
#
# The one-time privileged step (run by a human, once):
#   sudo useradd -m -s /bin/bash deploy
#   sudo usermod -aG docker deploy
#   sudo mkdir -p /mnt/uisce && sudo chown -R deploy:deploy /mnt/uisce

set -euo pipefail

HOST=${1:?Usage: $0 <host> <user> <path> <port> <image_digest>}
USER=${2:?Usage: $0 <host> <user> <path> <port> <image_digest>}
PATH_ARG=${3:?Usage: $0 <host> <user> <path> <port> <image_digest>}
PORT=${4:?Usage: $0 <host> <user> <path> <port> <image_digest>}
IMAGE_DIGEST=${5:?Usage: $0 <host> <user> <path> <port> <image_digest>}

SSH_OPTS="-o BatchMode=yes -o StrictHostKeyChecking=accept-new -p $PORT"
REGISTRY="ghcr.io"
REPO=${6:?Usage: $0 <host> <user> <path> <port> <image_digest> <repo>}  # e.g. "hondyman/uisce"
IMAGE="${REGISTRY}/${REPO}/kafka-connect-iceberg"
: "${GHCR_TOKEN:?GHCR_TOKEN must be set (the workflow passes GITHUB_TOKEN)}"

echo "==> Deploying bronze sink to ${USER}@${HOST}:${PATH_ARG}"
echo "    Image: ${IMAGE}@${IMAGE_DIGEST}"

REMOTE_CMD=$(cat <<'INNER_EOF'
set -euo pipefail

HOST='{{HOST}}'
USER='{{USER}}'
PATH_ARG='{{PATH_ARG}}'
PORT='{{PORT}}'
IMAGE_DIGEST='{{IMAGE_DIGEST}}'
REGISTRY='{{REGISTRY}}'
REPO='{{REPO}}'
IMAGE="${REGISTRY}/${REPO}/kafka-connect-iceberg"
CA_CERT_DST="$PATH_ARG/certs/keycloak-ca.crt"
GITHUB_REPO="https://github.com/hondyman/uisce.git"

echo "==> Deploying bronze sink to ${USER}@${HOST}:${PATH_ARG}"

# 1. Ensure repo exists at target path — clone if missing
if [ ! -d "$PATH_ARG/.git" ]; then
  echo "==> No git repo found at $PATH_ARG — cloning fresh"
  mkdir -p "$(dirname "$PATH_ARG")"
  git clone "$GITHUB_REPO" "$PATH_ARG"
else
  echo "==> Git repo found — fetching latest origin/main"
  cd "$PATH_ARG"
  git fetch origin --prune
  git reset --hard origin/main
fi

cd "$PATH_ARG"

# 2. Extract Keycloak CA into the compose bind-mount directory ($PATH_ARG/certs, owned by deploy)
echo "==> Fetching Keycloak CA from ${HOST}:8443"
mkdir -p "$(dirname "$CA_CERT_DST")"
openssl s_client -connect "${HOST}:8443" </dev/null 2>/dev/null | \
  openssl x509 -outform PEM -out "$CA_CERT_DST"
chmod 0444 "$CA_CERT_DST"
echo "    CA cert saved to $CA_CERT_DST ($(wc -l < "$CA_CERT_DST") lines)"

# 3. Pull image by digest and tag as bronze-latest (avoids depending on a moving tag)
#    The token arrives on stdin (see the ssh line below), so it never appears in argv.
echo "==> Logging in to ${REGISTRY}"
printf '%s' "$GHCR_TOKEN" | docker login "$REGISTRY" -u "${REPO%%/*}" --password-stdin >/dev/null
echo "==> Pulling image ${IMAGE}@${IMAGE_DIGEST}"
docker pull "${IMAGE}@${IMAGE_DIGEST}" || {
  echo "ERROR: docker pull failed — check GHCR_TOKEN has packages:read permission"
  docker logout "$REGISTRY" >/dev/null 2>&1 || true
  exit 1
}
docker logout "$REGISTRY" >/dev/null
docker tag "${IMAGE}@${IMAGE_DIGEST}" "${IMAGE}:bronze-latest"
echo "    Tagged as ${IMAGE}:bronze-latest"

# 4. Start the kafka-connect-iceberg container
echo "==> Starting kafka-connect-iceberg container"
docker compose -f docker-compose.remote.yml up -d kafka-connect-iceberg

# 5. Wait for worker to be healthy (REST API responding)
echo "==> Waiting for Kafka Connect worker to be healthy"
HEALTHY=0
for i in $(seq 1 30); do
  if curl -s -f http://localhost:8084/ > /dev/null 2>&1; then
    echo "    Worker healthy after ${i}s"
    HEALTHY=1
    break
  fi
  echo "    Waiting... ($i/30)"
  sleep 2
done
if [ "$HEALTHY" -eq 0 ]; then
  echo "ERROR: Kafka Connect worker did not become healthy within 60s"
  docker logs kafka-connect-iceberg 2>&1 | tail -30
  exit 1
fi

# 6. Register (or update) the connector — PUT is idempotent, no 409 on re-run
echo "==> Registering connector iceberg-bronze-sink"
REGISTER_RESP=$(curl -s -X PUT http://localhost:8084/connectors/iceberg-bronze-sink/config \
  -H 'Content-Type: application/json' \
  -d @infrastructure/iceberg-bronze-sink.json)
echo "    $REGISTER_RESP" | head -c 200

# 7. Wait for connector to reach RUNNING state (connector AND tasks)
echo "==> Waiting for connector to reach RUNNING"
CONNECTOR_RUNNING=0
for i in $(seq 1 30); do
  STATUS=$(curl -s http://localhost:8084/connectors/iceberg-bronze-sink/status)
  C_STATE=$(echo "$STATUS" | python3 -c "import sys,json; print(json.load(sys.stdin)['connector']['state'])" 2>/dev/null || echo "UNKNOWN")
  T_STATES=$(echo "$STATUS" | python3 -c "import sys,json; print(json.load(sys.stdin)['tasks'])" 2>/dev/null || echo "[]")
  echo "    [${i}/30] connector=${C_STATE}"

  if [ "$C_STATE" = "RUNNING" ]; then
    # Check all tasks are also RUNNING
    ALL_RUNNING=1
    for TASK_STATE in $(echo "$T_STATES" | python3 -c "import sys,json; print([t['state'] for t in json.load(sys.stdin)])" 2>/dev/null); do
      if [ "$TASK_STATE" != "RUNNING" ]; then
        ALL_RUNNING=0
      fi
    done
    if [ "$ALL_RUNNING" -eq 1 ]; then
      echo "    Connector and all tasks RUNNING"
      CONNECTOR_RUNNING=1
      break
    fi
  fi
  sleep 3
done

if [ "$CONNECTOR_RUNNING" -eq 0 ]; then
  echo "ERROR: connector did not reach RUNNING state"
  curl -s http://localhost:8084/connectors/iceberg-bronze-sink/status | python3 -m json.tool
  docker logs kafka-connect-iceberg 2>&1 | grep -i 'error\|exception\|failed' | tail -20
  exit 1
fi

echo ""
echo "==> Deploy complete"
echo "    Worker:  http://localhost:8084"
echo "    Connector: iceberg-bronze-sink (RUNNING)"
echo "    Next: three-probe — insert/update/delete one row in Postgres, verify op: c/u/d in Bronze"
INNER_EOF
)

# Substitute placeholders
REMOTE_CMD="${REMOTE_CMD//\{\{HOST\}\}/$HOST}"
REMOTE_CMD="${REMOTE_CMD//\{\{USER\}\}/$USER}"
REMOTE_CMD="${REMOTE_CMD//\{\{PATH_ARG\}\}/$PATH_ARG}"
REMOTE_CMD="${REMOTE_CMD//\{\{PORT\}\}/$PORT}"
REMOTE_CMD="${REMOTE_CMD//\{\{IMAGE_DIGEST\}\}/$IMAGE_DIGEST}"
REMOTE_CMD="${REMOTE_CMD//\{\{REGISTRY\}\}/$REGISTRY}"
REMOTE_CMD="${REMOTE_CMD//\{\{REPO\}\}/$REPO}"

# The GHCR token goes over stdin, not argv, so it stays out of the remote process list.
printf '%s\n' "$GHCR_TOKEN" | ssh $SSH_OPTS "${USER}@${HOST}" "read -r GHCR_TOKEN; export GHCR_TOKEN; $REMOTE_CMD"

echo "==> Remote deploy finished."
