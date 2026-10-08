#!/usr/bin/env python3
"""Backfill per-tenant CDC history into tenant StarRocks databases.

## Why this exists

Enabling per-tenant routing splits the destination: before, every row landed in the
shared `oms` database, and after, each row lands in its tenant's own database. Rows
written before the cutover stay in `oms` forever, because nothing re-reads them --
the loader starts from its committed offset and the past is behind it.

So immediately after cutover every tenant database is missing all of its history,
and any analytics over it is silently incomplete. The audit reports the gap; this
closes it.

## Why batch export rather than a Debezium snapshot

Both were on the table. A Debezium ad-hoc snapshot is the "correct" machinery, but
it reuses the CDC path and interacts with offset state that is already partly
consumed -- fiddly for a one-time job, and it would make a data-migration problem
depend on connector internals.

This reads each table filtered by `tenant_id` and stream-loads it, using the same
endpoint, auth, strict mode and label discipline the loader already uses. The
hardened path is the one under test, and correctness is checkable immediately
afterwards with scripts/cdc_value_audit.sh.

## Idempotency, two independent layers

1. These are PRIMARY KEY tables in StarRocks, so re-loading the same rows replaces
   them rather than duplicating them. A re-run is safe by table design.
2. Each batch carries a `label` derived from tenant, table and batch position, and
   StarRocks rejects a repeated label as `Label Already Exists`. That turns "did I
   already do this batch" into a question with a definite answer, and it is the same
   mechanism the loader relies on.

Neither layer replaces the other. Layer 1 means a re-run converges; layer 2 means a
partially-completed run can be resumed and completed without double-writing.

## Row encoding

Rows are built in Postgres with explicit per-column casts, generated from
`information_schema.columns` rather than hardcoded. That matters: the loader writes
decimals as strings (`decimal.handling.mode: string`), datetimes as UTC
`YYYY-MM-DD HH:MM:SS`, dates as `YYYY-MM-DD` and jsonb as text. Emitting native JSON
numbers or timestamps instead would either be rejected under `strict_mode` or, worse,
be silently coerced into something that audits differently from CDC-loaded data --
so a backfilled row and a CDC-loaded row for the same source row would not compare
equal.

Usage:
    SR_TENANT_DIR=/etc/uisce/sr-tenants \\
    STARROCKS_HTTP=http://starrocks-fe:8030 \\
        scripts/backfill_tenant_cdc_history.sh --tenant <uuid>       # one tenant
        scripts/backfill_tenant_cdc_history.sh --all                 # every provisioned tenant
        scripts/backfill_tenant_cdc_history.sh --all --dry-run
"""

import argparse
import base64
import http.client
import json
import os
import subprocess
import sys
from urllib.parse import urlparse

# Source table -> (starrocks table, primary key column). The connector captures
# exactly these six tables; debezium_heartbeat is Debezium's own and has no
# counterpart in StarRocks.
TABLES = [
    ("orm.order", "orm_order", "id"),
    ("orm.execution", "orm_execution", "id"),
    ("orm.placement", "orm_placement", "id"),
    ("orm.order_allocation", "orm_order_allocation", "id"),
    ("orm.execution_allocation", "orm_execution_allocation", "id"),
]

BATCH_ROWS = 2000

# The FE answers a stream load with a 307 to the BE that owns the database. One hop is
# normal; the limit exists so a misconfigured FE cannot spin a backfill forever.
MAX_REDIRECTS = 3


def log(msg):
    sys.stderr.write("%s\n" % msg)
    sys.stderr.flush()


def psql(sql, db="alpha"):
    env = dict(os.environ)
    if db:
        env.setdefault("PGPASSWORD", "postgres")
    proc = subprocess.run(
        ["psql", "-h", "127.0.0.1", "-U", "postgres", "-d", db,
         "-tAF", "\x1f", "-q", "-v", "ON_ERROR_STOP=1", "-c", sql],
        capture_output=True, text=True, env=env,
    )
    if proc.returncode != 0:
        raise RuntimeError("psql failed: %s" % proc.stderr.strip())
    return proc.stdout


def tenant_database_name(tenant_id):
    """Mirror of the loader's normalisation: hyphens dropped, lowercase."""
    return "tenant_" + tenant_id.replace("-", "").lower()


def load_routes(directory):
    routes = {}
    if not os.path.isdir(directory):
        raise RuntimeError("credential directory %s not found" % directory)
    for name in os.listdir(directory):
        if not name.endswith(".json"):
            continue
        with open(os.path.join(directory, name)) as fh:
            route = json.load(fh)
        key = route.get("tenant_id", "").replace("-", "").lower()
        if key and route.get("database"):
            routes[key] = route
    if not routes:
        raise RuntimeError("no usable tenant routes in %s" % directory)
    return routes


# ---- row construction -------------------------------------------------------
#
# One JSON expression per column, chosen by Postgres data type. Built once per
# table from information_schema so a schema change shows up as a new branch rather
# than as silently missing columns.

def json_expr_for(col, data_type):
    name = '"%s"' % col.replace('"', '""')
    t = data_type.lower()

    if t in ("uuid", "character varying", "text", "character", "citext"):
        return "to_jsonb(%s::text)" % name
    if t in ("numeric", "decimal", "money", "real", "double precision"):
        # Strings, matching decimal.handling.mode: string. A JSON number would be
        # re-parsed by StarRocks with its own precision rules and stop matching what
        # CDC-loaded rows contain for the same source value.
        return "to_jsonb(%s::text)" % name
    if t in ("timestamp with time zone",):
        # UTC, second precision -- the same normalisation the loader applies. Only
        # timestamptz is shifted; a bare timestamp carries no zone and shifting it
        # would move values that were never zoned in the first place.
        return "to_jsonb(to_char(%s AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS'))" % name
    if t == "timestamp without time zone":
        return "to_jsonb(to_char(%s, 'YYYY-MM-DD HH24:MI:SS'))" % name
    if t == "date":
        return "to_jsonb(%s::text)" % name
    if t in ("json", "jsonb"):
        # Text, not embedded JSON: the destination column is STRING.
        return "to_jsonb(%s::text)" % name
    if t in ("boolean",):
        return "to_jsonb(%s::text)" % name
    if t in ("bigint", "integer", "smallint"):
        return "to_jsonb(%s::text)" % name
    # Unknown type: fall back to text rather than guessing a representation.
    log("    note: column %s has unmapped type %s; exporting as text" % (col, data_type))
    return "to_jsonb(%s::text)" % name


def row_select(pg_table, tenant_id, after_key, limit):
    """Build the SELECT that returns one page of rows as JSON objects, one per line."""
    schema, table = pg_table.split(".", 1)
    cols = psql(
        "select column_name, data_type from information_schema.columns "
        "where table_schema='%s' and table_name='%s' order by ordinal_position;"
        % (schema, table)
    ).strip().split("\n")
    if not cols:
        raise RuntimeError("no columns found for %s" % pg_table)

    pairs = []
    for line in cols:
        if not line.strip():
            continue
        col, dtype = line.split("\x1f", 1)
        col = col.strip()
        if col == "tenant_id":
            continue  # implied by the filter; re-added below from the filter value
        # json_build_object takes positional key/value pairs, and the key is a
        # *string literal* -- single quotes. Double quotes would make Postgres read it
        # as a column reference, so every row would be keyed by that column's own
        # value, and a NULL value would become a null object key and fail the load.
        pairs.append("'%s', %s" % (col.replace("'", "''"), json_expr_for(col, dtype)))

    # tenant_id is emitted from the filter value, guaranteeing the backfilled row
    # carries the tenant it was fetched for even if the column were somehow absent.
    pairs.append("'tenant_id', to_jsonb('%s'::text)" % tenant_id)

    pk = '"id"'
    where = "tenant_id = '%s'::uuid" % tenant_id
    if after_key:
        where += " and %s > '%s'::uuid" % (pk, after_key)

    return (
        "select coalesce(json_build_object(%s)::text, '') from %s "
        "where %s and %s is not null order by %s limit %d;"
        % (", ".join(pairs), pg_table, where, pk, pk, limit)
    )


def fetch_page(pg_table, tenant_id, after_key, limit):
    out = psql(row_select(pg_table, tenant_id, after_key, limit))
    return [json.loads(line) for line in out.splitlines() if line.strip()]


def count_rows(pg_table, tenant_id):
    schema, table = pg_table.split(".", 1)
    quoted = '"%s"' % table.replace('"', '""')
    out = psql("select count(*) from %s.%s where tenant_id = '%s'::uuid;" % (schema, quoted, tenant_id))
    return int(out.strip() or 0)


# ---- stream load ------------------------------------------------------------

def sanitize_label(s):
    """StarRocks validates labels as ^[-A-Za-z0-9_]{1,128}$ -- no dots."""
    return "".join(c if (c.isalnum() or c in "_-") else "_" for c in s)[:128]


def _put_once(url, headers, body):
    """One PUT, returning (status, headers, payload) with no redirect handling."""
    u = urlparse(url)
    if u.scheme == "https":
        conn = http.client.HTTPSConnection(u.hostname, u.port or 443, timeout=300)
    else:
        conn = http.client.HTTPConnection(u.hostname, u.port or 80, timeout=300)
    try:
        conn.putrequest("PUT", u.path or "/", skip_accept_encoding=True)
        for k, v in headers:
            conn.putheader(k, v)
        conn.endheaders()
        conn.send(body)
        resp = conn.getresponse()
        return resp.status, resp.getheaders(), resp.read().decode()
    finally:
        conn.close()


def stream_load(http_base, database, table, rows, label, user, password, dry_run=False):
    body = ("[" + ",".join(json.dumps(r, separators=(",", ":")) for r in rows) + "]").encode()
    url = "%s/api/%s/%s/_stream_load" % (http_base.rstrip("/"), database, table)

    if dry_run:
        log("    dry-run: would PUT %d rows to %s with label %s" % (len(rows), url, label))
        return "Dry run"

    headers = [
        ("Authorization", "Basic " + base64.b64encode(
            ("%s:%s" % (user, password)).encode()).decode()),
        ("Expect", "100-continue"),
        ("Content-Type", "application/json"),
        ("format", "json"),
        ("strip_outer_array", "true"),
        ("label", label),
        # Same setting the loader uses. Without it StarRocks filters unconvertible
        # rows and still reports Success, which is precisely the silent-loss mode.
        ("strict_mode", "true"),
        ("partial_update", "false"),
        ("Content-Length", str(len(body))),
    ]

    # The FE answers a stream load with 307 pointing at the BE that owns the
    # database -- verified live: location http://172.20.0.6:8040/... for an FE on
    # 127.0.0.1:8030. Go's loader client follows that automatically, which is why the
    # container never notices. Python does not, so it is followed here explicitly and
    # bounded -- an unbounded loop against a misconfigured FE would hang a backfill.
    #
    # Credentials are replayed to the redirect target. That is required for StarRocks
    # (the BE re-checks the grant) and is confined to the address the FE itself
    # advertised; the hop limit keeps that from becoming an open redirect.
    status, hdrs, payload = _put_once(url, headers, body)
    hops = 0
    while status in (301, 302, 307, 308) and hops < MAX_REDIRECTS:
        location = dict((k.lower(), v) for k, v in hdrs).get("location")
        if not location:
            break
        hops += 1
        url = location
        status, hdrs, payload = _put_once(url, headers, body)

    if status in (301, 302, 307, 308):
        raise RuntimeError(
            "stream load to %s.%s exceeded %d redirects (last: %s)"
            % (database, table, MAX_REDIRECTS, url)
        )

    # Judge the body, not the HTTP code. A repeated label answers 200 with
    # "Label Already Exists", which is a success for an idempotent re-run.
    try:
        parsed = json.loads(payload)
    except ValueError:
        raise RuntimeError(
            "stream load to %s.%s returned HTTP %s with a non-JSON body: %r"
            % (database, table, status, payload[:300])
        )

    st = str(parsed.get("Status", ""))
    if status == 200 and st in ("Success", "Label Already Exists"):
        rejected = int(parsed.get("NumberFilteredRows", 0) or 0)
        if rejected:
            raise RuntimeError(
                "stream load accepted but filtered %d row(s) to %s.%s: %s"
                % (rejected, database, table, parsed.get("ErrorURL", ""))
            )
        return st

    raise RuntimeError(
        "stream load to %s.%s failed: HTTP %s Status=%s msg=%s"
        % (database, table, status, st, str(parsed.get("Message", ""))[:300])
    )


# ---- driver -----------------------------------------------------------------

def backfill_table(routes, tenant_id, route, http_base, pg_table, sr_table, dry_run):
    database = route["database"]
    total = count_rows(pg_table, tenant_id)
    if total == 0:
        log("  %-26s 0 rows" % sr_table)
        return 0, 0

    user, password = route.get("user", ""), route.get("password", "")
    after_key, batches, loaded, already = None, 0, 0, 0

    while True:
        rows = fetch_page(pg_table, tenant_id, after_key, BATCH_ROWS)
        if not rows:
            break
        # The label is derived from content position, not wall-clock time, so a
        # re-run produces byte-identical labels and StarRocks dedupes them.
        first = rows[0].get("id", "")
        last = rows[-1].get("id", "")
        label = sanitize_label("backfill_%s_%s_%s_%s" % (tenant_id, sr_table, first, last))

        status = stream_load(http_base, database, sr_table, rows, label,
                             user, password, dry_run=dry_run)
        if status == "Label Already Exists":
            already += len(rows)
        else:
            loaded += len(rows)
        batches += 1
        after_key = last

        if len(rows) < BATCH_ROWS:
            break

    log("  %-26s %d rows -> %d batches (loaded=%d already_present=%d)"
        % (sr_table, total, batches, loaded, already))
    return total, batches


def mysql_query(host, port, user, password, sql):
    """Run SQL over the StarRocks MySQL protocol, returning stdout lines."""
    cmd = ["mysql", "-h", host, "-P", str(port), "-u", user, "-N", "-B", "-e", sql]
    if password:
        cmd.append("-p" + password)
    proc = subprocess.run(cmd, capture_output=True, text=True)
    if proc.returncode != 0:
        raise RuntimeError("mysql failed: %s" % proc.stderr.strip()[:300])
    return proc.stdout


def reconcile_table(route, tenant_id, pg_table, sr_table, host, port, dry_run=False):
    """Remove rows from a tenant database that no longer exist in Postgres.

    Postgres is the source of truth and this only ever moves StarRocks toward it --
    never the reverse. A row is deleted only when its key is absent from Postgres
    entirely, so no amount of ambiguity can cause a live row to be removed.

    This exists because event replay cannot fix an unattributable delete. When a
    delete is dead-lettered, the tenant is unknowable from the event and may also be
    unknowable from Postgres (the row is gone there too), so there is nothing to
    re-drive. Reconciling by key needs no attribution at all.

    Returns (found, deleted).
    """
    database = route["database"]
    user, password = route.get("user", ""), route.get("password", "")

    sr_rows = mysql_query(
        host, port, user, password,
        "SELECT id FROM `%s`.`%s`" % (database, sr_table))
    if not sr_rows.strip():
        return 0, 0
    # splitlines, not the string itself: iterating a str yields characters, which then
    # reach Postgres as one-character uuid literals and fail the cast.
    keys = [ln.strip() for ln in sr_rows.splitlines() if ln.strip()]

    phantoms = []
    for key in keys:
        present = psql(
            "select 1 from %s where id = '%s'::uuid;" % (pg_table, key.replace("'", "''"))
        ).strip()
        if not present:
            phantoms.append(key)

    if not phantoms:
        return len(keys), 0

    if dry_run:
        log("    dry-run: would delete %d phantom row(s) from %s.%s"
            % (len(phantoms), database, sr_table))
        return len(keys), 0

    # Deletes go over the MySQL protocol on the query port: StarRocks 3.3 removed the
    # HTTP SQL endpoint (/api/query returns 501), and this is the same path the loader
    # uses for keyed deletes.
    statements = " ".join(
        "DELETE FROM `%s`.`%s` WHERE `id` = '%s';" % (database, sr_table, k.replace("'", "''"))
        for k in phantoms
    )
    out = mysql_query(host, port, user, password, statements)
    if out.strip():
        raise RuntimeError("delete returned output: %s" % out.strip()[:200])
    # len(keys), not len(sr_rows): the latter is the character length of the raw
    # stdout, which is ~37x the row count and reads like a plausible but meaningless
    # number in the log. Every path out of this function returns a row count.
    return len(keys), len(phantoms)


def main():
    ap = argparse.ArgumentParser(description=__doc__,
                                 formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--tenant", action="append", default=[],
                    help="tenant uuid; repeatable. Omit with --all for every provisioned tenant.")
    ap.add_argument("--all", action="store_true", help="backfill every tenant with a route file.")
    ap.add_argument("--tenant-dir", default=os.environ.get("SR_TENANT_DIR", "/etc/uisce/sr-tenants"))
    ap.add_argument("--http", default=os.environ.get("STARROCKS_HTTP", "http://starrocks-fe:8030"))
    ap.add_argument("--dry-run", action="store_true",
                    help="report what would be sent without writing anything.")
    ap.add_argument("--reconcile", action="store_true",
                    help="remove rows from the tenant databases that no longer exist in "
                         "Postgres, instead of backfilling. Postgres is the source of "
                         "truth; this only ever moves StarRocks toward it.")
    ap.add_argument("--mysql-port", type=int, default=int(os.environ.get("STARROCKS_QUERY_PORT", "9030")),
                    help="StarRocks MySQL-protocol port, used for the keyed deletes.")
    ap.add_argument("--only-active", action="store_true",
                    help="skip tenants with no active route file.")
    args = ap.parse_args()

    if not args.tenant and not args.all:
        ap.error("pass --tenant <uuid> (repeatable) or --all")

    routes = load_routes(args.tenant_dir)

    wanted = []
    if args.all:
        wanted = list(routes.keys())
    else:
        for t in args.tenant:
            wanted.append(t.replace("-", "").lower())

    missing = [t for t in wanted if t not in routes]
    if missing:
        log("no route for tenant(s): %s" % ", ".join(missing))
        log("run scripts/provision_starrocks_tenants.sh first; refusing to guess a database")
        return 1

    grand_rows = 0
    grand_deleted = 0
    failures = []
    mysql_host = urlparse(args.http if "//" in args.http else "http://" + args.http).hostname

    for key in wanted:
        route = routes[key]
        tenant_id = route["tenant_id"]
        log("tenant %s (%s) -> %s"
            % (tenant_id, route.get("tenant_name", "?"), route["database"]))
        for pg_table, sr_table, _pk in TABLES:
            try:
                if args.reconcile:
                    found, deleted = reconcile_table(
                        route, tenant_id, pg_table, sr_table,
                        mysql_host, args.mysql_port, dry_run=args.dry_run)
                    grand_rows += found
                    grand_deleted += deleted
                    log("  %-26s %d rows, %d phantom(s) removed" % (sr_table, found, deleted))
                else:
                    rows, _ = backfill_table(routes, tenant_id, route, args.http,
                                             pg_table, sr_table, args.dry_run)
                    grand_rows += rows
            except Exception as e:  # noqa: BLE001 - report and continue to next table
                # One bad table must not abandon the rest: a partial backfill plus an
                # explicit failure list is more useful than an abort halfway, and the
                # audit will show exactly which tables are short.
                log("  %-26s FAILED: %s" % (sr_table, e))
                failures.append("%s/%s: %s" % (tenant_id, sr_table, e))

    log("")
    if args.reconcile:
        log("rows examined: %d, phantoms removed: %d" % (grand_rows, grand_deleted))
    else:
        log("total rows seen: %d" % grand_rows)
    if failures:
        log("FAILURES (%d):" % len(failures))
        for f in failures:
            log("  %s" % f)
        return 1
    verb = "reconciled" if args.reconcile else "backfilled"
    log("RESULT: %s %d tenant(s)" % (verb, len(wanted)))
    return 0


if __name__ == "__main__":
    sys.exit(main())