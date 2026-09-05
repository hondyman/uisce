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

- **`secrets/audit` 401**: Root cause is `ValidateIssuerTenant` rejecting the e2e Keycloak token because `KEYCLOAK_ISSUER_URL` (server-side env) doesn't match the e2e Keycloak issuer (`https://100.84.50.65:8443/realms/uisce`). This is a **harness configuration gap**: the e2e test's Keycloak instance is not registered as a trusted issuer in the backend. Fix: either set `KEYCLOAK_ISSUER_URL=https://100.84.50.65:8443/realms/uisce` on the test server, or use the same Keycloak instance for e2e that the server trusts. The page renders its error state gracefully — crash is fixed, 401 is a harness/config issue.
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

[pending]

---

## Finding: GetClaimsFromContext Direct Access Without Nil Check — Main Server Internal Handlers

**Name:** `GetClaimsFromContext-MainServer-NilDeref`
**Severity:** HIGH
**Found:** 2026-09-05
**Status:** Open

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

### Fix

Replace each call site with the canonical `TenantIDFromRequest(r)` helper, which checks `security.AuthInfoFromContext` first, then `jwtmiddleware.GetClaimsFromContext` as fallback, and returns `("", false)` if neither is populated. Handlers should then fail-closed (401) when `ok == false`.

### Also: Audit Logs Empty Reply — Suspected Panic

`/api/admin/tenants/audit-logs` returns "empty reply from server" (connection closed) with a valid Keycloak token, while the same token works for `/api/v1/triggers`. Without server logs, the cause is unconfirmed, but the pattern is consistent with a panic in the handler or middleware when `ValidateIssuerTenant` rejects the e2e Keycloak issuer and triggers a nil-deref in the fallback path. This is likely another instance of the same `AuthInfo`/`jwt_claims` mismatch class.

### Remaining

- GitNexus reindex running — bulk reachability query from `main.go` will confirm which Category 1 call sites are live-panic vs. dead-code
- `pkg/meta/api.go` and `local/cmd/proxy/main.go` not yet audited

---

## Finding: connections_routes.go Header Fallback Resolves Tenant from Client-Controlled Header

**Name:** `Connections-Tenant-Header-Fallback`
**Severity:** MEDIUM
**Found:** 2026-09-05
**Status:** Open

### Description

`internal/api/connections_routes.go:getTenantIDFromRequest` (separate implementation from the RBAC one) uses:
```go
if claims, err := jwtmiddleware.ValidateTokenFromRequest(r); err == nil && claims != nil && claims.TenantID != "" {
    return claims.TenantID
}
return r.Header.Get("X-Tenant-ID")  // ← untrusted fallback
```

When `ValidateTokenFromRequest` fails or returns nil claims, the function falls back to reading `X-Tenant-ID` directly from the request header. This is the client-controlled input path. The token-validation failure could occur for reasons unrelated to identity (expired token, wrong signature key, etc.), and the fallback would accept any tenant ID the client sends.

The `Require*Permission` middleware in `rbac_enforcement.go` was the panic case; this is the silent-authorization case — requests that fail token validation getting a different tenant than intended based on a client-supplied header.

### Required Action

Audit all 14 call sites in `connections_routes.go` to determine whether the header fallback is intentional (e.g., for API key auth where the header is trusted) or a bug. Replace with canonical `TenantIDFromRequest` after audit.
