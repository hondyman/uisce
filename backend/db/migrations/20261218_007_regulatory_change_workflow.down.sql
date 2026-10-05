-- 20261218_007_regulatory_change_workflow.down.sql
--
-- Rollback for Migration 007: Regulatory Change Workflow & Custom-Rule Version Semantics

-- 1. Drop Operational Metrics View
DROP VIEW IF EXISTS compliance.v_unaddressed_regulatory_cases CASCADE;

-- 2. Drop In-App Compliance Notification Table
DROP TABLE IF EXISTS compliance.compliance_notification CASCADE;

-- 3. Drop Append-Only Case Event Log
DROP TABLE IF EXISTS compliance.regulatory_case_event CASCADE;

-- 4. Drop Durable Draft Rules Table
DROP TABLE IF EXISTS compliance.regulatory_draft_rule CASCADE;

-- 5. Drop Regulatory Change Case Table and Triggers
DROP TRIGGER IF EXISTS trg_case_transition ON compliance.regulatory_change_case;
DROP FUNCTION IF EXISTS compliance.validate_case_transition();
DROP TABLE IF EXISTS compliance.regulatory_change_case CASCADE;

-- 6. Revert enforce_rule_version_snapshot Trigger to 006 (pinned_core_version)
CREATE OR REPLACE FUNCTION compliance.enforce_rule_version_snapshot() RETURNS trigger AS $$
BEGIN
    IF OLD.ast_condition IS DISTINCT FROM NEW.ast_condition
       OR OLD.parameter_thresholds IS DISTINCT FROM NEW.parameter_thresholds
       OR OLD.citation IS DISTINCT FROM NEW.citation
       OR OLD.pinned_core_version IS DISTINCT FROM NEW.pinned_core_version THEN

        IF NOT EXISTS (
            SELECT 1 FROM compliance.compliance_rule_version
            WHERE rule_id = NEW.id 
              AND version = COALESCE(NEW.pinned_core_version, 1)
              AND content_hash = compliance.compute_rule_content_hash(NEW.ast_condition, NEW.parameter_thresholds, NEW.citation)
        ) THEN
            RAISE EXCEPTION 'Audit Violation: Mutation of compliance_rule (id=%, version=%) disallowed without inserting matching compliance_rule_version snapshot in the same transaction.', NEW.id, COALESCE(NEW.pinned_core_version, 1);
        END IF;
    END IF;

    RETURN NEW;
END $$ LANGUAGE plpgsql;

-- 7. Drop current_version column from compliance_rule
ALTER TABLE compliance.compliance_rule
    DROP COLUMN IF EXISTS current_version;
