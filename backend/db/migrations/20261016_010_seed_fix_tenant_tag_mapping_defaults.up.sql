-- 20261016_010_seed_fix_tenant_tag_mapping_defaults.up.sql
-- Seeds default FIX 4.4 tag → semantic-field mappings under the gold-copy
-- tenant. These are the inheritance defaults every non-gold tenant sees
-- via the GSIFI read-inheritance RLS policy (migration 004).
--
-- Source: FIX 4.4 specification, the canonical protocol reference
-- (NOT crims_fix_schema.txt — that file is the CRIMS operational
-- Postgres schema dump, not the FIX protocol tag dictionary).
--
-- Scope: two message types only — NewOrderSingle (D) and ExecutionReport
-- (8). The data-pipeline tile (fix_tag_map, fix_order_emit) only
-- needs these to function for the order entry + execution flow; other
-- message types (OrderCancel F, OrderCancelReplace G, etc.) are left
-- for per-tenant addition. Adding them here would just bloat the seed
-- without serving any current call site.
--
-- Idempotent: every INSERT uses ON CONFLICT (tenant_id, fix_version,
-- msg_type, fix_tag) DO NOTHING — re-running this migration against
-- the same DB is a no-op. Per-tenant overrides always win via the
-- tile's "later entry overrides earlier" merge (see HANDOFF_FIX_OVER_
-- PIPELINE.md §9 fix_tag_map precedence rule).
--
-- Idempotency note: this migration depends on the gold-copy tenant
-- row existing. Migration 002a creates it. If 002a hasn't been applied
-- (e.g. someone cherry-picked this migration), the DO block below
-- no-ops with a NOTICE — gold-copy inheritance won't work but the
-- migration won't fail.

BEGIN;

DO $$
DECLARE
    gold_id UUID;
BEGIN
    SELECT id INTO gold_id
    FROM public.tenants
    WHERE gold_copy = true
    LIMIT 1;

    IF gold_id IS NULL THEN
        RAISE NOTICE 'fix_tenant_tag_mapping seed skipped: no gold_copy tenant found. Apply migration 002a first.';
        RETURN;
    END IF;

    -- SET LOCAL ROLE so the INSERT bypasses the strict RLS WITH CHECK
    -- on fix_tenant_tag_mapping. The uisce_gold_copy_sync role has
    -- BYPASSRLS (migration 002) and was granted to app_user there;
    -- SET LOCAL ROLE reverts on transaction end, same semantics as
    -- SET LOCAL <parameter>. This matches the existing pattern in
    -- internal/db/cross_tenant.go's WithGoldCopySync.
    --
    -- If the connecting role isn't a member of uisce_gold_copy_sync
    -- (e.g. an operator running the migration as a role that wasn't
    -- granted membership), this SET LOCAL ROLE will fail — that's
    -- intentional; the operator should run the migration as app_user
    -- (or another role granted membership) per migration 002.
    SET LOCAL ROLE uisce_gold_copy_sync;

    -- ─── MsgType D: NewOrderSingle ────────────────────────────────────
    -- The canonical outbound order entry tags. Each is the FIX 4.4
    -- tag number (numeric) mapped to a semantic_field name that the
    -- downstream pipeline tiles recognize.

    INSERT INTO fix_tenant_tag_mapping
        (id, tenant_id, fix_version, msg_type, fix_tag, semantic_field, required, transform_fn)
    VALUES
        (gen_random_uuid(), gold_id, 'FIX.4.4', 'D', 11, 'external_order_id', TRUE,  NULL),
        (gen_random_uuid(), gold_id, 'FIX.4.4', 'D',  1, 'account',          FALSE, NULL),
        (gen_random_uuid(), gold_id, 'FIX.4.4', 'D', 55, 'symbol',           TRUE,  'upper'),
        (gen_random_uuid(), gold_id, 'FIX.4.4', 'D', 48, 'isin',             FALSE, NULL),
        (gen_random_uuid(), gold_id, 'FIX.4.4', 'D', 54, 'side',             TRUE,  NULL),
        (gen_random_uuid(), gold_id, 'FIX.4.4', 'D', 38, 'quantity',         TRUE,  'numeric'),
        (gen_random_uuid(), gold_id, 'FIX.4.4', 'D', 44, 'price',            TRUE,  'numeric'),
        (gen_random_uuid(), gold_id, 'FIX.4.4', 'D', 40, 'order_type',       FALSE, NULL),
        (gen_random_uuid(), gold_id, 'FIX.4.4', 'D', 59, 'time_in_force',    FALSE, NULL)
    ON CONFLICT (tenant_id, fix_version, msg_type, fix_tag) DO NOTHING;

    -- ─── MsgType 8: ExecutionReport ────────────────────────────────────
    -- Inbound execution report tags. The exec_id field is critical for
    -- the dedup key (HANDOFF §14 inbound idempotency) — required=true.

    INSERT INTO fix_tenant_tag_mapping
        (id, tenant_id, fix_version, msg_type, fix_tag, semantic_field, required, transform_fn)
    VALUES
        (gen_random_uuid(), gold_id, 'FIX.4.4', '8', 11, 'external_order_id', TRUE,  NULL),
        (gen_random_uuid(), gold_id, 'FIX.4.4', '8', 37, 'order_id',          TRUE,  NULL),
        (gen_random_uuid(), gold_id, 'FIX.4.4', '8', 17, 'exec_id',           TRUE,  NULL),
        (gen_random_uuid(), gold_id, 'FIX.4.4', '8', 150,'exec_type',         TRUE,  NULL),
        (gen_random_uuid(), gold_id, 'FIX.4.4', '8', 39, 'ord_status',        TRUE,  NULL),
        (gen_random_uuid(), gold_id, 'FIX.4.4', '8', 55, 'symbol',            FALSE, 'upper'),
        (gen_random_uuid(), gold_id, 'FIX.4.4', '8', 54, 'side',              FALSE, NULL),
        (gen_random_uuid(), gold_id, 'FIX.4.4', '8', 38, 'quantity',          FALSE, 'numeric'),
        (gen_random_uuid(), gold_id, 'FIX.4.4', '8', 44, 'price',             FALSE, 'numeric'),
        (gen_random_uuid(), gold_id, 'FIX.4.4', '8', 32, 'last_quantity',     FALSE, 'numeric'),
        (gen_random_uuid(), gold_id, 'FIX.4.4', '8', 14, 'cum_quantity',      FALSE, 'numeric'),
        (gen_random_uuid(), gold_id, 'FIX.4.4', '8', 151,'leaves_quantity',   FALSE, 'numeric')
    ON CONFLICT (tenant_id, fix_version, msg_type, fix_tag) DO NOTHING;
END $$;

COMMIT;
