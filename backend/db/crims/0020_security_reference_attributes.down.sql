-- 0020_security_reference_attributes.down.sql
-- Drops the columns added by 0020. sector/industry are NOT dropped: they
-- predate this migration and only received a backfill here.
--
-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0020_security_reference_attributes.down.sql

\set ON_ERROR_STOP on
BEGIN;

ALTER TABLE mdm.security_master
    DROP COLUMN IF EXISTS bloomberg_unique_id,
    DROP COLUMN IF EXISTS callable_flag,
    DROP COLUMN IF EXISTS cic_code,
    DROP COLUMN IF EXISTS contract_size,
    DROP COLUMN IF EXISTS country_of_incorporation,
    DROP COLUMN IF EXISTS coupon_type,
    DROP COLUMN IF EXISTS day_count,
    DROP COLUMN IF EXISTS esg_score,
    DROP COLUMN IF EXISTS figi_share_class,
    DROP COLUMN IF EXISTS lei,
    DROP COLUMN IF EXISTS lot_size,
    DROP COLUMN IF EXISTS mifid_target_market,
    DROP COLUMN IF EXISTS multiplier,
    DROP COLUMN IF EXISTS option_style,
    DROP COLUMN IF EXISTS payment_frequency,
    DROP COLUMN IF EXISTS puttable_flag,
    DROP COLUMN IF EXISTS sfdr_article,
    DROP COLUMN IF EXISTS strike_price,
    DROP COLUMN IF EXISTS tick_size,
    DROP COLUMN IF EXISTS trading_status,
    DROP COLUMN IF EXISTS underlying_security_id;

COMMIT;
