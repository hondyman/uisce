-- Data plane (crims / tenant datasource): Account master + staging landing + warnings.
-- Apply to the tenant data-plane database (crims), NOT alpha.

CREATE SCHEMA IF NOT EXISTS staging;
CREATE SCHEMA IF NOT EXISTS mdm;

CREATE TABLE IF NOT EXISTS mdm.account_master (
    id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         uuid NOT NULL,
    account_cd        varchar(50) NOT NULL,
    account_name      varchar(500) NOT NULL,
    account_type_cd   varchar(50) NOT NULL DEFAULT 'RETAIL',
    holder_party_id   uuid,
    client_group_id   uuid,
    orm_account_id    uuid,
    status_cd         varchar(30) NOT NULL DEFAULT 'ACTIVE',
    opened_date       date,
    closed_date       date,
    base_currency     varchar(3),
    domicile          varchar(2),
    custodian_id      uuid,
    manager_id        uuid,
    custom_attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
    confidence_score  int NOT NULL DEFAULT 0,
    source_systems    jsonb NOT NULL DEFAULT '{}'::jsonb,
    valid_from        timestamptz NOT NULL DEFAULT now(),
    valid_to          timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_account_master_cd_from UNIQUE (tenant_id, account_cd, valid_from),
    CONSTRAINT chk_account_master_type CHECK (account_type_cd IN (
        'RETAIL','INSTITUTIONAL','CUSTODY','PENSION','RETIREMENT','TRUST',
        'SMA','UMA','FAMILY_OFFICE','PRIVATE_WEALTH','INSURANCE','ENDOWMENT'
    ))
);

CREATE INDEX IF NOT EXISTS idx_account_master_current
    ON mdm.account_master (tenant_id, account_cd)
    WHERE valid_to IS NULL;
CREATE INDEX IF NOT EXISTS idx_account_master_type
    ON mdm.account_master (tenant_id, account_type_cd)
    WHERE valid_to IS NULL;
CREATE INDEX IF NOT EXISTS idx_account_master_custom_gin
    ON mdm.account_master USING gin (custom_attributes jsonb_path_ops);

ALTER TABLE mdm.account_master ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm.account_master FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS account_master_tenant ON mdm.account_master;
CREATE POLICY account_master_tenant ON mdm.account_master
    USING (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''))
    WITH CHECK (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''));

CREATE TABLE IF NOT EXISTS staging.account_data (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        uuid NOT NULL,
    source_system    varchar(50) NOT NULL,
    source_row_id    varchar(200),
    loaded_at        timestamptz NOT NULL DEFAULT now(),
    loaded_at_master timestamptz,
    load_run_id      uuid,
    account_cd       varchar(50),
    account_name     varchar(500),
    account_type_cd  varchar(50),
    status_cd        varchar(30),
    base_currency    varchar(3),
    domicile         varchar(2),
    custodian_id     uuid,
    manager_id       uuid,
    opened_date      date,
    closed_date      date,
    holder_party_id  uuid,
    client_group_id  uuid,
    custom_attributes jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX IF NOT EXISTS idx_staging_account_pending
    ON staging.account_data (tenant_id, loaded_at)
    WHERE loaded_at_master IS NULL;
CREATE INDEX IF NOT EXISTS idx_staging_account_run
    ON staging.account_data (load_run_id)
    WHERE load_run_id IS NOT NULL;

ALTER TABLE staging.account_data ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.account_data FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS staging_account_data_tenant ON staging.account_data;
CREATE POLICY staging_account_data_tenant ON staging.account_data
    USING (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''))
    WITH CHECK (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''));

CREATE TABLE IF NOT EXISTS staging.account_warnings (
    id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     uuid NOT NULL,
    staging_row_id uuid NOT NULL REFERENCES staging.account_data(id) ON DELETE CASCADE,
    load_run_id   uuid NOT NULL,
    field_cd      varchar(100) NOT NULL,
    value         text,
    referenced_bo varchar(50),
    reason        varchar(50) NOT NULL,
    severity      varchar(20) NOT NULL DEFAULT 'WARNING',
    created_at    timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_staging_account_warnings_run
    ON staging.account_warnings (load_run_id, created_at DESC);

ALTER TABLE staging.account_warnings ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.account_warnings FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS staging_account_warnings_tenant ON staging.account_warnings;
CREATE POLICY staging_account_warnings_tenant ON staging.account_warnings
    USING (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''))
    WITH CHECK (tenant_id::text = COALESCE(current_setting('app.current_tenant', true), current_setting('uisce.current_tenant', true), ''));

-- Safe casts for custom attribute preview/load
CREATE OR REPLACE FUNCTION mdm.safe_int(text)
RETURNS int LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT CASE WHEN $1 IS NULL THEN NULL WHEN $1 ~ '^-?\d+$' THEN $1::int ELSE NULL END
$$;
CREATE OR REPLACE FUNCTION mdm.safe_numeric(text)
RETURNS numeric LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT CASE WHEN $1 IS NULL THEN NULL
        WHEN $1 ~ '^-?\d+(\.\d+)?([eE][-+]?\d+)?$' THEN $1::numeric ELSE NULL END
$$;
CREATE OR REPLACE FUNCTION mdm.safe_date(text)
RETURNS date LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT CASE WHEN $1 IS NULL THEN NULL WHEN $1 ~ '^\d{4}-\d{2}-\d{2}$' THEN $1::date ELSE NULL END
$$;
CREATE OR REPLACE FUNCTION mdm.safe_bool(text)
RETURNS bool LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
    SELECT CASE WHEN $1 IS NULL THEN NULL
        WHEN lower($1) IN ('true','t','1','yes','y') THEN true
        WHEN lower($1) IN ('false','f','0','no','n') THEN false
        ELSE NULL END
$$;
