#!/usr/bin/env bash
#
# import-realm.sh — Provisions (or updates) the `uisce` Keycloak realm using
# the Admin REST API, including the `semlayer-frontend` SPA client and the
# `tenant-groups` scope/claim mapper.
#
# Usage:
#   ./import-realm.sh                  # uses defaults (env or .env in repo root)
#   ./import-realm.sh --dry-run        # show what would happen
#   KEYCLOAK_HOST=k.example.com ./import-realm.sh
#
# Reads (from env, then from repo-root .env if present):
#   KEYCLOAK_HOST         (default: 100.84.50.65)
#   KEYCLOAK_PORT         (default: 8443)
#   KEYCLOAK_ADMIN        (default: admin)
#   KEYCLOAK_ADMIN_PASS   (default: password)
#   KEYCLOAK_INSECURE_SKIP_VERIFY (default: true)
#
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REALM_FILE="${SCRIPT_DIR}/uisce-realm.json"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
ENV_FILE="${REPO_ROOT}/.env"

# ── Args ────────────────────────────────────────────────────────────────────
DRY_RUN=0
FORCE=0
for arg in "$@"; do
  case "${arg}" in
    --dry-run) DRY_RUN=1 ;;
    --force) FORCE=1 ;;
    -h|--help)
      sed -n '2,16p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
      exit 0
      ;;
    *) echo "Unknown arg: ${arg}" >&2; exit 2 ;;
  esac
done
export FORCE

# ── Config (env → .env → defaults) ──────────────────────────────────────────
[[ -f "${ENV_FILE}" ]] && set -a && . "${ENV_FILE}" && set +a

KEYCLOAK_HOST="${KEYCLOAK_HOST:-100.84.50.65}"
KEYCLOAK_PORT="${KEYCLOAK_PORT:-8443}"
KEYCLOAK_ADMIN="${KEYCLOAK_ADMIN:-admin}"
KEYCLOAK_ADMIN_PASS="${KEYCLOAK_ADMIN_PASS:-password}"
KEYCLOAK_INSECURE_SKIP_VERIFY="${KEYCLOAK_INSECURE_SKIP_VERIFY:-true}"
REALM_NAME="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["realm"])' "${REALM_FILE}")"

KC_BASE="https://${KEYCLOAK_HOST}:${KEYCLOAK_PORT}"
ADMIN_REALM="master"

if [[ "${KEYCLOAK_INSECURE_SKIP_VERIFY}" == "true" ]]; then
  CURL_KSSL="-k"
else
  CURL_KSSL=""
fi

CURL=(curl --silent --show-error --fail-with-body ${CURL_KSSL})

log()  { printf '\033[1;34m[import-realm]\033[0m %s\n' "$*" >&2; }
warn() { printf '\033[1;33m[import-realm]\033[0m %s\n' "$*" >&2; }
die()  { printf '\033[1;31m[import-realm]\033[0m %s\n' "$*" >&2; exit 1; }

[[ -f "${REALM_FILE}" ]] || die "Realm file not found: ${REALM_FILE}"
command -v curl >/dev/null   || die "curl is required"
command -v python3 >/dev/null || die "python3 is required (for JSON shaping)"

log "Target: ${KC_BASE}/admin/realms/${REALM_NAME}"
log "Realm file: ${REALM_FILE}"
log "Admin user: ${KEYCLOAK_ADMIN} @ realm ${ADMIN_REALM}"

# ── Get admin token ─────────────────────────────────────────────────────────
log "Acquiring admin token from realm '${ADMIN_REALM}'…"
TOKEN_RESPONSE="$( "${CURL[@]}" -X POST \
  -H "Content-Type: application/x-www-form-urlencoded" \
  -d "username=${KEYCLOAK_ADMIN}" \
  -d "password=${KEYCLOAK_ADMIN_PASS}" \
  -d "grant_type=password" \
  -d "client_id=admin-cli" \
  "${KC_BASE}/realms/${ADMIN_REALM}/protocol/openid-connect/token" \
)" || die "Failed to reach Keycloak at ${KC_BASE} — check host/port/TLS."

ADMIN_TOKEN="$(printf '%s' "${TOKEN_RESPONSE}" | python3 -c 'import json,sys; print(json.load(sys.stdin).get("access_token",""))')"
[[ -n "${ADMIN_TOKEN}" ]] || die "Could not extract access_token from response: ${TOKEN_RESPONSE}"
log "Got admin token (${#ADMIN_TOKEN} chars)"

KC_ADMIN_AUTH=(-H "Authorization: Bearer ${ADMIN_TOKEN}")

# ── Check if realm exists ───────────────────────────────────────────────────
log "Checking if realm '${REALM_NAME}' exists…"
REALM_PROBE_BODY="$( "${CURL[@]}" -w '\n%{http_code}' \
  "${KC_ADMIN_AUTH[@]}" \
  "${KC_BASE}/admin/realms/${REALM_NAME}" \
  || true )"
HTTP_CODE="$(printf '%s' "${REALM_PROBE_BODY}" | tail -n1)"

if [[ "${HTTP_CODE}" == "200" ]]; then
  ACTION="update"
  log "Realm '${REALM_NAME}' already exists — will UPDATE (PUT)."
elif [[ "${HTTP_CODE}" == "404" ]]; then
  ACTION="create"
  log "Realm '${REALM_NAME}' does not exist — will CREATE (POST)."
else
  die "Unexpected HTTP ${HTTP_CODE} when probing /admin/realms/${REALM_NAME}"
fi

# ── Safety check: refuse to overwrite a realm that already has state ────────
# A blind PUT replaces, not merges. If the realm on the server has data we
# don't have in our export (groups, users, identityProviders, clients, etc.),
# we will nuke it. Bail out unless --force is passed.
#
# Note: /admin/realms/{realm} does NOT embed clients/users/groups in its
# response — those are separate endpoints. We probe each one.
if [[ "${ACTION}" == "update" && "${FORCE:-0}" != "1" ]]; then
  log "Probing existing realm for content that would be lost…"
  HAS_CONTENT=0
  for endpoint in "clients" "users" "groups" "identity-providers" "client-scopes" "components"; do
    # Probe depth via full GET and parse length.
    # The "?" fallback is captured by `||` only when curl itself fails (network,
    # 4xx/5xx). For 200 responses with empty body, python3 prints "0".
    PROBE="$(
      RAW="$(
        "${CURL[@]}" -w '\n%{http_code}' \
          "${KC_ADMIN_AUTH[@]}" \
          "${KC_BASE}/admin/realms/${REALM_NAME}/${endpoint}" \
          2>/dev/null
      )"
      CODE="$(printf '%s' "${RAW}" | tail -n1)"
      BODY="$(printf '%s' "${RAW}" | sed '$d')"
      if [[ "${CODE}" == "200" ]]; then
        printf '%s' "${BODY}" | python3 -c 'import json,sys; print(len(json.load(sys.stdin)))' 2>/dev/null \
          || echo 0
      else
        echo "?"
      fi
    )"
    FULL_LEN="${PROBE}"
    if [[ "${FULL_LEN}" == "?" || -z "${FULL_LEN}" ]]; then
      continue
    fi
    if [[ "${FULL_LEN}" -gt 0 ]]; then
      # Exclude built-in client scopes (always present, not "content")
      if [[ "${endpoint}" == "client-scopes" ]]; then
        BUILTIN_COUNT="$(
          "${CURL[@]}" "${KC_ADMIN_AUTH[@]}" \
            "${KC_BASE}/admin/realms/${REALM_NAME}/${endpoint}" \
            | python3 <<'PYCOUNT' 2>/dev/null || echo 0
import json, sys
_builtin = {'profile','email','role_list','saml_organization','phone','acr','basic','organization','web-origins','microprofile-jwt','roles','address'}
_xs = json.load(sys.stdin)
print(sum(1 for _x in _xs if _x.get('name') in _builtin))
PYCOUNT
        )"
        CUSTOM=$((FULL_LEN - BUILTIN_COUNT))
        if [[ "${CUSTOM}" -gt 0 ]]; then
          HAS_CONTENT=1
          warn "  existing realm has ${CUSTOM} custom ${endpoint} -- overwrite would lose them"
        fi
        continue
      fi
      HAS_CONTENT=1
      warn "  existing realm has ${FULL_LEN} ${endpoint} -- overwrite would lose them"
    fi
  done
  if [[ "${HAS_CONTENT}" == "1" ]]; then
    die "Refusing to overwrite a realm with existing content. Re-run with --force if you really mean it."
  fi
fi

# ── Shape payload: strip fields Keycloak rejects on import ──────────────────
# Keycloak rejects some realm-export fields on import (e.g. 'access', 'client',
# 'user', 'groups'/'users' as maps-with-implicit-state). The Admin REST API
# accepts a slimmer payload than the full export; we strip the ones known to
# cause 400s on POST.
log "Preparing realm payload…"
REALM_PAYLOAD="$(python3 - <<PY
import json, sys
with open("${REALM_FILE}") as f:
    d = json.load(f)
# Strip fields not accepted by Admin REST import / older RealmRepresentation schemas
for k in ("access", "client", "user", "groups", "users"):
    d.pop(k, None)
# 'users' carries only service-account entries we want to drop on import.
# Re-add an empty default.
d.setdefault("users", [])
# Keycloak 26.x PUT semantics: replace, not merge. Strip any field that the
# server-side realm might have populated since initial import (keycloakVersion,
# organizationsEnabled, verifiableCredentialsEnabled, adminPermissionsEnabled,
# clientProfiles, clientPolicies, parRequestUriLifespan, authenticationFlows,
# etc.) so we never clobber hand-configured state.
for k in (
    "keycloakVersion", "organizationsEnabled", "verifiableCredentialsEnabled",
    "adminPermissionsEnabled", "clientProfiles", "clientPolicies",
    "parRequestUriLifespan", "parPolicy", "userVerificationTypes",
    "optionalDefaultClientScopes", "authenticationFlows", "authenticatorConfig",
    "requiredActions", "browserFlow", "registrationFlow", "directGrantFlow",
    "resetCredentialsFlow", "clientAuthenticationFlow", "dockerAuthenticationFlow",
    "firstBrokerLoginFlow", "attributes",
):
    d.pop(k, None)
print(json.dumps(d))
PY
)"

if [[ "${DRY_RUN}" == "1" ]]; then
  log "DRY RUN — would ${ACTION} realm with payload of ${#REALM_PAYLOAD} bytes."
  log "First 200 chars of payload:"
  printf '%s\n' "${REALM_PAYLOAD:0:200}"
  exit 0
fi

# ── Push to Keycloak ────────────────────────────────────────────────────────
if [[ "${ACTION}" == "create" ]]; then
  "${CURL[@]}" -X POST \
    "${KC_ADMIN_AUTH[@]}" \
    -H "Content-Type: application/json" \
    --data "${REALM_PAYLOAD}" \
    "${KC_BASE}/admin/realms" \
    >/dev/null \
    || die "POST /admin/realms failed"
  log "✓ Realm '${REALM_NAME}' created."
else
  "${CURL[@]}" -X PUT \
    "${KC_ADMIN_AUTH[@]}" \
    -H "Content-Type: application/json" \
    --data "${REALM_PAYLOAD}" \
    "${KC_BASE}/admin/realms/${REALM_NAME}" \
    >/dev/null \
    || die "PUT /admin/realms/${REALM_NAME} failed"
  log "✓ Realm '${REALM_NAME}' updated."
fi

# ── Verify the client is now visible ────────────────────────────────────────
log "Verifying client 'semlayer-frontend' is registered…"
INTERNAL_ID="$(
  "${CURL[@]}" "${KC_ADMIN_AUTH[@]}" \
    "${KC_BASE}/admin/realms/${REALM_NAME}/clients?clientId=semlayer-frontend" \
    | python3 -c 'import json,sys; xs=json.load(sys.stdin); print(xs[0]["id"] if xs else "")'
)"
if [[ -n "${INTERNAL_ID}" ]]; then
  log "Client semlayer-frontend present, internal id: ${INTERNAL_ID:0:8}"
else
  warn "Client semlayer-frontend not found after import -- investigate via Admin UI."
  exit 1
fi

log "Done. Next:"
log "  - Set frontend/.env.local: VITE_OIDC_ISSUER=${KC_BASE}/realms/${REALM_NAME}"
log "  - Restart 'npm run dev' in frontend/."
log "  - Browse ${KC_BASE}/admin/master/console/#/realms/${REALM_NAME}/clients to verify."