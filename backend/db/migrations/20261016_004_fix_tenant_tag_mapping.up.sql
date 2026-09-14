-- 20261016_004_fix_tenant_tag_mapping.up.sql
-- Per-tenant FIX tag → semantic field mapping. Seeds from
-- /Users/eganpj/GitHub/uisce/crims_fix_schema.txt via a one-shot backfill
-- script (NOT in this migration — schema-tenant pairs vary by broker).
--
-- Application reads use the explicit GSIFI OR-clause so that Gold Copy
-- defaults are visible to every tenant, with tenant rows overriding per
-- (fix_version, msg_type, fix_tag). The precedence rule is:
--   ORDER BY (tenant_id = $1) DESC NULLS LAST
-- See HANDOFF_FIX_OVER_PIPELINE.md §9 `fix_tag_map` for the precedence rationale
-- (parenthesization is critical — AND binds tighter than OR, so without the
-- parens tenant rows would bypass fix_version/msg_type filters entirely).
--
-- RLS: same regime as 20261016_003. Tenant rows are scoped to the current
-- tenant via the helper function; gold-copy rows are bypass-readable for admin.

BEGIN;

CREATE TABLE IF NOT EXISTS fix_tenant_tag_mapping (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    fix_version TEXT NOT NULL,
    msg_type TEXT NOT NULL,             -- 'D' = NewOrderSingle, '8' = ExecutionReport, 'F' = OrderCancel, etc.
    fix_tag INT NOT NULL,                -- numeric FIX tag id (e.g. 11 = ClOrdID, 55 = Symbol)
    semantic_field TEXT NOT NULL,        -- logical name (e.g. 'external_order_id', 'symbol')
    required BOOLEAN NOT NULL DEFAULT FALSE,
    default_value TEXT,
    transform_fn TEXT,                  -- e.g. 'upper', 'numeric', 'parse_iso_currency'
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, fix_version, msg_type, fix_tag)
);

CREATE INDEX IF NOT EXISTS idx_fix_tag_mapping_lookup
    ON fix_tenant_tag_mapping (fix_version, msg_type, fix_tag);

ALTER TABLE fix_tenant_tag_mapping ENABLE ROW LEVEL SECURITY;
ALTER TABLE fix_tenant_tag_mapping FORCE ROW LEVEL SECURITY;

-- Same read-inheritance / write-isolation split as fix_tenant_config.
-- See migration 003 for the rationale. The fix_tag_map tile depends
-- on gold-copy rows being visible to non-gold tenant sessions so the
-- "tenant overrides gold copy" precedence rule (HANDOFF §9) works.
DROP POLICY IF EXISTS fix_tenant_tag_mapping_isolation_policy ON fix_tenant_tag_mapping;
CREATE POLICY fix_tenant_tag_mapping_isolation_policy ON fix_tenant_tag_mapping
    FOR ALL
    USING (
        tenant_id = uisce_get_current_tenant()
        OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
    )
    WITH CHECK (
        tenant_id = uisce_get_current_tenant()
    );

-- Grant INSERT/UPDATE/DELETE to the gold-copy sync role so the
-- seed migration (010) can populate default mappings. Without this,
-- the SET LOCAL ROLE in 010 bypasses RLS but the role still lacks
-- the table-level privilege and the INSERT fails with "permission
-- denied". The role itself was created in 002; this is the
-- per-table grant that goes alongside.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'uisce_gold_copy_sync') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON fix_tenant_tag_mapping TO uisce_gold_copy_sync;
    END IF;
END $$;

COMMIT;
