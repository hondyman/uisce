# Runbook: cut the ORM (crims) over to per-tenant databases

**Owner:** platform. **Applies to:** the ORM app (`App = "orm"`), moving from one shared ORM database
to one database per tenant. **Decisions:** ADR-030 (tenant data is reached only through `tenantdb`),
ADR-042 (the schema a tenant database gets), ADR-043 (the move and the cutover), ADR-040 (the
authorization cache that makes the cutover a bounded wait rather than a restart).

## What this changes

Before: every tenant's orders, placements, executions and allocations lived in **one** shared ORM
database, and `internal/trading/persist.go` opened it from `CRIMS_ORM_DSN`. After: each tenant has
its **own** ORM database, the FIX activities and `trading.LoadOrder` reach it through
`tenantdb.Router.ResolveApp(ctx, "orm")`, and the `tenant_id` fence is a statement about which
database was opened rather than which rows came back.

`CRIMS_ORM_DSN` is no longer read anywhere. Its removal is a code change, not an environment
change, so there is no setting left to clear.

## Before you start

- Every tenant that will be cut over must already have an **`orm` datasource row and an active
  binding** in `alpha` for the `orm` app. The provisioning saga (`App = "orm"`) creates the
  database, its role, the binding and runs the isolation probe. A tenant with no active binding is
  refused at resolve time, which is correct but is a cutover outage for that tenant.
- The cluster must already be hardened (`scripts/harden-tenant-cluster.sh`): the saga's isolation
  probe requires `CONNECT` to be revoked on other databases, or provisioning fails.
- **Writes must be pausable** for the FIX activities on the tenants in the wave. See Step 1.

## Step 0 — the gate

Do not start on a branch whose CI signal you cannot read. `Build Frontend` on `main` has been red
roughly 81% of a recent 20-run window (`Test Files 113 passed (114)`, `Tests 662 passed (668)`,
`[vitest-worker]: Timeout calling "onTaskUpdate"` — every assertion green, six tests silently
unreported). While that is the state, a red check on your branch is not evidence about your change.

## Step 1 — pause writes

The cutover is a flip, not a deployment, but a process that has not yet converged is still writing
to the **old** database (ADR-040). Stop the FIX write path for the tenants in the wave:

- stop the `temporal-worker` instances that run `PersistFIXRouteActivity` / `PersistFIXFillActivity`
  for those tenants, or
- put the tenant into `offboarding`-equivalent quiescence in the registry so routing stops.

Reads may continue during the pause.

## Step 2 — run the move

`internal/migrations/ormmove` copies each tenant's rows and then **verifies** them.

```go
m := &ormmove.Mover{Source: sharedORM}          // the one shared database
f := &ormmove.Fleet{
    Mover:   m,
    Connect: <resolve the tenant's orm database and return (*sql.DB, release, error)>,
    WaveSize:        25,                        // tune against your pool and disk
    Concurrency:     4,
    MaxWaveFailures: 0,                         // 0 = stop the rollout on the first bad tenant
}
rep, err := f.Apply(ctx, targets)               // targets are migrations.Target{TenantID, App: "orm"}
```

Read the report, do not just check `err`:

- `rep.Moved` — tenants whose every table now matches the source.
- `rep.Failed` — tenants with a hard error (unreachable, unbound, a row that will not scan).
- `rep.Skipped` — tenants never attempted because an earlier wave exceeded `MaxWaveFailures`.
- `rep.Done` — true only when `Failed == 0 && Skipped == 0`.

**A tenant that is not `Moved` is not cut over.** Leaving it on the old path is the safe outcome;
flipping it anyway is how a tenant loses fills.

**Re-running is safe and is the repair path.** The copy is idempotent, so a tenant that lost rows
gets them back on the next run. It cannot remove a row the source does not have, so a target that
holds an extra row stays mismatched and stays reported — that is the case to investigate by hand.

The move copies the tenant's rows **and the shared reference tenant's** (`…-0001`), because the
RLS policies it replaces let a tenant read both. `orm.quote` is deliberately **not** copied; it is
shared market data and stays readable from `alpha` (ADR-042, `ormmove.ExcludedTables`).

## Step 3 — flip

For each tenant in `rep.Moved`, update the `orm` datasource's binding to the per-tenant database
and **bump the binding version**. The version bump is what makes the router build a new pool rather
than reuse the old one (`tenantdb.cacheKey.version`).

## Step 4 — let every process converge

`AuthTTL` bounds how long a process keeps resolving the old answer: **3s** in production
(`tenantdb.DefaultAuthTTL`). So:

- wait at least `AuthTTL` plus a margin, **or** call `router.InvalidateDatasource(id)` in-process,
- then verify from each process that it now resolves the per-tenant database
  (`pool.Database()` is the per-tenant name, not the shared one).

**Do not resume writes before every process has converged.** A lagging process is still writing to
the shared database, and the window is silent: nothing errors, the rows just go to the old place.

## Step 5 — verify

Per tenant, in its own database:

- row counts for the order chain match the source for the tenant's rows;
- the FIX route and fill paths succeed end to end (`PersistFIXRouteActivity`, `PersistFIXFillActivity`);
- `trading.LoadOrder` returns the order;
- the isolation probe still holds: the tenant's role is denied `CONNECT` to another tenant's
  database.

Then resume writes.

## Rollback

**The rollback is the status quo.** Until the shared database is dropped, reverting is a binding
flip back plus the `AuthTTL` wait. That is why this procedure deletes nothing.

Dropping the shared ORM database is a **separate, later, reversible** step, and it is the point of
no return for the data. Before dropping it:

- every tenant in the fleet reports `Moved`;
- the shared database has been read-only for a full retention window, not minutes;
- a restore path exists for it.

### Dropping the shared database is a gated, two-person decision

Until this section existed, the point of no return was reachable **by omission**: nobody had to
decide to destroy anything, only to not decide not to. That is now closed. Dropping the shared
ORM database requires all of the following, recorded in the change that does it:

1. **A named role.** The platform owner named in this runbook's header, in the change description.
2. **A second, separate approver.** It is irreversible; one person must not be both proposer and
   approver.
3. **The fleet report attached, showing `Done` for every tenant** — the query output, not a
   claim. `ormmove.FleetReport` carries `Moved`, `Failed` and `Done`; a non-zero `Failed` blocks
   the drop outright.
4. **The retention window with a start and end timestamp**, not "a while".
5. **A named, executed restore.** "A restore path exists" was previously a precondition that
   named no path, and so was not checkable by anyone. As of #383 it is not satisfied at all:
   `docs/runbooks/dr-playbook.md` invokes fifteen scripts, none of which exists, and applies five
   Kubernetes manifest paths, none of which exists, for a platform deployed with Docker Compose.
   **Until #383 is closed, this precondition cannot be met and the shared database must not be
   dropped.** The absence of a restore is the one condition here that is checkable today, and it
   currently fails.

The first four are procedural. The fifth is a real blocker, and it is deliberately stated as one:
a read-only window protects against a mistaken drop, and nothing protects against the drop being
the only copy.

## If a tenant is stuck

| Symptom | Meaning | Action |
|---|---|---|
| `connect: tenant … has no active orm binding` | No binding, or it is not `active` | Provision or activate the binding; the refusal is correct |
| `source has N rows, target has M` (M > N) | The target holds rows the source does not | Investigate by hand. The move will not delete them and will keep reporting the mismatch |
| `count source: … does not exist` | The shared database has no `orm` schema | Wrong source DSN. The move refuses rather than copying nothing and reporting success |
| `source has no columns` | Same, caught earlier | Same |

## Related

- `backend/db/tenant_migrations/orm/0001_orm_schema.up.sql` — the schema a tenant ORM database gets.
- `docs/ARCHITECTURAL_DECISIONS.md` — ADR-030, ADR-040, ADR-042, ADR-043.
- `docs/runbooks/tenant-cluster-hardening.md` — the prerequisite cluster hardening.
