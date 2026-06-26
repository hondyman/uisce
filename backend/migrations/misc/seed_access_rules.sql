-- Seed sample access rules and default roles per tenant personas
-- Note: adjust bo_id values to match actual business_object ids in your environment.

-- Tenant and datasource placeholders
\set tenant_id '00000000-0000-0000-0000-000000000000'

-- Sample Business Object IDs (replace with real IDs)
-- For demonstration we assume these BOs exist:
--   portfolio, position, client
\set bo_portfolio 'bo:portfolio'
\set bo_position  'bo:position'
\set bo_client    'bo:client'

-- Default LDAP groups per tenant persona
--   cn=tenant_admins        : full control (WRITE, no filters)
--   cn=tenant_developers    : WRITE with mask on PII
--   cn=tenant_business_users: READ with regional row filter + mask
--   cn=tenant_operators     : READ with narrower filter, hide sensitive terms

-- Admins: full write, no masks or filters
INSERT INTO access_rule (rule_id, tenant_id, bo_id, group_dn, row_filter_dsl, column_masks, access_level, status, created_by)
VALUES
  ('ar-admin-portfolio', :tenant_id, :bo_portfolio, 'cn=tenant_admins,ou=groups,dc=example,dc=com', NULL, '[]', 'WRITE', 'APPROVED', 'system'),
  ('ar-admin-position',  :tenant_id, :bo_position,  'cn=tenant_admins,ou=groups,dc=example,dc=com', NULL, '[]', 'WRITE', 'APPROVED', 'system'),
  ('ar-admin-client',    :tenant_id, :bo_client,    'cn=tenant_admins,ou=groups,dc=example,dc=com', NULL, '[]', 'WRITE', 'APPROVED', 'system')
ON CONFLICT (rule_id) DO NOTHING;

-- Developers: write, mask PII
INSERT INTO access_rule (rule_id, tenant_id, bo_id, group_dn, row_filter_dsl, column_masks, access_level, status, created_by)
VALUES
  ('ar-dev-portfolio', :tenant_id, :bo_portfolio, 'cn=tenant_developers,ou=groups,dc=example,dc=com', NULL,
   '[{"term":"client_ssn","mask_type":"HIDE"},{"term":"client_email","mask_type":"MASK"}]', 'WRITE', 'APPROVED', 'system'),
  ('ar-dev-position',  :tenant_id, :bo_position,  'cn=tenant_developers,ou=groups,dc=example,dc=com', NULL,
   '[{"term":"client_ssn","mask_type":"HIDE"},{"term":"client_email","mask_type":"MASK"}]', 'WRITE', 'APPROVED', 'system'),
  ('ar-dev-client',    :tenant_id, :bo_client,    'cn=tenant_developers,ou=groups,dc=example,dc=com', NULL,
   '[{"term":"client_ssn","mask_type":"HIDE"},{"term":"client_email","mask_type":"MASK"}]', 'WRITE', 'APPROVED', 'system')
ON CONFLICT (rule_id) DO NOTHING;

-- Business users: read, regional filter, mask PII
INSERT INTO access_rule (rule_id, tenant_id, bo_id, group_dn, row_filter_dsl, column_masks, access_level, status, created_by)
VALUES
  ('ar-biz-portfolio', :tenant_id, :bo_portfolio, 'cn=tenant_business_users,ou=groups,dc=example,dc=com',
   "(region = 'EMEA' OR region = 'APAC') AND client_type != 'internal'",
   '[{"term":"client_ssn","mask_type":"HIDE"},{"term":"client_email","mask_type":"MASK"}]', 'READ', 'APPROVED', 'system'),
  ('ar-biz-position',  :tenant_id, :bo_position,  'cn=tenant_business_users,ou=groups,dc=example,dc=com',
   "(region = 'EMEA' OR region = 'APAC') AND client_type != 'internal'",
   '[{"term":"client_ssn","mask_type":"HIDE"},{"term":"client_email","mask_type":"MASK"}]', 'READ', 'APPROVED', 'system'),
  ('ar-biz-client',    :tenant_id, :bo_client,    'cn=tenant_business_users,ou=groups,dc=example,dc=com',
   "(region = 'EMEA' OR region = 'APAC') AND client_type != 'internal'",
   '[{"term":"client_ssn","mask_type":"HIDE"},{"term":"client_email","mask_type":"MASK"}]', 'READ', 'APPROVED', 'system')
ON CONFLICT (rule_id) DO NOTHING;

-- Operators: read, narrow filter, hide sensitive terms entirely
INSERT INTO access_rule (rule_id, tenant_id, bo_id, group_dn, row_filter_dsl, column_masks, access_level, status, created_by)
VALUES
  ('ar-ops-portfolio', :tenant_id, :bo_portfolio, 'cn=tenant_operators,ou=groups,dc=example,dc=com',
   "(region = 'NA') AND client_tier = 'STANDARD'",
   '[{"term":"client_ssn","mask_type":"HIDE"},{"term":"client_email","mask_type":"HIDE"}]', 'READ', 'APPROVED', 'system'),
  ('ar-ops-position',  :tenant_id, :bo_position,  'cn=tenant_operators,ou=groups,dc=example,dc=com',
   "(region = 'NA') AND client_tier = 'STANDARD'",
   '[{"term":"client_ssn","mask_type":"HIDE"},{"term":"client_email","mask_type":"HIDE"}]', 'READ', 'APPROVED', 'system'),
  ('ar-ops-client',    :tenant_id, :bo_client,    'cn=tenant_operators,ou=groups,dc=example,dc=com',
   "(region = 'NA') AND client_tier = 'STANDARD'",
   '[{"term":"client_ssn","mask_type":"HIDE"},{"term":"client_email","mask_type":"HIDE"}]', 'READ', 'APPROVED', 'system')
ON CONFLICT (rule_id) DO NOTHING;
