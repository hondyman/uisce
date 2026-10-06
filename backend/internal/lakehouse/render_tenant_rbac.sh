#!/usr/bin/env bash
# Renders tenant_rbac.sql.tpl with env secrets and applies it to StarRocks FE.
# Templates and tenants.env.example are committed; tenants.env and *.rendered.sql are not.
#
# Usage:
#   cp tenants.env.example tenants.env && $EDITOR tenants.env   # fill real secrets
#   ./render_tenant_rbac.sh                                      # applies via 127.0.0.1:9030
#   FE_HOST=starrocks-fe ./render_tenant_rbac.sh                 # from inside docker network
#   docker exec -i starrocks-fe bash < render_tenant_rbac.sh     # run inside the FE container
#
# Refuses to run if any tenant password is empty — guards against recreating the
# empty-root situation this whole script is meant to fix.

set -euo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
ENV_FILE="${ENV_FILE:-$DIR/tenants.env}"
TPL="$DIR/tenant_rbac.sql.tpl"
OUT="/tmp/tenant_rbac.rendered.sql"   # rendered output: never commit, removed after apply

if [[ ! -f "$ENV_FILE" ]]; then
  echo "ERROR: env file not found: $ENV_FILE" >&2
  echo "       cp tenants.env.example tenants.env && edit it" >&2
  exit 1
fi

# shellcheck disable=SC1090
source "$ENV_FILE"

# Refuse empty secrets — this is the guard against another empty-root situation.
missing=0
for var in TENANT_NORTHWINDS_PASSWORD TENANT_CRD_BAKEOFF_PASSWORD STREAM_LOADER_PASSWORD; do
  if [[ -z "${!var:-}" ]]; then
    echo "ERROR: $var is empty in $ENV_FILE" >&2
    missing=1
  fi
done
[[ $missing -eq 0 ]] || exit 1

if [[ -z "${FE_HOST:-}" || -z "${FE_QUERY_PORT:-}" ]]; then
  echo "ERROR: FE_HOST and FE_QUERY_PORT must be set in $ENV_FILE" >&2
  exit 1
fi

if ! command -v envsubst >/dev/null 2>&1; then
  echo "ERROR: envsubst (gettext-base) is required but not installed" >&2
  exit 1
fi

if ! command -v mysql >/dev/null 2>&1; then
  echo "ERROR: mysql client is required but not installed" >&2
  exit 1
fi

# Always remove the rendered file on exit, success or failure — never leave
# plaintext passwords on disk.
trap 'rm -f "$OUT"' EXIT

envsubst '${TENANT_NORTHWINDS_PASSWORD} ${TENANT_CRD_BAKEOFF_PASSWORD} ${STREAM_LOADER_PASSWORD}' \
  < "$TPL" > "$OUT"
chmod 600 "$OUT"

# Apply. STARROCKS_ROOT_PASSWORD is optional: empty means the cluster still has
# the no-password root user (the original situation). The render script does
# NOT rotate root — that happens in a follow-up pass once the new infra is in
# place, per the comment block in the template.
#
# When the password is empty we MUST omit -p entirely: this server's root auth
# accepts no-password connections and rejects `mysql -uroot -p""` (different
# protocol path: empty-string vs absent-password).
if [[ -n "${STARROCKS_ROOT_PASSWORD:-}" ]]; then
  mysql -h "${FE_HOST}" -P "${FE_QUERY_PORT}" -u root -p"${STARROCKS_ROOT_PASSWORD}" < "$OUT"
else
  mysql -h "${FE_HOST}" -P "${FE_QUERY_PORT}" -u root < "$OUT"
fi

echo "Tenant RBAC + workload groups applied to ${FE_HOST}:${FE_QUERY_PORT}."