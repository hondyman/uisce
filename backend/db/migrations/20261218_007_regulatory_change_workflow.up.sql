-- 20261218_007_regulatory_change_workflow.up.sql
--
-- Core Compliance Engine — Regulatory Change Workflow & Custom-Rule Version Semantics:
-- 1. Unified current_version column on compliance.compliance_rule for all inherit modes
-- 2. Update structural snapshot trigger trg_enforce_rule_version_snapshot to validate current_version
-- 3. compliance.regulatory_change_case with strict structural transition validation
-- 4. compliance.regulatory_draft_rule for durable proposal storage & cryptographic approval binding
-- 5. compliance.regulatory_case_event append-only audit ledger
-- 6. compliance.compliance_notification for blotter queryable history and drift alerts
-- 7. compliance.v_unaddressed_regulatory_cases operational metrics view
-- 8. Privileges and Row Level Security for app_user

-- 1. Custom-rule & Unified Version Semantics
ALTER TABLE compliance.compliance_rule
    ADD COLUMN IF NOT EXISTS current_version INT NOT NULL DEFAULT 1;

-- Update trigger trg_enforce_rule_version_snapshot to validate NEW.current_version
CREATE OR REPLACE FUNCTION compliance.enforce_rule_version_snapshot() RETURNS trigger AS $$
BEGIN
    -- If logic, thresholds, citation, or current_version changed, verify that a corresponding rule version snapshot exists
    IF OLD.ast_condition IS DISTINCT FROM NEW.ast_condition
       OR OLD.parameter_thresholds IS DISTINCT FROM NEW.parameter_thresholds
       OR OLD.citation IS DISTINCT FROM NEW.citation
       OR OLD.current_version IS DISTINCT FROM NEW.current_version THEN

        IF NOT EXISTS (
            SELECT 1 FROM compliance.compliance_rule_version
            WHERE rule_id = NEW.id 
              AND version = COALESCE(NEW.current_version, 1)
              AND content_hash = compliance.compute_rule_content_hash(NEW.ast_condition, NEW.parameter_thresholds, NEW.citation)
        ) THEN
            RAISE EXCEPTION 'Audit Violation: Mutation of compliance_rule (id=%, version=%) disallowed without inserting matching compliance_rule_version snapshot in the same transaction.', NEW.id, COALESCE(NEW.current_version, 1);
        END IF;
    END IF;

    RETURN NEW;
END $$ LANGUAGE plpgsql;

-- 2. Regulatory Change Case Table
CREATE TABLE IF NOT EXISTS compliance.regulatory_change_case (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_code               TEXT NOT NULL,
    source                  TEXT NOT NULL CHECK (source IN
                                ('REGULATOR_PUBLICATION','ESMA_QA','SCHEDULED_REVIEW',
                                 'TENANT_STEWARD','INTERNAL','CLIENT_INTAKE')),
    source_reference        TEXT,
    title                   TEXT NOT NULL,
    description             TEXT NOT NULL,
    affected_rule_ids       UUID[] NOT NULL DEFAULT '{}',
    classification          TEXT CHECK (classification IN
                                ('NO_IMPACT','INTERPRETATION_ONLY','EDITORIAL','PARAMETER_CHANGE',
                                 'SEMANTIC_CHANGE','NEW_RULE_REQUIRED')),
    triage_notes            TEXT,
    triaged_by              TEXT,
    triaged_at              TIMESTAMPTZ,
    status                  TEXT NOT NULL DEFAULT 'INTAKED' CHECK (status IN
                                ('INTAKED','TRIAGED','UNDER_REVIEW','APPROVED_FOR_PUBLISH',
                                 'PUBLISHED','CLOSED_NO_IMPACT','REJECTED','ESCALATED',
                                 'NEW_RULE_BACKLOG','EXPIRED')),
    is_escalated            BOOLEAN NOT NULL DEFAULT false,
    escalation_count        INT NOT NULL DEFAULT 0,
    escalated_at            TIMESTAMPTZ,
    last_escalated_at       TIMESTAMPTZ,
    published_rule_versions JSONB,
    due_at                  TIMESTAMPTZ NOT NULL,
    created_by              TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS unq_regulatory_case_code 
    ON compliance.regulatory_change_case (case_code);

CREATE INDEX IF NOT EXISTS idx_regulatory_case_status_due 
    ON compliance.regulatory_change_case (status, due_at);

ALTER TABLE compliance.regulatory_change_case ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.regulatory_change_case FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_regulatory_case ON compliance.regulatory_change_case;
CREATE POLICY tenant_isolation_regulatory_case ON compliance.regulatory_change_case
    FOR ALL USING (true); -- Regulatory cases are platform-wide master governance artifacts

-- Structural Case State Machine Validator
CREATE OR REPLACE FUNCTION compliance.validate_case_transition()
RETURNS trigger AS $$
DECLARE
    v_legal TEXT[];
BEGIN
    -- If status has not changed, validate in-place requirements and allow update
    IF NEW.status = OLD.status THEN
        IF NEW.status = 'TRIAGED' AND NEW.classification IS NULL THEN
            RAISE EXCEPTION 'triage requires classification';
        END IF;
        NEW.updated_at := now();
        RETURN NEW;
    END IF;

    v_legal := CASE OLD.status
        WHEN 'INTAKED'              THEN ARRAY['TRIAGED','REJECTED','ESCALATED','NEW_RULE_BACKLOG']
        WHEN 'TRIAGED'              THEN ARRAY['UNDER_REVIEW','CLOSED_NO_IMPACT','REJECTED','NEW_RULE_BACKLOG','ESCALATED']
        WHEN 'UNDER_REVIEW'         THEN ARRAY['APPROVED_FOR_PUBLISH','REJECTED','ESCALATED']
        WHEN 'APPROVED_FOR_PUBLISH' THEN ARRAY['PUBLISHED','REJECTED','ESCALATED','EXPIRED']
        WHEN 'NEW_RULE_BACKLOG'     THEN ARRAY['UNDER_REVIEW','CLOSED_NO_IMPACT','REJECTED','ESCALATED']
        WHEN 'ESCALATED'            THEN ARRAY['TRIAGED','UNDER_REVIEW','APPROVED_FOR_PUBLISH','PUBLISHED','CLOSED_NO_IMPACT','REJECTED','NEW_RULE_BACKLOG','ESCALATED']
        ELSE '{}' END;

    IF NOT (NEW.status = ANY(v_legal)) THEN
        RAISE EXCEPTION 'illegal case transition % -> %', OLD.status, NEW.status;
    END IF;

    IF NEW.status = 'TRIAGED' AND NEW.classification IS NULL THEN
        RAISE EXCEPTION 'triage requires classification';
    END IF;

    IF NEW.status = 'ESCALATED' OR (NEW.is_escalated = true AND OLD.is_escalated = false) THEN
        NEW.is_escalated := true;
        NEW.escalated_at := COALESCE(OLD.escalated_at, now());
        NEW.last_escalated_at := now();
        NEW.escalation_count := OLD.escalation_count + 1;
    END IF;

    NEW.updated_at := now();
    RETURN NEW;
END $$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_case_transition ON compliance.regulatory_change_case;
CREATE TRIGGER trg_case_transition 
    BEFORE UPDATE ON compliance.regulatory_change_case
    FOR EACH ROW EXECUTE FUNCTION compliance.validate_case_transition();

-- 3. Durable Draft Rule Storage (Content-Bound Approval & Verification)
CREATE TABLE IF NOT EXISTS compliance.regulatory_draft_rule (
    id                            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id                       UUID NOT NULL REFERENCES compliance.regulatory_change_case(id) ON DELETE CASCADE,
    rule_id                       UUID NOT NULL REFERENCES compliance.compliance_rule(id) ON DELETE RESTRICT,
    proposed_ast                  JSONB NOT NULL,
    proposed_parameter_thresholds JSONB NOT NULL,
    proposed_citation             TEXT,
    proposed_content_hash         TEXT NOT NULL CHECK (char_length(proposed_content_hash) = 64),
    corpus_results                JSONB NOT NULL DEFAULT '{}',
    is_approved                   BOOLEAN NOT NULL DEFAULT false,
    approved_by                   TEXT,
    approved_at                   TIMESTAMPTZ,
    created_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unq_regulatory_draft_rule_case_rule UNIQUE (case_id, rule_id)
);

CREATE INDEX IF NOT EXISTS idx_regulatory_draft_rule_lookup 
    ON compliance.regulatory_draft_rule (case_id, rule_id);

ALTER TABLE compliance.regulatory_draft_rule ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.regulatory_draft_rule FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_draft_rules ON compliance.regulatory_draft_rule;
CREATE POLICY tenant_isolation_draft_rules ON compliance.regulatory_draft_rule
    FOR ALL USING (true);

-- 4. Append-Only Case Event Log
CREATE TABLE IF NOT EXISTS compliance.regulatory_case_event (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    case_id     UUID NOT NULL REFERENCES compliance.regulatory_change_case(id) ON DELETE RESTRICT,
    event_type  TEXT NOT NULL CHECK (event_type IN
                    ('CASE_OPENED','TRIAGED','REVIEW_STARTED','STAKEHOLDER_NOTED',
                     'CORPUS_RUN','APPROVED','PUBLISHED','CLOSED','ESCALATED',
                     'NEW_RULE_ROUTED','EXPIRED','REJECTED')),
    actor       TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_case_events 
    ON compliance.regulatory_case_event (case_id, created_at);

ALTER TABLE compliance.regulatory_case_event ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.regulatory_case_event FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_case_events ON compliance.regulatory_case_event;
CREATE POLICY tenant_isolation_case_events ON compliance.regulatory_case_event
    FOR ALL USING (true);

DROP TRIGGER IF EXISTS trg_prevent_case_event_mutation ON compliance.regulatory_case_event;
CREATE TRIGGER trg_prevent_case_event_mutation
    BEFORE UPDATE OR DELETE ON compliance.regulatory_case_event
    FOR EACH ROW EXECUTE FUNCTION compliance.prevent_evaluation_mutation();

DROP TRIGGER IF EXISTS trg_prevent_case_event_truncate ON compliance.regulatory_case_event;
CREATE TRIGGER trg_prevent_case_event_truncate
    BEFORE TRUNCATE ON compliance.regulatory_case_event
    FOR EACH STATEMENT EXECUTE FUNCTION compliance.prevent_evaluation_mutation();

-- 5. In-App Queryable Compliance Notification Table
CREATE TABLE IF NOT EXISTS compliance.compliance_notification (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL,
    kind        TEXT NOT NULL CHECK (kind IN
                    ('INHERIT_ADVANCE','DRIFT_FLAG','STALE_REGULATION','CASE_ESCALATED','SYSTEM')),
    title       TEXT NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}',
    is_read     BOOLEAN NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_compliance_notification_lookup 
    ON compliance.compliance_notification (tenant_id, created_at DESC);

ALTER TABLE compliance.compliance_notification ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_notification FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS tenant_isolation_notification ON compliance.compliance_notification;
CREATE POLICY tenant_isolation_notification ON compliance.compliance_notification
    FOR ALL USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR tenant_id = public.uisce_gold_copy_tenant_id()
        OR current_setting('app.is_admin', true) = 'true'
    );

-- 6. Operational Metrics View: Unaddressed & Overdue Regulatory Cases
CREATE OR REPLACE VIEW compliance.v_unaddressed_regulatory_cases AS
SELECT 
    id AS case_id,
    case_code,
    title,
    source,
    source_reference,
    status,
    classification,
    is_escalated,
    escalation_count,
    due_at,
    created_at,
    triaged_at,
    published_rule_versions,
    CASE 
        WHEN status IN ('PUBLISHED','CLOSED_NO_IMPACT','REJECTED') THEN 0
        ELSE GREATEST(0, EXTRACT(EPOCH FROM (now() - due_at))::BIGINT)
    END AS overdue_seconds,
    CASE 
        WHEN status IN ('PUBLISHED','CLOSED_NO_IMPACT','REJECTED') THEN false
        ELSE now() > due_at
    END AS is_overdue,
    CASE 
        WHEN triaged_at IS NOT NULL THEN EXTRACT(EPOCH FROM (triaged_at - created_at))::BIGINT
        ELSE EXTRACT(EPOCH FROM (now() - created_at))::BIGINT
    END AS intake_to_triage_seconds
FROM compliance.regulatory_change_case;

-- 7. Privileges
GRANT SELECT, INSERT, UPDATE ON compliance.regulatory_change_case TO app_user;
GRANT SELECT, INSERT, UPDATE, DELETE ON compliance.regulatory_draft_rule TO app_user;
GRANT SELECT, INSERT ON compliance.regulatory_case_event TO app_user;
GRANT SELECT, INSERT, UPDATE ON compliance.compliance_notification TO app_user;
GRANT SELECT ON compliance.v_unaddressed_regulatory_cases TO app_user;
