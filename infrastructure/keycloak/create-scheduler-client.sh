#!/usr/bin/env bash
#
# create-scheduler-client.sh — Provisions a Keycloak service account for an
# enterprise scheduler (Tidal, Control-M, AutoSys, ...) to drive one tenant's
# Uisce schedules through the trigger API / uisce-job CLI.
#
# Creates, idempotently:
#   * realm roles uisce_service_account (marks machine callers: read and
#     trigger only) and schedule_trigger (may trigger)
#   * a confidential client with ONLY the client-credentials grant
#     (no browser login, no password grant, no implicit flow)
#   * a hardcoded tenant_id claim on the client's tokens: the tenant is fixed
#     by Keycloak, never chosen by the caller
#   * both roles on the client's service-account user
# The client secret is written to --secret-out (mode 0600) or to Infisical
# (--infisical-path); it is never printed.
#
# Usage:
#   ./create-scheduler-client.sh --tenant-id <uuid> --client-id tidal-<tenant> \
#       --secret-out ~/.uisce/tidal-<tenant>.secret [--dry-run]
#   ./create-scheduler-client.sh --tenant-id <uuid> --client-id tidal-<tenant> \
#       --infisical-path /schedulers [--dry-run]
#
# Reads (from env, then from repo-root .env if present):
#   KEYCLOAK_HOST (default 100.84.50.65), KEYCLOAK_PORT (default 8443),
#   KEYCLOAK_ADMIN, KEYCLOAK_ADMIN_PASS, KEYCLOAK_REALM (default uisce),
#   KEYCLOAK_INSECURE_SKIP_VERIFY (default true, dev only)
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
ENV_FILE="${REPO_ROOT}/.env"

TENANT_ID="" CLIENT_ID="" SECRET_OUT="" INFISICAL_PATH="" DRY_RUN=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --tenant-id) TENANT_ID="$2"; shift 2 ;;
    --client-id) CLIENT_ID="$2"; shift 2 ;;
    --secret-out) SECRET_OUT="$2"; shift 2 ;;
    --infisical-path) INFISICAL_PATH="$2"; shift 2 ;;
    --dry-run) DRY_RUN=1; shift ;;
    -h|--help) sed -n '2,27p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "Unknown arg: $1" >&2; exit 2 ;;
  esac
done

[[ "${TENANT_ID}" =~ ^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$ ]] || { echo "--tenant-id must be a tenant uuid" >&2; exit 2; }
[[ "${CLIENT_ID}" =~ ^[a-z0-9][a-z0-9._-]{2,63}$ ]] || { echo "--client-id: 3-64 chars, a-z 0-9 . _ -" >&2; exit 2; }
if [[ -z "${SECRET_OUT}" && -z "${INFISICAL_PATH}" ]]; then
  echo "give --secret-out <file> or --infisical-path <path>: the secret is never printed" >&2; exit 2
fi
command -v jq >/dev/null || { echo "jq is required" >&2; exit 2; }

env_or_file() { local v="${!1:-}"; if [[ -z "$v" && -f "${ENV_FILE}" ]]; then v="$(grep -E "^$1=" "${ENV_FILE}" | tail -1 | cut -d= -f2- | tr -d '"')"; fi; echo "${v:-$2}"; }
KC_HOST="$(env_or_file KEYCLOAK_HOST 100.84.50.65)"
KC_PORT="$(env_or_file KEYCLOAK_PORT 8443)"
KC_REALM="$(env_or_file KEYCLOAK_REALM uisce)"
KC_ADMIN="$(env_or_file KEYCLOAK_ADMIN '')"
KC_ADMIN_PASS="$(env_or_file KEYCLOAK_ADMIN_PASS '')"
INSECURE="$(env_or_file KEYCLOAK_INSECURE_SKIP_VERIFY true)"
[[ -n "${KC_ADMIN}" && -n "${KC_ADMIN_PASS}" ]] || { echo "set KEYCLOAK_ADMIN and KEYCLOAK_ADMIN_PASS" >&2; exit 2; }
BASE="https://${KC_HOST}:${KC_PORT}"
CURL=(curl -sS --fail-with-body)
[[ "${INSECURE}" == "true" ]] && CURL+=(-k)

say() { echo "• $*"; }
if (( DRY_RUN )); then
  say "dry run against ${BASE} realm ${KC_REALM}:"
  say "ensure realm roles uisce_service_account, schedule_trigger"
  say "ensure confidential client ${CLIENT_ID}: client-credentials only; claim tenant_id=${TENANT_ID}"
  say "grant both roles to service-account-${CLIENT_ID}"
  say "store the secret in ${SECRET_OUT:-Infisical ${INFISICAL_PATH}/UISCE_CLIENT_SECRET_${CLIENT_ID//[-.]/_}}"
  exit 0
fi

TOKEN="$("${CURL[@]}" -d grant_type=password -d client_id=admin-cli \
  --data-urlencode "username=${KC_ADMIN}" --data-urlencode "password=${KC_ADMIN_PASS}" \
  "${BASE}/realms/master/protocol/openid-connect/token" | jq -r .access_token)"
api() { local m="$1" p="$2"; shift 2; "${CURL[@]}" -X "$m" -H "Authorization: Bearer ${TOKEN}" -H 'Content-Type: application/json' "${BASE}/admin/realms/${KC_REALM}${p}" "$@"; }

for role in uisce_service_account schedule_trigger; do
  if api GET "/roles/${role}" >/dev/null 2>&1; then say "role ${role} exists"; else
    desc="Enterprise scheduler service account: reads and triggers schedules only"
    [[ "${role}" == schedule_trigger ]] && desc="May trigger externally triggered schedules"
    api POST /roles -d "$(jq -n --arg n "$role" --arg d "$desc" '{name:$n, description:$d}')" >/dev/null
    say "created role ${role}"
  fi
done

CID="$(api GET "/clients?clientId=${CLIENT_ID}" | jq -r '.[0].id // empty')"
CLIENT_JSON="$(jq -n --arg c "$CLIENT_ID" --arg t "$TENANT_ID" '{
  clientId: $c, name: ("Enterprise scheduler: " + $c), enabled: true, protocol: "openid-connect",
  publicClient: false, serviceAccountsEnabled: true, standardFlowEnabled: false,
  directAccessGrantsEnabled: false, implicitFlowEnabled: false, frontchannelLogout: false,
  attributes: {"access.token.lifespan": "300"},
  protocolMappers: [{name: "tenant-id", protocol: "openid-connect", protocolMapper: "oidc-hardcoded-claim-mapper",
    config: {"claim.name": "tenant_id", "claim.value": $t, "jsonType.label": "String",
             "access.token.claim": "true", "id.token.claim": "false", "userinfo.token.claim": "false"}}]}')"
if [[ -z "${CID}" ]]; then
  api POST /clients -d "${CLIENT_JSON}" >/dev/null
  CID="$(api GET "/clients?clientId=${CLIENT_ID}" | jq -r '.[0].id')"
  say "created client ${CLIENT_ID}"
else
  # Keep an existing client's tenant claim, flows and lifespan as defined here.
  api PUT "/clients/${CID}" -d "$(jq '. + {id: $id}' --arg id "$CID" <<<"${CLIENT_JSON}")" >/dev/null
  say "updated client ${CLIENT_ID}"
fi

SA="$(api GET "/clients/${CID}/service-account-user" | jq -r .id)"
ROLES="$(jq -s '.' <(api GET /roles/uisce_service_account) <(api GET /roles/schedule_trigger))"
api POST "/users/${SA}/role-mappings/realm" -d "${ROLES}" >/dev/null
say "granted uisce_service_account, schedule_trigger to service-account-${CLIENT_ID}"

SECRET="$(api GET "/clients/${CID}/client-secret" | jq -r .value)"
if [[ -n "${SECRET_OUT}" ]]; then
  umask 077
  mkdir -p "$(dirname "${SECRET_OUT}")"
  printf '%s' "${SECRET}" > "${SECRET_OUT}"
  chmod 600 "${SECRET_OUT}"
  say "secret written to ${SECRET_OUT} (0600) - point UISCE_CLIENT_SECRET_FILE at it on the agent"
else
  command -v infisical >/dev/null || { echo "infisical CLI not found" >&2; exit 2; }
  NAME="UISCE_CLIENT_SECRET_${CLIENT_ID//[-.]/_}"
  infisical secrets set "${NAME}=${SECRET}" --path "${INFISICAL_PATH}" >/dev/null
  say "secret stored in Infisical ${INFISICAL_PATH}/${NAME}"
fi
unset SECRET TOKEN
say "token URL for the agent: ${BASE}/realms/${KC_REALM}/protocol/openid-connect/token"
