-- 20260805000002_fact_customization_telemetry.up.sql
-- Phase 2: Customization Intelligence fact table + Python ETL read target
-- See: intelligence-engine/jobs/customization_clusters.py

BEGIN;

CREATE TABLE IF NOT EXISTS fact_customization_telemetry (
    id              TEXT        NOT NULL,
    cluster_id      TEXT        NOT NULL,
    pattern_hash    TEXT        NOT NULL,
    entity_type     TEXT        NOT NULL,   -- 'bp_roles' | 'bp_dynamic_policies'
    sample_name     TEXT        NOT NULL,
    tenant_count    INTEGER     NOT NULL DEFAULT 0,
    recommended_for_core  BOOLEAN NOT NULL DEFAULT false,
    confidence_score NUMERIC(5,4) NOT NULL DEFAULT 0,
    detected_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fact_customization_telemetry_pkey
        PRIMARY KEY (cluster_id, pattern_hash)
);

COMMENT ON TABLE fact_customization_telemetry IS
    'Clustered customization patterns detected from CREATE events in audit_logs. '
    'Written by intelligence-engine/jobs/customization_clusters.py; '
    'read by the Go product-evolution API (Phase 3).';

CREATE INDEX IF NOT EXISTS idx_fact_telemetry_recommended
    ON fact_customization_telemetry (recommended_for_core)
    WHERE recommended_for_core = true;

CREATE INDEX IF NOT EXISTS idx_fact_telemetry_confidence
    ON fact_customization_telemetry (confidence_score DESC);

COMMIT;
