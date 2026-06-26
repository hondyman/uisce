#!/usr/bin/env bash
# Deploy repository to remote host via SSH. Intended for manual use and for the
# GitHub Actions deploy workflow. Key-based SSH is recommended.

set -euo pipefail

# Load local .env.remote variables if present (REMOTE_HOST, image overrides, secrets)
if [ -f .env.remote ]; then
  set -o allexport
  # shellcheck disable=SC1091
  source .env.remote
  set +o allexport
fi

REMOTE_HOST=${1:-${REMOTE_SSH_HOST:-}}
REMOTE_USER=${2:-${REMOTE_SSH_USER:-ubuntu}}
REMOTE_PATH=${3:-${REMOTE_SSH_PATH:-~/semlayer}}
SSH_PORT=${4:-${REMOTE_SSH_PORT:-22}}

if [ -z "$REMOTE_HOST" ]; then
  echo "Usage: $0 <remote-host> [remote-user] [remote-path] [ssh-port]"
  echo "Or set REMOTE_SSH_HOST / REMOTE_SSH_USER / REMOTE_SSH_PATH env vars."
  exit 2
fi

SSH_OPTS="-o BatchMode=yes -o StrictHostKeyChecking=accept-new -p $SSH_PORT"

echo "Deploying to $REMOTE_USER@$REMOTE_HOST:$REMOTE_PATH (port $SSH_PORT)"

# Remote commands: fetch latest, reset to origin/main, pull images and bring up Trino
REMOTE_CMD=$(cat <<'CMD'
set -euo pipefail
cd "$REMOTE_PATH"
# Ensure repo exists and is a git repo
if [ ! -d .git ]; then
  echo "No git repo at $REMOTE_PATH. Aborting."
  exit 1
fi
# Fetch and hard-reset to origin/main
git fetch origin --prune
git reset --hard origin/main
# Pull docker images and restart trino service
if [ -f docker-compose.remote.yml ]; then
  docker compose -f docker-compose.remote.yml pull || true
  docker compose -f docker-compose.remote.yml up -d trino || true
  docker compose -f docker-compose.remote.yml ps trino || true
else
  echo "docker-compose.remote.yml not found in $REMOTE_PATH"
fi
# Quick Trino healthcheck if mapped to host port 8084 (matches repo example)
if curl -s -f http://localhost:8084/v1/info > /dev/null 2>&1; then
  echo "Trino appears to be running on remote (http://localhost:8084)"
else
  echo "Trino healthcheck failed or Trino not listening on 8084"
fi
CMD
)

ssh $SSH_OPTS ${REMOTE_USER}@${REMOTE_HOST} "$REMOTE_CMD"

echo "Deploy finished."
