-- 20261218_004_core_rule_library.down.sql
--
-- Revert Core Rule Library Schema Extensions

DROP TABLE IF EXISTS compliance.compliance_ruleset_membership;
DROP TABLE IF EXISTS compliance.tenant_rule_activation;

ALTER TABLE compliance.governance_audit_event 
    DROP CONSTRAINT IF EXISTS governance_audit_event_event_type_check;

ALTER TABLE compliance.governance_audit_event 
    ADD CONSTRAINT governance_audit_event_event_type_check 
    CHECK (event_type IN ('CORE_VERSION_PUBLISHED', 'RULE_REPINNED', 'DRIFT_RECONCILED', 'THRESHOLD_OVERRIDE'));

DROP INDEX IF EXISTS compliance.idx_compliance_rule_tenant_effective;
DROP INDEX IF EXISTS compliance.unq_compliance_rule_tenant_code;

ALTER TABLE compliance.compliance_rule
    DROP COLUMN IF EXISTS library_status,
    DROP COLUMN IF EXISTS source_version,
    DROP COLUMN IF EXISTS jurisdictions,
    DROP COLUMN IF EXISTS citation,
    DROP COLUMN IF EXISTS effective_to,
    DROP COLUMN IF EXISTS effective_from;
