-- backend/migrations/016_performance_settings.sql
-- Portfolio Master — Performance Settings Entity + Metadata
-- Implements: performance_settings table, BO registration, and linkage to portfolio_master.

-- ============================================================
-- 1. PERFORMANCE SETTINGS MASTER
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.performance_settings (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    core_id                  UUID REFERENCES edm.performance_settings(id),
    portfolio_id             VARCHAR(100) NOT NULL,
    valuation_method         VARCHAR(50) 
                                 CHECK (valuation_method IN ('Daily_TIB','Monthly_Weighted','Simple_Dietz','Modified_Dietz')),
    fee_treatment            VARCHAR(50) 
                                 CHECK (fee_treatment IN ('Gross','Net','Both')),
    cash_flow_method         VARCHAR(100),           -- Beginning of day vs end of day
    currency_hedging_policy  VARCHAR(100),
    lookthrough_policy       TEXT,                   -- Policy on underlying fund lookthrough
    treatment_of_derivatives TEXT,                   -- Notional vs Market Value vs Delta Adjusted
    confidence_score         INT NOT NULL DEFAULT 90 CHECK (confidence_score BETWEEN 0 AND 100),
    source_systems           JSONB NOT NULL DEFAULT '{}',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by               UUID NOT NULL,
    updated_by               UUID,
    valid_from               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_performance_settings UNIQUE (tenant_id, portfolio_id, valid_from)
);

CREATE INDEX IF NOT EXISTS idx_ps_tenant   ON edm.performance_settings(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ps_portfolio ON edm.performance_settings(portfolio_id);
CREATE INDEX IF NOT EXISTS idx_ps_current   ON edm.performance_settings(tenant_id) WHERE valid_to IS NULL;

ALTER TABLE edm.performance_settings ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS performance_settings_isolation ON edm.performance_settings;
CREATE POLICY performance_settings_isolation ON edm.performance_settings
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 2. LINK TO PORTFOLIO MASTER
-- ============================================================
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'edm'
          AND table_name   = 'portfolio_master'
          AND column_name  = 'performance_settings_id'
    ) THEN
        ALTER TABLE edm.portfolio_master
            ADD COLUMN performance_settings_id UUID REFERENCES edm.performance_settings(id);
        
        CREATE INDEX IF NOT EXISTS idx_pm_performance_settings ON edm.portfolio_master(performance_settings_id);
    END IF;
END
$$;

-- ============================================================
-- 3. METADATA SEEDING (Business Object + Fields)
-- ============================================================
DO $$
DECLARE
    v_gold_tenant   UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    bo_ps_id        UUID;
    bo_portfolio_id UUID;
    semantic_term_type_id    UUID;
BEGIN
    -- 1. Register Performance Settings BO
    INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, category, created_at)
    VALUES (gen_random_uuid(), v_gold_tenant, 'performance_settings', 'Performance Settings', 'Performance Settings', 'performance_settings',
            'Calculation methodologies, fee treatments, and hedging policies for portfolio performance reporting.',
            'chart-line', true, 'Investment', NOW())
    ON CONFLICT DO NOTHING;
    
    SELECT id INTO bo_ps_id FROM business_objects WHERE key = 'performance_settings' AND tenant_id = v_gold_tenant LIMIT 1;

    -- 2. Add Fields to Performance Settings BO
    INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, sequence)
    SELECT gen_random_uuid(), v_gold_tenant, bo_ps_id, key, label, label, key, type, true, required, seq
    FROM (VALUES
        ('portfolio_id',             'Portfolio ID',            'text',     true,  1),
        ('valuation_method',         'Valuation Method',        'picklist', true,  2),
        ('fee_treatment',            'Fee Treatment',           'picklist', true,  3),
        ('cash_flow_method',         'Cash Flow Method',        'text',     false, 4),
        ('currency_hedging_policy',  'Currency Hedging Policy', 'text',     false, 5),
        ('lookthrough_policy',       'Lookthrough Policy',      'text',     false, 6),
        ('treatment_of_derivatives', 'Derivatives Treatment',   'text',     false, 7)
    ) AS t(key, label, type, required, seq)
    ON CONFLICT DO NOTHING;

    -- 3. Add link field to Portfolio BO
    SELECT id INTO bo_portfolio_id FROM business_objects WHERE key = 'portfolio' LIMIT 1;
    IF bo_portfolio_id IS NOT NULL THEN
        INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, sequence)
        VALUES (gen_random_uuid(), v_gold_tenant, bo_portfolio_id, 'performance_settings_id', 'Performance Settings', 'Performance Settings', 'performance_settings_id', 'reference', true, false, 70)
        ON CONFLICT DO NOTHING;
    END IF;

    -- 4. Seed Semantic Terms
    SELECT id INTO semantic_term_type_id FROM catalog_node_type WHERE catalog_type_name = 'semantic_term' LIMIT 1;
    
    IF semantic_term_type_id IS NOT NULL THEN
        INSERT INTO catalog_node (id, node_name, node_type_id, tenant_id, properties, qualified_path, created_at, updated_at)
        VALUES
            (gen_random_uuid(), 'ValuationMethod',   semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"performance"}', 'semantic/performance/ValuationMethod', NOW(), NOW()),
            (gen_random_uuid(), 'FeeTreatment',      semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"performance"}', 'semantic/performance/FeeTreatment',    NOW(), NOW()),
            (gen_random_uuid(), 'CashFlowMethod',    semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"performance"}', 'semantic/performance/CashFlowMethod',  NOW(), NOW()),
            (gen_random_uuid(), 'HedgingPolicy',     semantic_term_type_id, v_gold_tenant, '{"data_type":"text","domain":"performance"}', 'semantic/performance/HedgingPolicy',   NOW(), NOW())
        ON CONFLICT (node_name, node_type_id, tenant_id) DO NOTHING;
    END IF;

END $$;
