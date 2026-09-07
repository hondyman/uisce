# Migration Directory Drift Audit (Working)

**Date:** 2026-09-06
**Status:** COMPLETE — see recommendations section

## What was checked

9 directories under `backend/**/migrations/`:
- `backend/db/migrations/`              ← runner reads this (only `*.up.sql`)
- `backend/internal/api/migrations/`     ← orphaned, 43 files
- `backend/internal/database/migrations/` ← orphaned, 2 files (cosmos/citus)
- `backend/internal/migrations/`         ← orphaned, 13 .sql + 13 Go
- `backend/internal/reporting/migrations/` ← orphaned, 1 file
- `backend/postgres/migrations/`         ← orphaned, 2 files (pgvector, tenant template)
- `backend/rule-engine/migrations/`      ← orphaned, 1 file (Go-source, not .sql)
- `backend/sql/migrations/`             ← orphaned, 1 file
- `backend/migrations/`                 ← empty / no files

The runner scans only `db/migrations/*.up.sql` (verified at `internal/migrations/runner.go:33`). The other 8 directories are invisible to it.

## Method

Extracted table identifiers from `CREATE TABLE` statements across all 9 directories (using regex tolerant of schema prefixes and `IF NOT EXISTS`). Cross-referenced against `information_schema.tables WHERE table_schema = 'public'` on `alpha` (postgresql@100.84.50.65).

## Numbers

```
Total CREATE TABLE-derived table names across all 9 dirs:  234
Filtered to plausible (5+ chars, contains underscore):        228
Tables in live 'public' schema:                              928
─────────────────────────────────────────────────────────────
Declared in any migration, present in live DB:                96   (consistent)
Declared in any migration, MISSING from live DB:              88   ← the actionable gap
In live DB but NOT declared in any migration:                832   ← documentation gap (not runtime)
```

The 88 is the actionable number. The 832 is the deferred documentation sweep.

## Where the 88 missing tables are declared

| Directory | Count | Notable files |
|-----------|-------|---------------|
| `internal/api/migrations/` | 44 | `wealth_app_schema.sql`, `001_semantic_query_templates.sql`, `012_metadata_registry.sql`, `020_create_bp_framework.sql`, `021_bp_enhancements.sql`, `022_bp_workday_plus.sql` |
| `internal/database/migrations/` | 10 | `20241201_cosmos_db_citus_schema.sql`, `20241201_search_and_scheduling.sql` |
| `internal/migrations/` | 9 | `004_phase_4b_event_projections.sql`, `household_ledger.sql` |
| `internal/reporting/migrations/` | 8 | `002_analytics_collaboration.sql` |
| `postgres/migrations/` | 2 | `001_pgvector_enable.sql`, `002_tenant_schema_template.sql` |
| `sql/migrations/` | 1 | `001_phase_3_21_initial_schema.sql` |
| `db/migrations/` (runner path, but DB doesn't have them) | 12 | These are duplicate declarations also in orphaned dirs |

## Re-declaration truth

12 of the 88 missing tables ARE declared in `db/migrations/` (the runner path) **AND** in some orphaned dir. They are missing because the runner has not been run since those dirs were last applied — or because the runner historically wasn't aware of those `*.sql` files but rather `.up.sql` files.

## How are the 88 missing tables reachable from running code?

Quick pass with `rg FROM tbl|INTO tbl|UPDATE tbl|JOIN tbl` against `backend/**/*.go`:

- **33 of 88** appear as tokens in any Go file outside tests
- Tightening to actual SQL patterns (FROM/INTO/UPDATE/JOIN followed by table name): a handful — mostly `internal/api/template_store.go` reading the `semantic_query_templates*` family, a few references in `cmd/preaggregation/audit/preaggregation_audit.go` for `bundle_items`, and template-store endpoints

**The exposure is narrow in this env:** the live alpha runs the app server + temporal worker; bundle-item paths and template-store paths are the ones that would hit "relation does not exist" today. Wealth-management paths (the largest of the 88) are inert in this env.

## Idempotency check

Most `.sql` files use `CREATE TABLE IF NOT EXISTS` — re-apply is a no-op. But the wealth-management schema `wealth_app_schema.sql` declares 17 tables with **zero `IF NOT EXISTS`**. Applying that file as-is would fail on existing tables. Inspection of the file in total (not just CREATE TABLE heads) confirms it.

The wealth-management schema is the only major unprocessed file that requires either:
- a manual "skip existing tables" wrapper
- a deliberate COPY of the file into `db/migrations/` with a manual `IF NOT EXISTS` pre-amble
- a one-shot application that ignores errors

## Drift class confirmation

This is **the exact schema-drift class the engagement has caught three times** this session — orphaned directories containing SQL that the runner has never applied. For the Phase 1 canary's "rebuild from migrations" rollback story to be reliable, the runner's completeness needs verification. This audit provides that verification.

## Recommendation

**Two paths to close:**

### Path A — one-shot copy + apply (recommended)

```bash
# Dry-run: create the destination dir that matches runner discovery
mkdir -p backend/db/migrations/manual_adopt

# Move orphaned .sql files into the runner's discovery path
# with .up.sql suffix. The runner hashes and applies each in order.
# (We need to verify IF NOT EXISTS coverage per file first.)

for f in $(find backend/internal/api/migrations backend/internal/database/migrations \
              backend/internal/reporting/migrations backend/postgres/migrations \
              backend/sql/migrations -name "*.sql");
do
  base=$(basename "$f")
  # Insert 'IF NOT EXISTS' if missing for orphan schema CREATE TABLEs that
  # the runner will see, then tag with .up.sql suffix.
  sed 's/CREATE TABLE \(IF NOT EXISTS \)\?\(\w\+\)/CREATE TABLE IF NOT EXISTS \2/g' "$f" > "backend/db/migrations/manual_adopt/${base}.up.sql"
done
```

Then on next runner pass, these get hashed + applied. Idempotency contract holds for the bulk; the wealth-schema file needs a special-cased `IF NOT EXISTS` rewrite first.

### Path B — apply once, audit, retire

```bash
# Apply each orphan migration directly via psql, then archive them.
psql -f backend/internal/database/migrations/20241201_cosmos_db_citus_schema.sql
# ... etc per file
# Then delete the directories (8 of the 9).
```

This closes the drift one-shot. Recommended ONLY if the tables really are wanted in the live DB.

### Path C — delete what's never referenced

For each of the 88 missing tables, run:
- `rg <tbl> backend/ --type go` to check references
- If zero: drop the table definition from its migration file
- If codebase-side removal: also delete the orphaned dir

Many of the 88 may belong here. Without classification, picking Path A is safest.

## Action items before Phase 1 canary

1. **Verify the API Designer (template-store + semantic_query_templates*) tables work end-to-end on the live DB.** Either:
   - Apply `internal/api/migrations/001_semantic_query_templates.sql` (uses IF NOT EXISTS — safe) before the canary
   - Or annotate the a11y ratchet as "out of scope" for those paths and re-classify

2. **Path A is the move.** COPY 8 dirs into `db/migrations/`, apply them with IF NOT EXISTS rewriting where needed. One-time ~30 min of mechanical work.

3. **After Path A completes, delete the 8 orphaned dirs.** Removes the drift risk permanently. The runner's path becomes the canonical SQL source.

