-- 20261024_002_bob_copy_plural_bindings.up.sql
--
-- Step 2 (data only). The plural rows are the only binding for the ORM core BOs (account, broker,
-- order, position, security). Register their backend in physical_backend if missing (the singular
-- table has an FK to it), copy them across keeping their ids, and give every BO with exactly one
-- binding that binding as its default. backend_type is not carried over: it is derivable from
-- physical_backend.dialect_name. Idempotent; skips the copy once the plural table is gone.
--
-- Split into four migrations because the runner executes each file as ONE transaction and
-- business_object_binding has a deferred FK (driving_node_id): rows inserted or updated in a
-- transaction leave pending trigger events, and CREATE INDEX / ALTER TABLE on that table in the same
-- transaction then fail with "pending trigger events". So DDL and data changes are separate files:
--   001 add is_default + one-default index   (DDL only)
--   002 register backends, copy plural rows, backfill defaults   (data only)
--   003 repoint field_bindings / relationship_bindings FKs   (DDL only)
--   004 gold-copy read policy   (DDL only)

DO $$
BEGIN
    IF to_regclass('public.business_object_bindings') IS NOT NULL THEN
        -- Register the backends the plural rows point at.
        INSERT INTO public.physical_backend
            (backend_id, backend_name, description, storage_tier, dialect_name, driver_class, is_system)
        SELECT DISTINCT p.backend_id,
               COALESCE(tpd.source_name, 'backend ' || p.backend_id::text),
               'Registered by bo binding consolidation',
               'oltp', lower(p.backend_type), '*sql.DB', false
        FROM public.business_object_bindings p
        LEFT JOIN public.tenant_product_datasource tpd ON tpd.id = p.backend_id
        ON CONFLICT (backend_id) DO NOTHING;

        -- Copy the bindings, keeping their ids.
        INSERT INTO public.business_object_binding
            (bo_binding_id, tenant_id, bo_id, backend_id, driving_node_id, base_sql, temporal_override,
             is_core, is_active, is_default, binding_name, created_at, updated_at)
        SELECT p.id, p.tenant_id, p.bo_id, p.backend_id, p.driving_node_id, p.base_sql, p.temporal_override,
               true, true, p.is_default,
               COALESCE(bo.bo_name, 'Business Object') || ' Binding', p.created_at, p.updated_at
        FROM public.business_object_bindings p
        LEFT JOIN public.business_objects bo ON bo.id = p.bo_id
        ON CONFLICT DO NOTHING;
    END IF;
END
$$;

-- A BO with exactly one binding has that binding as its default.
UPDATE public.business_object_binding b
SET is_default = true
WHERE NOT b.is_default
  AND (SELECT count(*) FROM public.business_object_binding x
       WHERE x.tenant_id = b.tenant_id AND x.bo_id = b.bo_id) = 1;
