-- Restores the previous policies (including the FOR ALL gold policy on business_objects, which lets a
-- tenant write gold-copy business objects; that is the reason for the fix, so only roll back deliberately).
DROP POLICY IF EXISTS business_objects_read_gold_copy ON public.business_objects;
DROP POLICY IF EXISTS business_objects_tenant_own ON public.business_objects;
CREATE POLICY business_objects_tenant_gold_policy ON public.business_objects
    FOR ALL
    USING ((tenant_id = public.uisce_get_current_tenant()) OR (tenant_id = public.uisce_get_gold_tenant()));

DROP POLICY IF EXISTS catalog_node_read_gold_copy_rules ON public.catalog_node;
CREATE POLICY catalog_node_read_gold_copy_rules ON public.catalog_node
    FOR SELECT
    USING (
        tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
        AND node_type_id = (SELECT id FROM public.catalog_node_type WHERE catalog_type_name = 'validation_rule')
    );

DROP POLICY IF EXISTS bob_read_gold_copy ON public.business_object_binding;
CREATE POLICY bob_read_gold_copy ON public.business_object_binding
    FOR SELECT
    USING (tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1));

DROP FUNCTION IF EXISTS public.uisce_gold_copy_tenant_id();
