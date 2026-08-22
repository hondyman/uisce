-- 20260824_arrow_vector_materialized_cache.sql
-- High-throughput calculation cache and incremental bitemporal snapshots

CREATE TABLE IF NOT EXISTS public.calc_cache (
    cache_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    bo_id UUID NOT NULL,
    field_id UUID NOT NULL,
    entity_sid VARCHAR(100) NOT NULL,
    cache_key VARCHAR(64) NOT NULL, -- SHA-256 of AST formula + parameter signatures
    computed_value DOUBLE PRECISION NOT NULL,
    vector_payload BYTEA,           -- Packed Arrow IPC / binary cashflow vector
    last_event_timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_calc_cache_entry UNIQUE (tenant_id, bo_id, field_id, entity_sid, cache_key)
);

CREATE TABLE IF NOT EXISTS public.calc_cache_history (
    history_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    cache_id UUID NOT NULL REFERENCES public.calc_cache(cache_id) ON DELETE CASCADE,
    entity_sid VARCHAR(100) NOT NULL,
    computed_value DOUBLE PRECISION NOT NULL,
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to TIMESTAMPTZ,
    vector_payload BYTEA,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.calc_cube_snapshots (
    cube_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    cube_name VARCHAR(100) NOT NULL,
    dimension_grain VARCHAR(50) NOT NULL, -- PORTFOLIO_MONTH, ACCOUNT_DAILY
    time_bucket DATE NOT NULL,
    metrics_payload JSONB NOT NULL,       -- Pre-computed IRR, TWR, Duration, Exposures
    refreshed_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_cube_grain_bucket UNIQUE (tenant_id, cube_name, dimension_grain, time_bucket)
);

CREATE INDEX IF NOT EXISTS idx_calc_cache_lookup 
ON public.calc_cache (tenant_id, bo_id, entity_sid, cache_key);

CREATE INDEX IF NOT EXISTS idx_calc_history_asof 
ON public.calc_cache_history (tenant_id, entity_sid, valid_from, valid_to);
