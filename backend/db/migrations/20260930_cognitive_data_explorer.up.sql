-- 20260930_cognitive_data_explorer.up.sql
-- ThoughtSpot-Grade Search Tokens, SpotIQ Attributions, Golden Answers & Threshold Subscriptions

CREATE SCHEMA IF NOT EXISTS explorer;

-- 1. Certified Golden Answers Store
CREATE TABLE IF NOT EXISTS explorer.certified_golden_answers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    query_title VARCHAR(255) NOT NULL,
    query_description TEXT,
    canonical_intent_hash VARCHAR(64) NOT NULL,
    semantic_ast JSONB NOT NULL,
    certified_by VARCHAR(100) NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    view_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_tenant_canonical_intent UNIQUE (tenant_id, canonical_intent_hash)
);

CREATE INDEX IF NOT EXISTS idx_golden_answers_tenant 
ON explorer.certified_golden_answers (tenant_id, is_active);

-- 2. SpotIQ Metric Driver Analysis Results Cache
CREATE TABLE IF NOT EXISTS explorer.spotiq_driver_analysis_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    bo_id UUID NOT NULL,
    metric_field_key VARCHAR(100) NOT NULL,
    dimension_field_key VARCHAR(100) NOT NULL,
    baseline_period VARCHAR(50) NOT NULL,
    comparison_period VARCHAR(50) NOT NULL,
    variance_pct NUMERIC(10, 4) NOT NULL,
    driver_breakdown JSONB NOT NULL, -- [{ "dimension_value": "Tech", "impact_pct": -1.8, "z_score": 3.4 }]
    generated_narrative TEXT NOT NULL,
    calculated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_spotiq_cache UNIQUE (tenant_id, bo_id, metric_field_key, dimension_field_key, baseline_period, comparison_period)
);

-- 3. Liveboard & Search Metric Alert Subscriptions
CREATE TABLE IF NOT EXISTS explorer.metric_alert_subscriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    bo_id UUID NOT NULL,
    metric_field_key VARCHAR(100) NOT NULL,
    operator VARCHAR(10) NOT NULL CHECK (operator IN ('>', '<', '>=', '<=', '=', 'DRIFT_PCT')),
    threshold_value NUMERIC(18, 6) NOT NULL,
    delivery_channel VARCHAR(50) NOT NULL DEFAULT 'IN_APP', -- IN_APP, SLACK, TEAMS, WEBHOOK
    webhook_url TEXT,
    is_triggered BOOLEAN NOT NULL DEFAULT FALSE,
    last_evaluated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_alert_subscriptions_eval 
ON explorer.metric_alert_subscriptions (tenant_id, is_triggered, last_evaluated_at);
