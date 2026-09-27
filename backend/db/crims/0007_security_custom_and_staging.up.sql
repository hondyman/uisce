-- Align Security master with Account dual-registry pattern:
-- custom_attributes on mdm.security_master + staging.security_data for pipeline loads.

ALTER TABLE mdm.security_master
    ADD COLUMN IF NOT EXISTS custom_attributes jsonb NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_security_master_custom_gin
    ON mdm.security_master USING gin (custom_attributes jsonb_path_ops);

CREATE SCHEMA IF NOT EXISTS staging;

CREATE TABLE IF NOT EXISTS staging.security_data (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         uuid NOT NULL,
    source_system     varchar(50) NOT NULL DEFAULT 'PIPELINE',
    source_row_id     varchar(200),
    loaded_at         timestamptz NOT NULL DEFAULT now(),
    loaded_at_master  timestamptz,
    load_run_id       uuid,
    _load_run_id      uuid,
    _source_row_num   integer,
    -- natural keys / typed subset mirroring security_master
    security_id       varchar(100),
    primary_identifier varchar(50),
    isin              varchar(12),
    cusip             varchar(9),
    sedol             varchar(7),
    figi              varchar(12),
    ticker            varchar(20),
    security_name     varchar(255),
    asset_class       varchar(50),
    currency          varchar(3),
    status            varchar(20),
    custom_attributes jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_staging_security_pending
    ON staging.security_data (tenant_id, loaded_at)
    WHERE loaded_at_master IS NULL;
CREATE INDEX IF NOT EXISTS idx_staging_security_load_run
    ON staging.security_data (_load_run_id)
    WHERE _load_run_id IS NOT NULL;

ALTER TABLE staging.security_data ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.security_data FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS staging_security_data_tenant ON staging.security_data;
CREATE POLICY staging_security_data_tenant ON staging.security_data
    USING (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''))
    WITH CHECK (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''));

CREATE TABLE IF NOT EXISTS staging.security_warnings (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL,
    staging_row_id uuid NOT NULL REFERENCES staging.security_data(id) ON DELETE CASCADE,
    load_run_id    uuid NOT NULL,
    field_cd       varchar(100) NOT NULL,
    value          text,
    referenced_bo  varchar(50),
    reason         varchar(50) NOT NULL,
    severity       varchar(20) NOT NULL DEFAULT 'WARNING',
    created_at     timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_staging_security_warnings_run
    ON staging.security_warnings (load_run_id, created_at DESC);

ALTER TABLE staging.security_warnings ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.security_warnings FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS staging_security_warnings_tenant ON staging.security_warnings;
CREATE POLICY staging_security_warnings_tenant ON staging.security_warnings
    USING (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''))
    WITH CHECK (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''));
