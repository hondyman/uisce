-- 20261224_018_phase7_entity_graph_and_rule_families.down.sql
--
-- Rollback Phase VII Schema, Seeded Rules (Rules 37-42), and Rule Families

DO $$
DECLARE
    v_rule_37 UUID := 'c0000000-0000-4000-a000-000000000037'::uuid;
    v_rule_38 UUID := 'c0000000-0000-4000-a000-000000000038'::uuid;
    v_rule_39 UUID := 'c0000000-0000-4000-a000-000000000039'::uuid;
    v_rule_40 UUID := 'c0000000-0000-4000-a000-000000000040'::uuid;
    v_rule_41 UUID := 'c0000000-0000-4000-a000-000000000041'::uuid;
    v_rule_42 UUID := 'c0000000-0000-4000-a000-000000000042'::uuid;
BEGIN
    -- Delete ruleset memberships
    DELETE FROM compliance.compliance_ruleset_membership
    WHERE rule_id IN (v_rule_37, v_rule_38, v_rule_39, v_rule_40, v_rule_41, v_rule_42);

    -- Delete rule versions
    DELETE FROM compliance.compliance_rule_version
    WHERE rule_id IN (v_rule_37, v_rule_38, v_rule_39, v_rule_40, v_rule_41, v_rule_42);

    -- Delete rules
    DELETE FROM compliance.compliance_rule
    WHERE id IN (v_rule_37, v_rule_38, v_rule_39, v_rule_40, v_rule_41, v_rule_42);

    -- Delete Rule Families
    DELETE FROM compliance.compliance_rule_family
    WHERE family_code IN ('MAJOR_SHAREHOLDING_DISCLOSURE', 'TAKEOVER_PANEL_MONITORING', 'NET_SHORT_POSITION_DISCLOSURE');
END $$;

-- Drop restricted lists
DROP TABLE IF EXISTS compliance.compliance_restricted_list_item;
DROP TABLE IF EXISTS compliance.compliance_restricted_list;

-- Drop finding deadline column and restore standard status check constraint
ALTER TABLE compliance.compliance_finding
    DROP COLUMN IF EXISTS filing_deadline_at;

ALTER TABLE compliance.compliance_finding
    DROP CONSTRAINT IF EXISTS compliance_finding_status_check;

ALTER TABLE compliance.compliance_finding
    ADD CONSTRAINT compliance_finding_status_check
    CHECK (status IN ('OPEN', 'SUPERSEDED', 'RESOLVED', 'DISMISSED'));

-- Drop rule extensions
ALTER TABLE compliance.compliance_rule
    DROP COLUMN IF EXISTS filing_deadline_hours,
    DROP COLUMN IF EXISTS filing_cutoff_time,
    DROP COLUMN IF EXISTS filing_deadline_value,
    DROP COLUMN IF EXISTS filing_deadline_type,
    DROP COLUMN IF EXISTS aggregation_scope,
    DROP COLUMN IF EXISTS jurisdiction_code,
    DROP COLUMN IF EXISTS rule_family_id;

-- Drop rule family table
DROP TABLE IF EXISTS compliance.compliance_rule_family;

-- Drop fund hierarchy table
DROP TABLE IF EXISTS master.fund_hierarchy_edge;
