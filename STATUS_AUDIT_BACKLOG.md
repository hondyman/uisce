# STATUS_AUDIT_BACKLOG.md — Open Findings

## Finding: ColumnMapper-Delete-On-Rename

**Name:** `ColumnMapper-Delete-On-Rename`
**Severity:** Medium (silent data-loss footgun)
**Found:** 2026-09-04
**Status:** Open

### Description

`ColumnMapper.Transform` in `backend/internal/datapipeline/transforms.go` deletes source fields when mapping to a different name. Given `Mappings: {"out_name": "node_name"}`:

1. `transformed[targetKey] = val` — copies value to `out_name`
2. `delete(transformed, srcKey)` — **deletes** `node_name`

Any downstream node that expects both `out_name` AND `node_name` receives only `out_name`. The data loss is silent — no error, no warning — because the value was successfully mapped to its target field.

### Production Impact

All pipeline authors configuring `column_mapper` nodes with rename-style mappings (e.g. `{"external_id": "id"}`, `{"out_name": "node_name"}`) will silently lose the source field. Any loader or subsequent transform that reads the original field name gets an empty value.

### Workaround in Tests

The diamond persistence test (`diamond_persistence_test.go`) uses self-referential mappings to avoid this behavior: `Mappings: {"node_name": "node_name", "value": "value"}`. This copies fields without deleting originals.

### Fix Direction

In `ColumnMapper.Transform`, remove the `delete(transformed, srcKey)` call. The default-copy behavior (copy all existing fields, then apply renames on top) means renamed fields are effectively renamed without losing the original — which is the behavior pipeline authors would expect. If the intent is to actually move (not copy) a field, that should be an explicit opt-in option.

### References

- `backend/internal/datapipeline/transforms.go:41-46` — the delete-on-rename logic
- `backend/internal/datapipeline/diamond_persistence_test.go` — current test workaround

---

## Finding: Unauthenticated Pipeline Run Execution (SEV-HIGH)

**Name:** `Unauthenticated-Pipeline-Run-Execution`
**Severity:** HIGH (upgraded from prior audit Item A)
**Found:** 2026-09-04
**Status:** Open

### Description

`POST /api/v1/data-pipelines/{id}/run` (both sync and durable) executes without any `Authorization` header, returning 202 and triggering Temporal workflow dispatch. The `getTenantID` handler helper falls back to the Gold Copy system tenant (`00000000-0000-0000-0000-000000000001`) when no valid JWT is present.

Evidence (2026-09-04 dev DB alpha):
```sql
SELECT tenant_id FROM public.data_pipeline_runs
WHERE id = '83c0605c-3988-45fe-a764-bf181c976f49';
-- Result: 00000000-0000-0000-0000-000000000001 (Gold Copy system tenant)
```

This means the entire pipeline execution surface is an unauthenticated API endpoint. Any caller can trigger pipeline runs as the Gold Copy system tenant.

### Fix Direction

Apply `claimTenantIDFromRequest` 401 hardening to `RunPipeline` and `GetRunStatus` handlers, matching the protection already applied to `GET /api/v1/data-pipelines?compact=true`. The pattern exists and is proven.

---

## Finding: GetGoldCopyInfo — Invalid Temporal Activity Signature

**Name:** `GetGoldCopyInfo-Invalid-Activity-Signature`
**Severity:** Medium (dead code, latent crash)
**Found:** 2026-09-04
**Status:** Open

### Description

`GetGoldCopyInfo` in `internal/temporal/activities/tenant_instance_activities.go:365` returns four values:

```go
func (a *TenantProvisioningActivities) GetGoldCopyInfo(ctx context.Context) (string, string, string, error) {
    return tenantID, instanceID, database, err
}
```

Temporal activity functions may return at most two values (result, error). This function was registered as an activity in `cmd/worker/main.go:355` and `internal/temporal/worker.go:126`. It cannot have ever executed successfully — any invocation would crash the worker goroutine.

The function is not called by any workflow in the codebase (confirmed by grep). It is dead code that would cause a panic if reached.

### Fix Direction

Either convert to a proper 2-return signature (combine the three strings into a struct), or remove the registration entirely if no caller exists. Currently commented out in both registration sites.

---

## Finding: Non-Idempotent Migration Causes Server Crash on Deploy

**Name:** `Non-Idempotent-Migration-API-Crash`
**Severity:** HIGH (deployment hazard)
**Found:** 2026-09-04
**Status:** Open

### Description

The migration `20260909_001_datapipeline_run_persistence.up.sql` created tables and indexes for `data_pipeline_runs` and `data_pipeline_step_telemetry`. The index `CREATE` statements lack `IF NOT EXISTS`, making them non-idempotent.

When a future deploy re-runs the migration (e.g., after a crash or after manual table creation), the server process calls `log.Fatal("Server cannot start with failed migrations")` and terminates. A single non-idempotent migration can take down the entire API on every subsequent deploy until the conflict is resolved.

Note: this finding was created after a live incident where the migration was applied manually (for testing) and then the fabricated migration_log entry was deleted, creating a cycle of schema state with falsified provenance. The immediate incident is resolved (migration made idempotent, entry recorded with correct SHA), but the deployment-hazard pattern remains.

### Fix Direction

Audit all migrations in `backend/db/migrations/` for non-idempotent `CREATE INDEX`/`CREATE TABLE` statements. Apply `IF NOT EXISTS` / `IF NOT EXISTS ... ON CONFLICT DO NOTHING` pattern consistently. Add a migration-review checklist item.

---

## Finding: Migration Runner Silently No-Ops Based on Launch CWD (SEV-HIGH)

**Name:** `Migration-Runner-CWD-Dependent`
**Severity:** HIGH (deployment hazard)
**Found:** 2026-09-05
**Status:** Open

### Description

`ApplyMigrations` in `backend/internal/migrations/runner.go` resolves `db/migrations` relative to the process current working directory. When the binary is started from a directory other than `backend/`, both `db/migrations` and `../db/migrations` fail to resolve, and the runner silently skips all migrations — no warning, no error, no log entry. The server starts, functional, with zero migrations applied.

Evidence (2026-09-05):
- Server started from `backend/`: runner logs `✅ Applied migration: 20260909_001...`
- Same binary, started from repo root or `/tmp`: runner logs nothing, skips silently

The runner's own diagnostic (`⚠️ Migrations directory ... not found; skipping`) only fires when `os.Stat` fails, which does not happen when the directory doesn't exist — `os.Stat` returns `os.IsNotExist` on both `db/migrations` and `../db/migrations` from wrong cwd.

### Fix Direction

Add `MIGRATIONS_DIR` env var as an explicit absolute path override. Bake absolute path at build time via `ldflags -X` or resolve from `os.Executable()`. The runner must either find the directory or fail with a fatal error, not silently skip. Consider: if migrations are required for correct operation, a missing migrations directory should be fatal at startup.

---

## Finding: ListPipelines Endpoint Unauthenticated (SEV-MEDIUM)

**Name:** `Unauthenticated-Pipeline-List`
**Severity:** MEDIUM
**Found:** 2026-09-05
**Status:** Closed (fixed in commit de2a60f7e)

### Description

`GET /api/v1/data-pipelines` (without `compact=true`) used `getTenantID` Gold Copy fallback, returning pipeline definitions to any unauthenticated caller. The `compact=true` variant was already hardened. Fix: `ListPipelines` now uses `claimTenantIDFromRequest` like the compact path.

---

## Finding: Evidence Table Dropped During Provenance Test

**Name:** `Evidence-Table-Dropped`
**Severity:** Low (operational hygiene)
**Found:** 2026-09-05
**Status:** Open

### Description

During the migration provenance test cycle, `data_pipeline_runs` and `data_pipeline_step_telemetry` were dropped to test clean re-application. This destroyed the durable-run evidence rows including `83c0605c` (the Gold Copy tenant execution proof). Four silent-deletion events in this engagement: stashed WIP, `placeholder-lint.test.ts`, commits reset away, and this table drop.

Pattern: even "just test data" deletions should name what they destroy in the commit message or a tracking artifact.

### Fix Direction

Establish a rule: table/data deletions include the object name in the operation description. For test evidence, snapshot rows to a file before destructive tests.
