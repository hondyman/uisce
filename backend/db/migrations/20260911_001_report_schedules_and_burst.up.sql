-- =============================================================================
-- 20260911_001_report_schedules_and_burst.up.sql
--
-- Creates the report scheduling, bursting, and exchange calendar tables.
-- report_schedules links to public.report_templates via FK ON DELETE CASCADE:
--   a schedule is meaningless without its template. The burst orchestrator
--   falls back to mock client IDs when report_definition_id is null — leaving
--   a live cron schedule after template deletion would execute against fake data.
--
-- NOTE: report_schedules did not previously exist on alpha.
-- The 20260822 migration was never applied; this migration supersedes it
-- with the FK constraint baked in at table creation (not added retroactively).
-- =============================================================================

-- 1. Exchange Calendars & Holiday Configurations
CREATE TABLE IF NOT EXISTS public.tenant_exchange_calendars (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    calendar_code VARCHAR(50)  NOT NULL,
    calendar_name TEXT         NOT NULL,
    timezone      VARCHAR(50)  NOT NULL DEFAULT 'America/New_York',
    is_active     BOOLEAN      DEFAULT TRUE,
    created_at    TIMESTAMPTZ  DEFAULT NOW(),
    CONSTRAINT uq_tenant_calendar UNIQUE (tenant_id, calendar_code)
);

CREATE TABLE IF NOT EXISTS public.tenant_calendar_holidays (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calendar_id      UUID NOT NULL REFERENCES public.tenant_exchange_calendars(id) ON DELETE CASCADE,
    holiday_date     DATE NOT NULL,
    holiday_name     TEXT NOT NULL,
    holiday_type     VARCHAR(50) DEFAULT 'FULL_CLOSE',
    early_close_time TIME,
    CONSTRAINT uq_calendar_holiday UNIQUE (calendar_id, holiday_date)
);

-- 2. Report Schedules
--    report_definition_id FK ON DELETE CASCADE: deleting a template cascades
--    to all its schedules. A dangling schedule with a null report_definition_id
--    would cause burst_orchestrator to fall back to mock execution -- unacceptable.
--    owner_id records the creating user for authorization enforcement.
--    deleted_at supports soft delete (is_active = false AND deleted_at = NOW()).
--    DESIGN DECISION: owner_id TEXT NOT NULL has no FK to app_user.
--    Consistent with report_templates.created_by_id (FK ON DELETE SET NULL — open backlog).
--    Rationale: schedules survive owner deletion to preserve the audit trail of past
--    report_executions (which reference schedule runs). An orphaned owner means only
--    a tenant admin can manage the schedule — acceptable and enforced in Phase 2 auth gate.
--    FK alignment with app_user is a follow-up (same cleanup as created_by_id orphaning).
CREATE TABLE IF NOT EXISTS public.report_schedules (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID         NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    report_definition_id  UUID         NOT NULL REFERENCES public.report_templates(id) ON DELETE CASCADE,
    owner_id              TEXT         NOT NULL,  -- no FK to app_user: see design decision comment above
    schedule_name         TEXT         NOT NULL,
    cron_expression       VARCHAR(100) NOT NULL,
    region                VARCHAR(50)  NOT NULL DEFAULT 'us-west',
    calendar_id           UUID         REFERENCES public.tenant_exchange_calendars(id),
    start_of_day_time     TIME         NOT NULL DEFAULT '08:00:00',
    unscheduled_behavior  VARCHAR(50)  DEFAULT 'SKIP',
    business_day_offset   INT          DEFAULT 0,
    burst_dimension       TEXT         NOT NULL DEFAULT 'client_id',
    export_format         VARCHAR(20)  NOT NULL DEFAULT 'PDF',
    notification_channels JSONB        DEFAULT '{"in_app": true, "email": false}'::jsonb,
    is_active             BOOLEAN      DEFAULT TRUE,
    deleted_at            TIMESTAMPTZ,
    last_run_at           TIMESTAMPTZ,
    next_run_at           TIMESTAMPTZ,
    created_at            TIMESTAMPTZ  DEFAULT NOW(),
    CONSTRAINT uq_tenant_schedule_name UNIQUE (tenant_id, report_definition_id, schedule_name)
);

CREATE INDEX IF NOT EXISTS idx_report_schedules_tenant
    ON public.report_schedules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_report_schedules_template
    ON public.report_schedules(report_definition_id);
CREATE INDEX IF NOT EXISTS idx_report_schedules_owner
    ON public.report_schedules(tenant_id, owner_id);
CREATE INDEX IF NOT EXISTS idx_report_schedules_active
    ON public.report_schedules(tenant_id, is_active)
    WHERE is_active = TRUE AND deleted_at IS NULL;

-- 3. Batch Run Tracking
CREATE TABLE IF NOT EXISTS public.report_burst_batches (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    schedule_id        UUID NOT NULL REFERENCES public.report_schedules(id) ON DELETE CASCADE,
    effective_date     DATE NOT NULL,
    total_clients      INT  DEFAULT 0,
    successful_renders INT  DEFAULT 0,
    failed_renders     INT  DEFAULT 0,
    status             VARCHAR(50) DEFAULT 'RUNNING',
    started_at         TIMESTAMPTZ DEFAULT NOW(),
    completed_at       TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_burst_batches_schedule
    ON public.report_burst_batches(schedule_id);
CREATE INDEX IF NOT EXISTS idx_burst_batches_tenant_status
    ON public.report_burst_batches(tenant_id, status);

-- 4. Per-Client Artifact Ledger
CREATE TABLE IF NOT EXISTS public.report_burst_artifacts (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    batch_id           UUID NOT NULL REFERENCES public.report_burst_batches(id) ON DELETE CASCADE,
    client_id          VARCHAR(100) NOT NULL,
    file_format        VARCHAR(20)  NOT NULL,
    storage_path       TEXT         NOT NULL,
    file_size_bytes    BIGINT,
    sha256_checksum    VARCHAR(64)  NOT NULL,
    render_duration_ms INT,
    status             VARCHAR(50)  DEFAULT 'READY',
    created_at         TIMESTAMPTZ  DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_burst_artifacts_lookup
    ON public.report_burst_artifacts(tenant_id, client_id, created_at DESC);

-- 5. report_executions: ensure columns exist (already present on alpha; IF NOT EXISTS no-ops)
ALTER TABLE public.report_executions
    ADD COLUMN IF NOT EXISTS output_format TEXT,
    ADD COLUMN IF NOT EXISTS requested_by  TEXT,
    ADD COLUMN IF NOT EXISTS lineage       JSONB;

-- 6. report_cache_metadata: composite index for tenant-scoped TTL lookups
--    (tenant_id + template_id columns confirmed present; UNIQUE(template_id, cache_key) already exists)
CREATE INDEX IF NOT EXISTS idx_rcm_tenant_template
    ON public.report_cache_metadata(tenant_id, template_id);

DO $$ BEGIN
    RAISE NOTICE 'report_schedules created with FK ON DELETE CASCADE to report_templates';
    RAISE NOTICE 'tenant_exchange_calendars, tenant_calendar_holidays created';
    RAISE NOTICE 'report_burst_batches, report_burst_artifacts created';
    RAISE NOTICE 'report_executions columns verified (IF NOT EXISTS)';
    RAISE NOTICE 'idx_rcm_tenant_template index created on report_cache_metadata';
END $$;
