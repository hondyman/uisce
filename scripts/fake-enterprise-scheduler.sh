#!/usr/bin/env bash
#
# fake-enterprise-scheduler.sh — Plays Tidal / Control-M / AutoSys for local
# testing: runs a Uisce schedule the way their command job would, with
# uisce-job, and reacts to the exit code the way a job definition would.
#
#   scripts/fake-enterprise-scheduler.sh --schedule "Order volume - London close"
#   scripts/fake-enterprise-scheduler.sh --schedule <id|name> --retry     # simulate an agent restart
#   scripts/fake-enterprise-scheduler.sh --schedule <id|name> --run-id TIDAL-42 --no-wait
#
# --retry runs the job twice with the same run id, like a scheduler re-running
# after an agent restart: the second call must return the first run, not
# start another.
#
# Uses the same settings a real agent would (defaults suit local testing):
#   UISCE_URL (http://localhost:8080), UISCE_TOKEN_URL, UISCE_CLIENT_ID
#   (tidal-northwind), UISCE_CLIENT_SECRET_FILE (~/.uisce/<client>.secret),
#   UISCE_REGION (us-west), UISCE_JOB (path to uisce-job; built if absent)
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SCHEDULE="" RUN_ID="" RETRY=0 WAIT=1 TIMEOUT="30m" SYSTEM="tidal" JOB_NAME="FAKE_TIDAL_JOB"
while [[ $# -gt 0 ]]; do
  case "$1" in
    --schedule) SCHEDULE="$2"; shift 2 ;;
    --run-id) RUN_ID="$2"; shift 2 ;;
    --retry) RETRY=1; shift ;;
    --no-wait) WAIT=0; shift ;;
    --timeout) TIMEOUT="$2"; shift 2 ;;
    --system) SYSTEM="$2"; shift 2 ;;
    --job-name) JOB_NAME="$2"; shift 2 ;;
    -h|--help) sed -n '2,20p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'; exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done
[[ -n "$SCHEDULE" ]] || { echo "--schedule <id or name> is required" >&2; exit 2; }

export UISCE_URL="${UISCE_URL:-http://localhost:8080}"
export UISCE_TOKEN_URL="${UISCE_TOKEN_URL:-https://100.84.50.65:8443/realms/uisce/protocol/openid-connect/token}"
export UISCE_CLIENT_ID="${UISCE_CLIENT_ID:-tidal-northwind}"
export UISCE_CLIENT_SECRET_FILE="${UISCE_CLIENT_SECRET_FILE:-$HOME/.uisce/${UISCE_CLIENT_ID}.secret}"
export UISCE_REGION="${UISCE_REGION:-us-west}"
[[ -r "$UISCE_CLIENT_SECRET_FILE" ]] || { echo "no client secret at $UISCE_CLIENT_SECRET_FILE (infrastructure/keycloak/create-scheduler-client.sh makes it)" >&2; exit 2; }

JOB="${UISCE_JOB:-}"
if [[ -z "$JOB" ]]; then
  JOB="$REPO/backend/bin/uisce-job"
  if [[ ! -x "$JOB" ]]; then
    echo "[scheduler] building uisce-job ..."
    (cd "$REPO/backend" && go build -o bin/uisce-job ./cmd/uisce-job) || exit 2
  fi
fi

# A run id like the scheduler's own: the idempotency key for this job run.
RUN_ID="${RUN_ID:-$(printf %s "$SYSTEM" | tr a-z A-Z)-$(date +%Y%m%d-%H%M%S)-$$}"

run_job() { # attempt number
  echo "[scheduler] job ${JOB_NAME} run ${RUN_ID} (attempt $1): starting"
  local args=(run --schedule "$SCHEDULE" --key "$RUN_ID" --system "$SYSTEM" --ref "$JOB_NAME" --timeout "$TIMEOUT")
  (( WAIT )) && args+=(--wait)
  "$JOB" "${args[@]}"
  local rc=$?
  case $rc in
    0) echo "[scheduler] run ${RUN_ID}: COMPLETED NORMALLY (exit 0)" ;;
    1) echo "[scheduler] run ${RUN_ID}: COMPLETED ABNORMALLY - the Uisce run failed (exit 1); alert the owner" ;;
    2) echo "[scheduler] run ${RUN_ID}: SKIPPED - business calendar closed (exit 2); dependents may continue" ;;
    3) echo "[scheduler] run ${RUN_ID}: TIMED OUT waiting (exit 3); re-running with the same run id only waits again" ;;
    4) echo "[scheduler] run ${RUN_ID}: NOT AUTHORIZED (exit 4); check the client and its roles" ;;
    5) echo "[scheduler] run ${RUN_ID}: MISCONFIGURED job (exit 5)" ;;
    *) echo "[scheduler] run ${RUN_ID}: ERROR from Uisce (exit $rc)" ;;
  esac
  return $rc
}

run_job 1; rc=$?
if (( RETRY )); then
  echo "[scheduler] simulating an agent restart: re-running run ${RUN_ID} with the same key"
  run_job 2; rc2=$?
  if [[ $rc2 -ne $rc ]]; then
    echo "[scheduler] WARNING: the retry ended differently ($rc2 vs $rc)"
  fi
  echo "[scheduler] check Build > Automation > Schedules > Run history: one run for key ${RUN_ID}, not two"
  rc=$rc2
fi
exit $rc
