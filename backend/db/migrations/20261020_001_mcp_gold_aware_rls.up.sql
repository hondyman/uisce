-- MCP / Page Studio / BO gold-aware RLS (post gold-copy-widen).
-- Policies encode: tenant_id = uisce.current_tenant OR gold visibility.
-- Pages: gold rows also require is_core (column on page_definitions — not a caller flag).
-- FORCE so table owners cannot bypass.
--
-- Choke point: db.WithTenantGoldTransaction / ApplyTenantGUCs sets
--   SET LOCAL uisce.current_tenant, app.tenant_id, and optionally uisce.gold_tenant.

CREATE OR REPLACE FUNCTION uisce_get_current_tenant() RETURNS uuid AS $$
BEGIN
    RETURN NULLIF(current_setting('uisce.current_tenant', true), '')::uuid;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION uisce_get_gold_tenant() RETURNS uuid AS $$
BEGIN
    RETURN NULLIF(current_setting('uisce.gold_tenant', true), '')::uuid;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$ LANGUAGE plpgsql STABLE;

-- page_definitions ----------------------------------------------------------
ALTER TABLE IF EXISTS public.page_definitions ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS public.page_definitions FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS page_definitions_tenant_gold_policy ON public.page_definitions;
CREATE POLICY page_definitions_tenant_gold_policy ON public.page_definitions
    FOR ALL
    USING (
        tenant_id = uisce_get_current_tenant()
        OR (is_core = true AND tenant_id = uisce_get_gold_tenant())
    )
    WITH CHECK (
        tenant_id = uisce_get_current_tenant()
    );

-- business_objects ----------------------------------------------------------
ALTER TABLE IF EXISTS public.business_objects ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS public.business_objects FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS business_objects_tenant_gold_policy ON public.business_objects;
CREATE POLICY business_objects_tenant_gold_policy ON public.business_objects
    FOR ALL
    USING (
        tenant_id = uisce_get_current_tenant()
        OR tenant_id = uisce_get_gold_tenant()
    )
    WITH CHECK (
        tenant_id = uisce_get_current_tenant()
    );

-- business_object_fields ----------------------------------------------------
ALTER TABLE IF EXISTS public.business_object_fields ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS public.business_object_fields FORCE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS business_object_fields_tenant_policy ON public.business_object_fields;
CREATE POLICY business_object_fields_tenant_policy ON public.business_object_fields
    FOR ALL
    USING (
        tenant_id = uisce_get_current_tenant()
        OR tenant_id = uisce_get_gold_tenant()
    )
    WITH CHECK (
        tenant_id = uisce_get_current_tenant()
    );

-- catalog_edge (tenant_id may be text) --------------------------------------
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'catalog_edge' AND column_name = 'tenant_id'
    ) THEN
        ALTER TABLE public.catalog_edge ENABLE ROW LEVEL SECURITY;
        ALTER TABLE public.catalog_edge FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS catalog_edge_tenant_gold_policy ON public.catalog_edge;
        EXECUTE $p$
            CREATE POLICY catalog_edge_tenant_gold_policy ON public.catalog_edge
                FOR ALL
                USING (
                    tenant_id::text = uisce_get_current_tenant()::text
                    OR tenant_id::text = uisce_get_gold_tenant()::text
                )
                WITH CHECK (
                    tenant_id::text = uisce_get_current_tenant()::text
                )
        $p$;
    END IF;
END $$;

-- MDM exception queue -------------------------------------------------------
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'mdm' AND table_name = 'universal_exception_queue'
    ) THEN
        ALTER TABLE mdm.universal_exception_queue ENABLE ROW LEVEL SECURITY;
        ALTER TABLE mdm.universal_exception_queue FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS mdm_exception_queue_tenant_policy ON mdm.universal_exception_queue;
        CREATE POLICY mdm_exception_queue_tenant_policy ON mdm.universal_exception_queue
            FOR ALL
            USING (tenant_id = uisce_get_current_tenant())
            WITH CHECK (tenant_id = uisce_get_current_tenant());
    END IF;
END $$;

-- schema drift proposals ----------------------------------------------------
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.tables
        WHERE table_schema = 'catalog_drift' AND table_name = 'schema_drift_proposals'
    ) THEN
        ALTER TABLE catalog_drift.schema_drift_proposals ENABLE ROW LEVEL SECURITY;
        ALTER TABLE catalog_drift.schema_drift_proposals FORCE ROW LEVEL SECURITY;
        DROP POLICY IF EXISTS schema_drift_proposals_tenant_policy ON catalog_drift.schema_drift_proposals;
        CREATE POLICY schema_drift_proposals_tenant_policy ON catalog_drift.schema_drift_proposals
            FOR ALL
            USING (tenant_id = uisce_get_current_tenant())
            WITH CHECK (tenant_id = uisce_get_current_tenant());
    END IF;
END $$;
