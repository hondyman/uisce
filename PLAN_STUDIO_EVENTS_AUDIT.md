# Plan: Page/API Studio Events → Expression Engine → Audit + Datalake

Branch: `feature/studio-events-audit-expression-engine` (created off baseline commit `65ad9f685`)

## What already exists (build on, don't rebuild)

| Capability | Location | Status |
|---|---|---|
| Expression engine (CEL, PeopleCode-equivalent) | `backend/internal/rulefabric/` (cel_compile.go, evaluator.go, vm_manager.go), `frontend/src/components/ExpressionBuilder/ExpressionBuilder.tsx`, `frontend/src/components/rulefabric/UnifiedRuleEditor.tsx` | Mature |
| Workflow triggering | `backend/temporal/` + `backend/internal/temporal/`, hook at `trigger_engine.go:473 startTemporalWorkflow` | Mature (hook exists, currently thin) |
| Audit pipeline (who/what/when/where, outbox, Kafka, Iceberg sink) | `backend/internal/audit/` (~35 files: service.go, event_logger.go, outbox_manager.go, iceberg_audit_service.go, explorer_*, compliance_reporter.go) | Mature — read `backend/internal/audit/README.md` + `INTEGRATION.md` before extending |
| Semantic layer | `backend/internal/analytics/`, `backend/internal/semantic*`, `backend/internal/boresolver/` | Mature |
| StarRocks client + pre-agg refresh | `backend/internal/calcengine/starrocks.go`, `backend/temporal/workflows/preagg_refresh_workflow.go` | Mature |
| Hot/cold tiering | `backend/internal/data_intelligence/tiering/` | Present, smaller — verify `watermark_router.go` exists (only its test was found; may be renamed/merged elsewhere) |
| BO row-event triggers (the template to copy) | `bo_crud_handler.go` emits `row_insert/update/delete` → `trigger_types` table → `trigger_engine.go` `EvaluateTriggers` | Functional, landed 2026-08-29 |

## The actual gap

**Page Studio and API Studio have no wiring from their save/submit events into RuleFabric/trigger_engine.** Everything downstream (expression eval, workflow kickoff, audit, semantic/StarRocks impact) already exists — it just isn't invoked from these two studios today.

## Trigger-type naming (mirrors `20260829_001_bo_row_event_trigger_types.up.sql` convention exactly)

Same table (`trigger_types`), same columns (`key`, `label`, `description`, `category`), new categories `page` and `api`:

```sql
INSERT INTO trigger_types (id, key, label, description, category)
VALUES
    (gen_random_uuid(), 'page_load',      'Page Loaded',       'Fires when a Page Studio page finishes loading.',              'page'),
    (gen_random_uuid(), 'page_save',      'Page Saved',        'Fires after a Page Studio page record is saved.',              'page'),
    (gen_random_uuid(), 'field_change',   'Field Changed',     'Fires when a field value changes on a Page Studio page.',      'page'),
    (gen_random_uuid(), 'api_request',    'API Request',       'Fires when an API Studio endpoint receives a request.',        'api'),
    (gen_random_uuid(), 'api_response',   'API Response',      'Fires before an API Studio endpoint returns a response.',      'api')
ON CONFLICT (key) DO NOTHING;
```

This lets `trigger_engine.go EvaluateTriggers` treat Page/API Studio events exactly like BO row events — same lookup, same evaluation path, no new engine.

## Datalake architecture (three tiers — clarified)

Not one Iceberg sink — three separate, purpose-built stores:

1. **Per-tenant OLAP datalake (Iceberg)** — one Iceberg database per tenant, structured for OLAP, fed from Postgres OLTP. This is the analytical/reporting copy. Existing `iceberg_audit_service.go`/`iceberg_sink.go` need to be checked: confirm whether they already partition per-tenant or currently write to a single shared Iceberg store — if shared, this is a real gap to fix (per-tenant isolation), not just a rename.
2. **Per-tenant audit/rollback datalake (Iceberg, OLTP-shaped)** — same schema shape as the tenant's Postgres OLTP tables (not OLAP-flattened), used to (a) roll back the hot DB if needed and (b) archive rows once they age past the tiering watermark, keeping the hot OLTP database lean. This is effectively the "cold" tier for `data_intelligence/tiering/`. Verify: does `iceberg_audit_service.go` already write in OLTP shape, or does it need a second sink alongside the OLAP one? Also resolve the missing `watermark_router.go` implementation here — this is the component that should move rows from hot Postgres → this cold Iceberg store at the watermark date.
3. **uisce metadata audit (separate from tenant data audits)** — a distinct audit trail for metadata changes only: both tenant-authored custom metadata (custom fields, BO defs, page/API Studio definitions, rule/expression definitions) and core uisce platform metadata changes. This is schema/definition-change history, not row-data history — likely belongs with the semantic layer's own migration/version tracking (`backend/internal/semantic*`, `backend/SEMANTIC_LAYER_IMPLEMENTATION.md`) rather than the row-audit path. Need to check whether this already exists as e.g. a metadata versioning table, or is a net-new audit stream.

Mapping back to the original request: "semantic layer audit datalake" = tier 1 (OLAP); "actual data amended... into tenant's audit datalake... hot DB... moved to cold data lake at watermark" = tier 2; the new ask just now = tier 3 (metadata-change audit, uisce + tenant-custom).

## Work items

1. **Studio event types** — land the `trigger_types` migration above; confirm `page`/`api` are acceptable new categories (existing categories besides `data` — check `trigger_types` table for any prior page/API rows before assuming these are net-new).
2. **Wire studio save/submit handlers** (`PageStudioPage.tsx` onSave, `APIStudioPage.tsx` onSave/handleSave) to emit these events through the same code path `bo_crud_handler.go` uses to reach `trigger_engine.go`.
3. **Expression builder surfacing in the two Studios** — reuse `ExpressionBuilder.tsx`/`UnifiedRuleEditor.tsx` as a panel/dialog in Page Studio and API Studio property panels.
4. **Tier-1 verification** — confirm/fix per-tenant Iceberg database isolation for the OLAP audit sink (`iceberg_audit_service.go`/`iceberg_sink.go`).
5. **Tier-2 build-out** — OLTP-shaped per-tenant rollback/archive Iceberg store; wire `watermark_router` (find or rebuild) to move hot Postgres rows there at the watermark date; confirm `bo_crud_handler.go`'s existing audit write already targets this shape and reuse verbatim for Studio-originated writes.
6. **Tier-3 build-out (new)** — separate metadata-change audit stream covering both tenant custom metadata and core uisce metadata edits; check `backend/internal/semantic*` and any existing schema-versioning tables first, extend rather than duplicate if something's already there.
7. **Audit interceptor for Studio expression execution** — extend `analytical_audit_interceptor.go` so every Studio-triggered expression evaluation logs who/what-expression/what-BO/SQL/performance into tier 1.
8. **StarRocks fan-out on calculation impact** — when a triggered expression affects a calculated/derived field, call the existing `preagg_refresh_workflow.go` / `ExecutePreAggregation` path.
9. **GSIFI-grade completeness check** — pass against `compliance_reporter.go` to confirm full who/what/when/where/before-after capture across all three tiers for regulated-tenant use; extend fields only where a genuine gap is found.

## Explicit non-goals

- No new expression engine, no new workflow engine, no new StarRocks client — all reused as-is.
- No single generic "datalake" — three distinct, purpose-built stores per above, not one abstraction.

## Investigation findings (confirmed against source)

**Q1 — Iceberg tenant isolation: GAP CONFIRMED.** `iceberg_audit_service.go` and `iceberg_sink.go` write all tenants into one shared bucket/catalog (`iceberg.audit.*`, `iceberg.platform.*` — hardcoded namespace literals) with `tenant_id` used only as a row column / S3 path prefix / partition key, never as a separate catalog, database, or bucket. This is row-level isolation, not store-level isolation.

**GSIFI verdict: this does not meet the isolation bar a GSIFI counterparty will require.** Recommended fix for tier 1 (and tier 2, same sink family): provision a **separate Iceberg catalog/database (and ideally separate bucket) per tenant**, not a shared table filtered by column. Concretely: `iceberg.audit_<tenant_id>.*` namespaces (or one bucket per tenant, e.g. `audit-logs-<tenant_id>`), driven by a per-tenant catalog config rather than a single global `AUDIT_BUCKET` env var. This is the single most consequential change in this plan — everything else is additive, this one is corrective.

**Q2 — Metadata versioning: partially exists, fragmented.** `semantic/version_store.go` versions semantic objects; `metadata_versioning_handlers.go` versions BO field changes only (`metadata_versions` table). Nothing exists for Page Studio, API Studio, or rule/expression definition changes. Tier 3 is a real gap for those three, but should **extend the existing `metadata_versions` pattern** (same table/shape, new `change_type` values) rather than invent a new mechanism — reuse the pattern, not a parallel one.

**Q3 — trigger_types: clear runway.** Only `row_insert/update/delete` under category `data` exist. `page`/`api` categories and the proposed keys are safely net-new, no collision.

**Q4 — Exception/AI analysis: a stub already exists.** `backend/internal/platform_intelligence/exceptions/aggregator.go` defines the right shape (`ExceptionType` enum incl. `tenant_anomaly`, `pii_exposure`, `data_quality`, etc.; `Exception{Source, Evidence, DetectedAt, Resolved}`) but `GetAllExceptions` is a hardcoded mock with no persistence and no AI hookup. Separately, `aso/anomaly_detection_service.go`, `services/anomaly_service.go`, and `scheduler_intelligence/ai/anomaly_detector.go` are disjoint anomaly detectors not wired to the aggregator. **Recommendation: make `exceptions/aggregator.go` real (persist to the tier-1 per-tenant Iceberg store, populate from the anomaly detectors that already exist) rather than building a new exception system.**

## GSIFI design stance: centralized vs. decentralized

- **Data plane must be decentralized per tenant** — all three audit tiers, once tier 1/2 are fixed per Q1, should be isolated per-tenant stores (separate catalog/bucket), not shared-table-with-filter. This is the corrective work above.
- **Control plane can stay centralized** — the expression engine (RuleFabric/CEL), Temporal workflow orchestration, trigger_engine.go, and the audit/versioning *code* should remain one shared, versioned, auditable codebase. GSIFIs generally prefer this: one certifiable code path beats N per-tenant forks.
- **Tier 3 (metadata audit) splits by ownership**: core uisce platform metadata changes → one central log (not tenant-confidential); tenant-custom metadata changes (their field defs, their business rules) → tenant-isolated, same as tiers 1/2, since to a GSIFI their business logic is itself sensitive/proprietary.
- **AI/exception analysis must run in-tenant-scope.** Wire the `exceptions/aggregator.go` fix to read only from that tenant's isolated Iceberg store. Any cross-tenant rollup (e.g. platform health dashboards) must be aggregated signals only (counts, categories, rates) — never raw event/payload content crossing tenant boundaries. Do not build a cross-tenant AI analysis feature that reads raw exception evidence from multiple tenants into one model context.

## Work items (revised, in priority order)

1. **Fix tier-1/tier-2 tenant isolation (blocking, do first)** — introduce per-tenant Iceberg catalog/bucket provisioning in `iceberg_audit_service.go`/`iceberg_sink.go`, replacing the single shared `AUDIT_BUCKET`/`iceberg.audit.*` namespace. This underlies everything else in the plan being GSIFI-acceptable.
2. **Land `trigger_types` migration** for `page`/`api` categories (`page_load`, `page_save`, `field_change`, `api_request`, `api_response`) — no naming collisions, safe to do independently of item 1.
3. **Wire Studio save/submit handlers** (`PageStudioPage.tsx`, `APIStudioPage.tsx`) into `trigger_engine.go` via the same path `bo_crud_handler.go` already uses.
4. **Surface expression builder in both Studios** — reuse `ExpressionBuilder.tsx`/`UnifiedRuleEditor.tsx` as-is.
5. **Extend `metadata_versions` pattern (tier 3)** to cover Page Studio, API Studio, and rule/expression definition changes, with a central/tenant split per the GSIFI stance above — reuse the existing table shape, add `change_type` values.
6. **Wire `watermark_router`** (missing implementation — build or locate) to move hot Postgres rows into the now-per-tenant tier-2 Iceberg store at the watermark date.
7. **Audit interceptor for Studio expression execution** into tier 1 (now per-tenant).
8. **StarRocks fan-out on calculation impact** via existing `preagg_refresh_workflow.go`.
9. **Make `exceptions/aggregator.go` real**, per-tenant scoped, backed by tier-1 storage and the existing anomaly detectors, as the foundation for the requested AI exception analysis — strictly in-tenant, cross-tenant only as aggregated signals.
10. **GSIFI completeness pass** against `compliance_reporter.go` across all three (now-isolated) tiers.

## Explicit non-goals

- No new expression engine, no new workflow engine, no new StarRocks client, no new exception-tracking data model — all reused/fixed in place.
- No cross-tenant raw-data AI analysis — isolation is a hard requirement, not a nice-to-have.
