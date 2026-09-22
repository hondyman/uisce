-- Extends data_explorer.saved_query with the report-builder-style
-- organizational features it was missing: folders, favorites, and
-- private/shared visibility, plus a lightweight schedule table.
--
-- Modeled directly on internal/reports' ReportFolder/ReportSchedule shape
-- (see internal/reports/folder_model.go, schedule_model.go) rather than
-- reusing those tables outright, because report_folders/report_schedules
-- are keyed to report_definitions/templates, not saved queries - a
-- generic item_type column would have made every existing report query
-- there conditional. Same tenant-isolation pattern as
-- 20261016_014_saved_query_rest_endpoint.up.sql: no RLS, enforced by
-- explicit WHERE tenant_id = $1 in application code (querybuilder package).

CREATE TABLE IF NOT EXISTS data_explorer.saved_query_folder (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id UUID NOT NULL,
    parent_id UUID REFERENCES data_explorer.saved_query_folder(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_de_saved_query_folder_tenant_user
    ON data_explorer.saved_query_folder(tenant_id, user_id, parent_id);

ALTER TABLE data_explorer.saved_query
    ADD COLUMN IF NOT EXISTS folder_id UUID REFERENCES data_explorer.saved_query_folder(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS is_favorite BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS visibility TEXT NOT NULL DEFAULT 'private' CHECK (visibility IN ('private', 'shared')),
    ADD COLUMN IF NOT EXISTS is_core BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS related_bo_ids TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS created_by TEXT;

CREATE INDEX IF NOT EXISTS idx_de_saved_query_tenant_folder
    ON data_explorer.saved_query(tenant_id, folder_id);
CREATE INDEX IF NOT EXISTS idx_de_saved_query_tenant_favorite
    ON data_explorer.saved_query(tenant_id, user_id, is_favorite) WHERE is_favorite;

-- Schedule metadata only (cron/frequency + who to notify). Actually
-- dispatching runs is a follow-up: it needs the same trigger/dispatcher
-- internal/reports/reconciler.go and temporal_executor.go use for report
-- schedules, generalized to run a saved query instead of rendering a
-- report template - out of scope for this migration, which only adds the
-- schedule the UI can create/edit/list.
CREATE TABLE IF NOT EXISTS data_explorer.saved_query_schedule (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    saved_query_id UUID NOT NULL REFERENCES data_explorer.saved_query(id) ON DELETE CASCADE,
    owner_id UUID NOT NULL,
    schedule_name TEXT NOT NULL,
    cron_expression TEXT NOT NULL,
    export_format TEXT NOT NULL DEFAULT 'csv',
    notification_channels JSONB NOT NULL DEFAULT '{}',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_de_saved_query_schedule_tenant_query
    ON data_explorer.saved_query_schedule(tenant_id, saved_query_id);

COMMENT ON TABLE data_explorer.saved_query_folder IS 'User-scoped folder tree for organizing saved queries, mirroring report_folders';
COMMENT ON TABLE data_explorer.saved_query_schedule IS 'Recurring-run metadata for a saved query; dispatching is not yet wired (see comment above)';
