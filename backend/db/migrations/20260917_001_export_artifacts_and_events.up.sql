-- =============================================================================
-- 20260917_001_export_artifacts_and_events.up.sql
--
-- Phase 1: Export with Watermarking & Data Classification feature
--
-- Two tables (named to avoid collision with existing report_exports, job_exports):
--   export_artifacts     — export artifact registry
--   export_artifact_events — append-only audit trail
--
-- Design decisions documented inline:
--   - tenant_id stamping: caller's (exporter's) tenant at INSERT time
--   - watermark attribution: exporter unconditionally (export belongs to initiator)
--   - immutable post-INSERT: column-scoped UPDATE grant enforces
--   - NO ACTION FK: audit trail survives artifact row deletion
--   - sweeper tombstone model: row stays after TTL; expires_at guard in predicate C
--   - TTL policy set at INSERT by workflow: internal=90d, confidential=30d, restricted=7d,
--     public=NULL(no expiry) — pinned here for migration and tests
--
-- References (do not re-litigate):
--   §4 DDL header in docs/features/reports/exports-design.md
--   §5.3 Predicate C + sweeper row-mutation rule in docs/features/reports/exports-design.md
-- =============================================================================

-- ---------------------------------------------------------------------------
-- Table 1: export_artifacts
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS public.export_artifacts (
    id                        UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id               UUID NOT NULL REFERENCES public.report_templates(id),

    exporter_user_id          TEXT NOT NULL,   -- Keycloak user ID; globally unique across tenants

    tenant_id               UUID NOT NULL,   -- Exporter's (caller's) tenant_id at INSERT time;
                                              -- watermark attribution: export belongs to initiator

    status                 TEXT NOT NULL DEFAULT 'pending',
    --   'pending'   : row inserted, render not started
    --   'running'   : render in progress (workflow updates in same transaction)
    --   'completed' : artifact stored and downloadable
    --   'failed'    : render failed; artifact not stored

    format                 TEXT NOT NULL,   -- 'pdf' | 'xlsx'

    export_classification  TEXT NOT NULL DEFAULT 'internal',
    --   'public'       : no watermark, no access restrictions
    --   'internal'      : watermark applied, logged
    --   'confidential'  : watermark applied, logged, audit trail required
    --   'restricted'    : watermark applied, logged, audit trail, download count limited
    -- TTL policy (set at INSERT by workflow):
    --   'public'      -> expires_at = NULL  (no expiry)
    --   'internal'     -> expires_at = NOW() + INTERVAL '90 days'
    --   'confidential' -> expires_at = NOW() + INTERVAL '30 days'
    --   'restricted'  -> expires_at = NOW() + INTERVAL '7 days'

    storage_key            TEXT NOT NULL,   -- MinIO path: exports/{tenant_key}/{template_id}/{export_id}.{format}

    size_bytes             BIGINT NULL,
    rows_processed         INTEGER NULL,
    execution_time_ms       INTEGER NULL,

    watermark_text          TEXT NULL,     -- Rendered watermark string
    watermark_hash           TEXT NULL,     -- HMAC-SHA256 of identity block; see §7

    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at           TIMESTAMPTZ NULL,
    expires_at            TIMESTAMPTZ NULL  -- TTL; NULL = no expiry (set by workflow at INSERT)
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_ea_tenant_id_created_at
    ON public.export_artifacts(tenant_id, created_at);
CREATE INDEX IF NOT EXISTS idx_ea_exporter_user_id
    ON public.export_artifacts(exporter_user_id);
CREATE INDEX IF NOT EXISTS idx_ea_template_id
    ON public.export_artifacts(template_id);
CREATE INDEX IF NOT EXISTS idx_ea_status
    ON public.export_artifacts(status);

-- RLS
ALTER TABLE public.export_artifacts ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.export_artifacts FORCE ROW LEVEL SECURITY;

-- Tenant isolation: same policy shape as report_executions
-- USING: SELECT filter; WITH CHECK: INSERT filter (catches wrong-tenant INSERT bugs)
CREATE POLICY ea_tenant_isolation ON public.export_artifacts
    USING (tenant_id = current_setting('uisce.current_tenant', true)::UUID)
    WITH CHECK (tenant_id = current_setting('uisce.current_tenant', true)::UUID);

-- Grants
-- INSERT: API creates pending row
-- SELECT: listing and download via proxy predicate (Predicate C)
-- UPDATE: column-scoped to 7 rendering-state columns; storage_key/template_id/exporter_user_id/
--   tenant_id/format/created_at are immutable post-INSERT (enforced by grant, not just comment)
GRANT INSERT, SELECT ON public.export_artifacts TO app_user;
GRANT UPDATE (status, size_bytes, rows_processed, execution_time_ms,
              completed_at, watermark_text, watermark_hash)
    ON public.export_artifacts TO app_user;
REVOKE DELETE ON public.export_artifacts FROM PUBLIC;
REVOKE DELETE ON public.export_artifacts FROM app_user;  -- insert-only; soft-delete via expires_at

-- ---------------------------------------------------------------------------
-- Table 2: export_artifact_events
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS public.export_artifact_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    export_id      UUID NOT NULL REFERENCES public.export_artifacts(id) ON DELETE NO ACTION,
    tenant_id      UUID NOT NULL,   -- denormalized for RLS + CDC partitioning

    event           TEXT NOT NULL,
    --   'PENDING'     : artifact row inserted, render not started (writer 1)
    --   'RUNNING'     : render in progress (writer 2 — system:export-workflow)
    --   'COMPLETED'   : render succeeded, artifact stored (writer 2)
    --   'FAILED'      : render failed (writer 2)
    --   'DOWNLOADED'  : proxy download served (writer 3 — system:download-proxy)
    --   'EXPIRED'     : TTL reached, artifact purged; row kept as tombstone (writer 4)

    from_status     TEXT NULL,
    to_status       TEXT NOT NULL,

    actor_id        TEXT NOT NULL,
    --   Writer 1 (PersistExportRowActivity):    user_id of the export API caller (from JWT)
    --   Writer 2 (RenderArtifactActivity):     'system:export-workflow'
    --   Writer 3 (download proxy):             'system:download-proxy'
    --   Writer 4 (TTL sweeper):               'system:export-sweeper'

    detail          JSONB NULL,
    --   PENDING:     {}
    --   RUNNING:     {}
    --   COMPLETED:   {output_url, size_bytes, format, classification}
    --   FAILED:      {error_message}
    --   DOWNLOADED:  {download_count, user_agent}
    --   EXPIRED:     {}

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX IF NOT EXISTS idx_eae_export_id_created_at
    ON public.export_artifact_events(export_id, created_at);
CREATE INDEX IF NOT EXISTS idx_eae_export_created_at
    ON public.export_artifact_events(created_at);
CREATE INDEX IF NOT EXISTS idx_eae_tenant_id_created_at
    ON public.export_artifact_events(tenant_id, created_at);

-- RLS: FORCE to match report_executions posture
ALTER TABLE public.export_artifact_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE public.export_artifact_events FORCE ROW LEVEL SECURITY;

-- Tenant isolation: same policy shape as report_executions and report_execution_events
CREATE POLICY eae_tenant_isolation ON public.export_artifact_events
    USING (tenant_id = current_setting('uisce.current_tenant', true)::UUID)
    WITH CHECK (tenant_id = current_setting('uisce.current_tenant', true)::UUID);

-- Grant block
-- INSERT+SELECT only; UPDATE+DELETE revoked
GRANT INSERT, SELECT ON public.export_artifact_events TO app_user;
REVOKE UPDATE, DELETE ON public.export_artifact_events FROM PUBLIC;
REVOKE UPDATE, DELETE ON public.export_artifact_events FROM app_user;

DO $$ BEGIN
    RAISE NOTICE 'export_artifacts and export_artifact_events tables created';
    RAISE NOTICE '  export_artifacts indexes: idx_ea_tenant_id_created_at, idx_ea_exporter_user_id, idx_ea_template_id, idx_ea_status';
    RAISE NOTICE '  export_artifacts RLS: FORCE + tenant isolation policy + WITH CHECK';
    RAISE NOTICE '  export_artifacts grants: INSERT+SELECT to app_user; UPDATE (7 cols) to app_user; DELETE revoked';
    RAISE NOTICE '  export_artifact_events indexes: idx_eae_export_id_created_at, idx_eae_export_created_at, idx_eae_tenant_id_created_at';
    RAISE NOTICE '  export_artifact_events RLS: FORCE + tenant isolation policy + WITH CHECK';
    RAISE NOTICE '  export_artifact_events grants: INSERT+SELECT to app_user; UPDATE+DELETE revoked';
    RAISE NOTICE '  Sweeper: tombstone model — row kept after TTL; expires_at guard in predicate C';
    RAISE NOTICE '  TTL policy (set at INSERT by workflow): internal=90d, confidential=30d, restricted=7d, public=NULL';
END $$;
