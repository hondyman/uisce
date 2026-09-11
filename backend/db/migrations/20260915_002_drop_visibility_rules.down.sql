CREATE TABLE IF EXISTS visibility_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    rule_name TEXT NOT NULL,
    target_entity TEXT NOT NULL,
    target_id UUID NOT NULL,
    expression TEXT NOT NULL,
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_to TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT visibility_rules_tenant_fkey FOREIGN KEY (tenant_id) REFERENCES master.tenant(id)
);

CREATE INDEX visibility_rules_tenant_idx ON visibility_rules(tenant_id);
CREATE INDEX visibility_rules_target_idx ON visibility_rules(target_entity, target_id);
