-- 20261024_005_catalog_node_read_gold_copy_rules.up.sql
--
-- Rules authored in the gold-copy tenant are core: every tenant inherits them read-only
-- (docs/mdm-rules.md). catalog_node has forced RLS with a single current-tenant-only policy, so a
-- regular tenant could not read the gold-copy tenant's validation_rule nodes and would be held to none
-- of the core rules.
--
-- One SELECT-only policy for exactly that node type in the gold-copy tenant. Every other gold-copy
-- catalog node stays invisible to tenants as before, and writes remain governed by
-- tenant_isolation_policy (unchanged), so no tenant can edit or delete a core rule. Policies are
-- permissive, so this ORs with the existing one for reads.
--
-- Idempotent: safe if the policy was created by hand before this migration ran.

DROP POLICY IF EXISTS catalog_node_read_gold_copy_rules ON public.catalog_node;
CREATE POLICY catalog_node_read_gold_copy_rules ON public.catalog_node
    FOR SELECT
    USING (
        tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
        AND node_type_id = (SELECT id FROM public.catalog_node_type WHERE catalog_type_name = 'validation_rule')
    );
