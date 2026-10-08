-- REPLICA IDENTITY FULL on every CDC-captured business table.
--
-- WHY. Per-tenant routing (PR #424) dispatches a row by the tenant_id carried on the
-- event. Inserts and updates carry the whole row, so they route. Deletes carry only
-- what Postgres' logical decoding plugin was asked to send, which under the default
-- replica identity is the table's key columns -- and on orm.* the key is `id` alone.
-- Debezium fills every non-key field in `before` with its schema zero value, so
--
--     before.tenant_id == ""      (zero-fill, NOT the tenant)
--     before.created_at == "1970-01-01T00:00:00.000000Z"
--     before.trade_date == 0
--
-- and the loader, correctly refusing to guess a destination, dead-letters the delete.
--
-- This was invisible before routing: with a single configured destination the delete
-- still executed, because the destination did not depend on tenant_id. Routing turned
-- a working delete into a guaranteed dead-letter -- every delete, every table, every
-- tenant, with rows left in StarRocks that no longer exist in Postgres.
--
-- FULL puts the old row in `before`, so a delete carries the same tenant_id an insert
-- would, and routes through the same path. Verified live on alpha.orm."order" on
-- 2026-10-08: with DEFAULT the delete dead-lettered as ERR_TENANT_UNATTRIBUTED;
-- after FULL the identical delete routed to the correct tenant database.
--
-- SCOPE. Exactly the tables named by the connector's table.include.list for
-- orm-oms-connector-v2, minus orm.debezium_heartbeat -- that table is Debezium's own
-- heartbeat store, has no tenant_id, and gains nothing from a fuller row image.
-- publication.autocreate.mode is `disabled` and the publication is explicit, so
-- replica identity changes need no publication change.
--
-- COST. A fuller old-row image means more WAL for DELETE/UPDATE on these five tables.
-- That is the correct trade: the alternative is a warehouse that cannot remove a row.

-- Guarded, because this migration set is applied to more than one database: a
-- tenant data plane or a test fixture has no `orm` schema at all, and an unguarded
-- ALTER would fail the whole migration there. A missing table is not a silent skip --
-- a database with no orm.* tables has no connector reading them either -- so the
-- branch raises a NOTICE rather than doing nothing quietly.
DO $$
DECLARE
    t   text;
    n   int := 0;
BEGIN
    FOREACH t IN ARRAY ARRAY[
        'orm.execution', 'orm.order', 'orm.placement',
        'orm.order_allocation', 'orm.execution_allocation'
    ] LOOP
        IF to_regclass(t) IS NULL THEN
            RAISE NOTICE 'cdc replica identity: % not present, skipped', t;
            CONTINUE;
        END IF;
        EXECUTE format('ALTER TABLE %s REPLICA IDENTITY FULL', t);
        n := n + 1;
    END LOOP;
    IF n > 0 THEN
        RAISE NOTICE 'cdc replica identity: FULL set on % table(s)', n;
    END IF;
END $$;

-- Verification: every captured business table must report 'f' (full). Anything else
-- means deletes on that table will dead-letter as ERR_TENANT_UNAVAILABLE_ON_DELETE.
--
--   SELECT c.relname, c.relreplident
--     FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
--    WHERE n.nspname = 'orm'
--      AND c.relname IN ('execution','order','placement','order_allocation','execution_allocation')
--    ORDER BY c.relname;
--
-- Expected: five rows, relreplident = 'f' for all five.