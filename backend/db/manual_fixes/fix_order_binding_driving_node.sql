-- Repoint the Order BO's binding at the /orm/order TABLE node instead of
-- the /orm/order/id COLUMN node it was wrongly wired to.
--
-- Why: business_object_binding.driving_node_id for order's only binding
-- (90cd2335-aa78-482b-b503-7d2b9c5f2545) resolves to catalog_node
-- 34e8ab17-... ("id", qualified_path /orm/order/id) -- a column, not a
-- table. Every other binding in the tenant points at a table-level node
-- (verified: this is the only binding whose driving node's qualified_path
-- has more than 2 path segments). The practical symptom: BusinessObjectDetailsPage
-- shows the binding's "driving table" as "id" instead of "order", and the
-- Validations tab's per-binding rule lookup (ValidationsAndTriggersTab,
-- which treats a second binding's nodeName as another BO's bo_name) tried
-- to also fetch rules for a nonexistent bo_name "id".
--
-- The correct table node (6b267260-a5fa-549b-a80a-7ddb1c712da0,
-- qualified_path /orm/order) already exists; this only repoints the FK,
-- it does not create or delete any catalog_node.
--
-- Usage:
--   psql "$DATABASE_URL" -v mode=plan  -f fix_order_binding_driving_node.sql
--   psql "$DATABASE_URL" -v mode=apply -f fix_order_binding_driving_node.sql
\if :{?mode}
\else
  \set mode 'plan'
\endif

\set ON_ERROR_STOP on

\echo '== mode:' :mode

BEGIN;

SELECT (:'mode' = 'apply') AS is_apply \gset

DO $$
BEGIN
  IF (SELECT count(*) FROM public.catalog_node WHERE id = '6b267260-a5fa-549b-a80a-7ddb1c712da0' AND qualified_path = '/orm/order') = 0 THEN
    RAISE EXCEPTION 'ABORT: target table node 6b267260-a5fa-549b-a80a-7ddb1c712da0 (/orm/order) not found';
  END IF;
  IF (SELECT driving_node_id FROM public.business_object_binding WHERE bo_binding_id = '90cd2335-aa78-482b-b503-7d2b9c5f2545') <> '34e8ab17-d71b-5ef6-9334-c2032c860fef' THEN
    RAISE EXCEPTION 'ABORT: order binding no longer points at the expected column node -- re-check before proceeding (already fixed?)';
  END IF;
END $$;

\echo '-- before --'
SELECT b.bo_binding_id, cn.node_name, cn.qualified_path
FROM public.business_object_binding b
JOIN public.catalog_node cn ON cn.id = b.driving_node_id
WHERE b.bo_binding_id = '90cd2335-aa78-482b-b503-7d2b9c5f2545';

UPDATE public.business_object_binding
SET driving_node_id = '6b267260-a5fa-549b-a80a-7ddb1c712da0'
WHERE bo_binding_id = '90cd2335-aa78-482b-b503-7d2b9c5f2545';

\echo '-- after --'
SELECT b.bo_binding_id, cn.node_name, cn.qualified_path
FROM public.business_object_binding b
JOIN public.catalog_node cn ON cn.id = b.driving_node_id
WHERE b.bo_binding_id = '90cd2335-aa78-482b-b503-7d2b9c5f2545';

\if :is_apply
  \echo '== APPLY: committing'
  COMMIT;
\else
  \echo '== PLAN: rolling back (no changes made)'
  ROLLBACK;
\endif
