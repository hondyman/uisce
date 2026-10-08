-- 20261224_014_provenance_least_privilege_and_audit.up.sql
--
-- Governance & Provenance Hardening:
-- 1. Invert data_provenance DEFAULT to 'SYNTHETIC_FIXTURE' (least-privilege trust baseline)
-- 2. Expand compliance.governance_audit_event event_type CHECK to include 'DATA_PROVENANCE_PROMOTED'
-- 3. Stored procedure compliance.promote_rule_data_provenance for governance-audited trust tier promotion
-- 4. Explicitly ensure Phase I rules are VENDOR_PROVEN and Phase II rules are SYNTHETIC_FIXTURE

-- 1. Invert column default to SYNTHETIC_FIXTURE
ALTER TABLE compliance.compliance_rule
    ALTER COLUMN data_provenance SET DEFAULT 'SYNTHETIC_FIXTURE';

-- 2. Ensure explicit trust tiers for existing rules
UPDATE compliance.compliance_rule
SET data_provenance = 'SYNTHETIC_FIXTURE'
WHERE rule_code IN (
    'POST_TRADE_UNCLASSIFIED_CEILING_5',
    'POST_TRADE_SECTOR_CONCENTRATION_25',
    'POST_TRADE_INDUSTRY_GROUP_15',
    'POST_TRADE_CYCLICAL_SECTOR_35',
    'POST_TRADE_EMERGING_MARKET_20',
    'POST_TRADE_NON_OECD_EXPOSURE_10',
    'POST_TRADE_FRONTIER_MARKET_5'
);

UPDATE compliance.compliance_rule
SET data_provenance = 'VENDOR_PROVEN'
WHERE rule_code NOT IN (
    'POST_TRADE_UNCLASSIFIED_CEILING_5',
    'POST_TRADE_SECTOR_CONCENTRATION_25',
    'POST_TRADE_INDUSTRY_GROUP_15',
    'POST_TRADE_CYCLICAL_SECTOR_35',
    'POST_TRADE_EMERGING_MARKET_20',
    'POST_TRADE_NON_OECD_EXPOSURE_10',
    'POST_TRADE_FRONTIER_MARKET_5'
);

-- 3. Expand governance_audit_event event_type CHECK constraint
ALTER TABLE compliance.governance_audit_event
    DROP CONSTRAINT IF EXISTS governance_audit_event_event_type_check;

ALTER TABLE compliance.governance_audit_event
    ADD CONSTRAINT governance_audit_event_event_type_check
    CHECK (event_type IN (
        'CORE_VERSION_PUBLISHED',
        'RULE_REPINNED',
        'DRIFT_RECONCILED',
        'THRESHOLD_OVERRIDE',
        'RULE_ACTIVATED',
        'RULE_DEACTIVATED',
        'REGULATORY_CHANGE_INTAKED',
        'REGULATORY_CHANGE_TRIAGED',
        'REGULATORY_CHANGE_PUBLISHED',
        'REGULATORY_CHANGE_CLOSED',
        'DATA_PROVENANCE_PROMOTED'
    ));

-- 4. Audited Promotion Procedure
CREATE OR REPLACE FUNCTION compliance.promote_rule_data_provenance(
    p_rule_id UUID,
    p_steward_id TEXT,
    p_steward_notes TEXT,
    p_vendor_source TEXT
) RETURNS VOID AS $$
DECLARE
    v_tenant_id UUID;
    v_current_version INT;
BEGIN
    SELECT tenant_id, COALESCE(current_version, 1)
    INTO v_tenant_id, v_current_version
    FROM compliance.compliance_rule
    WHERE id = p_rule_id AND valid_to IS NULL;

    IF v_tenant_id IS NULL THEN
        RAISE EXCEPTION 'Rule % not found or inactive', p_rule_id;
    END IF;

    IF p_steward_id IS NULL OR trim(p_steward_id) = '' THEN
        RAISE EXCEPTION 'Steward ID is required for data provenance promotion';
    END IF;

    IF p_vendor_source IS NULL OR trim(p_vendor_source) = '' THEN
        RAISE EXCEPTION 'Vendor data source citation is required for data provenance promotion';
    END IF;

    -- Update trust tier
    UPDATE compliance.compliance_rule
    SET data_provenance = 'VENDOR_PROVEN',
        updated_at = now()
    WHERE id = p_rule_id;

    -- Record immutable governance audit event
    INSERT INTO compliance.governance_audit_event (
        tenant_id,
        rule_id,
        event_type,
        old_pinned_version,
        new_pinned_version,
        steward_id,
        steward_notes,
        ast_diff,
        corpus_run_results
    ) VALUES (
        v_tenant_id,
        p_rule_id,
        'DATA_PROVENANCE_PROMOTED',
        v_current_version,
        v_current_version,
        p_steward_id,
        p_steward_notes,
        jsonb_build_object('vendor_source', p_vendor_source, 'prior_provenance', 'SYNTHETIC_FIXTURE', 'promoted_provenance', 'VENDOR_PROVEN'),
        '{}'::jsonb
    );
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;
