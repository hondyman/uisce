-- 20261026_010_orm_model_portfolios.up.sql
-- account_model_assignment.account_id → oms.account(id) (cross-schema).
-- model_portfolio_target.security_id left as plain UUID.

CREATE TABLE IF NOT EXISTS orm.model_portfolio (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    model_cd varchar(50) NOT NULL,
    name varchar(250) NOT NULL,
    model_type varchar(30) NOT NULL,
    base_currency varchar(3) NOT NULL,
    manager_id uuid,
    benchmark_id uuid,
    risk_profile varchar(20),
    rebalance_frequency varchar(20),
    rebalance_threshold_pct numeric(7,4),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_active bool DEFAULT true NOT NULL,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT model_portfolio_pkey PRIMARY KEY (id),
    CONSTRAINT uq_mp_cd UNIQUE (tenant_id, model_cd, effective_from),
    CONSTRAINT chk_mp_type CHECK (model_type IN (
        'BALANCED','GROWTH','INCOME','ESG','THEMATIC','TARGET_RISK',
        'TARGET_DATE','CUSTOM'))
);
CREATE INDEX IF NOT EXISTS idx_mp_cd ON orm.model_portfolio (model_cd, is_current);

CREATE TABLE IF NOT EXISTS orm.model_portfolio_target (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    model_portfolio_id uuid NOT NULL,
    target_type varchar(30) NOT NULL,
    target_reference_cd varchar(50),
    security_id uuid,
    target_weight_pct numeric(7,4) NOT NULL,
    min_weight_pct numeric(7,4),
    max_weight_pct numeric(7,4),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT model_portfolio_target_pkey PRIMARY KEY (id),
    CONSTRAINT chk_mpt_type CHECK (target_type IN (
        'SECURITY','ASSET_CLASS','SECTOR','REGION','SLEEVE','ETF','MUTUAL_FUND')),
    CONSTRAINT fk_mpt_model FOREIGN KEY (model_portfolio_id) REFERENCES orm.model_portfolio(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_mpt_model ON orm.model_portfolio_target (model_portfolio_id, is_current);

CREATE TABLE IF NOT EXISTS orm.account_model_assignment (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    account_id uuid NOT NULL,
    model_portfolio_id uuid NOT NULL,
    allocation_pct numeric(7,4) DEFAULT 100.0 NOT NULL,
    drift_tolerance_pct numeric(7,4),
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    is_current bool DEFAULT true NOT NULL,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    tenant_id uuid NOT NULL,
    CONSTRAINT account_model_assignment_pkey PRIMARY KEY (id),
    CONSTRAINT fk_ama_account FOREIGN KEY (account_id) REFERENCES oms.account(id),
    CONSTRAINT fk_ama_model FOREIGN KEY (model_portfolio_id) REFERENCES orm.model_portfolio(id)
);
CREATE INDEX IF NOT EXISTS idx_ama_account ON orm.account_model_assignment (account_id, is_current);
