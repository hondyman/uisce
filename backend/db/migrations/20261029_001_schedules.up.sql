-- The one scheduler (internal/schedule): every schedule, whatever it runs
-- (report, saved query, data pipeline, ...), fired by Temporal Schedules and
-- recorded in one run history.
--
-- Tenant-keyed with RLS on uisce.current_tenant; the application also
-- filters on tenant_id explicitly.

CREATE TABLE IF NOT EXISTS public.schedules (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    name text NOT NULL,
    description text,
    -- What it runs: kind (report | saved_query | data_pipeline | ...) and
    -- the id of that thing in its own store.
    target_kind text NOT NULL,
    target_ref text NOT NULL,
    target_params jsonb NOT NULL DEFAULT '{}'::jsonb,
    -- When: a 5-field cron in an IANA time zone, optionally judged against a
    -- business calendar (mdm.calendar_master.calendar_cd).
    cron text NOT NULL,
    time_zone text NOT NULL DEFAULT 'UTC',
    calendar_cd text,
    calendar_rule text NOT NULL DEFAULT 'none'
        CHECK (calendar_rule IN ('none', 'skip', 'next_business_day', 'business_day_of_month')),
    business_day integer,
    start_at timestamptz,
    end_at timestamptz,
    enabled boolean NOT NULL DEFAULT true,
    -- Runs execute as the owner, in the datasource and region the schedule
    -- was created in.
    owner_id text NOT NULL,
    datasource_id text,
    region text,
    version integer NOT NULL DEFAULT 1,
    created_by text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_by text,
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz,
    CHECK (calendar_rule = 'none' OR calendar_cd IS NOT NULL),
    CHECK (calendar_rule <> 'business_day_of_month' OR (business_day BETWEEN -23 AND 23 AND business_day <> 0))
);
CREATE INDEX IF NOT EXISTS idx_schedules_tenant ON public.schedules (tenant_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_schedules_target ON public.schedules (tenant_id, target_kind, target_ref) WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS public.schedule_runs (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    schedule_id uuid NOT NULL REFERENCES public.schedules(id),
    target_kind text NOT NULL,
    target_ref text NOT NULL,
    trigger text NOT NULL CHECK (trigger IN ('schedule', 'manual')),
    triggered_by text,
    scheduled_for timestamptz NOT NULL,
    started_at timestamptz,
    finished_at timestamptz,
    status text NOT NULL CHECK (status IN ('running', 'succeeded', 'failed', 'skipped')),
    -- Why a firing did not run (calendar), in plain words.
    skip_reason text,
    -- The calendar decision the run was judged by (calendar, date, business
    -- day, tenant layer), for audit.
    calendar_decision jsonb,
    -- What the runner reports: rows, execution ids, output reference.
    outcome jsonb,
    -- Catalog code + params for users; detail is internal and never sent.
    error_code text,
    error_params jsonb,
    error_detail text,
    workflow_id text,
    workflow_run_id text
);
CREATE INDEX IF NOT EXISTS idx_schedule_runs_schedule ON public.schedule_runs (tenant_id, schedule_id, scheduled_for DESC);
CREATE INDEX IF NOT EXISTS idx_schedule_runs_recent ON public.schedule_runs (tenant_id, scheduled_for DESC);

-- Files a run produced (e.g. a saved query's CSV), served from the run.
CREATE TABLE IF NOT EXISTS public.schedule_run_outputs (
    run_id uuid PRIMARY KEY REFERENCES public.schedule_runs(id) ON DELETE CASCADE,
    tenant_id uuid NOT NULL,
    file_name text NOT NULL,
    content_type text NOT NULL,
    content bytea NOT NULL,
    row_count integer,
    created_at timestamptz NOT NULL DEFAULT now()
);

-- Every change to a schedule: who, when, before and after.
CREATE TABLE IF NOT EXISTS public.schedule_audit (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id uuid NOT NULL,
    schedule_id uuid NOT NULL,
    action text NOT NULL,
    actor text NOT NULL,
    at timestamptz NOT NULL DEFAULT now(),
    before jsonb,
    after jsonb
);
CREATE INDEX IF NOT EXISTS idx_schedule_audit_schedule ON public.schedule_audit (tenant_id, schedule_id, at DESC);

DO $$
DECLARE t text;
BEGIN
    FOREACH t IN ARRAY ARRAY['schedules', 'schedule_runs', 'schedule_run_outputs', 'schedule_audit'] LOOP
        EXECUTE format('ALTER TABLE public.%I ENABLE ROW LEVEL SECURITY', t);
        EXECUTE format('ALTER TABLE public.%I FORCE ROW LEVEL SECURITY', t);
        IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname = 'public' AND tablename = t AND policyname = 'tenant_isolation_policy') THEN
            EXECUTE format('CREATE POLICY tenant_isolation_policy ON public.%I USING (tenant_id::text = current_setting(''uisce.current_tenant'', true))', t);
        END IF;
        IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
            EXECUTE format('GRANT SELECT, INSERT, UPDATE, DELETE ON public.%I TO app_user', t);
        END IF;
    END LOOP;
END $$;

COMMENT ON TABLE public.schedules IS 'The one scheduler: every schedule, any target kind, fired by Temporal Schedules';
COMMENT ON TABLE public.schedule_runs IS 'Run history for every schedule (the Process Monitor)';
COMMENT ON COLUMN public.schedule_runs.error_detail IS 'Internal cause for operators; never sent to clients';
