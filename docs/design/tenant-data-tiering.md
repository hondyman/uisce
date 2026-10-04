# Tenant data tiering: design for review

**Status: proposed. Nothing here is built. Revised after owner direction on `tenant_id`, Debezium Server and Spark, and after checking the deployed environment; the decisions still open are listed at the end.** It follows ADR-035
(hot is the tenant's Postgres, warm is StarRocks, cold is Iceberg; `alpha` is never tiered) and is
written after reading what is actually in the tree, because the tree differs from the ADR in one way
that matters.

## What exists (checked in the tree)

**The tenant ORM schema** (`backend/db/tenant_migrations/orm/0001_orm_schema.up.sql`, 34 tables).
Classified by whether they grow without bound and are written once:

| Table | Shape | Partitioned today | Notes |
|---|---|---|---|
| `quote` | time series, highest volume | yes, `RANGE (quote_time)`, **default partition only** | PK `(id, quote_time)` already includes the key; no partition manager carried across (ADR-042) |
| `pnl_intraday` | time series | no | PK `(id)`; key `as_of_timestamp` |
| `market_data_snapshot`, `position_history` | daily snapshots | no | key `as_of_date` |
| `order_event`, `order_history`, `order_reject`, `execution`, `execution_quality`, `pre_trade_check`, `order_amendment` | order and trade records | no | **books and records**; PK `(id)` and foreign keys to `order`; time columns differ per table |
| the rest | state (accounts, limits, rules, sessions) | n/a | never tiered |

**CDC today.**
- `docker-compose.debezium.yml`: Debezium **Server** on Redpanda, one source connector, pointed at
  `alpha` and capturing five `iam.*` tables (`debezium/application.properties`). Debezium Server runs one
  connector per instance.
- `debezium/orm-oms-connector.json`: a Debezium Postgres connector on Kafka Connect capturing the
  **shared `crims` database**, schema `orm`, one slot (`orm_oms_slot`), topics `orm_oms.orm.<table>`.
- `cmd/stream_loader`, deployed one instance per topic in `docker-compose.remote.yml`
  (`CDC_TOPIC: orm_oms.orm.execution`, ...): consumes those topics, asserts an `AssignedTenantID`,
  validates, and stream-loads StarRocks `oms.*`.
- `internal/audit/iceberg_sink.go` is a Kafka-to-Parquet writer for audit events, not tenant data.

## What is actually deployed (read from the Docker host, 2026-10-04, read-only)

- **No Debezium Server is running.** The only CDC container is `uisce-debezium`, image `debezium/connect:2.3`,
  which is **Kafka Connect**. `docker-compose.debezium.yml` (Debezium Server) exists in the repo and is not
  deployed.
- **The one deployed connector, `orm-oms-connector`, is `RUNNING` with its task `FAILED`**: `Couldn't obtain
  encoding for database alpha` at start. So the OMS CDC feed is **already down**, independent of #376. I did not
  investigate the cause (it is a connection or permission failure reading the database's encoding; the
  connector's certificate paths are under `/tmp` inside the container, which does not survive a recreate) and
  did not touch the environment.
- **It points at `alpha`, not `crims`** (`database.dbname = alpha`, `schema.include.list = orm`), unlike
  `debezium/orm-oms-connector.json` in the repo. The repo file and the deployed config have drifted, and the
  five tables it captures are `execution`, `order`, `placement`, `order_allocation`, `execution_allocation`.
- Redpanda holds exactly those five `orm_oms.orm.*` topics (1 partition, 1 replica each), and there are five
  `uisce-stream-loader-*` containers, one per topic.
- **There is no Spark anywhere on the host.** StarRocks 3.3 (FE and BE), Lakekeeper (and a gold-copy
  Lakekeeper), Redis, Temporal and Redpanda are there.
- **Postgres (read with the `postgres` client certificate, read-only queries):**
  - `max_replication_slots = 10` and **8 are in use**, all of them **inactive**; `max_wal_senders = 10`;
    `wal_level = logical`; **`max_slot_wal_keep_size = -1` (unlimited)**.
  - The inactive slots retain **29 to 36 GB** each (overlapping, so the WAL directory is **13 GB**), including
    `orm_oms_slot` at 4.1 GB since its task failed. The host's disk is **90% full, 15 GB free**. Another 15 GB of
    retained WAL stops Postgres, and with it Keycloak, Temporal and everything else on this instance. This is the
    failure mode of decision 2 below, and it is already live.
  - **No tenant has moved and Phase 4a is not deployed here:** there is no `tenant_datasource_binding` or
    `tenant_lakehouse` table in `alpha`, no `ivy_t_*` role, and no tenant database. The ORM is `alpha.orm` (plus a
    separate `crims` and an `orm` database). So the `oms.*` gap from #376 is latent, not live; the connector's
    failure is not caused by it.
  - **`pg_hba` has no rule for a tenant role** (`ivy_t_*`). Today a role outside the cert-authenticated list can
    only connect from the Docker network (`host all all 172.20.0.0 scram-sha-256`); from the dev Mac or the
    Tailscale range there is no matching rule. The tenant roles use a password from `dscreds`, so a tenant
    database hosted on this instance would refuse them from anywhere but Docker. I do not know where tenant
    databases will be hosted; if here, this needs a rule (a group role with one `hostssl ... scram-sha-256`
    line, or per-tenant client certificates mapped by `pg_ident`) before provisioning can work.
  - The live `pg_hba` also differs from the hardening handoff: `temporal` is `host ... scram-sha-256` (not cert),
    and the replication rule for `172.16.0.0` is `scram-sha-256` (the handoff says `trust`).

## The finding that shapes this

**The existing CDC pipeline reads the shared `crims` database, and #376 moved tenants out of it.** The
cutover runbook (`docs/runbooks/orm-cutover.md`) does not mention Debezium, `stream_loader` or StarRocks.
For a tenant that has been moved, writes go to its own database, so nothing reaches `orm_oms.*` and
StarRocks `oms.*` stops updating for that tenant. The pipeline is also single-tenant by construction (one
`AssignedTenantID`, topics named for the shared schema). This is not a tiering problem but it is the
same problem: **CDC has to become per tenant database**, and the tiering design below depends on that
being solved once. I have not confirmed whether any tenant has actually been moved in a deployed
environment; if one has, its `oms.*` feed is already stale.

## Proposal

### 0. `tenant_id` is an invariant in every store, so a misrouted service fails closed
If tenant A's service is pointed at tenant B's database and asks `WHERE tenant_id = 'A'`, it gets zero rows, not
B's books. Partitioning does not give that; the predicate does. Requirements:

- `tenant_id` on every table in every store: the Postgres source, the Debezium envelope, Iceberg, StarRocks
  `oms.*`; `NOT NULL`; leading column of every key and index in the stores we design from here.
- **What the tenant ORM schema has today:** every table already has `tenant_id uuid NOT NULL`, but it is not the
  leading column of any primary key (`PRIMARY KEY (id)` throughout, `(id, quote_time)` on `quote`). Reshaping
  keys means a forward migration per table through the fleet runner; `0001` is applied and is not edited. The
  cheap, additive lock is a per-tenant `CHECK (tenant_id = '<that tenant>')`, generated from the tenant id at
  provisioning, which rejects a misrouted write outright. I propose adding that first and reshaping keys only
  where a table is being partitioned anyway.
- **Postgres partitioning:** with one tenant per database, `LIST (tenant_id)` prunes nothing and competes with a
  drop job that detaches by age. Take **(a) `RANGE (time)` plus the column and the CHECK**, unless you want a
  detach-by-tenant lever, in which case **(b) `LIST (tenant_id)` parents with `RANGE (time)` sub-partitions**.
  Either way the ADR records that tenant partitioning supplements the predicate and never replaces it.
- **Iceberg:** `PARTITIONED BY (tenant_id, month(event_time))`. Here it prunes, and it gives the verifier and
  the drop job clean boundaries.
- **Every read path carries the predicate structurally, not by review discipline:** a view per store or one
  repository helper that injects it, and the verifier and the drop job go through the same path. A second lock
  is Postgres RLS keyed on a session setting, which returns nothing if the predicate is missing.
- **`:tenant` comes from the authenticated session context, never from a connection string or a config
  default** (the standing no-fallback rule).

### 1. CDC: one Debezium Server instance per tenant database
Per the owner's direction. It runs the same connector code as Kafka Connect, so capture cost per record is the
same; what differs is the operating model: no Connect cluster to run, and one instance per tenant matches
one-database-per-tenant provisioning.

- The provisioning saga generates each instance's `application.properties` from a template: `topic.prefix` per
  tenant, `pgoutput`, one slot and one publication per database, `slot.drop.on.stop=false`, a table include
  list. Credentials come from `dscreds` for the tenant's own replication-capable role, never a shared one.
- Offsets and slot state on a volume. Downstream writes are idempotent (section 2), so a lost offset store costs
  re-emitted events, not wrong data.
- Kafka sink tuned for volume: larger `linger.ms` and `batch.size`, `compression.type=zstd`.
- Topics keep the shape `orm_oms.orm.<table>`, tenant-prefixed. Both consumers derive the tenant from the topic
  name, and Redpanda ACLs are scoped per tenant prefix.
- Hot path unchanged: `stream_loader` keeps feeding `oms.*`; the cold writer is a second consumer group with its
  own offsets.
- **Recorded tradeoff:** no centralized connector management. At tens to hundreds of tenants, fleet operation is
  the cost Connect would have absorbed. **Revisit trigger: 25 tenants, or a need for shared transforms.**
- **Still blocking either way:** `stream_loader`'s hard-wired `AssignedTenantID` (it is per tenant by
  construction today) and the cutover runbook's silence on CDC. And, new from the host: the deployed connector is
  failing today.

### 2. Cold: Spark microbatch to Iceberg (a new component, see decision 5)
Debezium Server has no Iceberg sink, so a consumer between Redpanda and Iceberg is required. This design uses
**Spark Structured Streaming**, not StarRocks, for this path: CDC carries updates and deletes, StarRocks 3.3's
Iceberg write path is append-style, and a correct cold copy needs `MERGE`. (ADR-036's audit copy stays on
StarRocks: it is append-only by construction.)

- Trigger every 5 to 15 minutes; that is the cold tier's freshness, which is acceptable for cold.
- Per microbatch: collapse to the latest state per key ordered by source LSN, then `MERGE` into an Iceberg v2
  table; `op = d` becomes a row-level delete. Guard the match with `source_lsn > target_lsn` so a re-emitted or
  out-of-order event cannot resurrect an old value. Scheduled compaction.
- Bulk backfills do not go through the WAL: Spark JDBC reads the source and writes Iceberg directly, recording
  the source LSN, then Debezium starts from there. The verifier is told a window was batch-loaded.
- One writer per tenant and table (a Temporal-owned job id), per ADR-036's reason.
- Tenant credential: Spark is given the tenant's own bucket-scoped credential for its own warehouse, never a
  shared one, so the isolation of ADR-032 holds on this path.

### 3. Verification before any drop
A per-tenant, per-table, per-closed-partition verifier shaped like the audit-copy one (#377): page by page with a
cursor; **row count and an order-independent checksum** (sum or XOR of per-row hashes) computed on both sides
**with the tenant predicate**, compared at an **Iceberg snapshot pinned to a watermark offset** so delivery lag
is not read as a failure. Findings name a partition and a kind, never a value. The outcome is recorded
non-monotonically per `(tenant, table, partition)`. There is no seal chain on these tables, so the checksum is
the evidence (ADR-035: "where present").

### 4. The drop job
Reads `hot_window_days` and `legal_hold` from the binding (ADR-038). Detaches a partition, and drops it in a
later run, only if all hold: it is wholly older than the window, `legal_hold` is false, and its recorded
verification is clean and newer than the last time the partition could have changed (a closed partition is
immutable by rule; a late row into it is a finding, not something to absorb).

### 5. Order of work
1. **Repair and re-home the existing CDC.** The deployed connector is failing now. Decide, before moving more
   tenants, what a moved tenant's `oms.*` feed is, and put it in the cutover runbook.
2. **Per-tenant CDC** (section 1) in the provisioning saga, with the `tenant_id` CHECK (section 0).
3. **Time partitioning:** give `quote` real partitions; partition `pnl_intraday` and the snapshot tables. Only
   `quote` needs no key change.
4. Spark cold sink (2), then the verifier (3), then the drop job (4).
5. **The books-and-records tables last, and not without decision 3.** Until it is answered they get the
   `tenant_id` invariant and no drop policy.

## Decisions

1. **Resolved by owner direction:** Debezium Server per tenant database, with the revisit trigger above.
2. **Replication slot failure mode: still needs a yes.** One slot per tenant database makes a slot loss one
   tenant, not the fleet. I propose a lag alarm on every slot and `max_slot_wal_keep_size` set so a stalled slot
   is dropped before it fills a shared cluster's disk; a dropped slot means a re-snapshot, which the idempotent
   `MERGE` tolerates. Do you accept slot loss and re-snapshot as the failure mode? **The cluster is at
   8 of 10 slots with unlimited WAL retention and 15 GB of disk headroom, so this is not hypothetical:** a
   per-tenant slot design cannot start until the slot limit is raised (a restart) and a retention cap is set.
3. **Order and trade tables (`order_event`, `execution`, ...): needs the business owner.** Books-and-records
   retention, and whether a copy in object storage may be the record once the OLTP row is gone.
4. **Pilot table:** `quote`, unless it is itself under a best-execution retention rule, in which case it moves to
   decision 3.
5. **Spark is not deployed and is a new component.** The `MERGE` argument above is why I prefer it for CDC;
   the cost is one more engine to run, size and secure per the isolation rules. Alternatives: StarRocks with
   append-only history tables and a view that resolves the latest row, which needs no new engine but makes cold
   storage a change log rather than a copy. Which do you want?

## What is still unverified
- Why the deployed connector's task fails (its certificate files under `/tmp` and the post-hardening `pg_hba` are
  the places to look; I did not open the container).
- Where tenant databases will be hosted, which decides the `pg_hba` question above.
- Whether Spark's Iceberg `MERGE` against Lakekeeper with the tenant's credential works in this environment; no
  Spark here to try it on.
