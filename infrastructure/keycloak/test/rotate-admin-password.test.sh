#!/usr/bin/env bash
# Tests rotate-admin-password.sh against a fake Keycloak (fake_keycloak.py)
# and a fake Infisical CLI (secrets as files). Never touches a real server.
#   infrastructure/keycloak/test/rotate-admin-password.test.sh
set -uo pipefail
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="${HERE}/../rotate-admin-password.sh"
WORK="$(mktemp -d)"; KC_PID=""
stop_kc() { if [[ -n "$KC_PID" ]]; then kill "$KC_PID" 2>/dev/null; wait "$KC_PID" 2>/dev/null; fi; KC_PID=""; }
trap 'stop_kc; rm -rf "$WORK"' EXIT
PORT=18$((RANDOM % 900 + 100))
fails=0

# Fake infisical: "secrets get NAME --plain", "secrets set NAME=@file",
# "secrets delete NAME"; FAKE_INF_FAIL_SET=NAME makes that set fail.
cat >"$WORK/infisical" <<'EOF'
#!/usr/bin/env bash
store="$FAKE_INF_DIR"; cmd="" sub="" args=()
while [[ $# -gt 0 ]]; do
  case "$1" in
    --domain|--projectId|--env|--path|--token) shift 2 ;;
    --silent|--plain) shift ;;
    *) args+=("$1"); shift ;;
  esac
done
cmd="${args[0]}"; sub="${args[1]}"; arg="${args[2]:-}"
case "$sub" in
  get) [[ -f "$store/$arg" ]] && cat "$store/$arg" && exit 0; exit 1 ;;
  set) name="${arg%%=*}"; val="${arg#*=}"
       [[ "${FAKE_INF_FAIL_SET:-}" == "$name" ]] && exit 1
       [[ "$val" == @* ]] || { echo "value on argv" >&2; exit 9; }
       cp "${val#@}" "$store/$name"; cp "${val#@}" "$store/$name.v$(date +%s%N)" ;;
  delete) rm -f "$store/$arg" ;;
esac
EOF
chmod +x "$WORK/infisical"

start_kc() { # $1 = behaviour
  stop_kc; sleep 0.2
  python3 "$HERE/fake_keycloak.py" "$PORT" "$WORK/kc_password" "$1" & KC_PID=$!
  for _ in $(seq 1 30); do curl -s "http://127.0.0.1:$PORT/health" >/dev/null && return; sleep 0.1; done
}
setup() { # $1 = Keycloak behaviour
  rm -rf "$WORK/inf"; mkdir -p "$WORK/inf"
  printf admin >"$WORK/inf/KEYCLOAK_ADMIN"; printf 'old-pass' >"$WORK/inf/KEYCLOAK_ADMIN_PASS"
  printf 'old-pass' >"$WORK/kc_password"
  start_kc "$1"
}
run() {
  FAKE_INF_DIR="$WORK/inf" INFISICAL_BIN="$WORK/infisical" KEYCLOAK_BASE="http://127.0.0.1:$PORT" \
    "$SCRIPT" "$@" >"$WORK/out" 2>&1
}
expect() { # name, want-exit, got-exit, condition-description, condition
  if [[ "$2" == "$3" ]] && eval "$5"; then echo "ok   $1"; else
    echo "FAIL $1 (exit $3, want $2; $4)"; sed 's/^/     /' "$WORK/out"; fails=$((fails + 1)); fi
}
kc() { cat "$WORK/kc_password"; }
inf() { cat "$WORK/inf/$1" 2>/dev/null; }

setup ok; run; rc=$?
expect "rotates" 0 $rc "keycloak and infisical agree on a new password, pending gone" \
  '[[ "$(kc)" != old-pass && "$(kc)" == "$(inf KEYCLOAK_ADMIN_PASS)" && ! -f "$WORK/inf/KEYCLOAK_ADMIN_PASS_PENDING" ]]'
expect "never prints a password" 0 $rc "output must not contain it" '! grep -q -- "$(kc)" "$WORK/out" && ! grep -q old-pass "$WORK/out"'
expect "new password is strong" 0 $rc ">= 32 chars" '[[ $(kc | wc -c) -ge 32 ]]'

setup ok; printf 'wrong' >"$WORK/inf/KEYCLOAK_ADMIN_PASS"; run; rc=$?
expect "stops when the stored login is refused" 2 $rc "nothing changed" '[[ "$(kc)" == old-pass && ! -f "$WORK/inf/KEYCLOAK_ADMIN_PASS_PENDING" ]]'

setup ok; FAKE_INF_FAIL_SET=KEYCLOAK_ADMIN_PASS_PENDING run; rc=$?
expect "stops when the new password can't be stored" 3 $rc "keycloak untouched" '[[ "$(kc)" == old-pass ]]'

setup reject_reset; run; rc=$?
expect "keycloak refuses the change" 4 $rc "old still valid, pending removed" '[[ "$(kc)" == old-pass && ! -f "$WORK/inf/KEYCLOAK_ADMIN_PASS_PENDING" && "$(inf KEYCLOAK_ADMIN_PASS)" == old-pass ]]'

setup break_new_login; run; rc=$?
expect "new password doesn't verify -> old restored" 5 $rc "keycloak back to old, infisical unchanged" '[[ "$(kc)" == old-pass && "$(inf KEYCLOAK_ADMIN_PASS)" == old-pass ]]'

setup ok; FAKE_INF_FAIL_SET=KEYCLOAK_ADMIN_PASS run; rc=$?
expect "promote fails -> the live password is in PENDING" 6 $rc "pending == keycloak" '[[ "$(inf KEYCLOAK_ADMIN_PASS_PENDING)" == "$(kc)" && "$(kc)" != old-pass ]]'

setup ok; printf 'x' >"$WORK/inf/KEYCLOAK_ADMIN_PASS_PENDING"; run; rc=$?
expect "refuses to run over an interrupted rotation" 7 $rc "nothing changed" '[[ "$(kc)" == old-pass ]]'

setup ok; run --check; rc=$?
expect "--check only proves the login" 0 $rc "nothing changed" '[[ "$(kc)" == old-pass && "$(inf KEYCLOAK_ADMIN_PASS)" == old-pass ]]'

echo; [[ $fails -eq 0 ]] && echo "all passed" || { echo "$fails failed"; exit 1; }
