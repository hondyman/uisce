-- 20260904_001_violation_audit_chain.up.sql
-- Cryptographic Glassbox Audit Chain & Root Anchoring for validation rule violations.

ALTER TABLE validation_rule_violations
    ADD COLUMN IF NOT EXISTS rule_version INT NOT NULL DEFAULT 1,
    ADD COLUMN IF NOT EXISTS record_hash TEXT,
    ADD COLUMN IF NOT EXISTS seq BIGINT GENERATED ALWAYS AS IDENTITY,
    ADD COLUMN IF NOT EXISTS chain_hash TEXT;

CREATE INDEX IF NOT EXISTS idx_vrv_seq ON validation_rule_violations (seq) WHERE chain_hash IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_vrv_unchained ON validation_rule_violations (tenant_id, seq) WHERE chain_hash IS NULL;

CREATE TABLE IF NOT EXISTS violation_audit_anchors (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID NOT NULL,
    seq_from       BIGINT NOT NULL,
    seq_to         BIGINT NOT NULL,
    anchor_hash    TEXT NOT NULL,
    row_count      INT NOT NULL,
    external_sink  TEXT,
    anchored_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, seq_to)
);

CREATE INDEX IF NOT EXISTS idx_vaa_tenant ON violation_audit_anchors (tenant_id, anchored_at DESC);
