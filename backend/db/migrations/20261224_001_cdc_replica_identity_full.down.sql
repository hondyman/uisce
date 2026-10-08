-- Revert to the default replica identity.
--
-- Read the warning in 20261224_001_cdc_replica_identity_full.up.sql before running
-- this. Reverting reintroduces the routing bug: with tenant_id absent from `before`,
-- every DELETE on these tables dead-letters as ERR_TENANT_UNAVAILABLE_ON_DELETE and no
-- row is ever removed from the tenant's StarRocks database. Use this only to undo a
-- partial application, not as routine cleanup.
--
-- Guarded for the same reason as the up migration: applied to a database with no
-- `orm` schema, this must not fail the whole migration run.

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
        EXECUTE format('ALTER TABLE %s REPLICA IDENTITY DEFAULT', t);
        n := n + 1;
    END LOOP;
    IF n > 0 THEN
        RAISE NOTICE 'cdc replica identity: DEFAULT restored on % table(s)', n;
    END IF;
END $$;