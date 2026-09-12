# Export Feature Design — Reports with Watermarking & Data Classification

**Feature branch:** `feat/reports-exports` (cut from `main`)
**Design doc version:** 1 (assembly 2026-09-11)
**Status:** Design review — pending approval before Phase 1 implementation

---

## §1. Pre-flight Context

Seven features/phases shipped through evidence gates: personal/custom reports, private folders, scheduled reports, full-text search, Temporal orchestrator wiring, monitoring Phase 1 (audit trail) and Phase 2 (read paths) merged, Phase 3 admin surface (PR #66) in review.

### Relevant prior-art infrastructure

| Component | File | Notes |
|---|---|---|
| GenerateArtifactActivity | `internal/temporal/activities/report_activities.go:104` | Honest placeholder; `engine='temporal_workflow'` marker; fabricated `/artifacts/` paths; rendering deferred |
| `layout_config.sections` schema | `internal/reports/model.go:17` | Untyped `map[string]interface{}`; only `metadata` accessed; schema does not exist in code |
| Classification | Greenfield | No existing concept for export classification; unrelated analogues in `internal/reporting/` (q.v. §3.5) |
| Legacy EDM exports | `internal/handlers/export_handlers.go:168-247` | `/v1/exports/{exportId}/download` (streaming), `/v1/exports/{exportId}/download-url` (presigned URL); presigned-URL pattern is the door this feature closes |
| MinIO | Confirmed | `iceberg-warehouse` bucket; prefix convention uses `tenant_key` (string slug), NOT `tenant_id` (UUID) |
| Audit topic convention | Confirmed | `audit.<domain>.<noun>` |

---

## §2. Four PINNED Decisions (do not re-litigate)

1. **Rendering**: `excelize` for XLSX + `github.com/go-pdf/fpdf` (`engine='purego_pdf'` marker); Chromium-print-to-PDF is an explicit labeled follow-up, not smuggled scope.

2. **Legacy presigned-URL retirement**: Phase 3, separate commit/PR, sequence = deprecate → proxy-route → drop columns → grep-enforce repo-wide. Two doors to the walled garden: (a) EDM streaming + presigned-URL at `/v1/exports/...` and (b) semantic-reporting download at `/api/reporting/reports/instances/{id}/download` (q.v. §3.5 Amendment A).

3. **Sections schema**: minimal v1 JSON Schema (`{sections:[{kind:"table"|"metrics", title, source, view_id?}]}`), validated at template save, `{}` passes as default single-table layout (backward compatible).

4. **Storage path**: `exports/{tenant_key}/{template_id}/{export_id}.{pdf|xlsx}` — `tenant_key`, not `tenant_id`.

---

## §3. Five Clarifications (recorded)

1. **Watermark attribution** = the exporter, unconditionally. User A exports B's template → watermark shows A. Rationale: `requested_by` on the execution provenance already records the template owner's identity; watermark records the actor who initiated the export. These are intentionally distinct.

2. **`report_export_events`** = dedicated table (not extending execution events). Full discipline: insert-only, FORCE RLS + WITH CHECK, scoped grants, actor vocabulary mirrored from `report_execution_events` with cross-reference comments in both DDLs. New actors: `exporter_user_id`, `system:export-workflow`, `system:download-proxy`.

3. **Storage path tenant_key**: look up `tenant.key` via `tenants` table; fail if not found (no slug → no export path).

4. **Visible predicates locked and pasteable from** `report_handlers.go:788+` and `execution_repository.go:131-137, 181-186`.

5. **`glassbox.go:30` excluded from grep rule**: endpoint returns JSON metadata (fabricated `url`, `hash`, `signature`), not artifact bytes; no handler for `/download/bundles/{runID}.zip`; not a door.

---

## §3.5 Amendment A — Scope Addition: Semantic-Reporting Download Route (Scope Decision)

### Discovery

During pre-flight grep for "doors to the walled garden," the route `GET /api/reporting/reports/instances/{id}/download` at `internal/reporting/handler.go:423-473` was identified as a second door not enumerated in the handover's §2 PINNED decision #2. This section presents the evidence and proposes an amendment to PINNED decision #2 to include this route in the legacy retirement sequence.

### Evidence

**Route exists and is mounted.** `internal/reporting/handler.go:53` registers `r.Get("/{id}/download", h.DownloadInstance)`. The handler at lines 423-473 has two branches:
- Line 449-450: if `inst.OutputURL != ""` → `http.Redirect(w, r, inst.OutputURL, 302)` — **presigned-URL redirect pattern** (the exact door this feature exists to close).
- Line 455-468: else if `inst.OutputData != nil` → `w.Write(inst.OutputData)` — **direct byte streaming** pattern.

**Frontend calls this route today.** `frontend/src/hooks/useSemanticReporting.ts:255` calls `client.downloadInstance(instanceId)`, mounted in `useDownloadReport` which is consumed by `ReportViewer.tsx:75` and `ReportHistory.tsx:68`. The download button is live UI.

**The route currently returns 500 in alpha.** `backend/docs/DISCOVERY_UNAUTH_ROUTES.md` records: `GET /reports/instances/{id}/download` → 500 `pq: relation "report_instances" does not exist at column 1`. The table referenced in `internal/reporting/repository.go:373,401,433` is never created by any migration (the runner uses `backend/db/migrations/`, not `internal/reporting/migrations/`). So the route is live in the UI but broken in practice.

**The renderer behind it is a stub.** `internal/reporting/renderer.go:78-205` `renderPDF` writes `%PDF-1.4\n` to a `strings.Builder`. Comment at line 80: `// This would use a PDF library like gofpdf or unidoc`. No real PDF is produced. The `Watermark string` field in `PDFExportConfig` is never rendered.

### Amendment Proposal

**Amend PINNED decision #2** to include `GET /api/reporting/reports/instances/{id}/download` (both redirect and streaming branches) in the legacy retirement sequence. The route is live in the UI, currently broken (500), and behind a stub renderer — 410 is the correct response.

### Confirmation criterion (before Step B of the retirement sequence)

Step B (410) for this route should only proceed when confirmed dead by all three of:
1. **Zero writers**: grep `INSERT INTO report_instances` in `backend/internal/` returns no non-test Go files.
2. **Zero frontend call sites**: grep `downloadInstance` in `frontend/src/` returns zero results.
3. **Zero E2E references**: grep `instances.*download|/reports/instances` in `e2e/` specs returns zero results.

These are pasted verbatim from the codebase at Amendment approval time. If any check fails, the route stays live until the caller is retired. The three-confirmation rule prevents closing a route that is still referenced by live code.

**Current state at design assembly**: Criterion #2 (zero frontend call sites) is **failed** — `frontend/src/hooks/useSemanticReporting.ts:255` calls `client.downloadInstance(instanceId)`, mounted in `useDownloadReport` consumed by `ReportViewer.tsx:75` and `ReportHistory.tsx:68`. The download button is live UI. This means Step B for this route is blocked until Phase 4 rewires the button. The Phase 4 frontend task must include retiring the semantic-reporting download wire as a prerequisite to the Step B 410.

---

## §4. Artifact 1 — Phase 1 DDL

### Legacy classification vocabulary — disposition

The grep in pre-flight found `data_classification` in `internal/reporting/migrations/002_analytics_collaboration.sql` and `Watermark` in `internal/reporting/collaboration.go:587` and `model.go:454`. The call-path evidence shows:

- **`internal/reporting/migrations/` is never loaded by the migration runner.** `backend/internal/migrations/runner.go:50` sets `migrationsDir := "db/migrations"` (top-level `backend/db/migrations/`). The `internal/reporting/migrations/` directory is ignored. The `data_classification VARCHAR(50)` column on `report_audit_log` and `data_classifications VARCHAR(50)[]` on `report_data_masking_rules` are dead DDL — never applied to the live schema.

- **Go struct fields are real but never read for rendering.** `reporting/security.go:61 DataClassification string`, `reporting/collaboration.go:587 Watermark bool`, `reporting/model.go:454 Watermark string` — these are live struct fields on live types (`AuditEvent`, `ShareConfig`, `PDFExportConfig`). No code path writes them to an output document or reads them at render time. The stub renderer at `renderer.go:78-205` never references them.

- **`internal/reporting/handler.go:449-450`** (`http.Redirect(w, r, inst.OutputURL, 302)`) is a live path for the semantic-reporting door (q.v. §3.5 Amendment A).

- **The legacy `internal/reporting/` Renderer is out of scope for this feature.** The export feature is the new path for rendering reports, not a retrofit. The retirement of the semantic-reporting download route is the only intersection (Amendment A).

**Naming decision for the new column**: use `export_classification` rather than `data_classification` to avoid ambiguity with the dead DDL column name. The classification is a property of the export artifact, not the underlying data.

### DDL — `report_export_events`

```sql
-- =============================================================================
-- report_export_events — append-only audit trail for the export feature (Phase 1)
--
-- Design discipline mirrors report_execution_events (20260913_002_create_report_execution_events.up.sql):
--   - actor_id vocabulary: defined per writer in the column comment
--   - tenant_id denormalized: required for RLS + CDC partitioning
--   - FORCE ROW LEVEL SECURITY: matches report_executions posture
--   - WITH CHECK on inserts: catches writer bugs that set wrong tenant_id
--   - Insert-only enforcement: app role gets INSERT+SELECT; PUBLIC revoked
--   - ON DELETE NO ACTION: export row deletion requires explicit event-table handling;
--     CASCADE would bypass the insert-only audit trail via a single DELETE
--   - System-writer RLS: all writers use withTenantTx pattern (q.v. system-writer RLS story below)
--
-- Sibling table: public.report_execution_events (monitoring Phase 1)
--   Shared actor vocabulary: 'system:executor', 'system:activity', 'system:sweep'
--   New actors introduced here: 'system:export-workflow', 'system:download-proxy'
-- =============================================================================

CREATE TABLE IF NOT EXISTS public.report_export_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    export_id      UUID NOT NULL REFERENCES public.report_exports(id) ON DELETE NO ACTION,
    tenant_id      UUID NOT NULL,

    -- Event vocabulary
    event           TEXT NOT NULL,
    --   'PENDING'       : export row inserted, render not started (writer 1)
    --   'RUNNING'       : render in progress (writer 2 — system:export-workflow)
    --   'COMPLETED'     : render succeeded, artifact stored (writer 2)
    --   'FAILED'        : render failed (writer 2)
    --   'DOWNLOADED'    : proxy download served (writer 3 — system:download-proxy)
    --   'EXPIRED'       : TTL reached, artifact purged (writer 4 — background sweeper)

    from_status     TEXT NULL,
    to_status       TEXT NOT NULL,

    -- actor_id vocabulary (by writer):
    --   Writer 1 (PersistExportRowActivity):   user_id of the export API caller (from JWT / context)
    --   Writer 2 (RenderArtifactActivity):    'system:export-workflow'
    --   Writer 3 (download proxy handler):     'system:download-proxy'
    --   Writer 4 (TTL sweeper):                'system:export-sweeper'
    -- Cross-reference: report_execution_events actor vocabulary at report_execution_events.actor_id comments
    actor_id        TEXT NOT NULL,

    detail          JSONB NULL,
    --   PENDING:    {}
    --   RUNNING:    {}
    --   COMPLETED:  {output_url, size_bytes, format, classification}
    --   FAILED:     {error_message}
    --   DOWNLOADED: {download_count, user_agent}
    --   EXPIRED:   {}

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for common query shapes
CREATE INDEX IF NOT EXISTS idx_ree_export_id_created_at
    ON public.report_export_events(export_id, created_at);
CREATE INDEX IF NOT EXISTS idx_ree_export_created_at
    ON public.report_export_events(created_at);
CREATE INDEX IF NOT EXISTS idx_ree_tenant_id_created_at
    ON public.report_export_events(tenant_id, created_at);

-- RLS: FORCE to match report_executions posture
ALTER TABLE public.report_export_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.report_export_events FORCE ROW LEVEL SECURITY;

-- Tenant isolation: same policy shape as report_executions and report_execution_events
-- USING clause: governs SELECT (read filter)
-- WITH CHECK clause: governs INSERT (write filter — catches writer bugs setting wrong tenant_id)
CREATE POLICY ree_tenant_isolation ON public.report_export_events
    USING (tenant_id = current_setting('uisce.current_tenant', true)::UUID)
    WITH CHECK (tenant_id = current_setting('uisce.current_tenant', true)::UUID);

-- Grant block
GRANT INSERT, SELECT ON public.report_export_events TO app_user;
REVOKE UPDATE, DELETE ON public.report_export_events FROM PUBLIC;
REVOKE UPDATE, DELETE ON public.report_export_events FROM app_user;

DO $$ BEGIN
    RAISE NOTICE 'report_export_events table created';
    RAISE NOTICE '  indexes: idx_ree_export_id_created_at, idx_ree_export_created_at, idx_ree_tenant_id_created_at';
    RAISE NOTICE '  RLS: FORCE ROW LEVEL SECURITY with tenant isolation policy + WITH CHECK';
    RAISE NOTICE '  grants: INSERT+SELECT to app_user; UPDATE+DELETE revoked from PUBLIC and app_user';
    RAISE NOTICE 'Sibling: public.report_execution_events (monitoring Phase 1) — shared actor vocabulary';
END $$;
```

### System-writer RLS story

Every writer to `report_export_events` must set `uisce.current_tenant` to the export row's tenant_id inside the transaction, before any INSERT. This is the same pattern used by monitoring Phase 1's `system:executor` on `report_execution_events`. The verbatim pattern from `internal/temporal/activities/report_activities.go:149-152, 209-218` (monitoring Phase 1):

```go
// RLS INVARIANT: All writes are wrapped in db.WithTenantTransaction using
// template.TenantID, so set_config('uisce.current_tenant', tenantID, true) is
// set transaction-locally before any INSERT/UPDATE, satisfying the FORCE ROW
// LEVEL SECURITY policy on report_executions.
//
// withTenantTx opens a transaction, sets the tenant GUC transaction-locally
// via set_config (equivalent to SET LOCAL uisce.current_tenant = tenantID),
// and executes fn within that transaction.
//
// This satisfies the FORCE ROW LEVEL SECURITY policy:
//   policy: ((tenant_id)::text = current_setting('uisce.current_tenant', true))
//   relforcerowsecurity: true
//
// NOTE: set_config with is_local=true reverts the GUC at transaction end.
func withTenantTx(ctx context.Context, db *sql.DB, tenantID string, fn func(*sql.Tx) error) error {
    if tenantID == "" {
        return fmt.Errorf("withTenantTx: tenantID cannot be empty")
    }
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return fmt.Errorf("withTenantTx: BeginTx: %w", err)
    }
    defer func() {
        if p := recover(); p != nil {
            _ = tx.Rollback()
            panic(p)
        }
    }()
    if _, err := tx.ExecContext(ctx, "SELECT set_config('uisce.current_tenant', $1, true)", tenantID); err != nil {
        _ = tx.Rollback()
        return fmt.Errorf("withTenantTx: set_config failed: %w", err)
    }
    if err := fn(tx); err != nil {
        _ = tx.Rollback()
        return err
    }
    return tx.Commit()
}```

For this feature's writers:

- **Writer 1** (`PersistExportRowActivity`): calls `withTenantTx(ctx, db, callerTenantID, ...)` — uses the exporter's (caller's) tenant ID, which is already resolved from the API caller's JWT context.
- **Writer 2** (`RenderArtifactActivity`): calls `withTenantTx(ctx, db, callerTenantID, ...)` — same caller tenant as writer 1, because the export row is stamped with the caller's tenant. The workflow does not need a separate tenant context.
- **Writer 3** (`download proxy`): calls `withTenantTx(ctx, db, exporterTenantID, ...)` — the exporter's tenant from the export row, resolved from the `report_exports` row before the event INSERT.
- **Writer 4** (TTL sweeper): calls `withTenantTx(ctx, db, exportRowTenantID, ...)` — resolved from the export row before the event INSERT.

All four writers set `uisce.current_tenant` inside the transaction before INSERT, satisfying the WITH CHECK policy.

### DDL — `report_exports`

```sql
-- =============================================================================
-- report_exports — export artifact registry for the export feature (Phase 1)
--
-- tenant_id stamping rule: the API caller's (exporter's) tenant_id at export creation time.
-- The watermark attribution rule (the exporter unconditionally) is implemented as: the export
-- belongs to the actor who initiated it, so the export row is stamped with the caller's
-- tenant. This makes the export row RLS-visible to the caller immediately (INSERT passes RLS).
-- Predicate C Clause A (self-download) passes RLS naturally (tenant matches session).
-- Predicate C Clause B (same-template visibility) is fail-closed for gold-copy cross-tenant:
-- a user in tenant B who can see tenant A's gold-copy template cannot re-download tenant A's
-- existing export of it — RLS filters the row before the predicate evaluates.
-- This asymmetry is documented in §5.3 and is the intended behavior.
--
-- report_executions comparison: report_executions stamps the TEMPLATE'S tenant
-- (tmpl.TenantID, not the caller's), and grants caller access via the
-- 'triggered_by' column in the predicate. Exports have no 'triggered_by' — the exporter
-- IS the initiator, so caller-tenant stamping is the correct model. The two tables
-- use different stamping strategies. The cross-reference comment in the prior draft
-- was incorrect — this note replaces it.
--
-- User-ID uniqueness: exporter_user_id must be globally unique (Keycloak mandate).
-- =============================================================================

CREATE TABLE IF NOT EXISTS public.report_exports (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id         UUID NOT NULL REFERENCES public.report_templates(id),
    exporter_user_id    TEXT NOT NULL,  -- Keycloak user ID; globally unique across tenants

    tenant_id           UUID NOT NULL,  -- Exporter's (caller's) tenant_id at INSERT time
    -- The tenant_id is stamped from the API caller's session tenant at export creation time.
    -- This makes the row RLS-accessible to the caller immediately. The watermark attribution
    -- rule (export belongs to the actor who initiated it) is implemented as: the export row's
    -- tenant is the exporter's tenant.

    status              TEXT NOT NULL DEFAULT 'pending',
    --   'pending'   : row inserted, render not started
    --   'running'   : render in progress (workflow updates status in same transaction)
    --   'completed' : artifact stored and downloadable
    --   'failed'    : render failed; artifact not stored

    format              TEXT NOT NULL,  -- 'pdf' | 'xlsx'

    -- Classification — the export artifact's sensitivity label
    -- Named export_classification (not data_classification) to avoid ambiguity with the
    -- dead DDL column in internal/reporting/migrations/002_analytics_collaboration.sql
    export_classification  TEXT NOT NULL DEFAULT 'internal',
    --   'public'      : no watermark, no access restrictions beyond normal auth
    --   'internal'    : watermark applied, logged in export events
    --   'confidential': watermark applied, logged, audit trail required
    --   'restricted'  : watermark applied, logged, audit trail, download count limited

    -- Storage
    storage_key         TEXT NOT NULL,  -- MinIO path: exports/{tenant_key}/{template_id}/{export_id}.{format}

    size_bytes          BIGINT NULL,
    rows_processed      INTEGER NULL,
    execution_time_ms   INTEGER NULL,

    -- Watermark content (populated at render time, immutable after)
    watermark_text       TEXT NULL,     -- Rendered watermark string
    watermark_hash       TEXT NULL,     -- HMAC-SHA256 of identity block; see §7 for algorithm

    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at        TIMESTAMPTZ NULL,
    expires_at          TIMESTAMPTZ NULL  -- TTL; NULL = no expiry
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_re_tenant_id_created_at
    ON public.report_exports(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_re_exporter_user_id
    ON public.report_exports(exporter_user_id);
CREATE INDEX IF NOT EXISTS idx_re_template_id
    ON public.report_exports(template_id);
CREATE INDEX IF NOT EXISTS idx_re_status
    ON public.report_exports(status);

-- RLS
ALTER TABLE public.report_exports ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.report_exports FORCE ROW LEVEL SECURITY;

-- Tenant isolation: same policy as report_executions
-- USING: SELECT filter; WITH CHECK: INSERT filter (catches wrong-tenant INSERT bugs)
CREATE POLICY re_tenant_isolation ON public.report_exports
    USING (tenant_id = current_setting('uisce.current_tenant', true)::UUID)
    WITH CHECK (tenant_id = current_setting('uisce.current_tenant', true)::UUID);

-- Grants
-- app_user: INSERT (API creates pending row) + SELECT (listing/downloads via proxy predicate)
-- UPDATE to app_user is column-scoped to enforce the "all other columns immutable after INSERT"
-- promise in the table COMMENT. The workflow (temporal executor, runs as app_user) updates only
-- the rendering-state columns. RLS additionally scopes UPDATE to own-tenant rows.
--
-- Immutable post-INSERT (no UPDATE grant): storage_key, template_id, exporter_user_id,
--   tenant_id, format, created_at. storage_key is deterministic
--   (`exports/{tenant_key}/{template_id}/{export_id}.{format}`) and set at INSERT — workflow
--   never needs to mutate it. Including it in UPDATE would let an attacker who can run SQL as
--   app_user rewrite row provenance (e.g. swap storage_key to another tenant's path on an own-tenant
--   row, then download through the fully-authorized proxy); excluding it closes that path.
GRANT INSERT, SELECT ON public.report_exports TO app_user;
GRANT UPDATE (status, size_bytes, rows_processed, execution_time_ms,
              completed_at, watermark_text, watermark_hash)
    ON public.report_exports TO app_user;
REVOKE DELETE ON public.report_exports FROM PUBLIC;
REVOKE DELETE ON public.report_exports FROM app_user;  -- insert-only; soft-delete via expires_at

-- Cross-reference comment
COMMENT ON TABLE public.report_exports IS 'Export artifact registry. tenant_id stamped from exporter (caller) at creation. INSERT-then-mutate (status/size_bytes/completed_at updated by workflow). All other columns immutable after INSERT.';
COMMENT ON COLUMN public.report_exports.tenant_id IS 'Exporter (caller) tenant_id at INSERT time; watermark attribution implemented as export belongs to initiator. NOT the template tenant (differs from report_executions stamping model).';
COMMENT ON COLUMN public.report_exports.exporter_user_id IS 'Keycloak user ID; globally unique — cross-tenant safety for Predicate C Clause A depends on this.';
```

### Rollback-atomicity test shape

For each writer to `report_export_events`:

```go
// Example for writer 1 (PersistExportRowActivity)
func TestPersistExportRowActivity_RollbackAtomicity(t *testing.T) {
    db, err := sql.Open("postgres", testDSN)
    require.NoError(t, err)
    defer db.Close()

    tx, err := db.BeginTx(ctx, nil)
    require.NoError(t, err)
    defer tx.Rollback()

    // Insert export row and event in same transaction
    _, err = tx.ExecContext(ctx, `
        INSERT INTO report_exports (id, template_id, exporter_user_id, tenant_id, status, format, storage_key, export_classification)
        VALUES ($1, $2, $3, $4, 'pending', 'pdf', 'exports/test/t1/e1.pdf', 'internal')
    `, uuid.New(), templateID, userID, tenantID)
    require.NoError(t, err)

    _, err = tx.ExecContext(ctx, `
        INSERT INTO report_export_events (export_id, tenant_id, event, from_status, to_status, actor_id, detail)
        VALUES ($1, $2, 'PENDING', NULL, 'pending', $3, '{}')
    `, exportID, tenantID, userID)
    require.NoError(t, err)

    tx.Rollback()

    // Assert no rows in either table
    var exportCount, eventCount int
    err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM report_exports WHERE id = $1", exportID).Scan(&exportCount)
    require.NoError(t, err)
    err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM report_export_events WHERE export_id = $1", exportID).Scan(&eventCount)
    require.NoError(t, err)
    require.Equal(t, 0, exportCount)
    require.Equal(t, 0, eventCount)
}
```

---

## §5. Artifact 2 — Three Predicates Verbatim

### §5.1 — Predicate A: Template listing for fresh exports

**Source:** `backend/internal/reports/repository.go:230-245`

```sql
SELECT t.id, t.tenant_id, t.template_name, t.description, t.category,
       t.layout_config, t.parameter_schema,
       t.is_active, t.is_public, t.is_personal, t.created_by_id, t.created_by,
       t.created_at, t.updated_at, t.version,
       (f.template_id IS NOT NULL) AS is_favorite
FROM report_templates t
LEFT JOIN report_favorites f
       ON f.template_id = t.id
      AND f.tenant_id = $2
      AND f.user_id = $1
WHERE t.tenant_id IN ($2, $3)
  AND t.is_active = true
  AND (t.is_personal = false OR t.created_by_id = $1)
ORDER BY t.template_name
```

Bindings: `$1 = callerUserID, $2 = callerTenantID, $3 = goldCopyID`. Source comments at `repository.go:220-222` describe the personal-template carve-out and the gold-copy inheritance.

### §5.2 — Predicate B (list) and B' (single): Execution visibility for run-exports

**Source:** `backend/internal/reports/execution_repository.go:122-139`

```sql
SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status,
       e.parameters, e.output_url, e.output_size_bytes, e.rows_processed,
       e.execution_time_ms, e.error_message, e.workflow_id, e.run_id,
       e.requested_by, e.triggered_by, e.metadata, e.created_at, e.completed_at,
       t.is_personal, t.created_by_id
FROM public.report_executions e
JOIN public.report_templates t ON t.id = e.template_id
WHERE ($1::timestamptz IS NULL OR (e.created_at, e.id) < ($1, $2))
  AND (e.tenant_id = $3 OR e.triggered_by = $4)
  AND (
      t.is_personal = false
      OR (t.created_by_id IS NOT NULL AND t.created_by_id = $4)
      OR $5 = true
  )
ORDER BY e.created_at DESC, e.id DESC
LIMIT $6
```

Bindings: `$1 = cursorCreatedAt, $2 = cursorID, $3 = callerTenantID, $4 = userID, $5 = isAdmin, $6 = limit`.

**Source:** `backend/internal/reports/execution_repository.go:172-187`

```sql
SELECT e.id, e.tenant_id, e.template_id, e.schedule_id, e.report_key, e.status,
       e.parameters, e.output_url, e.output_size_bytes, e.rows_processed,
       e.execution_time_ms, e.error_message, e.workflow_id, e.run_id,
       e.requested_by, e.triggered_by, e.metadata, e.created_at, e.completed_at,
       t.is_personal, t.created_by_id
FROM public.report_executions e
JOIN public.report_templates t ON t.id = e.template_id
WHERE e.id = $1
  AND (e.tenant_id = $2 OR e.triggered_by = $3)
  AND (
      t.is_personal = false
      OR (t.created_by_id IS NOT NULL AND t.created_by_id = $3)
      OR $4 = true
  )
```

Bindings: `$1 = execID, $2 = callerTenantID, $3 = userID, $4 = isAdmin`.

### §5.3 — Predicate C: Export lookup for the download proxy

**Derived from** Predicates A and B; composition is explicit here.

```sql
-- export_lookup_predicate (download proxy)
-- $1 = callerUserID, $2 = callerTenantID, $3 = isAdmin, $4 = goldCopyID, $5 = exportID
SELECT e.*
FROM public.report_exports e
JOIN public.report_templates t ON t.id = e.template_id
WHERE e.id = $5
  AND (
      -- Clause A: caller exported it themselves (exporter is on the row)
      -- Safety assumption: exporter_user_id is globally unique (Keycloak mandate).
      -- If this assumption breaks, Clause A grants cross-tenant access.
      e.exporter_user_id = $1
      OR
      -- Clause B: template is visible to caller (composes with Predicate A's visibility shape)
      -- The JOIN on t.id = e.template_id reuses the template-listing predicate's shape.
      (
          t.tenant_id IN ($2, $4)
          AND t.is_active = true
          AND (t.is_personal = false OR t.created_by_id = $1)
      )
      OR $3 = true  -- global admin bypass, matches Predicate B/B' pattern
  )
  AND e.status = 'completed'  -- no in-flight artifact reads
  AND (e.expires_at IS NULL OR e.expires_at > NOW())  -- TTL guard: sweeper sets expires_at;
                                                     -- status stays 'completed'; row remains
                                                     -- visible in listings but proxy rejects.
```

**Composition notes:**
- Clause A uses only `exporter_user_id` — no tenant guard. This is consistent with `triggered_by` in Predicate B, which also carries no tenant assumption. Cross-tenant safety for Clause A relies on Keycloak user-ID global uniqueness (stated in DDL comment on `report_exports.exporter_user_id`).
- Clause B reuses Predicate A's visibility shape verbatim via the JOIN on `t.id = e.template_id`.
- `$4 = goldCopyID` is the resolved gold-copy tenant ID; pass `uuid.Nil` if resolution fails (same fallback pattern as `repository.go:224-228`).
- `e.status = 'completed'` is a new guard not present in B/B' — necessary because the proxy must not serve partial artifacts from an in-flight render.

**Fail-closed asymmetry from caller-tenant stamping:** because `report_exports.tenant_id` is stamped from the caller's (exporter's) session tenant (not the template's tenant), RLS filters the export row before the predicate evaluates for cross-tenant gold-copy downloads. Specifically: a user in tenant B who can see tenant A's gold-copy template cannot re-download tenant A's existing export of it — RLS removes the row (tenant B session ≠ tenant A export row), Predicate C is never evaluated. The user can re-export from the gold-copy template and download their own export. This asymmetry is intentional and documented as fail-closed behavior in the DDL header.

**Sweeper row-mutation rule:** the TTL sweeper (Writer 4 in §4) queries for rows where `expires_at < NOW()` **and** no prior `EXPIRED` event exists for that export (`NOT EXISTS (SELECT 1 FROM report_export_events ev WHERE ev.export_id = e.id AND ev.event = 'EXPIRED')`). For each match it: purges the MinIO artifact (idempotent delete-if-exists); emits one `EXPIRED` event; **leaves the row in place as a tombstone** (`status` stays `completed'`, `expires_at` stays in the past). The sweeper does not delete the export row and does not modify `expires_at`. Tombstoning is safe because Predicate C's `expires_at` guard (`expires_at IS NULL OR expires_at > NOW()`) blocks the proxy — the artifact bytes are gone from MinIO, the predicate refuses download, and the row remains as a visible, auditable record.

**TTL policy (set at INSERT time, not by the sweeper):** `expires_at` is assigned when the export row is created, based on `export_classification`. The policy values are a Phase 1 implementation decision (example: `internal` → 90 days, `confidential` → 30 days, `restricted` → 7 days, `public` → NULL/no expiry). The workflow sets `expires_at = NOW() + classification_ttl_days` in the same INSERT statement as the export row. The sweeper only enforces rows where the TTL has already passed — it is judge of enforcement, not setter of policy. This is pinned here because the migration and the born-complete tests will need the policy values.

---

## §6. Artifact 3 — Sections v1 JSON Schema

### Schema file: `backend/schemas/report_sections_v1.json`

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "$id": "https://uisce.example.com/schemas/report_sections_v1.json",
  "title": "Report Layout Sections v1",
  "description": "Minimal v1 schema for report layout sections. Empty object {} defaults to single-table layout.",
  "type": "object",
  "properties": {
    "sections": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["kind", "title", "source"],
        "properties": {
          "kind": {
            "type": "string",
            "enum": ["table", "metrics"],
            "description": "table: tabular data grid. metrics: KPI cards with aggregates."
          },
          "title": {
            "type": "string",
            "description": "Human-readable section title."
          },
          "source": {
            "type": "string",
            "description": "Semantic view ID or SQL view name providing data for this section."
          },
          "view_id": {
            "type": "string",
            "format": "uuid",
            "description": "Optional. Mutually exclusive with source: exactly one of view_id or source must be present."
          }
        },
        "additionalProperties": false,
        "oneOf": [
          { "required": ["source"], "properties": { "view_id": { "type": "null" } } },
          { "required": ["view_id"], "properties": { "source": { "type": "string", "maxLength": 0 } } }
        ]
      },
      "default": []
    }
  },
  "additionalProperties": false,
  "minProperties": 0,
  "maxProperties": 1
}
```

### Validation placement

Validated at **template save time** in the service layer. Read `ReportService.UpdateTemplate` and `CreateTemplate` to pin the exact call site. The `//go:embed` directive loads the schema:

```go
//go:embed ../schemas/report_sections_v1.json
var sectionsSchema []byte

func ValidateSections(raw json.RawMessage) error {
    if len(raw) == 0 || string(raw) == "{}" {
        return nil  // empty {} is valid — defaults to single-table layout
    }
    var v any
    if err := json.Unmarshal(raw, &v); err != nil {
        return fmt.Errorf("invalid JSON: %w", err)
    }
    // jsonschema.Validate bytes against embedded schema
    ...
}
```

Backward compatibility: `{}` validates as valid — the renderer treats an empty `sections` array as a single default table section.

---

## §7. Artifact 4 — Watermark Composition Spec + Rendered-Fixture Tests

### Identity block contents

Every rendered export carries an identity block. **PDF**: rendered in the page footer using fpdf's built-in footer function. **XLSX**: rendered as a **visible frozen row at the top of each sheet** (readable on screen, by `excelize`'s cell reader, and by the rendered-fixture test). Excel's print header/footer is NOT used — it is print-only and invisible to the cell reader. Pin: the renderer places the identity block as row 1 (frozen pane) in each sheet, with the label:value pairs in columns A–F.

```
Tenant: {tenant_name}
Exported by: {exporter_display_name} ({exporter_user_id})
Export ID: {export_id}
Classification: {classification_label}
Timestamp: {timestamp_utc}
Watermark hash: {watermark_hash}
```

### `watermark_hash` definition

**Algorithm**: HMAC-SHA256 over the concatenation of the following fields (pipe-separated, UTF-8):

```
{export_id}|{exporter_user_id}|{export_classification}|{created_at_unix}
```

**Key source**: `EXPORT_WATERMARK_SECRET` environment variable. If unset, the secret is derived from `API_TOKEN_ENCRYPTION_KEY` (a per-deployment secret already managed). **Key reuse is deliberate**: `API_TOKEN_ENCRYPTION_KEY` is used for this additional purpose to avoid operational overhead of a second secret. If `API_TOKEN_ENCRYPTION_KEY` is rotated, the watermark hash of all existing exports becomes unverifiable — this is explicitly acceptable for the stated purpose (tamper-evidence for support diagnosis, not cryptographic signature). If neither is available, rendering fails with a clear error — no silent fallback to an unkeyed hash.

**Implementation**:
```go
func ComputeWatermarkHash(exportID, exporterUserID, classification string, createdAt time.Time, secret []byte) string {
    input := fmt.Sprintf("%s|%s|%s|%d", exportID, exporterUserID, classification, createdAt.Unix())
    h := hmac.New(sha256.New, secret)
    h.Write([]byte(input))
    return hex.EncodeToString(h.Sum(nil))
}
```

**Purpose**: tamper-evidence for support diagnosis. Not a cryptographic signature (no public-key component); not a MAC that can be verified by a third party. The hash proves the export was generated from this system at this time; it cannot be forged without the server-side secret.

### Rendered-fixture test shape

**PDF**: `TestRender_PDF_IdentityBlockEveryPage`
1. Generate an export with fixed inputs: known `export_id`, `exporter_user_id`, `classification = "internal"`, `created_at = 2026-09-11T00:00:00Z`.
2. Extract text from the generated PDF using `github.com/ledongthuc/pdf`.
3. Assert the identity block (tenant name, exporter display name, export ID, classification, timestamp) appears in every page's footer.
4. Assert `watermark_hash` field is present and matches the expected HMAC-SHA256.

**Fallback** if `ledongthuc/pdf` is unmaintained at `go get` time: extract text from the PDF content stream using a pure-Go regex over `Tj`/`TJ` operators in the test binary itself (zero external deps, deterministic, only parses fixtures this repo generates).

**XLSX**: `TestRender_XLSX_IdentityBlockEverySheet`
1. Generate an XLSX export with the same fixed inputs.
2. Read back using `excelize`'s built-in cell reader — specifically read row 1 of each sheet (the frozen identity row).
3. Assert the identity block (tenant name, exporter display name, export ID, classification, timestamp) appears in every sheet's row 1.
4. Assert `watermark_hash` matches.

**Run-first gate**: both tests run on this Mac before any commit. Green output pasted in the Phase 2 PR body.

### Dependencies

| Library | Purpose | Pin |
|---|---|---|
| `github.com/go-pdf/fpdf` | PDF generation (`engine='purego_pdf'`) | Pin latest tagged release; verify < 12 months old at `go get` |
| `github.com/qax-os/excelize` | XLSX generation (`engine='excelize'`) | Note: `360EntSecGroup-Skylar/excelize` is archived; `qax-os/excelize` is the maintained fork |
| `github.com/ledongthuc/pdf` | PDF text extraction in tests | Primary; fallback: in-test content-stream extractor |

---

## §8. Artifact 5 — Proxy Security Spec + Legacy-Retirement Sequence

### New proxy route

**Route**: `GET /api/v1/reports/exports/{exportId}/download`
**Handler file**: `backend/internal/api/export_download_proxy.go` (new)

**Predicate**: Predicate C (§5.3)
**Audit-write-before-serve**: before streaming the artifact, write a `DOWNLOADED` event to `report_export_events` (actor = `system:download-proxy`, detail = `{download_count, user_agent}`). If audit write fails → 500 with `"audit log write failed: ..."` (mirrors `admin_report_handlers.go:130-140`).
**Zero-existence leak**: predicate returns no rows → 404 (not 403). Matches existing `GetExecution` behavior (`report_handlers.go:812`).

### Legacy-retirement 4-step sequence

**Step A** (deploy proxy; both doors still serve):
- Deploy `GET /api/v1/reports/exports/{exportId}/download` alongside existing handlers.
- Legacy: `GET /v1/exports/{exportId}/download` and `POST /v1/exports/{exportId}/download-url` and `GET /api/reporting/reports/instances/{id}/download` still serve.

**Step B** (410 both doors):
For each legacy door, return `410 Gone` with `Link: </api/v1/reports/exports/{exportId}/download>; rel="successor-version"`:

- `GET /v1/exports/{exportId}/download` (`internal/handlers/export_handlers.go:168-201`) → 410.
- `POST /v1/exports/{exportId}/download-url` (`internal/handlers/export_handlers.go:204-247`) → 410.
- `GET /api/reporting/reports/instances/{id}/download` (`internal/reporting/handler.go:423-473`, both branches) → 410. **Precondition for this route**: confirm dead per §3.5 Amendment A's three-confirmation criterion (zero writers, zero frontend call sites, zero E2E references, each pasted verbatim at Step B time).

**Step C** (drop dead columns and fields):
After 1+ deploy cycle with both doors 410:

- Drop `presigned_url` and `presigned_url_expires` columns from `edm.job_exports`. Migration file: `backend/db/migrations/<date>_drop_edm_export_presigned_columns.up.sql`.
- Drop `OutputURL` and `OutputData` fields from `internal/reporting/cache.go:392` and `internal/reporting/workday_report_service.go:179` if the §3.5 Amendment A three-confirmation criterion is met.
- **Test-inclusive sweep at Step C**: before committing the drop migration, run `grep -rn 'presigned_url\|OutputURL\|OutputData' backend/ --include="*.go"` (no test excludes). Any match is a compile-time break that the migration will cause — fix before committing the drop.

**Step D** (grep-enforce):
Add to `Makefile`:
```makefile
.PHONY: lint-export-proxy-grep
lint-export-proxy-grep:
	@bash scripts/lint_export_proxy_grep.sh
```

`scripts/lint_export_proxy_grep.sh`:
```bash
#!/usr/bin/env bash
set -euo pipefail
# Patterns that indicate a "door to the walled garden" outside the export proxy.
# Update this list when a new door is intentionally opened (with review).
forbidden=(
  'PresignedURL'
  'presigned_url'
  'GetDownloadURL'
  'http\.Redirect.*OutputURL'
  'w\.Write\(inst\.OutputData\)'
)
excludes=(
  '--glob=!*_test.go'
  '--glob=!*migrations*'
  '--glob=!*docs*'
)
hits=0
for pat in "$${forbidden[@]}"; do
  if rg -n "$${pat}" backend/ "$${excludes[@]}" >/dev/null; then
    echo "FORBIDDEN PATTERN: $${pat}"
    hits=$$((hits+1))
  fi
done
if [ "$${hits}" -gt 0 ]; then
  echo "lint-export-proxy-grep: $${hits} forbidden pattern(s) found"
  exit 1
fi
echo "lint-export-proxy-grep: clean"
```

**Named gap**: this `make` target has no CI invocation today. Framed as a Phase 4 task to wire into CI before the next deploy — matching the §3 "a gate that can't run is 'not run'" rule. Failures ledger entry added.

---

## §9. Artifact 6 — Export Workflow Temporal Params

**New workflow**: `ReportExportWorkflow`

### Activity graph

```
ReportExportWorkflow
  └─ RenderArtifactActivity       (excelize + fpdf, uploads to MinIO)
       └─ PersistExportRowActivity (writes report_exports + report_export_events in same tx)
       └─ SignalExecutionActivity  (links export to originating execution row if template_id is set)
```

### Pinned timeouts and retries (mirrors `internal/temporal/workflows/report_workflows.go:118`)

```go
// RenderArtifactActivity
activityOptions := temporal.StartToCloseTimeout(5 * time.Minute)
retryPolicy := temporal.RetryPolicy{
    MaximumAttempts:        3,
    BackoffCoefficient:      2.0,
    InitialInterval:         10 * time.Second,
    MaximumInterval:         5 * time.Minute,
}

// PersistExportRowActivity — same tx, short
persistOptions := temporal.StartToCloseTimeout(30 * time.Second)
persistRetry := temporal.RetryPolicy{
    MaximumAttempts: 3,
    BackoffCoefficient: 2.0,
}

// SignalExecutionActivity — fire-and-forget, can be async
signalOptions := temporal.StartToCloseTimeout(10 * time.Second)
```

All values are identical to the execution workflow's pinned params so behavior is predictable and the operations team has a single reference config.

---

## §10. Implementation Phases

| Phase | Content | Gate |
|---|---|---|
| 1 | DDL + export_classification column (DEFAULT 'internal', no backfill) + export events table + born-complete tests | Tests pass on this Mac |
| 2 | Renderer: excelize + fpdf + ledongthuc/pdf; rendered-fixture tests as gate | Run-first on this Mac; paste output in PR |
| 3 | Workflow + endpoints + download proxy + legacy retirement Steps A–D | Step D grep passes; legacy doors 410 |
| 4 | Frontend + E2E | Run-first |
