-- 0014_mdm_party_crims.up.sql
-- Verbatim port of mdm.party from alpha db/migrations/20261028_001_mdm_party_expand.up.sql.
-- Idempotent (CREATE TABLE IF NOT EXISTS, ADD COLUMN IF NOT EXISTS).
-- Adds RLS policies (alpha's mdm.party has no RLS; crims requires FORCE).
--
-- Apply against crims. Run AFTER 0010..0013.
-- Run AFTER 0015 (no dependency, but ordering keeps party with the rest
-- of the mdm family).

\set ON_ERROR_STOP on
BEGIN;

CREATE SCHEMA IF NOT EXISTS mdm;

CREATE TABLE IF NOT EXISTS mdm.party (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    tenant_id uuid NOT NULL,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    valid_from timestamptz DEFAULT CURRENT_TIMESTAMP NOT NULL,
    valid_to timestamptz,
    custom_attributes jsonb DEFAULT '{}'::jsonb NOT NULL,
    CONSTRAINT party_pkey PRIMARY KEY (id)
);

ALTER TABLE mdm.party
    ADD COLUMN IF NOT EXISTS name                         varchar(250),
    ADD COLUMN IF NOT EXISTS party_sub_type              varchar(50),
    ADD COLUMN IF NOT EXISTS status                      varchar(20) DEFAULT 'ACTIVE',
    ADD COLUMN IF NOT EXISTS lifecycle_stage             varchar(20) DEFAULT 'ACTIVE',
    ADD COLUMN IF NOT EXISTS is_individual               bool DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_legal_entity             bool DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_financial_institution    bool DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_regulated                bool DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_pep                      bool DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_sanctioned               bool DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_restricted               bool DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_us_person                bool DEFAULT false,
    ADD COLUMN IF NOT EXISTS primary_segment_id          uuid,
    ADD COLUMN IF NOT EXISTS parent_party_id             uuid,
    ADD COLUMN IF NOT EXISTS ultimate_parent_party_id    uuid,
    ADD COLUMN IF NOT EXISTS client_group_id             uuid,
    ADD COLUMN IF NOT EXISTS primary_tax_country_cd      varchar(2),
    ADD COLUMN IF NOT EXISTS primary_residence_country_cd varchar(2),
    ADD COLUMN IF NOT EXISTS primary_nationality_cd      varchar(2),
    ADD COLUMN IF NOT EXISTS country_of_risk_cd          varchar(2),
    ADD COLUMN IF NOT EXISTS lei                         varchar(20),
    ADD COLUMN IF NOT EXISTS giin                        varchar(20),
    ADD COLUMN IF NOT EXISTS registration_number         varchar(100),
    ADD COLUMN IF NOT EXISTS incorporation_date          date,
    ADD COLUMN IF NOT EXISTS incorporation_jurisdiction  varchar(10),
    ADD COLUMN IF NOT EXISTS legal_form                  varchar(50),
    ADD COLUMN IF NOT EXISTS date_of_birth               date,
    ADD COLUMN IF NOT EXISTS is_deceased                 bool DEFAULT false,
    ADD COLUMN IF NOT EXISTS date_of_death               date,
    ADD COLUMN IF NOT EXISTS client_since_date           date,
    ADD COLUMN IF NOT EXISTS relationship_end_date       date,
    ADD COLUMN IF NOT EXISTS relationship_manager_id     uuid,
    ADD COLUMN IF NOT EXISTS servicing_team_id           uuid,
    ADD COLUMN IF NOT EXISTS primary_currency            varchar(3),
    ADD COLUMN IF NOT EXISTS source_system_id            uuid,
    ADD COLUMN IF NOT EXISTS golden_record_id            uuid,
    ADD COLUMN IF NOT EXISTS is_golden_record            bool DEFAULT true,
    ADD COLUMN IF NOT EXISTS merged_into_id              uuid,
    ADD COLUMN IF NOT EXISTS dq_score                    numeric(5,2),
    ADD COLUMN IF NOT EXISTS steward_id                  uuid,
    ADD COLUMN IF NOT EXISTS last_reviewed_at            timestamptz,
    ADD COLUMN IF NOT EXISTS review_frequency_months     int4;

-- Backfill defaults so NOT NULL can be applied where safe
UPDATE mdm.party SET status = 'ACTIVE'          WHERE status IS NULL;
UPDATE mdm.party SET lifecycle_stage = 'ACTIVE' WHERE lifecycle_stage IS NULL;
UPDATE mdm.party SET is_golden_record = true    WHERE is_golden_record IS NULL;

ALTER TABLE mdm.party
    ALTER COLUMN status SET NOT NULL,
    ALTER COLUMN lifecycle_stage SET NOT NULL,
    ALTER COLUMN is_golden_record SET NOT NULL;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_party_status') THEN
        ALTER TABLE mdm.party
            ADD CONSTRAINT chk_party_status
            CHECK (status IN ('ACTIVE','DORMANT','INACTIVE','DISSOLVED','MERGED',
                              'ACQUIRED','IN_LIQUIDATION','IN_RECEIVERSHIP',
                              'IN_ADMINISTRATION','DECEASED','CLOSED'))
            NOT VALID;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_party_lifecycle') THEN
        ALTER TABLE mdm.party
            ADD CONSTRAINT chk_party_lifecycle
            CHECK (lifecycle_stage IN ('PROSPECT','ONBOARDING','ACTIVE','DORMANT','CLOSED','MERGED'))
            NOT VALID;
    END IF;
END $$;

-- Self-referential FKs (deferred — cannot add FK on self-referencing column
-- in the same migration that adds the column, per project convention. The
-- application layer enforces these; add FKs in a later constraint-only pass.)

CREATE INDEX IF NOT EXISTS idx_party_tenant      ON mdm.party (tenant_id);
CREATE INDEX IF NOT EXISTS idx_party_status      ON mdm.party (status);
CREATE INDEX IF NOT EXISTS idx_party_sub_type    ON mdm.party (party_sub_type);
CREATE INDEX IF NOT EXISTS idx_party_client_group ON mdm.party (client_group_id);
CREATE INDEX IF NOT EXISTS idx_party_parent      ON mdm.party (parent_party_id);
CREATE INDEX IF NOT EXISTS idx_party_lei         ON mdm.party (lei) WHERE lei IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_party_dob         ON mdm.party (date_of_birth) WHERE date_of_birth IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_party_pep         ON mdm.party (is_pep) WHERE is_pep = true;
CREATE INDEX IF NOT EXISTS idx_party_sanctioned  ON mdm.party (is_sanctioned) WHERE is_sanctioned = true;
CREATE INDEX IF NOT EXISTS idx_party_individual  ON mdm.party (is_individual) WHERE is_individual = true;

-- ── RLS (added in crims; alpha's mdm.party has no RLS) ─────────────────
ALTER TABLE mdm.party ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm.party FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS party_tenant_read ON mdm.party;
CREATE POLICY party_tenant_read ON mdm.party
    AS PERMISSIVE FOR SELECT
    USING (
        (tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
                (current_setting('app.shared_reference_tenant'::text, true))::uuid,
                '00000000-0000-0000-0000-000000000001'::uuid))
    );

DROP POLICY IF EXISTS party_tenant_write ON mdm.party;
CREATE POLICY party_tenant_write ON mdm.party
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

DO $verify$
DECLARE n int;
BEGIN
    SELECT count(*) INTO n
    FROM information_schema.columns
    WHERE table_schema='mdm' AND table_name='party' AND column_name='lei';
    IF n <> 1 THEN
        RAISE EXCEPTION '0014_mdm_party_crims: mdm.party.lei column missing';
    END IF;
    RAISE NOTICE '0014_mdm_party_crims: mdm.party created/verified';
END
$verify$;

COMMIT;
