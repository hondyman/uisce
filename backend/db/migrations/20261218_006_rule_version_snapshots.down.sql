-- 20261218_006_rule_version_snapshots.down.sql
--
-- Revert Rule Version Snapshots & Evaluation Event Schema Extensions

-- 1. Remove FK constraint and index from compliance_evaluation_event
ALTER TABLE compliance.compliance_evaluation_event
    DROP CONSTRAINT IF EXISTS fk_compliance_eval_rule_version;

DROP INDEX IF EXISTS compliance.idx_compliance_eval_rule_ver;

ALTER TABLE compliance.compliance_evaluation_event
    DROP COLUMN IF EXISTS rule_content_hash;

ALTER TABLE compliance.compliance_archive_manifest
    DROP COLUMN IF EXISTS rule_registry_merkle_root,
    DROP COLUMN IF EXISTS rule_registry_s3_key;

-- 2. Drop structural mutation trigger from compliance_rule
DROP TRIGGER IF EXISTS trg_enforce_rule_version_snapshot ON compliance.compliance_rule;
DROP FUNCTION IF EXISTS compliance.enforce_rule_version_snapshot();

-- 3. Drop append-only triggers and compliance_rule_version table
DROP TRIGGER IF EXISTS trg_prevent_rule_version_mutation ON compliance.compliance_rule_version;
DROP TRIGGER IF EXISTS trg_prevent_rule_version_truncate ON compliance.compliance_rule_version;

DROP TABLE IF EXISTS compliance.compliance_rule_version;
