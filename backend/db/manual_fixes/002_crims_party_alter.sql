-- 002_crims_party_alter.sql
-- Extends crims.mdm.party with alpha-only golden-record fields.
-- Idempotent: ADD COLUMN IF NOT EXISTS so reruns are safe.
-- Runs against CRIMS as the master of the party table under data-plane.
-- Per Option A (2026-09-23): alpha.mdm.party will be DROPped in batch 900;
--   rich model lives in crims.mdm.party. RLS key (tenant_id) is shared ->
--   existing crims RLS policies continue to apply without modification.
-- The ALTER preserves crims' lightweight identity columns (party_cd,
--   legal_name, party_type, segment, tax_id, domicile) and layers alpha's
--   golden-record fields on top.

SET search_path = migration, crims, public;

BEGIN;

-- Notify migration.progress that 002 is running (per migration.plan id=1)
-- The actual party row has classification=DATA in migration.plan; progress is
-- driven manually by the executor when this script completes.

-- === ADD COLUMN: alpha-only fields layered onto crims.mdm.party ===
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS client_group_id uuid NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS client_since_date date NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS country_of_risk_cd varchar(2) NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS date_of_birth date NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS date_of_death date NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS dq_score numeric NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS giin varchar(20) NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS golden_record_id uuid NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS incorporation_date date NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS incorporation_jurisdiction varchar(10) NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS is_deceased boolean NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS is_financial_institution boolean NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS is_golden_record boolean ;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS is_individual boolean NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS is_legal_entity boolean NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS is_pep boolean NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS is_regulated boolean NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS is_restricted boolean NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS is_sanctioned boolean NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS is_us_person boolean NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS last_reviewed_at timestamptz NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS legal_form varchar(50) NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS lei varchar(20) NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS lifecycle_stage varchar(20) ;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS merged_into_id uuid NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS parent_party_id uuid NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS party_sub_type varchar(50) NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS primary_currency varchar(3) NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS primary_nationality_cd varchar(2) NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS primary_residence_country_cd varchar(2) NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS primary_segment_id uuid NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS primary_tax_country_cd varchar(2) NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS registration_number varchar(100) NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS relationship_end_date date NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS relationship_manager_id uuid NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS review_frequency_months integer NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS servicing_team_id uuid NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS source_system_id uuid NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS status varchar(20) ;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS steward_id uuid NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS ultimate_parent_party_id uuid NULL;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS valid_from timestamptz ;
ALTER TABLE crims.mdm.party ADD COLUMN IF NOT EXISTS valid_to timestamptz NULL;

-- === RLS unchanged ===
-- crims.mdm.party RLS policies already key on tenant_id and custom_attributes.
-- Adding new columns does NOT affect policy validity. The 5 RLS policies
-- (party_tenant_read, party_tenant_write, tenant_isolation, rating_scale*,
-- source_system*) attached to mdm.party / mdm.rating_scale / mdm.source_system
-- remain in effect. alpha.mdm.party's policies (alpha.mdm.party has no RLS —
-- see step0 RLS probe 2026-09-23) disappear with the DROP batch.

COMMIT;

