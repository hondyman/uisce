-- 20261218_002_governance_audit_and_privileges.down.sql
--
-- Rollback Governance Audit Trail, Incident Views & Privilege Scoping

DROP VIEW IF EXISTS compliance.v_default_partition_incident CASCADE;
DROP TABLE IF EXISTS compliance.governance_audit_event CASCADE;

ALTER TABLE compliance.compliance_evaluation_event 
    DROP CONSTRAINT IF EXISTS compliance_evaluation_event_rule_id_fkey;

ALTER TABLE compliance.compliance_evaluation_event 
    ADD CONSTRAINT compliance_evaluation_event_rule_id_fkey 
    FOREIGN KEY (rule_id) REFERENCES compliance.compliance_rule(id) ON DELETE CASCADE;
