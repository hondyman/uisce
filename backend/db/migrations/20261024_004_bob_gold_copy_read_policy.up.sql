-- 20261024_004_bob_gold_copy_read_policy.up.sql
--
-- Regular tenants inherit gold-copy metadata read-only. business_object_bindings (plural), which had
-- no RLS, let every tenant read the gold-copy tenant's bindings for inherited BOs; the singular table
-- that replaces it has forced RLS with current-tenant-only policies, so without this the inherited
-- BOs would lose their bindings for regular tenants.
--
-- One new SELECT-only policy, same shape as the gold-copy read policies on other tables. It exposes
-- only the gold-copy tenant's rows and grants no write: inserts, updates and deletes remain governed by
-- tenant_isolation_bob / tenant_isolation_policy, which are unchanged. Policies are permissive, so
-- this ORs with them for reads.

DROP POLICY IF EXISTS bob_read_gold_copy ON public.business_object_binding;
CREATE POLICY bob_read_gold_copy ON public.business_object_binding
    FOR SELECT
    USING (tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1));
