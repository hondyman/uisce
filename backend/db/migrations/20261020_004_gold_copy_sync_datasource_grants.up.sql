-- RECONSTRUCTION (2026-09-21). The original file was applied to the shared dev DB
-- on 2026-09-18 21:35 UTC and its SQL is unrecoverable: it exists in no git ref
-- and no worktree. See backend/docs/migrations/MIGRATION_LOSS_RECORD.md.
--
-- What the live DB has for role uisce_gold_copy_sync that NO other migration grants
-- (20261016_002 and 20261016_004 already grant the tenant_* / connections /
-- audit_logs / fix_tenant_tag_mapping write privileges):
--   1. SELECT on every relation in schema public (1001 of 1001 at reconstruction
--      time, zero exceptions).
--   2. A default-privileges entry (uisce_gold_copy_sync=r/postgres) so relations
--      created later are readable too.
--   3. USAGE on schema public.
--
-- Purpose (from code): uisce_gold_copy_sync is the structurally cross-tenant system
-- role. internal/security/datasource_resolver.go resolves which tenant owns a
-- datasource before tenant membership is checked, and needs it; gold-copy sync and
-- tenant provisioning use it too. Cross-tenant support for global admins depends on it.
--
-- THIS IS A FAITHFUL RECONSTRUCTION OF LIVE STATE, NOT A DESIGN ENDORSEMENT. The role
-- is LOGIN + BYPASSRLS and this grants it read access to all of public. Narrowing it
-- to the tables it needs is a tracked security follow-up; it is deliberately not
-- changed here so parity with the live DB is exact.
--
-- Behaviour: idempotent (GRANT and ALTER DEFAULT PRIVILEGES are). On the shared dev
-- DB this is already recorded as applied under a different content hash, so the
-- runner logs "content has changed" and skips it. On a fresh database it reproduces
-- the live grants. The role itself is created by 20261016_002.

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'uisce_gold_copy_sync') THEN
        RAISE NOTICE 'gold_copy_sync_datasource_grants: role uisce_gold_copy_sync absent (created by 20261016_002), skipping';
        RETURN;
    END IF;

    GRANT USAGE ON SCHEMA public TO uisce_gold_copy_sync;
    GRANT SELECT ON ALL TABLES IN SCHEMA public TO uisce_gold_copy_sync;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO uisce_gold_copy_sync;
END
$$;
