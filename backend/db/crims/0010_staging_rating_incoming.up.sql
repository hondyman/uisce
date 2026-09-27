-- 0010_staging_rating_incoming.up.sql
-- Canonical landing zone for rating facts across agencies (S&P, Moody's,
-- Fitch, Internal). One staging table, source_cd discriminates vendor.
-- Run against crims.
--
-- Design notes (mirroring staging.product_incoming, 008_staging_product.sql):
--   * No CHECK on rated_party_type — loader is the gate. Staging accepts
--     anything that parses; rejection produces a visible loader error and
--     a row in staging._mapping_error, never a silent drop.
--   * No rating_rank column. Single source of truth is mdm.rating_scale.
--     numeric_equivalent. The loader joins scale at write time and stamps
--     rating_rank on mdm.rating. Staging CSV does not carry rank.
--   * rated_party_key is the business key from the source feed. Loader
--     resolves to mdm.party.id (NOT NULL in mdm.rating); unresolved rows
--     are rejected, not NULL'd.
--   * RLS with the standard shared-reference fallback
--     (app.shared_reference_tenant).
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0010_staging_rating_incoming.up.sql

\set ON_ERROR_STOP on
BEGIN;

CREATE SCHEMA IF NOT EXISTS staging;

CREATE TABLE IF NOT EXISTS staging.rating_incoming (
    id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    source_system_id      uuid NOT NULL,
    source_system_cd      varchar(30) NOT NULL,
    source_row_id         text NOT NULL,
    load_run_id           uuid NOT NULL REFERENCES staging._load_run(id) ON DELETE CASCADE,
    tenant_id             uuid NOT NULL,

    -- Rated party (business key + nullable resolved UUID for the loader)
    rated_party_type      varchar(30) NOT NULL,
    rated_party_key       varchar(100),
    rated_party_id        uuid,

    -- Rating
    agency_cd             varchar(30) NOT NULL,
    rating_type_cd        varchar(50) NOT NULL,
    rating_scale_cd       varchar(50),
    rating_value          varchar(20) NOT NULL,

    -- Context
    outlook_cd            varchar(20),
    watch_cd              varchar(30),
    credit_watch_direction varchar(20),
    is_credit_event       bool NOT NULL DEFAULT false,

    -- Dates
    rating_date           date NOT NULL,
    effective_from        date NOT NULL DEFAULT CURRENT_DATE,
    effective_to          date,

    source_document       varchar(500),

    -- Full mapped payload for replay / forensic
    canonical_payload     jsonb NOT NULL DEFAULT '{}'::jsonb,
    mapping_version       int  NOT NULL DEFAULT 1,
    mapped_at             timestamptz NOT NULL DEFAULT now(),

    -- Validation state. loader rejects invalid rows by setting
    -- is_valid=false and writing to staging._mapping_error; the row
    -- stays in the table for audit.
    is_valid              bool  NOT NULL DEFAULT true,
    validation_errors     jsonb NOT NULL DEFAULT '[]'::jsonb,

    -- Master-load bookkeeping. loaded_at_master is set by the loader on
    -- successful insert into mdm.rating.
    loaded_at            timestamptz NOT NULL DEFAULT now(),
    loaded_at_master      timestamptz,

    CONSTRAINT chk_rating_incoming_valid CHECK (
        ((is_valid = true) AND (jsonb_array_length(validation_errors) = 0))
        OR (is_valid = false)
    )
);

CREATE INDEX IF NOT EXISTS idx_rating_inc_party
    ON staging.rating_incoming (rated_party_type, rated_party_key)
    WHERE rated_party_key IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rating_inc_party_id
    ON staging.rating_incoming (rated_party_type, rated_party_id)
    WHERE rated_party_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_rating_inc_agency
    ON staging.rating_incoming (agency_cd, rating_date DESC);
CREATE INDEX IF NOT EXISTS idx_rating_inc_load
    ON staging.rating_incoming (load_run_id);
CREATE INDEX IF NOT EXISTS idx_rating_inc_tenant
    ON staging.rating_incoming (tenant_id);
CREATE INDEX IF NOT EXISTS idx_rating_inc_pending
    ON staging.rating_incoming (tenant_id, loaded_at)
    WHERE loaded_at_master IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_rating_inc_source_row
    ON staging.rating_incoming (tenant_id, source_system_id, source_row_id);

ALTER TABLE staging.rating_incoming ENABLE ROW LEVEL SECURITY;
ALTER TABLE staging.rating_incoming FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rating_inc_tenant_read ON staging.rating_incoming;
CREATE POLICY rating_inc_tenant_read ON staging.rating_incoming
    AS PERMISSIVE FOR SELECT
    USING (
        (tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
                (current_setting('app.shared_reference_tenant'::text, true))::uuid,
                '00000000-0000-0000-0000-000000000001'::uuid))
    );

DROP POLICY IF EXISTS rating_inc_tenant_write ON staging.rating_incoming;
CREATE POLICY rating_inc_tenant_write ON staging.rating_incoming
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

DO $verify$
DECLARE
    t text;
    tables text[] := ARRAY['rating_incoming'];
BEGIN
    FOREACH t IN ARRAY tables LOOP
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.tables
            WHERE table_schema='staging' AND table_name=t
        ) THEN
            RAISE EXCEPTION 'staging.% was not created', t;
        END IF;
    END LOOP;
    RAISE NOTICE '0010_staging_rating_incoming: 1 table created/verified';
END
$verify$;

COMMIT;
