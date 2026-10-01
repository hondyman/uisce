-- Apply:
--   psql "$CRIMS_DSN" -1 -v ON_ERROR_STOP=1 -f 0019_mastering_config_changes.down.sql
\set ON_ERROR_STOP on
BEGIN;
DROP TABLE IF EXISTS mdm.mastering_config_change;
COMMIT;
