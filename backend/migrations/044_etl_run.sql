-- migrations/044_etl_run.sql

CREATE TABLE IF NOT EXISTS edm.etl_run (
    run_id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES auth.tenants(id) ON DELETE CASCADE,
    valuation_date DATE NOT NULL,
    engine_name VARCHAR(50) NOT NULL, -- 'COMPLIANCE', 'RISK', 'STRESS_TESTING'
    status VARCHAR(20) NOT NULL,      -- 'RUNNING', 'SUCCESS', 'FAILED'
    start_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    end_time TIMESTAMP WITH TIME ZONE,
    portfolios_processed INTEGER DEFAULT 0,
    evaluations_count INTEGER DEFAULT 0,
    breaches_count INTEGER DEFAULT 0,
    metrics_jsonb JSONB,             -- e.g. latency breakdowns
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for querying recent tenant runs
CREATE INDEX IF NOT EXISTS idx_etl_run_tenant_date ON edm.etl_run (tenant_id, valuation_date DESC);
CREATE INDEX IF NOT EXISTS idx_etl_run_status ON edm.etl_run (status);
