# Tenant data tiering: design for review

**Status: proposed. Nothing here is built. Revised after owner direction on `tenant_id`, Debezium Server, a Spark-free write path and restore semantics, and after checking the deployed environment; the decisions still open are listed at the end.** It follows ADR-035
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
- **The one deployed connector, `orm-oms-connector`, is `RUNNING` with its task `FAILED`**, so the OMS CDC feed is
  **already down**, independent of #376. Cause, read from the task's trace and the host (nothing restarted or
  changed): the trace is `PSQLException: Connection to 100.84.50.65:5432 refused` / `java.net.ConnectException:
  Connection refused`, not an authentication error. Postgres was last started **2026-09-14 12:46 UTC**, after the
  connector's certificate files were placed (2026-09-10 and 09-13), so the most likely story is that the task
  started while Postgres was restarting for the mTLS hardening and Kafka Connect, which never restarts a
  `FAILED` task by itself, left it down. Port 5432 is reachable from inside the container now. Ruled out: the
  certificates are fine (`/tmp/orm_ca.crt` is byte-identical to the current CA, and `orm_client.crt` is
  `CN=postgres` issued by it, valid to 2028-12-01). **Not ruled out, and the thing to check before restarting:**
  `pg_hba` line 121 admits the Docker network (`172.20.0.0`) with `scram-sha-256` *before* the `postgres` cert
  rule, so a connection from the container may be asked for a password, and whether the connector's
  `database.password` is still right is unknown.
- **It points at `alpha`, which is the wrong database.** `database.dbname = alpha`, `schema.include.list = orm`,
  where `debezium/orm-oms-connector.json` in the repo says `crims`, and `crims` is the tenants' OLTP data plane
  (owner statement). The repo file is right and the deployment has drifted. A connector on `alpha` captures a
  schema that is not where tenant orders are written, so even a healthy task would not have fed `oms.*` from the
  real data. The five tables it lists are `execution`, `order`, `placement`, `order_allocation`,
  `execution_allocation`.
- Redpanda holds exactly those five `orm_oms.orm.*` topics (1 partition, 1 replica each), and there are five
  `uisce-stream-loader-*` containers, one per topic.
- **The Connect image has no Iceberg sink.** `uisce-debezium` is `debezium/connect:2.3`; `/kafka/connect` holds the
  Debezium source connectors (Postgres, MySQL, Oracle, ...) and the JDBC sink, and no Iceberg plugin.
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
    `tenant_lakehouse` table in `alpha`, no `ivy_t_*` role, and no tenant database. By the owner's statement,
    **`crims` is the OLTP data plane for tenants (today one shared database) and `alpha` is the multi-tenant control
    plane for all tenants.** So the `oms.*` gap from #376 is latent, not live; the connector's failure is not
    caused by it.
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
  **Retention is per tenant (decided 2026-10-05), which does not change this choice in a tenant database.** A 5-year and a
  15-year tenant never share a table here: each has its own database, so the drop job for a database takes its cutoff from that
  one tenant's policy and (a) is enough. (b) is the answer only where tenants share a database (any shared-database tier); it is
  not needed for mixed windows across tenants.
  **Guard the drop job's target.** Before it detaches or drops anything, the job asserts the database's tenant identity (the
  registry value for the datasource it was handed) against the database it is actually connected to, and refuses on any mismatch.
  A per-tenant `CHECK (tenant_id = ...)` stops a misrouted write but not a `DROP`; age-based detach on the wrong database is the
  one way a misrouted job destroys another tenant's books-and-records. It is the drop-side twin of the read-path predicate.
  Offboarding a tenant is dropping its database, so no detach-by-tenant lever is needed in a tenant database. Iceberg already drops per `(tenant_id, month)` once that tenant's window closes.
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

### 2. Cold: Redpanda to Iceberg in append mode, three layers, no Spark
Debezium Server has no first-party Iceberg sink, so the hop from Redpanda to Iceberg is the **Apache Iceberg Kafka
Connect sink** (REST catalog, so Lakekeeper). It runs on the Kafka Connect fleet already operated for
`uisce-debezium`, which then has only a sink role. The Connect image deployed today has no Iceberg plugin and
would need it added. The alternative with no Kafka Connect at all is a community Debezium Server Iceberg sink;
it is community-maintained, so its activity must be checked before the ADR depends on it. **Both need a version
check against current documentation; this section is written from knowledge that may be dated.**

The sink writes **append only**, never `MERGE`: ordering is per Kafka partition keyed by primary key, and current
state is derived from the log by LSN rank, so a re-emitted or out-of-order event cannot resurrect a stale value
and there is no merge window.

| Layer | Contents | Built by |
|---|---|---|
| `bronze.<table>__log` | exact Postgres columns plus `_lsn`, `_op`, `_ts`, `_tenant`; a delete is a row with `_op = 'd'`; `PARTITIONED BY (tenant_id, month)` | the sink, append only. Rollback and replay source |
| `silver.<table>` | exact Postgres schema, current state per primary key | a scheduled dedupe over bronze, `row_number() OVER (PARTITION BY pk ORDER BY _lsn DESC) = 1` and `_op <> 'd'`, written by StarRocks (`INSERT OVERWRITE`, or a materialized view). Restore source |
| `gold.<table>_flat` | the flattened, joined BI layer | StarRocks materialized views, or dbt, off silver |

(Written as a subquery with `row_number()`, not `QUALIFY`, because `QUALIFY` support in StarRocks 3.3 is a version
check; likewise `INSERT OVERWRITE` and async materialized views over an Iceberg REST catalog.)

**The isolation cost to record.** ADR-032 puts every tenant in its own warehouse with its own bucket-scoped
credential. A sink connector holds one catalog and one credential, so the cold path is **one sink connector per
tenant**. The reasoning that moved capture to one Debezium Server per tenant applies here too: Connect absorbs
connector management, at the price of a connector per tenant on a shared worker. The same revisit trigger
applies (25 tenants, or a need for shared transforms), and a Connect worker is now a place holding many tenants'
storage credentials, which the sink's secret handling (config providers reading `dscreds`, never inline
secrets) has to justify.

**Bulk moves are not the WAL path.** Two options, both Spark-free: a **Debezium incremental snapshot** (signal-table
chunks; consistent, and arrives as ordinary events into the same bronze log), or a **StarRocks JDBC external
table `INSERT INTO` Iceberg** for a one-shot backfill of a very large table. Either way the verifier is told a
window was batch-loaded rather than CDC'd.

### 3. Verification before any drop
A per-tenant, per-table, per-closed-partition verifier shaped like the audit-copy one (#377), comparing Postgres
against **silver, not the log**: page by page with a cursor; **row count and an order-independent checksum** (sum or XOR of per-row hashes) computed on both sides
**with the tenant predicate**, compared at an **Iceberg snapshot pinned to a watermark offset** so delivery lag
is not read as a failure. Findings name a partition and a kind, never a value. The outcome is recorded
non-monotonically per `(tenant, table, partition)`. There is no seal chain on these tables, so the checksum is
the evidence (ADR-035: "where present").

### 3b. Restore: what "exact schema for rollback" actually requires
Keeping the column shape is necessary, not sufficient. Restore is: **a fresh tenant database, the tenant migrations
replayed, then `COPY` from silver.** Iceberg holds data parity, not DDL; **the migration repository is the schema
source of truth.** Three things have to be decided and documented:

1. **A type-mapping table.** Checked against `0001_orm_schema.up.sql` (the types the six-schema template's `orm`, `mdm` and `cash_flow` tables
   use; the counts below were taken from the tenant `orm/0001` migration only, 34 tables, and **the array row is corrected
   to the whole template**: 32 array columns, not 1):

   | Postgres | Columns | Iceberg | Round trip |
   |---|---|---|---|
   | `uuid` | 107 | `uuid` | exact |
   | `numeric(p,s)` | 87 | `decimal(p,s)` (precision up to 38) | exact while `p <= 38`; the widest here is `numeric(24,6)` and none is unconstrained; a wider or unconstrained `numeric` would not be representable |
   | `varchar(n)` / `text` | 81 / 4 | `string` | exact; the length limit is not enforced in Iceberg, only by the restored DDL |
   | `timestamptz` | 37 | `timestamptz` (microseconds) | exact; Postgres' microsecond resolution matches |
   | `jsonb` | 28 | `string` (JSON text) | **lossy**: key order and duplicate keys are normalised by `jsonb` already; whitespace is not preserved |
   | `bool` / `date` / `int4` / `time` | 21 / 19 / 5 / 2 | `boolean` / `date` / `int` / `time` | exact |
   | arrays: `text[]` 18, `uuid[]` 8, `character varying[]` 5 (3 with a length), `integer[]` 1 | 32 | `list<string>`, `list<uuid>`, `list<int>` | exact element by element; the element length limit is not kept |

   There are no enum types in the schema. **Each new table or type needs a row here before it is tiered;** a
   verifier that compares row count and checksum on the Postgres side must hash the same canonical text it
   will restore from, or a lossy column makes every partition look different.
2. **Sequences and identity.** Restore writes explicit ids, then `setval` per sequence from `max(id)`. **The tenant
   ORM schema has none:** every primary key is a `uuid` with `gen_random_uuid()`, and the one name that looks
   like a sequence (`sequence_number`) is an ordinary `int4` column. The step still belongs in the restore
   runbook for any schema that is added later.
3. **Rollback semantics: both (decision 6).** Silver gives "restore as of now", the default runbook. "As of T" (a bad
   migration, silent corruption found a day later) is the incident runbook, rebuilt from bronze; its window is the bronze
   retention, which is the same open decision as the order and trade tables' retention (decision 3), not a second one.

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
4. The Iceberg sink and the three layers (2), then the verifier (3), the restore runbook (3b), then the drop job (4).
5. **The books-and-records tables last, and not without decision 3.** Until it is answered they get the
   `tenant_id` invariant and no drop policy.

## Decisions

1. **Resolved by owner direction:** Debezium Server per tenant database, with the revisit trigger above.
2. **Slot policy (accepted by the owner, 2026-10-04): this is policy, not an open question.**
   - **One replication slot per tenant database**, so losing a slot costs one tenant, never the fleet.
   - **A lag alarm on every slot**, so a stalled consumer is seen long before it matters.
   - **A WAL retention cap** (`max_slot_wal_keep_size`) set so a stalled slot is **dropped before it fills a shared cluster's
     disk**. The recovery for a dropped slot is a **re-snapshot**, which append-only bronze plus the silver dedupe tolerates
     (re-emitted events are ranked by LSN, so a replay cannot resurrect a stale value).
   - **Preconditions not yet met on the dev host:** it had 8 of 10 slots in use, unlimited retention and 15 GB of free disk. Per-tenant
     slots cannot start until `max_replication_slots` is raised (a restart) and the cap is a standing setting. (The cap was set to
     10 GB on 2026-10-04 and seven dead slots went `lost`; see "Immediate host actions".)
3. **Order and trade tables (`order_event`, `execution`, ...): needs the business owner. This is the one open item left.** It decides
   three things at once: books-and-records retention, whether a copy in object storage may be the record once the OLTP row is gone, and
   how long bronze is kept (which is the point-in-time restore window of decision 6).
   **Revised shape of this decision (2026-10-05): retention is tenant configuration, bounded by the platform.**
   - Stored per tenant in the gold copy's contractual config, `NOT NULL` at provisioning (the saga requires it the way it requires
     `app`), with optional per-table overrides if `order_event` and `execution` differ from the default. The owner supplies the
     **default, minimum and maximum**; values outside them are refused at provisioning and on update. **Bounds: TBD pending the
     business owner.**
   - **Extending is a config update; shortening is a delete.** A shortening either refuses until the data ages out, or is an explicit,
     audited purge operation, never a config edit.
   - **The drop job is tenant-aware and fails closed:** each partition's expiry is computed from its tenant's policy; a missing or
     unreadable value means that tenant's data is not dropped (never a default of zero).
   - **Bronze retention is per tenant too** (the restore window each tenant contracted for), independent of books-and-records
     retention, held in the same config row.
   - Questions to the owner: default/min/max for `order_event` and `execution`; who signs off on a value outside the default;
     whether shortening is ever allowed after provisioning (our position: only by an audited purge).
4. **Pilot table:** `quote`, unless it is itself under a best-execution retention rule, in which case it moves to
   decision 3.
5. **Resolved by owner direction:** no Spark. The write side is the Iceberg Kafka Connect sink in append mode, with
   bronze, silver and gold. Two checks remain before the ADR: the sink's capabilities in the target version and,
   if Kafka Connect is to be avoided entirely, whether the community Debezium Server Iceberg sink is maintained.
6. **Rollback semantics: decided, point-in-time as well as now (owner, 2026-10-04).** Two runbooks, not one:
   - **Restore as of now** (the default runbook): a fresh tenant database, the migrations replayed, `COPY` from **silver**.
   - **Restore as of time T** (the incident runbook: a bad migration, silent corruption found a day later): rebuilt from the **bronze log**
     up to T. **Its window is the bronze retention**, nothing else: a T older than the oldest retained bronze partition cannot be restored.
   The bronze retention is **not set here and is not a second open item**: it is the same answer as decision 3. Until the business owner
   answers for `order_event`, `execution` and the other order and trade tables, those tables get the `tenant_id` invariant, read-side
   `tenant_id` partitioning, **no drop policy, and no defined bronze retention.**
7. **Resolved by the owner's statement:** the repo is right and the deployment is wrong. The connector must read
   `crims`, not `alpha`.

## Immediate host actions (state as of 2026-10-04 ~13:20 UTC)
1. **WAL retention cap: done.** `max_slot_wal_keep_size = '10GB'` (a reload) and a `CHECKPOINT`. WAL fell from
   13.7 GB to 4.2 GB. Seven slots, all more than 10 GB behind and inactive for weeks, went to `wal_status = lost`
   (unusable and unrecoverable; their consumers would have to re-snapshot): `debezium`, `debezium_alpha`,
   `debezium_alpha_orders`, `dbz_northwinds_customers`, `northwinds_cdc_slot`, `semlayer_cdc_slot`,
   `semlayer_lookups_sub_slot`. A pre-change record of all eight slots is in the session scratchpad
   (`pre_state.txt`: names, restart LSNs, retained bytes). `orm_oms_slot` (4.1 GB) was inside the cap and is intact.
2. **Dropping the seven lost slots: not done.** The attempt was denied by the session's permission classifier
   (shared resource), and was not retried. They are dead entries, but they still occupy 7 of the 10
   `max_replication_slots`. Someone with authority over the host runs
   `select pg_drop_replication_slot(slot_name) from pg_replication_slots where wal_status = 'lost'`.
3. **Repointing the ORM connector at `crims`: not done, because the data contradicts the premise.** On this host:
   - **`alpha.orm` holds the data:** 34 tables, 187 orders, 99 executions, 86 placements, 91 order allocations, 3
     execution allocations. This is the schema `0001_orm_schema.up.sql` was derived from.
   - **`crims.orm` is empty:** 80 tables, **0 rows** in all five captured tables, and the publication
     `orm_cdc_publication` already exists there with those five tables and no slot.
   - **`crims` holds `mdm`:** 441 tables there, with data (`calendar_day` 7,671 rows and others), plus `wlth`,
     `ref`, `streaming`, `staging`, `migration`, `cash_flow`, `vend`.
   So the deployed connector's `alpha` is where the ORM data is written *in this environment*, whatever the intended
   design, and repointing it at `crims` would capture an empty schema and strand the only real data. The owner
   states that `crims` is the tenants' OLTP data plane; either that is the target and the ORM data has not been
   put there yet (then there is nothing to capture until it is), or `alpha.orm` is where the working ORM
   lives and the statement describes the intended end state. **This needs the owner's answer before any
   connector change.** The failed task could be restarted unchanged (it would resume `orm_oms_slot` on `alpha`),
   but that decision was not made either.

**Onboarding, as the owner describes it:** a new tenant's metadata is added to `alpha`, and the tenant's structure
is created from the `crims` database's `orm` and `mdm` schemas. The tenant migrations in the tree cover **only
`orm`** (`db/tenant_migrations/orm`, derived from `alpha`'s 34-table `orm`, not `crims`'s 80); there is **no `mdm`
tenant migration**, and `crims.mdm` is 441 tables. Which schema is the template (`alpha.orm` 34 tables or
`crims.orm` 80), and how `mdm` is carved into a tenant migration, are open and decide what per-tenant CDC captures.

## What is still unverified
- Why the deployed connector's task fails (its certificate files under `/tmp` and the post-hardening `pg_hba` are
  the places to look; I did not open the container).
- Where tenant databases will be hosted, which decides the `pg_hba` question above.
- The Iceberg Kafka Connect sink against Lakekeeper with a tenant's credential, and StarRocks 3.3's `INSERT
  OVERWRITE` and async materialized views over that catalog. Nothing here has run them.
