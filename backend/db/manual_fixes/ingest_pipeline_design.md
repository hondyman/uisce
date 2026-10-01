# Product Master — Ingest Pipeline Design

Owner: [CONFIRM]
Status: design
Target: `crims.staging` → `crims.mdm.product_golden_record`

---

## 1. Component map

```
┌─────────────────┐
│  Vendor file    │  SFTP / API / manual upload
│  (FactSet,      │
│   BBG, RDP)     │
└────────┬────────┘
         │
         ▼
┌─────────────────┐      ┌──────────────────────┐
│  ingest-loader  │─────▶│  staging.ff_product  │
│  (Go service)   │      │  staging.bbg_product │
│                 │      │  staging.rdp_product │
└─────────────────┘      └──────────┬───────────┘
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │  map-transformer     │
                         │  (Go service)        │
                         │                      │
                         │  reads: product_     │
                         │    field_mapping     │
                         │  reads: product_     │
                         │    type_mapping      │
                         └──────────┬───────────┘
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │ staging.product_     │
                         │ incoming             │
                         └──────────┬───────────┘
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │ match-resolver       │
                         │ (Go service)         │
                         │                      │
                         │ writes: entity_xref  │
                         │ writes: product_     │
                         │   match_candidate    │
                         └──────────┬───────────┘
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │ survivorship-engine  │
                         │ (Go service)         │
                         │                      │
                         │ reads: survivorship_ │
                         │   rule               │
                         │ reads: source_       │
                         │   priority           │
                         │ writes: product_     │
                         │   survivorship_log   │
                         └──────────┬───────────┘
                                    │
                                    ▼
                         ┌──────────────────────┐
                         │ golden-publisher     │
                         │ (Go service)         │
                         │                      │
                         │ writes: product_     │
                         │   golden_record      │
                         │ writes: product_     │
                         │   golden_field       │
                         └──────────────────────┘
```

Five services. Each idempotent, each with its own `_load_run` progress marker.

---

## 2. Service 1 — `ingest-loader`

**Trigger:** file arrival (SFTP poll, S3 event, manual upload via API).

**Responsibility:** land the file verbatim into the staging table.

**Flow:**

```
1. Compute file hash (SHA-256)
2. Check idempotency:
     SELECT * FROM staging._load_run
     WHERE source_system_cd = :src
       AND domain = :domain
       AND run_ref = :run_ref
   If exists AND status = COMPLETED: return early (already ingested).
3. Insert _load_run (status=RUNNING)
4. Stream file → parse rows → INSERT into staging.<vendor>_product
   Each row gets _load_run_id and _source_row_num
5. Update _load_run: received_rows, status=COMPLETED
6. Emit event: ingest.completed
```

**Failure modes:**

| Failure | Action |
|---|---|
| File schema mismatch | Reject file, _load_run status=FAILED, alert |
| Row parse failure | Write row to `_mapping_error` with `TRANSFORM_ERROR` |
| Duplicate run_ref | Early return (idempotent) |
| Partial file | _load_run status=PARTIAL |

**CLI:**
```
ingest-loader --source FACTSET --file /path/to/file.txt --run-ref 20260924-01
```

**Config:**
```yaml
sources:
  FACTSET:
    format: pipe_delimited
    delimiter: "|"
    header_row: true
    table: staging.ff_product
    columns: [fsym_id, isin, cusip, ...]   # maps file position → column
  BBG:
    format: csv
    table: staging.bbg_product
  RDP:
    format: parquet
    table: staging.rdp_product
```

---

## 3. Service 2 — `map-transformer`

**Trigger:** scheduled (after ingest) or event-driven.

**Responsibility:** apply `product_field_mapping` and `product_type_mapping` to convert staging rows into canonical shape.

**Flow:**

```
1. For each _load_run where status=COMPLETED and not yet mapped:
2. Load field mappings for the source:
     SELECT vendor_field, internal_table, internal_field, transform_expression
     FROM mdm.product_field_mapping
     WHERE mdm_source_system_id = :src_id AND is_active
3. For each staging row:
   a. Build canonical_payload JSONB
   b. For each mapping:
      - If transform is NULL: raw value → target
      - If transform is '__type_map__':
          look up vendor value in product_type_mapping → internal_type_cd
      - If transform is '__parse_bbg_ticker__':
          parse "TICKER COUNTRY Type" → extract ticker, country
      - If transform is SQL expression (e.g. UPPER(TRIM(%s))):
          substitute %s with escaped value, eval
   c. Validate required fields (is_required=true)
   d. INSERT into staging.product_incoming
4. Update _load_run: accepted_rows, rejected_rows
5. Emit: map.completed
```

**Transform expression engine:**

Start with a small set of named transforms (no arbitrary SQL):
- `UPPER(TRIM(%s))`
- `LOWER(TRIM(%s))`
- `__type_map__` — lookup
- `__parse_bbg_ticker__` — BBG-specific
- `__to_date__` — parse various date formats
- `__to_numeric__` — locale-safe numeric parse

Later: allow SQL, evaluate in a sandbox.

**Error handling:** any row that fails validation → `is_valid=false`, `validation_errors` populated, continues. Row still written (for auditing).

---

## 4. Service 3 — `match-resolver`

**Trigger:** scheduled (after map).

**Responsibility:** link each incoming row to a golden entity (existing or new).

**Flow:**

```
1. For each product_incoming row (is_valid=true):
2. Apply match rules in priority order:
     SELECT * FROM mdm.product_match_rule
     WHERE is_active ORDER BY priority ASC
3. For each rule:
   a. Extract match keys from canonical_payload
   b. Query entity_xref for existing golden_id with same keys
   c. If deterministic key match (ISIN, CUSIP, SEDOL): confidence=100
   d. If fuzzy key match: compute similarity score
4. Decision:
   - confidence >= auto_merge_threshold → write entity_xref, done
   - review_threshold <= confidence < auto → write product_match_candidate
   - confidence < review → create new product, new entity_xref
5. For new products: INSERT mdm.product with product_cd (from primary ID)
6. Emit: match.completed
```

**Match rules (initial):**

```
Rule 1 (priority 10): ISIN exact           → 100% auto-merge
Rule 2 (priority 20): CUSIP exact          → 100% auto-merge
Rule 3 (priority 30): SEDOL exact          → 100% auto-merge
Rule 4 (priority 40): FSID/BBGID/RIC exact → 95% auto-merge
Rule 5 (priority 50): Ticker+CCY+Domicile  → 80% auto-merge
Rule 6 (priority 60): Legal name fuzzy+Dom → 60-85% review
```

---

## 5. Service 4 — `survivorship-engine`

**Trigger:** scheduled (after match).

**Responsibility:** for each golden entity, pick winning value per field.

**Flow:**

```
1. For each golden_id with new contributions since last run:
2. For each field in the survivorship ruleset:
   a. Gather all values:
        SELECT source_system_id, value, _ingested_at
        FROM staging.product_incoming
        JOIN mdm.entity_xref
          ON entity_xref.source_entity_id = product_incoming.source_row_id
        WHERE entity_xref.golden_entity_id = :gid
          AND product_incoming.canonical_payload ? :field
   b. Look up rule:
        SELECT * FROM mdm.survivorship_rule
        WHERE entity_type='PRODUCT' AND attribute_name=:field
   c. Apply strategy:
        - SOURCE_PRIORITY: pick highest-priority source per source_priority
        - PROVIDER_AUTHORITATIVE: pick from rule.priority_vendors[0]
        - MOST_RECENT: pick latest _ingested_at
        - HIGHEST_CONFIDENCE: pick highest confidence
   d. Write product_survivorship_log row
3. Emit: survivorship.completed
```

**Field groups** drive batching: process IDENTITY fields first (fast), then NAME, CLASSIFICATION, etc.

---

## 6. Service 5 — `golden-publisher`

**Trigger:** scheduled (after survivorship).

**Responsibility:** materialize the golden record.

**Flow:**

```
1. For each golden_id with new survivorship decisions:
2. Determine next golden_version:
     SELECT COALESCE(MAX(golden_version),0)+1
     FROM mdm.product_golden_record
     WHERE tenant_id=:t AND product_id=:gid
3. Mark prior version: is_current=false
4. Insert new product_golden_record:
     - golden_attributes = JSONB of all winning values
     - winning_sources = JSONB of {field → source_system_id}
     - overall_dq_score = computed from dq_rules
     - status = 'PUBLISHED'
5. Insert product_golden_field rows (one per field) with provenance
6. Update mdm.product: is_golden_record=true, dq_score
7. Emit: golden.published
```

**Golden record is immutable.** Re-running creates a new version. Never overwrites.

---

## 7. Orchestration

**Option A: Airflow / Dagster / Prefect**
```
DAG: product_master_daily
  ingest_factset → ingest_bbg → ingest_rdp
    → map_transformer
    → match_resolver
    → survivorship_engine
    → golden_publisher
    → notify
```

**Option B: Go orchestrator + Kafka**
Each service is a consumer; emits `*.completed` events.
Chain: `ingest.completed` → `map` → `map.completed` → `match` → ... → `golden.published`.

**Option C: Cron + CLI**
```
0 2 * * *  ingest-loader --source FACTSET --file $FACTSET_DAILY
0 3 * * *  ingest-loader --source BBG    --file $BBG_DAILY
0 4 * * *  ingest-loader --source RDP    --file $RDP_DAILY
0 5 * * *  map-transformer --domain PRODUCT
0 6 * * *  match-resolver --domain PRODUCT
0 7 * * *  survivorship-engine --domain PRODUCT
0 8 * * *  golden-publisher --domain PRODUCT
```

Start with Option C. Move to A or B when volume justifies.

---

## 8. Observability

Every service writes to `staging._load_run` and `mdm.product_feed_health`.

| Metric | Source | Alert threshold |
|---|---|---|
| Rows in per load | `_load_run.received_rows` | ±20% vs 7-day avg |
| Rejected rows | `_load_run.rejected_rows` | > 5% of received |
| Mapping errors | `_mapping_error` count per run | > 1% of rows |
| Match auto-merge rate | `entity_xref` inserts | < 60% → review queue growing |
| Survivorship manual overrides | `product_survivorship_log` where strategy=MANUAL | > 0 unexpected |
| Golden version churn | `product_golden_record` inserts per day | > 2× baseline |
| End-to-end latency | `_load_run.started_at` → `golden_record.published_at` | > 8h |

Dashboard: one panel per metric, 30-day rolling.

---

## 9. Backfill strategy

When rules change (new field mapping, new survivorship rule):

```
1. Snapshot current golden state:
     CREATE TABLE mdm.product_golden_record_v_YYYYMMDD AS
     SELECT * FROM mdm.product_golden_record;
2. Update rules (mapping / survivorship / priority)
3. Re-run map → match → survivorship → publish
   Do NOT truncate staging.
4. Compare new golden version to snapshot
5. If acceptable: keep new version
   If not: revert rules, re-run, or manual rollback
```

**Golden records are versioned.** Every re-run creates new versions. The old ones stay.

---

## 10. Build sequence

| Week | Deliverable |
|---|---|
| 1 | `008` applied. `ingest-loader` for FactSet. Load one real file. |
| 2 | `map-transformer`. `009` applied. First `product_incoming` rows. |
| 2 | `match-resolver`. First `entity_xref` rows. |
| 3 | `survivorship-engine`. `010` applied. First `survivorship_log` rows. |
| 3 | `golden-publisher`. First `product_golden_record` rows. |
| 4 | Add BBG. Repeat weeks 1–3 flow. |
| 4 | Add RDP. Repeat. |
| 5 | Observability + backfill tooling. |

**End of week 5:** three-source Product master producing golden records daily.

---

## 11. What "done" looks like

- [ ] `008`, `009`, `010` applied
- [ ] `ingest-loader` loads FactSet, BBG, RDP daily
- [ ] `map-transformer` produces `product_incoming` with < 1% reject rate
- [ ] `match-resolver` auto-merges > 90% of incoming
- [ ] `survivorship-engine` produces `survivorship_log` for every field
- [ ] `golden-publisher` produces golden record version N+1 daily
- [ ] Dashboard live with all 7 metrics
- [ ] Backfill procedure tested on one field change
- [ ] Runbook for "one vendor fails" scenario
```

---

## 12. Apply order

```bash
cd /Users/eganpj/GitHub/uisce/backend/db/manual_fixes

# 1. Staging schema + tables
psql "$CRIMS_URL" -1 -v ON_ERROR_STOP=1 -f 008_staging_product.sql

# 2. Seed sources + mappings (depends on staging existing for FK-free inserts)
psql "$CRIMS_URL" -1 -v ON_ERROR_STOP=1 -f 009_seed_product_sources.sql

# 3. Seed survivorship + priority
psql "$CRIMS_URL" -1 -v ON_ERROR_STOP=1 -f 010_seed_product_survivorship.sql

# 4. Verify
psql "$CRIMS_URL" -Atc "
SELECT 'staging tables: '||count(*) FROM information_schema.tables
WHERE table_schema='staging'
UNION ALL
SELECT 'sources: '||count(*) FROM mdm.source_system
WHERE source_cd IN ('FACTSET','BBG','RDP')
UNION ALL
SELECT 'field mappings: '||count(*) FROM mdm.product_field_mapping
UNION ALL
SELECT 'type mappings: '||count(*) FROM mdm.product_type_mapping
UNION ALL
SELECT 'survivorship rules: '||count(*) FROM mdm.survivorship_rule
WHERE entity_type='PRODUCT';
"
```

Expected:
```
staging tables: 6
sources: 3
field mappings: 44
type mappings: 20
survivorship rules: 12
