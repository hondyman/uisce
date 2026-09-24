-- 1. Full interaction telemetry log
CREATE TABLE IF NOT EXISTS ai_query_telemetry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(64) NOT NULL,
    user_role VARCHAR(64) DEFAULT 'analyst',
    prompt TEXT NOT NULL,
    generated_query JSONB NOT NULL,
    executed_query JSONB NOT NULL,
    was_edited BOOLEAN DEFAULT FALSE,
    was_saved BOOLEAN DEFAULT FALSE,
    was_exported BOOLEAN DEFAULT FALSE,
    cloned_to_report BOOLEAN DEFAULT FALSE,
    rating SMALLINT DEFAULT 0, -- 1 = Up, -1 = Down, 0 = Neutral
    feedback_notes TEXT,
    execution_duration_ms INT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- 2. Verified few-shot golden query store
CREATE TABLE IF NOT EXISTS ai_golden_queries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    category VARCHAR(64) DEFAULT 'valuation', -- valuation, trading, performance, risk
    prompt_pattern TEXT NOT NULL,
    verified_query JSONB NOT NULL,
    score INT DEFAULT 1,
    is_global BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- 3. Staged knowledge candidates awaiting promotion
CREATE TABLE IF NOT EXISTS ai_knowledge_candidates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    type VARCHAR(32) NOT NULL, -- 'alias', 'calculated_measure', 'default_filter'
    term VARCHAR(128) NOT NULL,
    target_field_id VARCHAR(128),
    expression TEXT,
    occurrences INT DEFAULT 1,
    confidence NUMERIC(4, 3) DEFAULT 0.500,
    status VARCHAR(32) DEFAULT 'pending_review', -- pending_review, approved, rejected
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_type_term UNIQUE(tenant_id, type, term)
);

-- 4. Active semantic dictionary overrides (aliases & calculated fields)
CREATE TABLE IF NOT EXISTS semantic_field_aliases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id VARCHAR(64) NOT NULL,
    alias_term VARCHAR(128) NOT NULL,
    target_field_id VARCHAR(128) NOT NULL,
    is_global BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT uq_tenant_alias UNIQUE(tenant_id, alias_term)
);

CREATE INDEX IF NOT EXISTS idx_telemetry_tenant_created ON ai_query_telemetry(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_golden_tenant_score ON ai_golden_queries(tenant_id, score DESC);
CREATE INDEX IF NOT EXISTS idx_candidates_status ON ai_knowledge_candidates(tenant_id, status);
