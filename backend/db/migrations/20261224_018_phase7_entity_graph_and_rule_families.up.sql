-- 20261224_018_phase7_entity_graph_and_rule_families.up.sql
--
-- Phase VII: Firm-Wide Ownership Aggregation, Fund Hierarchy Graph & Multi-Jurisdiction Rule Families
-- 1. Create master.fund_hierarchy_edge for multi-tier institutional fund lookthrough
-- 2. Create compliance.compliance_rule_family for cross-jurisdiction family concepts
-- 3. Extend compliance.compliance_rule with family, jurisdiction, aggregation scope, and deadline fields
-- 4. Extend compliance.compliance_finding with ACKNOWLEDGED and FILING_SUBMITTED statuses + filing_deadline_at
-- 5. Create compliance.compliance_restricted_list and compliance_restricted_list_item for Class B pipeline
-- 6. Seed Rule Families (MAJOR_SHAREHOLDING_DISCLOSURE, TAKEOVER_PANEL_MONITORING, NET_SHORT_POSITION_DISCLOSURE)
-- 7. Seed Phase VII Tranche 1 Rules (Rules 37-42) into Master Gold-Copy Tenant with Canonical Snapshots

-- ============================================================================
-- 1. Fund Hierarchy & Multi-Tier Entity Graph
-- ============================================================================
CREATE TABLE IF NOT EXISTS master.fund_hierarchy_edge (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    parent_account_id  UUID NOT NULL,
    child_account_id   UUID NOT NULL,
    relationship_type  TEXT NOT NULL CHECK (relationship_type IN (
        'MASTER_FEEDER',       -- Master fund holds underlying assets; feeder is investor
        'UMBRELLA_SUBFUND',    -- Legal umbrella entity containing sub-fund compartments
        'SMA_SLEEVE',          -- Separately Managed Account subdivided into trading sleeves
        'BENEFICIAL_OWNER',    -- Ultimate parent entity exercising investment discretion
        'PARALLEL_FUND'        -- Co-investing alongside master fund under common mandate
    )),
    economic_share_pct NUMERIC(10, 6) NOT NULL DEFAULT 1.000000,
    voting_control_pct NUMERIC(10, 6) NOT NULL DEFAULT 1.000000,
    valid_from         TIMESTAMPTZ NOT NULL DEFAULT now(),
    valid_to           TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unq_fund_hierarchy_active UNIQUE (tenant_id, parent_account_id, child_account_id, relationship_type, valid_from)
);

CREATE INDEX IF NOT EXISTS idx_fund_hierarchy_lookup
    ON master.fund_hierarchy_edge (tenant_id, parent_account_id, child_account_id)
    WHERE valid_to IS NULL;

-- Enable RLS
ALTER TABLE master.fund_hierarchy_edge ENABLE ROW LEVEL SECURITY;
ALTER TABLE master.fund_hierarchy_edge FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_fund_hierarchy ON master.fund_hierarchy_edge;
CREATE POLICY tenant_isolation_fund_hierarchy ON master.fund_hierarchy_edge
    FOR ALL USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
        OR tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid
    );

GRANT SELECT, INSERT, UPDATE, DELETE ON master.fund_hierarchy_edge TO app_user;

-- ============================================================================
-- 2. Compliance Rule Families
-- ============================================================================
CREATE TABLE IF NOT EXISTS compliance.compliance_rule_family (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    family_code       TEXT NOT NULL UNIQUE,
    display_name      TEXT NOT NULL,
    description       TEXT NOT NULL,
    domain            TEXT NOT NULL CHECK (domain IN ('OWNERSHIP_DISCLOSURE', 'SHORT_SELLING', 'TAKEOVER_CONTROL', 'PLAN_ASSETS', 'INSIDER_RESTRICTIONS')),
    base_metric_path  TEXT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

GRANT SELECT, INSERT, UPDATE, DELETE ON compliance.compliance_rule_family TO app_user;

-- ============================================================================
-- 3. Extend compliance.compliance_rule
-- ============================================================================
ALTER TABLE compliance.compliance_rule
    ADD COLUMN IF NOT EXISTS rule_family_id UUID REFERENCES compliance.compliance_rule_family(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS jurisdiction_code TEXT,
    ADD COLUMN IF NOT EXISTS aggregation_scope TEXT NOT NULL DEFAULT 'ACCOUNT_LEVEL'
        CHECK (aggregation_scope IN ('ACCOUNT_LEVEL', 'MASTER_FEEDER_LOOKTHROUGH', 'FIRM_WIDE_UBO', 'LEGAL_ENTITY_AGGREGATION')),
    ADD COLUMN IF NOT EXISTS filing_deadline_type TEXT DEFAULT 'BUSINESS_HOURS'
        CHECK (filing_deadline_type IN ('TRADING_DAYS', 'BUSINESS_HOURS', 'NEXT_TRADING_DAY_CUTOFF', 'IMMEDIATE')),
    ADD COLUMN IF NOT EXISTS filing_deadline_value INT DEFAULT 0,
    ADD COLUMN IF NOT EXISTS filing_cutoff_time TEXT,
    ADD COLUMN IF NOT EXISTS filing_deadline_hours INT DEFAULT 0;

-- ============================================================================
-- 4. Extend compliance.compliance_finding Status & Deadline Tracking
-- ============================================================================
ALTER TABLE compliance.compliance_finding
    DROP CONSTRAINT IF EXISTS compliance_finding_status_check;

ALTER TABLE compliance.compliance_finding
    ADD CONSTRAINT compliance_finding_status_check
    CHECK (status IN ('OPEN', 'ACKNOWLEDGED', 'FILING_SUBMITTED', 'SUPERSEDED', 'RESOLVED', 'DISMISSED'));

ALTER TABLE compliance.compliance_finding
    ADD COLUMN IF NOT EXISTS filing_deadline_at TIMESTAMPTZ;

-- ============================================================================
-- 5. Class B Restricted & Insider Lists
-- ============================================================================
CREATE TABLE IF NOT EXISTS compliance.compliance_restricted_list (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    list_code      TEXT NOT NULL,
    list_name      TEXT NOT NULL,
    list_type      TEXT NOT NULL CHECK (list_type IN ('RESTRICTED_TRADING', 'WATCHLIST', 'SANCTIONS_LOOKTHROUGH', 'EMPLOYEE_PRECLEAR')),
    is_active      BOOLEAN NOT NULL DEFAULT true,
    version        INT NOT NULL DEFAULT 1,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unq_restricted_list_code UNIQUE (tenant_id, list_code)
);

CREATE TABLE IF NOT EXISTS compliance.compliance_restricted_list_item (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID NOT NULL,
    list_id            UUID NOT NULL REFERENCES compliance.compliance_restricted_list(id) ON DELETE CASCADE,
    security_id        TEXT,
    issuer_id          TEXT NOT NULL,
    restriction_reason TEXT NOT NULL,
    restriction_scope  TEXT NOT NULL DEFAULT 'ALL_TRADING' CHECK (restriction_scope IN ('ALL_TRADING', 'BUYS_ONLY', 'SELLS_ONLY', 'DERIVATIVES_ONLY')),
    effective_from     TIMESTAMPTZ NOT NULL DEFAULT now(),
    effective_to       TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_restricted_list_item_lookup
    ON compliance.compliance_restricted_list_item (tenant_id, list_id, issuer_id)
    WHERE effective_to IS NULL;

ALTER TABLE compliance.compliance_restricted_list ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_restricted_list FORCE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_restricted_list_item ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_restricted_list_item FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_restricted_list ON compliance.compliance_restricted_list;
CREATE POLICY tenant_isolation_restricted_list ON compliance.compliance_restricted_list
    FOR ALL USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
        OR tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid
    );

DROP POLICY IF EXISTS tenant_isolation_restricted_list_item ON compliance.compliance_restricted_list_item;
CREATE POLICY tenant_isolation_restricted_list_item ON compliance.compliance_restricted_list_item
    FOR ALL USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
        OR tenant_id = '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid
    );

GRANT SELECT, INSERT, UPDATE, DELETE ON compliance.compliance_restricted_list TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON compliance.compliance_restricted_list_item TO app_user;

-- ============================================================================
-- 6. Seed Rule Families
-- ============================================================================
INSERT INTO compliance.compliance_rule_family (id, family_code, display_name, description, domain, base_metric_path)
VALUES
    (
        'f0000000-0000-4000-a000-000000000001'::uuid,
        'MAJOR_SHAREHOLDING_DISCLOSURE',
        'Major Shareholding & Beneficial Ownership Disclosure',
        'Multi-jurisdiction threshold crossing monitoring for major equity shareholdings and voting rights across US SEC 13D, UK FCA DTR5, and EU Transparency Directive.',
        'OWNERSHIP_DISCLOSURE',
        'portfolio.firmwide_equity_voting_pct'
    ),
    (
        'f0000000-0000-4000-a000-000000000002'::uuid,
        'TAKEOVER_PANEL_MONITORING',
        'Takeover Panel & Mandatory Bid Monitoring',
        'Monitors voting rights accumulation approaching or exceeding statutory takeover and mandatory cash offer thresholds under UK Takeover Code Rule 9.',
        'TAKEOVER_CONTROL',
        'portfolio.firmwide_voting_control_pct'
    ),
    (
        'f0000000-0000-4000-a000-000000000003'::uuid,
        'NET_SHORT_POSITION_DISCLOSURE',
        'Net Short Position Regulatory Disclosure',
        'Multi-jurisdiction monitoring of firm-wide net short positions in sovereign debt and equities under EU and UK Short Selling Regulations.',
        'SHORT_SELLING',
        'portfolio.firmwide_net_short_pct'
    )
ON CONFLICT (family_code) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    domain = EXCLUDED.domain,
    base_metric_path = EXCLUDED.base_metric_path;

-- ============================================================================
-- 7. Seed Phase VII Tranche 1 Rules (Rules 37-42) into Gold-Copy Master
-- ============================================================================
DO $$
DECLARE
    v_gold_tenant UUID;
    v_fam_major UUID := 'f0000000-0000-4000-a000-000000000001'::uuid;
    v_fam_takeover UUID := 'f0000000-0000-4000-a000-000000000002'::uuid;
    v_fam_ssr UUID := 'f0000000-0000-4000-a000-000000000003'::uuid;

    v_r1 UUID;
    v_r2 UUID;
    v_r3 UUID;
    v_r4 UUID;
    v_r5 UUID;
    v_r6 UUID;
BEGIN
    SELECT id INTO v_gold_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant IS NULL THEN
        SELECT '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid INTO v_gold_tenant;
    END IF;

    -- Rule 37: POST_TRADE_SEC_SCHEDULE_13D_5PCT
    v_r1 := compliance.seed_core_rule(
        'POST_TRADE_SEC_SCHEDULE_13D_5PCT',
        'SEC Schedule 13D/13G Major Shareholding Disclosure (>=5%)',
        'POST',
        'HARD_BLOCK',
        90,
        'Securities Exchange Act of 1934 Section 13(d)(1) (15 U.S.C. § 78m(d)); SEC Rule 13d-1(a) (17 CFR § 240.13d-1(a)); SEC Modernized Beneficial Ownership Reporting Release No. 33-11253',
        ARRAY['US_SEC', 'GLOBAL'],
        '{"max_voting_equity_pct": "0.050000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.firmwide_equity_voting_pct"},"operator":"LESS_THAN","right":{"type":"PARAM","name":"max_voting_equity_pct"}}'::jsonb,
        'ACTIVE'
    );

    UPDATE compliance.compliance_rule
    SET description = 'Flags when firm-wide beneficial ownership across all accounts and sleeves crosses or exceeds 5.0% of a registered class of voting equity securities under SEC Section 13(d)/13(g).',
        data_provenance = 'SYNTHETIC_FIXTURE',
        rule_family_id = v_fam_major,
        jurisdiction_code = 'US_SEC',
        aggregation_scope = 'FIRM_WIDE_UBO',
        filing_deadline_type = 'TRADING_DAYS',
        filing_deadline_value = 5,
        filing_cutoff_time = '17:30:00',
        filing_deadline_hours = 120
    WHERE id = v_r1;

    -- Rule 38: POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT
    v_r2 := compliance.seed_core_rule(
        'POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT',
        'UK FCA DTR 5 Major Shareholding Notification (>=3% + 1% Steps)',
        'POST',
        'HARD_BLOCK',
        90,
        'UK FCA Disclosure Guidance and Transparency Rules (DTR) Sourcebook 5.1.2R & 5.8.3R; UK Companies Act 2006 Part 43',
        ARRAY['UK_FCA', 'GLOBAL'],
        '{"initial_disclosure_threshold_pct": "0.030000", "step_size_pct": "0.010000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.firmwide_equity_voting_pct"},"operator":"LESS_THAN","right":{"type":"PARAM","name":"initial_disclosure_threshold_pct"}}'::jsonb,
        'ACTIVE'
    );

    UPDATE compliance.compliance_rule
    SET description = 'Flags when firm-wide voting rights in a UK issuer reach or cross 3.0% and each 1% integer step thereafter under UK FCA DTR 5.1.2R.',
        data_provenance = 'SYNTHETIC_FIXTURE',
        rule_family_id = v_fam_major,
        jurisdiction_code = 'UK_FCA',
        aggregation_scope = 'FIRM_WIDE_UBO',
        filing_deadline_type = 'TRADING_DAYS',
        filing_deadline_value = 2,
        filing_cutoff_time = '17:30:00',
        filing_deadline_hours = 48
    WHERE id = v_r2;

    -- Rule 39: POST_TRADE_EU_TRANSPARENCY_DIR_5PCT
    v_r3 := compliance.seed_core_rule(
        'POST_TRADE_EU_TRANSPARENCY_DIR_5PCT',
        'EU Transparency Directive Major Shareholding Notification (>=5%, 10%... Tiers)',
        'POST',
        'HARD_BLOCK',
        90,
        'Directive 2004/109/EC of the European Parliament and of the Council (Transparency Directive) Art. 9(1) & Art. 12; Commission Delegated Regulation (EU) 2015/761',
        ARRAY['EU_ESMA', 'GLOBAL'],
        '{"initial_threshold_pct": "0.050000", "tier_step_size_pct": "0.050000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.firmwide_equity_voting_pct"},"operator":"LESS_THAN","right":{"type":"PARAM","name":"initial_threshold_pct"}}'::jsonb,
        'ACTIVE'
    );

    UPDATE compliance.compliance_rule
    SET description = 'Flags when firm-wide voting rights in an EU issuer reach or cross statutory tiers (5%, 10%, 15%, 20%, 25%, 30%, 50%, 75%) under EU Transparency Directive Art. 9.',
        data_provenance = 'SYNTHETIC_FIXTURE',
        rule_family_id = v_fam_major,
        jurisdiction_code = 'EU_ESMA',
        aggregation_scope = 'FIRM_WIDE_UBO',
        filing_deadline_type = 'TRADING_DAYS',
        filing_deadline_value = 4,
        filing_cutoff_time = '17:30:00',
        filing_deadline_hours = 96
    WHERE id = v_r3;

    -- Rule 40: POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30
    v_r4 := compliance.seed_core_rule(
        'POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30',
        'UK Takeover Code Rule 9 Mandatory Cash Offer Trigger (>=30%)',
        'POST',
        'HARD_BLOCK',
        95,
        'The Takeover Code (The City Code on Takeovers and Mergers) Rule 9.1(a) & Rule 9.5; Companies Act 2006 Part 28',
        ARRAY['UK_TAKEOVER_PANEL', 'GLOBAL'],
        '{"mandatory_bid_threshold_pct": "0.300000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.firmwide_voting_control_pct"},"operator":"LESS_THAN","right":{"type":"PARAM","name":"mandatory_bid_threshold_pct"}}'::jsonb,
        'ACTIVE'
    );

    UPDATE compliance.compliance_rule
    SET description = 'Flags when aggregate firm-wide voting control in a UK takeover-target company reaches or exceeds 30.0%, triggering mandatory general cash offer obligations under Rule 9.1.',
        data_provenance = 'SYNTHETIC_FIXTURE',
        rule_family_id = v_fam_takeover,
        jurisdiction_code = 'UK_TAKEOVER_PANEL',
        aggregation_scope = 'FIRM_WIDE_UBO',
        filing_deadline_type = 'IMMEDIATE',
        filing_deadline_value = 0,
        filing_cutoff_time = 'IMMEDIATE',
        filing_deadline_hours = 0
    WHERE id = v_r4;

    -- Rule 41: POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01
    v_r5 := compliance.seed_core_rule(
        'POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01',
        'EU SSR Net Short Position Regulatory Notification (>=0.10% + 0.1% Steps)',
        'POST',
        'HARD_BLOCK',
        90,
        'Regulation (EU) No 236/2012 on short selling and certain aspects of credit default swaps (SSR) Art. 5(1) & Art. 6(1); Commission Delegated Regulation (EU) 2022/27',
        ARRAY['EU_ESMA', 'GLOBAL'],
        '{"notification_threshold_pct": "0.001000", "step_increment_pct": "0.001000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.firmwide_net_short_pct"},"operator":"LESS_THAN","right":{"type":"PARAM","name":"notification_threshold_pct"}}'::jsonb,
        'ACTIVE'
    );

    UPDATE compliance.compliance_rule
    SET description = 'Flags when firm-wide net short position in EU shares reaches or crosses 0.10% (and each 0.1% increment thereafter) under EU Short Selling Regulation Art. 5.',
        data_provenance = 'SYNTHETIC_FIXTURE',
        rule_family_id = v_fam_ssr,
        jurisdiction_code = 'EU_ESMA',
        aggregation_scope = 'FIRM_WIDE_UBO',
        filing_deadline_type = 'NEXT_TRADING_DAY_CUTOFF',
        filing_deadline_value = 1,
        filing_cutoff_time = '15:30:00',
        filing_deadline_hours = 24
    WHERE id = v_r5;

    -- Rule 42: POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02
    v_r6 := compliance.seed_core_rule(
        'POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02',
        'UK FCA SSR Net Short Position Regulatory Notification (>=0.20% + 0.1% Steps)',
        'POST',
        'HARD_BLOCK',
        90,
        'UK Short Selling Regulation (SI 2012/2911) Art. 5 & Art. 6; FCA Handbook Short Selling Sourcebook',
        ARRAY['UK_FCA', 'GLOBAL'],
        '{"notification_threshold_pct": "0.002000", "step_increment_pct": "0.001000"}'::jsonb,
        '{"type":"COMPARISON","left":{"type":"METRIC","path":"portfolio.firmwide_net_short_pct"},"operator":"LESS_THAN","right":{"type":"PARAM","name":"notification_threshold_pct"}}'::jsonb,
        'ACTIVE'
    );

    UPDATE compliance.compliance_rule
    SET description = 'Flags when firm-wide net short position in UK shares reaches or crosses 0.20% (and each 0.1% increment thereafter) under UK Short Selling Regulation.',
        data_provenance = 'SYNTHETIC_FIXTURE',
        rule_family_id = v_fam_ssr,
        jurisdiction_code = 'UK_FCA',
        aggregation_scope = 'FIRM_WIDE_UBO',
        filing_deadline_type = 'NEXT_TRADING_DAY_CUTOFF',
        filing_deadline_value = 1,
        filing_cutoff_time = '15:30:00',
        filing_deadline_hours = 24
    WHERE id = v_r6;

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
            WHEN 'POST_TRADE_SEC_SCHEDULE_13D_5PCT'        THEN '07cf6f49dbae8e504a023b524e219304d70643e68b578d0bed03ea682bfa9f0d'
            WHEN 'POST_TRADE_UK_FCA_DTR5_INITIAL_3PCT'     THEN 'd3a84cfe55cc68b5dec1745b18a3654a85b9aeb16b332ce9db3cb3e7a80e272e'
            WHEN 'POST_TRADE_EU_TRANSPARENCY_DIR_5PCT'     THEN '0a8c5e70ef00a764812ac4cbe0ca0dac3165b68bb9916e2f0554602e0edd24df'
            WHEN 'POST_TRADE_UK_TAKEOVER_MANDATORY_BID_30' THEN '44f9f009f478a4f5e9faffac72abb096d61660753a8f4dea132824b149d4bbec'
            WHEN 'POST_TRADE_EU_SSR_SHORT_DISCLOSURE_01'   THEN '1d55c8582845eca1cbdabf972743a74c7972da46c6515e3f1f32a32633fd2a63'
            WHEN 'POST_TRADE_UK_FCA_SSR_SHORT_DISCLOSURE_02' THEN '16b6af93f5c454fa845b70bfd3446cd9d7bee5246bb0b4d2255e5db4a5cec549'
            ELSE encode(sha256(('v1|' || r.ast_condition::text || '|' || r.parameter_thresholds::text || '|' || COALESCE(r.citation, ''))::bytea), 'hex')
        END,
        COALESCE(encode(sha256(r.compiled_bytecode), 'hex'), encode(sha256(''::bytea), 'hex')),
        'system_seed_v1',
        now()
    FROM compliance.compliance_rule r
    WHERE r.id IN (v_r1, v_r2, v_r3, v_r4, v_r5, v_r6)
    ON CONFLICT (rule_id, version) DO NOTHING;

    -- Assign all 6 Phase VII rules to POST_TRADE_MONITORING ruleset pack (Pack 4)
    INSERT INTO compliance.compliance_ruleset_membership (ruleset_code, rule_id)
    SELECT 'POST_TRADE_MONITORING', r.id
    FROM compliance.compliance_rule r
    WHERE r.tenant_id = v_gold_tenant
      AND r.id IN (v_r1, v_r2, v_r3, v_r4, v_r5, v_r6)
    ON CONFLICT (ruleset_code, rule_id) DO NOTHING;

END $$;
