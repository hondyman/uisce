-- 20261222_011_phase1_counterparty_and_group_schema.up.sql
--
-- Post-Trade Phase 1 (Issuer & Counterparty Concentration, Group Exposure, and Cash Limits)
-- 1. master.entity_relationship_snapshot table (Generic point-in-time hierarchy graph for group issuers and funds)
-- 2. Immutability triggers and append-only guards for entity relationships
-- 3. Seed 4 Phase 1 Tranche 1 Post-Trade Rules into Gold-Copy Master (POST_TRADE_GROUP_ISSUER_20, POST_TRADE_ISSUER_DEBT_15, POST_TRADE_COUNTERPARTY_PFE_10, POST_TRADE_CASH_MIN_5)
-- 4. Seed Synthetic Entity Relationships (Source: SYNTHETIC_FIXTURE_PHASE1)

CREATE SCHEMA IF NOT EXISTS master;

-- 1. Generic Entity Relationship Snapshot Table
CREATE TABLE IF NOT EXISTS master.entity_relationship_snapshot (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_entity_id        VARCHAR(64) NOT NULL,
    parent_entity_type      TEXT NOT NULL CHECK (parent_entity_type IN ('ISSUER', 'COUNTERPARTY', 'FUND', 'PARENT_COMPANY')),
    child_entity_id         VARCHAR(64) NOT NULL,
    child_entity_type       TEXT NOT NULL CHECK (child_entity_type IN ('ISSUER', 'SECURITY', 'SUB_FUND', 'SUBSIDIARY', 'BRANCH')),
    relationship_type       TEXT NOT NULL CHECK (relationship_type IN ('PARENT_COMPANY', 'MAJORITY_OWNED_SUBSIDIARY', 'WHOLLY_OWNED_SUBSIDIARY', 'AFFILIATE', 'SERIES_MASTER', 'GUARANTOR')),
    ownership_pct           NUMERIC(8, 4) NOT NULL DEFAULT 1.0000,
    effective_date          DATE NOT NULL,
    valid_to                DATE,
    content_hash            TEXT NOT NULL,
    source                  TEXT NOT NULL DEFAULT 'SYNTHETIC_FIXTURE_PHASE1',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unq_entity_rel_snapshot UNIQUE (parent_entity_id, child_entity_id, relationship_type, effective_date)
);

CREATE INDEX IF NOT EXISTS idx_entity_rel_parent ON master.entity_relationship_snapshot (parent_entity_id);
CREATE INDEX IF NOT EXISTS idx_entity_rel_child ON master.entity_relationship_snapshot (child_entity_id);
CREATE INDEX IF NOT EXISTS idx_entity_rel_effective ON master.entity_relationship_snapshot (effective_date DESC);

-- 2. Mutation and Truncate Trigger Guards on master.entity_relationship_snapshot
CREATE OR REPLACE FUNCTION master.prevent_entity_rel_mutation()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'Audit Violation: master.entity_relationship_snapshot is strictly append-only. Mutation (UPDATE/DELETE) is disallowed.'
        USING ERRCODE = '55000';
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION master.prevent_entity_rel_truncate()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'Audit Violation: master.entity_relationship_snapshot is strictly append-only. TRUNCATE is disallowed.'
        USING ERRCODE = '55000';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_prevent_entity_rel_mutation ON master.entity_relationship_snapshot;
CREATE TRIGGER trg_prevent_entity_rel_mutation
    BEFORE UPDATE OR DELETE ON master.entity_relationship_snapshot
    FOR EACH ROW
    EXECUTE FUNCTION master.prevent_entity_rel_mutation();

DROP TRIGGER IF EXISTS trg_prevent_entity_rel_truncate ON master.entity_relationship_snapshot;
CREATE TRIGGER trg_prevent_entity_rel_truncate
    BEFORE TRUNCATE ON master.entity_relationship_snapshot
    FOR EACH STATEMENT
    EXECUTE FUNCTION master.prevent_entity_rel_truncate();

-- Grant privileges to app_user
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'app_user') THEN
        GRANT SELECT ON master.entity_relationship_snapshot TO app_user;
    END IF;
END $$;

-- 3. Seed 4 Phase 1 Tranche 1 Post-Trade Rules into Gold-Copy Master
DO $$
DECLARE
    v_gold_tenant UUID;
    v_r1 UUID;
    v_r2 UUID;
    v_r3 UUID;
    v_r4 UUID;
BEGIN
    SELECT id INTO v_gold_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant IS NULL THEN
        SELECT '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid INTO v_gold_tenant;
    END IF;

    -- 1. POST_TRADE_GROUP_ISSUER_20 (Group & Related-Party Issuer Aggregate Concentration Limit)
    v_r1 := compliance.seed_core_rule(
        'POST_TRADE_GROUP_ISSUER_20',
        'Group & Related-Party Issuer Aggregate Concentration Limit',
        'POST',
        'HARD_BLOCK',
        95,
        'UCITS Directive 2009/65/EC Art. 52(3) / Investment Company Act Sec. 12(d)',
        ARRAY['EU', 'US', 'GLOBAL'],
        '{"max_group_issuer_pct": "0.200000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_group_issuer_exposure_pct"},"op":"LTE","right":{"type":"PARAM","name":"max_group_issuer_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- 2. POST_TRADE_ISSUER_DEBT_15 (Single-Issuer Debt & Fixed Income Concentration Limit)
    v_r2 := compliance.seed_core_rule(
        'POST_TRADE_ISSUER_DEBT_15',
        'Single-Issuer Debt & Fixed Income Concentration Limit',
        'POST',
        'HARD_BLOCK',
        90,
        'FINRA Rule 4210 / Institutional Fixed Income Mandate',
        ARRAY['US', 'EU', 'GLOBAL'],
        '{"max_issuer_debt_pct": "0.150000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_issuer_debt_exposure_pct"},"op":"LTE","right":{"type":"PARAM","name":"max_issuer_debt_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- 3. POST_TRADE_COUNTERPARTY_PFE_10 (OTC Derivative Counterparty Net Exposure & PFE Limit)
    v_r3 := compliance.seed_core_rule(
        'POST_TRADE_COUNTERPARTY_PFE_10',
        'OTC Derivative Counterparty Net Exposure & PFE Limit',
        'POST',
        'HARD_BLOCK',
        94,
        'BCBS 279 Standardised Approach for Counterparty Credit Risk (SA-CCR) / EMIR Art. 11',
        ARRAY['GLOBAL', 'EU', 'US'],
        '{"max_counterparty_pfe_pct": "0.100000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.max_counterparty_pfe_exposure_pct"},"op":"LTE","right":{"type":"PARAM","name":"max_counterparty_pfe_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- 4. POST_TRADE_CASH_MIN_5 (Minimum Portfolio Cash & Cash Equivalents Liquidity Floor)
    v_r4 := compliance.seed_core_rule(
        'POST_TRADE_CASH_MIN_5',
        'Minimum Portfolio Cash & Cash Equivalents Liquidity Floor',
        'POST',
        'SOFT_WARNING',
        80,
        'ESMA Guidelines on Liquidity Stress Testing / UCITS Liquidity Management',
        ARRAY['EU', 'GLOBAL'],
        '{"min_cash_pct": "0.050000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.cash_and_equivalent_pct"},"op":"GTE","right":{"type":"PARAM","name":"min_cash_pct"}}'::jsonb,
        'ACTIVE'
    );

    -- Insert v1 snapshots for each seeded rule
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
            WHEN 'POST_TRADE_GROUP_ISSUER_20' THEN '2df67ab6dd5d50acc9c0edfff9345db6df94a8154867b92aa0e1568d9c867408'
            WHEN 'POST_TRADE_ISSUER_DEBT_15' THEN '612925dd6fd4598d3c8050dd5731ea0213f580bb9c763fb9008f2b8264e48e1e'
            WHEN 'POST_TRADE_COUNTERPARTY_PFE_10' THEN '40be3254468ff521d266e61f822488e0118cdc57b86f13558929c60fb118d002'
            WHEN 'POST_TRADE_CASH_MIN_5' THEN '6438860a9643d7d305ff9511c113ee5d7f128c8dbb7bbdcdd32a690c75be8a43'
            ELSE encode(sha256(('v1|' || r.ast_condition::text || '|' || r.parameter_thresholds::text || '|' || COALESCE(r.citation, ''))::bytea), 'hex')
        END,
        COALESCE(encode(sha256(r.compiled_bytecode), 'hex'), encode(sha256(''::bytea), 'hex')),
        'system_seed_v1',
        now()
    FROM compliance.compliance_rule r
    WHERE r.id IN (v_r1, v_r2, v_r3, v_r4)
    ON CONFLICT (rule_id, version) DO NOTHING;
END $$;

-- 4. Seed Synthetic Entity Relationships (Source: SYNTHETIC_FIXTURE_PHASE1)
INSERT INTO master.entity_relationship_snapshot (
    parent_entity_id, parent_entity_type, child_entity_id, child_entity_type,
    relationship_type, ownership_pct, effective_date, content_hash, source
) VALUES
    ('GRP_ALPHA_PARENT', 'PARENT_COMPANY', 'ISSUER_ALPHA_1', 'ISSUER', 'WHOLLY_OWNED_SUBSIDIARY', 1.0000, '2026-01-01', 'hash_rel_001', 'SYNTHETIC_FIXTURE_PHASE1'),
    ('GRP_ALPHA_PARENT', 'PARENT_COMPANY', 'ISSUER_ALPHA_2', 'ISSUER', 'MAJORITY_OWNED_SUBSIDIARY', 0.7500, '2026-01-01', 'hash_rel_002', 'SYNTHETIC_FIXTURE_PHASE1'),
    ('GRP_BETA_HOLDINGS', 'PARENT_COMPANY', 'ISSUER_BETA_BANK', 'ISSUER', 'WHOLLY_OWNED_SUBSIDIARY', 1.0000, '2026-01-01', 'hash_rel_003', 'SYNTHETIC_FIXTURE_PHASE1')
ON CONFLICT (parent_entity_id, child_entity_id, relationship_type, effective_date) DO NOTHING;
