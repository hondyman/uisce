#!/bin/bash
# provision_tenant_secrets.sh
# Provisions placeholder secrets in Infisical for a tenant's data source connections.
# Run this before the scanner can successfully authenticate against a new connection.
#
# Usage:
#   PROJECT_ID="..." ENV="dev" ./provision_tenant_secrets.sh <tenant_id> <conn_name> <db_user> [<conn_name> <db_user>...]
#
# Example:
#   INFISICAL_PROJECT_ID="9af25976-bafc-4895-a057-2effec13c620" \
#   INFISICAL_ENVIRONMENT="dev" \
#     ./provision_tenant_secrets.sh \
#       "99e99e99-99e9-49e9-89e9-99e99e99e999" \
#       "northwinds" "postgres" \
#       "trade" "wealth_user"
#
set -euo pipefail

PROJECT_ID="${INFISICAL_PROJECT_ID:?required}"
ENV="${INFISICAL_ENVIRONMENT:-dev}"

if [ $# -lt 3 ]; then
  echo "Usage: $0 <tenant_id> <conn_name> <db_user> [<conn_name> <db_user>...]"
  echo "  tenant_id   - UUID of the tenant (used to construct the Infisical path)"
  echo "  conn_name   - Name of the connection (used in the Infisical path)"
  echo "  db_user     - Database username for this connection"
  echo ""
  echo "Environment variables:"
  echo "  INFISICAL_PROJECT_ID    - Infisical project ID (required)"
  echo "  INFISICAL_ENVIRONMENT   - Infisical environment (default: dev)"
  exit 1
fi

TENANT_ID="$1"
shift

provision_connection_secret() {
  local conn_name="$1"
  local db_user="$2"
  local secret_path="/connections/tenant-${TENANT_ID}/${conn_name}"

  echo "[*] Provisioning secrets at ${secret_path}"

  # Create the folder path if it doesn't exist
  infisical secrets folders create "${secret_path}" \
    --env="${ENV}" \
    --projectId="${PROJECT_ID}" 2>/dev/null || true

  # Set DB_USERNAME
  infisical secrets set "DB_USERNAME=${db_user}" \
    --env="${ENV}" \
    --path="${secret_path}" \
    --projectId="${PROJECT_ID}"

  # Set DB_PASSWORD as placeholder — operator must replace with real value
  infisical secrets set "DB_PASSWORD=TODO_SET_LIVE_PASSWORD" \
    --env="${ENV}" \
    --path="${secret_path}" \
    --projectId="${PROJECT_ID}"

  echo "[+] ${secret_path}/DB_USERNAME = ${db_user}"
  echo "[+] ${secret_path}/DB_PASSWORD = TODO_SET_LIVE_PASSWORD"
  echo ""
}

if [ $# -eq 2 ]; then
  provision_connection_secret "$1" "$2"
else
  while [ $# -ge 2 ]; do
    provision_connection_secret "$1" "$2"
    shift 2
  done
fi

echo "[+] All connection secrets provisioned for tenant ${TENANT_ID}"
echo "[!] IMPORTANT: Replace TODO_SET_LIVE_PASSWORD with real database passwords in the Infisical UI"
echo "    before running any scans against these connections."
