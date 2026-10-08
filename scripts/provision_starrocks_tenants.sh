#!/bin/bash
# Provision per-tenant StarRocks databases, tables, principals and grants.
#
# Decision A: tenant isolation in StarRocks is database-per-tenant with
# DB-scoped grants. Verified on 3.3.22 that this actually enforces:
#
#   own-db   stream load  -> HTTP 200 Status=Success
#   cross-db stream load  -> HTTP 401 Access denied
#   cross-db SELECT       -> ERROR 5203 Access denied
#
# Note the asymmetry that matters for the loader: a cross-DB stream load answers
# 401 with no JSON body, while a bad value answers 200 with a JSON Status. The
# loader must treat 401 as an authorization/routing failure (DLQ the batch), never
# as a retryable one -- the same batch will never succeed on retry.
#
# Postgres has NO structural tenant boundary today (alpha is one shared database
# discriminated by a tenant_id column, and tenant_datasource_binding is empty).
# This script is therefore the only place tenant isolation is actually enforced
# on the analytics side.
#
# Idempotent. Safe to re-run: existing databases, tables, users and credential
# files are left alone, so a tenant's password never changes underneath a running
# loader.
#
# Usage:
#   scripts/provision_starrocks_tenants.sh [--dry-run] [--allow-drift]
#
# Env:
#   PGHOST PGPORT PGUSER PGPASSWORD PGDATABASE   source of the tenant list
#   SR_HOST SR_PORT SR_USER SR_PASSWORD           StarRocks admin (MySQL protocol)
#   SR_TENANT_DIR                                 credential output directory
set -euo pipefail

ALLOW_DRIFT=0

# 127.0.0.1 by default: this host's pg_hba allows loopback without TLS, but
# requires a client certificate for any other address. Point PGHOST elsewhere and
# set PGSSLMODE=verify-full plus PGSSLCERT/PGSSLKEY if you want the remote path.
PGHOST="${PGHOST:-127.0.0.1}"
PGPORT="${PGPORT:-5432}"
PGUSER="${PGUSER:-postgres}"
PGDATABASE="${PGDATABASE:-alpha}"
PGSSLMODE="${PGSSLMODE:-}"
SR_HOST="${SR_HOST:-127.0.0.1}"
SR_PORT="${SR_PORT:-9030}"
SR_USER="${SR_USER:-root}"
SR_PASSWORD="${SR_PASSWORD:-}"
SR_TENANT_DIR="${SR_TENANT_DIR:-/etc/uisce/sr-tenants}"

DRY_RUN=0
for arg in "$@"; do
  case "$arg" in
    --dry-run) DRY_RUN=1 ;;
    --allow-drift) ALLOW_DRIFT=1 ;;
    -h|--help)
      # The header comment is the usage text; print everything above `set -euo`.
      sed -n '2,/^set -euo/p' "$0" | sed '$d'
      exit 0
      ;;
    *)
      echo "unknown argument: $arg (try --help)" >&2
      exit 2
      ;;
  esac
done

# list_contains answers "is this exact line in this newline-separated list".
#
# It walks the list with `read` and compares whole lines. Two alternatives were
# tried and rejected: `grep -Fxq` SIGPIPEs its writer under `set -o pipefail` and
# kills the script with exit 141, and a `case "*\n$1\n*"` glob degenerates to a
# substring test, which would make every lookup match. Exact equality is the only
# behaviour safe for a check whose whole job is to report a precise difference.
#
# The list is fed through process substitution rather than a here-string: `<<<`
# writes a temporary file, which fails outright in a sandboxed or read-only
# environment, and this function runs on every provisioning pass.
list_contains() {
  local needle="$1" line
  [ -z "$needle" ] && return 1
  while IFS= read -r line; do
    [ "$line" = "$needle" ] && return 0
  done < <(printf '%s\n' "$2")
  return 1
}

# Tables mirrored into every tenant database. Keep in step with
# migrations/starrocks/002_cdc_orm_tables.sql -- that file owns the column
# definitions; this is only the set of names, and a missing table here shows up
# immediately as a stream-load failure rather than as a missing column.
TABLES=(orm_order orm_execution orm_placement orm_order_allocation orm_execution_allocation)

# pgq runs one query against the source database and prints tuples.
#
# Two details are load-bearing and were both bugs first:
#   - psql takes the port as -p. -P is --pset and would parse the port as a variable
#     name, so the connection never happens.
#   - the SSL options are expanded as "${arr[@]+...}". A plain "${arr[@]}" on an
#     empty array aborts under `set -u` on bash < 4.4, and PGSSLMODE is empty on the
#     loopback path this script is actually run on.
pgq() {
  local -a pgssl=()
  [ -n "$PGSSLMODE" ] && pgssl=(--sslmode="$PGSSLMODE")
  PGPASSWORD="${PGPASSWORD:-postgres}" psql -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" \
    -d "$PGDATABASE" -tAF'|' "${pgssl[@]+"${pgssl[@]}"}" -c "$1"
}

sr() {
  if [ "$DRY_RUN" = 1 ]; then
    printf 'DRY-RUN %s\n' "$1"
    return 0
  fi
  mysql -h "$SR_HOST" -P "$SR_PORT" -u "$SR_USER" ${SR_PASSWORD:+-p"$SR_PASSWORD"} -N -e "$1"
}

# exists answers "does this object already exist". It captures rather than pipes to
# grep -q: an early-exiting grep sends SIGPIPE to the writer, which under
# `set -o pipefail` kills the whole script with exit 141. It also must not run the
# existence query in dry-run mode, where sr() echoes the SQL text -- and that text
# would itself contain the "1" being searched for.
exists() {
  local kind="$1" name="$2" result
  if [ "$DRY_RUN" = 1 ]; then
    return 1
  fi
  case "$kind" in
    db) result=$(sr "SELECT count(*) FROM information_schema.schemata WHERE schema_name='${name}'") ;;
    tbl) result=$(sr "SELECT count(*) FROM \`${name%%.*}\`.information_schema.tables WHERE table_name='${name##*.}'") ;;
    *) return 1 ;;
  esac
  [ "$result" -gt 0 ] 2>/dev/null
}

# slug converts a tenant uuid to the identifier form StarRocks accepts and that
# cube_materializer.go already produces: hyphens removed.
slug() { echo "$1" | tr -d '-'; }

# password_for is stable: a tenant's password is read back from its credential
# file if one exists, so re-running never invalidates a running loader's DSN.
# openssl reads a fixed number of bytes and exits without closing a pipe early --
# `tr ... | head -c N` sends SIGPIPE to tr, which under pipefail kills the script.
password_for() {
  local uuid="$1" file="$SR_TENANT_DIR/$1.json"
  if [ -f "$file" ]; then
    sed -n 's/.*"password"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' "$file"
  elif command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 16
  else
    LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | dd bs=32 count=1 2>/dev/null || true
  fi
}

main() {
  if [ "$DRY_RUN" != 1 ]; then
    mkdir -p "$SR_TENANT_DIR"
    chmod 700 "$SR_TENANT_DIR"
  fi

  # Only active tenants are provisioned; a deprovisioned tenant's database is
  # left in place deliberately (dropping it would silently destroy history).
  local tenants
  tenants=$(pgq "select id, name from public.tenants where status = 'active' order by name;")

  local created=0 reused=0
  while IFS='|' read -r uuid name; do
    [ -z "${uuid:-}" ] && continue
    local db user pass
    db="tenant_$(slug "$uuid")"
    user="${db}_app"

    if exists db "$db"; then
      reused=$((reused + 1))
    else
      sr "CREATE DATABASE IF NOT EXISTS \`${db}\`"
      created=$((created + 1))
    fi

    pass=$(password_for "$uuid")

    for t in "${TABLES[@]}"; do
      # Definitions live in 002_cdc_orm_tables.sql; create the table by replaying
      # it with the oms. database prefix swapped for this tenant's database, so
      # the two cannot drift.
      if exists tbl "${db}.${t}"; then
        continue
      fi
      if [ "$DRY_RUN" = 1 ]; then
        printf 'DRY-RUN table %s.%s\n' "$db" "$t"
        continue
      fi
      # Column definitions are supplied by the caller via TENANT_DDL_FILE.
      : "${TENANT_DDL_FILE:?set TENANT_DDL_FILE to migrations/starrocks/002_cdc_orm_tables.sql}"
      sed -e "s/oms\.${t}/\`${db}\`.\`${t}\`/g" -e "s/^CREATE DATABASE IF NOT EXISTS oms;//" \
        "$TENANT_DDL_FILE" \
        | mysql -h "$SR_HOST" -P "$SR_PORT" -u "$SR_USER" ${SR_PASSWORD:+-p"$SR_PASSWORD"}
    done

    # Idempotent user creation. CREATE USER IF NOT EXISTS keeps an existing
    # password, which is why password_for reads the stored one back.
    sr "CREATE USER IF NOT EXISTS '${user}'@'%' IDENTIFIED BY '${pass}'"
    # DB-scoped only. This is the whole isolation boundary: nothing grants this
    # principal anything in another tenant's database.
    sr "GRANT ALL ON \`${db}\`.* TO '${user}'@'%'"

    if [ "$DRY_RUN" != 1 ]; then
      cat > "$SR_TENANT_DIR/$uuid.json" <<JSON
{
  "tenant_id": "$uuid",
  "tenant_name": "$name",
  "database": "$db",
  "user": "$user",
  "password": "$pass",
  "dsn": "${user}:${pass}@tcp(${SR_HOST}:${SR_PORT})/${db}",
  "key_columns": ["id"]
}
JSON
      chmod 600 "$SR_TENANT_DIR/$uuid.json"
    fi

    printf '%-28s %-32s %s\n' "$name" "$db" "$(sr "SHOW GRANTS FOR '${user}'@'%'" | tail -1)"
  done < <(printf '%s\n' "$tenants")

  echo
  echo "created=$created reused=$reused credentials=${SR_TENANT_DIR}"

  report_drift
}

# report_drift diffs the credential store against the authoritative tenant list.
#
# Without it, Postgres-tenants <-> StarRocks-principals drift is something the loader
# discovers as an unexplained DLQ entry at 3am: a tenant exists in Postgres, no
# credential file was written, and every one of its rows is dead-lettered with
# "no provisioned route". Here it is a printed diff instead.
#
# The orphan direction is reported too. Deprovisioning deliberately never drops a
# tenant's StarRocks data, so a leftover credential file is expected rather than a
# fault -- but it is only explicable if we say which tenant it belongs to and what
# that tenant's status is upstream.
report_drift() {
  # Every tenant, not just the active ones: an orphan is only explicable if we know
  # the tenant still exists and is merely inactive.
  #
  # Plain id lists, never printable strings: status text leaking into a lookup turns
  # an exact comparison into a substring test that matches everything.
  local all
  all=$(pgq "select id, coalesce(status,'?') from public.tenants order by id;") || all=""

  local known="" inactive="" id status
  while IFS='|' read -r id status; do
    [ -z "${id:-}" ] && continue
    known="${known}${id}"$'\n'
    [ "${status:-active}" != "active" ] && inactive="${inactive}${id}"$'\n'
  done < <(printf '%s\n' "$all")

  local stored=""
  if [ -d "$SR_TENANT_DIR" ]; then
    local f base
    for f in "$SR_TENANT_DIR"/*.json; do
      [ -e "$f" ] || continue
      base=$(basename "$f" .json)
      stored="${stored}${base}"$'\n'
    done
  fi

  local active missing="" orphans="" notable=""
  active=$(pgq "select id from public.tenants where status = 'active' order by id;") || active=""

  # An active tenant with no credential file has no route at all: the loader
  # dead-letters every one of its rows. That is the failure this check exists for.
  while IFS= read -r id; do
    [ -z "${id:-}" ] && continue
    list_contains "$id" "$stored" || missing="${missing}  ${id}"$'\n'
  done < <(printf '%s\n' "$active")

  while IFS= read -r id; do
    [ -z "${id:-}" ] && continue
    if ! list_contains "$id" "$known"; then
      orphans="${orphans}  ${id} (no longer in Postgres -- data left in place deliberately)"$'\n'
    elif list_contains "$id" "$inactive"; then
      # Tested per tenant, not with `[ -n "$inactive" ]`: that is true whenever any
      # inactive tenant exists and would label every credential in the store.
      notable="${notable}  ${id} (inactive upstream, credential retained)"$'\n'
    fi
  done < <(printf '%s\n' "$stored")

  if [ -z "$missing" ] && [ -z "$orphans" ] && [ -z "$notable" ]; then
    echo "drift: none (credential store matches public.tenants)"
    return 0
  fi

  echo
  echo "=== credential-store drift ==="
  if [ -n "$missing" ]; then
    echo "active tenants with NO StarRocks route (their rows will DLQ):"
    printf '%s' "$missing"
  fi
  if [ -n "$orphans" ]; then
    echo "credentials with no matching tenant:"
    printf '%s' "$orphans"
  fi
  if [ -n "$notable" ]; then
    echo "credentials for inactive tenants:"
    printf '%s' "$notable"
  fi

  if [ "$ALLOW_DRIFT" = 1 ]; then
    echo "(--allow-drift given: reporting only, exiting 0)"
    return 0
  fi
  echo "Provisioning completed; drift above is unresolved. Re-run, or pass --allow-drift to acknowledge."
  return 1
}

main "$@"