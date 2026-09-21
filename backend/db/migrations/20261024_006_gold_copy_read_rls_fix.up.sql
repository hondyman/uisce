-- 20261024_006_gold_copy_read_rls_fix.up.sql
--
-- Gold-copy read access under RLS.
--
-- bob_read_gold_copy (004) and catalog_node_read_gold_copy_rules (005) resolved the gold tenant with
-- (SELECT id FROM public.tenants WHERE gold_copy). public.tenants is under RLS and shows a tenant only its
-- own row, so for a regular tenant that subquery is NULL and neither policy could ever match. It went
-- unnoticed while the backend connected as a superuser (RLS bypassed).
--
-- Separately, business_objects_tenant_gold_policy was FOR ALL, so its USING expression also gated writes and
-- would let a regular tenant modify gold-copy business objects. Tenants inherit gold-copy metadata
-- read-only; they never write back to it.
--
-- 1. uisce_gold_copy_tenant_id(): SECURITY DEFINER, no arguments, returns only the gold-copy tenant id.
--    Derived server-side, so it cannot be steered by a session setting.
-- 2. The two policies above now call it. Still SELECT-only and narrow.
-- 3. business_objects: a tenant writes only its own rows and reads the gold copy.
--
-- Idempotent.

CREATE OR REPLACE FUNCTION public.uisce_gold_copy_tenant_id() RETURNS uuid
    LANGUAGE sql STABLE SECURITY DEFINER SET search_path = pg_catalog, public
AS $$ SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1 $$;

REVOKE ALL ON FUNCTION public.uisce_gold_copy_tenant_id() FROM PUBLIC;
GRANT EXECUTE ON FUNCTION public.uisce_gold_copy_tenant_id() TO PUBLIC;

DROP POLICY IF EXISTS bob_read_gold_copy ON public.business_object_binding;
CREATE POLICY bob_read_gold_copy ON public.business_object_binding
    FOR SELECT
    USING (tenant_id = public.uisce_gold_copy_tenant_id());

DROP POLICY IF EXISTS catalog_node_read_gold_copy_rules ON public.catalog_node;
CREATE POLICY catalog_node_read_gold_copy_rules ON public.catalog_node
    FOR SELECT
    USING (
        tenant_id = public.uisce_gold_copy_tenant_id()
        AND node_type_id = (SELECT id FROM public.catalog_node_type WHERE catalog_type_name = 'validation_rule')
    );

DROP POLICY IF EXISTS business_objects_tenant_gold_policy ON public.business_objects;
DROP POLICY IF EXISTS business_objects_tenant_own ON public.business_objects;
CREATE POLICY business_objects_tenant_own ON public.business_objects
    FOR ALL
    USING (tenant_id = public.uisce_get_current_tenant())
    WITH CHECK (tenant_id = public.uisce_get_current_tenant());

DROP POLICY IF EXISTS business_objects_read_gold_copy ON public.business_objects;
CREATE POLICY business_objects_read_gold_copy ON public.business_objects
    FOR SELECT
    USING (tenant_id = public.uisce_gold_copy_tenant_id());
