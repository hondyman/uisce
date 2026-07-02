#!/usr/bin/env bash
# ------------------------------------------------------------------------------
# setup-keycloak-realm.sh
#
# One-shot provisioning script for the `uisce` Keycloak realm and the
# `semlayer-frontend` SPA client (PKCE-protected, matching oidc-client-ts).
#
# Why this script exists
# ----------------------
# The frontend (oidc-client-ts) was returning:
#   GET .../realms/uisce/protocol/openid-connect/auth?...&code_challenge=...
#   → 400 (Bad Request)
# because Keycloak was started fresh with no `uisce` realm. The realm & client
# only exist as environment variables in `frontend/.env.local` — they had never
# been created inside the running Keycloak. This script creates them via the
# Keycloak Admin REST API so the OIDC auth flow can complete.
#
# Usage
# -----
#   ./scripts/setup-keycloak-realm.sh                          # use defaults
#   KEYCLOAK_URL=https://100.84.50.65:8443 ./scripts/setup-keycloak-realm.sh
#   REALM_NAME=uisce CLIENT_ID=semlayer-frontend ./scripts/setup-keycloak-realm.sh
#
# Environment variables (all optional):
#   KEYCLOAK_URL        base URL of Keycloak                 (default https://100.84.50.65:8443)
#   KEYCLOAK_ADMIN      bootstrap admin username              (default admin)
#   KEYCLOAK_ADMIN_PASS bootstrap admin password              (default password)
#   REALM_NAME          realm to create                       (default uisce)
#   CLIENT_ID           OIDC client id used by the frontend   (default semlayer-frontend)
#   REDIRECT_URI        comma-sep redirect URIs               (default http://localhost:5173/auth/callback)
#   WEB_ORIGINS         comma-sep web origins                 (default http://localhost:5173,+)
#   POST_LOGOUT_URI     comma-sep post-logout URIs            (default http://localhost:5173/login)
#   SKIP_TLS_VERIFY     set to 1 to use -k for curl           (default 1, since cert is self-signed)
#
# Global-admin / IdP-federation provisioning (off by default in prod):
#   BOOTSTRAP_USERS     set to 1 to also create john.b@example.com and add him
#                       to Uisce-Global-Admins. Off in prod pipelines — leave
#                       human account management to your IdP / Azure AD sync.
#   JOHN_B_PASSWORD     override the generated initial password for john.b
#                       (otherwise an openssl rand -base64 18 string is shown once)
#
# What this script wires up (when fully run):
#   Realm roles:        global_admin, global_ops, professional_services, helpdesk
#   Groups:             Uisce-Global-Admins, Uisce-Professional-Services, Uisce-Helpdesk
#   Role→Group map:     each group inherits its corresponding realm role
#   (opt) User:         john.b@example.com → member of Uisce-Global-Admins
#
# When the frontend logs john.b in, his ID token carries:
#   realm_access.roles: ["global_admin", ...]
#   groups:             ["/Uisce-Global-Admins"]   (or the UUID)
# The Go backend's AuthContextMiddleware then sets IsGlobalAdmin=true and the
# frontend AccessContext routes him to /api/tenants/all (cross-tenant view).
# ------------------------------------------------------------------------------

set -euo pipefail

KEYCLOAK_URL="${KEYCLOAK_URL:-https://100.84.50.65:8443}"
KEYCLOAK_ADMIN="${KEYCLOAK_ADMIN:-admin}"
KEYCLOAK_ADMIN_PASS="${KEYCLOAK_ADMIN_PASS:-Gu1nn3ss!}"
REALM_NAME="${REALM_NAME:-uisce}"
CLIENT_ID="${CLIENT_ID:-semlayer-frontend}"
REDIRECT_URI="${REDIRECT_URI:-http://localhost:5173/auth/callback}"
WEB_ORIGINS="${WEB_ORIGINS:-http://localhost:5173,+}"
POST_LOGOUT_URI="${POST_LOGOUT_URI:-http://localhost:5173/login}"
SKIP_TLS_VERIFY="${SKIP_TLS_VERIFY:-1}"

# Global-admin / IdP-federation provisioning (off by default; see header).
BOOTSTRAP_USERS="${BOOTSTRAP_USERS:-0}"
JOHN_B_EMAIL="${JOHN_B_EMAIL:-john.b@example.com}"
JOHN_B_FIRST="${JOHN_B_FIRST:-John}"
JOHN_B_LAST="${JOHN_B_LAST:-B}"
# If unset, a fresh base64 password is generated and printed once.
JOHN_B_PASSWORD="${JOHN_B_PASSWORD:-}"

# Canonical (group → role) inheritance table. Keep aligned with the backend's
# fallback mapping at backend/internal/services/security_manager.go:706-720.
GROUP_ROLE_PAIRS=(
  "Uisce-Global-Admins:global_admin"
  "Uisce-Professional-Services:professional_services"
  "Uisce-Helpdesk:helpdesk"
)
REALM_ROLES=("global_admin" "global_ops" "professional_services" "helpdesk")

# Filled in by section 6 / 7 below, consumed by sections 8 / 9 / 10.
# Stored as discrete shell variables (ROLE_ID_<sanitized_name> /
# GROUP_ID_<sanitized_name>) so the script works on bash 3.2 (macOS default),
# which lacks associative arrays. The sanitization rule is: replace `-` with `_`.
# Helpers below use parameter indirection (${!var}) to read these.

# Bash 3.2-compatible accessors. Use printf -v to write (bash 3.1+).
get_role_id() {
  local v="ROLE_ID_${1//-/_}"
  printf '%s' "${!v:-}"
}
get_group_id() {
  local v="GROUP_ID_${1//-/_}"
  printf '%s' "${!v:-}"
}
set_role_id() {
  printf -v "ROLE_ID_${1//-/_}" '%s' "$2"
}
set_group_id() {
  printf -v "GROUP_ID_${1//-/_}" '%s' "$2"
}

CURL_TLS=()
if [[ "${SKIP_TLS_VERIFY}" == "1" ]]; then
  CURL_TLS=(-k)
fi

# Pretty log helpers
log()  { printf '\033[1;34m[setup-keycloak]\033[0m %s\n' "$*"; }
warn() { printf '\033[1;33m[setup-keycloak]\033[0m %s\n' "$*" >&2; }
err()  { printf '\033[1;31m[setup-keycloak]\033[0m %s\n' "$*" >&2; }

require() {
  command -v "$1" >/dev/null 2>&1 || { err "Missing required command: $1"; exit 1; }
}

require curl
require jq

log "Target: ${KEYCLOAK_URL}  realm=${REALM_NAME}  client=${CLIENT_ID}"

# ---------------------------------------------------------------------------
# 1. Wait for Keycloak to be reachable
# ---------------------------------------------------------------------------
log "Waiting for Keycloak master realm to become reachable…"
for i in {1..30}; do
  if curl -sS "${CURL_TLS[@]}" -o /dev/null -w '%{http_code}' \
       "${KEYCLOAK_URL}/realms/master/.well-known/openid-configuration" \
       | grep -q '^200$'; then
    log "Keycloak master realm is up."
    break
  fi
  if [[ $i -eq 30 ]]; then
    err "Keycloak did not become reachable at ${KEYCLOAK_URL} within 30s."
    err "Start it first:  cd /Users/eganpj/GitHub/uisce && docker compose -f docker-compose.remote.yml up -d uisce-keycloak"
    exit 1
  fi
  sleep 1
done

# ---------------------------------------------------------------------------
# 2. Get an admin access token from the `master` realm
# ---------------------------------------------------------------------------
log "Requesting admin access token…"
TOKEN_JSON=$(curl -sS "${CURL_TLS[@]}" \
  -X POST "${KEYCLOAK_URL}/realms/master/protocol/openid-connect/token" \
  -H 'Content-Type: application/x-www-form-urlencoded' \
  -d "grant_type=password" \
  -d "client_id=admin-cli" \
  -d "username=${KEYCLOAK_ADMIN}" \
  -d "password=${KEYCLOAK_ADMIN_PASS}")

ADMIN_TOKEN=$(printf '%s' "${TOKEN_JSON}" | jq -r '.access_token // empty')
if [[ -z "${ADMIN_TOKEN}" ]]; then
  err "Failed to obtain admin token. Response was:"
  echo "${TOKEN_JSON}" | jq . || echo "${TOKEN_JSON}"
  exit 1
fi
log "Admin token acquired."

ADMIN_AUTH=(-H "Authorization: Bearer ${ADMIN_TOKEN}" -H 'Content-Type: application/json')

# ---------------------------------------------------------------------------
# 3. Create the realm (idempotent)
# ---------------------------------------------------------------------------
log "Ensuring realm '${REALM_NAME}' exists…"
REALM_EXISTS=$(curl -sS "${CURL_TLS[@]}" -o /dev/null -w '%{http_code}' \
  "${ADMIN_AUTH[@]}" "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}")

if [[ "${REALM_EXISTS}" == "200" ]]; then
  log "Realm '${REALM_NAME}' already exists — skipping creation."
else
  log "Creating realm '${REALM_NAME}'…"
  curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    -X POST "${KEYCLOAK_URL}/admin/realms" \
    -d "$(jq -n \
      --arg realm "${REALM_NAME}" \
      '{
        realm: $realm,
        enabled: true,
        displayName: "Uisce",
        registrationAllowed: false,
        loginWithEmailAllowed: true,
        duplicateEmailsAllowed: false,
        resetPasswordAllowed: true,
        editUsernameAllowed: false,
        bruteForceProtected: true,
        sslRequired: "external",
        accessTokenLifespan: 1800,
        ssoSessionIdleTimeout: 1800,
        ssoSessionMaxLifespan: 36000,
        internationalizationEnabled: false,
        loginTheme: "keycloak",
        accountTheme: "keycloak",
        adminTheme: "keycloak",
        emailTheme: "keycloak"
      }')"
  log "Realm '${REALM_NAME}' created."
fi

# ---------------------------------------------------------------------------
# 4. Create / update the SPA client (PKCE S256)
# ---------------------------------------------------------------------------
log "Ensuring client '${CLIENT_ID}' exists in realm '${REALM_NAME}'…"

# Build JSON arrays from comma-separated env values
REDIRECT_ARR=$(jq -nc --arg s "${REDIRECT_URI}" '($s | split(",")) | map(gsub("^\\s+|\\s+$";""))')
WEBORIG_ARR=$(jq -nc --arg s "${WEB_ORIGINS}" '($s | split(",")) | map(gsub("^\\s+|\\s+$";""))')
POSTLOGOUT_ARR=$(jq -nc --arg s "${POST_LOGOUT_URI}" '($s | split(",")) | map(gsub("^\\s+|\\s+$";""))')
POSTLOGOUT_ATTR=$(printf '%s' "${POST_LOGOUT_URI}" | sed 's/,/##/g')

CLIENT_INTERNAL_ID=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
  "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/clients?clientId=${CLIENT_ID}" \
  | jq -r '.[0].id // empty')

if [[ -z "${CLIENT_INTERNAL_ID}" ]]; then
  log "Creating client '${CLIENT_ID}'…"
  CREATE_PAYLOAD=$(jq -n \
    --arg cid "${CLIENT_ID}" \
    --argjson redirect "${REDIRECT_ARR}" \
    --argjson origins "${WEBORIG_ARR}" \
    --arg postlogout "${POSTLOGOUT_ATTR}" \
    '{
      clientId: $cid,
      name: "SemLayer Frontend (Vite/React)",
      description: "PKCE-protected public SPA client for the SemLayer frontend.",
      enabled: true,
      publicClient: true,
      directAccessGrantsEnabled: false,
      standardFlowEnabled: true,
      implicitFlowEnabled: false,
      serviceAccountsEnabled: false,
      frontchannelLogout: true,
      attributes: {
        "pkce.code.challenge.method": "S256",
        "post.logout.redirect.uris": $postlogout
      },
      redirectUris: $redirect,
      webOrigins: $origins,
      origin: "http://localhost:5173",
      rootUrl: "http://localhost:5173",
      baseUrl: "http://localhost:5173",
      protocol: "openid-connect",
      fullScopeAllowed: true,
      consentRequired: false
    }')

  CREATE_RESP=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    -w '\n%{http_code}' \
    -X POST "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/clients" \
    -d "${CREATE_PAYLOAD}")
  CREATE_CODE=$(printf '%s' "${CREATE_RESP}" | tail -n1)
  if [[ "${CREATE_CODE}" != "201" ]]; then
    err "Client creation failed (HTTP ${CREATE_CODE}):"
    printf '%s\n' "${CREATE_RESP}" | head -n -1
    exit 1
  fi
  CLIENT_INTERNAL_ID=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/clients?clientId=${CLIENT_ID}" \
    | jq -r '.[0].id // empty')
  log "Client '${CLIENT_ID}' created (internal id ${CLIENT_INTERNAL_ID})."
else
  log "Client '${CLIENT_ID}' already exists (internal id ${CLIENT_INTERNAL_ID}) — updating PKCE / URIs…"
  UPDATE_PAYLOAD=$(jq -n \
    --arg cid "${CLIENT_ID}" \
    --argjson redirect "${REDIRECT_ARR}" \
    --argjson origins "${WEBORIG_ARR}" \
    --arg postlogout "${POSTLOGOUT_ATTR}" \
    '{
      clientId: $cid,
      enabled: true,
      publicClient: true,
      standardFlowEnabled: true,
      directAccessGrantsEnabled: false,
      implicitFlowEnabled: false,
      attributes: {
        "pkce.code.challenge.method": "S256",
        "post.logout.redirect.uris": $postlogout
      },
      redirectUris: $redirect,
      webOrigins: $origins
    }')
  curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    -X PUT "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/clients/${CLIENT_INTERNAL_ID}" \
    -d "${UPDATE_PAYLOAD}" >/dev/null
  log "Client '${CLIENT_ID}' updated."
fi

# ---------------------------------------------------------------------------
# 5. Add 'openid profile email roles groups' as default client scopes
# ---------------------------------------------------------------------------
log "Ensuring default client scopes (openid/profile/email/roles/groups) are enabled…"
for scope in openid profile email roles groups; do
  SCOPE_ID=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/client-scopes?exact=true&name=${scope}" \
    | jq -r '.[0].id // empty')
  if [[ -n "${SCOPE_ID}" ]]; then
    curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
      -X PUT "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/default-default-client-scopes/${SCOPE_ID}" \
      >/dev/null || true
  fi
done

# ---------------------------------------------------------------------------
# 5b. Ensure a client scope with a 'groups' OIDC group mapper is present in
# default-default-client-scopes. We prefer reusing the realm's existing
# 'tenant-groups' scope (created by the platform's bootstrap) if it already
# carries an OIDC group-membership mapper. If not, we attach the mapper to
# whichever scope ends up in default-default-client-scopes and is intended
# for federated group emission.
# ---------------------------------------------------------------------------
log "Verifying that a 'groups' claim is emitted by some default client scope…"

# Find every client scope that has a protocol mapper producing the 'groups' claim.
GROUPS_PRODUCING_SCOPE_ID=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
  "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/client-scopes" \
  | jq -r '.[] | select(.protocol == "openid-connect") | .id' \
  | while read -r SCID; do
      MAPPERS=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
        "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/client-scopes/${SCID}/protocol-mappers/models" 2>/dev/null)
      if echo "${MAPPERS}" | jq -e '.[] | select(.config["claim.name"] == "groups" and .protocolMapper == "oidc-group-membership-mapper")' >/dev/null 2>&1; then
        echo "${SCID}"
        break
      fi
    done | head -1)

if [[ -z "${GROUPS_PRODUCING_SCOPE_ID}" ]]; then
  warn "No client scope is producing a 'groups' claim yet; creating 'uisce-groups' scope + mapper."
  curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    -X POST "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/client-scopes" \
    -d "$(jq -n '{
      name: "uisce-groups",
      protocol: "openid-connect",
      attributes: {
        "include.in.token.scope": "true",
        "display.on.consent.screen": "true",
        "consent.screen.text": "${groupsScopeConsentText}"
      }
    }')" >/dev/null
  GROUPS_PRODUCING_SCOPE_ID=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/client-scopes?exact=true&name=uisce-groups" \
    | jq -r '.[0].id // empty')
  curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    -X POST "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/client-scopes/${GROUPS_PRODUCING_SCOPE_ID}/protocol-mappers/models" \
    -d "$(jq -n '{
      name: "groups",
      protocol: "openid-connect",
      protocolMapper: "oidc-group-membership-mapper",
      config: {
        "claim.name": "groups",
        "full.path": "false",
        "multivalued": "true",
        "userinfo.token.claim": "true",
        "id.token.claim": "true",
        "access.token.claim": "true"
      }
    }')" >/dev/null
  log "Created 'uisce-groups' scope + 'groups' OIDC mapper."
fi

# Make sure that scope is in default-default-client-scopes.
ALREADY_DEFAULT=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
  "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/default-default-client-scopes" \
  | jq --arg id "${GROUPS_PRODUCING_SCOPE_ID}" '[.[] | select(.id == $id)] | length')

if [[ "${ALREADY_DEFAULT}" == "0" && -n "${GROUPS_PRODUCING_SCOPE_ID}" ]]; then
  log "Adding scope ${GROUPS_PRODUCING_SCOPE_ID} to default-default-client-scopes…"
  curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    -X PUT "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/default-default-client-scopes/${GROUPS_PRODUCING_SCOPE_ID}" \
    -w '\nHTTP: %{http_code}\n' || true
fi

# Clean up legacy duplicate 'groups' scopes (keep 'tenant-groups' if it's the
# default producer, otherwise keep the most recent).
DUPES=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
  "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/client-scopes" \
  | jq -r '.[] | select(.name == "groups") | .id')
for DUP_ID in ${DUPES}; do
  if [[ "${DUP_ID}" != "${GROUPS_PRODUCING_SCOPE_ID}" ]]; then
    log "Deleting legacy duplicate 'groups' scope ${DUP_ID}…"
    curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
      -X DELETE "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/client-scopes/${DUP_ID}" \
      -w 'HTTP: %{http_code}\n' || true
  fi
done

# ---------------------------------------------------------------------------
# 6. Create Realm Roles (global_admin, global_ops, professional_services,
#    helpdesk). Idempotent: skipped if the role already exists.
# ---------------------------------------------------------------------------
log "Ensuring realm roles exist…"
for ROLE_NAME in "${REALM_ROLES[@]}"; do
  ROLE_ID=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/roles/${ROLE_NAME}" \
    | jq -r '.id // empty')

  if [[ -n "${ROLE_ID}" ]]; then
    log "Realm role '${ROLE_NAME}' exists (id ${ROLE_ID})."
  else
    log "Creating realm role '${ROLE_NAME}'…"
    CREATE_ROLE_CODE=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
      -o /tmp/keycloak_role_create.json \
      -w '%{http_code}' \
      -X POST "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/roles" \
      -d "$(jq -n \
        --arg name "${ROLE_NAME}" \
        '{
          name: $name,
          description: ("Uisce Realm Role: " + $name),
          composite: false,
          clientRole: false
        }')")
    if [[ "${CREATE_ROLE_CODE}" != "201" && "${CREATE_ROLE_CODE}" != "409" ]]; then
      err "Failed to create realm role '${ROLE_NAME}' (HTTP ${CREATE_ROLE_CODE}):"
      cat /tmp/keycloak_role_create.json
      exit 1
    fi
    ROLE_ID=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
      "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/roles/${ROLE_NAME}" \
      | jq -r '.id // empty')
    log "Realm role '${ROLE_NAME}' created (id ${ROLE_ID})."
  fi
  set_role_id "${ROLE_NAME}" "${ROLE_ID}"
done

# ---------------------------------------------------------------------------
# 7. Create Groups (Uisce-Global-Admins, Uisce-Professional-Services,
#    Uisce-Helpdesk). Idempotent: skipped if the group already exists.
# ---------------------------------------------------------------------------
log "Ensuring groups exist…"
for PAIR in "${GROUP_ROLE_PAIRS[@]}"; do
  GROUP_NAME="${PAIR%%:*}"
  ROLE_NAME="${PAIR##*:}"
  GROUP_ID=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/groups?exact=true&name=${GROUP_NAME}" \
    | jq -r '.[0].id // empty')

  if [[ -n "${GROUP_ID}" ]]; then
    log "Group '${GROUP_NAME}' exists (id ${GROUP_ID})."
  else
    log "Creating group '${GROUP_NAME}'…"
    CREATE_GROUP_CODE=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
      -o /tmp/keycloak_group_create.json \
      -w '%{http_code}' \
      -X POST "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/groups" \
      -d "$(jq -n \
        --arg name "${GROUP_NAME}" \
        --arg role "${ROLE_NAME}" \
        '{
          name: $name,
          attributes: {
            "uisce.role": [$role],
            "uisce.purpose": ["platform-operator"]
          }
        }')")
    if [[ "${CREATE_GROUP_CODE}" != "201" && "${CREATE_GROUP_CODE}" != "409" ]]; then
      err "Failed to create group '${GROUP_NAME}' (HTTP ${CREATE_GROUP_CODE}):"
      cat /tmp/keycloak_group_create.json
      exit 1
    fi
    GROUP_ID=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
      "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/groups?exact=true&name=${GROUP_NAME}" \
      | jq -r '.[0].id // empty')
    log "Group '${GROUP_NAME}' created (id ${GROUP_ID})."
  fi
  set_group_id "${GROUP_NAME}" "${GROUP_ID}"
done

# ---------------------------------------------------------------------------
# 8. Map each Realm Role onto its Group (Role→Group inheritance).
#    This is the critical link: the user joins the group, Keycloak resolves
#    the role from the group at token-issuance time, and realm_access.roles
#    in the JWT carries 'global_admin'.
# ---------------------------------------------------------------------------
log "Mapping realm roles onto groups (Role→Group inheritance)…"
for PAIR in "${GROUP_ROLE_PAIRS[@]}"; do
  GROUP_NAME="${PAIR%%:*}"
  ROLE_NAME="${PAIR##*:}"
  GROUP_ID="$(get_group_id "${GROUP_NAME}")"
  ROLE_ID="$(get_role_id "${ROLE_NAME}")"

  if [[ -z "${GROUP_ID}" || -z "${ROLE_ID}" ]]; then
    warn "Skipping mapping for '${GROUP_NAME}' → '${ROLE_NAME}' (missing id)."
    continue
  fi

  MAPPING_EXISTS=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/groups/${GROUP_ID}/role-mappings/realm" \
    | jq --arg rid "${ROLE_ID}" '[.[] | select(.id == $rid)] | length')

  if [[ "${MAPPING_EXISTS}" == "0" ]]; then
    log "Mapping role '${ROLE_NAME}' onto group '${GROUP_NAME}'…"
    MAP_CODE=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
      -o /tmp/keycloak_rolemap.json \
      -w '%{http_code}' \
      -X POST "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/groups/${GROUP_ID}/role-mappings/realm" \
      -d "$(jq -n \
        --arg rid "${ROLE_ID}" \
        --arg rname "${ROLE_NAME}" \
        '[{id: $rid, name: $rname}]')")
    if [[ "${MAP_CODE}" != "204" && "${MAP_CODE}" != "200" ]]; then
      err "Role→Group mapping failed for '${GROUP_NAME}' ← '${ROLE_NAME}' (HTTP ${MAP_CODE}):"
      cat /tmp/keycloak_rolemap.json
      exit 1
    fi
  else
    log "Mapping role '${ROLE_NAME}' → group '${GROUP_NAME}' already exists."
  fi
done

# ---------------------------------------------------------------------------
# 9. Verify the role→group inheritance is wired correctly (programmatic check)
# ---------------------------------------------------------------------------
log "Verifying role→group inheritance…"
for PAIR in "${GROUP_ROLE_PAIRS[@]}"; do
  GROUP_NAME="${PAIR%%:*}"
  ROLE_NAME="${PAIR##*:}"
  GROUP_ID="$(get_group_id "${GROUP_NAME}")"

  if [[ -z "${GROUP_ID}" ]]; then
    warn "Cannot verify '${GROUP_NAME}' (no id)."
    continue
  fi

  EFFECTIVE=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/groups/${GROUP_ID}/role-mappings/realm" \
    | jq -r --arg rn "${ROLE_NAME}" '[.[] | select(.name == $rn)] | length')

  if [[ "${EFFECTIVE}" == "0" ]]; then
    err "❌ Role '${ROLE_NAME}' is NOT mapped onto group '${GROUP_NAME}'."
    exit 1
  fi
  log "✅ '${GROUP_NAME}' carries realm role '${ROLE_NAME}'."
done

# ---------------------------------------------------------------------------
# 10. Optional: Bootstrap user (off by default — see BOOTSTRAP_USERS env).
#     Creates john.b@example.com if missing and adds him to Uisce-Global-Admins.
# ---------------------------------------------------------------------------
if [[ "${BOOTSTRAP_USERS}" == "1" ]]; then
  log "BOOTSTRAP_USERS=1 — provisioning user '${JOHN_B_EMAIL}'…"

  if [[ -z "${JOHN_B_PASSWORD}" ]] && command -v openssl >/dev/null 2>&1; then
    JOHN_B_PASSWORD="$(openssl rand -base64 18 | tr -d '\n')"
    PASSWORD_WAS_GENERATED=1
  elif [[ -z "${JOHN_B_PASSWORD}" ]]; then
    JOHN_B_PASSWORD="ChangeMe!$(date +%s)"
    PASSWORD_WAS_GENERATED=1
  fi
  PASSWORD_WAS_GENERATED="${PASSWORD_WAS_GENERATED:-0}"

  USER_ID=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
    "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/users?exact=true&username=${JOHN_B_EMAIL}" \
    | jq -r '.[0].id // empty')

  if [[ -z "${USER_ID}" ]]; then
    log "Creating user '${JOHN_B_EMAIL}'…"
    CREATE_USER_CODE=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
      -o /tmp/keycloak_user_create.json \
      -w '%{http_code}' \
      -X POST "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/users" \
      -d "$(jq -n \
        --arg email "${JOHN_B_EMAIL}" \
        --arg pass "${JOHN_B_PASSWORD}" \
        --arg first "${JOHN_B_FIRST}" \
        --arg last  "${JOHN_B_LAST}" \
        '{
          username: $email,
          email: $email,
          emailVerified: true,
          enabled: true,
          firstName: $first,
          lastName: $last,
          credentials: [{
            type: "password",
            value: $pass,
            temporary: false
          }]
        }')")
    if [[ "${CREATE_USER_CODE}" != "201" ]]; then
      err "User creation failed for '${JOHN_B_EMAIL}' (HTTP ${CREATE_USER_CODE}):"
      cat /tmp/keycloak_user_create.json
      exit 1
    fi
    USER_ID=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
      "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/users?exact=true&username=${JOHN_B_EMAIL}" \
      | jq -r '.[0].id // empty')
    log "User '${JOHN_B_EMAIL}' created (id ${USER_ID})."
    if [[ "${PASSWORD_WAS_GENERATED}" == "1" ]]; then
      echo
      echo "  ┏━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo "  ┃  Initial password for ${JOHN_B_EMAIL}:"
      echo "  ┃    ${JOHN_B_PASSWORD}"
      echo "  ┃  ⚠️  Store this securely — it is shown only once."
      echo "  ┗━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
      echo
    fi
  else
    log "User '${JOHN_B_EMAIL}' exists (id ${USER_ID})."
  fi

  GROUP_ID_FOR_USER="$(get_group_id Uisce-Global-Admins)"
  if [[ -n "${GROUP_ID_FOR_USER}" ]]; then
    ALREADY_MEMBER=$(curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
      "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/users/${USER_ID}/groups" \
      | jq --arg gid "${GROUP_ID_FOR_USER}" '[.[] | select(.id == $gid)] | length')

    if [[ "${ALREADY_MEMBER}" == "0" ]]; then
      log "Adding user to group 'Uisce-Global-Admins'…"
      curl -sS "${CURL_TLS[@]}" "${ADMIN_AUTH[@]}" \
        -X PUT "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/users/${USER_ID}/groups/${GROUP_ID_FOR_USER}" \
        -w 'HTTP: %{http_code}\n' >/dev/null
      log "User added to 'Uisce-Global-Admins'."
    else
      log "User already member of 'Uisce-Global-Admins'."
    fi
  else
    warn "Group 'Uisce-Global-Admins' not found — user was created but not added."
  fi
else
  log "Skipping user bootstrap (set BOOTSTRAP_USERS=1 to enable)."
fi

# ---------------------------------------------------------------------------
# 11. Verify the OIDC discovery doc is reachable
# ---------------------------------------------------------------------------
log "Verifying OIDC discovery doc for realm '${REALM_NAME}'…"
DISCOVERY=$(curl -sS "${CURL_TLS[@]}" -o /dev/null -w '%{http_code}' \
  "${KEYCLOAK_URL}/realms/${REALM_NAME}/.well-known/openid-configuration")
if [[ "${DISCOVERY}" != "200" ]]; then
  err "OIDC discovery failed for ${REALM_NAME} (HTTP ${DISCOVERY})."
  err "Check Keycloak logs:  docker logs uisce-keycloak --tail=200"
  exit 1
fi
log "OIDC discovery OK at ${KEYCLOAK_URL}/realms/${REALM_NAME}/.well-known/openid-configuration"

cat <<EOF

✅ Done. You can now log in from http://localhost:5173.

Quick checks:

  # ── 1. Verify the SPA client config (should show "S256") ──
  curl -sk "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/clients?clientId=${CLIENT_ID}" \\
    -H "Authorization: Bearer \$(curl -sk -X POST \\
        '${KEYCLOAK_URL}/realms/master/protocol/openid-connect/token' \\
        -d 'grant_type=password&client_id=admin-cli' \\
        -d 'username=${KEYCLOAK_ADMIN}&password=${KEYCLOAK_ADMIN_PASS}' \\
        | jq -r .access_token)" \\
    | jq '.[0].attributes'

  # ── 2. End-to-end PKCE handshake sanity check (no real login) ──
  curl -sk -o /dev/null -w '%{http_code} %{redirect_url}\\n' \\
    "${KEYCLOAK_URL}/realms/${REALM_NAME}/protocol/openid-connect/auth?client_id=${CLIENT_ID}&response_type=code&scope=openid+profile+email&redirect_uri=http%3A%2F%2Flocalhost%3A5173%2Fauth%2Fcallback&code_challenge=abc&code_challenge_method=S256"
  # Expect: 200 (login page) — NOT 400.

  # ── 3. List the realm roles (should include global_admin) ──
  curl -sk "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/roles" \\
    -H "Authorization: Bearer \$(curl -sk -X POST \\
        '${KEYCLOAK_URL}/realms/master/protocol/openid-connect/token' \\
        -d 'grant_type=password&client_id=admin-cli' \\
        -d 'username=${KEYCLOAK_ADMIN}&password=${KEYCLOAK_ADMIN_PASS}' \\
        | jq -r .access_token)" \\
    | jq '[.[] | select(.name | startswith("global_") or . == "professional_services" or . == "helpdesk")] | map(.name)'

  # ── 4. List the Uisce-* groups (should include Uisce-Global-Admins) ──
  curl -sk "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/groups?search=Uisce" \\
    -H "Authorization: Bearer \$(curl -sk -X POST \\
        '${KEYCLOAK_URL}/realms/master/protocol/openid-connect/token' \\
        -d 'grant_type=password&client_id=admin-cli' \\
        -d 'username=${KEYCLOAK_ADMIN}&password=${KEYCLOAK_ADMIN_PASS}' \\
        | jq -r .access_token)" \\
    | jq '[.[] | {name: .name, id: .id}]'

  # ── 5. Decode John's JWT to confirm global_admin appears in realm_access.roles ──
  # (Only works after the user has logged in once. Use any client — semlayer-frontend
  # requires PKCE, so we use the password grant on a separate confidential client or
  # the admin-cli for an interactive decode. The frontend browser flow emits the
  # same claims; this is just a structural smoke test.)
  ACCESS_TOKEN="\$(curl -sk -X POST \\
    '${KEYCLOAK_URL}/realms/${REALM_NAME}/protocol/openid-connect/token' \\
    -H 'Content-Type: application/x-www-form-urlencoded' \\
    -d 'grant_type=password' \\
    -d 'client_id=admin-cli' \\
    -d 'username=${JOHN_B_EMAIL}' \\
    -d 'password=\${JOHN_B_PASSWORD}' \\
    | jq -r .access_token)"
  echo "\${ACCESS_TOKEN}" | awk -F. '{print \$2}' | tr '_-' '/+' | base64 -d 2>/dev/null | jq .

  # Expected (key claims):
  #   "realm_access": { "roles": [ "global_admin", "uma_authorization", ... ] }
  #   "groups":       [ "/Uisce-Global-Admins" ]    (or the UUID, depending on mapper)

EOF
