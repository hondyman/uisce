#!/usr/bin/env bash
# Onboards one tenant through the existing provisioning saga:
#   POST /api/system/tenants/provision, then polls the run until it finishes.
#
# The saga runs inside the shared platform. It creates the tenant's records and database
# on the shared cluster. It starts no containers for the tenant.
#
# The caller must be a global admin. The token comes from the environment, never an argument,
# so it stays out of shell history and the process list.
#
# Usage:
#   UISCE_API_URL=https://<host> UISCE_ADMIN_TOKEN=... \
#     scripts/onboard-tenant.sh --tenant-name "Acme Ltd" --instance-name "acme-dev" \
#       [--tenant-code acme] [--app orm] [--structure-from-gold-copy] [--dry-run]
#
# Exit codes: 0 completed, 1 request or run failed, 2 bad usage, 3 timed out.

set -euo pipefail

TENANT_NAME=""
INSTANCE_NAME=""
TENANT_CODE=""
APP=""
STRUCTURE_FROM_GOLD_COPY=false
DRY_RUN=false
TIMEOUT_SECONDS=1800
POLL_SECONDS=10

usage() {
  sed -n '2,16p' "$0" | sed 's/^# \{0,1\}//'
}

while [ $# -gt 0 ]; do
  case "$1" in
    --tenant-name)            TENANT_NAME=${2:-}; shift 2 ;;
    --instance-name)          INSTANCE_NAME=${2:-}; shift 2 ;;
    --tenant-code)            TENANT_CODE=${2:-}; shift 2 ;;
    --app)                    APP=${2:-}; shift 2 ;;
    --structure-from-gold-copy) STRUCTURE_FROM_GOLD_COPY=true; shift ;;
    --dry-run)                DRY_RUN=true; shift ;;
    -h|--help)                usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
done

if [ -z "$TENANT_NAME" ] || [ -z "$INSTANCE_NAME" ]; then
  echo "--tenant-name and --instance-name are required" >&2
  usage >&2
  exit 2
fi

BODY=$(jq -n \
  --arg tenant_name "$TENANT_NAME" \
  --arg instance_name "$INSTANCE_NAME" \
  --arg tenant_code "$TENANT_CODE" \
  --arg app "$APP" \
  --argjson structure "$STRUCTURE_FROM_GOLD_COPY" \
  '{tenant_name: $tenant_name, instance_name: $instance_name}
   + (if $tenant_code != "" then {tenant_code: $tenant_code} else {} end)
   + (if $app != "" then {app: $app} else {} end)
   + (if $structure then {structure_from_gold_copy: true} else {} end)')

if [ "$DRY_RUN" = true ]; then
  echo "$BODY"
  exit 0
fi

: "${UISCE_API_URL:?set UISCE_API_URL to the platform base URL}"
: "${UISCE_ADMIN_TOKEN:?set UISCE_ADMIN_TOKEN to a global admin bearer token}"

# curl reads the auth header from stdin, so the token is never on a command line.
api() {
  local method=$1 path=$2 data=${3:-}
  if [ -n "$data" ]; then
    printf 'header = "Authorization: Bearer %s"\nheader = "Content-Type: application/json"\n' "$UISCE_ADMIN_TOKEN" \
      | curl -sS --config - -X "$method" -d "$data" -w '\n%{http_code}' "$UISCE_API_URL$path"
  else
    printf 'header = "Authorization: Bearer %s"\n' "$UISCE_ADMIN_TOKEN" \
      | curl -sS --config - -X "$method" -w '\n%{http_code}' "$UISCE_API_URL$path"
  fi
}

# api prints the body, then the status code on the last line.
split_response() {
  RESP_CODE=$(printf '%s' "$1" | tail -n1)
  RESP_BODY=$(printf '%s' "$1" | sed '$d')
}

echo "==> Requesting provisioning for tenant '$TENANT_NAME' (instance '$INSTANCE_NAME')"
split_response "$(api POST /api/system/tenants/provision "$BODY")"
if [ "$RESP_CODE" -lt 200 ] || [ "$RESP_CODE" -ge 300 ]; then
  echo "ERROR: provisioning request refused (HTTP $RESP_CODE): $RESP_BODY" >&2
  exit 1
fi

WORKFLOW_ID=$(printf '%s' "$RESP_BODY" | jq -r '.workflow_id')
TENANT_ID=$(printf '%s' "$RESP_BODY" | jq -r '.tenant_id')
INSTANCE_ID=$(printf '%s' "$RESP_BODY" | jq -r '.instance_id')
echo "    workflow_id=$WORKFLOW_ID"
echo "    tenant_id=$TENANT_ID instance_id=$INSTANCE_ID"

echo "==> Waiting for the run to finish (timeout ${TIMEOUT_SECONDS}s)"
DEADLINE=$(( $(date +%s) + TIMEOUT_SECONDS ))
while :; do
  split_response "$(api GET "/api/system/tenants/$TENANT_ID/provision/$WORKFLOW_ID" || printf '\n000')"
  if [ "$RESP_CODE" = "200" ]; then
    STATUS=$(printf '%s' "$RESP_BODY" | jq -r '.status')
    echo "    status=$STATUS"
    case "$STATUS" in
      completed)
        DB=$(printf '%s' "$RESP_BODY" | jq -r '.database_name // empty')
        echo "==> Tenant onboarded: tenant_id=$TENANT_ID instance_id=$INSTANCE_ID database=${DB:-n/a}"
        exit 0 ;;
      failed|canceled|terminated)
        ERR=$(printf '%s' "$RESP_BODY" | jq -r '.error // "no error text"')
        echo "ERROR: provisioning $STATUS: $ERR" >&2
        exit 1 ;;
    esac
  else
    echo "    status check returned HTTP $RESP_CODE; retrying"
  fi
  if [ "$(date +%s)" -ge "$DEADLINE" ]; then
    echo "ERROR: timed out waiting for workflow $WORKFLOW_ID" >&2
    exit 3
  fi
  sleep "$POLL_SECONDS"
done
