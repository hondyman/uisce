-- ============================================================================
-- EXPRESSION RULES SCHEMA
-- Starlark-based calculated fields, validations, and condition rules
-- ============================================================================

CREATE TABLE IF NOT EXISTS expression_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    business_object_id UUID REFERENCES business_objects(id) ON DELETE CASCADE,
    field_key TEXT, -- For calculated fields
    
    rule_type TEXT NOT NULL CHECK (rule_type IN ('validation', 'calculation', 'condition')),
    name TEXT NOT NULL,
    description TEXT,
    script TEXT NOT NULL, -- Starlark code
    
    -- Metadata
    is_active BOOLEAN DEFAULT true,
    version INTEGER DEFAULT 1,
    severity TEXT DEFAULT 'error', -- For validations: error, warning, info
    
    -- Audit
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT,
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_expr_tenant ON expression_rules(tenant_id);
CREATE INDEX IF NOT EXISTS idx_expr_bo ON expression_rules(business_object_id);
CREATE INDEX IF NOT EXISTS idx_expr_type ON expression_rules(rule_type);
CREATE INDEX IF NOT EXISTS idx_expr_active ON expression_rules(is_active);
CREATE INDEX IF NOT EXISTS idx_expr_field ON expression_rules(field_key);

-- ============================================================================
-- EXPRESSION HISTORY (Version Control)
-- ============================================================================

CREATE TABLE IF NOT EXISTS expression_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expression_id UUID NOT NULL REFERENCES expression_rules(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    script TEXT NOT NULL,
    changed_by TEXT,
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    change_reason TEXT
);

CREATE INDEX IF NOT EXISTS idx_expr_history ON expression_history(expression_id, version DESC);

-- ============================================================================
-- EXPRESSION TEST RESULTS
-- ============================================================================

CREATE TABLE IF NOT EXISTS expression_test_runs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    expression_id UUID NOT NULL REFERENCES expression_rules(id) ON DELETE CASCADE,
    input_data JSONB NOT NULL,
    output_result JSONB,
    is_success BOOLEAN,
    error_message TEXT,
    execution_time_ms INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    created_by TEXT
);

CREATE INDEX IF NOT EXISTS idx_expr_tests ON expression_test_runs(expression_id, created_at DESC);

-- ============================================================================
-- SEED SAMPLE EXPRESSIONS
-- ============================================================================

-- Validation: Amount limit
INSERT INTO expression_rules (tenant_id, rule_type, name, description, script, severity)
VALUES (
    (SELECT id FROM tenants WHERE name = 'Default Tenant' LIMIT 1),
    'validation',
    'Transaction Amount Limit',
    'Validates that transaction amount does not exceed configured limit',
    'result = (amount <= context.limit, "Amount exceeds limit of $" + str(context.limit)) if amount > context.limit else (True, None)',
    'error'
) ON CONFLICT DO NOTHING;

-- Calculation: Net worth
INSERT INTO expression_rules (tenant_id, rule_type, name, description, script, field_key)
VALUES (
    (SELECT id FROM tenants WHERE name = 'Default Tenant' LIMIT 1),
    'calculation',
    'Calculate Net Worth',
    'Calculates net worth from assets and liabilities',
    'result = context.assets - context.liabilities',
    'net_worth'
) ON CONFLICT DO NOTHING;

-- Calculation: Portfolio return
INSERT INTO expression_rules (tenant_id, rule_type, name, description, script, field_key)
VALUES (
    (SELECT id FROM tenants WHERE name = 'Default Tenant' LIMIT 1),
    'calculation',
    'Portfolio Return Percentage',
    'Calculates portfolio return as percentage',
    'result = ((context.current_value - context.cost_basis) / context.cost_basis * 100) if context.cost_basis > 0 else 0',
    'return_pct'
) ON CONFLICT DO NOTHING;

-- Condition: Approval routing
INSERT INTO expression_rules (tenant_id, rule_type, name, description, script)
VALUES (
    (SELECT id FROM tenants WHERE name = 'Default Tenant' LIMIT 1),
    'condition',
    'Approval Routing Rule',
    'Determines approval path based on amount',
    E'if amount > 1000000:\n    result = "committee_review"\nelif amount > 100000:\n    result = "manager_approval"\nelse:\n    result = "auto_approve"'
) ON CONFLICT DO NOTHING;

-- Validation: Required fields
INSERT INTO expression_rules (tenant_id, rule_type, name, description, script, severity)
VALUES (
    (SELECT id FROM tenants WHERE name = 'Default Tenant' LIMIT 1),
    'validation',
    'Required Portfolio Name',
    'Ensures portfolio has a name',
    'result = (len(context.name) > 0, "Portfolio name is required") if not context.name else (True, None)',
    'warning'
) ON CONFLICT DO NOTHING;

-- ============================================================================
-- SUCCESS MESSAGE
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '✓ Expression rules schema created';
    RAISE NOTICE '✓ Expression history (versioning) created';
    RAISE NOTICE '✓ Expression test runs table created';
    RAISE NOTICE '✓ 5 sample expressions seeded';
END $$;
