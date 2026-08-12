-- Complete Setup Script for CRIMS ORM Schema Business Objects, Connections, Semantic Terms, Relationships, Calculated Fields, and Validations

BEGIN;

-- 1. Ensure Connections & Alpha Datasource exist for CRIMS ORM in Postgres DB
-- Connection
INSERT INTO connections (
  id, host, port, database, username, password, schema, created_at, updated_at
) VALUES (
  'a0e4e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e6',
  '100.84.50.65',
  5432,
  'crims',
  'postgres',
  'postgres',
  'orm',
  NOW(),
  NOW()
) ON CONFLICT (id) DO UPDATE SET updated_at = NOW();

-- Alpha Datasource
INSERT INTO alpha_datasource (
  id, datasource_code, name, description, created_at
) VALUES (
  'b0e4e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e6',
  'crims_orm',
  'CRIMS Front Office ORM',
  'CRIMS Front Office order management and trading system relational model (ORM schema)',
  NOW()
) ON CONFLICT (id) DO NOTHING;

-- Tenant Product Datasource Mapping
INSERT INTO tenant_product_datasource (
  id, tenant_product_id, alpha_datasource_id, connection_id, source_name, is_active, created_at, updated_at
) VALUES (
  'e0e4e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e6',
  'd0e4e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e5',
  'b0e4e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e6',
  'a0e4e5e5-e5e5-e5e5-e5e5-e5e5e5e5e5e6',
  'crims_orm_datasource',
  true,
  NOW(),
  NOW()
) ON CONFLICT (id) DO NOTHING;

COMMIT;
