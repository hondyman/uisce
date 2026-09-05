# STATUS_AUDIT_BACKLOG.md — Open Findings

## Finding: APICallerTransformer — Phase 3 Stub Never Executed

**Name:** `APICallerTransformer-Unimplemented`
**Severity:** HIGH (live footgun)
**Found:** 2026-09-05
**Status:** Open

### Description

`APICallerTransformer` in `backend/internal/datapipeline/transforms.go` is a Phase 3 transformer that stamps `{"verified": true}` without making any HTTP call. It is registered in the transform palette and selectable by pipeline authors. A transformer that declares it calls an external API but doesn't is a silent correctness failure — pipeline authors trust the output of a "verified" step that made no external request.

### Production Impact

Pipelines using `api_caller` in a production context will produce outputs claiming verification was performed when no API was contacted. Downstream decisions made on that output are based on fabricated data.

### Fix Direction

Bind to the server-side API Studio endpoint registry (no arbitrary URLs — SSRF prevention). Support service-held auth tokens (OAuth2 client credentials, API keys). Cap responses at 1MB. Propagate real HTTP errors through the pipeline failure path.

### References

- `backend/internal/datapipeline/transforms.go` — `APICallerTransformer` stub
- API Studio endpoint registry (platform component, not yet wired)

---

## Finding: Outbox Publish Outside BO Write Transaction

**Name:** `Outbox-Event-Transactional-Gap`
**Severity:** HIGH (silent eventual-consistency bug)
**Found:** 2026-09-05
**Status:** Open

### Description

BO event-driven pipeline triggers use an outbox pattern where `PublishPipelineTrigger` opens its own transaction, independent of the BO write transaction. A BO write can roll back *after* the trigger event commits, causing a pipeline to fire for data that doesn't exist. This is an eventual-consistency bug at the exact seam event-driven pipelines are built on.

### Production Impact

Any pipeline wired to fire on BO create/update will fire spuriously when the originating write fails after the trigger event commits. Operators will see pipeline runs with no corresponding BO change.

### Fix Direction

Refactor `PublishPipelineTrigger` to accept the BO write's `*sqlx.Tx` and participate in the same transaction. One interface change — the outbox row and the BO write commit or roll back atomically. Pipelines fire only when the event that triggered them is durable.

---

## Finding: ColumnMapper Delete-On-Rename (Silent Data Loss)

**Name:** `ColumnMapper-Delete-On-Rename`
**Severity:** Medium (silent data-loss footgun)
**Found:** 2026-09-04
**Status:** Open

### Description

`ColumnMapper.Transform` in `backend/internal/datapipeline/transforms.go` deletes source fields when mapping to a different name. Given `Mappings: {"out_name": "node_name"}`:

1. `transformed[targetKey] = val` — copies value to `out_name`
2. `delete(transformed, srcKey)` — **deletes** `node_name`

Any downstream node that expects both `out_name` AND `node_name` receives only `out_name`. The data loss is silent.

### Production Impact

All pipeline authors configuring `column_mapper` nodes with rename-style mappings will silently lose the source field. Any loader or subsequent transform reading the original field name gets an empty value.

### Workaround in Tests

`diamond_persistence_test.go` uses self-referential mappings: `Mappings: {"node_name": "node_name", "value": "value"}` — copies without deleting.

### Fix Direction

Remove `delete(transformed, srcKey)`. Default-copy behavior gives rename-without-loss semantics. If move semantics are needed, make it an explicit opt-in flag.

### References

- `backend/internal/datapipeline/transforms.go:41-46`
- `backend/internal/datapipeline/diamond_persistence_test.go`

---

## Finding: Trigger Surface — Create-Only, No Visibility or Lifecycle

**Name:** `TriggerSurface-CreateOnly`
**Severity:** MEDIUM (event-driven is half-built)
**Found:** 2026-09-05
**Status:** Open

### Description

The trigger-binding UI (TriggerAuthoringPage) can create bindings but cannot edit, deactivate, or view firing history. Event-driven pipelines stop being a demo only when operators can see which events fired which runs, and control when they fire.

### Fix Direction

Add: trigger list view with last-fired timestamp and run link; activate/deactivate toggle; run-history page per trigger.

---

## Finding: Event Coalescing / Debouncing Missing — Thundering Herd at Production Scale

**Name:** `Event-Coalescing-Missing`
**Severity:** MEDIUM (scalability hazard)
**Found:** 2026-09-05
**Status:** Open

### Description

The sync trigger path fires a pipeline per BO write with no debouncing. A bulk import of 10,000 trade orders triggers 10,000 pipeline runs. At demo scale this is fine; at production scale it is a thundering-herd problem.

### Fix Direction

Add windowed event coalescing ("collect events for 30s, run once with the batch"). Add watermarks on sources so triggered runs are incremental.

---

## Finding: Dead-Letter Quarantine Unimplemented

**Name:** `DeadLetter-Quarantine-Unimplemented`
**Severity:** MEDIUM (reliability loop incomplete)
**Found:** 2026-09-05
**Status:** Open

### Description

`error_policy: dead_letter` is declared in pipeline schemas but unimplemented. Failed records from event-triggered runs disappear into logs with no recovery path.

### Fix Direction

Failed records land in a quarantine table with the triggering event payload attached. Replay re-executes against the original business event.

---

## Finding: WorkflowCallerTransformer — Synchronous Execution Blocks Pipeline Run

**Name:** `WorkflowCallerTransformer-SyncBlocking`
**Severity:** MEDIUM (pipeline/workflow coupling)
**Found:** 2026-09-05
**Status:** Open

### Description

`WorkflowCallerTransformer` runs workflows synchronously inside pipeline execution. A slow workflow blocks the entire pipeline run. Pipelines and workflows are coupled by execution time.

### Fix Direction

Make always-async with a correlation-ID handshake: pipeline emits run_id, workflow completion event lands back on the bus. Add timeout policy for the async wait.

---

## Finding: Lineage Auto-Write Missing

**Name:** `Lineage-AutoWrite-Missing`
**Severity:** MEDIUM (flagship differentiator unbuilt)
**Found:** 2026-09-05
**Status:** Open

### Description

Run persistence now produces `run→step→written-catalog-node` rows, but they are not wired to the semantic layer or lineage viewer. No ETL tool can do this because none own the catalog. This is the natural flagship differentiator.

### Fix Direction

Stitch persisted run→step→catalog-node rows to the semantic layer and lineage viewer. Every pipeline run updates the graph.

---

## Finding: Pipeline-as-Code Missing

**Name:** `PipelineAsCode-Missing`
**Severity:** LOW (governance gap)
**Found:** 2026-09-05
**Status:** Open

### Description

Pipeline definitions exist only as UI state. No export/import in the bundle format, no CI dry-run validation, no GitOps review path.

### Fix Direction

Export/import pipeline definitions in the bundle format. CI dry-run validation against the real engine. GitOps review path for pipeline changes.

---

## Finding: DB Runtime Connection Uses Postgres Superuser With BYPASSRLS

**Name:** `DBRuntime-Superuser-BYPASSRLS`
**Severity:** SEV-HIGH (multi-tenant isolation not database-guaranteed)
**Found:** 2026-09-05
**Status:** Open

### Description

The application DB connection runs as the `postgres` superuser with `BYPASSRLS`. Tenant isolation is application-convention, not database-enforced. Every integration added (per-tenant trigger authoring, per-tenant API surfaces) increases multi-tenant exposure.

### Fix Direction

Demote the runtime connection to a role with only required permissions. Row-level security policies must enforce tenant isolation at the database layer.

---

## Finding: Migration Runner Silently No-Ops Based on Launch CWD (SEV-HIGH)

**Name:** `Migration-Runner-CWD-Dependent`
**Severity:** HIGH (deployment hazard)
**Found:** 2026-09-05
**Status:** Open (workaround: `MIGRATIONS_DIR` env var)

### Description

`ApplyMigrations` resolves `db/migrations` relative to process cwd. Starting from wrong directory skips all migrations silently. Workaround exists: `MIGRATIONS_DIR` env var override.

### Fix Direction

Resolve from `os.Executable()` or bake absolute path at build time. Missing migrations directory should be fatal at startup.

### References

- `backend/internal/migrations/runner.go`
- `MIGRATIONS_DIR` env var — current workaround

---

## Finding: GetGoldCopyInfo — Invalid Temporal Activity Signature (Dead Code)

**Name:** `GetGoldCopyInfo-Invalid-Activity-Signature`
**Severity:** Low (dead code, commented out)
**Found:** 2026-09-04
**Status:** Open

### Description

`GetGoldCopyInfo` returns four values. Temporal activities may return at most two. Registered as activity in `cmd/worker/main.go:355` and `internal/temporal/worker.go:126` — any invocation would crash. Not called by any workflow. Currently commented out.

### Fix Direction

Remove the function or convert to a proper 2-return signature.

---

## Finding: Evidence Table Dropped During Provenance Test

**Name:** `Evidence-Table-Dropped`
**Severity:** Low (operational hygiene)
**Found:** 2026-09-05
**Status:** Open

### Description

During the migration provenance test cycle, `data_pipeline_runs` and `data_pipeline_step_telemetry` were dropped to test clean re-application, destroying evidence rows. Four silent-deletion events in this engagement.

### Fix Direction

Establish a rule: table/data deletions include the object name in the operation description. For test evidence, snapshot rows to a file before destructive tests.

---

## Finding: Non-Idempotent Migration Causes Server Crash on Deploy

**Name:** `Non-Idempotent-Migration-API-Crash`
**Severity:** HIGH (deployment hazard)
**Found:** 2026-09-04
**Status:** Closed (migration made idempotent in 20260909_001)

### Description

`20260909_001_datapipeline_run_persistence.up.sql` lacked `IF NOT EXISTS` on index creation. Re-running would call `log.Fatal`. Migration is now idempotent.

### Fix Direction

Audit all migrations for non-idempotent `CREATE INDEX`/`CREATE TABLE`. Add a migration-review checklist item.

---

## Finding: Unauthenticated Pipeline Run Execution (SEV-HIGH)

**Name:** `Unauthenticated-Pipeline-Run-Execution`
**Severity:** HIGH
**Found:** 2026-09-04
**Status:** Closed (fixed in commit de2a60f7e)

### Description

`POST /api/v1/data-pipelines/{id}/run` and `GET /api/v1/data-pipelines/runs/{id}` fell back to Gold Copy tenant without JWT. Hardened to 401 via `claimTenantIDFromRequest`. Tenant ownership check added to `GetRunStatus`.

---

## Finding: ListPipelines Endpoint Unauthenticated (SEV-MEDIUM)

**Name:** `Unauthenticated-Pipeline-List`
**Severity:** MEDIUM
**Found:** 2026-09-05
**Status:** Closed (fixed in commit 29d9e48bd)

### Description

`GET /api/v1/data-pipelines` used `getTenantID` Gold Copy fallback. Now uses `claimTenantIDFromRequest` like the `compact=true` path.

---

## Finding: Unauthenticated SSE Telemetry Endpoint

**Name:** `Unauthenticated-SSE-Telemetry`
**Severity:** MEDIUM
**Found:** 2026-09-05
**Status:** Closed (verified in session)

### Description

`GET /api/v1/data-pipelines/runs/{id}/telemetry` with no token and no JWT returned 401 after the stream token fix (token-fix commit). Stream token auth is separate from JWT auth and is functioning correctly.
