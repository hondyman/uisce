-- 20260930_130_personalized_calculation_suggestions.up.sql
-- AI Calculation Suggestions, User Personalization & Dismissal Memory

CREATE SCHEMA IF NOT EXISTS catalog_calc;

-- 1. AI Calculation Suggestions Catalog Store
CREATE TABLE IF NOT EXISTS catalog_calc.ai_calculation_suggestions (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    suggested_calc_key   VARCHAR(100) NOT NULL,
    suggested_name       VARCHAR(200) NOT NULL,
    expression_sql       TEXT NOT NULL,
    return_type          VARCHAR(50) NOT NULL DEFAULT 'DECIMAL',
    rationale_narrative  TEXT NOT NULL,
    applicable_bo_key    VARCHAR(100) NOT NULL,
    input_terms          JSONB NOT NULL DEFAULT '[]'::jsonb, -- e.g. ["market_value", "total_aum"]
    confidence_score     NUMERIC(4, 3) NOT NULL DEFAULT 0.950,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_calc_suggestions_tenant_bo 
ON catalog_calc.ai_calculation_suggestions (tenant_id, applicable_bo_key);

-- 2. User Personalization & Suggestion Dismissal Memory (Never Suggest Again to this user)
CREATE TABLE IF NOT EXISTS catalog_calc.user_suggestion_dismissals (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    user_id              VARCHAR(100) NOT NULL,
    suggested_calc_key   VARCHAR(100) NOT NULL,
    applicable_bo_key    VARCHAR(100) NOT NULL,
    dismissed_reason     VARCHAR(100) DEFAULT 'USER_REJECTED',
    dismissed_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_calc_dismissal UNIQUE (tenant_id, user_id, suggested_calc_key, applicable_bo_key)
);

CREATE INDEX IF NOT EXISTS idx_user_dismissals_lookup 
ON catalog_calc.user_suggestion_dismissals (tenant_id, user_id, applicable_bo_key);
