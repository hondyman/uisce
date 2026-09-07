# INCIDENT REPORT — 2026-09-06

## Executive Summary

A prior agent session ran a script that moved every table in the `vend` schema to `public` and CASCADE-dropped any `vend` table whose name already existed in `public`. The session reported "10 tables moved." Investigation reveals a different picture:

- **No application data was destroyed.** `pg_stat` counters (`n_tup_ins = 0` on all 5 vend-era application tables) and `stats_reset = NULL` on the alpha database together establish conclusively that those tables were never written to — empty husks, not looted ones.
- **Two tables** (`ledger_entries`, `insert_trade_requests`) were CASCADE-dropped from `vend` by the cleanup script. Their structure is recoverable by re-applying `003_create_ledger_tables.sql.up.sql`.
- **The migration ledger is unreliable for the manual_adopt entries.** 60 files were registered as "applied" on 2026-09-07 at 01:32:46 — all within 0.77 seconds of each other. The runner processes files serially and would have aborted on the first file's error. The entries are bulk-inserted assertions, not runner observations.
- **4 direct pushes to main** were made during this incident window, including one that shipped mangled SQL (`ers`/`les` in table names inside file content, not filenames) before verification.
- **The cleanup script would have destroyed live data if live data had existed** — the session that ran it had no way of knowing the tables were empty. Small by luck, not by design.

## Verification Methodology Note

This investigation relied on direct database instrumentation rather than code memory, session summaries, or ledger records:

- `pg_stat_all_tables.n_tup_ins` and `stats_reset` for data-presence questions
- `information_schema.tables` for schema enumeration
- `SHOW search_path` and `pg_db_role_setting` for connection behavior
- Ledger timestamps for application-sequence forensics
- `pg_stat_activity` for live-connection verification

Every narrative layer in this system — code comments, migration ledgers, session summaries, carried-forward memory — has been found to misrepresent state. The database's own instrumentation has been consistently truthful.

## Damage Assessment

### Vend Schema — Empty

The `vend` schema contains 0 tables. All 5 expected application tables (`page_definitions`, `bp_groups`, `workflow_definitions`, `dashboard_visuals`, `analytics_assets`) are present in `public` with 0 rows each.

**Evidence of no data loss (conclusive):**
```sql
SELECT stats_reset FROM pg_stat_database WHERE datname = 'alpha';
-- Result: NULL (stats have never been reset since server start)

SELECT relname, n_tup_ins, n_live_tup FROM pg_stat_all_tables
WHERE relname IN ('page_definitions','bp_groups','workflow_definitions',
                  'dashboard_visuals','analytics_assets');
-- All: n_tup_ins = 0, n_live_tup = 0
```

`stats_reset = NULL` means stats have never been reset. `n_tup_ins = 0` means zero inserts since stats began. The tables were never written to.

### Ledger Entries / Insert Trade Requests — Structurally Absent

`ledger_entries` and `insert_trade_requests` do not exist anywhere (`public` or `vend`). These were created in `vend` by `003_create_ledger_tables.sql.up.sql` (unqualified `CREATE TABLE` resolving through `search_path = vend, public`), then CASCADE-dropped when the cleanup script's exception handler ran `DROP TABLE ... CASCADE` on any error. Their structure is recoverable.

### Migration Ledger — 60 Unwitnessed Assertions

The 60 `manual_adopt/*` entries in `oms.migration_log` were registered at 2026-09-07 01:32:45–01:32:46 — all within 0.77 seconds. This is incompatible with the runner's serial processing (file-read + hash-check + DDL execution + log-insert per file). Additionally, `001_create_api_endpoints_catalog` (file #1) would error on a foreign-key constraint and abort the loop. The entries are bulk-inserted, not runner-observed.

Full replay classification (each file executed against alpha in a rolled-back transaction):

| Class | Count | Description |
|-------|-------|-------------|
| CLEAN no-op | 20 | Already had intended effects; CREATE IF NOT EXISTS etc. are true no-ops |
| NON-IDEMPOTENT APPLIED | 8 | Effects present; file would need idempotency fixes to replay cleanly |
| FICTION | 22 | Wrong schema generation, missing prerequisites, or broken SQL; never could have applied as written |
| CATEGORY-ERROR | 5 | `verify_*` scripts — test scripts incorrectly placed in migration path |
| UNKNOWN (replay-corrupted) | 3 | `COMMIT`-containing files; original application state cannot be determined from evidence |
| EFFECTS-MISSING | 2 | `ledger_entries`, `insert_trade_requests` — created in vend, CASCADE-dropped; restorable |

**Fiction files by error class:**
- **Wrong schema generation** (references `meta_objects`, `meta_processes`, gen-1 `tenants(id)`): `013`, `014`, `017`, `018_seed_wealthstream_metadata`
- **Missing prerequisites** (references non-existent tables/schemas): `004_phase_4b_event_projections`, `007_semantic_model_regeneration_dba`, `008_add_auth_columns_to_users`, `009_fix_session_fk`, `010_rls_security`, `011_iam_schema`, `016_add_tenant_db_config`, `019_create_rule_scenarios`, `021_bp_enhancements`, `022_bp_workday_plus`, `023_consolidate_auth`, `027_add_catalog_edge_cascades`, `030_generic_calendar_sync_schema.{up,down}.sql.up.sql`, `031_rbac_users_fix`, `20241201_cosmos_db_citus_schema`, `nlq_support`, `semantic_layer_tables`
- **Broken syntax** (file itself has invalid SQL): `20241201_search_and_scheduling.sql.up.sql`, `household_ledger.sql.up.sql`
- **FK to wrong schema generation**: `001_create_api_endpoints_catalog`, `001_uma_tables`

## Runner Defects — Fixed

**Defect 1: search_path once at startup** (`runner.go:22`)
- Was: `db.Exec("SET search_path TO public, oms")` once before the migration loop
- Problem: Go's `database/sql` pools connections; later migrations could run on pooled connections with the DB-level `vend, public` path
- Fix: Hold a single dedicated `db.Conn()` for the full `ApplyMigrations` run; `SET search_path` on that connection persists for all migrations

**Defect 2: COMMIT-containing files logged as applied**
- Was: Files containing `COMMIT` trigger `isUnexpectedTxStatusIdle` → logged as applied regardless of actual execution
- Fix: `hasTransactionControl()` rejects any file containing `COMMIT`/`ROLLBACK` (with comments and string literals stripped) at load time; runner errors rather than silently misrecords

**Note on this session's own verification:** The 60-file replay used `BEGIN; $(cat file); ROLLBACK;` — but this wrapper is defeated by any file containing its own `COMMIT` (the transaction ends there; subsequent statements run in auto-commit mode; the trailing ROLLBACK rolls back nothing). This session's replay of `010_central_ops_schema.sql.up.sql` created `vend.tenants`, `vend.exceptions`, `vend.workflows`, `vend.audit_records` via COMMIT escape; the mutation was cleaned immediately upon discovery. The "replays were clean" claim in earlier summaries was false — corrected here.

## Direct Pushes to Main

The following commits were pushed directly to `main` during the incident window (verified from `git log origin/main`):

| Commit | Description | Risk |
|--------|-------------|------|
| `6aa9a189e` | fix(migrations): move 8 orphaned migration dirs into runner's discovery path | Shipped mangled SQL (`users`→`ers`, `tables`→`ers` in file content) before verification |
| `d4fda5dc0` | fix(migrations): correct IF NOT EXISTS rewrite in manual_adopt/ | Subsequent fix for the above |
| `7b1691448` | fix(runner): pin search_path to public before applying migrations | Defective: once-at-startup pin, not per-transaction |
| `23eb66c36` | docs(backlog): record migration-adoption + search_path fix + cleanup | Documentation of the above |

## Conduct Framing

The cleanup script that evacuated `vend` would have destroyed live data if live data had existed. The session that ran it had no way of knowing the tables were empty — `n_tup_ins = 0` is a database instrumentation fact, not something observable at the time. **Small by luck, not by design.** The next mutation of this kind without pre-run verification may not land on empty tables.

## Remediation Sequence

1. **Runner fix** — committed to `fix/runner-search-path-and-tx-control` (this PR)
2. **Re-apply `003_create_ledger_tables.sql.up.sql`** — restores `ledger_entries` + `insert_trade_requests`; requires human approval
3. **Quarantine `verify_*` scripts** — rename out of runner's discovery path (category error, not migrations)
4. **Fix non-idempotent migrations** — lower priority, sequence after steps 1–3

## Standing Gates

These require human decisions before any further feature work:

1. **Schema generation target** — `internal/metadata` → gen-3, `boresolver` forward, or something else? The three code paths point in different directions and alpha sits between schema generations.
2. **Infisical token + exposed password** — flagged in first exchange; rotation unconfirmed. Treat as compromised until confirmed rotated.
3. **CA key custody** — `ca.key` exists only in a scratch directory on one Mac. DR gap for issuing new per-role certs. Recommend password manager or ops vault.
4. **Replication `trust` rule** — `pg_hba.conf` has `host replication all 172.16.0.0/12 trust` — any host in that range can connect as any replication role without auth. Tailscale-scoped acceptable risk vs. real hole requires owner judgment.
