-- 0011_staging_load_run_crims.up.sql
-- Verbatim port of staging._load_run + staging._mapping_error from
-- db/manual_fixes/008_staging_product.sql (lines 20-110).
-- 0010 references _load_run via FK; without this file, fresh-crims
-- apply of 0010 fails with "relation staging._load_run does not exist".
--
-- Apply against crims. Run AFTER 0009, BEFORE 0010.

\set ON_ERROR_STOP on
BEGIN;

CREATE SCHEMA IF NOT EXISTS staging;

CREATE TABLE IF NOT EXISTS staging._load_run (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_system_cd  varchar(30) NOT NULL,
    domain            varchar(30) NOT NULL,
    run_ref           varchar(100) NOT NULL,
    file_name         text,
    file_hash         varchar(64),
    expected_rows     int,
    received_rows     int,
    accepted_rows     int,
    rejected_rows     int,
    started_at        timestamptz NOT NULL DEFAULT now(),
    completed_at      timestamptz,
    status            varchar(20) NOT NULL DEFAULT 'RUNNING',
    error_summary     text,
    tenant_id         uuid NOT NULL,
    CONSTRAINT uq_lr UNIQUE (tenant_id, source_system_cd, domain, run_ref),
    CONSTRAINT chk_lr_status CHECK (status IN (
        'RUNNING','COMPLETED','FAILED','PARTIAL','CANCELLED'))
);

CREATE INDEX IF NOT EXISTS idx_lr_status
    ON staging._load_run (status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_lr_source
    ON staging._load_run (source_system_cd, domain, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_lr_tenant
    ON staging._load_run (tenant_id);

ALTER TABLE staging._load_run ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging._load_run FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS _lr_tenant_read ON staging._load_run;
CREATE POLICY _lr_tenant_read ON staging._load_run
    AS PERMISSIVE FOR SELECT
    USING (
        (tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
                (current_setting('app.shared_reference_tenant'::text, true))::uuid,
                '00000000-0000-0000-0000-000000000001'::uuid))
    );

DROP POLICY IF EXISTS _lr_tenant_write ON staging._load_run;
CREATE POLICY _lr_tenant_write ON staging._load_run
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

CREATE TABLE IF NOT EXISTS staging._mapping_error (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    load_run_id       uuid NOT NULL REFERENCES staging._load_run(id) ON DELETE CASCADE,
    source_row_id     text,
    error_type        varchar(30) NOT NULL,
    field_name        text,
    field_value       text,
    error_message     text,
    detected_at       timestamptz NOT NULL DEFAULT now(),
    tenant_id         uuid NOT NULL,
    CONSTRAINT chk_me_error_type CHECK (error_type IN (
        'UNKNOWN_FIELD','TYPE_MISMATCH','MISSING_REQUIRED',
        'ENUM_MISMATCH','DUPLICATE_ROW','TRANSFORM_ERROR'))
);

CREATE INDEX IF NOT EXISTS idx_me_run
    ON staging._mapping_error (load_run_id, error_type);
CREATE INDEX IF NOT EXISTS idx_me_tenant
    ON staging._mapping_error (tenant_id);

ALTER TABLE staging._mapping_error ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging._mapping_error FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS _me_tenant_read ON staging._mapping_error;
CREATE POLICY _me_tenant_read ON staging._mapping_error
    AS PERMISSIVE FOR SELECT
    USING (
        (tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
                (current_setting('app.shared_reference_tenant'::text, true))::uuid,
                '00000000-0000-0000-0000-000000000001'::uuid))
    );

DROP POLICY IF EXISTS _me_tenant_write ON staging._mapping_error;
CREATE POLICY _me_tenant_write ON staging._mapping_error
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

DO $verify$
DECLARE
    t text;
    tables text[] := ARRAY['_load_run','_mapping_error'];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema='staging' AND table_name=t
        ) THEN
            RAISE EXCEPTION '0011_staging_load_run_crims: staging.% was not created', t;
        END IF;
    END LOOP;
    RAISE NOTICE '0011_staging_load_run_crims: 2 tables created/verified';
END
$verify$;

COMMIT;
