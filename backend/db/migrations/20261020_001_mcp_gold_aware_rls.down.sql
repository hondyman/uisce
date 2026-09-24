DROP POLICY IF EXISTS page_definitions_tenant_gold_policy ON public.page_definitions;
DROP POLICY IF EXISTS business_objects_tenant_gold_policy ON public.business_objects;
DROP POLICY IF EXISTS business_object_fields_tenant_policy ON public.business_object_fields;
DROP POLICY IF EXISTS catalog_edge_tenant_gold_policy ON public.catalog_edge;
DROP POLICY IF EXISTS mdm_exception_queue_tenant_policy ON mdm.universal_exception_queue;
DROP POLICY IF EXISTS schema_drift_proposals_tenant_policy ON catalog_drift.schema_drift_proposals;

DROP FUNCTION IF EXISTS uisce_get_gold_tenant();
-- uisce_get_current_tenant may be shared with 20261016_001 — leave in place.
