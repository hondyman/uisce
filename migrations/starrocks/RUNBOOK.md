# Runbook — StarRocks OLAP coverage for trigger-derived CDC

## What was built

| File | Purpose |
|---|---|
| `migrations/starrocks/002_agg_layer.sql` | DDL for 15 destination tables in the `agg` database |
| `migrations/starrocks/003_agg_mvs.sql` | 5 materialized views for high-value rollups |
| `docker-compose.remote.yml` | 15 new `uisce-stream-loader-alpha-*` and `uisce-stream-loader-crims-security-identifier` services |

The 5 existing `orm_oms.*` stream loaders keep running unchanged. The new 15 loaders feed the separate `agg` database to keep concerns isolated from the OMS speed layer.

## Topology

```
alpha DB ──Debezium──> alpha_trg.public.<table> ──Kafka──> stream loader ──StarRocks agg.<table>
crims DB ──Debezium──> crims_trg.orm.security_identifier ──Kafka──> stream loader ──StarRocks agg.crims_security_identifier
```

Each loader is one container per topic. The loader uses the existing `backend/cmd/stream_loader/main.go` unchanged — the topic name and target table are env vars.

## Execute — step by step

### 1. Apply the base DDL

```bash
docker exec -i starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root \
  < migrations/starrocks/002_agg_layer.sql
```

Expected output: 15 `CREATE TABLE` acknowledgements. Verify:
```bash
docker exec starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root -e \
  "USE agg; SHOW TABLES;"
```

Should list 15 tables prefixed `alpha_` or `crims_`.

### 2. Apply the materialized views

```bash
docker exec -i starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root \
  < migrations/starrocks/003_agg_mvs.sql
```

Verify:
```bash
docker exec starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root -e \
  "USE agg; SHOW MATERIALIZED VIEWS;"
```

Should list 5 MVs ending in `_mv`.

### 3. Build and bring up the new stream loaders

**Pre-check: validate topic names match the connector.**

Before deploying, verify the topic names embedded in each loader's env match what the alpha-trg / crims-trg connectors actually publish:

```bash
# Topic names referenced by the loaders (must list exactly 14 alpha_trg.* + 1 crims_trg.orm.security_identifier)
grep -oE '(alpha_trg|crims_trg)\.[a-z_.]+' docker-compose.remote.yml \
  | sort -u

# Topic names declared by the connectors (must match):
jq -r '.config."table.include.list"' debezium/alpha-trg-connector.json debezium/crims-trg-connector.json
# Then the topic becomes: <topic.prefix>.<schema>.<table>
# alpha-trg: alpha_trg.public.<table>
# crims-trg: crims_trg.orm.<table>
```

If a topic name in compose doesn't match what the connector publishes, the loader will silently sit at "no messages" forever. The most likely failure mode with 15 near-identical blocks.

```bash
cd /home/eganpj/uisce  # or /mnt/github/uisce - whichever compose file is authoritative
docker compose -f docker-compose.remote.yml build \
  $(docker compose -f docker-compose.remote.yml config --services | grep stream-loader-alpha) \
  uisce-stream-loader-crims-security-identifier

docker compose -f docker-compose.remote.yml up -d \
  $(docker compose -f docker-compose.remote.yml config --services | grep stream-loader-alpha) \
  uisce-stream-loader-crims-security-identifier
```

Verify all are running:
```bash
docker ps --filter "name=stream-loader" --format "{{.Names}}: {{.Status}}"
```

The 14 alpha + 1 crims loaders will all show "Up" but their logs will say "Listening for CDC events... no messages yet" until the alpha-trg / crims-trg connector snapshots complete (5–15 min). That's expected.

### 4. Verify data flowing

Each loader logs its first consume on startup. To smoke-test:

```sql
-- pick a loader that should have data quickly (template_ratings is mid-volume)
docker exec starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root -e \
  "SELECT COUNT(*) FROM agg.alpha_template_ratings;"

-- screening funnel MV
docker exec starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root -e \
  "SELECT current_stage, COUNT(*), AVG(screening_score) FROM agg.alpha_screening_funnel_mv GROUP BY current_stage;"
```

Empty counts at first are normal — the CDC connectors may not have snapshotted yet. Wait until phase 6 (connector snapshot) of the trigger migration finishes before expecting data.

### 5. Schema sanity check (catches loader/type mismatches)

After the connector snapshot finishes (typically 5–15 min), run the null-column validator. It surfaces columns that landed as 100% NULL because the stream loader couldn't decode the source type:

```bash
./scripts/check_agg_nulls.sh
```

The script exits non-zero with a list of all-NULL columns — those indicate a schema mismatch to investigate (interval / date / numeric precision). All-NULL is rare in production data and almost always means `decodeDecimals()` in `cmd/stream_loader/main.go` doesn't handle that yet. Add a case there or change the column to VARCHAR.

### 6. Run unit tests for the loader (optional, pre-deploy only)

```bash
cd backend && go test ./cmd/stream_loader/... -v
```

Tests cover `decodeDecimals`, `durationMicros`, date formatting, and the two main `decodeRecord` paths (delete vs upsert).

## Operational notes

### Refresh lag

The 5 MVs use `REFRESH ASYNC EVERY (INTERVAL N MINUTE)`. The slowest is `crims_identifier_coverage_mv` at 30min. Dashboards reading the MVs can lag by up to that interval. For real-time accuracy, hit the base tables directly.

### Screening funnel MV depends on consumer health

`alpha_screening_funnel_mv` rolls up `current_stage` from `alpha_investment_opportunities`. **The `current_stage` column is written by `aggregate-consumer-alpha`'s screening handler** (the in-Postgres CDC echo — see `backend/cmd/aggregate_consumer/handlers/screening.go`). The funnel MV therefore has the consumer as a hard dependency: if the consumer is DLQ-bound or down, the funnel silently freezes (data is not wrong, just stale).

When investigating "MV looks stale":
1. Check the consumer DLQ: `docker logs uisce-aggregate-consumer-alpha --tail=100 | grep -i dlq`
2. Check the consumer is healthy: `docker ps --filter name=uisce-aggregate-consumer-alpha`
3. If consumer is fine, check the stream-loader for `alpha_investment_opportunities` lag, not the MV itself.

### Source of truth per MV

| MV | StarRocks use | Postgres source of truth |
|---|---|---|
| `alpha_screening_funnel_mv` | Analytics, dashboards | `public.investment_opportunities.current_stage` (set by consumer) |
| `alpha_template_rating_distribution_mv` | Analytics, leaderboards (all ratings) | `public.process_templates.rating_average` for in-app display (approved-only) |
| `alpha_crypto_throughput_mv` | Portfolio analytics | `public.crypto_transactions` (raw) |
| `alpha_workflow_latency_mv` | SLA / reliability | `public.process_execution_metrics` (raw) |
| `crims_identifier_coverage_mv` | Identifier coverage | `crims.orm.security_identifier` (raw) |

Use Postgres for transactions / in-app display; use the MV for bulk analytics. Never read from a MV in OLTP hot paths.

### Replication

StarRocks tables use `DUPLICATE KEY` with `replication_num = "1"` because the cluster is single-node. If you go multi-node, bump to 3 once the BE cluster is set up.

### Bucket count

Buckets are sized by the largest expected per-key cardinality:
- `alpha_investment_opportunities`: 16 (advisor portfolios × stages)
- `alpha_crypto_transactions`: 16 (high write rate)
- `alpha_template_ratings`: 8 (mid-volume)
- `alpha_metrics_registry`: 4 (low-volume, ~100s of rows)
- `alpha_screening_funnel_mv`: 8 (grouped by client_id × advisor_id × stage × type)

If OLAP queries get slow on a specific table, bump the bucket count and re-issue `ALTER TABLE` with the new distribution.

### Topic auto-creation

Stream loaders pre-create topics via their own consumer-group launches, so if you bring them up before the alpha-trg connector is registered, they'll just idle until data arrives. Order of operations doesn't matter for the loaders — they reconnect on retry.

### Dropping or replacing a loader

Each loader is a one-container-per-topic pattern. To kill:
```bash
docker stop uisce-stream-loader-alpha-template-ratings
```

The corresponding topic in Kafka retains the unread offsets. New container resumes from where the old one left off (consumer-group offset persists in Redpanda).

## Post-cutover backlog (do not lose these)

### Backlog 1: Convert `oms.*` stream loaders to PRIMARY KEY model + delete handling

The 5 existing `uisce-stream-loader-{execution,order,placement,order-allocation,execution-allocation}` services still write to `oms.*` tables defined outside this file under `DUPLICATE KEY`. They have the same latent gap this work closed for `agg.*`: deletes in Postgres source tables land as no-op upserts in StarRocks, leaving phantom rows that the source no longer has.

**Conversion steps (only after the alpha-trg StarRocks delete integration test above proves the working DDL/curl pattern):**

1. Apply the winning variant's `__op` column handling to each `oms.*` table (likely adding `__op INT DEFAULT 0`).
2. Convert each `oms.*` table from `DUPLICATE KEY (pk)` → `PRIMARY KEY (pk)`.
3. The stream loader's `cmd/stream_loader/main.go` already handles delete-emission; no Go code changes needed.
4. Add `PRIMARY_KEY_COLUMN` env var to each oms loader's compose service — defaults to `id`, override for any table whose PK isn't named `id`.
5. Run the parity window again to confirm row counts match between alpha source tables and StarRocks oms.* tables.

This is parked for after go-live because it doesn't touch the trigger-drop critical path.

### Backlog 2: Real-vector Duration decoder test

After the first `process_execution_metrics` CDC row with `end_time` set has been emitted to the topic, capture one event's `payload.after.duration` value (`rpk topic consume ... | jq '.payload.after.duration'`), bake the base64 string into a `TestDecodeDurationB64_HardcodedRealVector` test, and assert decode returns the expected microseconds. This catches any wire-format asymmetry with the actual producer that the round-trip tests can't.

Until this lands, `alpha_process_execution_metrics.duration` may be NULL if the wire format doesn't match the round-trip assumption — the null-column validator (`scripts/check_agg_nulls.sh`) surfaces that as an all-NULL column.

### Backlog 3: Compliance check CI gate

The trigger-drop compliance check (`backend/db/migrations/_reference/20260920_002_trigger_compliance_check.sql`) should run on every PR touching `backend/db/migrations/` to prevent re-introduction of non-`updated_at` triggers. Implementation: a CI job that spins up a postgres, applies migrations, then runs the compliance query and exits non-zero on any row.

## Rollback

### Drop the `agg` database

```sql
docker exec starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root -e "DROP DATABASE IF EXISTS agg CASCADE;"
```

Then bring down the loaders:
```bash
docker compose -f docker-compose.remote.yml stop \
  $(docker compose -f docker-compose.remote.yml config --services | grep stream-loader-alpha) \
  uisce-stream-loader-crims-security-identifier
```

The topics in Redpanda stay (with unread offsets if the loaders were stopped before consuming).

## What's NOT covered here

- `crypto_holdings` — this is the consumer-maintained aggregate; doesn't have a CDC publication of its own (only `crypto_transactions` does, and the consumer recomputes from the full transaction history per event).

## StarRocks delete integration test (5-min time-box)

The stream-loader emits a StarRocks stream-load DELETE for op=d events via the columns-projection pattern (`columns: __op=1,<pk_col>`). **This was not integration-tested end-to-end before merge** — the unit tests cover the loader's encoding logic, but the live cluster's PK-table merge semantics are what determine whether the delete actually fires.

Run this on the host after the alpha-trg connector has captured its first events (so the agg database exists and the stream_load path is verified working for inserts). Each variant takes ~30 seconds. **Stop at the first variant that produces a true delete.** Run them in this order because Variant 1b is the most likely correct shape:

### Variant 1b — `__op` column in DDL + `merge_type` directive + `columns` header

```bash
docker exec starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root <<'SQL'
DROP TABLE IF EXISTS delete_test;
CREATE TABLE delete_test (
  id      VARCHAR(36),
  name    VARCHAR(255),
  __op    INT DEFAULT 0
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(id) BUCKETS 4
PROPERTIES ("replication_num" = "1");
INSERT INTO delete_test VALUES ('a1', 'row-a1', 0), ('a2', 'row-a2', 0);
SELECT 'before-delete', * FROM delete_test;
SQL

curl -s -u root: -XPUT http://starrocks-fe:8030/api/agg/delete_test/_stream_load \
  -H "Expect: 100-continue" \
  -H "Content-Type: application/json" \
  -H "format: json" \
  -H "strip_outer_array: false" \
  -H "merge_type: columns" \
  -H "columns: __op=1,id" \
  -d '{"id":"a1"}'

docker exec starrocks-fe mysql -h 127.0.0.1 -P 9030 -u root -e \
  "SELECT 'after-delete', * FROM agg.delete_test;"
```

**Expected:** Only `a2/row-a2` remains. If you see `a1/row-a1/0` — Stream load with `__op=1` row inserted a literal row with that PK. Variant 1b failed.

### Variant 1 — `__op` column in DDL + `columns` header only (no merge_type)

Same DDL as above. Same body. The only difference is dropping the `merge_type: columns` header.

**Expected:** Same as 1b — either delete fires or literal row is inserted.

### Variant 2 — No `__op` column in DDL + `columns` header

```sql
DROP TABLE IF EXISTS delete_test;
CREATE TABLE delete_test (
  id   VARCHAR(36),
  name VARCHAR(255)
) ENGINE = olap
PRIMARY KEY (id)
DISTRIBUTED BY HASH(id) BUCKETS 4
PROPERTIES ("replication_num" = "1");
INSERT INTO delete_test VALUES ('a1','row-a1'), ('a2','row-a2');
```

Same curl as 1b with `columns: __op=1,id` and body `{"id":"a1"}`.

**Expected:** A delete without a `__op` column declared usually fails because StarRocks doesn't recognize the projection. If a row materializes anyway, that's a successful upsert of a row that has only the `id` populated — i.e., partial-update delete. If row count drops to 1, this DDL shape works.

### Variant 3 — Control: no `__op` declaration, just send `{"id":"a1"}`

Same DDL as Variant 2. Body is just `{"id":"a1"}`, no header projection.

**Expected:** This is the baseline. If THIS doesn't produce a row, the entire stream-load config has a deeper issue and all other variant results are untrustworthy. Must show `a2/row-a2` + `a1/NULL` (PK-only partial update). If anything else happens, debug the stream-load setup before retrying variants.

### Recording the result

Once a variant produces a true delete:

1. **Append the working curl verbatim** to the `002_agg_layer.sql` header comment next to "Delete shape" — replace the placeholder.
2. **Apply the same DDL shape + curl to all 15 agg.* tables**: add `__op` column if winning variant requires it, or omit it if the columns projection without declaration works. The 002 DDL is the single source of truth.
3. **If all four variants fail**: stop the 5-minute window. Open the post-cutover backlog for Option B (batched MySQL-protocol deletes; sync.Map flush-on-nonempty, no idle timer).

## Real-vector Duration capture (gated, post-first CDC row)

The `decodeDurationB64` decoder handles the Kafka Connect Duration wire format on the assumption that:

- 12-byte structure
- 3 × big-endian int32 (months, days, millis)
- base64-encoded in the JSON value field

The unit tests pin this with round-trip vectors, but they don't prove the wire format matches what Debezium's JSON converter actually emits at runtime. **Decode failures fail soft** (length guard returns `ok=false`, decoder leaves the value untouched → StarRocks reads it as the undecoded string → NULL on type mismatch), so a wrong assumption doesn't cause silent garbage — it just produces NULL durations in `alpha_process_execution_metrics.duration` until the decoder is fixed.

**Capture procedure (gated; run after step 6 of the master sequence lands):**

```bash
# On the host, capture one event from the topic once a row with end_time set has arrived:
rpk topic consume alpha_trg.public.process_execution_metrics \
  --num 1 --format json --timeout 10s \
  | jq '.payload.after.duration'
# The value is the base64 string we'll hardcode into the test.

# Bake it in:
# 1. Copy the printed base64 string
# 2. Add a TestDecodeDurationB64_HardcodedRealVector test in cmd/stream_loader/main_test.go
# 3. Assert decodeDurationB64(<that base64>) == the expected micros value
#    (compute expected from the source row's end_time - start_time, multiplied by 1000)
# 4. Re-run `go test ./cmd/stream_loader/... -v`
```

If the round-trip assertion fails, the decoder assumption is wrong and `alpha_process_execution_metrics.duration` will be all-NULL. Fix the decoder (likely an endianness or component-order assumption) and re-deploy.

Track this in the post-cutover backlog; it does not block go-live.
