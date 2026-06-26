-- Sample Data for Investment Management LLM Testing
-- Run this after all migrations are complete

-- Create demo tenant
INSERT INTO tenants (id, name, created_at) VALUES 
('demo-tenant-123', 'Demo Investment Firm', NOW())
ON CONFLICT (id) DO NOTHING;

-- Create Holding node type if not exists
INSERT INTO catalog_node_type (id, name, description, icon, color, properties_schema)
VALUES (
    uuid_generate_v4(),
    'Holding',
    'Portfolio holding/position',
    'briefcase',
    '#4CAF50',
    '{
        "type": "object",
        "properties": {
            "ticker": {"type": "string"},
            "quantity": {"type": "string"},
            "currency": {"type": "string"},
            "portfolio_id": {"type": "string"}
        }
    }'::jsonb
)
ON CONFLICT (name) DO NOTHING;

-- Create sample portfolio holdings
INSERT INTO catalog_node (
    id, node_name, qualified_path, node_type_id, tenant_id, properties, description, created_at
)
SELECT
    uuid_generate_v4(),
    'Microsoft Equity',
    'demo.holdings.msft',
    (SELECT id FROM catalog_node_type WHERE name = 'Holding'),
    'demo-tenant-123',
    '{"ticker": "MSFT", "quantity": "10000", "currency": "USD", "portfolio_id": "demo-portfolio"}'::jsonb,
    'Microsoft Corporation equity holding',
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM catalog_node WHERE qualified_path = 'demo.holdings.msft')
UNION ALL
SELECT
    uuid_generate_v4(),
    'Apple Equity',
    'demo.holdings.aapl',
    (SELECT id FROM catalog_node_type WHERE name = 'Holding'),
    'demo-tenant-123',
    '{"ticker": "AAPL", "quantity": "5000", "currency": "USD", "portfolio_id": "demo-portfolio"}'::jsonb,
    'Apple Inc equity holding',
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM catalog_node WHERE qualified_path = 'demo.holdings.aapl')
UNION ALL
SELECT
    uuid_generate_v4(),
    'Euro Bond',
    'demo.holdings.euro_bond',
    (SELECT id FROM catalog_node_type WHERE name = 'Holding'),
    'demo-tenant-123',
    '{"ticker": "EURO_BOND", "quantity": "5000", "currency": "EUR", "portfolio_id": "demo-portfolio"}'::jsonb,
    'European corporate bond holding',
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM catalog_node WHERE qualified_path = 'demo.holdings.euro_bond');

-- Create NAV calculation node
INSERT INTO catalog_node (
    id, node_name, qualified_path, node_type_id, tenant_id, properties, description,
    lineage, data_quality_contract, sla, created_at
)
SELECT
    uuid_generate_v4(),
    'Portfolio NAV',
    'demo.metrics.nav',
    (SELECT id FROM catalog_node_type WHERE name = 'Calculation'),
    'demo-tenant-123',
    '{"calculation_type": "NAV", "portfolio_id": "demo-portfolio"}'::jsonb,
    'Net Asset Value calculation for demo portfolio',
    '["demo.holdings.msft", "demo.holdings.aapl", "demo.holdings.euro_bond"]'::jsonb,
    '{
        "freshness_sla_hours": 3,
        "null_rate_threshold": 0.05,
        "completeness_target": 0.95
    }'::jsonb,
    '{
        "availability_target": 0.999,
        "update_frequency": "daily",
        "calculation_window": "EOD"
    }'::jsonb,
    NOW()
WHERE NOT EXISTS (SELECT 1 FROM catalog_node WHERE qualified_path = 'demo.metrics.nav');

-- Create edges: NAV depends on holdings
INSERT INTO catalog_edge (source_node_id, target_node_id, relationship_type, tenant_id, created_at)
SELECT
    (SELECT id FROM catalog_node WHERE qualified_path = 'demo.metrics.nav'),
    (SELECT id FROM catalog_node WHERE qualified_path = 'demo.holdings.msft'),
    'depends_on',
    'demo-tenant-123',
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM catalog_edge 
    WHERE source_node_id = (SELECT id FROM catalog_node WHERE qualified_path = 'demo.metrics.nav')
    AND target_node_id = (SELECT id FROM catalog_node WHERE qualified_path = 'demo.holdings.msft')
)
UNION ALL
SELECT
    (SELECT id FROM catalog_node WHERE qualified_path = 'demo.metrics.nav'),
    (SELECT id FROM catalog_node WHERE qualified_path = 'demo.holdings.aapl'),
    'depends_on',
    'demo-tenant-123',
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM catalog_edge 
    WHERE source_node_id = (SELECT id FROM catalog_node WHERE qualified_path = 'demo.metrics.nav')
    AND target_node_id = (SELECT id FROM catalog_node WHERE qualified_path = 'demo.holdings.aapl')
)
UNION ALL
SELECT
    (SELECT id FROM catalog_node WHERE qualified_path = 'demo.metrics.nav'),
    (SELECT id FROM catalog_node WHERE qualified_path = 'demo.holdings.euro_bond'),
    'depends_on',
    'demo-tenant-123',
    NOW()
WHERE NOT EXISTS (
    SELECT 1 FROM catalog_edge 
    WHERE source_node_id = (SELECT id FROM catalog_node WHERE qualified_path = 'demo.metrics.nav')
    AND target_node_id = (SELECT id FROM catalog_node WHERE qualified_path = 'demo.holdings.euro_bond')
);

-- Verify data
SELECT 'Holdings created:' as status, COUNT(*) as count 
FROM catalog_node 
WHERE tenant_id = 'demo-tenant-123' 
AND node_type_id = (SELECT id FROM catalog_node_type WHERE name = 'Holding');

SELECT 'NAV calculation created:' as status, COUNT(*) as count
FROM catalog_node 
WHERE qualified_path = 'demo.metrics.nav';

SELECT 'Edges created:' as status, COUNT(*) as count
FROM catalog_edge 
WHERE tenant_id = 'demo-tenant-123';
