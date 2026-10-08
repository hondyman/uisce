#!/bin/bash
# Regression tests for scripts/provision_starrocks_tenants.sh.
#
# The drift report is the only thing standing between "a tenant exists in Postgres
# with no StarRocks route" and "every one of its rows is dead-lettered with no
# explanation". It has to be right, and it has to be right about the difference
# between a credential that is missing, orphaned, and merely inactive.
#
# The script itself ends with `main "$@"`, so everything but that last line is
# evaluated here rather than sourced -- that is the only way to test the functions
# without running a provisioning pass against a live cluster.
#
# Usage: scripts/test_provision_drift.sh
set -uo pipefail

HERE="$(cd "$(dirname "$0")" && pwd)"
SRC="$HERE/provision_starrocks_tenants.sh"
WORK="$(mktemp -d)"
# Cleanup is best-effort: some sandboxes route rm through a trash helper that refuses
# paths outside the workspace, and a noisy failure here would mask the real result.
trap 'rm -rf "$WORK" >/dev/null 2>&1 || true' EXIT

A="99e99e99-99e9-49e9-89e9-99e99e99e999"
B="88e88e88-88e8-48e8-88e8-88e88e88e888"
C="77e77e77-77e7-47e7-87e7-77e77e77e777"

fail=0
ok()   { echo "ok:   $1"; }
bad()  { echo "FAIL: $1"; fail=1; }

# ---- list_contains ----
# Exact line matching. Every case below is one that a substring or glob
# implementation gets wrong, which is why they exist.

test_list_contains() {
  local desc="$1" want="$2" needle="$3" list="$4"
  list_contains "$needle" "$list"
  local rc=$?
  if [ "$rc" = "$want" ]; then ok "list_contains: $desc"; else bad "list_contains: $desc (want rc=$want got $rc)"; fi
}

# ---- psql stub ----
# The script builds its psql flags inside pgq(); the stub reads the query text out of
# the argument list so it stays correct even if the flags change.
psql() {
  local q=""
  while [ $# -gt 0 ]; do
    case "$1" in
      -c) q="$2"; shift 2 ;;
      *) shift ;;
    esac
  done
  case "$q" in
    *"coalesce(status"*) printf '%s|active\n%s|active\n%s|inactive\n' "$A" "$B" "$C" ;;
    *"where status = 'active'"*) printf '%s\n%s\n' "$A" "$B" ;;
    *) printf '\n' ;;
  esac
}

eval "$(sed '$d' "$SRC")"

# eval runs in this shell and the script body sets its own `set -euo pipefail`.
# report_drift returns 1 by design when it finds drift, which would abort the run.
set +e
set -u
set -o pipefail

echo "== list_contains =="
test_list_contains "exact match"          0 "$A" "$A"$'\n'"$B"
test_list_contains "absent"                1 "$C" "$A"$'\n'"$B"
test_list_contains "empty list"            1 "$A" ""
test_list_contains "empty needle"          1 ""  "$A"$'\n'"$B"
test_list_contains "prefix is not a match" 1 "$A" "${A:0:20}"$'\n'"$B"
test_list_contains "substring is not a match" 1 "99e9" "$A"$'\n'"$B"
test_list_contains "last line"             0 "$B" "$A"$'\n'"$B"

echo
echo "== report_drift =="
run_drift() { out=$(report_drift 2>&1); rc=$?; }
has() {
  case "$out" in
    *"$2"*) ok "$1" ;;
    *) bad "$1 (missing: $2)"; echo "--- got:"; echo "$out" ;;
  esac
}
hasnt() {
  case "$out" in
    *"$2"*) bad "$1 (unexpectedly present: $2)"; echo "--- got:"; echo "$out" ;;
    *) ok "$1" ;;
  esac
}
rc_is() { if [ "$rc" = "$2" ]; then ok "$1"; else bad "$1 (want rc=$2 got $rc)"; fi; }

# 1. An active tenant with no credential file has no route at all.
SR_TENANT_DIR="$WORK/store"; mkdir -p "$SR_TENANT_DIR"
: > "$SR_TENANT_DIR/$A.json"
: > "$SR_TENANT_DIR/$C.json"
run_drift
rc_is "unrouted active tenant fails the check" 1 "$rc"
has "names the unrouted tenant" "$B"
has "explains the retained inactive credential" "$C (inactive upstream, credential retained)"
hasnt "does not mislabel an active tenant as inactive" "$A (inactive upstream"

# 2. A credential whose tenant no longer exists.
SR_TENANT_DIR="$WORK/store2"; mkdir -p "$SR_TENANT_DIR"
: > "$SR_TENANT_DIR/deadbeef-0000-0000-0000-000000000000.json"
run_drift
rc_is "orphan credential fails the check" 1 "$rc"
has "reports the orphan" "deadbeef-0000-0000-0000-000000000000 (no longer in Postgres"

# 3. Store exactly matches the active tenants.
SR_TENANT_DIR="$WORK/store3"; mkdir -p "$SR_TENANT_DIR"
: > "$SR_TENANT_DIR/$A.json"
: > "$SR_TENANT_DIR/$B.json"
run_drift
rc_is "matching store passes" 0 "$rc"
has "says so" "drift: none"

# 4. --allow-drift reports but does not fail.
: > "$SR_TENANT_DIR/eeee1111-1111-4111-8111-111111111111.json"
ALLOW_DRIFT=1
run_drift
rc_is "--allow-drift exits zero" 0 "$rc"
has "still reports the orphan" "eeee1111-1111-4111-8111-111111111111"

# 5. Empty store against two active tenants.
ALLOW_DRIFT=0
SR_TENANT_DIR="$WORK/store4"; mkdir -p "$SR_TENANT_DIR"
run_drift
rc_is "empty store fails" 1 "$rc"
has "lists tenant A" "$A"
has "lists tenant B" "$B"

# 6. No credential directory at all is drift, not a crash.
SR_TENANT_DIR="$WORK/absent"
run_drift
rc_is "absent store fails" 1 "$rc"

echo
if [ "$fail" = 0 ]; then
  echo "ALL PASS"
else
  echo "FAILURES PRESENT"
fi
exit "$fail"