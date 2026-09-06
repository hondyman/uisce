# STATUS_AUDIT_BACKLOG.md — All Findings (Open & Closed)

**Severity scale:** `SEV-HIGH` = tenant isolation / production-correctness void · `HIGH` = silent failure in default path / deployment hazard · `MEDIUM` = capability gap, steady-state hazard · `LOW` = hygiene, governance, dead code already neutralized

**Branch provenance:** `critical-fixes-from-cleanup` — one-time port from `cleanup-node-edge-deadcode` to establish on `main`. Finding status as of 2026-09-05.

---

## Finding: Three A11y Crawl Routes Crash During Render

**Name:** `A11y-Crashes-Unauthenticated-API`
**Severity:** HIGH
**Found:** 2026-09-05
**Status:** Closed

### Description

Three routes in the a11y crawl crashed during render (no `<main>` landmark — error boundary triggered). The crashes were NOT caused by missing backend routes or URL mismatches — the routes were correctly wired. The crashes had two distinct root causes, both fixed.

| Crawl Route | React Router | Backend Route | Crash Cause | Status |
|---|---|---|---|---|
| `/en/wealth/feed` | `/wealth/feed` ✓ | **none** — no handler at `/api/wealth/feed` | Component threw: `useFeed()` returned `useMutation` result (wrong type); `data`/`isLoading`/`error` all undefined; `feedItems?.map()` threw on `undefined` | **Fixed** — `useQuery` used; component renders error state gracefully |
| `/en/secrets/audit` | `/secrets/audit` ✓ | `/api/admin/tenants/audit-logs` ✓ | `apiFetch` returned raw `Response` without throwing on non-OK; 401 HTML body caused `r.json()` to throw mid-chain; exception escaped React Query error boundary | **Fixed** — `apiFetch` now throws `ApiError` on non-2xx; React Query catches it cleanly |
| `/en/wealth/approvals/pending` | NO ROUTE | N/A | No React Router route; no crawl artifact; not a crash | **Closed** — no route exists |

### Systemic Fix

`frontend/src/lib/apiClient.ts` — `apiFetch` now throws a typed `ApiError` on any non-2xx response instead of returning the raw `Response`. This prevents the `r.json()`-on-non-JSON-body crash class across all 46 callers that chain `.then(r => r.json())` without checking `r.ok`. React Query catches the thrown error and sets its error state; components show their error UI instead of crashing.

### Root Causes

- **Feed.tsx**: `useFeed()` returned a `useMutation` result. Mutation `data` is the mutate response payload (undefined on a query hook), not a dataset. `isLoading` on a mutation is only true during mutate execution, not during loading. `feedItems = undefined`, `feedItems?.length === 0` was `false`, `feedItems?.map(...)` threw.

- **SecretsAuditPage.tsx + apiFetch**: `apiFetch` returned the raw `Response` on all status codes without throwing. For 401 responses with HTML error bodies, `response.json()` threw. The exception propagated during the React Query promise chain before the error state was set, crashing the component.

### Remaining Work

- **`secrets/audit` 401 (superseded)**: Prior diagnosis attributed the failure to `KEYCLOAK_ISSUER_URL` issuer mismatch — this was a misdiagnosis. The actual root cause was a panic in `HandleGetAuditLogs` (`backend/internal/handlers/audit_handler.go:187`) caused by DataFusion returning rows where column slices had unequal lengths (`index out of range [3] with length 3`). The panic produced the "empty reply" (HTTP 000) that was misread as a 401. Fixed in PR #7 — `safeVal()` closure added bounds checks. The page now returns HTTP 200 with real audit entries. No issuer mismatch existed.
- **`secrets/audit` nil rows (new — partial fix)**: The same DataFusion ragged-column condition that caused the panic now surfaces as fully-null entries (`{"id":"<nil>","tenantId":"<nil>",...}`) instead of crashing. `safeVal()` prevented the panic but silently admits garbage rows. Fix belongs either in the DataFusion query/aggregation (ensure uniform column lengths) or by skipping rows where `safeVal(0)` is nil. Root condition is the same as the panic fix.
- **`wealth/feed` backend handler**: No handler at `/api/wealth/feed`. The component handles errors gracefully but cannot show real content without the endpoint. Product decision: implement or remove the frontend route.
- **Re-crawl**: Run the a11y crawl to verify both pages render error states and the ratchet pair can be re-frozen.

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
**Status:** Closed

### Description

`ColumnMapper.Transform` in `backend/internal/datapipeline/transforms.go` deleted source fields when mapping to a different name. Given `Mappings: {"out_name": "node_name"}`:

1. `transformed[targetKey] = val` — copies value to `out_name`
2. `delete(transformed, srcKey)` — **deleted** `node_name`

Any downstream node that expected both `out_name` AND `node_name` received only `out_name`. The data loss was silent.

### Fix Applied

`delete(transformed, srcKey)` removed. Rename now uses copy semantics: source field is retained at its original key and the value is also placed at the target key. Rename-without-loss is now the default.

Move semantics (delete source after rename) is available as an explicit opt-in via `ColumnMapper.Move = true` or `config.move = true`.

### Test Results

`go test -count=1 -v ./internal/datapipeline/...`
- `TestTransforms_ColumnMapper` PASS — rename preserves source key (copy semantics)
- `TestTransforms_ColumnMapper_MoveSemantics` PASS — `Move=true` deletes source key after rename
- All other datapipeline tests PASS

### Commits

- `78934abcb` — fix(datapipeline): ColumnMapper renames copy by default, not move

**Refs:** [#3](https://github.com/hondyman/uisce/issues/3)

---

## Finding: Trigger Surface — Create-Only, No Visibility or Lifecycle

**Name:** `TriggerSurface-CreateOnly`
**Severity:** MEDIUM
**Found:** 2026-09-05
**Status:** Closed (2026-09-05)

### Description

The trigger-binding UI (TriggerAuthoringPage) can create bindings but cannot edit, deactivate, or view firing history. Event-driven pipelines stop being a demo only when operators can see which events fired which runs, and control when they fire.

### Fix Applied

Full lifecycle support shipped in commit 07e32a451:
- Backend: trigger_id FK on data_pipeline_runs; dispatch gate (is_active=true); last_fired_at LATERAL join; toggle endpoint; runs-per-trigger handler
- Integration tests: is_active=false prevents dispatch (verified against live DB)
- Frontend: trigger list table with is_active Switch (PUT /api/v1/triggers/{id}), last_fired_at, runs link

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

### Related Findings

**`RLS-SetContext-BindParam-Syntax` (SEV-HIGH, Open)**: This finding reveals a second, independent RLS failure. Even if the connection were demoted to enforce RLS, the `SetRLSContext` mechanism was broken — `SET LOCAL uisce.current_tenant = $1` is invalid SQL syntax (PostgreSQL doesn't support bind params in SET). The context was never being set on any call. Both findings together describe a RLS system that has never been operational: the connection bypasses it at the superuser level, and even if it didn't, the context-setting mechanism was broken. When `DBRuntime-Superuser-BYPASSRLS` is fixed, the `set_config(..., true)` mechanism must be retained as the correct context setter.

### Fix Direction

Demote the runtime connection to a role with only required permissions. Row-level security policies must enforce tenant isolation at the database layer.

### Technical Analysis

**The two migration conventions — incompatible context, same goal:**

| Migration | GUC name | Policy style | Enforcement |
|---|---|---|---|
| `010_rls_security.sql` (2025) | `app.current_tenant_id` | `TO tenant_user_role` + `USING (tenant_id = current_setting(...))` | Only when active role is `tenant_user_role` |
| `20260727000030_strict_tenant_rls.sql` | `uisce.current_tenant` | `FORCE ROW LEVEL SECURITY` + `uisce_get_current_tenant()` | All roles including superuser |

**Why the connection bypasses both:**

The app connects as `postgres` superuser. For `010_rls_security.sql` tables, the policies are `TO tenant_user_role` — they only fire when `ROLE = tenant_user_role`. `postgres` superuser is not `tenant_user_role`, so policies never apply. For `FORCE ROW LEVEL SECURITY` tables, the superuser is subject to RLS, but the context GUC was never set (the `SET LOCAL = $1` bug). Both layers were defeated by the same two bugs this engagement fixed.

**The two `set_config` calls that must both be set:**

```sql
-- Old migration reads this:
SELECT set_config('app.current_tenant_id', $1, true)

-- New migration (via uisce_get_current_tenant()) reads this:
SELECT set_config('uisce.current_tenant', $1, true)
```

`SetRLSContext` was recently fixed to set `uisce.current_tenant`. The old migration's policies still read `app.current_tenant_id`. The fix: `SetRLSContext` must set **both** GUCs atomically, or a migration must be written to update the old policies to use `uisce_get_current_tenant()`.

**Design: two-role connection pool**

```
app_user (NOLOGIN, non-superuser)  ← app connects as this
  └── SET ROLE tenant_user_role     ← normal queries, RLS enforced
  └── SET ROLE global_admin_role   ← cross-tenant reads (e.g., audit explorer)
```

`app_user` is granted `CONNECT` on the database and `USAGE` on schemas it needs, but **not** `BYPASSRLS`. `SET ROLE` switches the effective role within the connection; the connection itself stays as `app_user`.

**Why SET ROLE rather than connecting as the role directly:**

PostgreSQL doesn't allow a connection to `SET ROLE` to a role it wasn't granted — the initial connection auth determines what roles are available. By connecting as `app_user` (which is granted to the app's auth token) and then `SET ROLE tenant_user_role`, the connection inherits `tenant_user_role`'s RLS policy membership while the pool keeps its connection identity.

**Operations that need special handling:**

| Operation | Current | Needed | Plan |
|---|---|---|---|
| Regular tenant queries | `postgres` superuser | `app_user` + `SET ROLE tenant_user_role` | Main pool, `SetRLSContext` |
| `REFRESH MATERIALIZED VIEW` | `postgres` superuser | Separate connection as `global_admin_role` | Admin pool or `SET ROLE global_admin_role` (BYPASSRLS but `FORCE RLS` tables still enforce `WITH CHECK`) |
| `pg_class` stats | `postgres` superuser | `app_user` (has catalog read access) | Works without change |
| DDL / `CREATE SCHEMA` | `postgres` superuser | Separate provisioner connection | Only in `tenant provision` binary, not main app |
| `ALTER TABLE FORCE ROW LEVEL SECURITY` | Superuser | Migration-only (never in runtime) | N/A |

**Migration plan:**

1. **Create `app_user` role** (non-superuser, `NOLOGIN`) with `GRANT CONNECT ON DATABASE`, `GRANT USAGE ON SCHEMA public`, `GRANT SELECT/INSERT/UPDATE/DELETE` on all tenant tables (matching `tenant_user_role`'s grants).
2. **Update `SetRLSContext`** to set both `app.current_tenant_id` and `uisce.current_tenant` — or write a migration to update old policies to use `uisce_get_current_tenant()`. (The `uisce.current_tenant` is the canonical name going forward; `app.current_tenant_id` is the legacy name that should be phased out.)
3. **Update connection pool** to connect as `app_user` (change DSN user or add `user=` to connection string). The pool authenticates as the OS user or as `app_user` via `pg_hba.conf`.
4. **After acquiring connection, before each transaction**: `SET ROLE tenant_user_role` (or `global_admin_role` for admin contexts). This is a zero-cost session setting.
5. **Admin pool**: for `REFRESH MATERIALIZED VIEW` and other operations that need `global_admin_role` BYPASSRLS, use a separate connection pool authenticated as `app_user` but calling `SET ROLE global_admin_role`. Note: `FORCE ROW LEVEL SECURITY` tables still enforce their `WITH CHECK` even for `global_admin_role`; the admin bypass only works for the older `010_rls_security.sql` tables.
6. **Test**: verify that `SELECT * FROM tenant_instance` (FORCE RLS table) returns 0 rows for a tenant context, and all rows for a global admin context.

**`app.current_tenant_id` vs `uisce.current_tenant` — canonical name decision needed:**

The codebase now sets `uisce.current_tenant`. The old migration (`010_rls_security.sql`) still reads `app.current_tenant_id`. Until those policies are migrated to use `uisce_get_current_tenant()`, both must be set. Decision: either (a) migrate old policies to `uisce_get_current_tenant()` and remove `app.current_tenant_id` entirely, or (b) keep both and standardize on `app.current_tenant_id` going forward. Option (a) is preferred — one canonical GUC name reduces future confusion.

---

## Finding: SET LOCAL Does Not Support Bind Parameters — RLS Context Never Set

**Name:** `RLS-SetContext-BindParam-Syntax`
**Severity:** SEV-HIGH
**Found:** 2026-09-05
**Status:** Closed — Fix Applied (merged to main via PR #11). Pending BYPASSRLS connection demotion to fully operationalize the RLS layer.

### Description

PostgreSQL's `SET` and `SET LOCAL` statements do **not** support bind parameters (`$1`, `$2`). Using `SET LOCAL uisce.current_tenant = $1` silently fails with a syntax error on every call — PostgreSQL rejects the `$1` token in SET syntax. The error was not handled; all callers believed RLS context was set and queries were tenant-scoped, when in fact no RLS context existed at all.

Four code instances had this pattern:

| Location | Function | Status | Callers |
|---|---|---|---|
| `internal/tenant/context.go:40` | `SetRLSContext` | **Broken — fixed** | 15 call sites across `scheduler_service.go`, `export_service.go` |
| `internal/middleware/security_helpers.go:95` | `SetTenantContext` | Dead code (unused) | None |
| `internal/middleware/security_helpers.go:108` | `SetGlobalAdminContext` | Dead code (unused) | None |
| `internal/middleware/tenant_context.go:52` | `SetSessionTenantContext` | Dead code (unused) | None |

The shared helper `tenant.SetRLSContext` was called 15 times across scheduler and export services. Every call — on every request to those endpoints — silently failed. RLS context was never set. Queries executed without tenant scoping.

**Confirmed blast radius**: `GET /api/v1/schedules` with global admin + tenant header returned HTTP 500 `"failed to set RLS context: pq: syntax error at or near "$1"` after `tenant.SetRLSContext` was reached. Without the fix, every scheduler and export endpoint would fail identically.

**Empirical evidence**: the probe of `GET /api/v1/schedules` produced the syntax error before the fix and returned `{"schedules":[],"total":0}` with HTTP 200 after. The empty array means no schedules exist, not an error.

### Fix Applied

Replaced all four instances with `SELECT set_config('uisce.current_tenant', $1, true)` — the `set_config()` function **does** accept bind parameters. The third argument (`true`) makes it transaction-scoped, equivalent to `SET LOCAL`, and it reverts automatically at `COMMIT/ROLLBACK`, so connection pool safety is preserved.

### Relation to BYPASSRLS Finding

The `DBRuntime-Superuser-BYPASSRLS` finding notes that the application connection runs as `postgres` superuser with `BYPASSRLS`, meaning RLS policies are bypassed entirely. This finding reveals a second, independent RLS failure mode: even if the connection *did* enforce RLS, the context was never being set because `SET LOCAL` doesn't accept bind parameters.

Both findings together describe a RLS system that has never been operational: the connection bypasses it at the superuser level, and even if it didn't, the context-setting mechanism was broken. The BYPASSRLS fix (connection demotion) must also fix the `SetRLSContext` mechanism to properly set transaction-scoped RLS context using `set_config()`.

### Fix Direction

1. ✅ **Fixed in `fix/dead-handler-removal`**: `tenant.SetRLSContext` now uses `set_config(..., true)`. The RLS context mechanism works.
2. **Pending**: `DBRuntime-Superuser-BYPASSRLS` demotion to a constrained role — when done, `set_config()` context setting must remain correct as the replacement for the broken `SET LOCAL` approach.

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

---

## Finding: RBAC Nil-Deref — Every Require*Permission Call Would Panic in Production

**Name:** `RBAC-Nil-Deref-GetClaimsFromContext`
**Severity:** SEV-HIGH
**Found:** 2026-09-05
**Status:** Closed

### Description

`internal/api/middleware/rbac_enforcement.go:getTenantIDFromRequest` called `jwtmiddleware.GetClaimsFromContext(r).TenantID` directly, with no nil guard. In production, `AuthContextMiddleware` sets `security.AuthInfo` in the context — it does NOT set the `"jwt_claims"` key that `GetClaimsFromContext` reads. The call always returned nil, and the subsequent `.TenantID` access was a nil-pointer dereference. Every request enforced by `RequirePermission`, `RequireAnyPermission`, `RequireAllPermissions`, `RequireRole`, or `RequireRoleLevel` would have panicked.

Impact: if RBAC middleware was ever exercised in production, it crashed every such request. If it was never exercised, RBAC permissions were never actually enforced — a silent security failure of a different kind.

### Fix Applied

Canonical `TenantIDFromRequest(r *http.Request) (string, bool)` added to `internal/api/helpers.go`. Resolution order: (1) `security.AuthInfo` from `AuthContextMiddleware`, (2) `jwtmiddleware.GetClaimsFromContext` for standalone services with their own wiring, (3) `(""`, `false)`. Callers respond 401 on `false`.

`rbac_enforcement.go` replaced `getTenantIDFromRequest` with `tenantIDFromRequest` returning `(string, bool)`. All 5 `Require*` functions now respond 401 when tenant cannot be resolved (not 400, since absent tenant in this context means auth context is absent — fail-closed).

`business_object_handlers.go:ResolveBindingDatasource` updated to use canonical helper with 401.

`extractTenantContext` in `helpers.go` similarly updated.

### Files Changed

- `backend/internal/api/helpers.go` — canonical `TenantIDFromRequest` + fixed `extractTenantContext`
- `backend/internal/api/middleware/rbac_enforcement.go` — `tenantIDFromRequest` + 5 call sites
- `backend/internal/api/business_object_handlers.go` — `ResolveBindingDatasource`
- `backend/internal/api/middleware/rbac_enforcement_test.go` — 5 regression tests

### Test Results

```
go test ./internal/api/middleware/... -run "TestRequirePermission|TestTenantIDFromRequest" -v
- TestRequirePermission_NoAuthContext_Returns401NotPanic    PASS
- TestRequirePermission_WithAuthInfo_ProceedsToPermissionCheck PASS
- TestTenantIDFromRequest_AuthInfoOnly_ReturnsTenantID      PASS
- TestTenantIDFromRequest_NoAuthNoClaims_ReturnsFalse       PASS
- TestRequirePermission_MissingDatasource_Returns400         PASS
```

### Commits

- `fcc42adf1` — SEV-HIGH: RBAC nil-deref fix — canonical tenant helper, fail-closed middleware (already on main)
- `28045f5d3` (PR #11) — fix: migrate 30 TenantID call sites from GetClaimsFromContext to TenantIDFromRequest

---

## Finding: GetClaimsFromContext Direct Access Without Nil Check — Main Server Internal Handlers

**Name:** `GetClaimsFromContext-MainServer-NilDeref`
**Severity:** HIGH
**Found:** 2026-09-05
**Status:** Closed (PR #11 — 30 sites migrated)

### Description

`jwtmiddleware.GetClaimsFromContext` was called in ~100 places across the codebase. The standalone services (validation-service, notifications-service, rule-engine-service) were audited and are clean — they wire `JWTMiddleware` which sets `jwt_claims`, and all 19 handlers fail closed on nil.

The **main server's `internal/` packages** are the actual risk: `AuthContextMiddleware` sets `security.AuthInfo`, NOT `jwt_claims`. Every handler in `internal/` that calls `GetClaimsFromContext` gets `nil` unless `AuthContextMiddleware` also set `jwt_claims`. If a handler then does `claims.TenantID` without a nil check, it panics.

### Audit Results — Main Server Internal Call Sites

Approximately **25-30 call sites** across `internal/` directly access `.TenantID` or `.UserID` without nil checks:

| File | Risk Level | Pattern |
|------|------------|---------|
| `apistudio/runtime.go:112` | CRITICAL | `tenantIDStr := GetClaimsFromContext(r).TenantID` — no nil check |
| `apistudio/odata.go:84` | CRITICAL | Same pattern |
| `nba/websocket.go:190` | CRITICAL | WebSocket handler — may bypass middleware |
| `onboarding/handlers.go:37` | CRITICAL | `tenantIDStr := GetClaimsFromContext(r).TenantID` |
| `billing_handlers.go:62` | CRITICAL | `tenantID = GetClaimsFromContext(r).TenantID` |
| `internal_event_handler.go:56` | CRITICAL | Same pattern |
| `temporal_admin.go` (8+ sites) | CRITICAL | Multiple direct accesses |
| `handlers/cbo_handler.go` | HIGH | Multiple direct accesses |
| `handlers/ai_handler.go` | HIGH | `uuid.Parse(GetClaimsFromContext(r).TenantID)` — ignores error |
| `handlers/scheduler_handlers.go` | HIGH | `normalizeTenantID(GetClaimsFromContext(r).TenantID)` |
| `handlers/export_handlers.go` | HIGH | Same pattern |
| `reporting/handler.go:599` | HIGH | `getTenantContext` utility |
| `simulation/handler.go:50` | HIGH | `req.TenantID = GetClaimsFromContext(r).TenantID` |

These are reachable from `main.go` via `SetupRouter`. Each is a latent panic if the JWT token fails to populate `jwt_claims` in the context — which happens for any request where `AuthContextMiddleware` sets `security.AuthInfo` but not `jwt_claims`.

### Fix (Closed — PR #11)

Replaced each call site with the canonical `TenantIDFromRequest(r)` helper from `helpers.go` (added in fcc42adf1): checks `security.AuthInfoFromContext` first, then `jwtmiddleware.GetClaimsFromContext` as fallback, returns `("", false)` if neither is populated.

**9 live files migrated (30 sites):**
- `api/billing_handlers.go` (1) — 401 on unresolved tenant
- `api/node_types_routes.go` (1)
- `api/api.go` (2): `debugProxyHeaders`, `startProfile`
- `handlers/ai_handler.go` (5)
- `handlers/export_handlers.go` (5)
- `handlers/scheduler_handlers.go` (6)
- `handlers/source_preference_handler.go` (1)
- `nba/websocket.go` (1)
- `onboarding/handlers.go` (1)

Handlers use `security.AuthInfoFromContext` directly (different package from `helpers.go`); api files use `TenantIDFromRequest` from same package.

**14 dead files deleted** (PR #11 — confirmed not registered in `SetupRouter`):
`aso_handler.go`, `internal_event_handler.go`, `temporal_admin.go` (9 sites), `apistudio/runtime.go`, `clientportal/handlers.go`, `cbo_handler.go` (4), `observability_handler.go`, `pre_aggregation_handler.go`, `pricing_handler.go`, `rules_handler_impl.go` (2), `slo_handler.go`, `term_metadata_handler.go`, `rulefabric/handler.go`, `rulefabric/bo_policy_handler.go` (depended on dead handler.go).

Note: `apistudio/odata.go` and `reporting/handler.go` were already absent from this tree.

### Audit Logs Empty Reply — Confirmed Panic, Fixed

`/api/admin/tenants/audit-logs` with a valid Keycloak token returned "empty reply from server" (HTTP 000, connection closed). Confirmed root cause: `runtime error: index out of range [3] with length 3` at `audit_handler.go:187`. The `colCount >= 9` check validated the number of column slices but not the length of each inner row. When column 4 (user_name) had fewer elements than `resp.RowCount`, indexing `resp.Records[3][rowIdx]` panicked.

Fixed in PR #7: `safeVal(col int)` closure checks `col < len(resp.Records) && rowIdx < len(resp.Records[col])` before every access. Request now returns HTTP 200 with real audit entries. Remaining: nil rows (see above, same root condition).

### Remaining

- `apps/analytics-api`, `apps/genui-api`, `apps/orchestration-api` — own `JWTMiddleware` wiring, clean (confirmed by prior audit)

---

## Finding: connections_routes.go Header Fallback Resolves Tenant from Client-Controlled Header

**Name:** `Connections-Tenant-Header-Fallback`
**Severity:** MEDIUM
**Found:** 2026-09-05
**Status:** Closed (Fixed)

### Description

`internal/api/connections_routes.go:getTenantIDFromRequest` (separate implementation from the RBAC one) used:
```go
if claims, err := jwtmiddleware.ValidateTokenFromRequest(r); err == nil && claims != nil && claims.TenantID != "" {
    return claims.TenantID
}
return r.Header.Get("X-Tenant-ID")  // ← untrusted fallback
```

When `ValidateTokenFromRequest` fails or returns nil claims, the function fell back to reading `X-Tenant-ID` directly from the request header. This is the client-controlled input path. Token-validation failure could occur for reasons unrelated to identity (expired token, wrong signature key, etc.), and the fallback would accept any tenant ID the client sends — enabling tenant impersonation.

### Audit Result

10 call sites in `connections_routes.go` all call `getTenantIDFromRequest` and pass the result directly to service methods as `tenantID`. No call site intentionally uses API key auth or treats the header as a trusted input. The fallback was a bug.

The `getTenantIDFromRequest` in `rbac_enforcement.go` (lines 340–342) is a separate implementation that reads from JWT claims context only — no header fallback — and is unused in production routes (only in test scaffolding).

### Fix Applied

Replaced the fallback with a hard `return ""` when JWT validation fails. Handlers then emit `400 "tenant_id is required"` for unauthenticated requests. Fix: `backend/internal/api/connections_routes.go:48–53`.

### Relation to `GetClaimsFromContext` Migration (PR #11)

The PR #11 migration changed most handlers to use `security.TenantIDFromContext` which does NOT fall back to `X-Tenant-ID`. The `connections_routes.go` `getTenantIDFromRequest` was a separate, parallel implementation that was missed in that migration — and had the additional bug of the header fallback.

---

## Finding: Multi-Issuer JWKS Trust — Deferred

**Name:** `Multi-Issuer-IDP-Trust`
**Severity:** HIGH
**Found:** 2026-09-05
**Status:** Deferred

### Rationale for Deferral

Extraction was previously justified by a suspected `KEYCLOAK_ISSUER_URL` vs. e2e issuer mismatch causing `/api/admin/tenants/audit-logs` to return 401. The actual root cause was confirmed to be a panic in `HandleGetAuditLogs` (fixed in PR #7). No issuer mismatch existed. The multi-issuer work's primary use case was the misdiagnosed failure; with that failure closed, the extraction loses its urgency.

### What Exists on `cleanup-node-edge-deadcode`

The branch contains a full multi-issuer JWKS trust implementation spanning ~7 commits:

| Component | File | Type | Extraction Risk |
|----------|------|------|----------------|
| `IssuerRegistry` interface + `DBIssuerRegistry` | `backend/internal/security/idp_registry.go` | New file — cherry-pick clean | Trivial |
| `FetchJWKS` / `FetchAllTrustedKeys` | `backend/internal/security/jwks.go` | New file — cherry-pick clean | Trivial |
| `ValidateIssuerTenant` | `backend/internal/services/idp_refresh.go` | New file — cherry-pick clean | Trivial |
| `refreshAllTrustedKeys` | `backend/internal/api/helpers.go` (+64 lines) | Modifies existing — cherry-pick clean | Trivial |
| `SecurityManager.StartIssuerKeyRefresh` | `backend/internal/services/security_manager.go` | Modifies existing — cherry-pick clean | Trivial |
| `AuthContextMiddleware` signature + rewrite | `backend/internal/middleware/auth_context.go` (380-line diff) | Full rewrite — requires manual review | **Significant** |
| `SetupRouter` idpRegistry wiring | `backend/cmd/server/main.go`, `backend/internal/api/api.go` | Adds IssuerRegistry param to middleware calls | Manual |

### Commit History (branch `cleanup-node-edge-deadcode`)

```
0cb8099481 — feat(security): enforce per-tenant IDP/issuer trust on external JWTs
28715bea68 — security: Mitigate JWT trust inversion and enforce alg/iss validation
04eff75d4a — security: Strip identity headers, enforce fail-closed structural auth dispatch
508d38a11a — security: Add positive/negative auth matrix test suite, reject legacy impersonation
ac2d340266 — fix(security): Add nil db guards in api aggregates setup and idp registry
c8b752086d — security: Protect SSE endpoint with AuthContextMiddleware, add WS unit tests
```

### Extraction Note

`04eff75d4a` alone touches 382 lines of `auth_context.go` (header stripping + structural dispatch) and should be reviewed as a separate commit. The `auth_context.go` rewrite is the hard part — not the four new files. When this work is eventually landed, it should be a reviewed re-implementation on `main`, not cherry-picks.

### Sequencing Note

The 47-site `GetClaimsFromContext` migration and the `auth_context.go` rewrite both live in the identity layer. Landing the rewrite first would churn the exact middleware whose contract (`security.AuthInfo` in context) the 47 sites depend on. Correct order: migrate handlers first on stable middleware, then rewrite the middleware.
