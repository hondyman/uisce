-- MDM-side canonical account entity with custom_attributes JSONB.
-- Mirrors portfolio_master / product pattern alongside orm.account.

CREATE TABLE IF NOT EXISTS mdm.account_master (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       uuid NOT NULL,
    account_cd      varchar(50) NOT NULL,
    account_name    varchar(500) NOT NULL,
    account_type_cd varchar(50) NOT NULL,
    holder_party_id uuid,
    client_group_id uuid,
    orm_account_id  uuid,
    status_cd       varchar(30) NOT NULL DEFAULT 'ACTIVE',
    opened_date     date,
    closed_date     date,
    base_currency   varchar(3),
    domicile        varchar(2),
    custom_attributes jsonb NOT NULL DEFAULT '{}'::jsonb,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT uq_account_master_cd UNIQUE (tenant_id, account_cd),
    CONSTRAINT chk_account_master_type CHECK (account_type_cd IN (
        'RETAIL', 'INSTITUTIONAL', 'CUSTODY', 'PENSION', 'RETIREMENT',
        'TRUST', 'SMA', 'UMA', 'FAMILY_OFFICE', 'PRIVATE_WEALTH',
        'INSURANCE', 'ENDOWMENT'
    ))
);

CREATE INDEX IF NOT EXISTS idx_account_master_tenant
    ON mdm.account_master (tenant_id);
CREATE INDEX IF NOT EXISTS idx_account_master_type
    ON mdm.account_master (tenant_id, account_type_cd);
CREATE INDEX IF NOT EXISTS idx_account_master_orm
    ON mdm.account_master (orm_account_id)
    WHERE orm_account_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_account_master_custom_gin
    ON mdm.account_master USING gin (custom_attributes jsonb_path_ops);

-- Optional FKs when referenced tables exist
DO $fk$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='party') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint WHERE conname = 'fk_account_master_holder'
        ) THEN
            ALTER TABLE mdm.account_master
                ADD CONSTRAINT fk_account_master_holder
                FOREIGN KEY (holder_party_id) REFERENCES mdm.party(id);
        END IF;
    END IF;
    IF EXISTS (SELECT 1 FROM information_schema.tables
               WHERE table_schema='mdm' AND table_name='client_group') THEN
        IF NOT EXISTS (
            SELECT 1 FROM pg_constraint WHERE conname = 'fk_account_master_client_group'
        ) THEN
            ALTER TABLE mdm.account_master
                ADD CONSTRAINT fk_account_master_client_group
                FOREIGN KEY (client_group_id) REFERENCES mdm.client_group(id);
        END IF;
    END IF;
END
$fk$;

ALTER TABLE mdm.account_master ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm.account_master FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS account_master_tenant_read ON mdm.account_master;
CREATE POLICY account_master_tenant_read ON mdm.account_master
    AS PERMISSIVE FOR SELECT
    USING (
        tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid
        OR tenant_id = COALESCE(
            NULLIF(current_setting('app.shared_reference_tenant', true), '')::uuid,
            (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1),
            '00000000-0000-0000-0000-000000000001'::uuid
        )
    );

DROP POLICY IF EXISTS account_master_tenant_write ON mdm.account_master;
CREATE POLICY account_master_tenant_write ON mdm.account_master
    AS PERMISSIVE FOR ALL
    USING (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid)
    WITH CHECK (tenant_id = NULLIF(current_setting('app.current_tenant', true), '')::uuid);

-- Ensure safe casts exist for preview
DO $casts$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'mdm') THEN
        EXECUTE $f$
            CREATE OR REPLACE FUNCTION mdm.safe_int(text)
            RETURNS int LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
                SELECT CASE
                    WHEN $1 IS NULL THEN NULL
                    WHEN $1 ~ '^-?\d+$' THEN $1::int
                    ELSE NULL
                END
            $$
        $f$;
        EXECUTE $f$
            CREATE OR REPLACE FUNCTION mdm.safe_numeric(text)
            RETURNS numeric LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
                SELECT CASE
                    WHEN $1 IS NULL THEN NULL
                    WHEN $1 ~ '^-?\d+(\.\d+)?([eE][-+]?\d+)?$' THEN $1::numeric
                    ELSE NULL
                END
            $$
        $f$;
        EXECUTE $f$
            CREATE OR REPLACE FUNCTION mdm.safe_date(text)
            RETURNS date LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
                SELECT CASE
                    WHEN $1 IS NULL THEN NULL
                    WHEN $1 ~ '^\d{4}-\d{2}-\d{2}$' THEN $1::date
                    ELSE NULL
                END
            $$
        $f$;
        EXECUTE $f$
            CREATE OR REPLACE FUNCTION mdm.safe_bool(text)
            RETURNS bool LANGUAGE sql IMMUTABLE PARALLEL SAFE AS $$
                SELECT CASE
                    WHEN $1 IS NULL THEN NULL
                    WHEN lower($1) IN ('true','t','1','yes','y') THEN true
                    WHEN lower($1) IN ('false','f','0','no','n') THEN false
                    ELSE NULL
                END
            $$
        $f$;
    END IF;
END
$casts$;
