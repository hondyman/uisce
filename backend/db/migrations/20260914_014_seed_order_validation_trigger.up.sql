-- Seed a sample validation trigger for the Order BO. This is a post-commit
-- trigger that fires a webhook action when a row_insert lands on the Order
-- BO with status='NEW'. Pure proof that the trigger engine is wired up
-- end-to-end: BO write -> emitBORowEvent -> trigger_engine.EvaluateTriggers
-- -> this trigger's conditions match -> webhook fires.
--
-- Why post-commit and not pre-commit? The trigger engine is by design
-- post-commit: it can fire notifications/workflows/webhooks but cannot
-- block a write. The actual write-blocking enforcement is the Postgres
-- CHECK constraint chk_order_target_qty_positive on orm.order. Cardinal
-- Rule 6 (Semantic/OLTP Boundary) keeps graph/identity in the catalog
-- and mutable state in OLTP - this trigger lives in the catalog-side
-- metadata because it's a post-write event spec, not a financial state.
--
-- The conditions array is a JSON array of {field, operator, value} objects
-- evaluated with AND semantics against the BO write payload. Empty
-- conditions = always pass.
--
-- The action is type='webhook' pointing at a localhost URL that doesn't
-- exist - it will fire and fail with a logged error, which is the right
-- behavior for a smoke-test trigger (proves the engine ran without
-- risking a real outbound call). Swap webhook_url for a real endpoint
-- when the rule graduates from "demonstration" to "production."

INSERT INTO public.validation_triggers (
    id,
    tenant_id,
    trigger_type,
    target_entity,
    step_name,
    rule_ids,
    meta,
    is_active,
    created_at,
    updated_at
) VALUES (
    gen_random_uuid(),
    (SELECT id FROM public.tenants WHERE name = 'northwind'),
    'row_insert',
    'order',
    'order_inserted_new_status',
    '{}',
    jsonb_build_object(
        'conditions', jsonb_build_array(
            jsonb_build_object(
                'field', 'status',
                'operator', 'equals',
                'value', 'NEW'
            )
        ),
        'actions', jsonb_build_array(
            jsonb_build_object(
                'type', 'webhook',
                'webhook_url', 'http://localhost:9999/uisce/order-events'
            )
        )
    ),
    true,
    NOW(),
    NOW()
);

-- Verify the seed landed.
DO $$
DECLARE
    row_count INT;
BEGIN
    SELECT COUNT(*) INTO row_count
    FROM public.validation_triggers vt, public.tenants t
    WHERE vt.tenant_id = t.id
      AND t.name = 'northwind'
      AND vt.target_entity = 'order'
      AND vt.trigger_type = 'row_insert'
      AND vt.step_name = 'order_inserted_new_status';

    IF row_count = 0 THEN
        RAISE EXCEPTION 'Seed failed: order_inserted_new_status trigger not found';
    END IF;

    RAISE NOTICE 'Seeded order_inserted_new_status trigger (northwind tenant, order BO)';
END $$;
