CREATE TABLE IF NOT EXISTS semantic_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    datasource TEXT NOT NULL,
    sql_fingerprint TEXT NOT NULL,
    sql_latency_ms FLOAT NOT NULL,
    sql_rows INT NOT NULL,
    groupby_fields TEXT,  -- JSON array
    filter_fields TEXT,   -- JSON array
    measure_fields TEXT,  -- JSON array
    preagg_id UUID,
    preagg_hit BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for maximizing query performance in suggestion engine
CREATE INDEX IF NOT EXISTS idx_semantic_events_analysis 
ON semantic_events (tenant_id, created_at) 
INCLUDE (sql_latency_ms, sql_rows, preagg_hit);
