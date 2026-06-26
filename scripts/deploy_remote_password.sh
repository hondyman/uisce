#!/usr/bin/env bash
# Deploy repository to remote host using password-based SSH (sshpass).
# NOTE: Password-based auth is less secure than key-based. Use only if necessary.

set -euo pipefail

# Load local .env.remote variables if present (REMOTE_HOST, image overrides, secrets)
if [ -f .env.remote ]; then
  set -o allexport
  # shellcheck disable=SC1091
  source .env.remote
  set +o allexport
fi

REMOTE_HOST=${1:-}
REMOTE_PASS=${2:-}
REMOTE_USER=${3:-ubuntu}
REMOTE_PATH=${4:-~/semlayer}
SSH_PORT=${5:-22}

if [ -z "$REMOTE_HOST" ] || [ -z "$REMOTE_PASS" ]; then
  echo "Usage: $0 <remote-host> <password> [remote-user] [remote-path] [ssh-port]"
  exit 2
fi

# Ensure sshpass available
if ! command -v sshpass >/dev/null 2>&1; then
  echo "sshpass not found. Attempting to install..."
  if command -v apt-get >/dev/null 2>&1; then
    sudo apt-get update && sudo apt-get install -y sshpass
  elif command -v brew >/dev/null 2>&1; then
    brew install hudochenkov/sshpass/sshpass || true
    echo "If brew install failed, install sshpass manually (macOS may require special taps)."
  else
    echo "Please install sshpass (apt-get or brew) and re-run this script."; exit 1
  fi
fi

SSH_OPTS="-o StrictHostKeyChecking=accept-new -p $SSH_PORT"

REMOTE_CMD=$(cat <<'CMD'
set -euo pipefail
cd "$REMOTE_PATH"
if [ ! -d .git ]; then
  echo "No git repo at $REMOTE_PATH. Aborting."; exit 1
fi
# Fetch and hard reset to origin/main
git fetch origin --prune
git reset --hard origin/main
# Pull and start Trino
if [ -f docker-compose.remote.yml ]; then
  docker compose -f docker-compose.remote.yml pull || true
  docker compose -f docker-compose.remote.yml up -d trino || true
  docker compose -f docker-compose.remote.yml ps trino || true
else
  echo "docker-compose.remote.yml not found"
fi
if curl -s -f http://localhost:8084/v1/info >/dev/null 2>&1; then
  echo "Trino appears to be running on remote (http://localhost:8084)"
else
  echo "Trino healthcheck failed or Trino not listening on 8084"
fi
CMD
)

sshpass -p "$REMOTE_PASS" ssh $SSH_OPTS ${REMOTE_USER}@${REMOTE_HOST} "$REMOTE_CMD"

echo "Password-based deploy finished."
