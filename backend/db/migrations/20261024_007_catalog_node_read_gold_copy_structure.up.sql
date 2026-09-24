-- 20261024_007_catalog_node_read_gold_copy_structure.up.sql
--
-- A tenant inherits the gold-copy tenant's business objects, and resolving their semantic terms and
-- bindings to physical columns reads the gold-copy tenant's `table` and `column` catalog nodes
-- (MAPS_TO targets and binding driving nodes). catalog_node has forced RLS, and until now a tenant could
-- read only its own nodes plus the gold-copy validation rules (20261024_005), so inherited terms and
-- bindings could not be resolved and every inherited BO reported unresolved terms.
--
-- One more SELECT-only policy for exactly the physical-structure node types in the gold-copy tenant.
-- Semantic terms, business terms, API nodes and every other gold-copy node stay invisible to tenants,
-- and writes remain governed by tenant_isolation_policy (unchanged), so no tenant can modify the gold copy.
-- Policies are permissive, so this ORs with the existing ones for reads.
--
-- Uses uisce_gold_copy_tenant_id() (20261024_006), because public.tenants is not readable by a tenant.
--
-- Idempotent.

DROP POLICY IF EXISTS catalog_node_read_gold_copy_structure ON public.catalog_node;
CREATE POLICY catalog_node_read_gold_copy_structure ON public.catalog_node
    FOR SELECT
    USING (
        tenant_id = public.uisce_gold_copy_tenant_id()
        AND node_type_id IN (
            SELECT id FROM public.catalog_node_type WHERE catalog_type_name IN ('table', 'column')
        )
    );
