-- 20261224_014_provenance_least_privilege_and_audit.down.sql

DROP FUNCTION IF EXISTS compliance.promote_rule_data_provenance(UUID, TEXT, TEXT, TEXT);

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
        'REGULATORY_CHANGE_CLOSED'
    ));

ALTER TABLE compliance.compliance_rule
    ALTER COLUMN data_provenance SET DEFAULT 'VENDOR_PROVEN';
