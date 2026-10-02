-- Irreversible data drop. Recreate empty shells only so a down migration
-- does not leave the schema half-broken; do not expect row recovery.

CREATE TABLE IF NOT EXISTS public.report_schedules (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID         NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    report_definition_id  UUID         NOT NULL REFERENCES public.report_templates(id) ON DELETE CASCADE,
    owner_id              TEXT         NOT NULL,
    schedule_name         TEXT         NOT NULL,
    cron_expression       VARCHAR(100) NOT NULL,
    region                VARCHAR(50)  NOT NULL DEFAULT 'us-west',
    calendar_id           UUID,
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

CREATE TABLE IF NOT EXISTS public.report_burst_artifacts (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    batch_id         UUID NOT NULL REFERENCES public.report_burst_batches(id) ON DELETE CASCADE,
    client_id        TEXT NOT NULL,
    file_format      TEXT NOT NULL,
    storage_path     TEXT NOT NULL,
    sha256_checksum  TEXT NOT NULL,
    created_at       TIMESTAMPTZ DEFAULT NOW()
);

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'report_executions_schedule_id_fkey'
    ) THEN
        ALTER TABLE public.report_executions
            ADD CONSTRAINT report_executions_schedule_id_fkey
            FOREIGN KEY (schedule_id) REFERENCES public.report_schedules(id) ON DELETE SET NULL;
    END IF;
END $$;
