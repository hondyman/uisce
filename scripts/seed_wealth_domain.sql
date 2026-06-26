-- Seeder for Wealth Domain Semantic Terms

-- Get or Create System Tenant
WITH system_tenant AS (
    INSERT INTO tenants (id, name, display_name, created_at, updated_at)
    VALUES ('00000000-0000-0000-0000-000000000001', 'System Tenant', 'System Tenant', NOW(), NOW())
    ON CONFLICT (id) DO UPDATE SET updated_at = NOW()
    RETURNING id
),
semantic_type AS (
    INSERT INTO catalog_node_type (id, tenant_id, catalog_type_name, created_at, updated_at)
    SELECT '820b942a-9c9e-4abc-acdc-84616db33098', id, 'semantic_term', NOW(), NOW()
    FROM system_tenant
    ON CONFLICT (id) DO UPDATE SET updated_at = NOW()
    RETURNING id
)
INSERT INTO catalog_node (id, tenant_id, node_name, description, node_type_id, properties, qualified_path, created_at, updated_at)
SELECT 
    gen_random_uuid(),
    (SELECT id FROM system_tenant),
    term.node_name,
    term.description,
    (SELECT id FROM semantic_type),
    term.properties::jsonb,
    'semantic_term/' || term.node_name,
    NOW(),
    NOW()
FROM (VALUES 
    ('client.id', 'Unique client identifier', '{"type": "physical", "data_type": "string", "physical_mapping": {"table": "clients", "column": "id"}}'),
    ('client.name', 'Client full name', '{"type": "physical", "data_type": "string", "physical_mapping": {"table": "clients", "column": "name"}}'),
    ('client.risk_score', 'Current client risk score', '{"type": "physical", "data_type": "number", "physical_mapping": {"table": "clients", "column": "risk_score"}}'),
    ('client.target_risk_score', 'Target risk score from IPS', '{"type": "physical", "data_type": "number", "physical_mapping": {"table": "clients", "column": "target_risk_score"}}'),
    ('client.risk_drift', 'Difference between current and target risk score', '{"type": "calculated", "data_type": "number", "expression": "client.risk_score - client.target_risk_score"}'),
    ('client.risk_bucket', 'Risk bucket derived from risk score', '{"type": "calculated", "data_type": "string", "expression": "if(client.risk_score >= 70, ''High'', if(client.risk_score >= 40, ''Medium'', ''Low''))"}'),
    ('client.primary_advisor', 'Primary advisor for the client', '{"type": "relationship", "data_type": "json", "relationship": {"target_bo": "Advisor", "join_expression": "clients.primary_advisor_id = advisors.id"}}'),
    ('account.id', 'Account identifier', '{"type": "physical", "data_type": "string", "physical_mapping": {"table": "accounts", "column": "id"}}'),
    ('account.market_value', 'Current market value', '{"type": "physical", "data_type": "number", "physical_mapping": {"table": "accounts", "column": "market_value"}}'),
    ('account.is_taxable', 'Whether the account is taxable', '{"type": "physical", "data_type": "boolean", "physical_mapping": {"table": "accounts", "column": "is_taxable"}}'),
    ('client.total_assets', 'Sum of market value', '{"type": "calculated", "data_type": "number", "expression": "sum(account.market_value)"}'),
    ('client.account_count', 'Number of accounts', '{"type": "calculated", "data_type": "number", "expression": "count(account.id)"}')
) AS term(node_name, description, properties)
WHERE NOT EXISTS (
    SELECT 1 FROM catalog_node 
    WHERE node_name = term.node_name 
    AND tenant_id = (SELECT id FROM system_tenant)
);
