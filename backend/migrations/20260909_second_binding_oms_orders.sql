-- Second binding for the Order BO: alpha.oms.orders, alongside the
-- canonical MAPS_TO binding (alpha.orm.order). Proves rule portability
-- across bindings - see docs/unified-rule-engine-handoff.md, "the
-- binding layer is load-bearing".
--
-- field_bindings/business_object_bindings exist in the schema but were
-- empty everywhere in the system before this - verified live before
-- building on them (see the handoff doc's account of that check). This
-- is their first real population: MAPS_TO stays the canonical/default
-- binding (unchanged, zero migration risk to what already works),
-- field_bindings becomes the table for every *additional* binding a BO
-- picks up from here.
--
-- Deliberately partial: only the 4 terms this proof's rule needs
-- (TargetQuantity) plus 3 more free 1:1 name matches (LimitPrice,
-- ExecutedQuantity, LeavesQuantity). A rule referencing any other
-- semantic term against this binding must fail loud (rule_error), not
-- silently fall back to the canonical binding's map - see
-- ResolveSemanticFieldMapForBinding's doc comment.
BEGIN;

DO $$
DECLARE
    v_bo_id UUID;
    v_binding_id UUID;
    v_target_qty_field UUID := 'bbe30460-ff51-46b6-ac02-fbd2a23ad806';
    v_limit_price_field UUID := 'df6ed1b9-ccd8-4837-bc8a-3969582ccb48';
    v_executed_qty_field UUID := 'b3980a40-52d1-466e-8f14-34edb4cb6005';
    v_leaves_qty_field UUID := '04ee37d2-9a2e-4457-b071-5dcef7984f2d';
    v_tenant_id UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    v_driving_node UUID;
BEGIN
    SELECT id INTO v_bo_id FROM business_objects WHERE bo_key = 'order' AND tenant_id = v_tenant_id;
    SELECT id INTO v_driving_node FROM catalog_node WHERE qualified_path = '/oms/orders/id' AND tenant_id = v_tenant_id LIMIT 1;
    IF v_driving_node IS NULL THEN
        SELECT id INTO v_driving_node FROM catalog_node WHERE qualified_path = '/orm/order/id' AND tenant_id = v_tenant_id LIMIT 1;
    END IF;

    INSERT INTO business_object_bindings (id, tenant_id, bo_id, backend_id, backend_type, driving_node_id, is_default)
    VALUES (gen_random_uuid(), v_tenant_id, v_bo_id, gen_random_uuid(), 'POSTGRES', v_driving_node, false)
    RETURNING id INTO v_binding_id;

    -- No tenant_id filter on catalog_node here, deliberately: the
    -- existing /oms/orders/* catalog_node scan turned out to belong to
    -- tenant 840750a5-... ("Soul Trader"), not this BO's own tenant
    -- (99e99e99-...) - a real, minor tenant-scoping wrinkle in how that
    -- earlier scan was run, discovered while writing this migration and
    -- left as-is rather than re-scanning. field_bindings.source_node_id
    -- has no tenant-consistency constraint (a straight FK to
    -- catalog_node.id), so this is legal, just cross-tenant - fine for
    -- proving resolution portability, worth tidying if this binding ever
    -- becomes more than a proof fixture.
    INSERT INTO field_bindings (id, tenant_id, bo_id, binding_id, field_id, source_node_id, source_type, binding_status)
    SELECT gen_random_uuid(), v_tenant_id, v_bo_id, v_binding_id, f.field_id, cn.id, 'COLUMN', 'RESOLVED'
    FROM (VALUES
        (v_target_qty_field, '/oms/orders/quantity'),
        (v_limit_price_field, '/oms/orders/limit_price'),
        (v_executed_qty_field, '/oms/orders/filled_qty'),
        (v_leaves_qty_field, '/oms/orders/leaves_qty')
    ) AS f(field_id, col_path)
    JOIN catalog_node cn ON cn.qualified_path = f.col_path;

    RAISE NOTICE 'Created binding % for BO % with % field_bindings', v_binding_id, v_bo_id,
        (SELECT count(*) FROM field_bindings WHERE binding_id = v_binding_id);
END $$;

COMMIT;
