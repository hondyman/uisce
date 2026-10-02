-- 002c_crims_rating_alter.sql
-- Rating family reconciliation (per 2026-09-23 reclassification).
-- The rating_* domain moves wholesale to crims.mdm; alpha keeps no rating_*.
--
-- (a) Extend crims.mdm.rating_scale with alpha's additive columns.
--     identity model is (agency_id FK, scale_cd, value) per option (b);
--     alpha's `agency_cd` (denormalized display alias) stays.  Same RLS key
--     (tenant_id), so existing policies remain valid.
-- (b) Create crims.mdm.rating_agency (target of rating_scale.agency_id FK).
--     Doesn't exist yet in crims; alpha's 12 rows + 22-col schema.
-- (c) Create the other 4 zero-row rating_* targets (rating_outlook,
--     rating_watch, rating_type, rating_action_type) so copy_plan.sh's
--     `EXISTS` check doesn't try to CREATE TABLE 5 times (and so the
--     downstream RLS policies match alpha's shape).
-- (d) Optional: pre-stage rating_scale_map and the 9 rating-action satellites
--     (rating, rating_action, rating_bank, rating_default, rating_fund,
--     rating_insurance, rating_internal, rating_proprietary) for explicitness;
--     copy_plan.sh would CREATE them on its own, but pre-staging makes the
--     data plane visible from the start of Step 3 and matches the user's
--     intent that the rating family is a first-class domain.
--
-- Run against CRIMS. Idempotent (ADD COLUMN IF NOT EXISTS, CREATE IF NOT EXISTS).

SET search_path = migration, crims, public;

BEGIN;

-- ============================================================================
-- (a) Extend crims.mdm.rating_scale (existing 12-col stub) with alpha's
--     additive 18-col schema.  agency_cd is retained on alpha as a denormalized
--     display alias; agency_id is the canonical FK.  Both kept here.
-- ============================================================================
ALTER TABLE crims.mdm.rating_scale ADD COLUMN IF NOT EXISTS agency_id uuid NULL;
ALTER TABLE crims.mdm.rating_scale ADD COLUMN IF NOT EXISTS scale_name varchar(150) NULL;
ALTER TABLE crims.mdm.rating_scale ADD COLUMN IF NOT EXISTS agency_scale_type varchar(30) NULL;
ALTER TABLE crims.mdm.rating_scale ADD COLUMN IF NOT EXISTS rank_no integer NULL;
ALTER TABLE crims.mdm.rating_scale ADD COLUMN IF NOT EXISTS rating_category varchar(20) NULL;
ALTER TABLE crims.mdm.rating_scale ADD COLUMN IF NOT EXISTS is_not_rated boolean NULL;
ALTER TABLE crims.mdm.rating_scale ADD COLUMN IF NOT EXISTS is_withdrawn boolean NULL;
ALTER TABLE crims.mdm.rating_scale ADD COLUMN IF NOT EXISTS is_na boolean NULL;
ALTER TABLE crims.mdm.rating_scale ADD COLUMN IF NOT EXISTS display_order integer NULL;
ALTER TABLE crims.mdm.rating_scale ADD COLUMN IF NOT EXISTS effective_from date NULL;
ALTER TABLE crims.mdm.rating_scale ADD COLUMN IF NOT EXISTS effective_to date NULL;
ALTER TABLE crims.mdm.rating_scale ADD COLUMN IF NOT EXISTS is_active boolean NULL;

-- Replace the crims identity with alpha's:
--   old unique: (tenant_id, agency_cd, rating_scale, rating_value)
--   new unique: (tenant_id, scale_cd, value)  matching alpha
-- rating_scale + rating_value are repurposed as scale_cd + value.
-- The crims `rating_scale` becomes the alpha `scale_cd`
-- The crims `rating_value` becomes the alpha `value`
-- We rename rather than drop to preserve (zero-row) data semantics and to
-- avoid the unique constraint switch issue.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1 FROM information_schema.columns
     WHERE table_schema='mdm' AND table_name='rating_scale' AND column_name='rating_scale'
  ) AND NOT EXISTS (
    SELECT 1 FROM information_schema.columns
     WHERE table_schema='mdm' AND table_name='rating_scale' AND column_name='scale_cd'
  ) THEN
    ALTER TABLE crims.mdm.rating_scale RENAME COLUMN rating_scale TO scale_cd;
    ALTER TABLE crims.mdm.rating_scale RENAME COLUMN rating_value TO value;
    ALTER TABLE crims.mdm.rating_scale ALTER COLUMN scale_cd TYPE varchar(50);
    ALTER TABLE crims.mdm.rating_scale ALTER COLUMN value       TYPE varchar(20);
    ALTER TABLE crims.mdm.rating_scale ALTER COLUMN scale_cd SET NOT NULL;
    ALTER TABLE crims.mdm.rating_scale ALTER COLUMN value    SET NOT NULL;
    -- drop old unique, add new one matching alpha
    ALTER TABLE crims.mdm.rating_scale
      DROP CONSTRAINT IF EXISTS rating_scale_tenant_id_agency_cd_rating_scale_rating_value_key;
    ALTER TABLE crims.mdm.rating_scale
      ADD CONSTRAINT rating_scale_tenant_scale_value_uniq
      UNIQUE (tenant_id, scale_cd, value);
  END IF;
END $$;

-- Drop the now-redundant agency_cd?  Per (b): alpha style says agency_id is
-- canonical and agency_cd is a display alias.  We KEEP agency_cd here (backfilled
-- from rating_agency at copy time) because the alpha schema carries it as
-- a denormalized column too.  Don't drop.
-- Drop the obsolete numeric_equivalent / is_investment_grade / is_default /
-- is_high_yield / is_current columns?  No — they're nullable / unset in
-- alpha; keep them as legacy-extension points.  Document in notes.

-- FK to crims.mdm.rating_agency once that table exists (created below)
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint WHERE conname='rating_scale_agency_id_fkey'
  ) AND EXISTS (SELECT 1 FROM pg_tables WHERE schemaname='mdm' AND tablename='rating_agency') THEN
    ALTER TABLE crims.mdm.rating_scale
      ADD CONSTRAINT rating_scale_agency_id_fkey
      FOREIGN KEY (agency_id) REFERENCES crims.mdm.rating_agency(id)
      ON DELETE SET NULL;
  END IF;
END $$;

-- ============================================================================
-- (b) Create crims.mdm.rating_agency (target of (a)'s FK; mirror alpha's shape).
--     alpha.mdm.rating_agency has 12 rows, 22 cols, no RLS.
--     We add RLS here in crims to match the local control-plane pattern
--     (tenant_id is the key).
-- ============================================================================
CREATE TABLE IF NOT EXISTS crims.mdm.rating_agency (
  id                      uuid PRIMARY KEY,
  agency_cd               varchar(50) NOT NULL,
  name                    varchar(150) NOT NULL,
  short_name              varchar(60),
  agency_type             varchar(30),
  domicile                varchar(2),
  parent_agency_id        uuid,
  is_nrsro                boolean DEFAULT false,
  is_ecai                 boolean DEFAULT false,
  is_naic_acceptable      boolean DEFAULT false,
  is_active               boolean DEFAULT true,
  effective_from          date,
  effective_to            date,
  registration_number     varchar(50),
  registration_authority  varchar(100),
  website                 varchar(255),
  status                  varchar(20),
  custom_attributes       jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at              timestamptz,
  updated_at              timestamptz,
  tenant_id               uuid NOT NULL
);

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='rating_agency_parent_agency_id_fkey') THEN
    ALTER TABLE crims.mdm.rating_agency
      ADD CONSTRAINT rating_agency_parent_agency_id_fkey
      FOREIGN KEY (parent_agency_id) REFERENCES crims.mdm.rating_agency(id);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname='rating_agency_tenant_id_unique') THEN
    ALTER TABLE crims.mdm.rating_agency
      ADD CONSTRAINT rating_agency_tenant_id_unique UNIQUE (tenant_id, agency_cd);
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_rating_agency_tenant ON crims.mdm.rating_agency (tenant_id);
CREATE INDEX IF NOT EXISTS idx_rating_agency_parent ON crims.mdm.rating_agency (parent_agency_id);

ALTER TABLE crims.mdm.rating_agency ENABLE ROW LEVEL SECURITY;

DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='mdm' AND tablename='rating_agency' AND policyname='rating_agency_tenant_read') THEN
    CREATE POLICY rating_agency_tenant_read ON crims.mdm.rating_agency
      FOR SELECT USING (tenant_id::text = current_setting('app.tenant_id')::text);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='mdm' AND tablename='rating_agency' AND policyname='rating_agency_tenant_write') THEN
    CREATE POLICY rating_agency_tenant_write ON crims.mdm.rating_agency
      FOR INSERT WITH CHECK (tenant_id::text = current_setting('app.tenant_id')::text);
  END IF;
  IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='mdm' AND tablename='rating_agency' AND policyname='rating_agency_tenant_modify') THEN
    CREATE POLICY rating_agency_tenant_modify ON crims.mdm.rating_agency
      FOR UPDATE USING (tenant_id::text = current_setting('app.tenant_id')::text);
  END IF;
END $$;

-- ============================================================================
-- (c) Create crims.mdm.rating_outlook / rating_watch / rating_type /
--     rating_action_type.  All 0-row on alpha; mirror alpha's shape so
--     copy_plan.sh's EXISTS check doesn't try to CREATE these (and so they
--     fit the rating_agency FK topology).
-- ============================================================================

CREATE TABLE IF NOT EXISTS crims.mdm.rating_outlook (
  id                    uuid PRIMARY KEY,
  code                  varchar(20) NOT NULL,
  name                  varchar(80) NOT NULL,
  description           varchar(255),
  is_negative           boolean DEFAULT false,
  display_order         integer,
  is_active             boolean DEFAULT true,
  custom_attributes     jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at            timestamptz,
  updated_at            timestamptz,
  tenant_id             uuid NOT NULL,
  UNIQUE (tenant_id, code)
);

CREATE TABLE IF NOT EXISTS crims.mdm.rating_watch (
  id                    uuid PRIMARY KEY,
  code                  varchar(20) NOT NULL,
  name                  varchar(80) NOT NULL,
  description           varchar(255),
  direction             varchar(10),  -- POS/NEG/NEU
  is_active             boolean DEFAULT true,
  custom_attributes     jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at            timestamptz,
  updated_at            timestamptz,
  tenant_id             uuid NOT NULL,
  UNIQUE (tenant_id, code)
);

CREATE TABLE IF NOT EXISTS crims.mdm.rating_type (
  id                    uuid PRIMARY KEY,
  code                  varchar(20) NOT NULL,
  name                  varchar(80) NOT NULL,
  description           varchar(255),
  applies_to            varchar(30),  -- SOVEREIGN, ISSUER, ISSUE, ...
  display_order         integer,
  is_active             boolean DEFAULT true,
  custom_attributes     jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at            timestamptz,
  updated_at            timestamptz,
  tenant_id             uuid NOT NULL,
  UNIQUE (tenant_id, code)
);

CREATE TABLE IF NOT EXISTS crims.mdm.rating_action_type (
  id                    uuid PRIMARY KEY,
  code                  varchar(20) NOT NULL,
  name                  varchar(80) NOT NULL,
  description           varchar(255),
  direction             varchar(10),  -- UP/DOWN/UNCHANGED
  display_order         integer,
  is_active             boolean DEFAULT true,
  custom_attributes     jsonb NOT NULL DEFAULT '{}'::jsonb,
  created_at            timestamptz,
  updated_at            timestamptz,
  tenant_id             uuid NOT NULL,
  UNIQUE (tenant_id, code)
);

DO $$
DECLARE t text;
BEGIN
  FOR t IN SELECT unnest(ARRAY['rating_outlook','rating_watch','rating_type','rating_action_type']) LOOP
    EXECUTE format('CREATE INDEX IF NOT EXISTS idx_%I_tenant ON crims.mdm.%I (tenant_id)', t, t);
    EXECUTE format('ALTER TABLE crims.mdm.%I ENABLE ROW LEVEL SECURITY', t);
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='mdm' AND tablename=t AND policyname=t||'_tenant_read') THEN
      EXECUTE format('CREATE POLICY %I_tenant_read ON crims.mdm.%I FOR SELECT USING (tenant_id::text = current_setting(''app.tenant_id'')::text)', t, t);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='mdm' AND tablename=t AND policyname=t||'_tenant_write') THEN
      EXECUTE format('CREATE POLICY %I_tenant_write ON crims.mdm.%I FOR INSERT WITH CHECK (tenant_id::text = current_setting(''app.tenant_id'')::text)', t, t);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_policies WHERE schemaname='mdm' AND tablename=t AND policyname=t||'_tenant_modify') THEN
      EXECUTE format('CREATE POLICY %I_tenant_modify ON crims.mdm.%I FOR UPDATE USING (tenant_id::text = current_setting(''app.tenant_id'')::text)', t, t);
    END IF;
  END LOOP;
END $$;

-- ============================================================================
-- (d) rating_scale_map: 63 rows on alpha, FK to rating_scale + scale-related
--     lookups.  Doesn't exist on crims; copy_plan.sh will CREATE.  We don't
--     pre-stage here because copy_plan.sh is responsible for the full family
--     DATA, and adding (d) here duplicates the FK topology.  Document instead:
--     copy_plan.sh's CREATE TABLE will land the canonical alpha shape with
--     alpha.mdm FKs (now crims.mdm FKs after move).
-- ============================================================================

-- ============================================================================
-- Add rating family to migration.progress with status=stretched so the
-- executor knows the targets are now in place.
-- ============================================================================
DO $$
DECLARE pid bigint;
BEGIN
  SELECT id INTO pid FROM migration.plan
    WHERE source_schema='mdm' AND source_table='rating_scale' LIMIT 1;
  IF pid IS NOT NULL THEN
    INSERT INTO migration.progress (plan_id, status, started_at, finished_at, detail)
    VALUES (pid, 'ready_to_copy', now(), now(),
            '002c_crims_rating_alter applied; target schema extended (12->24 cols), agency_id FK pending, rename applied, unique key aligned with alpha')
    ON CONFLICT (plan_id) DO UPDATE SET status='ready_to_copy', finished_at=now();
  END IF;
END $$;

COMMIT;

-- QA:
-- SELECT column_name, data_type FROM information_schema.columns
--  WHERE table_schema='mdm' AND table_name='rating_scale' ORDER BY ordinal_position;
-- SELECT count(*) FROM crims.mdm.rating_agency;    -- 0 in dev (still)
