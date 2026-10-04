# Tenant data tiering: design for review

**Status: proposed. Nothing here is built, and four decisions below are yours.** It follows ADR-035
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

### 1. One CDC topology, per tenant database, feeding both warm and cold
- A Debezium connector per tenant database on **Kafka Connect** (not Debezium Server, which is one
  connector per instance and would mean hundreds of containers). Slot and publication named for the tenant,
  `topic.prefix = ivy_t_<tenant id without hyphens>`, so a tenant's topics are namespaced by construction
  and an ACL can be written per prefix.
- Provisioned by the existing tenant provisioning saga, as one more step after `ProbeTenantDatabase`, and
  removed by offboarding. Connection credentials come from `dscreds` by the tenant's own role (a
  replication-capable role per tenant, never a shared one).
- Consumers read by topic prefix, so a loader can only be given one tenant's topics. This replaces the
  hard-coded `AssignedTenantID`/`CDC_TOPIC` pairs.

### 2. Cold: Kafka to Iceberg per tenant, written by StarRocks
Same engine and the same reasons as ADR-036 (no mature Go Iceberg writer; StarRocks already holds the
tenant's catalog and credential): a StarRocks Routine Load or `INSERT ... SELECT` from the tenant's topic
into `<tenant catalog>.<app>.<table>`. One writer per tenant and table; resume point read from the
destination, not a counter (ADR-036's rule), because an Iceberg append is not idempotent. Batched, not per
event, for the Object Lock reason in ADR-036.

### 3. Verification before any drop (the gate ADR-044 pointed at, correctly this time)
A `TenantTableVerify` shaped like the audit-copy verifier (#377): per tenant and table, per closed
partition, page by page with a cursor; compare **row count and a per-partition checksum** computed in
Postgres against the same in Iceberg. Findings name a partition and a kind, never a value. The outcome is
recorded non-monotonically (a later bad run takes back an earlier pass) per `(tenant, table, partition)`.
There is no seal chain on these tables, so ADR-035's "where present" applies and the checksum is the
evidence.

### 4. The drop job
Reads `hot_window_days` and `legal_hold` from the binding (ADR-038). Detaches then drops a partition only if
**all** hold: the partition is wholly older than the window, `legal_hold` is false, and its recorded
verification is clean and newer than the last time the partition could have changed (a closed partition is
immutable by rule; late-arriving rows into it are a finding, not something to absorb). Detach first and drop
in a later run, so a mistake is recoverable by re-attaching until the drop.

### 5. Order of work, and where it stops
1. **CDC per tenant database** (section 1), including a statement of what happens to a moved tenant's
   `oms.*` today. This is the prerequisite and it fixes a live gap.
2. **Partition `pnl_intraday` and the snapshot tables, and give `quote` real partitions.** Only `quote`
   needs no key change. Tables whose PK is `(id)` need `(id, <time key>)` and their dependants' foreign keys
   reviewed; that is a migration per table in the tenant migrations, applied to each tenant database through
   the existing fleet runner.
3. Cold sink (section 2), then the verifier (3), then the drop job (4).
4. **The books-and-records tables last, and not without a decision** (question 3).

## Decisions I need

1. **Kafka Connect, not Debezium Server, for per-tenant CDC.** You said Debezium Server is what is deployed;
   it is, for `alpha`'s `iam` tables, and the ORM connector already runs on Kafka Connect. A connector per
   tenant on Debezium Server means one server instance per tenant. I recommend Kafka Connect. If you
   would rather keep Debezium Server for everything, say so and I will design around instances per tenant
   group, but I expect it to be harder to operate.
2. **Replication slots are a cluster-wide resource.** One slot per tenant database means `max_replication_slots`
   and `max_wal_senders` bound tenants per cluster, and **a stalled consumer makes Postgres retain WAL for that
   slot until the disk fills, on a cluster shared by other tenants.** Mitigation I propose: a lag alarm on every
   slot, `max_slot_wal_keep_size` set so a stalled slot is dropped before it takes the cluster down (a dropped
   slot means a re-snapshot, which the Iceberg write must tolerate), and a cap on tenants per cluster derived
   from slots. Do you accept slot loss and re-snapshot as the failure mode, or do you want a different one?
3. **Are the order and trade tables to be dropped from Postgres at all?** ADR-035 says event tables move to
   Iceberg after 90 days. For `order_event`, `order_history`, `execution` and friends that is a books-and-records
   retention question (what period, and whether the regulator's rules allow a copy in object storage to be the
   record once the OLTP row is gone). I will not partition or drop them on my own judgment.
4. **Pilot table.** I recommend `quote` first (already partitioned, needs no key change, highest volume, a
   market-data table and not a record of your own activity), unless it is itself in scope of a best-execution
   retention rule, in which case it moves to question 3.

## What I did not verify
- Whether any tenant is already on its own ORM database in a deployed environment (decides how urgent the
  `oms.*` gap is).
- Postgres cluster limits in your environment (`max_replication_slots`, WAL headroom).
- That StarRocks 3.3 can `INSERT ... SELECT` from a Kafka-fed table into an Iceberg REST catalog the way ADR-036
  already assumes for audit; the audit copy's smoke test passed, which covers the write path but not a
  Kafka source.
