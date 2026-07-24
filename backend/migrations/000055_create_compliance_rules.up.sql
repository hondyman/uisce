ALTER TABLE IF EXISTS compliance_rules RENAME TO workflow_compliance_rules;

CREATE TABLE IF NOT EXISTS compliance_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    rule_type VARCHAR(50) NOT NULL, -- 'drift', 'concentration', 'min_trade', etc.
    expression TEXT NOT NULL,       -- CEL expression
    severity VARCHAR(20) NOT NULL DEFAULT 'warning', -- 'warning', 'error'
    enabled BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_compliance_rules_type ON compliance_rules(rule_type);
