CREATE TABLE IF NOT EXISTS swift_compliance_rule_set (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    rule_id UUID NOT NULL,
    rule_category TEXT NOT NULL DEFAULT 'SWIFT_SETTLEMENT',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    severity_threshold TEXT NOT NULL DEFAULT 'WARNING',
    UNIQUE (tenant_id, rule_id)
);

ALTER TABLE swift_compliance_rule_set ENABLE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS swift_compliance_rule_set_isolation ON swift_compliance_rule_set;

CREATE POLICY swift_compliance_rule_set_isolation ON swift_compliance_rule_set
    USING (
        tenant_id = NULLIF(current_setting('app.tenant_id', 't'), '')::uuid
        OR tenant_id = (SELECT id FROM public.tenants WHERE gold_copy = true LIMIT 1)
    );

COMMENT ON TABLE swift_compliance_rule_set IS 'Associates compliance rules with SWIFT settlement processing per tenant. Gold-copy rules apply to all tenants unless overridden.';
