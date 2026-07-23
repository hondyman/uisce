-- Fix missing tables for Semantic Bundles API

CREATE TABLE IF NOT EXISTS public.dax_functions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_domain TEXT NOT NULL,
    name TEXT NOT NULL,
    class TEXT,
    badge TEXT,
    description TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS public.metrics_registry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    schema_domain TEXT NOT NULL,
    node_id TEXT NOT NULL,
    category TEXT,
    description TEXT,
    formula_type TEXT,
    formula TEXT,
    arguments JSONB DEFAULT '[]'::jsonb,
    badge TEXT,
    function_class TEXT,
    functions_used TEXT[],
    governance_status TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Seed initial data for 'banking' domain to verify fix
INSERT INTO public.dax_functions (schema_domain, name, class, badge, description)
VALUES 
('banking', 'CalculateInterest', 'financial', 'core', 'Calculates compound interest for a given period'),
('banking', 'RiskScore', 'risk', 'beta', 'Computes credit risk score based on transaction history')
ON CONFLICT DO NOTHING;

INSERT INTO public.metrics_registry (schema_domain, node_id, category, description, formula_type, formula, arguments, badge, function_class, functions_used, governance_status)
VALUES
('banking', 'net_interest_margin', 'profitability', 'Net Interest Margin', 'ratio', 'Interest Income - Interest Expense / Average Earning Assets', '[]'::jsonb, 'gold', 'financial', ARRAY['CalculateInterest'], 'approved')
ON CONFLICT DO NOTHING;
