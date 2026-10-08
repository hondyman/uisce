-- 20261224_019_phase7_tranche2_and_class_b_lists.up.sql
--
-- Phase VII Tranche 2: Passive Institutional Disclosure (13G), ERISA 25% BPI Aggregation & Class B List Pipeline
-- 1. Create compliance.compliance_investor_classification for upward ERISA Plan Asset evaluation
-- 2. Extend master.fund_hierarchy_edge relationship_type with BPI_INVESTOR, NON_BPI_INVESTOR, GP_DISREGARDED
-- 3. Extend compliance.compliance_restricted_list with staleness monitoring and match types
-- 4. Seed Rule Family: FIDUCIARY_AND_PLAN_ASSETS
-- 5. Seed Rule 43 (Rule 93 total): POST_TRADE_SEC_SCHEDULE_13G_PASSIVE
-- 6. Seed Rule 44 (Rule 94 total): POST_TRADE_ERISA_PLAN_ASSET_25PCT
-- 7. Insert v1 Snapshots with RFC 8785 Canonical Content Hashes into compliance_rule_version
-- 8. Assign to POST_TRADE_MONITORING Ruleset Pack

-- ============================================================================
-- 1. Upward Investor Classification Table for ERISA BPI Calculation
-- ============================================================================
CREATE TABLE IF NOT EXISTS compliance.compliance_investor_classification (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID NOT NULL,
    investor_account_id    UUID NOT NULL,
    investor_name          TEXT NOT NULL,
    investor_type          TEXT NOT NULL CHECK (investor_type IN (
        'ERISA_BENEFIT_PLAN',   -- Title I ERISA plan subject to fiduciary rules
        'IRA_INDIVIDUAL',       -- Individual Retirement Account under IRC § 4975
        'PLAN_ASSET_ENTITY',    -- Fund/vehicle with >= 25% BPI participation
        'PUBLIC_PENSION',       -- Governmental plan (Non-BPI under ERISA § 3(32))
        'FOREIGN_PENSION',      -- Non-US pension (Non-BPI)
        'CORPORATE_INVESTOR',   -- Operating company / non-plan corporate entity
        'GP_MANAGEMENT'         -- GP, manager, or affiliate equity (disregarded in denominator)
    )),
    is_benefit_plan_investor BOOLEAN NOT NULL DEFAULT false,
    is_gp_disregarded        BOOLEAN NOT NULL DEFAULT false,
    effective_from           TIMESTAMPTZ NOT NULL DEFAULT now(),
    effective_to             TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unq_investor_classification_active UNIQUE (tenant_id, investor_account_id, effective_from)
);

CREATE INDEX IF NOT EXISTS idx_investor_class_lookup
    ON compliance.compliance_investor_classification (tenant_id, investor_account_id)
    WHERE effective_to IS NULL;

ALTER TABLE compliance.compliance_investor_classification ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_investor_classification FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_investor_class ON compliance.compliance_investor_classification;
CREATE POLICY tenant_isolation_investor_class ON compliance.compliance_investor_classification
    FOR ALL USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
        OR tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid
    );

GRANT SELECT, INSERT, UPDATE, DELETE ON compliance.compliance_investor_classification TO app_user;

-- ============================================================================
-- 2. Extend master.fund_hierarchy_edge check constraint for investor relationships
-- ============================================================================
ALTER TABLE master.fund_hierarchy_edge 
    DROP CONSTRAINT IF EXISTS fund_hierarchy_edge_relationship_type_check;

ALTER TABLE master.fund_hierarchy_edge 
    ADD CONSTRAINT fund_hierarchy_edge_relationship_type_check 
    CHECK (relationship_type IN (
        'MASTER_FEEDER',       -- Master fund holds underlying assets; feeder is investor
        'UMBRELLA_SUBFUND',    -- Legal umbrella entity containing sub-fund compartments
        'SMA_SLEEVE',          -- Separately Managed Account subdivided into trading sleeves
        'BENEFICIAL_OWNER',    -- Ultimate parent entity exercising investment discretion
        'PARALLEL_FUND',       -- Co-investing alongside master fund under common mandate
        'BPI_INVESTOR',        -- Benefit plan investor equity interest into fund (ERISA test)
        'NON_BPI_INVESTOR',    -- Non-benefit plan investor equity interest
        'GP_DISREGARDED'       -- General Partner / manager equity (excluded from denominator)
    ));

-- ============================================================================
-- 3. Extend compliance_restricted_list with match metadata and staleness tracking
-- ============================================================================
ALTER TABLE compliance.compliance_restricted_list
    ADD COLUMN IF NOT EXISTS source_feed_name TEXT NOT NULL DEFAULT 'MANUAL_IMPORT',
    ADD COLUMN IF NOT EXISTS last_refreshed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS item_count INT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS content_hash TEXT NOT NULL DEFAULT '';

ALTER TABLE compliance.compliance_restricted_list_item
    ADD COLUMN IF NOT EXISTS match_type TEXT NOT NULL DEFAULT 'EXACT' CHECK (match_type IN ('EXACT', 'ALIAS', 'IDENTIFIER_LOOKUP')),
    ADD COLUMN IF NOT EXISTS match_key TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS issuer_name TEXT NOT NULL DEFAULT '';

ALTER TABLE compliance.compliance_rule
    DROP CONSTRAINT IF EXISTS compliance_rule_aggregation_scope_check;
ALTER TABLE compliance.compliance_rule
    ADD CONSTRAINT compliance_rule_aggregation_scope_check
    CHECK (aggregation_scope IN ('ACCOUNT_LEVEL', 'MASTER_FEEDER_LOOKTHROUGH', 'FIRM_WIDE_UBO', 'LEGAL_ENTITY_AGGREGATION', 'INVESTOR_AGGREGATION_UPWARD'));

ALTER TABLE compliance.compliance_rule
    DROP CONSTRAINT IF EXISTS compliance_rule_filing_deadline_type_check;
ALTER TABLE compliance.compliance_rule
    ADD CONSTRAINT compliance_rule_filing_deadline_type_check
    CHECK (filing_deadline_type IN ('TRADING_DAYS', 'BUSINESS_HOURS', 'NEXT_TRADING_DAY_CUTOFF', 'IMMEDIATE', 'CALENDAR_DAYS_POST_YEAR_END'));

-- ============================================================================
-- 4. Seed Rule Family: FIDUCIARY_AND_PLAN_ASSETS
-- ============================================================================
INSERT INTO compliance.compliance_rule_family (id, family_code, display_name, description, domain, base_metric_path)
VALUES
    (
        'f0000000-0000-4000-a000-000000000004'::uuid,
        'FIDUCIARY_AND_PLAN_ASSETS',
        'ERISA & Fiduciary Plan Asset Monitoring',
        'Monitors Benefit Plan Investor (BPI) equity participation against the 25% significant participation threshold under ERISA § 3(42) and DOL 29 CFR § 2510.3-101.',
        'PLAN_ASSETS',
        'portfolio.erisa_bpi_equity_pct'
    )
ON CONFLICT (family_code) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    domain = EXCLUDED.domain,
    base_metric_path = EXCLUDED.base_metric_path;

-- ============================================================================
-- 5. Seed Phase VII Tranche 2 Rules (Rules 43–44) into Gold-Copy Master
-- ============================================================================
DO $$
DECLARE
    v_gold_tenant UUID;
    v_fam_major UUID := 'f0000000-0000-4000-a000-000000000001'::uuid;
    v_fam_fiduciary UUID := 'f0000000-0000-4000-a000-000000000004'::uuid;

    v_r1 UUID;
    v_r2 UUID;
BEGIN
    SELECT id INTO v_gold_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant IS NULL THEN
        SELECT '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid INTO v_gold_tenant;
    END IF;

    -- Rule 43: POST_TRADE_SEC_SCHEDULE_13G_PASSIVE
    v_r1 := compliance.seed_core_rule(
        'POST_TRADE_SEC_SCHEDULE_13G_PASSIVE',
        'SEC Schedule 13G Passive Institutional Shareholding Disclosure (>=5% / >=10%)',
        'POST',
        'HARD_BLOCK',
        90,
        'Securities Exchange Act of 1934 Section 13(g) (15 U.S.C. § 78m(g)); SEC Rule 13d-1(b), (c), (d) (17 CFR § 240.13d-1); SEC Release No. 33-11253',
        ARRAY['US_SEC', 'GLOBAL'],
        '{"accelerated_threshold_pct": "0.100000", "filer_category": "QII_QUALIFIED_INSTITUTIONAL", "initial_passive_threshold_pct": "0.050000", "is_passive_intent": true, "max_passive_ownership_ceiling_pct": "0.200000"}'::jsonb,
        '{"conditions":[{"left":{"path":"portfolio.firmwide_equity_voting_pct","type":"METRIC"},"operator":"LESS_THAN","right":{"name":"initial_passive_threshold_pct","type":"PARAM"},"type":"COMPARISON"},{"left":{"path":"portfolio.is_passive_intent","type":"METRIC"},"operator":"EQUAL","right":{"name":"is_passive_intent","type":"PARAM"},"type":"COMPARISON"}],"operator":"AND","type":"LOGICAL_AND"}'::jsonb,
        'ACTIVE'
    );

    UPDATE compliance.compliance_rule
    SET description = 'Monitors passive institutional beneficial ownership (5% annual clock, 10% accelerated 5-day clock, 20% passive ceiling) and forces Schedule 13D conversion upon loss of passive intent.',
        data_provenance = 'SYNTHETIC_FIXTURE',
        rule_family_id = v_fam_major,
        jurisdiction_code = 'US_SEC',
        aggregation_scope = 'FIRM_WIDE_UBO',
        filing_deadline_type = 'CALENDAR_DAYS_POST_YEAR_END',
        filing_deadline_value = 45,
        filing_cutoff_time = '17:30:00',
        filing_deadline_hours = 1080
    WHERE id = v_r1;

    -- Rule 44: POST_TRADE_ERISA_PLAN_ASSET_25PCT
    v_r2 := compliance.seed_core_rule(
        'POST_TRADE_ERISA_PLAN_ASSET_25PCT',
        'ERISA Benefit Plan Investor (BPI) 25% Significant Participation Ceiling',
        'POST',
        'HARD_BLOCK',
        95,
        'Employee Retirement Income Security Act of 1974 (ERISA) § 3(42) (29 U.S.C. § 1002(42)); 29 CFR § 2510.3-101 (DOL Plan Asset Regulation); ERISA § 406 Prohibited Transactions',
        ARRAY['US_DOL_ERISA', 'GLOBAL'],
        '{"disregard_gp_interests": true, "max_bpi_equity_pct": "0.250000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.erisa_bpi_equity_pct"},"operator":"LESS_THAN","right":{"type":"PARAM","name":"max_bpi_equity_pct"}}'::jsonb,
        'ACTIVE'
    );

    UPDATE compliance.compliance_rule
    SET description = 'Flags when Benefit Plan Investor (BPI) equity interests in a private fund reach or exceed 25.0% of total equity (excluding GP interests), subjecting fund assets to ERISA Title I fiduciary regulation.',
        data_provenance = 'SYNTHETIC_FIXTURE',
        rule_family_id = v_fam_fiduciary,
        jurisdiction_code = 'US_DOL_ERISA',
        aggregation_scope = 'INVESTOR_AGGREGATION_UPWARD',
        filing_deadline_type = 'IMMEDIATE',
        filing_deadline_value = 0,
        filing_cutoff_time = 'IMMEDIATE',
        filing_deadline_hours = 0
    WHERE id = v_r2;

    -- Insert v1 snapshots with Go RFC 8785 canonical content hashes
    INSERT INTO compliance.compliance_rule_version (
        rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
        citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
        created_by, created_at
    )
    SELECT 
        r.id,
        1,
        r.tenant_id,
        r.ast_condition,
        r.parameter_thresholds,
        r.citation,
        r.effective_from,
        r.effective_to,
        CASE r.rule_code
            WHEN 'POST_TRADE_SEC_SCHEDULE_13G_PASSIVE' THEN '5a82c7d5241f092f1bb27218cae8cb97cadb9c7d2b173b45c9bf29ccf1e50cb4'
            WHEN 'POST_TRADE_ERISA_PLAN_ASSET_25PCT'   THEN '9b51f1a16678c5030f365ad8afbd7995cb3192f42830e9e90eea1d188d678ecf'
            ELSE encode(sha256(('v1|' || r.ast_condition::text || '|' || r.parameter_thresholds::text || '|' || COALESCE(r.citation, ''))::bytea), 'hex')
        END,
        COALESCE(encode(sha256(r.compiled_bytecode), 'hex'), encode(sha256(''::bytea), 'hex')),
        'system_seed_v1',
        now()
    FROM compliance.compliance_rule r
    WHERE r.id IN (v_r1, v_r2)
    ON CONFLICT (rule_id, version) DO NOTHING;

    -- Assign both rules to POST_TRADE_MONITORING ruleset pack (Pack 4)
    INSERT INTO compliance.compliance_ruleset_membership (ruleset_code, rule_id)
    SELECT 'POST_TRADE_MONITORING', r.id
    FROM compliance.compliance_rule r
    WHERE r.tenant_id = v_gold_tenant
      AND r.id IN (v_r1, v_r2)
    ON CONFLICT (ruleset_code, rule_id) DO NOTHING;

END $$;
