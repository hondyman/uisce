-- 0020_security_reference_attributes.up.sql
-- The 24 extension attributes staged into staging.security_data's
-- custom_attributes (staging binding 20260929) had no home on the golden
-- record: project() only writes attributes that are columns on the anchor
-- (mastering/publish.go), so every one of them was silently dropped at publish.
--
-- This adds first-class columns to mdm.security_master for them. Three are
-- renamed to the vocabulary the rest of the anchor already uses, so the
-- survivorship field groups and the field map line up:
--   gics_sector             -> sector
--   gics_industry           -> industry
--   country_of_incorporation-> country_of_incorporation (kept; distinct from
--                              country_of_issue, which is where the risk is)
-- trading_status is kept separate from status: status is the record's lifecycle
-- state, trading_status is the exchange's session state for the listing.
--
-- Additive and idempotent. Run against crims.
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0020_security_reference_attributes.up.sql

\set ON_ERROR_STOP on
BEGIN;

ALTER TABLE mdm.security_master
    ADD COLUMN IF NOT EXISTS bloomberg_unique_id       character varying(64),
    ADD COLUMN IF NOT EXISTS callable_flag              boolean,
    ADD COLUMN IF NOT EXISTS cic_code                   character varying(20),
    ADD COLUMN IF NOT EXISTS contract_size              numeric(20,6),
    ADD COLUMN IF NOT EXISTS country_of_incorporation   character varying(64),
    ADD COLUMN IF NOT EXISTS coupon_type                character varying(30),
    ADD COLUMN IF NOT EXISTS day_count                  character varying(20),
    ADD COLUMN IF NOT EXISTS esg_score                  numeric(5,2),
    ADD COLUMN IF NOT EXISTS figi_share_class           character varying(64),
    ADD COLUMN IF NOT EXISTS lei                        character varying(20),
    ADD COLUMN IF NOT EXISTS lot_size                   numeric(20,6),
    ADD COLUMN IF NOT EXISTS mifid_target_market        character varying(100),
    ADD COLUMN IF NOT EXISTS multiplier                 numeric(20,6),
    ADD COLUMN IF NOT EXISTS option_style               character varying(20),
    ADD COLUMN IF NOT EXISTS payment_frequency          character varying(20),
    ADD COLUMN IF NOT EXISTS puttable_flag              boolean,
    ADD COLUMN IF NOT EXISTS sfdr_article               character varying(10),
    ADD COLUMN IF NOT EXISTS strike_price               numeric(20,6),
    ADD COLUMN IF NOT EXISTS tick_size                  numeric(20,6),
    ADD COLUMN IF NOT EXISTS trading_status             character varying(30),
    ADD COLUMN IF NOT EXISTS underlying_security_id     character varying(100);

-- gics_sector and gics_industry were staged under those names but the anchor
-- already has sector/industry, and the "Security Master Binding" field map
-- binds Sector/Industry to them. Backfill from the already-staged rows before
-- the binding is switched over, so nothing is lost.
UPDATE mdm.security_master m
   SET sector   = COALESCE(m.sector,   s.custom_attributes->>'gics_sector')
  FROM staging.security_data s
 WHERE s.security_id = m.security_id
   AND m.tenant_id = s.tenant_id
   AND s.custom_attributes->>'gics_sector' IS NOT NULL
   AND m.sector IS NULL;

UPDATE mdm.security_master m
   SET industry = COALESCE(m.industry, s.custom_attributes->>'gics_industry')
  FROM staging.security_data s
 WHERE s.security_id = m.security_id
   AND m.tenant_id = s.tenant_id
   AND s.custom_attributes->>'gics_industry' IS NOT NULL
   AND m.industry IS NULL;

COMMENT ON COLUMN mdm.security_master.esg_score IS
    'Composite ESG score, 0-100. Advisory only; not a mandate constraint input until mapped.';
COMMENT ON COLUMN mdm.security_master.sfdr_article IS
    'EU SFDR article classification (6, 8, 9) for the instrument.';
COMMENT ON COLUMN mdm.security_master.mifid_target_market IS
    'MiFID II target market, comma-separated free text as supplied by the source.';
COMMENT ON COLUMN mdm.security_master.trading_status IS
    'Exchange session state for the listing. Distinct from status, which is the record lifecycle state.';
COMMENT ON COLUMN mdm.security_master.country_of_incorporation IS
    'ISO country of incorporation of the issuer. Distinct from country_of_issue and country_of_risk.';
COMMENT ON COLUMN mdm.security_master.bloomberg_unique_id IS
    'Bloomberg unique instrument identifier. Distinct from bbg_id, which is the source instrument key.';
COMMENT ON COLUMN mdm.security_master.underlying_security_id IS
    'Identifier of the instrument underlying a derivative.';

COMMIT;
