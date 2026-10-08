-- REPLICA IDENTITY FULL, derived from the publication rather than a hardcoded table list.
--
-- 20261224_001 set FULL on the six tables named by the connector's table.include.list
-- as of that date. That list is correct today and rots tomorrow: the day Debezium is
-- given another table, the publication grows and the new table sits at REPLICA IDENTITY
-- DEFAULT again -- and under per-tenant routing every delete on it dead-letters with
-- ERR_TENANT_UNAVAILABLE_ON_DELETE while inserts and updates keep working perfectly.
-- The pipeline looks healthy right up until somebody notices a row that cannot be
-- removed.
--
-- So this migration reads the publication, which IS the capture set, and covers whatever
-- is actually captured. When the connector adds a table to the publication and this
-- migration next runs, the new table is covered without anyone remembering this file.
--
-- THE COST, HONESTLY. FULL writes the complete old row into WAL on UPDATE and DELETE,
-- and the user constraint is "never slow down Postgres". That is real -- but only for
-- tables that are actually modified. A table nobody writes to produces no WAL whatever
-- its replica identity is. So covering the publication costs nothing for tables that
-- are declared but dormant, and the alternative -- covering only a hardcoded list --
-- costs correctness the first time the list is stale.
--
-- WAL retention and slot lag are the operational consequence to watch. The replication
-- slot already owes you that monitoring for its own sake -- a stalled slot retains WAL
-- until the disk fills -- and REPLICA IDENTITY FULL widens what flows through it on every
-- UPDATE and DELETE of a captured table. Add it to that existing alert rather than
-- discovering it as disk pressure at 3am. The "never slow down Postgres" constraint is
-- met here in the sense that matters: nothing extra runs on the write path, and the cost
-- is proportional to rows actually modified, not to the size of the schema.

DO $$
DECLARE
    r    record;
    n    int := 0;
    skipped text := '';
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_publication WHERE pubname = 'orm_cdc_publication') THEN
        RAISE NOTICE 'cdc replica identity: publication orm_cdc_publication not present; nothing to cover';
        RETURN;
    END IF;

    FOR r IN
        SELECT schemaname, tablename
          FROM pg_publication_tables
         WHERE pubname = 'orm_cdc_publication'
         ORDER BY schemaname, tablename
    LOOP
        -- debezium_heartbeat is Debezium's own liveness store. It has no tenant_id and
        -- is never a CDC target, so a fuller old-row image buys nothing.
        IF r.tablename = 'debezium_heartbeat' THEN
            skipped := skipped || ' ' || r.schemaname || '.' || r.tablename;
            CONTINUE;
        END IF;

        -- Skip only what is already FULL. Everything else in the publication needs
        -- the change; in particular REPLICA IDENTITY DEFAULT ('d') is the case this
        -- whole migration exists for, so it must NOT be skipped.
        IF EXISTS (
            SELECT 1 FROM pg_class c
              JOIN pg_namespace n ON n.oid = c.relnamespace
             WHERE n.nspname = r.schemaname
               AND c.relname = r.tablename
               AND c.relreplident = 'f'
        ) THEN
            CONTINUE;
        END IF;

        -- A published name with no matching table means the publication and the
        -- schema disagree, which is worth surfacing rather than silently skipping.
        IF NOT EXISTS (
            SELECT 1 FROM pg_class c
              JOIN pg_namespace n ON n.oid = c.relnamespace
             WHERE n.nspname = r.schemaname
               AND c.relname = r.tablename
        ) THEN
            RAISE WARNING 'cdc replica identity: % is in the publication but not in the schema', r.schemaname || '.' || r.tablename;
            CONTINUE;
        END IF;

        EXECUTE format('ALTER TABLE %I.%I REPLICA IDENTITY FULL', r.schemaname, r.tablename);
        n := n + 1;
    END LOOP;

    RAISE NOTICE 'cdc replica identity: FULL applied to % published table(s)', n;
    IF skipped <> '' THEN
        RAISE NOTICE 'cdc replica identity: deliberately skipped:%', skipped;
    END IF;
END $$;

-- Verification. Every published table except debezium_heartbeat -- which is skipped on
-- purpose -- must report 'f'. A 'd' row here is a table whose deletes will
-- dead-letter under routing.
--
--   SELECT p.schemaname, p.tablename, c.relreplident
--     FROM pg_publication_tables p
--     JOIN pg_class c ON c.relname = p.tablename
--     JOIN pg_namespace n ON n.oid = c.relnamespace AND n.nspname = p.schemaname
--    WHERE p.pubname = 'orm_cdc_publication'
--      AND p.tablename <> 'debezium_heartbeat'
--      AND c.relreplident <> 'f'
--    ORDER BY 1, 2;
--
-- Expected: no rows. (debezium_heartbeat is excluded because this migration skips it
-- deliberately; counting it would report a permanent phantom failure.)
--
-- If it does return rows, the table was added to the publication after this migration
-- last ran. Re-run this migration; there is no automatic detection, because the loader
-- holds no Postgres connection by design and inventing one here would couple the CDC
-- path to the control plane it is supposed to read from.