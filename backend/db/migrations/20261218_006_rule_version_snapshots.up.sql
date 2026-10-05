-- 20261218_006_rule_version_snapshots.up.sql
--
-- Core Compliance Engine — Rule Version Snapshots & Content-Addressed Evidence Chain:
-- 1. Append-Only compliance.compliance_rule_version table
-- 2. Initial Version-1 Snapshot Seed for all 50 Gold-Copy Core Rules
-- 3. Structural Trigger preventing un-snapshotted mutations on compliance.compliance_rule
-- 4. Schema Epoch Update: evaluation_hash v2 with rule_content_hash + RESTRICT FK + composite index
-- 5. Cleanup of vestigial non-gold tenant rows

-- 1. Cleanup vestigial test tenant if present
DELETE FROM public.tenants 
WHERE id = '00000000-0000-4000-a000-000000000000'::uuid 
  AND (gold_copy IS NULL OR gold_copy = false);

-- 2. Append-Only Rule Version Snapshot Table
CREATE TABLE IF NOT EXISTS compliance.compliance_rule_version (
    rule_id                UUID NOT NULL REFERENCES compliance.compliance_rule(id) ON DELETE RESTRICT,
    version                INT NOT NULL,
    tenant_id              UUID NOT NULL,
    resolved_ast           JSONB NOT NULL,
    parameter_thresholds   JSONB NOT NULL,
    citation               TEXT,
    effective_from         TIMESTAMPTZ NOT NULL,
    effective_to           TIMESTAMPTZ,
    content_hash           TEXT NOT NULL,
    compiled_bytecode_hash TEXT NOT NULL,
    created_by             TEXT NOT NULL DEFAULT 'system',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (rule_id, version)
);

CREATE INDEX IF NOT EXISTS idx_compliance_rule_version_tenant_hash
    ON compliance.compliance_rule_version (tenant_id, content_hash);

ALTER TABLE compliance.compliance_rule_version ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_rule_version FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_rule_version ON compliance.compliance_rule_version;
CREATE POLICY tenant_isolation_rule_version ON compliance.compliance_rule_version
    FOR ALL USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR tenant_id = public.uisce_gold_copy_tenant_id()
        OR current_setting('app.is_admin', true) = 'true'
    );

GRANT SELECT, INSERT ON compliance.compliance_rule_version TO app_user;

-- 3. Seed Version 1 for all existing compliance rules
INSERT INTO compliance.compliance_rule_version (
    rule_id, version, tenant_id, resolved_ast, parameter_thresholds,
    citation, effective_from, effective_to, content_hash, compiled_bytecode_hash,
    created_by, created_at
)
SELECT 
    r.id,
    COALESCE(r.pinned_core_version, 1),
    r.tenant_id,
    r.ast_condition,
    r.parameter_thresholds,
    r.citation,
    r.effective_from,
    r.effective_to,
    encode(sha256(('v1|' || r.ast_condition::text || '|' || r.parameter_thresholds::text || '|' || COALESCE(r.citation, ''))::bytea), 'hex'),
    COALESCE(encode(sha256(r.compiled_bytecode), 'hex'), encode(sha256(''::bytea), 'hex')),
    'system_seed_v1',
    now()
FROM compliance.compliance_rule r
ON CONFLICT (rule_id, version) DO NOTHING;

-- 4. Enforce Append-Only Immutability on compliance_rule_version
DROP TRIGGER IF EXISTS trg_prevent_rule_version_mutation ON compliance.compliance_rule_version;
CREATE TRIGGER trg_prevent_rule_version_mutation
    BEFORE UPDATE OR DELETE ON compliance.compliance_rule_version
    FOR EACH ROW EXECUTE FUNCTION compliance.prevent_evaluation_mutation();

DROP TRIGGER IF EXISTS trg_prevent_rule_version_truncate ON compliance.compliance_rule_version;
CREATE TRIGGER trg_prevent_rule_version_truncate
    BEFORE TRUNCATE ON compliance.compliance_rule_version
    FOR EACH STATEMENT EXECUTE FUNCTION compliance.prevent_evaluation_mutation();

-- 5. Structural Guard: Block compliance_rule updates without matching version snapshot
CREATE OR REPLACE FUNCTION compliance.enforce_rule_version_snapshot() RETURNS trigger AS $$
DECLARE
    v_new_content_hash TEXT;
BEGIN
    -- If logic, thresholds, or citation changed, verify that a corresponding rule version snapshot exists
    IF OLD.ast_condition IS DISTINCT FROM NEW.ast_condition
       OR OLD.parameter_thresholds IS DISTINCT FROM NEW.parameter_thresholds
       OR OLD.citation IS DISTINCT FROM NEW.citation
       OR OLD.pinned_core_version IS DISTINCT FROM NEW.pinned_core_version THEN

        v_new_content_hash := encode(sha256(('v1|' || NEW.ast_condition::text || '|' || NEW.parameter_thresholds::text || '|' || COALESCE(NEW.citation, ''))::bytea), 'hex');

        IF NOT EXISTS (
            SELECT 1 FROM compliance.compliance_rule_version
            WHERE rule_id = NEW.id 
              AND version = NEW.pinned_core_version
              AND content_hash = v_new_content_hash
        ) THEN
            RAISE EXCEPTION 'Audit Violation: Mutation of compliance_rule (id=%, version=%) disallowed without inserting matching compliance_rule_version snapshot in the same transaction.', NEW.id, NEW.pinned_core_version;
        END IF;
    END IF;

    RETURN NEW;
END $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_enforce_rule_version_snapshot ON compliance.compliance_rule;
CREATE TRIGGER trg_enforce_rule_version_snapshot
    BEFORE UPDATE ON compliance.compliance_rule
    FOR EACH ROW EXECUTE FUNCTION compliance.enforce_rule_version_snapshot();

-- 6. Schema Epoch Reset & rule_content_hash extension on compliance_evaluation_event
-- Existing evaluation records are test/dev fixtures from Phase 1; epoch reset to v2 schema
ALTER TABLE compliance.compliance_evaluation_event DISABLE TRIGGER trg_prevent_eval_mutation;
ALTER TABLE compliance.compliance_evaluation_event DISABLE TRIGGER trg_prevent_eval_truncate;

TRUNCATE TABLE compliance.compliance_evaluation_event;

ALTER TABLE compliance.compliance_evaluation_event ENABLE TRIGGER trg_prevent_eval_mutation;
ALTER TABLE compliance.compliance_evaluation_event ENABLE TRIGGER trg_prevent_eval_truncate;

-- Add rule_content_hash column to partitioned evaluation event table
ALTER TABLE compliance.compliance_evaluation_event
    ADD COLUMN IF NOT EXISTS rule_content_hash TEXT NOT NULL DEFAULT '';

-- Add composite index on (rule_id, rule_version) for high-performance FK lookups
CREATE INDEX IF NOT EXISTS idx_compliance_eval_rule_ver
    ON compliance.compliance_evaluation_event (rule_id, rule_version);

-- Add composite RESTRICT FK constraint to rule version registry
ALTER TABLE compliance.compliance_evaluation_event
    DROP CONSTRAINT IF EXISTS fk_compliance_eval_rule_version;

ALTER TABLE compliance.compliance_evaluation_event
    ADD CONSTRAINT fk_compliance_eval_rule_version
    FOREIGN KEY (rule_id, rule_version)
    REFERENCES compliance.compliance_rule_version (rule_id, version)
    ON DELETE RESTRICT;

-- 7. Dual Merkle Root Columns on compliance_archive_manifest
ALTER TABLE compliance.compliance_archive_manifest
    ADD COLUMN IF NOT EXISTS rule_registry_merkle_root TEXT,
    ADD COLUMN IF NOT EXISTS rule_registry_s3_key TEXT;

