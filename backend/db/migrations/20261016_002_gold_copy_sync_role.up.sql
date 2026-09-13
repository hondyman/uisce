-- 20261016_002_gold_copy_sync_role.up.sql
-- Prerequisite for landing 20261016_001_strict_tenant_rls.up.sql safely.
--
-- That migration replaces a fail-open RLS policy with a strict, fail-closed
-- one on tenant_instance, tenant_product, tenant_product_datasource,
-- connections, and audit_logs. Auditing every direct query against those
-- five tables (grep across backend/**, checked for tenant-context setting)
-- found the gold-copy sync/provisioning subsystem is *structurally*
-- cross-tenant: it reads the gold-copy tenant's rows and writes into every
-- other tenant's rows in the same operation, often in loops
-- (internal/db/instance_clone.go's CloneGoldCopyInstance,
-- SyncGoldCopyConnectionToAllInstances, SyncAllConnectionsForInstance;
-- internal/temporal/activities/gold_copy_activities.go's
-- syncConnectionToTenant; internal/api/connection_sync_handler.go's
-- SyncConnectionsFromGoldCopy; internal/sync/tenant_worker.go's tenant
-- deprovisioning). None of these set a single tenant's session GUC, because
-- there isn't one — the whole point is touching every tenant.
--
-- Postgres's BYPASSRLS is a role-level attribute, not a per-table one — it
-- can't be scoped by grant the way SELECT/INSERT/UPDATE/DELETE can. So the
-- scoping here comes from what this role is granted, not from BYPASSRLS
-- itself: it gets BYPASSRLS plus explicit grants on exactly the five tables
-- this subsystem touches, and nothing else. Same pattern as
-- 20260916_001_app_admin_read_role.up.sql (monitoring reads); this is the
-- write-capable equivalent for gold-copy sync and tenant provisioning.
--
-- Wiring update: the confirmed cross-tenant call sites now assume this
-- role transiently, inside an already-open transaction, via `SET LOCAL
-- ROLE uisce_gold_copy_sync` (internal/db/cross_tenant.go's
-- WithGoldCopySync). `SET LOCAL ROLE` is transaction-scoped, exactly like
-- `SET LOCAL <parameter>` — it reverts automatically when the
-- transaction ends, success or failure, which matters specifically
-- because Go's database/sql pools physical connections: without that
-- transaction-scoping, an elevated role could leak onto a pooled
-- connection and silently grant cross-tenant bypass to a completely
-- unrelated later request. Verified directly (not assumed) against a
-- real Go connection pool with MaxOpenConns=1 forcing physical reuse:
-- successful-completion path, aborted-transaction path,
-- panic-mid-transaction path, and 50 concurrent interleaved goroutines
-- all showed zero leakage — every non-elevated query on a reused
-- connection saw the plain connecting role, never uisce_gold_copy_sync.
--
-- For the connecting role to `SET ROLE` into uisce_gold_copy_sync, it
-- must be a member of it. Only app_user needs that grant here — a
-- superuser (e.g. `postgres`) can already `SET ROLE` to any role
-- unconditionally, verified directly: no grant required, and granting one
-- would be a no-op, not an additional safeguard.
--
-- Also verified directly: BYPASSRLS never propagates through role
-- membership, regardless of INHERIT/NOINHERIT — it is a role-level
-- attribute like SUPERUSER, not a privilege, and only takes effect via an
-- explicit SET ROLE. So granting membership to app_user with the default
-- INHERIT is safe: it does not cause app_user's ordinary, non-elevated
-- queries to silently bypass RLS. No NOINHERIT/Postgres-version-specific
-- syntax is needed.
--
-- public.tenants read access: found only by running the wired functions
-- end to end against a real database, not by re-reading the grant list —
-- findGoldCopyInstance, PropagateConnectionActivity,
-- SyncConnectionsFromGoldCopy, and the deletion-cascade functions all
-- JOIN or SELECT against public.tenants (to resolve "the gold-copy
-- tenant" or check a tenant's gold_copy flag) from inside the elevated
-- transaction. public.tenants itself carries no RLS policy, so this
-- isn't a bypass concern — it was a plain missing GRANT, and without it
-- every one of those calls fails outright with "permission denied for
-- table tenants" the moment SET LOCAL ROLE takes effect. SELECT only:
-- nothing in the wired call sites writes to the tenant registry itself.

CREATE ROLE uisce_gold_copy_sync WITH LOGIN NOCREATEDB NOCREATEROLE BYPASSRLS;

GRANT SELECT ON public.tenants TO uisce_gold_copy_sync;

GRANT SELECT, INSERT, UPDATE, DELETE ON
    public.tenant_instance,
    public.tenant_product,
    public.tenant_product_datasource,
    public.connections
TO uisce_gold_copy_sync;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'audit_logs') THEN
        GRANT SELECT, INSERT ON public.audit_logs TO uisce_gold_copy_sync;
    END IF;
END $$;

GRANT uisce_gold_copy_sync TO app_user;

COMMENT ON ROLE uisce_gold_copy_sync IS 'System role for gold-copy sync and tenant provisioning/deprovisioning: structurally cross-tenant operations that cannot run under a single tenant''s RLS context. BYPASSRLS, scoped in practice to tenant_instance, tenant_product, tenant_product_datasource, connections, and audit_logs via explicit grants only — no other table access. Assumed transiently via SET LOCAL ROLE inside a transaction (see internal/db/cross_tenant.go), never as a standing connection identity.';
