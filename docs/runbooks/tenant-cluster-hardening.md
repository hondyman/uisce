# Runbook: harden a Postgres cluster for per-tenant databases

**Owner:** platform / DBA on call. **Applies to:** every Postgres cluster that holds, or will hold,
tenant databases (ADR-030). **Script:** [`scripts/harden-tenant-cluster.sh`](../../scripts/harden-tenant-cluster.sh).

## Why this exists

Provisioning a tenant with an `app` gives its database its own role. Before the tenant's binding goes
active, the saga **proves that role can connect to exactly one database, its own** by trying every
other connectable database on the cluster as that role. If any accepts, provisioning fails with
`tenant database role can connect to other databases` and names each one.

Postgres grants `CONNECT` on every database to `PUBLIC` by default. A role that can connect to
`postgres` can list every database and role on the cluster, which is every tenant's code. So until
the default is revoked on the other databases (`postgres`, `template1`, **the control plane `alpha`**,
legacy `tenant_*` databases), `app` provisioning on that cluster is refused. That is deliberate:
the saga checks, it never changes cluster privileges itself.

## When to run it

- **Before enabling `app` provisioning in any environment** (dev, staging, prod).
- **Before any ORM move (Phase 4b) wave** onto a cluster: see "Sequencing gate" below.
- **After any out-of-band `CREATE DATABASE`** on a hardened cluster (see "Corrected fact" below).
- Routinely, as a drift check: a dry run exits `3` when something is open.

## Procedure

All commands use the usual libpq environment and must run as a **superuser**.

```bash
export PGHOST=<host> PGPORT=5432 PGUSER=<superuser> PGPASSWORD=<...>
```

**1. Dry run (changes nothing; the plan is exact).**

```bash
scripts/harden-tenant-cluster.sh
```

It prints every `REVOKE` it would run and, importantly, every ordinary login role that **would lose
access** (`LOSES-ACCESS role <r> on database <d>`). Tenant roles (names ending `_app`) are expected
to lose access to other databases and are not listed. Exit `0` = already hardened; exit `3` = work
to do.

**2. Decide the grants.** Every role listed under `LOSES-ACCESS` that legitimately needs a database
gets a `--grant`. Typical:

| Database | Who needs `CONNECT` |
|---|---|
| `alpha` (control plane) | the application's login role, `uisce_gold_copy_sync`, `uisce_mcp_app`, and any reporting/monitoring role that reads it |
| `postgres` | monitoring/exporter roles that use it as their maintenance database |
| each `tenant_*` legacy database | the role that application uses, if it is not already a tenant role |

A role you cannot account for is a question for its owner, not a reason to add a grant.

**3. Apply.**

```bash
scripts/harden-tenant-cluster.sh --apply \
  --grant alpha=<app_role>,uisce_gold_copy_sync,uisce_mcp_app \
  --grant postgres=<monitoring_role>
```

The script **refuses (exit 4) and changes nothing** while any ordinary role would be locked out and no
`--grant` covers it. The whole change is one transaction. It verifies in a fresh session afterwards
(exit `5` if a database is still open, for example one created while it ran: just run it again).

**4. Confirm.** Re-run with no arguments: `hardened: ... nothing to do`, exit `0`. Then retry the failed
provisioning; the isolation check should now pass.

The script is idempotent: running it again on a hardened cluster does nothing.

### The 2 a.m. version

1. Provisioning failed with `can connect to other databases: <list>`.
2. `scripts/harden-tenant-cluster.sh` (dry run). Read the `LOSES-ACCESS` lines.
3. Re-run with `--apply` and a `--grant` for each role that must keep a database.
4. Retry provisioning.

## Corrected fact: `template1` does not protect new databases

It is tempting to `REVOKE CONNECT ON DATABASE template1 FROM PUBLIC` and expect databases created from
it to inherit that. **They do not** (verified on PostgreSQL 16): a new database starts with an empty ACL,
which means `PUBLIC` may connect, whatever its template says, and there is no cluster-level
`ALTER DEFAULT PRIVILEGES` for databases. So:

- The saga's `CreateTenantDatabase` revokes `PUBLIC`'s `CONNECT` itself, on the created **and** the
  already-existed path. That is the real protection for databases the saga creates.
- **Any other path that creates databases (legacy scripts, manual `CREATE DATABASE`, other tools) must
  do the same**, or the database is open until this script runs. The isolation check is the safety net:
  it fails provisioning for the next tenant and names the database.
- Revoking on `template1` is still worth doing (the script covers it) so a role cannot connect *to
  `template1` itself*; it just does not reach later databases.

## Sequencing gate for the ORM move (Phase 4b)

The ORM move puts tenants onto new databases, possibly on clusters that still have legacy open
databases. Provisioning with an `app` would then **fail in the middle of a cutover wave**. So hardening
is a **precondition**, not something discovered during a wave:

- [ ] For every cluster that will receive a wave: `scripts/harden-tenant-cluster.sh` exits `0`.
- [ ] That check is re-run (and still `0`) on the day of the wave, since databases may have been created
      since.
- [ ] The wave's runbook records which `--grant`s were applied to the control-plane database.
- [ ] Expect provisioning to cost about **1.5 ms per existing database** for the isolation check
      (measured 0.35 s at 200 tenants; ~3 s at 2,000 is an extrapolation): size wave timeouts accordingly.
- [ ] Expect **one server connection per tenant pool at minimum**; confirm `max_connections` (or
      PgBouncer) covers the wave before it starts.

## If something goes wrong

| Symptom | Meaning | Action |
|---|---|---|
| Provisioning: `can connect to other databases: a, b` | those databases still allow `PUBLIC` | run the script (dry run, then `--apply`) |
| Provisioning: `cannot verify isolation ...` (retryable) | the check could not tell, e.g. a database refusing connections, or the credential unreadable | fix the cause named in the message and retry; it is not a proven hole |
| Script exit `4` (refused) | an ordinary role would lose access | add `--grant DB=ROLE` for it, or resolve who owns that role |
| Script exit `5` | a database appeared while it ran | run it again |
| A role was locked out after an apply | it had access only through `PUBLIC` and was granted nothing | `GRANT CONNECT ON DATABASE <db> TO <role>;` |
| Must undo a revoke on a database | (not recommended: it reopens the hole) | `GRANT CONNECT ON DATABASE <db> TO PUBLIC;` |

## What this does not cover

- **Password rejection and network access.** This is privilege hardening; `pg_hba.conf` and network
  policy are separate.
- **Managed Postgres.** The script needs a role that can `REVOKE`/`GRANT` on every database; on a
  managed service that may be a restricted "superuser". It has not been run on one: dry-run first and
  expect to adapt.
- **Roles that reach other databases through other paths** (`pg_read_all_data`, membership in a role that
  has `CONNECT`): the script's lock-out report accounts for inherited access, but review any role with
  broad memberships by hand.
