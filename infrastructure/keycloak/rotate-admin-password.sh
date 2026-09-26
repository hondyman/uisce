#!/usr/bin/env bash
#
# rotate-admin-password.sh — Rotates the Keycloak master-realm admin password
# with Infisical as the only place it lives, so a rotation can never strand
# the new password inside Keycloak alone.
#
# Order (each step only after the previous succeeded):
#   1. read the admin login from Infisical and prove it works (else: stop,
#      nothing changed)
#   2. generate a new password and store it in Infisical as
#      KEYCLOAK_ADMIN_PASS_PENDING                      <- saved before Keycloak changes
#   3. set it in Keycloak
#   4. log in with it (if that fails: put the old one back in Keycloak;
#      PENDING still holds the new one either way)
#   5. promote: KEYCLOAK_ADMIN_PASS = new (Infisical keeps the old version in
#      its history), delete PENDING
# A PENDING left behind by an interrupted run is detected on the next run.
#
# Passwords never appear on a command line, in output or in logs: they are
# passed to curl and the Infisical CLI through files in a private (0700)
# temp directory that is wiped on exit.
#
# Usage:
#   ./rotate-admin-password.sh            # rotate
#   ./rotate-admin-password.sh --check    # only prove the stored login works
#   ./rotate-admin-password.sh --dry-run  # say what would happen
#
# Environment:
#   INFISICAL_TOKEN         machine identity token (non-interactive runs)
#   INFISICAL_DOMAIN        default http://100.84.50.65:8085/api
#   INFISICAL_PROJECT_ID    default 860e3163-8e2d-410b-a9e5-c7dd44d1e343
#   INFISICAL_ENV           default dev
#   INFISICAL_ADMIN_PATH    default /platform-admin (restrict to platform admins)
#   KEYCLOAK_HOST/PORT      default 100.84.50.65 / 8443
#   KEYCLOAK_INSECURE_SKIP_VERIFY  default true (dev only)
#
# Secrets at INFISICAL_ADMIN_PATH: KEYCLOAK_ADMIN (user), KEYCLOAK_ADMIN_PASS.
#
# Exit codes: 0 rotated/ok, 1 bad usage, 2 current login refused (nothing
# changed), 3 could not store the new password (nothing changed),
# 4 Keycloak refused the change (old password still valid), 5 new password did
# not verify (old one restored), 6 promote failed (new password live and in
# PENDING - finish by hand), 7 a PENDING from an interrupted run exists.
set -euo pipefail

MODE=rotate
for arg in "$@"; do
  case "$arg" in
    --check) MODE=check ;;
    --dry-run) MODE=dry ;;
    -h|--help) sed -n '2,45p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown argument: $arg" >&2; exit 1 ;;
  esac
done

DOMAIN="${INFISICAL_DOMAIN:-http://100.84.50.65:8085/api}"
PROJECT="${INFISICAL_PROJECT_ID:-860e3163-8e2d-410b-a9e5-c7dd44d1e343}"
ENVIRONMENT="${INFISICAL_ENV:-dev}"
SPATH="${INFISICAL_ADMIN_PATH:-/platform-admin}"
KC_BASE="${KEYCLOAK_BASE:-https://${KEYCLOAK_HOST:-100.84.50.65}:${KEYCLOAK_PORT:-8443}}"
INFISICAL_BIN="${INFISICAL_BIN:-infisical}"

log() { echo "[$(date -u +%Y-%m-%dT%H:%M:%SZ)] $*"; }
for tool in curl jq openssl "$INFISICAL_BIN"; do
  command -v "$tool" >/dev/null || { echo "$tool is required" >&2; exit 1; }
done

TMP="$(mktemp -d)"
chmod 700 "$TMP"
trap 'rm -rf "$TMP"' EXIT

INF=("$INFISICAL_BIN" --domain "$DOMAIN" --projectId "$PROJECT" --env "$ENVIRONMENT" --path "$SPATH" --silent)
[[ -n "${INFISICAL_TOKEN:-}" ]] && INF+=(--token "$INFISICAL_TOKEN")
CURL=(curl -sS)
[[ "${KEYCLOAK_INSECURE_SKIP_VERIFY:-true}" == "true" ]] && CURL+=(-k)

# secret_to_file NAME FILE: the secret's value into FILE; fails if absent.
secret_to_file() {
  "${INF[@]}" secrets get "$1" --plain >"$2" 2>/dev/null && [[ -s "$2" ]]
}
# store NAME FILE: set NAME from FILE's content (value never on argv).
store() {
  "${INF[@]}" secrets set "$1=@$2" >/dev/null 2>&1
}

# login USERFILE PASSFILE -> access token on stdout, or the refusal on stderr.
login() {
  local out code
  out="$("${CURL[@]}" -w '\n%{http_code}' -d grant_type=password -d client_id=admin-cli \
    --data-urlencode "username@$1" --data-urlencode "password@$2" \
    "${KC_BASE}/realms/master/protocol/openid-connect/token")" || { echo "cannot reach Keycloak" >&2; return 1; }
  code="${out##*$'\n'}"; out="${out%$'\n'*}"
  if [[ "$code" != 200 ]]; then
    echo "HTTP ${code}: $(jq -r '[.error, .error_description] | map(select(. != null)) | join(": ")' <<<"$out" 2>/dev/null)" >&2
    return 1
  fi
  jq -r .access_token <<<"$out"
}

# set_password TOKEN USERID PASSFILE: reset the user's password (permanent).
set_password() {
  jq -Rs '{type: "password", temporary: false, value: .}' <"$3" >"$TMP/cred.json"
  local code
  code="$("${CURL[@]}" -o /dev/null -w '%{http_code}' -X PUT -H "Authorization: Bearer $1" \
    -H 'Content-Type: application/json' --data-binary "@$TMP/cred.json" \
    "${KC_BASE}/admin/realms/master/users/$2/reset-password")"
  rm -f "$TMP/cred.json"
  [[ "$code" == 204 ]]
}

log "Keycloak ${KC_BASE}; Infisical ${DOMAIN} project ${PROJECT} env ${ENVIRONMENT} path ${SPATH}"
if [[ "$MODE" == dry ]]; then
  log "dry run: would prove the stored login, store a new password as KEYCLOAK_ADMIN_PASS_PENDING, set it in Keycloak, verify it, then promote it to KEYCLOAK_ADMIN_PASS"
  exit 0
fi

# 0. An interrupted rotation leaves PENDING behind: never overwrite it blind.
if secret_to_file KEYCLOAK_ADMIN_PASS_PENDING "$TMP/pending" 2>/dev/null; then
  log "KEYCLOAK_ADMIN_PASS_PENDING exists from an interrupted rotation. Check which password Keycloak accepts (this script with --check, then with PENDING as the password), set KEYCLOAK_ADMIN_PASS to it and delete PENDING."
  exit 7
fi

# 1. The current login must work before anything changes.
secret_to_file KEYCLOAK_ADMIN "$TMP/user" || { log "KEYCLOAK_ADMIN is not in Infisical ${SPATH}"; exit 2; }
secret_to_file KEYCLOAK_ADMIN_PASS "$TMP/old" || { log "KEYCLOAK_ADMIN_PASS is not in Infisical ${SPATH}"; exit 2; }
printf '%s' "$(cat "$TMP/user")" >"$TMP/user.trim"; mv "$TMP/user.trim" "$TMP/user"
if ! TOKEN="$(login "$TMP/user" "$TMP/old" 2>"$TMP/why")"; then
  log "the stored admin login was refused ($(cat "$TMP/why")); nothing changed"
  exit 2
fi
log "stored admin login works"
[[ "$MODE" == check ]] && exit 0

USER_ID="$("${CURL[@]}" -H "Authorization: Bearer ${TOKEN}" --get --data-urlencode "username@$TMP/user" \
  --data-urlencode exact=true "${KC_BASE}/admin/realms/master/users" | jq -r '.[0].id // empty')"
[[ -n "$USER_ID" ]] || { log "admin user not found in the master realm; nothing changed"; exit 2; }

# 2. The new password is safe in Infisical before Keycloak changes.
openssl rand -base64 36 | tr -d '\n/+=' | head -c 40 >"$TMP/new"
store KEYCLOAK_ADMIN_PASS_PENDING "$TMP/new" || { log "could not store the new password in Infisical; nothing changed"; exit 3; }
log "new password stored as KEYCLOAK_ADMIN_PASS_PENDING"

# 3. Keycloak.
if ! set_password "$TOKEN" "$USER_ID" "$TMP/new"; then
  "${INF[@]}" secrets delete KEYCLOAK_ADMIN_PASS_PENDING >/dev/null 2>&1 || true
  log "Keycloak refused the password change; the old password is still valid"
  exit 4
fi
log "password changed in Keycloak"

# 4. Prove the new one works; put the old one back if not.
if ! login "$TMP/user" "$TMP/new" >/dev/null 2>"$TMP/why"; then
  if set_password "$TOKEN" "$USER_ID" "$TMP/old"; then
    "${INF[@]}" secrets delete KEYCLOAK_ADMIN_PASS_PENDING >/dev/null 2>&1 || true
    log "the new password did not verify ($(cat "$TMP/why")); the old one was restored"
  else
    log "the new password did not verify and the old one could not be restored: KEYCLOAK_ADMIN_PASS_PENDING holds the password Keycloak now has"
  fi
  exit 5
fi
log "new password verified"

# 5. Promote. Infisical keeps the previous KEYCLOAK_ADMIN_PASS as a version.
if ! store KEYCLOAK_ADMIN_PASS "$TMP/new"; then
  log "Keycloak has the new password but promoting it failed: it is in KEYCLOAK_ADMIN_PASS_PENDING - copy it to KEYCLOAK_ADMIN_PASS and delete PENDING"
  exit 6
fi
"${INF[@]}" secrets delete KEYCLOAK_ADMIN_PASS_PENDING >/dev/null 2>&1 || log "note: could not delete KEYCLOAK_ADMIN_PASS_PENDING (it equals the new password); delete it"
log "rotation complete"
