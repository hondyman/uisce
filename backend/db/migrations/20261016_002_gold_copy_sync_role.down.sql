-- 20261016_002_gold_copy_sync_role.down.sql
-- Reverts 20261016_002_gold_copy_sync_role.up.sql.
-- DROP ROLE fails outright while the role still holds granted privileges
-- (verified directly: "role ... cannot be dropped because some objects
-- depend on it" / "privileges for table ...") — REASSIGN/DROP OWNED first.

DROP OWNED BY uisce_gold_copy_sync;
DROP ROLE IF EXISTS uisce_gold_copy_sync;
