-- Restore REPLICA IDENTITY DEFAULT on every table in this tenant's orm schema.
--
-- Read 0002_replica_identity_full.up.sql first. Reverting makes every CDC delete in
-- this tenant database unroutable: `before` stops carrying the row, tenant_id arrives
-- zero-filled, and rows are dead-lettered instead of removed. Only undo a partial
-- application with this; it is not cleanup.

DO $$
DECLARE
    r    record;
    n    int := 0;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'orm') THEN
        RAISE NOTICE 'cdc replica identity: schema orm not present; nothing to revert';
        RETURN;
    END IF;

    FOR r IN
        SELECT c.relname
          FROM pg_class c
          JOIN pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname = 'orm'
           AND c.relkind IN ('r', 'p')
         ORDER BY c.relname
    LOOP
        EXECUTE format('ALTER TABLE orm.%I REPLICA IDENTITY DEFAULT', r.relname);
        n := n + 1;
    END LOOP;

    RAISE NOTICE 'cdc replica identity: DEFAULT restored on % orm table(s)', n;
END $$;