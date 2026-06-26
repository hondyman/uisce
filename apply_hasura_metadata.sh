#!/usr/bin/env bash
# Apply Hasura metadata for alpha (main) and wealth_app (separate source)
# This script DOES NOT run DB migrations. It only applies metadata to the
# Hasura endpoint you point it at. Set the following env vars before running:
#   HASURA_ENDPOINT (default: http://localhost:8080)
#   HASURA_ADMIN_SECRET (default: admin_secret_key)
#   WEALTH_APP_DATABASE_URL (required for adding wealth_app source)
#
# Requirements: hasura CLI installed and on PATH. See install_hasura_cli.sh in
# the repo if you don't have it.

set -euo pipefail

HASURA_ENDPOINT=${HASURA_ENDPOINT:-http://localhost:8080}
HASURA_ADMIN_SECRET=${HASURA_ADMIN_SECRET:-admin_secret_key}

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"

function check_hasura_cli() {
  if ! command -v hasura >/dev/null 2>&1; then
    echo "ERROR: 'hasura' CLI not found in PATH."
    echo "Run ./install_hasura_cli.sh or install the hasura CLI from https://github.com/hasura/graphql-engine/releases"
    exit 2
  fi
}

function apply_project() {
  local project_dir="$1"
  echo "Applying metadata project: ${project_dir} -> ${HASURA_ENDPOINT}"
  hasura metadata apply --project "${project_dir}" --endpoint "${HASURA_ENDPOINT}" --admin-secret "${HASURA_ADMIN_SECRET}"
}

check_hasura_cli

echo "Checking Hasura connectivity and admin secret before applying metadata..."

# If a .env file exists in the repo root and HASURA_* not explicitly set, prompt to source it
if [ -f "${ROOT_DIR}/.env" ]; then
  echo "Found .env in repo root. If you want to load it into this script's environment, press ENTER to load, Ctrl-C to skip."
  read -r -t 5 || true
  # shellcheck disable=SC1090
  set -a; source "${ROOT_DIR}/.env" 2>/dev/null || true; set +a
  # refresh values from env if they were present in .env
  HASURA_ENDPOINT=${HASURA_ENDPOINT:-${HASURA_ENDPOINT}}
  HASURA_ADMIN_SECRET=${HASURA_ADMIN_SECRET:-${HASURA_ADMIN_SECRET}}
fi

# 1) Reachability check (healthz doesn't require admin secret)
if ! curl -fsS --max-time 5 "${HASURA_ENDPOINT}/healthz" >/dev/null; then
  echo "Hasura endpoint ${HASURA_ENDPOINT} not reachable (healthz failed)."
  # If docker compose is available in the repo, try to bring up the hasura service
  if command -v docker >/dev/null 2>&1 && [ -f "${ROOT_DIR}/docker-compose.yml" ]; then
    echo "Attempting to start Hasura with 'docker compose up -d hasura'..."
    if docker compose up -d hasura >/dev/null 2>&1; then
      echo "Started hasura container, waiting up to 60s for readiness..."
      # wait for healthz to succeed
      for i in $(seq 1 12); do
        if curl -fsS --max-time 5 "${HASURA_ENDPOINT}/healthz" >/dev/null; then
          echo "Hasura is healthy."
          break
        fi
        sleep 5
      done
      if ! curl -fsS --max-time 5 "${HASURA_ENDPOINT}/healthz" >/dev/null; then
        echo "ERROR: Hasura still not reachable after starting container. Gathering diagnostic information..."
        echo "--- docker compose ps ---"
        docker compose ps || true
        echo "--- docker ps (hasura-like containers) ---"
        docker ps --filter name=hasura --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' || true
        echo "--- Attempting to show recent logs for hasura containers ---"
        # try to discover likely container names and show logs
        CONTAINERS=$(docker ps -a --filter name=hasura --format '{{.Names}}') || CONTAINERS=""
        if [ -n "${CONTAINERS}" ]; then
          for c in ${CONTAINERS}; do
            echo "--- logs for container: ${c} (last 200 lines) ---"
            docker logs --tail 200 "${c}" || true
          done
        else
          echo "No running containers named 'hasura' found. Showing all containers for context:"
          docker ps -a --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' || true
        fi
        echo "--- End diagnostic ---"
        exit 2
      fi
    else
      echo "Failed to start hasura via docker compose. Please start Hasura manually and re-run this script."
      exit 2
    fi
  else
    echo "ERROR: Hasura endpoint ${HASURA_ENDPOINT} not reachable (healthz failed)."
    echo " - Is Hasura running? Check docker compose or process."
    echo " - Example: docker compose ps | grep hasura"
    exit 2
  fi
fi

# 2) Try a metadata export to validate admin secret / access
TMPDIR=$(mktemp -d)
trap 'rm -rf "${TMPDIR}"' EXIT
echo "Testing admin secret by attempting a metadata export..."
if ! hasura metadata export --endpoint "${HASURA_ENDPOINT}" --admin-secret "${HASURA_ADMIN_SECRET}" --output-dir "${TMPDIR}" >/dev/null 2>&1; then
  echo "\nERROR: Failed to authenticate to Hasura with the provided HASURA_ADMIN_SECRET."
  echo "Hasura returned an 'access-denied' or similar error."
  echo "Checklist to fix:"
  echo "  - Confirm the admin secret matches Hasura's configured admin secret (check docker-compose or environment where Hasura runs)."
  echo "  - If Hasura is running in Docker Compose, inspect the service env (docker compose exec hasura env | grep ADMIN) or docker-compose.yml entry for HASURA_GRAPHQL_ADMIN_SECRET."
  echo "  - You can still open the Hasura Console (if running) and add the wealth_app DB manually (Data → Connect Database) and then re-run this script."
  echo "\nIf you believe the secret is correct, try running one of these commands locally to debug:" 
  echo "  curl -v ${HASURA_ENDPOINT}/v1/version" 
  echo "  curl -v -H \"X-Hasura-Admin-Secret: ${HASURA_ADMIN_SECRET}\" ${HASURA_ENDPOINT}/v1/metadata"
  exit 3
fi

echo "1) Apply main Hasura metadata (alpha / default source) from ./hasura"
apply_project "${ROOT_DIR}/hasura"

echo "\n2) Add/Apply wealth_app metadata (creates wealth_app source using WEALTH_APP_DATABASE_URL)"
if [ -z "${WEALTH_APP_DATABASE_URL:-}" ]; then
  echo "WEALTH_APP_DATABASE_URL not set. To add the wealth_app source automatically, set WEALTH_APP_DATABASE_URL to the connection string (postgres://...)."
  echo "You can still add the wealth_app database manually in Hasura Console (Data → Connect Database)."
  echo "Skipping wealth_app automatic add."
  echo "\nDone. Alpha metadata applied."
  exit 0
fi

export WEALTH_APP_DATABASE_URL

apply_project "${ROOT_DIR}/hasura_wealth_app"

echo "\nDone. Metadata applied. Verify in Hasura Console: ${HASURA_ENDPOINT} (admin secret respected)."
