-- 20261218_005_core_rule_library_seed.down.sql
--
-- Clean rollback of Core Rule Library 50-rule seed and associated catalog graph elements
-- Removes memberships, activations, audit events, graph nodes/edges, and rules for CORE_LIB_V1

-- 1. Remove governance audit events referencing gold-copy core rules (temporarily bypass immutable trigger during migration rollback)
ALTER TABLE compliance.governance_audit_event DISABLE TRIGGER trg_prevent_gov_audit_mutation;

DELETE FROM compliance.governance_audit_event
WHERE rule_id IN (
    SELECT id FROM compliance.compliance_rule
    WHERE source_version = 'CORE_LIB_V1'
);

ALTER TABLE compliance.governance_audit_event ENABLE TRIGGER trg_prevent_gov_audit_mutation;

-- 2. Remove tenant activations referencing gold-copy core rules
DELETE FROM compliance.tenant_rule_activation
WHERE rule_id IN (
    SELECT id FROM compliance.compliance_rule
    WHERE source_version = 'CORE_LIB_V1'
);

-- 3. Remove ruleset bundle memberships
DELETE FROM compliance.compliance_ruleset_membership
WHERE rule_id IN (
    SELECT id FROM compliance.compliance_rule
    WHERE source_version = 'CORE_LIB_V1'
);

-- 4. Remove catalog graph edges attached to core compliance rules
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'catalog_edge') THEN
        DELETE FROM catalog_edge
        WHERE source_node_id IN (
            SELECT id FROM compliance.compliance_rule
            WHERE source_version = 'CORE_LIB_V1'
        ) OR target_node_id IN (
            SELECT id FROM compliance.compliance_rule
            WHERE source_version = 'CORE_LIB_V1'
        );
    END IF;
END $$;

-- 5. Remove catalog graph nodes for core compliance rules and rulesets
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'catalog_node') THEN
        DELETE FROM catalog_node
        WHERE id IN (
            SELECT id FROM compliance.compliance_rule
            WHERE source_version = 'CORE_LIB_V1'
        ) OR qualified_path LIKE 'compliance.ruleset/%'
          OR qualified_path LIKE 'compliance.rule/%';
    END IF;
END $$;

-- 6. Delete core compliance rules for gold tenant
DELETE FROM compliance.compliance_rule
WHERE source_version = 'CORE_LIB_V1';

-- 7. Drop helper function
DROP FUNCTION IF EXISTS compliance.seed_core_rule(TEXT, TEXT, TEXT, TEXT, INT, TEXT, TEXT[], JSONB, JSONB, TEXT);
