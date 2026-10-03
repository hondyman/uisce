-- 20261206_001_tenant_datasource_binding (down)
--
-- Drops the tenant lakehouse registry and the datasource bindings. Both hold
-- references to resources (warehouses, buckets, KMS keys, databases, roles) that
-- continue to exist after this runs; dropping the tables orphans them from the
-- registry. tenant_lakehouse in particular is the only record tying a tenant to
-- its WORM audit bucket and the key that decrypts it. Do not run this against an
-- environment where tenants have been provisioned without first exporting both
-- tables.
DROP TABLE IF EXISTS public.tenant_datasource_binding;
DROP TABLE IF EXISTS public.tenant_lakehouse;
DROP FUNCTION IF EXISTS public.tenant_lakehouse_guard();
