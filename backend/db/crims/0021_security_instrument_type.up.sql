-- 0021_security_instrument_type.up.sql
-- mdm.security.required_terms (catalog_node 7c88ad55, severity BLOCK) requires
-- SecTypCd, which the field map resolves to instrument_type. The pipeline's map
-- node already produces SecTypCd from the file's SECURITY_TYP, but the
-- staging_sink's column mapping never persisted it, so every SECURITY record
-- reached mastering with a required attribute missing and was held for review:
-- 12 valid, 0 published.
--
-- This adds the staging column the sink needs and backfills what is already
-- staged, from staging.bbg_security which carries the same instruments' source
-- security_typ. The pipeline DAG must also map SecTypCd -> instrument_type;
-- this migration only makes the target legal and closes the gap on rows
-- already staged.
--
-- Additive and idempotent. Run against crims.
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0021_security_instrument_type.up.sql

\set ON_ERROR_STOP on
BEGIN;

ALTER TABLE staging.security_data
    ADD COLUMN IF NOT EXISTS instrument_type character varying(50);

COMMENT ON COLUMN staging.security_data.instrument_type IS
    'Instrument type from the source (Bloomberg SECURITY_TYP), mapped to the SecTypCd field name by the pipeline map node.';

-- Close the gap on rows already staged. bbg_security is the same feed loaded
-- through the vendor staging table; match on ISIN, then on the source row id as
-- a fallback for rows that carry no ISIN.
UPDATE staging.security_data s
   SET instrument_type = b.security_typ
  FROM staging.bbg_security b
 WHERE b.security_typ IS NOT NULL
   AND b.tenant_id = s.tenant_id
   AND ( (b.id_isin IS NOT NULL AND b.id_isin = s.isin)
      OR (s.isin IS NULL AND b.id_bb_global = s.security_id) )
   AND s.instrument_type IS NULL;

COMMIT;
