#!/usr/bin/env bash
# harden-tenant-cluster.sh: close PUBLIC's default CONNECT on every database of a Postgres cluster
# that holds tenant databases (ADR-030).
#
# Why: provisioning a tenant with an "app" proves the tenant's role can connect to exactly one
# database, its own, and fails if any other database on the cluster lets PUBLIC connect (postgres,
# template1, alpha, legacy tenant_* databases). See docs/runbooks/tenant-cluster-hardening.md.
#
# Safe by default: this is a DRY RUN. It computes the whole change inside one transaction and rolls
# it back, so the plan is exact. It reports every ordinary login role that would lose access, and it
# REFUSES to apply while any such role is not covered by a --grant. Idempotent: on a hardened
# cluster it does nothing and exits 0.
#
# Usage:
#   harden-tenant-cluster.sh [--apply] [--grant DB=ROLE[,ROLE...]]... [--tenant-role-suffix SUFFIX]
#
#   --apply                      make the change (default: show it and change nothing)
#   --grant DB=ROLE[,ROLE...]    GRANT CONNECT ON DATABASE DB TO each ROLE first (repeatable). Use it
#                                for the roles that legitimately need a database, e.g.
#                                --grant alpha=uisce_app,uisce_gold_copy_sync
#   --tenant-role-suffix SUFFIX  roles ending in SUFFIX are tenant roles (default _app): they are
#                                EXPECTED to lose access to other databases and are not reported
#
# Connection: the usual libpq environment (PGHOST, PGPORT, PGUSER, PGPASSWORD, PGSSLMODE), as a
# superuser. PGDATABASE defaults to postgres, which is fine: this script only changes privileges.
#
# Exit codes: 0 nothing to do / applied and verified; 3 dry run: changes are needed; 4 refused: a
# role would lose access and no --grant covers it; 5 applied but a database is still open; 2 usage;
# anything else: a failure.
set -euo pipefail

apply=0
grants=""
suffix="_app"

usage() { sed -n '2,/^set -euo/p' "$0" | sed '$d' | sed 's/^# \{0,1\}//'; }

while [ $# -gt 0 ]; do
  case "$1" in
    --apply) apply=1 ;;
    --grant)
      [ $# -ge 2 ] || { echo "--grant needs DB=ROLE[,ROLE...]" >&2; exit 2; }
      case "$2" in *=*) ;; *) echo "--grant expects DB=ROLE[,ROLE...], got: $2" >&2; exit 2 ;; esac
      grants="${grants:+$grants;}$2"; shift ;;
    --tenant-role-suffix)
      [ $# -ge 2 ] && [ -n "$2" ] || { echo "--tenant-role-suffix needs a value" >&2; exit 2; }
      suffix="$2"; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "unknown argument: $1" >&2; usage >&2; exit 2 ;;
  esac
  shift
done

command -v psql >/dev/null 2>&1 || { echo "psql not found on PATH" >&2; exit 1; }
export PGDATABASE="${PGDATABASE:-postgres}"

run_sql() {
  psql -X -q -At -v ON_ERROR_STOP=1 -v apply="$apply" -v grants="$grants" -v suffix="$suffix" "$@"
}

# One transaction holds the whole plan, so a dry run is exact (it rolls back) and an apply is atomic.
out="$(run_sql <<'SQL'
BEGIN;
SELECT set_config('hardening.grants', :'grants', true),
       set_config('hardening.suffix', :'suffix', true) \gset

DO $$
BEGIN
  IF NOT (SELECT rolsuper FROM pg_roles WHERE rolname = current_user) THEN
    RAISE EXCEPTION 'run this as a superuser (current_user is %)', current_user;
  END IF;
END $$;

-- Databases PUBLIC can connect to: no ACL at all (the default) or an explicit PUBLIC CONNECT.
CREATE TEMP TABLE h_open AS
  SELECT d.datname FROM pg_database d
  WHERE d.datallowconn
    AND (d.datacl IS NULL
         OR EXISTS (SELECT 1 FROM aclexplode(d.datacl) a WHERE a.grantee = 0 AND a.privilege_type = 'CONNECT'));

-- Ordinary login roles that are not tenant roles: the ones whose access this could take away.
CREATE TEMP TABLE h_roles AS
  SELECT oid AS rid, rolname FROM pg_roles
  WHERE rolcanlogin AND NOT rolsuper AND rolname NOT LIKE 'pg\_%'
    AND right(rolname, length(current_setting('hardening.suffix'))) <> current_setting('hardening.suffix');

CREATE TEMP TABLE h_before AS
  SELECT r.rolname, d.datname, has_database_privilege(r.rid, d.oid, 'CONNECT') AS ok
  FROM h_roles r CROSS JOIN pg_database d WHERE d.datallowconn;

CREATE TEMP TABLE h_grants AS
  SELECT split_part(e, '=', 1) AS datname, btrim(r) AS rolname
  FROM regexp_split_to_table(current_setting('hardening.grants'), ';') AS e,
       LATERAL regexp_split_to_table(split_part(e, '=', 2), ',') AS r
  WHERE e <> '' AND btrim(r) <> '';

DO $$
DECLARE g record;
BEGIN
  FOR g IN SELECT * FROM h_grants LOOP
    IF NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = g.datname) THEN
      RAISE EXCEPTION '--grant names a database that does not exist: %', g.datname;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = g.rolname) THEN
      RAISE EXCEPTION '--grant names a role that does not exist: %', g.rolname;
    END IF;
  END LOOP;
END $$;

CREATE TEMP TABLE h_plan (stmt text);

DO $$
DECLARE g record; o record;
BEGIN
  FOR g IN SELECT * FROM h_grants ORDER BY datname, rolname LOOP
    -- Skip a grant that is already an explicit ACL entry, so a second run changes nothing.
    IF NOT EXISTS (SELECT 1 FROM pg_database d
                   CROSS JOIN LATERAL aclexplode(d.datacl) a
                   JOIN pg_roles r ON r.oid = a.grantee
                   WHERE d.datname = g.datname AND r.rolname = g.rolname AND a.privilege_type = 'CONNECT') THEN
      EXECUTE format('GRANT CONNECT ON DATABASE %I TO %I', g.datname, g.rolname);
      INSERT INTO h_plan VALUES (format('GRANT CONNECT ON DATABASE %I TO %I;', g.datname, g.rolname));
    END IF;
  END LOOP;
  FOR o IN SELECT datname FROM h_open ORDER BY datname LOOP
    EXECUTE format('REVOKE CONNECT ON DATABASE %I FROM PUBLIC', o.datname);
    INSERT INTO h_plan VALUES (format('REVOKE CONNECT ON DATABASE %I FROM PUBLIC;', o.datname));
  END LOOP;
END $$;

-- What the change would take away from roles that are not tenant roles.
CREATE TEMP TABLE h_lost AS
  SELECT b.rolname, b.datname FROM h_before b
  JOIN pg_database d ON d.datname = b.datname
  JOIN pg_roles r ON r.rolname = b.rolname
  WHERE b.ok AND NOT has_database_privilege(r.oid, d.oid, 'CONNECT');

SELECT 'PLAN ' || stmt FROM h_plan ORDER BY stmt;
SELECT 'LOSES-ACCESS role ' || rolname || ' on database ' || datname FROM h_lost ORDER BY rolname, datname;
SELECT count(*) > 0 AS has_lost FROM h_lost \gset
SELECT count(*) AS changes FROM h_plan \gset
\echo CHANGES=:changes
\echo LOST=:has_lost

\if :apply
  \if :has_lost
    ROLLBACK;
    \echo RESULT=refused
  \else
    COMMIT;
    \echo RESULT=applied
  \endif
\else
  ROLLBACK;
  \echo RESULT=dryrun
\endif
SQL
)" || { echo "$out" >&2; echo "harden-tenant-cluster: the plan failed; nothing was changed" >&2; exit 1; }

changes="$(printf '%s\n' "$out" | sed -n 's/^CHANGES=//p')"
result="$(printf '%s\n' "$out" | sed -n 's/^RESULT=//p')"
printf '%s\n' "$out" | grep -E '^(PLAN|LOSES-ACCESS) ' || true

if [ "$changes" = "0" ] && ! printf '%s\n' "$out" | grep -q '^LOSES-ACCESS'; then
  echo "hardened: no database on this cluster lets PUBLIC connect; nothing to do"
  exit 0
fi

case "$result" in
  dryrun)
    echo "dry run: $changes change(s) above; nothing was changed. Re-run with --apply to make them."
    if printf '%s\n' "$out" | grep -q '^LOSES-ACCESS'; then
      echo "WARNING: the roles listed as LOSES-ACCESS would be locked out. Add --grant DB=ROLE for each before --apply." >&2
    fi
    exit 3 ;;
  refused)
    echo "REFUSED: the roles listed as LOSES-ACCESS would be locked out of those databases. Nothing was changed." >&2
    echo "Add --grant DB=ROLE for each, or fix the role's access first, then re-run." >&2
    exit 4 ;;
  applied)
    echo "applied $changes change(s)" ;;
  *)
    echo "unexpected result: $result" >&2; exit 1 ;;
esac

# Verify, in a fresh session: nothing may still be open (a database created while this ran, say).
still="$(run_sql -c "SELECT string_agg(datname, ', ' ORDER BY datname) FROM pg_database d WHERE d.datallowconn AND (d.datacl IS NULL OR EXISTS (SELECT 1 FROM aclexplode(d.datacl) a WHERE a.grantee = 0 AND a.privilege_type = 'CONNECT'))")"
if [ -n "$still" ]; then
  echo "verify failed: still open to PUBLIC: $still. Run again." >&2
  exit 5
fi
echo "verified: no database on this cluster lets PUBLIC connect"
