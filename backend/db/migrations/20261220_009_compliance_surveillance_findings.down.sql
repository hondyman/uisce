-- 20261220_009_compliance_surveillance_findings.down.sql
--
-- Rollback for Post-Trade Surveillance Workflow & Finding Persistence

DROP VIEW IF EXISTS compliance.v_unaddressed_surveillance_findings;

DROP TRIGGER IF EXISTS trg_guard_surveillance_event_append_only ON compliance.compliance_surveillance_event;
DROP FUNCTION IF EXISTS compliance.guard_surveillance_event_append_only();
DROP TABLE IF EXISTS compliance.compliance_surveillance_event;

DROP TRIGGER IF EXISTS trg_validate_surveillance_finding_transition ON compliance.compliance_surveillance_finding;
DROP FUNCTION IF EXISTS compliance.validate_surveillance_finding_transition();
DROP TABLE IF EXISTS compliance.compliance_surveillance_finding;
