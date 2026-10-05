-- 20261218_002_governance_audit_and_privileges.up.sql
--
-- Compliance Governance Audit Trail, Incident Views & Least-Privilege Scoping
-- Fixes referential integrity: ON DELETE RESTRICT prevents cascade backdoors into audit logs.

-- 1. Governance Audit Event Table
CREATE TABLE IF NOT EXISTS compliance.governance_audit_event (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID NOT NULL,
    rule_id              UUID NOT NULL REFERENCES compliance.compliance_rule(id) ON DELETE RESTRICT,
    event_type           TEXT NOT NULL CHECK (event_type IN ('CORE_VERSION_PUBLISHED', 'RULE_REPINNED', 'DRIFT_RECONCILED', 'THRESHOLD_OVERRIDE')),
    old_pinned_version   INT NOT NULL,
    new_pinned_version   INT NOT NULL,
    steward_id           TEXT NOT NULL,
    steward_notes        TEXT,
    ast_diff             JSONB NOT NULL DEFAULT '{}'::jsonb,
    corpus_run_results   JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_compliance_gov_audit_tenant 
    ON compliance.governance_audit_event (tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_compliance_gov_audit_rule 
    ON compliance.governance_audit_event (rule_id);

ALTER TABLE compliance.governance_audit_event ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.governance_audit_event FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_governance_audit ON compliance.governance_audit_event
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
    );

CREATE OR REPLACE TRIGGER trg_prevent_gov_audit_mutation
    BEFORE UPDATE OR DELETE ON compliance.governance_audit_event
    FOR EACH ROW EXECUTE FUNCTION compliance.prevent_evaluation_mutation();

CREATE OR REPLACE TRIGGER trg_prevent_gov_audit_truncate
    BEFORE TRUNCATE ON compliance.governance_audit_event
    FOR EACH STATEMENT EXECUTE FUNCTION compliance.prevent_evaluation_mutation();

-- 2. Enforce ON DELETE RESTRICT on evaluation event foreign key
ALTER TABLE compliance.compliance_evaluation_event 
    DROP CONSTRAINT IF EXISTS compliance_evaluation_event_rule_id_fkey;

ALTER TABLE compliance.compliance_evaluation_event 
    ADD CONSTRAINT compliance_evaluation_event_rule_id_fkey 
    FOREIGN KEY (rule_id) REFERENCES compliance.compliance_rule(id) ON DELETE RESTRICT;

-- 3. Default Partition Incident Monitoring View
CREATE OR REPLACE VIEW compliance.v_default_partition_incident AS
    SELECT count(*) AS default_partition_incident_count
    FROM compliance.compliance_evaluation_event_default;

-- 4. Scope Least-Privilege Grants for app_user
GRANT USAGE ON SCHEMA compliance TO app_user;
GRANT SELECT, INSERT ON compliance.compliance_evaluation_event TO app_user;
GRANT SELECT, INSERT ON compliance.governance_audit_event TO app_user;
GRANT SELECT, INSERT, UPDATE ON compliance.compliance_rule TO app_user;
GRANT SELECT, INSERT, UPDATE ON compliance.compliance_watermark_checkpoint TO app_user;
GRANT SELECT, INSERT ON compliance.compliance_archive_manifest TO app_user;
GRANT SELECT, INSERT, UPDATE ON compliance.compliance_orphan_object TO app_user;
GRANT SELECT ON compliance.v_default_partition_incident TO app_user;

-- Revoke all mutations and truncates from app role on audit and governance tables
REVOKE UPDATE, DELETE, TRUNCATE ON compliance.compliance_evaluation_event FROM app_user;
REVOKE UPDATE, DELETE, TRUNCATE ON compliance.governance_audit_event FROM app_user;
REVOKE TRUNCATE ON ALL TABLES IN SCHEMA compliance FROM app_user;
REVOKE TRUNCATE ON ALL TABLES IN SCHEMA compliance FROM public;
