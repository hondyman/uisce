-- 20261221_010_post_trade_pilot_schema.up.sql
--
-- Core Compliance Engine — Post-Trade Batch Evaluation, Portfolio Snapshots, Security Master & Finding Lineage
-- 1. compliance.compliance_portfolio_snapshot table (aggregated positions and metrics)
-- 2. master.security_master_snapshot table (shared security reference data)
-- 3. master.tenant_classification_override table (tenant-specific security classification overrides)
-- 4. compliance.compliance_finding table (deterministic lineage, UUIDv5, supersession restatements)
-- 5. Status transition triggers, indexes, and Row-Level Security policies
-- 6. Pilot Post-Trade Rules Seed (UCITS_5_10_40, SEC_144A_QIB_HOLDING, MARGIN_UTILIZATION_80)

CREATE SCHEMA IF NOT EXISTS master;

-- 1. Portfolio Snapshot Table
CREATE TABLE IF NOT EXISTS compliance.compliance_portfolio_snapshot (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    account_id              UUID NOT NULL,
    as_of_date              DATE NOT NULL,
    nav                     NUMERIC(28, 6) NOT NULL,
    gross_exposure          NUMERIC(28, 6) NOT NULL DEFAULT 0,
    net_exposure            NUMERIC(28, 6) NOT NULL DEFAULT 0,
    cash_balance            NUMERIC(28, 6) NOT NULL DEFAULT 0,
    positions               JSONB NOT NULL DEFAULT '[]'::jsonb,
    metrics                 JSONB NOT NULL DEFAULT '{}'::jsonb,
    content_hash            TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unq_portfolio_snapshot_tenant_account_date UNIQUE (tenant_id, account_id, as_of_date)
);

CREATE INDEX IF NOT EXISTS idx_portfolio_snapshot_tenant_date
    ON compliance.compliance_portfolio_snapshot (tenant_id, as_of_date DESC);

CREATE INDEX IF NOT EXISTS idx_portfolio_snapshot_tenant_account
    ON compliance.compliance_portfolio_snapshot (tenant_id, account_id);

ALTER TABLE compliance.compliance_portfolio_snapshot ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_portfolio_snapshot FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_portfolio_snapshot ON compliance.compliance_portfolio_snapshot;
CREATE POLICY tenant_isolation_portfolio_snapshot ON compliance.compliance_portfolio_snapshot
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
    );

-- 2. Shared Security Master Snapshot Table
CREATE TABLE IF NOT EXISTS master.security_master_snapshot (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    security_id             VARCHAR(64) NOT NULL,
    symbol                  VARCHAR(32) NOT NULL,
    isin                    VARCHAR(12),
    cusip                   VARCHAR(9),
    issuer_id               VARCHAR(64) NOT NULL,
    issuer_name             TEXT NOT NULL,
    country_of_risk         VARCHAR(3) NOT NULL,
    sector                  TEXT NOT NULL,
    asset_class             TEXT NOT NULL,
    is_qib_eligible         BOOLEAN NOT NULL DEFAULT false,
    is_144a                 BOOLEAN NOT NULL DEFAULT false,
    credit_rating           VARCHAR(16),
    effective_date          DATE NOT NULL,
    valid_to                DATE,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unq_security_master_snapshot UNIQUE (security_id, effective_date)
);

CREATE INDEX IF NOT EXISTS idx_sec_master_symbol ON master.security_master_snapshot (symbol);
CREATE INDEX IF NOT EXISTS idx_sec_master_issuer ON master.security_master_snapshot (issuer_id);
CREATE INDEX IF NOT EXISTS idx_sec_master_effective ON master.security_master_snapshot (effective_date DESC);

-- 3. Tenant Classification Override Table
CREATE TABLE IF NOT EXISTS master.tenant_classification_override (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    security_id             VARCHAR(64) NOT NULL,
    override_type           TEXT NOT NULL CHECK (override_type IN
                                ('SECTOR', 'ASSET_CLASS', 'COUNTRY_OF_RISK', 'CREDIT_RATING', 'IS_144A', 'IS_QIB_ELIGIBLE')),
    override_value          TEXT NOT NULL,
    reason                  TEXT NOT NULL,
    created_by              TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unq_tenant_class_override UNIQUE (tenant_id, security_id, override_type)
);

CREATE INDEX IF NOT EXISTS idx_tenant_override_lookup 
    ON master.tenant_classification_override (tenant_id, security_id);

ALTER TABLE master.tenant_classification_override ENABLE ROW LEVEL SECURITY;
ALTER TABLE master.tenant_classification_override FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_tenant_classification_override ON master.tenant_classification_override;
CREATE POLICY tenant_isolation_tenant_classification_override ON master.tenant_classification_override
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
    );

-- 4. Post-Trade Compliance Finding Table
CREATE TABLE IF NOT EXISTS compliance.compliance_finding (
    id                      UUID PRIMARY KEY, -- RFC 4122 UUIDv5 lineage ID
    tenant_id               UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    rule_id                 UUID NOT NULL REFERENCES compliance.compliance_rule(id) ON DELETE RESTRICT,
    rule_version            INT NOT NULL,
    rule_code               TEXT NOT NULL,
    account_id              UUID NOT NULL,
    as_of_date              DATE NOT NULL,
    evaluation_tier         TEXT NOT NULL CHECK (evaluation_tier IN ('PRE_TRADE', 'POST_TRADE', 'SURVEILLANCE')),
    status                  TEXT NOT NULL DEFAULT 'OPEN' CHECK (status IN ('OPEN', 'SUPERSEDED', 'RESOLVED', 'DISMISSED')),
    action                  TEXT NOT NULL CHECK (action IN ('APPROVED', 'BLOCKED', 'WARNING', 'BREACHED', 'BREACH_CONFIRMED', 'WITHIN_LIMITS')),
    finding_severity        TEXT NOT NULL CHECK (finding_severity IN ('HARD_BLOCK', 'SOFT_WARNING', 'APPROVAL_REQUIRED', 'WARNING', 'INFORMATIONAL', 'HIGH', 'CRITICAL', 'MEDIUM', 'LOW')),
    supersedes_finding_id   UUID REFERENCES compliance.compliance_finding(id) ON DELETE SET NULL,
    superseded_reason       TEXT,
    lineage_hash            TEXT NOT NULL,
    portfolio_snapshot_id   UUID REFERENCES compliance.compliance_portfolio_snapshot(id) ON DELETE CASCADE,
    details                 JSONB NOT NULL DEFAULT '{}'::jsonb,
    resolved_by             TEXT,
    resolved_at             TIMESTAMPTZ,
    resolution_notes        TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_compliance_finding_tenant_status 
    ON compliance.compliance_finding (tenant_id, status, as_of_date DESC);

CREATE INDEX IF NOT EXISTS idx_compliance_finding_rule_account 
    ON compliance.compliance_finding (tenant_id, rule_code, account_id);

CREATE UNIQUE INDEX IF NOT EXISTS unq_open_compliance_finding_per_rule_account_date 
    ON compliance.compliance_finding (tenant_id, rule_code, account_id, as_of_date) 
    WHERE status = 'OPEN';

ALTER TABLE compliance.compliance_finding ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_finding FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_compliance_finding ON compliance.compliance_finding;
CREATE POLICY tenant_isolation_compliance_finding ON compliance.compliance_finding
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
    );

-- 5. State Machine Trigger for Findings
CREATE OR REPLACE FUNCTION compliance.validate_compliance_finding_transition()
RETURNS trigger AS $$
DECLARE
    v_legal TEXT[];
BEGIN
    IF NEW.status = OLD.status THEN
        NEW.updated_at := now();
        RETURN NEW;
    END IF;

    v_legal := CASE OLD.status
        WHEN 'OPEN'        THEN ARRAY['SUPERSEDED', 'RESOLVED', 'DISMISSED']
        WHEN 'RESOLVED'    THEN ARRAY['OPEN']
        WHEN 'DISMISSED'   THEN ARRAY['OPEN']
        WHEN 'SUPERSEDED'  THEN ARRAY[]::TEXT[] -- Terminal State
        ELSE ARRAY[]::TEXT[]
    END;

    IF NOT (NEW.status = ANY(v_legal)) THEN
        RAISE EXCEPTION 'Illegal compliance finding status transition: % -> % (Allowed: %)',
            OLD.status, NEW.status, v_legal;
    END IF;

    IF NEW.status = 'SUPERSEDED' THEN
        IF NEW.superseded_reason IS NULL OR trim(NEW.superseded_reason) = '' THEN
            RAISE EXCEPTION 'Superseding a compliance finding requires superseded_reason';
        END IF;
    END IF;

    IF NEW.status IN ('RESOLVED', 'DISMISSED') THEN
        IF NEW.resolution_notes IS NULL OR trim(NEW.resolution_notes) = '' THEN
            RAISE EXCEPTION 'Resolution of compliance finding requires resolution_notes';
        END IF;
        IF NEW.resolved_by IS NULL OR trim(NEW.resolved_by) = '' THEN
            RAISE EXCEPTION 'Resolution of compliance finding requires resolved_by';
        END IF;
        IF NEW.resolved_at IS NULL THEN
            NEW.resolved_at := now();
        END IF;
    END IF;

    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_validate_compliance_finding_transition ON compliance.compliance_finding;
CREATE TRIGGER trg_validate_compliance_finding_transition
    BEFORE UPDATE ON compliance.compliance_finding
    FOR EACH ROW
    EXECUTE FUNCTION compliance.validate_compliance_finding_transition();

-- 6. Privileges
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        GRANT USAGE ON SCHEMA compliance TO app_user;
        GRANT USAGE ON SCHEMA master TO app_user;
        GRANT SELECT, INSERT, UPDATE, DELETE ON compliance.compliance_portfolio_snapshot TO app_user;
        GRANT SELECT, INSERT, UPDATE, DELETE ON compliance.compliance_finding TO app_user;
        GRANT SELECT, INSERT, UPDATE, DELETE ON master.tenant_classification_override TO app_user;
        GRANT SELECT ON master.security_master_snapshot TO app_user;
    END IF;
END $$;

-- 7. Seed 3 Pilot Post-Trade Rules into Gold-Copy Master
DO $$
DECLARE
    v_gold_tenant UUID;
    v_r1 UUID;
    v_r2 UUID;
    v_r3 UUID;
BEGIN
    SELECT id INTO v_gold_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant IS NULL THEN
        SELECT '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid INTO v_gold_tenant;
    END IF;

    -- 1. UCITS_5_10_40 (Post-Trade 5/10/40 aggregate threshold rule)
    v_r1 := compliance.seed_core_rule(
        'UCITS_5_10_40',
        'UCITS 5/10/40 Aggregate Concentration Limit',
        'POST',
        'HARD_BLOCK',
        98,
        'UCITS Directive 2009/65/EC Art. 52(1)-(2)',
        ARRAY['EU', 'GLOBAL'],
        '{"max_single_issuer_pct": "0.100000", "max_aggregate_above_5pct_pct": "0.400000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.ucits_aggregate_above_5pct_exposure"},"op":"LTE","right":{"type":"PARAM","name":"max_aggregate_above_5pct_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- 2. SEC_144A_QIB_HOLDING (Post-Trade 15% QIB / Illiquid Asset Limit)
    v_r2 := compliance.seed_core_rule(
        'SEC_144A_QIB_HOLDING',
        'SEC 144A / QIB Illiquid Asset Limit',
        'POST',
        'HARD_BLOCK',
        92,
        'SEC Rule 144A / Investment Company Act Rule 22e-4',
        ARRAY['US', 'GLOBAL'],
        '{"max_144a_non_qib_pct": "0.150000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.restricted_144a_exposure_pct"},"op":"LTE","right":{"type":"PARAM","name":"max_144a_non_qib_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- 3. MARGIN_UTILIZATION_80 (Post-Trade 80% Margin House Capacity Warning)
    v_r3 := compliance.seed_core_rule(
        'MARGIN_UTILIZATION_80',
        'Portfolio Margin Capacity Utilization Limit',
        'POST',
        'SOFT_WARNING',
        85,
        'FINRA Rule 4210 / House Margin Policy',
        ARRAY['US', 'EU', 'GLOBAL'],
        '{"max_margin_utilization_pct": "0.800000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.margin_utilization_pct"},"op":"LTE","right":{"type":"PARAM","name":"max_margin_utilization_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- Create v1 snapshots for each pilot rule
    INSERT INTO compliance.compliance_rule_version (
        rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
        citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
        created_by, created_at
    )
    SELECT 
        r.id,
        COALESCE(r.current_version, 1),
        r.tenant_id,
        r.ast_condition,
        r.parameter_thresholds,
        r.citation,
        r.effective_from,
        r.effective_to,
        CASE r.rule_code
            WHEN 'UCITS_5_10_40' THEN 'ae74e81024024ed0bdc8507d1baf8db97459a3babe636a2aa419f8313137b480'
            WHEN 'SEC_144A_QIB_HOLDING' THEN '54d3e559f1655951e7b5e6f4b84d914134364de64ee0c7cd91d11efd118deb0e'
            WHEN 'MARGIN_UTILIZATION_80' THEN '3426602c668bf78f5f6cd8b7b620dd7c55956c20754d72e178853655a3d71dcb'
            ELSE encode(sha256(('v1|' || r.ast_condition::text || '|' || r.parameter_thresholds::text || '|' || COALESCE(r.citation, ''))::bytea), 'hex')
        END,
        COALESCE(encode(sha256(r.compiled_bytecode), 'hex'), encode(sha256(''::bytea), 'hex')),
        'system_seed_v1',
        now()
    FROM compliance.compliance_rule r
    WHERE r.id IN (v_r1, v_r2, v_r3)
    ON CONFLICT (rule_id, version) DO NOTHING;
END $$;
