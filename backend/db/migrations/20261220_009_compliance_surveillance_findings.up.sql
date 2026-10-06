-- 20261220_009_compliance_surveillance_findings.up.sql
--
-- Core Compliance Engine — Post-Trade Surveillance Workflow & Finding Persistence:
-- 1. compliance.compliance_surveillance_finding workflow table with deterministic dedup key & temporal window semantics
-- 2. compliance.validate_surveillance_finding_transition structural state-machine validator
-- 3. compliance.compliance_surveillance_event append-only immutable audit ledger
-- 4. compliance.v_unaddressed_surveillance_findings operational queue view
-- 5. Row Level Security & Access Privileges

-- 1. Surveillance Finding Workflow Table
CREATE TABLE IF NOT EXISTS compliance.compliance_surveillance_finding (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID NOT NULL REFERENCES public.tenants(id) ON DELETE CASCADE,
    detector_type           TEXT NOT NULL CHECK (detector_type IN
                                ('WASH_SALE', 'PRO_RATA_ALLOCATION_FAIRNESS', 'CROSS_ACCOUNT_CONFLICT', 'LOOKTHROUGH_CONCENTRATION')),
    severity                TEXT NOT NULL CHECK (severity IN
                                ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL')),
    status                  TEXT NOT NULL DEFAULT 'OPEN' CHECK (status IN
                                ('OPEN', 'IN_REVIEW', 'ESCALATED', 'DISMISSED', 'REMEDIATED', 'CLOSED')),
    dedup_key               TEXT NOT NULL,
    title                   TEXT NOT NULL,
    description             TEXT NOT NULL,
    entity_id               UUID,
    entity_type             TEXT NOT NULL CHECK (entity_type IN
                                ('BENEFICIAL_OWNER', 'ACCOUNT', 'EXECUTION_BLOCK', 'SECURITY')),
    metadata                JSONB NOT NULL DEFAULT '{}'::jsonb,
    detected_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    activity_window_start   TIMESTAMPTZ NOT NULL,
    activity_window_end     TIMESTAMPTZ NOT NULL,
    assigned_to             TEXT,
    resolution_notes        TEXT,
    resolved_by             TEXT,
    resolved_at             TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unq_surveillance_finding_tenant_dedup UNIQUE (tenant_id, dedup_key)
);

CREATE INDEX IF NOT EXISTS idx_surveillance_finding_tenant_status_time 
    ON compliance.compliance_surveillance_finding (tenant_id, status, detected_at DESC);

CREATE INDEX IF NOT EXISTS idx_surveillance_finding_detector_severity 
    ON compliance.compliance_surveillance_finding (tenant_id, detector_type, severity);

CREATE INDEX IF NOT EXISTS idx_surveillance_finding_entity 
    ON compliance.compliance_surveillance_finding (tenant_id, entity_type, entity_id);

ALTER TABLE compliance.compliance_surveillance_finding ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_surveillance_finding FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_surveillance_finding ON compliance.compliance_surveillance_finding;
CREATE POLICY tenant_isolation_surveillance_finding ON compliance.compliance_surveillance_finding
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
    );

-- 2. State-Machine Transition Validator for Findings
CREATE OR REPLACE FUNCTION compliance.validate_surveillance_finding_transition()
RETURNS trigger AS $$
DECLARE
    v_legal TEXT[];
BEGIN
    -- Allow in-place updates if status has not changed
    IF NEW.status = OLD.status THEN
        NEW.updated_at := now();
        RETURN NEW;
    END IF;

    v_legal := CASE OLD.status
        WHEN 'OPEN'        THEN ARRAY['IN_REVIEW', 'DISMISSED', 'ESCALATED', 'REMEDIATED', 'CLOSED']
        WHEN 'IN_REVIEW'   THEN ARRAY['ESCALATED', 'DISMISSED', 'REMEDIATED', 'CLOSED']
        WHEN 'ESCALATED'   THEN ARRAY['IN_REVIEW', 'REMEDIATED', 'DISMISSED', 'CLOSED']
        WHEN 'REMEDIATED'  THEN ARRAY['CLOSED', 'IN_REVIEW']
        WHEN 'DISMISSED'   THEN ARRAY['IN_REVIEW', 'CLOSED']
        WHEN 'CLOSED'      THEN ARRAY[]::TEXT[] -- Terminal State
        ELSE ARRAY[]::TEXT[]
    END;

    IF NOT (NEW.status = ANY(v_legal)) THEN
        RAISE EXCEPTION 'Illegal surveillance finding status transition: % -> % (Allowed: %)',
            OLD.status, NEW.status, v_legal;
    END IF;

    -- Enforce resolution requirements
    IF NEW.status IN ('DISMISSED', 'REMEDIATED', 'CLOSED') THEN
        IF NEW.resolution_notes IS NULL OR trim(NEW.resolution_notes) = '' THEN
            RAISE EXCEPTION 'Resolution of surveillance finding (status=%) requires resolution_notes', NEW.status;
        END IF;
        IF NEW.resolved_by IS NULL OR trim(NEW.resolved_by) = '' THEN
            RAISE EXCEPTION 'Resolution of surveillance finding (status=%) requires resolved_by', NEW.status;
        END IF;
        IF NEW.resolved_at IS NULL THEN
            NEW.resolved_at := now();
        END IF;
    END IF;

    NEW.updated_at := now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_validate_surveillance_finding_transition ON compliance.compliance_surveillance_finding;
CREATE TRIGGER trg_validate_surveillance_finding_transition
    BEFORE UPDATE ON compliance.compliance_surveillance_finding
    FOR EACH ROW EXECUTE FUNCTION compliance.validate_surveillance_finding_transition();

-- 3. Append-Only Surveillance Event Ledger
CREATE TABLE IF NOT EXISTS compliance.compliance_surveillance_event (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    finding_id              UUID NOT NULL REFERENCES compliance.compliance_surveillance_finding(id) ON DELETE CASCADE,
    tenant_id               UUID NOT NULL,
    event_type              TEXT NOT NULL CHECK (event_type IN
                                ('DETECTED', 'STATUS_CHANGED', 'ASSIGNED', 'ESCALATED', 'NOTE_ADDED', 'REMEDIATED', 'DISMISSED', 'CLOSED')),
    actor                   TEXT NOT NULL,
    payload                 JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_surveillance_event_finding_time 
    ON compliance.compliance_surveillance_event (finding_id, created_at ASC);

ALTER TABLE compliance.compliance_surveillance_event ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_surveillance_event FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_surveillance_event ON compliance.compliance_surveillance_event;
CREATE POLICY tenant_isolation_surveillance_event ON compliance.compliance_surveillance_event
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
    );

-- Guard Append-Only for Surveillance Events
CREATE OR REPLACE FUNCTION compliance.guard_surveillance_event_append_only()
RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'Audit Violation: compliance.compliance_surveillance_event is strictly append-only. Mutation (UPDATE/DELETE/TRUNCATE) is disallowed.'
        USING ERRCODE = '55000';
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_guard_surveillance_event_append_only ON compliance.compliance_surveillance_event;
CREATE TRIGGER trg_guard_surveillance_event_append_only
    BEFORE UPDATE OR DELETE ON compliance.compliance_surveillance_event
    FOR EACH ROW EXECUTE FUNCTION compliance.guard_surveillance_event_append_only();

-- 4. Operational View for Active Surveillance Findings
CREATE OR REPLACE VIEW compliance.v_unaddressed_surveillance_findings AS
SELECT 
    f.id AS finding_id,
    f.tenant_id,
    f.detector_type,
    f.severity,
    f.status,
    f.dedup_key,
    f.title,
    f.description,
    f.entity_id,
    f.entity_type,
    f.metadata,
    f.detected_at,
    f.activity_window_start,
    f.activity_window_end,
    f.assigned_to,
    f.created_at,
    f.updated_at,
    EXTRACT(EPOCH FROM (now() - f.detected_at)) / 3600.0 AS age_hours
FROM compliance.compliance_surveillance_finding f
WHERE f.status IN ('OPEN', 'IN_REVIEW', 'ESCALATED');

-- 5. Privileges
GRANT SELECT, INSERT, UPDATE ON compliance.compliance_surveillance_finding TO app_user;
GRANT SELECT, INSERT ON compliance.compliance_surveillance_event TO app_user;
GRANT SELECT ON compliance.v_unaddressed_surveillance_findings TO app_user;
