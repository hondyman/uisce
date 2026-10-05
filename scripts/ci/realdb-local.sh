#!/usr/bin/env bash
# Runs the real-database gate locally on a throwaway cluster with password (scram) auth, exactly as CI does.
# Needs initdb/pg_ctl on PATH. macOS: LC_ALL=en_US.UTF-8 is set here.
set -euo pipefail
export LC_ALL=en_US.UTF-8
d="$(mktemp -d)"; port="${REALDB_PORT:-55439}"
trap 'pg_ctl -D "$d/data" stop -m fast >/dev/null 2>&1 || true; rm -rf "$d"' EXIT
boot="$(python3 -c 'import secrets;print(secrets.token_hex(8))')"
echo "$boot" > "$d/pw"
initdb -D "$d/data" -U postgres --auth=scram-sha-256 --pwfile="$d/pw" >/dev/null
pg_ctl -D "$d/data" -o "-p $port -k $d" -l "$d/log" -w start >/dev/null
PGHOST=127.0.0.1 PGPORT="$port" PGUSER=postgres PGPASSWORD="$boot" bash "$(dirname "$0")/realdb-run.sh"
