-- 20260822_client_reporting_scheduler_and_bursting.sql

-- 1. Exchange Calendars & Holiday Configurations
CREATE TABLE IF NOT EXISTS public.tenant_exchange_calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    calendar_code VARCHAR(50) NOT NULL, -- NYSE, LSE, TSX, TARGET2
    calendar_name TEXT NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'America/New_York',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_calendar UNIQUE (tenant_id, calendar_code)
);

CREATE TABLE IF NOT EXISTS public.tenant_calendar_holidays (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calendar_id UUID NOT NULL REFERENCES public.tenant_exchange_calendars(id) ON DELETE CASCADE,
    holiday_date DATE NOT NULL,
    holiday_name TEXT NOT NULL,
    holiday_type VARCHAR(50) DEFAULT 'FULL_CLOSE', -- FULL_CLOSE, EARLY_CLOSE
    early_close_time TIME,
    CONSTRAINT uq_calendar_holiday UNIQUE (calendar_id, holiday_date)
);

-- 2. Report Schedules & Bursting Specifications
CREATE TABLE IF NOT EXISTS public.report_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    report_definition_id UUID,
    schedule_name TEXT NOT NULL,
    cron_expression VARCHAR(100) NOT NULL,
    region VARCHAR(50) NOT NULL DEFAULT 'us-west',
    calendar_id UUID REFERENCES public.tenant_exchange_calendars(id),
    start_of_day_time TIME NOT NULL DEFAULT '08:00:00',
    unscheduled_behavior VARCHAR(50) DEFAULT 'SKIP', -- SKIP, RUN_PREVIOUS_BUS_DAY, RUN_NEXT_BUS_DAY, WARN_HALT
    business_day_offset INT DEFAULT 0, -- -1 (T-1), 0 (T), +1 (T+1)
    burst_dimension TEXT NOT NULL DEFAULT 'client_id', -- Slicing field
    export_format VARCHAR(20) NOT NULL DEFAULT 'PDF', -- PDF, EXCEL, BOTH
    notification_channels JSONB DEFAULT '{"in_app": true, "email": false}'::jsonb,
    is_active BOOLEAN DEFAULT TRUE,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Batch Run Tracking & Client Artifact Ledger
CREATE TABLE IF NOT EXISTS public.report_burst_batches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    schedule_id UUID NOT NULL REFERENCES public.report_schedules(id) ON DELETE CASCADE,
    effective_date DATE NOT NULL,
    total_clients INT DEFAULT 0,
    successful_renders INT DEFAULT 0,
    failed_renders INT DEFAULT 0,
    status VARCHAR(50) DEFAULT 'RUNNING', -- RUNNING, COMPLETED, FAILED, PARTIAL
    started_at TIMESTAMPTZ DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS public.report_burst_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    batch_id UUID NOT NULL REFERENCES public.report_burst_batches(id) ON DELETE CASCADE,
    client_id VARCHAR(100) NOT NULL,
    file_format VARCHAR(20) NOT NULL,
    storage_path TEXT NOT NULL,
    file_size_bytes BIGINT,
    sha256_checksum VARCHAR(64) NOT NULL,
    render_duration_ms INT,
    status VARCHAR(50) DEFAULT 'READY',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_burst_artifacts_lookup 
ON public.report_burst_artifacts (tenant_id, client_id, created_at DESC);
