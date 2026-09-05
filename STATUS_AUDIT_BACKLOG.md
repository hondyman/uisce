# STATUS_AUDIT_BACKLOG.md — Open Findings

**Severity scale:** `SEV-HIGH` = tenant isolation / production-correctness void · `HIGH` = silent failure in default path / deployment hazard · `MEDIUM` = capability gap, steady-state hazard · `LOW` = hygiene, governance, dead code already neutralized

---

## Finding: APICallerTransformer — Phase 3 Stub Never Executed

**Name:** `APICallerTransformer-Unimplemented`
**Severity:** HIGH
**Found:** 2026-09-05
**Status:** Closed

### Description

`APICallerTransformer` in `backend/internal/datapipeline/transforms.go` was a Phase 3 transformer that stamped `{"verified": true}` without making any HTTP call. It was registered in the transform palette and selectable by pipeline authors. A transformer that declared it calls an external API but doesn't is a silent correctness failure — pipeline authors trust the output of a "verified" step that made no external request.

### Production Impact

Pipelines using `api_caller` in a production context would produce outputs claiming verification was performed when no API was contacted. Downstream decisions made on that output were based on fabricated data.

### Fix Applied

`APICallerTransformer` fully implemented as a real HTTP transformer. A KYC-class silent-success scenario is no longer possible: an `api_caller` step either makes a real HTTP call to a registered API Studio endpoint or fails the pipeline visibly with the actual error surfaced (SSRF block, 4xx/5xx, timeout, 1MB cap exceeded).

Key implementation details:
- Bound to API Studio endpoint registry via `endpoint_id` — no `endpoint_url`, no arbitrary URLs, SSRF guard enforced by default
- Auth injection: bearer token, API key, Basic, OAuth2 client-credentials, or none — resolved from secrets vault per endpoint config
- 1MB response cap enforced on body read (not `Content-Length` header — chunked encoding makes it unreliable)
- Per-invocation telemetry INSERT into `api_telemetry` (endpoint, status, latency, error)
- OAuth2 in-memory token cache keyed by endpoint/secret_id, 30s pre-expiry refresh

### Test Results

`go test -count=1 -v ./internal/datapipeline/... -run TestAPICaller`
- T1  PASS — RealCall_Success
- T2  PASS — 5xx_ReturnsError
- T3a PASS — SSRF_PrivateIPBlocked
- T3b PASS — SSRF_LoopbackAllowedWithExemption
- T4  PASS — 1MB_Cap
- T5  PASS — MissingEndpointID_ReturnsError
- T6  PASS — EndpointNotFoundError
- T7  PASS — NoBaseURL_ReturnsError
- T8  PASS — BearerAuth
- T9  PASS — Timeout
- T10 PASS — PostMethod_WithRequestTemplate
- T11 PASS — MergeOutput
- T12 PASS — TargetField

### Commits

- `fb7389b9a` — feat(datapipeline): implement APICallerTransformer — real HTTP, SSRF guard, auth injection, 1MB cap

**Refs:** [#1](https://github.com/hondyman/uisce/issues/1)

---

## Finding: Outbox Publish Outside BO Write Transaction

**Name:** `Outbox-Event-Transactional-Gap`
**Severity:** HIGH
**Found:** 2026-09-05
**Status:** Closed (fixes landed across 4 commits)

### Description

BO event-driven pipeline triggers used an outbox pattern where `PublishPipelineTrigger` opened its own transaction, independent of the BO write transaction. A BO write could roll back *after* the trigger event committed, causing a pipeline to fire for data that doesn't exist.

### Fix Applied

`PublishPipelineTriggerTx` added as a new interface; `PublishPipelineTrigger` kept as legacy adapter with deprecation warning. Services layer (`CreateInstance`/`UpdateInstance`/`DeleteInstance`) now wraps writes in `BeginTxx`/`Commit` with `dispatchInstanceTriggerAsync` called in-tx. Atomicity proven by T1/T2/T3 tests (rollback wipes row, commit persists row, legacy path still works).

### Pending End-to-End Verification

Tests ran against `localhost:5432` with `search_path` routing to `test_outbox_schema.outbox` — tx semantics proven. One manual check against the deployed dev DB still recommended: bind an async trigger to a BO, write a record, force a write failure, confirm no pipeline dispatch.

### Commits

- `9b4b0f50e` — feat(validation): add phase-aware dispatch (DispatchWithPhase)
- `a3de512c1` — feat(datapipeline): widen outbox publisher to accept caller tx (adapter staged)
- `db0423dae` — fix(datapipeline): outbox events ride services BO write transaction (#2)
- `10f0b163f` — fix(validation): nil-tx regression in async legacy path + T1/T2/T3 PASS

**Refs:** [#2](https://github.com/hondyman/uisce/issues/2)

---

## Finding: ColumnMapper Delete-On-Rename (Silent Data Loss)

**Name:** `ColumnMapper-Delete-On-Rename`
**Severity:** MEDIUM
**Found:** 2026-09-04
**Status:** Open

### Description

`ColumnMapper.Transform` in `backend/internal/datapipeline/transforms.go` deletes source fields when mapping to a different name. Given `Mappings: {"out_name": "node_name"}`:

1. `transformed[targetKey] = val` — copies value to `out_name`
2. `delete(transformed, srcKey)` — **deletes** `node_name`

Any downstream node that expects both `out_name` AND `node_name` receives only `out_name`. The data loss is silent.

### Production Impact

All pipeline authors configuring `column_mapper` nodes with rename-style mappings will silently lose the source field. Any loader or subsequent transform reading the original field name gets an empty value. Severity is MEDIUM: annoying and discoverable in testing, recoverable by re-running with fixed mappings — not the same class as tenant isolation voids or transactional event gaps that corrupt *trust* in the system.

### Workaround in Tests

`diamond_persistence_test.go` uses self-referential mappings: `Mappings: {"node_name": "node_name", "value": "value"}` — copies without deleting.

### Fix Direction

Remove `delete(transformed, srcKey)`. Default-copy behavior gives rename-without-loss semantics. If move semantics are needed, make it an explicit opt-in flag.

### References

- `backend/internal/datapipeline/transforms.go:41-46`
- `backend/internal/datapipeline/diamond_persistence_test.go`

**Refs:** [#3](https://github.com/hondyman/uisce/issues/3)

---

## Finding: Trigger Surface — Create-Only, No Visibility or Lifecycle

**Name:** `TriggerSurface-CreateOnly`
**Severity:** MEDIUM
**Found:** 2026-09-05
**Status:** Open

### Description

The trigger-binding UI (TriggerAuthoringPage) can create bindings but cannot edit, deactivate, or view firing history. Event-driven pipelines stop being a demo only when operators can see which events fired which runs, and control when they fire.

### Fix Direction

Add: trigger list view with last-fired timestamp and run link; activate/deactivate toggle; run-history page per trigger.

**Refs:** [#4](https://github.com/hondyman/uisce/issues/4)

---

## Finding: Event Coalescing / Debouncing Missing — Thundering Herd at Production Scale

**Name:** `Event-Coalescing-Missing`
**Severity:** MEDIUM
**Found:** 2026-09-05
**Status:** Open

### Description

The sync trigger path fires a pipeline per BO write with no debouncing. A bulk import of 10,000 trade orders triggers 10,000 pipeline runs. At demo scale this is fine; at production scale it is a thundering-herd problem.

### Fix Direction

Add windowed event coalescing ("collect events for 30s, run once with the batch"). Add watermarks on sources so triggered runs are incremental.

---

## Finding: Dead-Letter Quarantine Unimplemented

**Name:** `DeadLetter-Quarantine-Unimplemented`
**Severity:** MEDIUM
**Found:** 2026-09-05
**Status:** Open

### Description

`error_policy: dead_letter` is declared in pipeline schemas but unimplemented. Failed records from event-triggered runs disappear into logs with no recovery path.

### Fix Direction

Failed records land in a quarantine table with the triggering event payload attached. Replay re-executes against the original business event.

---

## Finding: WorkflowCallerTransformer — Synchronous Execution Blocks Pipeline Run

**Name:** `WorkflowCallerTransformer-SyncBlocking`
**Severity:** MEDIUM
**Found:** 2026-09-05
**Status:** Open

### Description

`WorkflowCallerTransformer` runs workflows synchronously inside pipeline execution. A slow workflow blocks the entire pipeline run. Pipelines and workflows are coupled by execution time.

### Fix Direction

Make always-async with a correlation-ID handshake: pipeline emits run_id, workflow completion event lands back on the bus. Add timeout policy for the async wait.

---

## Finding: Lineage Auto-Write Missing

**Name:** `Lineage-AutoWrite-Missing`
**Severity:** MEDIUM
**Found:** 2026-09-05
**Status:** Open

### Description

Run persistence now produces `run→step→written-catalog-node` rows, but they are not wired to the semantic layer or lineage viewer. No ETL tool can do this because none own the catalog. This is the natural flagship differentiator.

### Fix Direction

Stitch persisted run→step→catalog-node rows to the semantic layer and lineage viewer. Every pipeline run updates the graph.

---

## Finding: Pipeline-as-Code Missing

**Name:** `PipelineAsCode-Missing`
**Severity:** LOW
**Found:** 2026-09-05
**Status:** Open

### Description

Pipeline definitions exist only as UI state. No export/import in the bundle format, no CI dry-run validation, no GitOps review path.

### Fix Direction

Export/import pipeline definitions in the bundle format. CI dry-run validation against the real engine. GitOps review path for pipeline changes.

---

## Finding: DB Runtime Connection Uses Postgres Superuser With BYPASSRLS

**Name:** `DBRuntime-Superuser-BYPASSRLS`
**Severity:** SEV-HIGH
**Found:** 2026-09-05
**Status:** Open

### Description

The application DB connection runs as the `postgres` superuser with `BYPASSRLS`. Tenant isolation is application-convention, not database-enforced. Every integration added (per-tenant trigger authoring, per-tenant API surfaces) increases multi-tenant exposure.

### Fix Direction

Demote the runtime connection to a role with only required permissions. Row-level security policies must enforce tenant isolation at the database layer.

---

## Finding: Migration Runner Silently No-Ops Based on Launch CWD

**Name:** `Migration-Runner-CWD-Dependent`
**Severity:** HIGH
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
**Severity:** LOW
**Found:** 2026-09-04
**Status:** Open

### Description

`GetGoldCopyInfo` returns four values. Temporal activities may return at most two. Registered as activity in `cmd/worker/main.go:355` and `internal/temporal/worker.go:126` — any invocation would crash. Not called by any workflow. Currently commented out.

### Fix Direction

Remove the function or convert to a proper 2-return signature.

---

## Finding: Evidence Table Dropped During Provenance Test

**Name:** `Evidence-Table-Dropped`
**Severity:** LOW
**Found:** 2026-09-05
**Status:** Open

### Description

During the migration provenance test cycle, `data_pipeline_runs` and `data_pipeline_step_telemetry` were dropped to test clean re-application, destroying evidence rows. Four silent-deletion events in this engagement.

### Fix Direction

Establish a rule: table/data deletions include the object name in the operation description. For test evidence, snapshot rows to a file before destructive tests.

---

## Finding: Non-Idempotent Migration Causes Server Crash on Deploy

**Name:** `Non-Idempotent-Migration-API-Crash`
**Severity:** HIGH
**Found:** 2026-09-04
**Status:** Closed (migration made idempotent in 20260909_001)

### Description

`20260909_001_datapipeline_run_persistence.up.sql` lacked `IF NOT EXISTS` on index creation. Re-running would call `log.Fatal`. Migration is now idempotent.

### Fix Direction

Audit all migrations for non-idempotent `CREATE INDEX`/`CREATE TABLE`. Add a migration-review checklist item.

---

## Finding: Unauthenticated Pipeline Run Execution

**Name:** `Unauthenticated-Pipeline-Run-Execution`
**Severity:** HIGH
**Found:** 2026-09-04
**Status:** Closed (fixed in commit de2a60f7e)

### Description

`POST /api/v1/data-pipelines/{id}/run` and `GET /api/v1/data-pipelines/runs/{id}` fell back to Gold Copy tenant without JWT. Hardened to 401 via `claimTenantIDFromRequest`. Tenant ownership check added to `GetRunStatus`.

---

## Finding: ListPipelines Endpoint Unauthenticated

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
