# Reporting Subsystem & Full-Text Search (FTS) Deployment Runbook

## 1. Overview & Scope
This runbook covers the pre-flight checks, schema prerequisites, Debezium CDC considerations, deployment steps, and rollback procedures for the Report Library Full-Text Search and related reporting migrations:

- `20260910_001_report_folders.up.sql` (Report Folders & Junction Table)
- `20260910_002_report_schedules.up.sql` (Report Schedules & Subscriptions)
- `20260912_001_report_library_fts.up.sql` (Full-Text Search tsvector generated column & GIN indexes)

---

## 2. Debezium CDC & Replica Identity Audit Findings

### 2.1 The PostgreSQL Generated Column & Replica Identity Constraint
In PostgreSQL, when logical replication or Debezium CDC captures changes:
- If a table has `REPLICA IDENTITY FULL`, PostgreSQL logs the entire old row in the WAL on `UPDATE` and `DELETE`.
- If a table containing **generated columns** is published with `publish_generated_columns = 'false'` (the PostgreSQL default), executing `UPDATE` or `DELETE` triggers a fatal database error:
  ```
  ERROR: cannot delete from table "report_templates": Replica identity must not contain unpublished generated columns
  ```
- **PostgreSQL Version Support**:
  - PostgreSQL $\le$ 17 does **not** support publishing generated columns.
  - PostgreSQL 18 introduced `pubgencols` on `pg_publication`, allowing `ALTER PUBLICATION ... SET (publish_generated_columns = true)`. However, most CDC drivers and downstream consumers do not yet handle generated columns in change events, and `pubgencols` defaults to `'n'` (none).
  - Therefore, setting `REPLICA IDENTITY DEFAULT` is the universal, safe posture for tables with generated columns.

### 2.2 Live Environment Findings (Alpha Host `100.84.50.65`)
1. **Engine Version**: `PostgreSQL 18.6 (Ubuntu 18.6-1.pgdg24.04+2)`
2. **Current `report_templates` Replica Identity**: `d` (`DEFAULT`).
   ```sql
   SELECT relname, relreplident, nspname FROM pg_class WHERE relname = 'report_templates';
   -- relname: report_templates | relreplident: d | nspname: public
   ```
3. **Publication Membership**:
   - `report_templates` is published **only** via `dbz_publication_alpha` (`puballtables = true`, `pubgencols = 'n'`).
   - It is **not** part of any targeted schema publications (`iam_security_publication`, `bo_cdc_publication`, etc.).
4. **Active Debezium Connector & Topic Audit**:
   - Kafka Connect (`http://100.84.50.65:8083/connectors`) currently runs only `orm-oms-connector`, capturing `orm.*` tables from the `crims` database.
   - Topics in Redpanda (`http://100.84.50.65:8082/topics`) confirm no active topics or consumers consume `report_templates` CDC events.
   - All consumers in the repo (`security-sync-worker`, `stream_loader`) exclusively consume `roles`, `user_roles`, `tenants`, or `orm.*` execution/order tables.
   - **Conclusion**: `REPLICA IDENTITY DEFAULT` on `public.report_templates` does **not** degrade any active downstream CDC consumer.

---

## 3. Production Pre-Flight Checklist

Before applying migration `20260912_001_report_library_fts.up.sql` in any target environment (staging/production):

### Step 1: Introspect Target Environment Replica Identity & Publications
Run the following read-only queries against the target environment using the authorized mTLS connection:

```sql
-- 1. Check current replica identity of report_templates ('d' = default, 'f' = full)
SELECT c.relname, c.relreplident, n.nspname 
FROM pg_class c 
JOIN pg_namespace n ON n.oid = c.relnamespace 
WHERE c.relname = 'report_templates';

-- 2. Check publication membership
SELECT pubname, schemaname, tablename 
FROM pg_publication_tables 
WHERE tablename = 'report_templates';

-- 3. Check if all-tables publications exist
SELECT pubname, puballtables, pubgencols 
FROM pg_publication;
```

### Step 2: Ensure `REPLICA IDENTITY DEFAULT`
If `relreplident` is `'f'` (`FULL`), you **MUST** alter the replica identity to `DEFAULT` before or during the deployment maintenance window:

```sql
ALTER TABLE public.report_templates REPLICA IDENTITY DEFAULT;
```

> [!WARNING]
> If `REPLICA IDENTITY FULL` is retained when `20260912_001_report_library_fts.up.sql` adds the `search_vector` generated column, subsequent `UPDATE` and `DELETE` queries on `report_templates` will fail with PostgreSQL error 42P01 / 55000 (`Replica identity must not contain unpublished generated columns`).

---

## 4. Migration Execution Sequence

Apply migrations in strict numerical order using the verified migration runner or direct idempotent psql execution:

1. `backend/db/migrations/20260910_001_report_folders.up.sql`
2. `backend/db/migrations/20260910_002_report_schedules.up.sql`
3. `backend/db/migrations/20260912_001_report_library_fts.up.sql`

### Verification Query Post-Migration
```sql
-- Verify generated column and backfill
SELECT count(*) AS total_rows, count(search_vector) AS populated_vectors 
FROM public.report_templates;

-- Verify GIN indexes exist and are valid
SELECT indexname, indexdef 
FROM pg_indexes 
WHERE tablename = 'report_templates' 
  AND indexname IN ('idx_report_templates_search_vector', 'idx_report_templates_trgm_name');
```

---

## 5. Rollback & Down-Migration Procedure

If a rollback of the Full-Text Search migration is required:

### Step 1: Run Down Migration
```sql
-- backend/db/migrations/20260912_001_report_library_fts.down.sql
DROP INDEX IF EXISTS public.idx_report_templates_trgm_name;
DROP INDEX IF EXISTS public.idx_report_templates_search_vector;

ALTER TABLE public.report_templates
    DROP COLUMN IF EXISTS search_vector;
```

### Step 2: Replica Identity Restoration Guard
> [!IMPORTANT]
> Dropping the `search_vector` generated column restores technical compatibility with `REPLICA IDENTITY FULL`.
> However, rolling back the migration does **not** automatically revert `REPLICA IDENTITY`. If the target environment previously required `REPLICA IDENTITY FULL` for an audited downstream consumer, you must explicitly restore it:
> ```sql
> ALTER TABLE public.report_templates REPLICA IDENTITY FULL;
> ```
> Verify the final replica identity state:
> ```sql
> SELECT relname, relreplident FROM pg_class WHERE relname = 'report_templates';
> ```
