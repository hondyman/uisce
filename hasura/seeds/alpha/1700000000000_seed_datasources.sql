INSERT INTO public.alpha_datasource (id, datasource_name, datasource_code, datasource_type, is_active, config) VALUES
('550e8400-e29b-41d4-a716-446655440001', 'Postgres', 'postgres', 'relational', true, '{}'),
('550e8400-e29b-41d4-a716-446655440002', 'SQL Server', 'sql_server', 'relational', true, '{}'),
('550e8400-e29b-41d4-a716-446655440003', 'Oracle', 'oracle', 'relational', true, '{}'),
('550e8400-e29b-41d4-a716-446655440004', 'Snowflake', 'snowflake', 'warehouse', true, '{}'),
('550e8400-e29b-41d4-a716-446655440005', 'Iceberg', 'iceberg', 'lakehouse', true, '{}')
ON CONFLICT (id) DO UPDATE SET 
  datasource_name = EXCLUDED.datasource_name,
  datasource_code = EXCLUDED.datasource_code,
  datasource_type = EXCLUDED.datasource_type;
