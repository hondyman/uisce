-- 20260904_001_violation_audit_chain.down.sql
DROP TABLE IF EXISTS violation_audit_anchors;
DROP INDEX IF EXISTS idx_vrv_unchained;
DROP INDEX IF EXISTS idx_vrv_seq;
ALTER TABLE validation_rule_violations
    DROP COLUMN IF EXISTS chain_hash,
    DROP COLUMN IF EXISTS seq,
    DROP COLUMN IF EXISTS record_hash,
    DROP COLUMN IF EXISTS rule_version;
