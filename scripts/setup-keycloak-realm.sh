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
# 5. Add 'openid profile email roles' as default client scopes
# ---------------------------------------------------------------------------
log "Ensuring default client scopes (openid/profile/email/roles) are enabled…"
for scope in openid profile email roles; do
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
# 6. Verify the OIDC discovery doc is reachable
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

  # Verify the client config (should show "S256")
  curl -sk "${KEYCLOAK_URL}/admin/realms/${REALM_NAME}/clients?clientId=${CLIENT_ID}" \\
    -H "Authorization: Bearer \$(curl -sk -X POST \\
        '${KEYCLOAK_URL}/realms/master/protocol/openid-connect/token' \\
        -d 'grant_type=password&client_id=admin-cli' \\
        -d 'username=${KEYCLOAK_ADMIN}&password=${KEYCLOAK_ADMIN_PASS}' \\
        | jq -r .access_token)" \\
    | jq '.[0].attributes'

  # End-to-end PKCE handshake sanity check (no real login, just a redirect)
  curl -sk -o /dev/null -w '%{http_code} %{redirect_url}\\n' \\
    "${KEYCLOAK_URL}/realms/${REALM_NAME}/protocol/openid-connect/auth?client_id=${CLIENT_ID}&response_type=code&scope=openid+profile+email&redirect_uri=http%3A%2F%2Flocalhost%3A5173%2Fauth%2Fcallback&code_challenge=abc&code_challenge_method=S256"
  # Expect: 200 (login page) — NOT 400.

EOF