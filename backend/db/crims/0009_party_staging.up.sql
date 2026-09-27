-- Party staging for dual-registry ingest → mdm.party (already has custom_attributes).

CREATE SCHEMA IF NOT EXISTS staging;

CREATE TABLE IF NOT EXISTS staging.party_data (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL,
    source_system    varchar(50) NOT NULL DEFAULT 'PIPELINE',
    source_row_id    varchar(200),
    loaded_at        timestamptz NOT NULL DEFAULT now(),
    loaded_at_master timestamptz,
    load_run_id      uuid,
    _load_run_id     uuid,
    _source_row_num  integer,
    party_cd         varchar(30),
    legal_name       varchar(250),
    party_type       varchar(20),
    segment          varchar(20),
    tax_id           varchar(50),
    domicile         varchar(2),
    status           varchar(20),
    lei              varchar(20),
    custom_attributes jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_staging_party_pending
    ON staging.party_data (tenant_id, loaded_at)
    WHERE loaded_at_master IS NULL;

ALTER TABLE staging.party_data ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.party_data FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS staging_party_data_tenant ON staging.party_data;
CREATE POLICY staging_party_data_tenant ON staging.party_data
    USING (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''))
    WITH CHECK (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''));

CREATE TABLE IF NOT EXISTS staging.party_warnings (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      uuid NOT NULL,
    staging_row_id uuid NOT NULL REFERENCES staging.party_data(id) ON DELETE CASCADE,
    load_run_id    uuid NOT NULL,
    field_cd       varchar(100) NOT NULL,
    value          text,
    referenced_bo  varchar(50),
    reason         varchar(50) NOT NULL,
    severity       varchar(20) NOT NULL DEFAULT 'WARNING',
    created_at     timestamptz NOT NULL DEFAULT now()
);

ALTER TABLE staging.party_warnings ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.party_warnings FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS staging_party_warnings_tenant ON staging.party_warnings;
CREATE POLICY staging_party_warnings_tenant ON staging.party_warnings
    USING (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''))
    WITH CHECK (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''));
