-- REPLICA IDENTITY FULL on every table in the tenant's orm schema.
--
-- WHY THIS FILE EXISTS IN A TENANT MIGRATION DIRECTORY. 0001 creates the ORM tables in
-- a tenant's own Postgres database, and it creates them the way Postgres creates every
-- table: at REPLICA IDENTITY DEFAULT. That is exactly the configuration that makes CDC
-- deletes unroutable -- under the default identity the logical decoding plugin sends
-- only the key columns, Debezium zero-fills the rest, and `before.tenant_id` arrives as
-- an empty string. Per-tenant routing dispatches on that value, so every delete in
-- every tenant database would dead-letter with ERR_TENANT_UNAVAILABLE_ON_DELETE while
-- inserts and updates sailed through. See ADR-042 for why the orm schema exists per
-- tenant, and backend/db/migrations/20261226_001_cdc_replica_identity_from_publication.up.sql
-- for the same fix applied to alpha.
--
-- This is a schema property, so it belongs in schema DDL. Backfilling it later means
-- every deployment that skipped the step silently accumulates undeletable rows -- the
-- three-weeks-silent-absence class, wearing a different hat.
--
-- Note this runs unconditionally over the schema rather than reading a publication:
-- a tenant database has no orm_cdc_publication of its own yet, and depending the
-- migration on connector configuration that does not exist there would make it a no-op
-- precisely when it is needed. Cost is bounded for the same reason as everywhere else --
-- FULL only costs WAL on tables that are actually modified.
--
-- See 0001_orm_schema.up.sql for the tables created here. If that file grows a table,
-- this covers it automatically.

DO $$
DECLARE
    r    record;
    n    int := 0;
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'orm') THEN
        RAISE NOTICE 'cdc replica identity: schema orm not present; nothing to cover';
        RETURN;
    END IF;

    FOR r IN
        SELECT c.relname
          FROM pg_class c
          JOIN pg_namespace n ON n.oid = c.relnamespace
         WHERE n.nspname = 'orm'
           AND c.relkind IN ('r', 'p')
           AND c.relreplident <> 'f'
         ORDER BY c.relname
    LOOP
        EXECUTE format('ALTER TABLE orm.%I REPLICA IDENTITY FULL', r.relname);
        n := n + 1;
    END LOOP;

    RAISE NOTICE 'cdc replica identity: FULL applied to % orm table(s) in this tenant database', n;
END $$;

-- Verification, for a tenant database:
--
--   SELECT count(*) FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
--    WHERE n.nspname = 'orm' AND c.relkind IN ('r','p') AND c.relreplident <> 'f';
--
-- Expected: 0.