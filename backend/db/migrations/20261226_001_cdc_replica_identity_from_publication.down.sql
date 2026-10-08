-- Restore REPLICA IDENTITY DEFAULT on every published table.
--
-- Read 20261226_001_cdc_replica_identity_from_publication.up.sql first. Reverting
-- reintroduces the routing bug: `before` stops carrying the row, so tenant_id is
-- unknowable on a delete, every delete dead-letters as ERR_TENANT_UNAVAILABLE_ON_DELETE,
-- and rows accumulate in each tenant's StarRocks database that no longer exist in
-- Postgres. This exists to undo a partial application, not as routine cleanup.

DO $$
DECLARE
    r    record;
    n    int := 0;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_publication WHERE pubname = 'orm_cdc_publication') THEN
        RAISE NOTICE 'cdc replica identity: publication orm_cdc_publication not present; nothing to revert';
        RETURN;
    END IF;

    FOR r IN
        SELECT schemaname, tablename
          FROM pg_publication_tables
         WHERE pubname = 'orm_cdc_publication'
         ORDER BY schemaname, tablename
    LOOP
        IF NOT EXISTS (
            SELECT 1 FROM pg_class c
              JOIN pg_namespace n ON n.oid = c.relnamespace
             WHERE n.nspname = r.schemaname
               AND c.relname = r.tablename
        ) THEN
            CONTINUE;
        END IF;
        EXECUTE format('ALTER TABLE %I.%I REPLICA IDENTITY DEFAULT', r.schemaname, r.tablename);
        n := n + 1;
    END LOOP;

    RAISE NOTICE 'cdc replica identity: DEFAULT restored on % published table(s)', n;
END $$;