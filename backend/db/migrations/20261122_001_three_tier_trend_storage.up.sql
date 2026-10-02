-- Migration: 20261122_001_three_tier_trend_storage.up.sql
-- Enables three-tier watermark storage for historical trend analytics:
-- Hot (0-30d) in StarRocks, Warm (31-365d) in PostgreSQL, Cold (366+d) in Lakekeeper/Iceberg.

-- 1. Storage watermark settings in mdm_eval.scoring_settings
ALTER TABLE mdm_eval.scoring_settings
    ADD COLUMN IF NOT EXISTS watermark_hot_days INT NOT NULL DEFAULT 30,
    ADD COLUMN IF NOT EXISTS watermark_warm_days INT NOT NULL DEFAULT 365;

-- 2. PostgreSQL Warm Tier Scorecard History table
CREATE TABLE IF NOT EXISTS mdm_eval.vendor_scorecard_history (
    history_id             BIGSERIAL PRIMARY KEY,
    tenant_id              UUID NOT NULL,
    as_of_date             DATE NOT NULL,
    vendor_id              VARCHAR(32) NOT NULL,
    vendor_name            VARCHAR(255) NOT NULL,
    entity_domain          VARCHAR(32) NOT NULL DEFAULT 'EQUITY',
    weight_profile_id      BIGINT NOT NULL DEFAULT 1,
    dimension_name         VARCHAR(64) NOT NULL,
    score_value            NUMERIC(6,4) NOT NULL,
    raw_metric_value       NUMERIC(14,4),
    tier1_coverage         NUMERIC(6,4),
    tier2_coverage         NUMERIC(6,4),
    tier3_coverage         NUMERIC(6,4),
    annual_spend           NUMERIC(12,2),
    rank_position          INT,
    storage_tier           VARCHAR(16) NOT NULL DEFAULT 'WARM',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_vendor_scorecard_history_point UNIQUE (tenant_id, vendor_id, entity_domain, dimension_name, as_of_date)
);

CREATE INDEX IF NOT EXISTS idx_vsh_tenant_vendor_date
    ON mdm_eval.vendor_scorecard_history (tenant_id, vendor_id, as_of_date);

CREATE INDEX IF NOT EXISTS idx_vsh_tenant_dim_date
    ON mdm_eval.vendor_scorecard_history (tenant_id, dimension_name, as_of_date);

-- Enable RLS for multi-tenant isolation
ALTER TABLE mdm_eval.vendor_scorecard_history ENABLE ROW LEVEL SECURITY;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_policies 
        WHERE schemaname = 'mdm_eval' 
          AND tablename = 'vendor_scorecard_history' 
          AND policyname = 'tenant_isolation_vsh'
    ) THEN
        CREATE POLICY tenant_isolation_vsh ON mdm_eval.vendor_scorecard_history
            USING (tenant_id = public.uisce_get_current_tenant())
            WITH CHECK (tenant_id = public.uisce_get_current_tenant());
    END IF;
END $$;
