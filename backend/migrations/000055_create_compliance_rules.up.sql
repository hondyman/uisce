DO $$
BEGIN
    IF EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'compliance_rules') 
       AND NOT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'workflow_compliance_rules') THEN
        ALTER TABLE public.compliance_rules RENAME TO workflow_compliance_rules;
    END IF;
END $$;

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

CREATE INDEX IF NOT EXISTS idx_compliance_rules_type ON compliance_rules(rule_type);
