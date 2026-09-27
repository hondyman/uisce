#!/usr/bin/env bash
# smoke-sql-fix.sh — verify each SQL fix.
#
# STATUS: ROUTES UNWIRED — this script will NOT work until RegisterDynamicHandlers
#         is wired into server/main.go. Currently (2026-09-19) the handler functions
#         exist in backend/internal/handlers/dynamic_handlers.go:1915 but are never
#         mounted to the router. All requests return 404.
#
#         To re-enable: mount RegisterDynamicHandlers in backend/cmd/server/main.go
#         then run: BASE_URL=http://localhost:8080 ./scripts/refactor/smoke-sql-fix.sh
#
# REQUIRES:
#   - Backend running at $BASE_URL (default http://localhost:8080).
#     Override with: BASE_URL=http://localhost:9000 ./smoke-sql-fix.sh
#   - REJECTION tests need only the backend running (no DB).
#   - HAPPY-PATH tests require the backend AND a reachable database
#     (they hit SELECT DISTINCT).
#
# Mechanical sanity step (run BEFORE first execution):
#   bash -n scripts/refactor/smoke-sql-fix.sh           # syntax check
#   command -v jq || { echo "jq not installed: brew install jq"; exit 1; }
#
# Exits non-zero on any failure.

set -euo pipefail

BASE="${BASE_URL:-http://localhost:8080}"

command -v jq >/dev/null || { echo "FAIL: jq not installed (brew install jq)"; exit 1; }

fail() { echo "FAIL: $1"; exit 1; }
pass() { echo "PASS: $1"; }

# Catch malformed payloads before they reach the wire and produce a confusing 400.
valid_json() {
  echo "$1" | jq -e . >/dev/null 2>&1 \
    || fail "smoke bug: malformed JSON payload: $1"
}

http_code() {
  curl -sk -o /dev/null -w "%{http_code}" "$@"
}

# Tightened: requires EXACTLY 400. Catches: (a) fix-not-applied (200),
# (b) misrouted (404), (c) DB-down on a rejection (500).
rejects() {
  local method="$1" path="$2" body="${3:-}" desc="$4"
  valid_json "$body"
  local code
  if [[ -n "$body" ]]; then
    code=$(http_code -X "$method" -H 'Content-Type: application/json' -d "$body" "$BASE$path")
  else
    code=$(http_code -X "$method" "$BASE$path")
  fi
  [[ "$code" == "400" ]] || fail "$desc: expected 400, got $code ($method $path)"
  pass "400 $method $path ($desc)"
}

# Happy path requires a reachable DB (SELECT DISTINCT must succeed).
allows() {
  local method="$1" path="$2" body="${3:-}" desc="$4"
  valid_json "$body"
  local code
  if [[ -n "$body" ]]; then
    code=$(http_code -X "$method" -H 'Content-Type: application/json' -d "$body" "$BASE$path")
  else
    code=$(http_code -X "$method" "$BASE$path")
  fi
  [[ "$code" =~ ^2 ]] || fail "$desc: expected 2xx, got $code ($method $path) — DB reachable?"
  pass "$code $method $path ($desc)"
}

echo "=== C1: GenerateDynamicMeasures ==="
rejects GET '/dynamic/measures/generate?table=users&column=id%20UNION%20SELECT%20password%20FROM%20users--' '' 'rejects SQLi in column'
rejects GET '/dynamic/measures/generate?table=unknown.tbl&column=x'                              '' 'rejects unknown table'
# Case is normalized (lowercased) and accepted — NOT rejected. Documented behavior.
allows  GET '/dynamic/measures/generate?table=OMS.Account&column=STATUS'                          '' 'normalizes case then accepts allow-listed source'

echo "=== C2: CreateDynamicUnion ==="
rejects POST '/dynamic/unions/' '{"name":"u","description":"","source_tables":["users; DROP TABLE x--"],"union_type":"UNION ALL","table_aliases":{},"owner":"me"}' 'rejects malicious source_table'
rejects POST '/dynamic/unions/' '{"name":"u","description":"","source_tables":["oms.account"],"union_type":"UNION ALL; DROP TABLE--","table_aliases":{},"owner":"me"}' 'rejects malicious union_type'
rejects POST '/dynamic/unions/' '{"name":"u","description":"","source_tables":["oms.account"],"union_type":"UNION ALL","table_aliases":{"oms.account":"a.b.c"},"owner":"me"}' 'rejects dot in alias'
allows  POST '/dynamic/unions/' '{"name":"u","description":"","source_tables":["oms.account"],"union_type":"UNION ALL","table_aliases":{"oms.account":"a"},"owner":"me"}' 'happy path returns 2xx'

echo "=== C3: CreateStringTimeDimension ==="
rejects POST '/dynamic/time-dimensions/' '{"name":"t","description":"","source_column":"x) UNION SELECT 1 --","source_table":"oms.account","date_format":"YYYY","time_format":"","timezone":"UTC","parsing_function":"TO_TIMESTAMP","owner":"me"}' 'rejects SQLi in source_column'
rejects POST '/dynamic/time-dimensions/' '{"name":"t","description":"","source_column":"status","source_table":"oms.account","date_format":"YYYY'\''); DROP TABLE x;--","time_format":"","timezone":"UTC","parsing_function":"TO_TIMESTAMP","owner":"me"}' 'rejects quote-breakout in date_format'
rejects POST '/dynamic/time-dimensions/' '{"name":"t","description":"","source_column":"status","source_table":"oms.account","date_format":"YYYY","time_format":"","timezone":"america/new_york","parsing_function":"TO_TIMESTAMP","owner":"me"}' 'rejects lowercased timezone (case-significant)'
rejects POST '/dynamic/time-dimensions/' '{"name":"t","description":"","source_column":"status","source_table":"oms.account","date_format":"YYYY","time_format":"","timezone":"UTC","parsing_function":"PARSE_TIMESTAMP","owner":"me"}' 'rejects invented PARSE_TIMESTAMP'
allows  POST '/dynamic/time-dimensions/' '{"name":"t","description":"","source_column":"status","source_table":"oms.account","date_format":"YYYY-MM-DD","time_format":"","timezone":"UTC","parsing_function":"TO_TIMESTAMP","owner":"me"}' 'happy path with real-world date_format returns 2xx'

echo "=== C4: CreateCustomGranularity ==="
rejects POST '/dynamic/granularities/' '{"name":"g","description":"","dimension":"x) UNION SELECT 1 --","interval":"day","offset_days":0,"offset_hours":0,"owner":"me"}' 'rejects SQLi in dimension'
rejects POST '/dynamic/granularities/' '{"name":"g","description":"","dimension":"status","interval":"day); DROP TABLE--","offset_days":0,"offset_hours":0,"owner":"me"}' 'rejects malicious interval'
allows  POST '/dynamic/granularities/' '{"name":"g","description":"","dimension":"status","interval":"day","offset_days":0,"offset_hours":0,"owner":"me"}' 'happy path returns 2xx'

echo ""
echo "ALL SMOKE TESTS PASSED."
