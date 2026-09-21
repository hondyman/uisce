-- Data pipeline run history persistence
-- Enables run history, reconnects for completed runs, and observability

CREATE TABLE IF NOT EXISTS public.data_pipeline_runs (
    id              UUID PRIMARY KEY,
    tenant_id       UUID NOT NULL,
    pipeline_id     UUID NOT NULL,
    status          TEXT NOT NULL DEFAULT 'queued',
    start_time      TIMESTAMPTZ NOT NULL,
    end_time        TIMESTAMPTZ,
    total_records_in   BIGINT DEFAULT 0,
    total_records_out  BIGINT DEFAULT 0,
    total_errors       BIGINT DEFAULT 0,
    peak_throughput_rows_sec FLOAT8 DEFAULT 0,
    step_order       JSONB DEFAULT '[]',
    error_details    JSONB DEFAULT '[]',
    dag_json         JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_data_pipeline_runs_tenant_id ON public.data_pipeline_runs(tenant_id);
CREATE INDEX IF NOT EXISTS idx_data_pipeline_runs_pipeline_id ON public.data_pipeline_runs(pipeline_id);
CREATE INDEX IF NOT EXISTS idx_data_pipeline_runs_status ON public.data_pipeline_runs(status);
CREATE INDEX IF NOT EXISTS idx_data_pipeline_runs_start_time ON public.data_pipeline_runs(start_time DESC);

CREATE TABLE IF NOT EXISTS public.data_pipeline_step_telemetry (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id              UUID NOT NULL REFERENCES public.data_pipeline_runs(id) ON DELETE CASCADE,
    node_id             TEXT NOT NULL,
    node_label          TEXT NOT NULL,
    node_type           TEXT NOT NULL,
    records_in          BIGINT DEFAULT 0,
    records_out         BIGINT DEFAULT 0,
    records_error       BIGINT DEFAULT 0,
    bytes_processed     BIGINT DEFAULT 0,
    duration_ms         BIGINT DEFAULT 0,
    rows_per_sec        FLOAT8 DEFAULT 0,
    status              TEXT NOT NULL,
    error_message       TEXT,
    step_order_index    INT NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(run_id, node_id)
);

CREATE INDEX IF NOT EXISTS idx_data_pipeline_step_telemetry_run_id ON public.data_pipeline_step_telemetry(run_id);
