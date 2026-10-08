-- 20261222_011_phase1_counterparty_and_group_schema.down.sql

DROP TRIGGER IF EXISTS trg_prevent_entity_rel_mutation ON master.entity_relationship_snapshot;
DROP TRIGGER IF EXISTS trg_prevent_entity_rel_truncate ON master.entity_relationship_snapshot;
DROP FUNCTION IF EXISTS master.prevent_entity_rel_mutation();
DROP FUNCTION IF EXISTS master.prevent_entity_rel_truncate();

DROP TABLE IF EXISTS master.entity_relationship_snapshot CASCADE;

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
            WHERE tenant_id = v_gold_tenant AND rule_code IN (
                'POST_TRADE_GROUP_ISSUER_20',
                'POST_TRADE_ISSUER_DEBT_15',
                'POST_TRADE_COUNTERPARTY_PFE_10',
                'POST_TRADE_CASH_MIN_5'
            )
        );
        ALTER TABLE compliance.compliance_rule_version ENABLE TRIGGER ALL;
    END IF;

    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'compliance' AND table_name = 'compliance_rule') THEN
        ALTER TABLE compliance.compliance_rule DISABLE TRIGGER ALL;
        DELETE FROM compliance.compliance_rule 
        WHERE tenant_id = v_gold_tenant AND rule_code IN (
            'POST_TRADE_GROUP_ISSUER_20',
            'POST_TRADE_ISSUER_DEBT_15',
            'POST_TRADE_COUNTERPARTY_PFE_10',
            'POST_TRADE_CASH_MIN_5'
        );
        ALTER TABLE compliance.compliance_rule ENABLE TRIGGER ALL;
    END IF;
END $$;
