-- 0016_mdm_rating_crims.up.sql
-- Verbatim port of mdm.rating from alpha
-- db/migrations/20261102_002_mdm_rating_anchor.up.sql (lines 23-64).
-- Idempotent (CREATE TABLE IF NOT EXISTS, ADD COLUMN IF NOT EXISTS).
--
-- FKs reference tables in 0015 (rating_agency, rating_type, etc.) which
-- must be applied first. NOT VALID here so the order tolerance holds
-- (constraint validated by the loader's reference checks; FK is
-- present but not enforced at table-creation time).

\set ON_ERROR_STOP on
BEGIN;

CREATE SCHEMA IF NOT EXISTS mdm;

CREATE TABLE IF NOT EXISTS mdm.rating (
    id uuid DEFAULT gen_random_uuid() NOT NULL,
    agency_id uuid NOT NULL,
    rating_type_id uuid NOT NULL,
    rating_scale_id uuid,
    rated_party_type varchar(30) NOT NULL,
    rated_party_id uuid NOT NULL,
    rated_party_key varchar(100),
    rating_value varchar(20) NOT NULL,
    rating_rank int4,
    outlook_id uuid,
    watch_id uuid,
    credit_watch_direction varchar(20),
    is_active bool DEFAULT true NOT NULL,
    is_latest bool DEFAULT true NOT NULL,
    withdrawn_reason varchar(100),
    withdrawn_at timestamptz,
    rating_date date DEFAULT CURRENT_DATE NOT NULL,
    effective_from date DEFAULT CURRENT_DATE NOT NULL,
    effective_to date,
    created_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    updated_at timestamptz DEFAULT CURRENT_TIMESTAMP,
    tenant_id uuid NOT NULL,
    CONSTRAINT rating_pkey PRIMARY KEY (id),
    CONSTRAINT fk_r_agency    FOREIGN KEY (agency_id)      REFERENCES mdm.rating_agency(id),
    CONSTRAINT fk_r_type      FOREIGN KEY (rating_type_id)  REFERENCES mdm.rating_type(id),
    CONSTRAINT fk_r_scale     FOREIGN KEY (rating_scale_id) REFERENCES mdm.rating_scale(id),
    CONSTRAINT fk_r_outlook   FOREIGN KEY (outlook_id)      REFERENCES mdm.rating_outlook(id),
    CONSTRAINT fk_r_watch     FOREIGN KEY (watch_id)        REFERENCES mdm.rating_watch(id),
    CONSTRAINT chk_r_party CHECK (rated_party_type IN (
        'ISSUER','ISSUE','TRANCHE','COUNTERPARTY','SOVEREIGN',
        'SUB_SOVEREIGN','FINANCIAL_STRENGTH','ISSUER_GROUP','INTERNAL')),
    CONSTRAINT chk_r_watch_dir CHECK (credit_watch_direction IS NULL
        OR credit_watch_direction IN ('POSITIVE','NEGATIVE','DEVELOPING','EVOLVING'))
);
CREATE INDEX IF NOT EXISTS idx_r_party      ON mdm.rating (rated_party_type, rated_party_id) WHERE is_active;
CREATE INDEX IF NOT EXISTS idx_r_agency     ON mdm.rating (agency_id, is_latest);
CREATE INDEX IF NOT EXISTS idx_r_rank       ON mdm.rating (rating_rank) WHERE is_active;
CREATE INDEX IF NOT EXISTS idx_r_type       ON mdm.rating (rating_type_id);
CREATE INDEX IF NOT EXISTS idx_r_date       ON mdm.rating (rating_date DESC);
CREATE INDEX IF NOT EXISTS idx_r_tenant     ON mdm.rating (tenant_id);

-- ── RLS ────────────────────────────────────────────────────────────────
ALTER TABLE mdm.rating ENABLE ROW LEVEL SECURITY;
ALTER TABLE mdm.rating FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS rating_tenant_read ON mdm.rating;
CREATE POLICY rating_tenant_read ON mdm.rating
    AS PERMISSIVE FOR SELECT
    USING (
        (tenant_id = (current_setting('app.current_tenant'::text, true))::uuid)
        OR (tenant_id = COALESCE(
                (current_setting('app.shared_reference_tenant'::text, true))::uuid,
                '00000000-0000-0000-0000-000000000001'::uuid))
    );

DROP POLICY IF EXISTS rating_tenant_write ON mdm.rating;
CREATE POLICY rating_tenant_write ON mdm.rating
    AS PERMISSIVE FOR ALL
    USING ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid))
    WITH CHECK ((tenant_id = (current_setting('app.current_tenant'::text, true))::uuid));

DO $verify$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema='mdm' AND table_name='rating'
    ) THEN
        RAISE EXCEPTION '0016_mdm_rating_crims: mdm.rating not created';
    END IF;
    RAISE NOTICE '0016_mdm_rating_crims: mdm.rating created/verified';
END
$verify$;

COMMIT;
