-- 20261221_010_post_trade_pilot_schema.down.sql

DROP TRIGGER IF EXISTS trg_validate_compliance_finding_transition ON compliance.compliance_finding;
DROP FUNCTION IF EXISTS compliance.validate_compliance_finding_transition();

DROP TABLE IF EXISTS compliance.compliance_finding CASCADE;
DROP TABLE IF EXISTS master.tenant_classification_override CASCADE;
DROP TABLE IF EXISTS master.security_master_snapshot CASCADE;
DROP TABLE IF EXISTS compliance.compliance_portfolio_snapshot CASCADE;

DO $$
DECLARE
    v_gold_tenant UUID;
BEGIN
    SELECT id INTO v_gold_tenant FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF v_gold_tenant IS NULL THEN
        v_gold_tenant := '99e99e99-99e9-49e9-89e9-99e99e99e999'::uuid;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'compliance' AND table_name = 'compliance_rule_version') THEN
        ALTER TABLE compliance.compliance_rule_version DISABLE TRIGGER ALL;
        DELETE FROM compliance.compliance_rule_version 
        WHERE rule_id IN (
            SELECT id FROM compliance.compliance_rule 
            WHERE tenant_id = v_gold_tenant AND rule_code IN ('UCITS_5_10_40', 'SEC_144A_QIB_HOLDING', 'MARGIN_UTILIZATION_80')
        );
        ALTER TABLE compliance.compliance_rule_version ENABLE TRIGGER ALL;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'compliance' AND table_name = 'compliance_evaluation_event') THEN
        ALTER TABLE compliance.compliance_evaluation_event DISABLE TRIGGER ALL;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'compliance' AND table_name = 'governance_audit_event') THEN
        ALTER TABLE compliance.governance_audit_event DISABLE TRIGGER ALL;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'compliance' AND table_name = 'compliance_rule') THEN
        ALTER TABLE compliance.compliance_rule DISABLE TRIGGER ALL;
        DELETE FROM compliance.compliance_rule 
        WHERE tenant_id = v_gold_tenant AND rule_code IN ('UCITS_5_10_40', 'SEC_144A_QIB_HOLDING', 'MARGIN_UTILIZATION_80');
        ALTER TABLE compliance.compliance_rule ENABLE TRIGGER ALL;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'compliance' AND table_name = 'governance_audit_event') THEN
        ALTER TABLE compliance.governance_audit_event ENABLE TRIGGER ALL;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'compliance' AND table_name = 'compliance_evaluation_event') THEN
        ALTER TABLE compliance.compliance_evaluation_event ENABLE TRIGGER ALL;
    END IF;
END $$;
