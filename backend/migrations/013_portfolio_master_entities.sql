-- backend/migrations/013_portfolio_master_entities.sql
-- Portfolio Master Gold Copy — Full Entity Driver Tables
-- Implements: portfolio_master, mandate_master, benchmark_master, strategy_master,
--             compliance_rule_master, portfolio_hierarchy, gold_copy_lineage, survivorship_rules

-- ============================================================
-- 1. PORTFOLIO MASTER
--    The canonical, institutional-grade portfolio metadata record.
--    This is the METADATA gold copy (distinct from edm.portfolio_golden
--    which tracks per-security positions).
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.portfolio_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    core_id                  UUID REFERENCES edm.portfolio_master(id),  -- NULL = core
    portfolio_id             VARCHAR(100) NOT NULL,
    portfolio_code           VARCHAR(50),
    portfolio_name           VARCHAR(255) NOT NULL,
    portfolio_type           VARCHAR(50) NOT NULL
                                 CHECK (portfolio_type IN ('SMA','Fund','ETF','Model','Mandate','Composite')),
    portfolio_category       VARCHAR(50)
                                 CHECK (portfolio_category IN ('Retail','Institutional','HNW','Advisory','PrivateWealth','PrivateMarkets')),
    inception_date           DATE NOT NULL,
    termination_date         DATE,
    base_currency            CHAR(3) NOT NULL,
    domicile                 VARCHAR(50),
    legal_structure          VARCHAR(100),           -- Trust, Fund, LP, UCITS, SICAV
    regulatory_classification VARCHAR(100),          -- UCITS, 40-Act, AIF
    liquidity_profile        VARCHAR(50)
                                 CHECK (liquidity_profile IN ('Daily','Weekly','Monthly','Quarterly','Locked')),
    risk_profile             VARCHAR(50)
                                 CHECK (risk_profile IN ('Conservative','Balanced','Moderate','Aggressive','Growth')),
    investment_objective     TEXT,
    investment_guidelines    TEXT,
    valuation_frequency      VARCHAR(50)
                                 CHECK (valuation_frequency IN ('Daily','Weekly','Monthly','Quarterly')),
    pricing_source           VARCHAR(100),
    is_model_portfolio       BOOLEAN NOT NULL DEFAULT false,
    is_composite_member      BOOLEAN NOT NULL DEFAULT false,
    -- Relationships (FK to sibling master tables)
    benchmark_id             UUID REFERENCES edm.benchmark_master(id),
    strategy_id              UUID REFERENCES edm.strategy_master(id),
    mandate_id               UUID REFERENCES edm.mandate_master(id),
    composite_id             UUID REFERENCES edm.portfolio_master(id),  -- if is_composite_member
    -- Operational references (stored as VARCHAR, not FK, to avoid tight coupling)
    portfolio_manager_id     VARCHAR(100),
    team_id                  VARCHAR(100),
    custodian_id             VARCHAR(100),
    administrator_id         VARCHAR(100),
    accounting_system_id     VARCHAR(100),
    -- Confidence and source tracking
    confidence_score         INT NOT NULL DEFAULT 80 CHECK (confidence_score BETWEEN 0 AND 100),
    source_systems           JSONB NOT NULL DEFAULT '{}', -- {"portfolio_name": "AccountingSystem", ...}
    -- Audit + multi-tenant
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by               UUID NOT NULL,
    updated_by               UUID,
    -- Bi-temporal
    valid_from               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_portfolio_master UNIQUE (tenant_id, portfolio_id, valid_from)
);

CREATE INDEX IF NOT EXISTS idx_pm_tenant       ON edm.portfolio_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_pm_portfolio    ON edm.portfolio_master(portfolio_id);
CREATE INDEX IF NOT EXISTS idx_pm_type         ON edm.portfolio_master(portfolio_type, portfolio_category);
CREATE INDEX IF NOT EXISTS idx_pm_current      ON edm.portfolio_master(tenant_id) WHERE valid_to IS NULL;
CREATE INDEX IF NOT EXISTS idx_pm_benchmark    ON edm.portfolio_master(benchmark_id);
CREATE INDEX IF NOT EXISTS idx_pm_strategy     ON edm.portfolio_master(strategy_id);
CREATE INDEX IF NOT EXISTS idx_pm_mandate      ON edm.portfolio_master(mandate_id);

ALTER TABLE edm.portfolio_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS portfolio_master_isolation ON edm.portfolio_master;
CREATE POLICY portfolio_master_isolation ON edm.portfolio_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 2. MANDATE MASTER
--    Client investment objectives, constraints, and guidelines.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.mandate_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    core_id                  UUID REFERENCES edm.mandate_master(id),
    mandate_code             VARCHAR(50) NOT NULL,
    mandate_name             VARCHAR(255) NOT NULL,
    mandate_type             VARCHAR(50) NOT NULL
                                 CHECK (mandate_type IN ('Advisory','Discretionary','ModelDriven','NonDiscretionary')),
    client_id                VARCHAR(100),
    investment_objective     VARCHAR(50)
                                 CHECK (investment_objective IN ('Growth','Income','Preservation','Balanced','Aggressive')),
    risk_tolerance           VARCHAR(50)
                                 CHECK (risk_tolerance IN ('Low','Medium','High','VeryHigh')),
    time_horizon             VARCHAR(50)
                                 CHECK (time_horizon IN ('Short','Medium','Long','VeryLong')),
    liquidity_needs          VARCHAR(50)
                                 CHECK (liquidity_needs IN ('High','Medium','Low','Locked')),
    tax_constraints          TEXT,
    esg_constraints          TEXT,
    custom_restrictions      TEXT,
    benchmark_id             UUID REFERENCES edm.benchmark_master(id),
    confidence_score         INT NOT NULL DEFAULT 80 CHECK (confidence_score BETWEEN 0 AND 100),
    source_systems           JSONB NOT NULL DEFAULT '{}',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by               UUID NOT NULL,
    updated_by               UUID,
    valid_from               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_mandate_master UNIQUE (tenant_id, mandate_code, valid_from)
);

CREATE INDEX IF NOT EXISTS idx_mandate_tenant  ON edm.mandate_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_mandate_client  ON edm.mandate_master(client_id);
CREATE INDEX IF NOT EXISTS idx_mandate_current ON edm.mandate_master(tenant_id) WHERE valid_to IS NULL;

ALTER TABLE edm.mandate_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS mandate_master_isolation ON edm.mandate_master;
CREATE POLICY mandate_master_isolation ON edm.mandate_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 3. BENCHMARK MASTER
--    Reference indices (S&P 500, MSCI ACWI, Bloomberg Agg, custom).
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.benchmark_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    core_id                  UUID REFERENCES edm.benchmark_master(id),
    benchmark_code           VARCHAR(50) NOT NULL,
    benchmark_name           VARCHAR(255) NOT NULL,
    benchmark_type           VARCHAR(50) NOT NULL
                                 CHECK (benchmark_type IN ('Equity','FixedIncome','MultiAsset','Commodity','Alternative','Custom')),
    currency                 CHAR(3) NOT NULL DEFAULT 'USD',
    provider                 VARCHAR(100),         -- MSCI, S&P, Bloomberg
    composition_source       VARCHAR(50) DEFAULT 'Vendor',
    is_custom                BOOLEAN NOT NULL DEFAULT false,
    custom_definition        JSONB NOT NULL DEFAULT '{}',  -- weights / constituent list for custom benchmarks
    rebalance_frequency      VARCHAR(50)
                                 CHECK (rebalance_frequency IN ('Daily','Monthly','Quarterly','Annually')),
    confidence_score         INT NOT NULL DEFAULT 90 CHECK (confidence_score BETWEEN 0 AND 100),
    source_systems           JSONB NOT NULL DEFAULT '{}',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by               UUID NOT NULL,
    updated_by               UUID,
    valid_from               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_benchmark_master UNIQUE (tenant_id, benchmark_code, valid_from)
);

CREATE INDEX IF NOT EXISTS idx_benchmark_tenant  ON edm.benchmark_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_benchmark_current ON edm.benchmark_master(tenant_id) WHERE valid_to IS NULL;

ALTER TABLE edm.benchmark_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS benchmark_master_isolation ON edm.benchmark_master;
CREATE POLICY benchmark_master_isolation ON edm.benchmark_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 4. STRATEGY MASTER
--    Investment strategies defining HOW a portfolio is managed.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.strategy_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    core_id                  UUID REFERENCES edm.strategy_master(id),
    strategy_code            VARCHAR(50) NOT NULL,
    strategy_name            VARCHAR(255) NOT NULL,
    strategy_category        VARCHAR(50) NOT NULL
                                 CHECK (strategy_category IN ('Equity','FixedIncome','MultiAsset','Alternative','Hybrid')),
    investment_style         VARCHAR(50)
                                 CHECK (investment_style IN ('Growth','Value','Core','Blend','Momentum','Quality')),
    geographic_focus         VARCHAR(50)
                                 CHECK (geographic_focus IN ('Global','US','DM','EM','EMEA','APAC','LatAm')),
    sector_focus             VARCHAR(100),
    risk_budget              TEXT,          -- Free text: "Volatility target 10%, tracking error <3%"
    leverage_policy          VARCHAR(50)
                                 CHECK (leverage_policy IN ('NotAllowed','Allowed','LimitedAllowed')),
    derivatives_policy       VARCHAR(50)
                                 CHECK (derivatives_policy IN ('NotAllowed','HedgingOnly','Allowed')),
    esg_policy               TEXT,         -- Exclusions, ratings thresholds, etc.
    confidence_score         INT NOT NULL DEFAULT 85 CHECK (confidence_score BETWEEN 0 AND 100),
    source_systems           JSONB NOT NULL DEFAULT '{}',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by               UUID NOT NULL,
    updated_by               UUID,
    valid_from               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_strategy_master UNIQUE (tenant_id, strategy_code, valid_from)
);

CREATE INDEX IF NOT EXISTS idx_strategy_tenant  ON edm.strategy_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_strategy_current ON edm.strategy_master(tenant_id) WHERE valid_to IS NULL;

ALTER TABLE edm.strategy_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS strategy_master_isolation ON edm.strategy_master;
CREATE POLICY strategy_master_isolation ON edm.strategy_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 5. COMPLIANCE RULE MASTER
--    Portfolio constraints expressed as DSL/rule expressions.
--    Examples: "No >5% single issuer", "Min 80% IG bonds"
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.compliance_rule_master (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    core_id                  UUID REFERENCES edm.compliance_rule_master(id),
    rule_code                VARCHAR(50) NOT NULL,
    rule_name                VARCHAR(255) NOT NULL,
    portfolio_id             VARCHAR(100),         -- NULL = strategy-level / mandate-level rule
    strategy_id              UUID REFERENCES edm.strategy_master(id),
    mandate_id               UUID REFERENCES edm.mandate_master(id),
    rule_type                VARCHAR(50) NOT NULL
                                 CHECK (rule_type IN ('Exposure','Issuer','Sector','Liquidity','ESG','Compliance','Custom')),
    rule_expression          TEXT NOT NULL,        -- DSL: "position.weight <= 0.05"
    severity                 VARCHAR(20) NOT NULL
                                 CHECK (severity IN ('Hard','Soft','Warning')),
    frequency                VARCHAR(50) NOT NULL
                                 CHECK (frequency IN ('PreTrade','PostTrade','Daily','Weekly','Monthly')),
    -- Bi-temporal validity
    effective_date           DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date                 DATE,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by               UUID NOT NULL,
    updated_by               UUID,
    valid_from               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to                 TIMESTAMPTZ,
    CONSTRAINT uq_compliance_rule UNIQUE (tenant_id, rule_code, valid_from)
);

CREATE INDEX IF NOT EXISTS idx_cr_tenant    ON edm.compliance_rule_master(tenant_id);
CREATE INDEX IF NOT EXISTS idx_cr_portfolio ON edm.compliance_rule_master(portfolio_id);
CREATE INDEX IF NOT EXISTS idx_cr_strategy  ON edm.compliance_rule_master(strategy_id);
CREATE INDEX IF NOT EXISTS idx_cr_type      ON edm.compliance_rule_master(rule_type, severity);
CREATE INDEX IF NOT EXISTS idx_cr_current   ON edm.compliance_rule_master(tenant_id) WHERE valid_to IS NULL;

ALTER TABLE edm.compliance_rule_master ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS compliance_rule_isolation ON edm.compliance_rule_master;
CREATE POLICY compliance_rule_isolation ON edm.compliance_rule_master
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 6. PORTFOLIO HIERARCHY
--    Parent/child relationships: Sleeve, Composite, Umbrella, ShareClass.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.portfolio_hierarchy (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    parent_portfolio_id      VARCHAR(100) NOT NULL,
    child_portfolio_id       VARCHAR(100) NOT NULL,
    relationship_type        VARCHAR(50) NOT NULL
                                 CHECK (relationship_type IN ('Sleeve','Composite','Umbrella','ShareClass','Model','SubFund')),
    weight                   NUMERIC(8,6),         -- Optional, for composites (0-1)
    effective_date           DATE NOT NULL DEFAULT CURRENT_DATE,
    end_date                 DATE,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by               UUID NOT NULL,
    updated_by               UUID,
    CONSTRAINT uq_portfolio_hierarchy UNIQUE (tenant_id, parent_portfolio_id, child_portfolio_id, effective_date),
    CONSTRAINT chk_no_self_reference CHECK (parent_portfolio_id <> child_portfolio_id)
);

CREATE INDEX IF NOT EXISTS idx_ph_tenant   ON edm.portfolio_hierarchy(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ph_parent   ON edm.portfolio_hierarchy(parent_portfolio_id);
CREATE INDEX IF NOT EXISTS idx_ph_child    ON edm.portfolio_hierarchy(child_portfolio_id);
CREATE INDEX IF NOT EXISTS idx_ph_active   ON edm.portfolio_hierarchy(tenant_id) WHERE end_date IS NULL;

ALTER TABLE edm.portfolio_hierarchy ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS portfolio_hierarchy_isolation ON edm.portfolio_hierarchy;
CREATE POLICY portfolio_hierarchy_isolation ON edm.portfolio_hierarchy
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 7. GOLD COPY LINEAGE
--    Append-only trace of every gold copy decision.
--    Which source provided each field value, which rule fired.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.gold_copy_lineage (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    entity_type              VARCHAR(50) NOT NULL,  -- portfolio_master | mandate_master | etc.
    entity_id                UUID NOT NULL,
    field_name               VARCHAR(100) NOT NULL,
    chosen_value             TEXT,
    chosen_source            VARCHAR(100),          -- AccountingSystem | OMS | Custodian | Manual
    rejected_sources         JSONB NOT NULL DEFAULT '[]',  -- [{source, value, reason}]
    survivorship_rule        VARCHAR(100),          -- prefer_source | earliest_non_null | latest_by
    dq_rules_passed          TEXT[],
    dq_rules_failed          TEXT[],
    confidence_contribution  INT,
    run_id                   UUID NOT NULL,          -- Groups all decisions from one gold copy run
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_gcl_tenant      ON edm.gold_copy_lineage(tenant_id);
CREATE INDEX IF NOT EXISTS idx_gcl_entity      ON edm.gold_copy_lineage(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_gcl_run         ON edm.gold_copy_lineage(run_id);
CREATE INDEX IF NOT EXISTS idx_gcl_created     ON edm.gold_copy_lineage(created_at);

ALTER TABLE edm.gold_copy_lineage ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS gold_copy_lineage_isolation ON edm.gold_copy_lineage;
CREATE POLICY gold_copy_lineage_isolation ON edm.gold_copy_lineage
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));

-- ============================================================
-- 8. SURVIVORSHIP RULES
--    Template field-level survivorship strategies.
--    Resolved by the gold copy engine at runtime.
-- ============================================================
CREATE TABLE IF NOT EXISTS edm.survivorship_rules (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID NOT NULL,
    core_id                  UUID REFERENCES edm.survivorship_rules(id),
    entity_type              VARCHAR(50) NOT NULL,  -- portfolio_master | mandate_master | etc.
    field_name               VARCHAR(100) NOT NULL,
    strategy                 VARCHAR(50) NOT NULL
                                 CHECK (strategy IN ('prefer_source','earliest_non_null','latest_by','highest_quality','conditional','manual')),
    -- For prefer_source: ordered list of preferred sources
    preferred_sources        TEXT[] NOT NULL DEFAULT '{}',
    -- For conditional: JSON rule expression
    condition_expression     TEXT,
    -- For time-based: which timestamp field to compare
    time_field               VARCHAR(100),
    priority                 INT NOT NULL DEFAULT 1,
    is_active                BOOLEAN NOT NULL DEFAULT true,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by               UUID NOT NULL,
    CONSTRAINT uq_survivorship_rule UNIQUE (tenant_id, entity_type, field_name, priority)
);

CREATE INDEX IF NOT EXISTS idx_sr_tenant  ON edm.survivorship_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_sr_entity  ON edm.survivorship_rules(entity_type, field_name);
CREATE INDEX IF NOT EXISTS idx_sr_active  ON edm.survivorship_rules(is_active);

ALTER TABLE edm.survivorship_rules ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS survivorship_rules_isolation ON edm.survivorship_rules;
CREATE POLICY survivorship_rules_isolation ON edm.survivorship_rules
    FOR ALL USING (tenant_id::text = current_setting('app.current_tenant_id', true));
