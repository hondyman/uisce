-- 20261016_007_fix_compliance_rule_set.up.sql
-- Per-tenant binding of FIX message flows to compliance rules. The actual
-- rule definitions live in the rules.* tables (managed by the existing
-- RuleEngine); this table is just a join + activation flag + severity
-- threshold.
--
-- The fix_compliance tile (HANDOFF_FIX_OVER_PIPELINE.md §9) reads this table
-- to know which rules to invoke for a given tenant's FIX flows. Multiple
-- rule sets can be active per tenant (e.g. pre-trade + post-trade).
--
-- RLS: same regime. Tenant-scoped via uisce_get_current_tenant().


CREATE TABLE IF NOT EXISTS fix_compliance_rule_set (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    rule_id UUID NOT NULL,               -- FK to rules.* in application code (no FK here — schema varies)
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    severity_threshold TEXT NOT NULL DEFAULT 'WARNING',  -- INFORMATIONAL | WARNING | HARD_BLOCK
    applies_to_msg_types TEXT[] NOT NULL DEFAULT ARRAY['D', 'F', 'G'],  -- NewOrderSingle, OrderCancel, OrderCancelReplace
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_modified_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, rule_id)
);

CREATE INDEX IF NOT EXISTS idx_fix_compliance_rule_set_active
    ON fix_compliance_rule_set (tenant_id, is_active);

ALTER TABLE fix_compliance_rule_set ENABLE ROW LEVEL SECURITY;
ALTER TABLE fix_compliance_rule_set FORCE ROW LEVEL SECURITY;

-- Same read-inheritance / write-isolation split as 003/004. A tenant
-- inheriting compliance rule sets from Gold Copy is the inheritance
-- path the fix_compliance tile relies on; with strict RLS only, that
-- inheritance silently fails.
DROP POLICY IF EXISTS fix_compliance_rule_set_isolation_policy ON fix_compliance_rule_set;
CREATE POLICY fix_compliance_rule_set_isolation_policy ON fix_compliance_rule_set
    FOR ALL
    USING (
        tenant_id = uisce_get_current_tenant()
        OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
    )
    WITH CHECK (
        tenant_id = uisce_get_current_tenant()
    );

