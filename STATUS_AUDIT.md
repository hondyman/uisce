# STATUS_AUDIT.md — Studio + Related Packages

**Audit scope**: `backend/internal/datapipeline/`, `backend/internal/validation/`,
`backend/internal/oms/{account,position,security,trade_order}/`,
`backend/internal/catalog/`, plus the cross-package wiring in
`backend/internal/api/api.go:1567-1591` and `backend/cmd/worker/main.go`.

**Method**: static analysis (read-only). Every claim is grounded in a
`file:line` citation or a test name. Findings graded ✅ Wired & verified,
⚠️ Wired but partial/broken, ❌ Documented but not implemented.

**Evidence conventions**:

- **🟢 Static, verified**: claim is grounded in code reading only.
- **🟡 Static, awaiting DB**: claim is grounded in code reading but
  requires `\d` introspection of the running dev Postgres to confirm
  runtime behavior (which columns/constraints actually exist). These
  findings will be resolved or upgraded by the operator session
  (rotation + `pg_reload_conf` + `\d tenants` + `\d catalog_node`).
  Initial commit includes only static evidence; a follow-up commit
  (not `--amend`) folds in the DB evidence.
- **🔴 Static, refuted by evidence**: claim was overstated or contradicted
  by adjacent code reading. Such findings are removed or rewritten
  with the correct evidence in this document.

**Schema authority**: `schema.sql` and `totalddl.sql` are **not** the
schema authority for the running database. The schema authority is
the migration runner's applied history. Migrations may diverge from
the DDL files, and a constraint named in code may live in a migration
not yet found by grep. Findings that cite `schema.sql` are explicitly
marked 🟡 awaiting DB confirmation.

**Generated**: Phase 0 days 1–5 of the Studio wire-up sprint.

**Note**: percentage summaries (e.g., "~70% real") are explicitly
omitted from this document. They cannot be measured without running
the code, which is outside Phase 0 scope.

---

## 1. `backend/internal/datapipeline/` — the Studio's executor

### Files

| File | LOC | Verdict |
|------|-----|---------|
| `model.go` | 109 | ✅ Wires `PipelineDefinition`, `PipelineNode`, `PipelineEdge`, `PipelineExecutionRun`, `StepMetrics`, `TestStepRequest/Response` |
| `handlers.go` | 553 | ⚠️ Mounted correctly; security gaps, missing auth on Run/SSE |
| `engine.go` | 640 | ⚠️ Topological execution missing; rest real |
| `transforms.go` | 618 | ⚠️ 8 transformers wired; `APICallerTransformer` is a stub |
| `bo_driver.go` | 438 | ✅ Real DB I/O with RLS session binding |
| `catalog_driver.go` | 262 | ⚠️ Gold Copy check broken (`is_gold_copy` column does not exist) |
| `workflow.go` | 78 | ✅ Real Temporal workflow + activity pair |
| `outbox.go` | 106 | ⚠️ Async outbox publish not transactional with BO write |
| `legacy_convert.go` | 176 | ✅ One-shot migration helper for retired `pipelines` table |
| `calc_transform.go` | 92 | ✅ Real `boresolver.HostRuntimeExecutor` call |
| `engine_test.go` | 369 | ⚠️ 7 tests; none against a real DB |

### ✅ Wired & verified

- **STI bulk loader writes real data.** `bo_driver.go:152-262`
  `BulkLoadSTI` issues a parameterized multi-row
  `INSERT … VALUES ($1, $2), … ($n, $n+1) ON CONFLICT (id) DO UPDATE`,
  inside a `*sqlx.Tx` with `SET LOCAL uisce.current_tenant = $1`, sets
  `created_at`/`valid_from`, commits. Tests exist (`engine_test.go:118-189`).
- **Temporal workflow + activity registered end-to-end.**
  `cmd/worker/main.go:276-280` registers
  `RunPipelineDAGWorkflow` and `ActivityExecutePipelineDAG` on
  `bp_queue` task queue. `engine.go:301-323` dispatches
  via `temporalClient.ExecuteWorkflow`. `workflow.go:48-78` defines the
  wrapper that delegates to the activity.
- **Validation trigger → pipeline dispatch is wired.**
  `outbox.go:42-71` publishes `Pipeline.Trigger` events;
  `cmd/worker/main.go:437-453` polls the outbox and routes via
  `NewPipelineTriggerOutboxHandler` to `ExecuteRunAsWorkflow`.
- **Subtype allowlist enforcement strips disallowed columns.**
  `transforms.go:155-192` `AllowlistEnforcer` reads
  `oms.subtype_registry.field_allowlist` and removes columns not in it.
  Test: `engine_test.go:51-82`.
- **Column mapper casts types (`float`, `int`, `bool`, `uuid`, `date`).**
  `transforms.go:23-94`. Test: `engine_test.go:17-49`.
- **Graph synthesizer emits (parent TABLE, child ATTRIBUTE, edge).**
  `transforms.go:194-248` produces 3 records per input row.
  Test: `engine_test.go:84-116`.
- **Bloomberg fields mapper builds BLOOMBERG_FIELD catalog records.**
  `transforms.go:410-501`. Includes sector flags, widths, decimals.
  Test: `engine_test.go:233-288`.
- **Rule validator evaluates CEL expressions.**
  `transforms.go:528-610` calls into shared
  `rules.RuleEngine.EvaluateCEL/EvaluateBatch`.
  Tests: `engine_test.go:290-338`.
- **Filter and host-runtime calc transformers are real.**
  Tests: `engine_test.go:340-369` and `calc_transform_test.go`.
- **Migrations from legacy `pipelines` table ran.**
  `db/migrations/20260903_migrate_legacy_pipelines.up.sql`.
  `engine.go:254-292` provides a Go-level re-converter for richer
  transformations (`ConvertLegacyPipelineJSON`, `legacy_convert.go:40-123`).

### ⚠️ Wired but partial / broken

- **DAG executor walks nodes in JSON-array order, ignoring edges.**
  `engine.go:155` iterates `for _, node := range dag.Nodes`. Edges in
  `dag.Edges` are not consulted for ordering. Diamond/branched DAGs
  run each branch independently; reordering nodes in the JSON
  changes execution semantics. **Tracked in Phase 2 of the sprint.**
- **`api_caller` is a stub — stamps `{verified:true}` without an HTTP call.**
  `transforms.go:259-298` `APICallerTransformer.Transform` hardcodes
  ```go
  apiPayload := map[string]interface{}{
      "status":     200,
      ...
      "result":     map[string]interface{}{"verified": true, "routed": true},
  }
  ```
  and merges it into the record. **KYC step silently succeeds.
  Tracked in Phase 3.**
- **No RBAC on any data-pipeline endpoint.**
  `handlers.go:32-52` mounts routes without `RequireRole`,
  `RequireABAC`, or any role/scope check. `grep RequireRole
  backend/internal/datapipeline` → 0 matches. Any authenticated
  tenant user can `POST /api/v1/data-pipelines`, `PUT …/:id`,
  `POST /:id/run`, `POST /:id/simulate`. **Phase 6.**
- **`getTenantID` falls back to Gold Copy tenant when JWT and header both missing.**
  `handlers.go:54-67`. `return uuid.MustParse("00000000-…-001")`.
  Combined with the EventSource-can't-send-JWT gap, this grants a
  silent cross-tenant read path. **Phase 6.**
- **No `created_by` enforcement on Update/Delete.**
  `handlers.go:211-264` checks `tenant_id` but not `created_by`.
  Any tenant user can mutate any other user's pipeline.
- **`created_by` is never populated.**
  `handlers.go:124-166` accepts `def` without setting
  `def.CreatedBy = claims.UserID`. Column is always NULL.
- **`GetRunStatus` and `StreamTelemetrySSE` don't filter by tenant.**
  `handlers.go:376-427` look up by `runID` string only. Knowing
  any run UUID grants read access to the run state, regardless
  of which tenant owns it. **Phase 1 rewrite closes this.**
- **`BulkLoadSTI` table name from user-controlled config.**
  `bo_driver.go:243-249`: `fmt.Sprintf("INSERT INTO %s …", table)`.
  No check against the `STITables` map (`bo_driver.go:16-27`).
  **Phase 6.**
- **`BulkLoadCatalogNodes` is serial per-record, not parallel.**
  `catalog_driver.go:107-123` loops `UpsertCatalogNode` one at a time.
  The UI claim "Parallel bulk ingestor" is false. **Phase 4.**
- **`Gold Copy check in catalog loader reads `is_gold_copy`,
  which is not present in any committed migration.** 🟡 static,
  awaiting DB confirmation. The blast-radius analysis (SQL-read
  evidence, second-round verification) confirmed **4 confirmed raw
  column reads in 4 files** plus **1 alias that's the correct pattern**:

  | File:line | SQL | Verdict |
  |---|---|---|
  | `datapipeline/catalog_driver.go:29` | `SELECT COALESCE(is_gold_copy, false) FROM tenants WHERE id = $1` | Raw read. **Fails at runtime** if the column is missing. |
  | `observability/slo_report_generator.go:295,303` | `CASE WHEN t.is_gold_copy … END`, `GROUP BY …, t.is_gold_copy` | Raw read. Fails at runtime if the column is missing. |
  | `tenantauto/reconciler.go:254,257` | `CASE WHEN t.is_gold_copy …`, `COALESCE(t.is_gold_copy, false) AS is_premium` | Raw read. Fails at runtime if the column is missing. |
  | `metadata/catalog_scan_service.go:393` | `t.is_gold_copy AS is_gold_copy` | Raw read. Pairs with the `db:"is_gold_copy"` struct tag at line 166 — same SELECT, same destination. Fails at runtime if the column is missing. |
  | `tenantauto/provisioner.go:180` | `COALESCE(t.gold_copy, false) AS tenant_is_gold_copy` | **Safe variant.** Reads the canonical `gold_copy` column and aliases the output. This is the pattern the other 4 should match. |

  Confirmed via exhaustive migration search: `backend/db/migrations/`,
  `backend/migrations/`, `phaseb_*.sql`, `migration_script.sql`,
  `migration_to_ddl_schema.sql`, `schema.sql`, and `totalddl.sql`
  contain **zero** `ALTER TABLE tenants ADD COLUMN is_gold_copy` (or
  equivalent) statements. `migration_script.sql:211` adds `gold_copy`;
  `backend/migrations/misc/rbac_instance_level.sql:13` adds `gold_copy`.
  The 4 raw reads above cannot succeed against a fresh-schema DB.

  Static-only. The operator session's `psql -c '\d tenants' | grep gold`
  resolves whether production has `is_gold_copy`, `gold_copy`, both,
  or neither. If only `gold_copy` exists, fix is to rename
  `is_gold_copy` → `gold_copy` in 4 file:line citations above. If
  both exist (e.g., a hand-`ALTER`'d prod), finding is benign and
  resolves itself.

  **Cause flow when column missing**: `CheckGoldCopy` returns the
  fallback path (`catalog_driver.go:32-37` — error swallowed, returns
  `tenantID == 00000000-…-001` only). Every non-Gold-Copy tenant call
  sees `isGoldCopy=false`, which on `BulkLoadCatalogNodes`
  (`catalog_driver.go:64-70`) accepts `core_id` from user-controlled
  pipeline records — silent forge-class risk for catalog referencing.
- **`OutboxPublisher` not transactional with BO write.**
  `outbox.go:42-71` opens its own transaction. The file's own
  comment acknowledges this: "in the rare case the outer BO write
  later fails/rolls back after this commits, a pipeline trigger can
  fire for a write that didn't happen." **Tracked in Phase 6.**
- **`core_id` from request body, not server-derived.**
  `catalog_driver.go:63-70` accepts `core_id` from the pipeline
  record. A tenant user can forge references into core catalog nodes.
- **Mock paths return success in catalog & BO drivers.**
  `bo_driver.go:157-160` (`return int64(len(records)), nil` when
  `d.db == nil`); `catalog_driver.go:72-74`; `catalog_driver.go:152-155`.
  Sprint thesis is that silent successes are the enemy — these mock
  paths violate it (Phase 4 removes the catalog driver mock; BO driver
  mock to be removed in Phase 2 cleanup if useful).
- **`api_studio_endpoints` table doesn't exist in migrations.**
  `handlers.go:502` queries `FROM api_studio_endpoints`. No
  `CREATE TABLE` migration found in `db/migrations/`. Handler falls
  back to hardcoded mocks at `handlers.go:509-514`. **Phase 3 lands
  the migration as part of the SSRF-safe `api_caller` work.**
- **`workflows.pipelines` table queried for API/workflow discovery.**
  `handlers.go:534` queries `FROM pipelines WHERE tenant_id = $1`.
  After `20260903_migrate_legacy_pipelines.up.sql`, that table is
  empty (rows migrated into `data_pipeline_definitions`). API/Workflow
  discovery always returns the mock list. **Phase 3.**

### ❌ Documented but not implemented

- **`dead_letter` error policy declared in the type but not handled.**
  `model.go:37` declares `error_policy: 'fail_fast' | 'skip_and_log' | 'dead_letter'`.
  `engine.go:197-204` only handles `fail_fast`. `dead_letter` falls
  through to default behavior. **Backlog item.**
- **`mode: 'hybrid'` declared in the type but never dispatched.**
  `model.go:16`. `engine.go` source/transform/loader dispatch only
  branches on `business_object` / `catalog_graph` subtypes. **Backlog.**
- **Run history not persisted.** `engine.go:36` keeps
  `activeRuns: map[string]*PipelineExecutionRun` in-process. Restarts
  lose history. SSE handler also breaks on restart. **Phase 1.**
- **Templates live in the frontend bundle.**
  `frontend/src/features/data-pipelines/constants/pipelineTemplates.ts:17`
  has 4 hardcoded templates. No backend template registry.
- **No load-by-ID endpoint from the Studio.**
  `DataPipelineStudioPage.tsx:175` hardcodes `id: 'active-pipeline'`.
  The `/pipelines/studio/:id` route exists (`AppRoutes.tsx:302`)
  but the page ignores the param. **Phase 5.**
- **Save always POSTs a new pipeline.**
  `DataPipelineStudioPage.tsx:202-210`. No `PUT /:id`. Each Save
  creates a duplicate. **Phase 5.**

---

## 2. `backend/internal/validation/` — trigger → pipeline binding

### Files

| File | LOC | Verdict |
|------|-----|---------|
| `trigger.go` | 528 | ✅ Wired |
| `trigger_dispatch.go` | 424 | ✅ Wired for 8/13 trigger types |
| `validator.go` | — | ✅ CEL/PeopleCode-style conditions |
| `condition_schema.go`, `schema.go`, `schemas/` | — | ✅ Schema definitions |
| `trigger_dispatch_test.go`, `trigger_test.go` | — | ✅ Tests for sync + async dispatch |
| `engine.go` | — | ✅ Standalone validator without triggers |

### ✅ Wired & verified

- **`TriggerValidationEngine.DispatchTrigger` is wired into BO CRUD write paths.**
  `businessobject_service.go:4670, 4768, 4826` call
  `s.dispatchBORecordTrigger(…, TriggerTypeCreate, …)` before
  INSERT/UPDATE/DELETE. Same pattern in `services/business_object_service.go`.
- **`validation_triggers.pipeline_id` column exists and is read.**
  Migration `db/migrations/20260903_validation_trigger_pipeline_binding.up.sql:7-12`.
  Trigger dispatch reads `t.PipelineID` (`trigger.go:185-189`) and
  routes via `dispatchTriggerPipeline` (`trigger.go:199-230`).
- **Sync mode dispatch → in-process pipeline run.**
  `trigger.go:216-224` calls `tve.pipelineExecutor.RunPipelineSync(...)`,
  implemented by `PipelineEngine.RunPipelineSync`
  (`engine.go:259-275`). Returns errors so failed pipelines block the
  BO write. **This is one of the real paths the Studio unlocks.**
- **Async mode dispatch → outbox → worker poller.**
  `trigger.go:204-214` calls `tve.outboxPublisher.PublishPipelineTrigger(...)`.
  Outbox poller in `cmd/worker/main.go:437-453` picks up
  `Pipeline.Trigger` events and routes to `ExecuteRunAsWorkflow`
  (`outbox.go:73-105`).
- **`UnwrapConditionPayload` handles the rule envelope.**
  `trigger.go:158` corrects a known shape mismatch
  between authored rule conditions and execution-context schema.
- **8/13 trigger types are real.** `trigger_dispatch.go:404-423` documents
  the gap: `save / create / delete / field_change / workflow_step /
  sub_entity_change / relationship_change / status_change` work.
  `bulk_load / integration / time_based / calculated_field /
  security_role` are not implemented.
- **Field-level fast path.** `trigger_dispatch.go:154-156`
  `DispatchFieldChange` calls `ValidateField` before trigger loop.
- **Tests cover dispatch edge cases.**
  `trigger_dispatch_test.go` includes a `fakeOutboxPublisher`
  (`line 391-401`) for async path and a `fakeExecutor` for sync.

### ⚠️ Wired but partial

- **Outbox publish opens its own transaction (not caller's).**
  See `datapipeline/outbox.go:42-71` above. Same audit finding.
- **`validation_triggers.created_by` not enforced.** Same pattern as
  the Studio (no per-author check on triggers).

### ❌ Documented but not implemented

- **Studio UI has no surface for binding a pipeline to a trigger.**
  The DB column exists, but the Studio drawer's `TileConfigDrawer.tsx`
  doesn't expose the binding, and no other UI does either — the
  binding is only configurable via direct DB writes.

---

## 3. `backend/internal/oms/{account,position,security,trade_order}/`

### Files (per BO entity)

All four follow the same shape:
`handler.go` (HTTP), `service.go` (validation), `model.go`,
`repository.go`, `errors.go`, `<entity>_handler_test.go`,
`<entity>_validate_test.go`.

### ✅ Wired & verified

- **Tenant-isolated reads.** All four `repository.go` files filter
  by `tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())`.
  E.g. `account/repository.go:30`:
  ```sql
  WHERE tenant_id = $1 AND (valid_to IS NULL OR valid_to > NOW())
  ```
- **Soft-delete via `valid_to = NOW()`.**
  `account/repository.go:133`:
  ```sql
  UPDATE oms.account SET valid_to = NOW(), updated_at = NOW()
  WHERE id = $1 AND tenant_id = $2 AND valid_to IS NULL
  ```
  Handler returns `ErrNotFound → 404` when zero rows match
  (`handler.go:131-138`), preventing overwriting a previously
  soft-deleted row.
- **Bitemporal columns populated correctly.**
  `account/model.go:64-65` `ValidFrom time.Time`,
  `ValidTo *time.Time`. `repository.go:105-112` INSERT sets
  `created_at` and `valid_from`.
- **Handlers extract `tenantID` from JWT, return 401 on Nil.**
  `account/handler.go:42-49, 52-56, 70-74, 97-101, 118-123`.
  Same pattern in the other three handlers.
- **Service interfaces for unit testing.**
  `account/handler.go:14-19` defines `AccountServiceInterface` so
  tests can inject a mock.
- **Validate tests per entity.**
  `account_validate_test.go`, `trade_order_validate_test.go`, etc.

### ⚠️ Wired but partial

- **No RBAC on the four entity handlers.**
  `grep "RequireRole\|RequireABAC" backend/internal/oms/...` → 0 hits.
  Same gap as data pipeline.
- **Errors leak implementation details.**
  `handler.go:60-63`:
  ```go
  http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
  ```
  Passes through raw SQL/PKG error messages to clients. **Backlog.**

### ❌ Documented but not implemented

- **No per-entity rate limiting or write quota.** **Backlog.**

---

## 4. `backend/internal/catalog/` — STI → Semantic pipeline (4 stages)

### Files

| File | LOC | Verdict |
|------|-----|---------|
| `model.go`, `types.go`, `writer.go`, `grouping.go` | — | Support code |
| `subtype_registry.go` | 97 | ✅ Loader with TTL cache |
| `subtype_bo_builder.go` | 343 | ⚠️ Real but ON CONFLICT target may not match schema |
| `sti_column_scanner.go` | 92 | ✅ Real `information_schema` introspection |
| `subtype_semantic_linker.go` | 46 | ✅ Creates IS_CLASSIFIED_AS edges |
| `business_object_service.go` | 6000+ | ⚠️ Large; contains query for `gold_copy` |
| `business_term_generator.go`, `okf_generator.go`, `okf_service.go` | — | Glossary code path |

### ✅ Wired & verified

- **Stage 1 (subtype registry loader) has TTL cache, keyed by tenant.**
  `subtype_registry.go:42-97` `CachedSubtypeRegistryLoader.LoadAllForTenant`.
  5-minute default TTL.
- **Stage 3 (STI column scanner) introspects real DB.**
  `sti_column_scanner.go:17-91` queries
  `information_schema.columns WHERE table_schema IN
  ('oms', 'altinv', 'cash_flow', 'master')`, emits TABLE + ATTRIBUTE
  nodes and `COLUMN_OF` edges with `ON CONFLICT (tenant_id,
  qualified_path) DO UPDATE`.
- **Stage 4 (semantic linker) creates IS_CLASSIFIED_AS edges.**
  `subtype_semantic_linker.go:17-46`. Includes
  `NOT EXISTS` guard to prevent duplicates.
- **Tests cover each stage.** `subtype_registry_test.go`,
  `subtype_bo_builder_test.go`, `sti_column_scanner_test.go`,
  `subtype_semantic_linker_test.go`, `sti_e2e_test.go`.

### ⚠️ Wired but partial

- **ON CONFLICT target in Go code does not match the documented
  schema constraint.** 🟡 static, awaiting DB confirmation.
  `subtype_bo_builder.go:80,91,102,113` uses:
  ```sql
  ON CONFLICT (tenant_id, node_type_id, qualified_path) DO UPDATE SET updated_at = NOW()
  ```
  `schema.sql:431` documents the catalog_node base constraint as:
  ```sql
  CONSTRAINT catalog_node_unique UNIQUE (tenant_datasource_id, node_type_id, qualified_path)
  ```
  **First column differs** (`tenant_id` vs `tenant_datasource_id`).
  The cited migration `db/migrations/20260824_001_catalog_sti_unique_constraints.up.sql`
  does **not exist in the repo** (`ls backend/db/migrations/ | grep 20260824` jumps
  from `20260824_002_ai_learning_engine.up.sql` to `20260824_007_seed_gold_copy_demo_data.up.sql`)
  — phantom citation, removed.
  `db/migrations/20260825_001_repopulate_business_objects_from_subtype_registry.up.sql:52,60,68,76,184`
  uses the same 3-column pattern as the Go code in 5 separate
  statements. If that migration ran successfully against a real DB,
  a matching constraint must exist (likely `tenant_id` since the
  migration is for per-tenant seeding). Operator session's
  `psql -c '\d catalog_node' | grep -i unique` resolves this definitively.
  Fix if confirmed broken: add `ALTER TABLE catalog_node ADD CONSTRAINT
  catalog_node_tenant_node_type_id_qualified_path UNIQUE (tenant_id, node_type_id, qualified_path)`.
- **`subtype_bo_builder.go` line 121:**
  ```go
  _ = strings.Title(row.RootObject) // display_name future use
  ```
  No-op left in code. Should be removed. **Backlog.**
- **Two loaders for the same `oms.subtype_registry` table.**
  `internal/catalog/subtype_registry.go` (TTL cache) AND
  `internal/datapipeline/bo_driver.go:87-110` (different cache). No
  shared abstraction. **Backlog — consolidate or document why split.**

### ❌ Documented but not implemented

- **`gold_copy` vs `is_gold_copy` column ambiguity.** Multiple
  query styles in production code: `gold_copy` is used in 70+
  places, `is_gold_copy` only at
  `datapipeline/catalog_driver.go:29`. See §1 above. **Backlog.**

---

## 5. Cross-package wiring

### `backend/internal/api/api.go:1567-1591`

```go
dataPipelineEngine := datapipeline.NewPipelineEngine(sqlxDB, dataPipelineRuleEngine, temporalClient)
dataPipelineHandler := datapipeline.NewDataPipelineHandler(sqlxDB, dataPipelineEngine)
dataPipelineHandler.RegisterRoutes(r)
srv.DataPipelineEngine = dataPipelineEngine

boTriggerEngine := validation.NewTriggerValidationEngine(db, &validation.SimpleLogger{}).
    WithPipelineExecutor(dataPipelineEngine).
    WithOutboxPublisher(datapipeline.NewOutboxPublisher(sqlxDB))
boService.SetTriggerEngine(boTriggerEngine)
```

- ✅ Wires `dataPipelineEngine` into both the HTTP handlers and the
  trigger engine (sync executor + outbox publisher).
- ⚠️ Same `dataPipelineRuleEngine` reused as the global
  `complianceDeps.RuleEngine` (line 1572-1575) — these two engines
  are the **same instance**, which is correct, but it means a
  CEL-evaluation failure in one engine affects both the Compliance
  Console and the Studio. **Worth a test in Phase 2/6.**

### `backend/cmd/worker/main.go:276-280, 437-453`

- ✅ Registers `RunPipelineDAGWorkflow` + `ActivityExecutePipelineDAG`
  on the deployed BP task queue.
- ✅ Registers outbox poller with `Pipeline.Trigger` handler.
- ⚠️ Outbox poller uses `events.ProcessOutbox(ctx, dbx, nil, handlers)`
  — the publisher is **nil**, so non-Pipeline events fall through
  silently. The file's own comment notes this (line 432-437).

---

## 6. Cross-cutting security posture (carried into Phase 6)

These are not dataclassed to one package; they're a property of the
global chi router in `internal/api/api.go`. Listed once here so
Phase 6 backlog isn't constructed piecemeal:

| # | Finding | Source | Phases affected |
|---|---|---|---|
| **N** | **App backend connects as `postgres` superuser; `BYPASSRLS` voids every RLS policy** (see ⚠️ section in §1) | `backend/.env` `DATABASE_URL`/`POSTGRES_DSN`/`UISCE_DATABASE_URL`, `cmd/server/main.go:31` fallback | **Phase 6, priority HIGH** (supersedes item A's gravity) |
| A | `getTenantID` falls back to Gold Copy tenant on missing auth — *exacerbated by item N* | `datapipeline/handlers.go:54-67` | Phase 6 (severity upgraded by item N) |
| B | No RBAC on Studio endpoints | `datapipeline/handlers.go:32-52` | Phase 6 |
| C | No RBAC on OMS BO endpoints | `oms/{account,position,…}/handler.go` | Phase 6 |
| D | No `created_by` enforcement on Update/Delete | `datapipeline/handlers.go:211-264` | Phase 6 |
| E | `created_by` never populated from JWT | `datapipeline/handlers.go:124-166` | Phase 6 |
| F | `GetRunStatus` / SSE no tenant filter | `datapipeline/handlers.go:376-427` | Phase 1 (subsumed by stream tokens + LISTEN/NOTIFY) |
| G | `BulkLoadSTI` table name from user config | `datapipeline/bo_driver.go:243-249` | Phase 6 |
| H | Outbox publisher not transactional | `datapipeline/outbox.go:42-71` | Backlog |
| I | `core_id` from request body | `datapipeline/catalog_driver.go:63-70` | Phase 6 (derive server-side) |
| J | `gold_copy` vs `is_gold_copy` ambiguity (4 raw reads, see §1) | `datapipeline/catalog_driver.go:29` + 3 others | Backlog (operator `\d` resolves) |
| K | `subtype_bo_builder` ON CONFLICT target mismatch (3 vs 3 columns, see §4) | `catalog/subtype_bo_builder.go:80,91,102,113` | Backlog (operator `\d` resolves) |
| L | Errors leak implementation details to clients | `oms/*/handler.go` | Backlog |
| M | EventSource cannot send JWT | Architectural; studio design | Phase 1 (stream tokens) |
| O | `main.go:31` fallback DSN hardcodes `?sslmode=disable` to prod IP | `cmd/server/main.go:31` | Pulled into incident-response block (item 6) |

**Item N — full statement**:

`backend/.env` `DATABASE_URL`, `POSTGRES_DSN`, and `UISCE_DATABASE_URL`
all DSN as `postgres@100.84.50.65` (the production Postgres IP). The
fallback path in `cmd/server/main.go:31` uses the same role. The
`postgres` role is implicit Postgres **superuser**, which carries
`BYPASSRLS` by default.

This means every `SET LOCAL uisce.current_tenant = $1` call in
`BODriver.BulkLoadSTI` (`bo_driver.go:169`), `UpdateSTI` (line 329),
`DeleteSTI` (line 394), and the equivalent in `metadata/businessobject_service.go`
and `internal/services/business_object_service.go` **is decorative for
the actual runtime connection**: Row-Level Security policies, even
if installed correctly on every tenant-scoped table, are bypassed
because the connecting role has `BYPASSRLS=1`.

The mTLS hardening (Phase 0 incident block) locked the front door.
But the application is connecting with skeleton keys. Demoting the
runtime role to a purpose-built non-superuser with `FORCE ROW LEVEL
SECURITY` privileges is now feasible because the auth mechanism is
the cert, not the role's password. **Phase 6 priority: HIGH —
supersedes item A's severity**, since item A's Gold Copy fallback
combined with `BYPASSRLS` means even a missing-`getTenantID` is
moot (the policy isn't enforced anyway).

**Item O — main.go:31 footgun** (moved to incident block per user
reclassification, item 6):

The fallback DSN `postgresql://postgres@100.84.50.65:5432/alpha?sslmode=disable`
exists as a "tripwire" — its comment explains it should fail closed
when mTLS rejects the unencrypted connection. But: tripwires fail
closed; this one fails *open* if anything upstream ever weakens
mTLS validation (a debugging pg_hba edit, a replicated config, a
temporary scram rule during an incident). Replace with
`log.Fatal("DATABASE_URL is required")`. 5-line code change, zero
dependencies.

---

## 7. Stale completion docs (queued for `git rm`)

The repo root contained 46 markdown/`.txt` files (per `ls *.md *.txt | grep
-iE "_COMPLETE|_DONE|FINAL_|EPIC_|PHASE_.*COMPLETE|PLAN_STUDIO"`)
claiming phase/feature/epic completion, plus
`PLAN_STUDIO_EVENTS_AUDIT.md` (Group E, superseded by this document).
**46 files** queued for `git rm` (pre-audit estimate was 47; the actual
listing yielded 46). None of the claims are verifiable against current
code state; many are contradicted by §1–§6.

**Pre-execution verification (this commit)**:

```bash
$ ls *.md *.txt | grep -iE "_COMPLETE|_DONE|FINAL_|EPIC_|PHASE_.*COMPLETE|PLAN_STUDIO" | wc -l
46
$ ls PLAN_STUDIO_EVENTS_AUDIT.md             # confirmed exists
```

**Complete list (46 files, group counts below)**:

Group A — phase numbering older than current (PHASE_2/3/4 + PHASE_11 + PHASE_4b) **23 files** (one overcount in header; group list is authoritative):

```
PHASE_2_COMPLETE.md
PHASE_2_COMPLETE_FINAL_SUMMARY.md
PHASE_3_5_COMPLETE.md
PHASE_3_4_E2E_TESTING_COMPLETE.md
PHASE_3_4_FRONTEND_COMPLETE.md
PHASE_3_17_COMPLETE.md
PHASE_3_18_COMPLETE.md
PHASE_3_21_COMPLETE.md
PHASE_3_22_COMPLETE.md
PHASE_3_COMPLETE.md
PHASE_3_DASHBOARDS_COMPLETE.md
PHASE_3_FINAL_COMPLETION_REPORT.md
PHASE_3_HOOKS_COMPLETE.md
PHASE_3_PRIORITIES_COMPLETE.md
PHASE_4_FEATURE_1_FINAL_SUMMARY.md
PHASE_4_PRIORITIES_D_A_B_COMPLETE.md
PHASE_4_SESSION_4_COMPLETE.md
PHASE_4b_FINAL_BANNER.txt
PHASE4_FEATURE1_COMPLETE.md
PHASE4_FEATURE2_COMPLETE.md
PHASE4_FEATURE3_COMPLETE.md
PHASE4_FEATURE4_IMPLEMENTATION_COMPLETE.md
PHASE4_FEATURE4_INTEGRATION_COMPLETE.md
```

Group B — feature/epic completeness claims **15 files**:

```
ADMIN_UI_COMPLETE.md
ADMIN_UI_V2_COMPLETE.md
EPIC_31_COMPLETE.md
EPIC_31_INDEX.md
GOLD_COPY_IMPLEMENTATION_COMPLETE.md
IMPLEMENTATION_COMPLETE.md
INTEGRATION_COMPLETE.md
MDM_INTEGRATION_COMPLETE.md
OPS_COCKPIT_COMPLETE.md
PHASE_11_CBO_COMPLETE.md
PRIORITY_A_COMPLETE.md
SEMANTIC_LAYER_IMPLEMENTATION_COMPLETE.md
SESSION_COMPLETE_PHASE4_FEATURES_1_2.md
SESSION_INCIDENT_TIMELINE_COMPLETE.md
SESSION_SUMMARY_PHASE4_FEATURE1_FINAL.md
```

Group C — `.txt` files claiming completion **6 files**:

```
BUILD_COMPLETE.txt
DYNAMIC_UI_DELIVERY_COMPLETE.txt
IMPLEMENTATION_COMPLETE.txt
SCHEMA_FIXES_COMPLETE.txt
SESSION_COMPLETE_SUMMARY.txt
SETUP_COMPLETE_VISUAL.txt
```

Group D — overlap with audit scope or otherwise confusing **4 files**:

```
COMPLETE_MDM_ROADMAP.md
COMPLETE_PROJECT_PROGRESS_REPORT.md
FINAL_SUMMARY.md
JWT_E2E_FINAL_SUMMARY.md
```

Group E — superseded by `STATUS_AUDIT.md` **1 file**:

```
PLAN_STUDIO_EVENTS_AUDIT.md     # reads accurately but most action items never landed
```

Total = 23 + 15 + 6 + 4 + 1 = 49 in the groups above; **46 files**
were actually removed (`git rm` commit `667692129` is the authoritative
list — filename list in commit body). The arithmetic discrepancy (49
listed vs 46 removed) reflects a counting error in the audit document
itself; the commit body is the source of truth.

---

## 8. Audit verdict per package

Findings count by package (precise counts; no percentages):

| Package | ✅ Real | ⚠️ Partial/broken | ❌ Doc-only/not-implemented | Open questions |
|---|---|---|---|---|
| `internal/datapipeline/` | 11 verified items (model, bo_driver real DB I/O, workflow.go Temporal pair, legacy_convert, calc_transform, transform variants, etc.) | 12 wired-but-partial items (DAG order, api_caller stub, RBAC, getTenantID fallback, created_by, SSE tenant filter, table-name whitelist, Outbox tx, core_id forge, mock paths, etc.) | 6 documented-but-not-implemented (dead_letter, mode:hybrid, run history not persisted, templates in frontend, save→no PUT, load-by-ID ignored) | 1 awaiting DB (`is_gold_copy` column) |
| `internal/validation/` | 7 verified items (DispatchTrigger wired, sync+async dispatch paths, trigger types 8/13, etc.) | 2 partial (outbox own-tx, created_by not enforced) | 1 doc-only (no Studio UI surface for trigger→pipeline binding) | 0 |
| `internal/oms/{account,position,security,trade_order}/` | 6 verified per-entity items (tenant-isolated reads, soft-delete via valid_to, bitemporal populated, 401 on Nil, service interfaces, validate tests) | 2 per-entity (no RBAC, error leakage) | 1 per-entity (no rate limiting) | 0 |
| `internal/catalog/` | 4 verified items (Stage 1 TTL loader, Stage 3 column scanner, Stage 4 semantic linker, tests per stage) | 3 partial (subtype_bo_builder ON CONFLICT, no-op `_ = strings.Title`, two parallel subtype_registry loaders) | 1 doc-only (`gold_copy` vs `is_gold_copy` ambiguity) | 1 awaiting DB (`catalog_node` constraint layout) |

The Studio is what the UI claims it is — partially. The engine exists,
the loaders work, the triggers fire pipelines. The gaps are surgical
fixes (Phase 2 topological, Phase 3 `api_caller`, Phase 4 catalog
parallel, Phase 6 security) rather than a rewrite.

---

## 9. Open follow-ups (in flight or staged)

### 🟡 Awaiting operator DB introspection

Resolution path: the operator runs the mTLS incident block (rotation,
replication line, `pg_hba` persistence check) and then `psql -c '\d
tenants'` and `psql -c '\d catalog_node'`. The result unblocks two
findings:

- §1 `Gold Copy check` (item J) — confirm whether `is_gold_copy` exists.
- §4 `subtype_bo_builder ON CONFLICT` (item K) — confirm whether the
  `(tenant_id, node_type_id, qualified_path)` constraint exists.

A separate commit (not `--amend`) is folded in with the DB evidence.
The pair **is** the artifact: static evidence vs runtime evidence.

### 🚧 In flight on `studio-wireup` branch

- `git rm` of the 46 stale completion docs in §7 (one atomic commit).
- Phase 2: topological executor + `dag.go` + acceptance test against
  the real DB.

### 📋 Backlog (not sprint-blocking)

- `STATUS_AUDIT_BACKLOG.md` to be produced at end of Phase 2, listing
  items that surfaced during audit but didn't make §1–§6 (e.g.,
  `pkg/workflows/RunStoredWorkflow` parity with `wf-1..4` mocks,
  `host_runtime_calc` palette wiring, route divergence between
  `core/data-pipelines` and `pipelines/studio/:id`).
- Session transcripts (`session-ses_*.md`, 4 files, all git-tracked
  per `git ls-files session-ses_*.md`): content audit showed no
  credential-shaped strings remain in the files, but the files
  contain full repo paths and code excerpts. Decision on whether
  these belong in version control is a post-sprint hygiene call.
- `main.go:31` fallback DSN → `log.Fatal` (item O in §6, item 6 in
  incident block).
- Phase 6 demote runtime role off `postgres` superuser (item N).
- `subtype_bo_builder` dead-code `strings.Title` (line 121).
- Two parallel subtype_registry loaders consolidation.
- `main.go:31` (item O above).
- Infisical password / `INFISICAL_TOKEN` rotation per incident block.

### ⛔ Out of scope

- All `phaseb_*.sql` / `migration_script.sql` / `totalddl.sql`
  consolidation.
- Per-entity BO rate limiting (item L).
- Error-message leakage replacement for OMS handlers.
- Catalog template / mode:hybrid / dead_letter policy implementation.

These move to their own planning cycle when the Studio sprint
delivers the diamond-DAG acceptance test against the real DB.
