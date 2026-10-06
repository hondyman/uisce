-- 20261218_001_compliance_engine_core_tables.up.sql
--
-- Compliance Engine Core Schema, Storage Lifecycle & Governance Tables
-- Supports Pre/Post Trade Compliance, Version-Pinned Tenant Extensions,
-- Multi-Tier LSN Watermarking, Tamper-Evident Merkle Archive Manifests,
-- Monthly Partitioned Evaluation Logs, Force RLS, Append-Only & Anti-Truncate Triggers,
-- and Quarantine Ledgers.

CREATE SCHEMA IF NOT EXISTS compliance;

-- 1. Compliance Rule Definition Table
CREATE TABLE IF NOT EXISTS compliance.compliance_rule (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID NOT NULL,
    core_rule_id           UUID REFERENCES compliance.compliance_rule(id) ON DELETE SET NULL,
    inherit_mode           TEXT NOT NULL CHECK (inherit_mode IN ('inherit', 'extend', 'custom')),
    pinned_core_version    INT NOT NULL DEFAULT 1,
    drift_status           TEXT NOT NULL DEFAULT 'CURRENT' CHECK (drift_status IN ('CURRENT', 'CORE_VERSION_UPDATED', 'DRIFT_DETECTED', 'RECONCILED')),
    rule_code              TEXT NOT NULL,
    name                   TEXT NOT NULL,
    description            TEXT,
    rule_phase             TEXT NOT NULL CHECK (rule_phase IN ('PRE_TRADE', 'POST_TRADE', 'BOTH')),
    severity               TEXT NOT NULL CHECK (severity IN ('HARD_BLOCK', 'SOFT_WARNING', 'APPROVAL_REQUIRED')),
    ast_condition          JSONB NOT NULL DEFAULT '{}'::jsonb,
    parameter_thresholds   JSONB NOT NULL DEFAULT '{}'::jsonb,
    compiled_wasm          BYTEA,
    compiled_bytecode      BYTEA,
    priority               INT NOT NULL DEFAULT 100,
    is_active              BOOLEAN NOT NULL DEFAULT true,
    valid_to               TIMESTAMPTZ,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_compliance_rule_tenant_code 
    ON compliance.compliance_rule (tenant_id, rule_code) 
    WHERE valid_to IS NULL;

CREATE INDEX IF NOT EXISTS idx_compliance_rule_drift 
    ON compliance.compliance_rule (tenant_id, drift_status) 
    WHERE valid_to IS NULL;

CREATE INDEX IF NOT EXISTS idx_compliance_rule_core_ref 
    ON compliance.compliance_rule (core_rule_id);

ALTER TABLE compliance.compliance_rule ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_rule FORCE ROW LEVEL SECURITY;

-- Drop-then-create for idempotency: matches the pattern used by migrations 006/007/009.
DROP POLICY IF EXISTS tenant_isolation_compliance_rule ON compliance.compliance_rule;
CREATE POLICY tenant_isolation_compliance_rule ON compliance.compliance_rule
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
    );

-- 2. Compliance Evaluation Event Table (Monthly Partitioned + Global Lineage ID)
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event (
    id                UUID NOT NULL DEFAULT gen_random_uuid(),
    lineage_id        UUID NOT NULL,
    tenant_id         UUID NOT NULL,
    order_id          UUID,
    rule_id           UUID NOT NULL REFERENCES compliance.compliance_rule(id) ON DELETE CASCADE,
    rule_version      INT NOT NULL DEFAULT 1,
    passed            BOOLEAN NOT NULL,
    action_taken      TEXT NOT NULL CHECK (action_taken IN ('APPROVED', 'BLOCKED', 'WARNED', 'APPROVAL_PENDING', 'BYPASSED')),
    latency_micros    BIGINT NOT NULL,
    evaluation_hash   TEXT NOT NULL,
    input_params      JSONB NOT NULL DEFAULT '{}'::jsonb,
    metric_snapshots  JSONB NOT NULL DEFAULT '{}'::jsonb,
    evaluated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_compliance_evaluation_event PRIMARY KEY (id, evaluated_at),
    CONSTRAINT unq_compliance_eval_lineage_time UNIQUE (lineage_id, evaluated_at)
) PARTITION BY RANGE (evaluated_at);

-- Pre-seed monthly partitions (2026 Q4 + All 12 Months of 2027)
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2026m10 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2026-10-01 00:00:00+00') TO ('2026-11-01 00:00:00+00');
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2026m11 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2026-11-01 00:00:00+00') TO ('2026-12-01 00:00:00+00');
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2026m12 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2026-12-01 00:00:00+00') TO ('2027-01-01 00:00:00+00');

CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2027m01 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2027-01-01 00:00:00+00') TO ('2027-02-01 00:00:00+00');
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2027m02 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2027-02-01 00:00:00+00') TO ('2027-03-01 00:00:00+00');
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2027m03 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2027-03-01 00:00:00+00') TO ('2027-04-01 00:00:00+00');
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2027m04 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2027-04-01 00:00:00+00') TO ('2027-05-01 00:00:00+00');
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2027m05 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2027-05-01 00:00:00+00') TO ('2027-06-01 00:00:00+00');
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2027m06 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2027-06-01 00:00:00+00') TO ('2027-07-01 00:00:00+00');
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2027m07 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2027-07-01 00:00:00+00') TO ('2027-08-01 00:00:00+00');
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2027m08 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2027-08-01 00:00:00+00') TO ('2027-09-01 00:00:00+00');
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2027m09 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2027-09-01 00:00:00+00') TO ('2027-10-01 00:00:00+00');
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2027m10 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2027-10-01 00:00:00+00') TO ('2027-11-01 00:00:00+00');
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2027m11 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2027-11-01 00:00:00+00') TO ('2027-12-01 00:00:00+00');
CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_y2027m12 PARTITION OF compliance.compliance_evaluation_event
    FOR VALUES FROM ('2027-12-01 00:00:00+00') TO ('2028-01-01 00:00:00+00');

CREATE TABLE IF NOT EXISTS compliance.compliance_evaluation_event_default PARTITION OF compliance.compliance_evaluation_event
    DEFAULT;

CREATE INDEX IF NOT EXISTS idx_compliance_eval_tenant_time 
    ON compliance.compliance_evaluation_event (tenant_id, evaluated_at DESC);

CREATE INDEX IF NOT EXISTS idx_compliance_eval_order 
    ON compliance.compliance_evaluation_event (order_id);

CREATE INDEX IF NOT EXISTS idx_compliance_eval_rule 
    ON compliance.compliance_evaluation_event (rule_id);

CREATE INDEX IF NOT EXISTS idx_compliance_eval_lineage 
    ON compliance.compliance_evaluation_event (lineage_id);

ALTER TABLE compliance.compliance_evaluation_event ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_evaluation_event FORCE ROW LEVEL SECURITY;

-- Drop-then-create for idempotency: matches the pattern used by migrations 006/007/009.
DROP POLICY IF EXISTS tenant_isolation_compliance_evaluation ON compliance.compliance_evaluation_event;
CREATE POLICY tenant_isolation_compliance_evaluation ON compliance.compliance_evaluation_event
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
    );

-- Append-Only & Anti-Truncate Triggers
CREATE OR REPLACE FUNCTION compliance.prevent_evaluation_mutation()
RETURNS TRIGGER AS $$
BEGIN
    RAISE EXCEPTION 'Audit Violation: compliance.compliance_evaluation_event is strictly append-only. Mutation (UPDATE/DELETE/TRUNCATE) is disallowed.'
        USING ERRCODE = '55000';
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trg_prevent_eval_mutation
    BEFORE UPDATE OR DELETE ON compliance.compliance_evaluation_event
    FOR EACH ROW EXECUTE FUNCTION compliance.prevent_evaluation_mutation();

CREATE OR REPLACE TRIGGER trg_prevent_eval_truncate
    BEFORE TRUNCATE ON compliance.compliance_evaluation_event
    FOR EACH STATEMENT EXECUTE FUNCTION compliance.prevent_evaluation_mutation();

REVOKE TRUNCATE ON ALL TABLES IN SCHEMA compliance FROM public;

-- 3. Compliance Archive Manifest (Cold Tier Sealed WORM Metadata)
CREATE TABLE IF NOT EXISTS compliance.compliance_archive_manifest (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL,
    lwm_start_lsn     PG_LSN NOT NULL,
    lwm_end_lsn       PG_LSN NOT NULL,
    merkle_root_hash  TEXT NOT NULL,
    s3_bucket         TEXT NOT NULL,
    s3_key            TEXT NOT NULL,
    etag              TEXT NOT NULL,
    record_count      BIGINT NOT NULL,
    file_size_bytes   BIGINT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'SEALED' CHECK (status IN ('PENDING_SEAL', 'SEALED', 'QUARANTINED', 'VERIFIED')),
    sealed_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_compliance_archive_manifest_lookup 
    ON compliance.compliance_archive_manifest (tenant_id, lwm_start_lsn, lwm_end_lsn);

CREATE UNIQUE INDEX IF NOT EXISTS unq_compliance_archive_manifest_s3_key 
    ON compliance.compliance_archive_manifest (s3_bucket, s3_key);

ALTER TABLE compliance.compliance_archive_manifest ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_archive_manifest FORCE ROW LEVEL SECURITY;

-- Drop-then-create for idempotency: matches the pattern used by migrations 006/007/009.
DROP POLICY IF EXISTS tenant_isolation_compliance_manifest ON compliance.compliance_archive_manifest;
CREATE POLICY tenant_isolation_compliance_manifest ON compliance.compliance_archive_manifest
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
    );

-- 4. Compliance Watermark Checkpoint Table
CREATE TABLE IF NOT EXISTS compliance.compliance_watermark_checkpoint (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID NOT NULL,
    tier          TEXT NOT NULL CHECK (tier IN ('HOT', 'WARM', 'COLD')),
    current_lsn   PG_LSN NOT NULL,
    certified_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unq_compliance_watermark_tenant_tier UNIQUE (tenant_id, tier)
);

ALTER TABLE compliance.compliance_watermark_checkpoint ENABLE ROW LEVEL SECURITY;
ALTER TABLE compliance.compliance_watermark_checkpoint FORCE ROW LEVEL SECURITY;

-- Drop-then-create for idempotency: matches the pattern used by migrations 006/007/009.
DROP POLICY IF EXISTS tenant_isolation_compliance_watermark ON compliance.compliance_watermark_checkpoint;
CREATE POLICY tenant_isolation_compliance_watermark ON compliance.compliance_watermark_checkpoint
    FOR ALL
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR current_setting('app.is_admin', true) = 'true'
    );

-- 5. Compliance Orphan / Quarantine Ledger
CREATE TABLE IF NOT EXISTS compliance.compliance_orphan_object (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID,
    s3_bucket      TEXT NOT NULL,
    s3_key         TEXT NOT NULL,
    etag           TEXT NOT NULL,
    discovered_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    status         TEXT NOT NULL DEFAULT 'QUARANTINED' CHECK (status IN ('QUARANTINED', 'INVESTIGATING', 'RESOLVED', 'DISCARDED')),
    triage_notes   TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT unq_compliance_orphan_key UNIQUE (s3_bucket, s3_key)
);

-- 6. Dynamic Partition Maintenance Function
CREATE OR REPLACE FUNCTION compliance.ensure_evaluation_partitions(p_months_ahead INT DEFAULT 3)
RETURNS VOID AS $$
DECLARE
    v_target_date DATE := date_trunc('month', now())::DATE;
    v_end_date DATE;
    v_table_name TEXT;
    v_start_str TEXT;
    v_end_str TEXT;
BEGIN
    FOR i IN 0..p_months_ahead LOOP
        v_end_date := (v_target_date + INTERVAL '1 month')::DATE;
        v_table_name := 'compliance_evaluation_event_y' || to_char(v_target_date, 'YYYY') || 'm' || to_char(v_target_date, 'MM');
        v_start_str := to_char(v_target_date, 'YYYY-MM-DD 00:00:00+00');
        v_end_str := to_char(v_end_date, 'YYYY-MM-DD 00:00:00+00');

        BEGIN
            EXECUTE format(
                'CREATE TABLE IF NOT EXISTS compliance.%I PARTITION OF compliance.compliance_evaluation_event FOR VALUES FROM (%L) TO (%L);',
                v_table_name, v_start_str, v_end_str
            );
        EXCEPTION WHEN duplicate_table THEN
            -- already exists
        END;

        v_target_date := v_end_date;
    END LOOP;
END;
$$ LANGUAGE plpgsql;
