-- 20261223_012_phase1_tranche2_rules_and_pack.down.sql
-- Rollback Phase 1 Tranche 2 Rules and Pack Membership

DO $$
DECLARE
    v_gold_tenant UUID;
BEGIN
    SELECT id INTO v_gold_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant IS NULL THEN
        SELECT '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid INTO v_gold_tenant;
    END IF;

    -- 1. Remove ruleset memberships
    DELETE FROM compliance.compliance_ruleset_membership
    WHERE ruleset_code = 'POST_TRADE_MONITORING';

    -- 2. Remove rule versions and rules
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'compliance' AND table_name = 'compliance_rule_version') THEN
        ALTER TABLE compliance.compliance_rule_version DISABLE TRIGGER ALL;
        DELETE FROM compliance.compliance_rule_version
        WHERE rule_id IN (
            SELECT id FROM compliance.compliance_rule
            WHERE tenant_id = v_gold_tenant
              AND rule_code IN (
                  'POST_TRADE_SOVEREIGN_EXPOSURE_35',
                  'POST_TRADE_AGENCY_SUPRA_25',
                  'POST_TRADE_MUNI_OBLIGOR_10',
                  'POST_TRADE_CCP_CLEARING_EXPOSURE_15',
                  'POST_TRADE_CUSTODIAN_CONCENTRATION_20',
                  'POST_TRADE_BANK_DEPOSIT_20',
                  'POST_TRADE_SEC_LENDING_COLLATERAL_102'
              )
        );
        ALTER TABLE compliance.compliance_rule_version ENABLE TRIGGER ALL;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'compliance' AND table_name = 'compliance_rule') THEN
        ALTER TABLE compliance.compliance_rule DISABLE TRIGGER ALL;
        DELETE FROM compliance.compliance_rule
        WHERE tenant_id = v_gold_tenant
          AND rule_code IN (
              'POST_TRADE_SOVEREIGN_EXPOSURE_35',
              'POST_TRADE_AGENCY_SUPRA_25',
              'POST_TRADE_MUNI_OBLIGOR_10',
              'POST_TRADE_CCP_CLEARING_EXPOSURE_15',
              'POST_TRADE_CUSTODIAN_CONCENTRATION_20',
              'POST_TRADE_BANK_DEPOSIT_20',
              'POST_TRADE_SEC_LENDING_COLLATERAL_102'
          );
        ALTER TABLE compliance.compliance_rule ENABLE TRIGGER ALL;
    END IF;
END $$;
