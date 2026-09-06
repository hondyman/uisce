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

### CI Workflow History Cleanup

The a11y-ratchet.yml gate went through six iteration rounds on main while we peeled configuration layers (parquet-go link error → npm peer-deps → locale-matrix → start-dev.sh portability → axe-core declaration → Tailscale runner constraint). The commit history shows the layer-peel nicely, but it's also CI-debugged-on-main history.

When the self-hosted runner is installed and the workflow stabilizes:
- One cleanup PR (or squash, if history allows) to consolidate the
  `test: a11y ratchet trigger (round N)` and `fix(a11y): remove locale-matrix`
  and `fix(frontend): declare @axe-core/playwright` commits into a single
  "install a11y ratchet gate, document Tailscale runner constraint" commit.
- The story this tells should be "CI evolved through review" not
  "CI was debugged on main."
- Actual command once runner is up and stable:
    git -C backend/   rebase -i HEAD~6  # squash test/fix pairs

Don't do this before the runner is online — the failure-mode commits
  are the most readable evidence that the gate is honest.

### BYPASSRLS Design — Evidence Pack

Gathered 2026-09-06 (PostgreSQL 18.6, db `alpha` on `100.84.50.65`).

**Q1 — Do RLS policies exist?**

YES. **603 policies across 529 distinct tables** in `public` schema
(and one in `calendar`). Sample (`pg_policies` first 50 rows by
tablename): every tenant-scoped table has a `tenant_isolation_policy`
on `{public}` role, plus a handful of specialized policies
(`{authenticated}`-bound for Keycloak-federated reads; per-table
admin policies; `_tenant_isolation` suffixed variants).

Conclusion: the policy authoring work is **largely done**. The
remaining work is roles, wiring, and the app-DSN switch (see Q3).
NOT a per-table authoring project.

**Q2 — Tenant-scoped table count**

`SELECT count(*) FROM information_schema.columns
WHERE table_schema='public' AND column_name='tenant_id'` → **533**.

Of those, ~529 have RLS policies (numbers align with Q1). The 4-5
delta is likely: tenant_id-bearing tables that are intentionally
unsecured (e.g. cross-tenant lookup tables like `trigger_types`),
plus any in-flight migrations.

**Q3 — Roles inventory**

```
 rolname                    | rolsuper | rolbypassrls | rolcanlogin
----------------------------+----------+--------------+------------
 app_user                   | f        | f            | t
 infisical                  | f        | f            | t
 keycloak                   | f        | f            | t
 nessie                     | f        | f            | t
 postgres                   | t        | t            | t   <-- only BYPASSRLS
 semlayer_lookups_replica   | f        | f            | t
 temporal                   | t        | f            | t   <-- superuser but NO BYPASSRLS
 usice_app                  | f        | f            | t
 usice_ops                  | f        | f            | t
```

Only `postgres` has BYPASSRLS. `temporal` is superuser but does
NOT have BYPASSRLS — meaning the migration runner respects RLS
(unless `temporal` reconnects as `postgres` to bypass).

**App connection (today):** `DATABASE_URL=postgres://postgres@100.84.50.65:...`
— the app connects as `postgres`, which has BYPASSRLS enabled.
**This means every `set_config('uisce.current_tenant', ...)` call
is currently inert** — the policy never fires because the role
bypasses it. The `set_config` mechanism is correct in isolation;
the gap is the connection role.

Implication: the BYPASSRLS fix is almost entirely an app-side
DSN switch (`postgres` → `usice_app` or `app_user`), plus wiring
the existing `SetRLSContext` helper into the request path. No
new policy authoring. No DSN-level connection pool rewrite (no
pgxpool in the codebase — see Q4).

**Q4 — Connection pool architecture (`rg sqlx.Connect|pgxpool|sql.Open`)**

Every cmd/entry point uses `sql.Open` / `sqlx.Connect` directly
from `DATABASE_URL`. No central connection pool. No `pgxpool`
anywhere — everything is `database/sql` / `sqlx`. Each cmd binary
opens its own connection(s) per process. `internal/multitenancy/manager.go`
keeps a connection ref but creates per-call (it's commented out in
the source — see that file).

`internal/middleware/security_helpers.go` already has the comment:
"a transaction (via db.BeginTx) — otherwise SET LOCAL reverts
immediately" — confirming the project understands the tx-scoped
constraint.

**Transaction boundaries:** `rg "BeginTx|Begin\(" backend/internal --type go` → **132 matches** across the internal packages. `set_config(..., true)` is tx-scoped: the call must be on the same tx as the query, or `SET LOCAL` reverts.

**Q5 — Pool architecture = one direct connection per binary, per process**

  - Per-cmd `sql.Open` / `sqlx.Connect` against `DATABASE_URL`
  - No pooled backend, no pgxpool
  - Connection role = postgres (BYPASSRLS) — `set_config` is currently inert
  - 132 explicit tx boundaries in internal/

**Implications for BYPASSRLS design:**

1. Roles are good. Policies are written. The gap is the app's
   connection role.
2. App-side DSN switch (`postgres` → `usice_app`) is the main lever.
   `usice_app` has `BYPASSRLS=false`, will honor every existing policy.
   Validate this with `psql` as `usice_app` once it's enabled with
   `LOGIN`.
3. `set_config('uisce.current_tenant', $1, true)` then becomes the
   active mechanism — driven by JWT claims from `security.AuthInfo`.
   The middleware at `internal/tenant/context.go:24` and
   `internal/middleware/security_helpers.go` already does this.
4. 132 tx boundaries mean SET LOCAL discipline must apply across all
   of them. Easiest: a tx-wrapping helper that always sets
   `uisce.current_tenant` first, then runs the user's queries.
   Mirrors the `BeginTx` pattern at
   `internal/middleware/security_helpers.go`.
5. Migration order: (a) create a non-BYPASSRLS role for app;
   (b) verify policies enforced against it with `SET ROLE` in
   dev; (c) flip DATABASE_URL; (d) monitor `pg_stat_activity` for
   the new role.

This is the design conversation to start next session.

### BYPASSRLS Evidence — Corrections (Post-Hoc)

Two corrections to the prior evidence pack, surfaced on review:

**Correction 1 — Superusers bypass RLS unconditionally.**

The Postgres docs are explicit: superusers and BYPASSRLS roles
**always** bypass RLS. The `rolbypassrls` flag is only meaningful
for non-superusers — for superusers it's decorative.

**This means `temporal` (rolsuper=true, rolbypassrls=false) is
still a leak path.** It does NOT respect RLS despite the flag.
Two leak paths, not one:

```
 rolname    | rolsuper | rolbypassrls | actually_bypasses
------------+----------+--------------+-------------------
 postgres   | t        | t            | YES — superuser
 temporal   | t        | f            | YES — superuser  ← also leaks
 usice_app  | f        | f            | NO  — honors RLS
 usice_ops  | f        | f            | NO  — honors RLS
 app_user   | f        | f            | NO  — honors RLS
```

Implication: phase 4 needs to demote `temporal` to a non-superuser
role (`usice_ops` or a new dedicated role with `LOGIN` + RLS-respecting
privileges) OR explicitly carve an exception with audit-logging.
The "superuser but no BYPASSRLS" appearance is **not** a defense —
it's a misleading column.

**Correction 2 — Policy defined ≠ RLS enforced.**

`pg_policies` lists defined policies, but RLS only fires when
`ALTER TABLE ... ENABLE ROW LEVEL SECURITY` was run
(`relrowsecurity = true`). And even when enabled, **the table
owner bypasses unless `FORCE ROW LEVEL SECURITY`**
(`relforcerowsecurity = true`).

Aggregate (psql on alpha, 2026-09-06):

```
 count(*) AS total_tables_public          → 882
 count(*) FILTER (relrowsecurity)         → 470  (RLS enabled)
 count(*) FILTER (relforcerowsecurity)    → 456  (forced)
 count(*) FILTER (NOT relrowsecurity)     → 412  (RLS off — accepted)
 count(*) FILTER (relrowsecurity
             AND NOT relforcerowsecurity) →  14  (unforced gap)
```

Cross-check with `pg_policies`: 603 policies across 529 tables.
Of those, all are on tables where `relrowsecurity = true`
(no RLS-off-with-policy tables found). So the policies that
exist are enforced. But:

- **14 tables are unforced** (RLS on, FORCE off). Three of them
  have a `tenant_id` column:
    - `calc_fields`
    - `notification_outbox`
    - `okf_concept_manifest`
    - `semantic_term_tags`
  These can be bypassed by the table owner today. Need a
  one-line `ALTER TABLE ... FORCE ROW LEVEL SECURITY` to close.
- **882 - 470 = 412 tables** have no RLS at all. This is much
  larger than the 4-5 the prior evidence suggested. Many of
  those are likely legitimate (lookup tables, platform-internal
  state, audit_log, tenants itself, portal_metrics), but the
  evidence pack's "policies are done" framing was incomplete —
  the question is now "which of the 412 are intentionally open?"

**Action items generated by these corrections:**

1. Fix the misleading `tenants row ` note: the claim "RLS-off
   tables that are intentionally unsecured like trigger_types"
   was correct in spirit but understated: 412 tables are
   RLS-off, not 4-5. Most are likely intentional, but they
   need a categorized list — not just a number.
2. Investigate the 14 unforced tables — determine which of the
   `tenant_id`-bearing four are accidentally unforsed vs.
   intentional.
3. Treat `temporal` as a real leak path, not a decorative one.
   Phase 4 cannot leave temporal as superuser.

**Q5 — Bare-query inventory size (the migration workstream):**

```
rg "\.db\.(Queryx|Query|Get|Select|Exec|NamedExec)
    |h\.db\.(Queryx|Query|Get|Select|Exec)" backend/internal --type go
```

→ 2,573 hits. Top 10 files account for ~440 of these:
```
 102 backend/internal/ops/store_postgres.go
  71 backend/internal/metadata/businessobject_service.go
  49 backend/internal/altinv/advisor_activities.go
  45 backend/internal/api/glossary_handler.go
  33 backend/internal/analytics/semantic_mapping_service.go
  31 backend/internal/api/bp_rbac_handlers.go
  29 backend/internal/wealth/client_portal_db.go
  28 backend/internal/api/marketplace_integration_handlers.go
  27 backend/internal/rulefabric/handler.go
  27 backend/internal/analytics/term_relationship_service.go
```

The 2,573 is **raw grep**, not migration inventory. Many of
these cluster around repository pattern helpers (`r.db.Queryx`,
`h.db.Query`, etc.) where the helper itself takes `*sqlx.Tx` —
those don't all need migration. Real inventory requires
call-graph analysis: which bare queries already flow through
a tx-wrapped helper, and which still need one.

**Phase 2 size reframed:** not "DSN switch" but "DSN switch +
tx-discipline migration across the query surface." The order
of magnitude is now visible (top 10 files × ~30 sites each is
where the work concentrates), and the migration surface is
substantial — calls for a phased rollout, not a flag flip.

