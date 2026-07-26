-- Gold Copy Catalog Registration & Default Survivorship Rules
-- Tenant: 99e99e99-99e9-49e9-89e9-99e99e99e999 (Master Core Gold Copy)

BEGIN;

-- 1. Register MDM Business Objects in catalog_node (Rule 2 Graph-First)
INSERT INTO catalog_node (node_id, tenant_id, node_key, node_name, node_type, term_type, data_type, is_active)
VALUES
  ('60000000-0000-0000-0000-000000000001', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'security_master', 'Security Master', 'BUSINESS_OBJECT', 'ENTITY', 'object', true),
  ('60000000-0000-0000-0000-000000000002', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'pricing_master', 'Pricing Master', 'BUSINESS_OBJECT', 'ENTITY', 'object', true),
  ('60000000-0000-0000-0000-000000000003', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'account_master', 'Account Master', 'BUSINESS_OBJECT', 'ENTITY', 'object', true)
ON CONFLICT (node_id) DO NOTHING;

-- 2. Seed Default Gold Copy Survivorship Matrix Rules (Rule 1 Config-Before-Code)
INSERT INTO mdm.survivorship_rule (tenant_id, bo_type, field_name, vendor_priority_json, tolerance_pct)
VALUES
  ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'SECURITY_MASTER', 'coupon_rate', '{"BLOOMBERG": 100, "REFINITIV": 80, "IDC": 60}', 5.00),
  ('99e99e99-99e9-49e9-89e9-99e99e99e999', 'PRICING_MASTER', 'closing_price', '{"BLOOMBERG": 100, "IDC": 90, "REFINITIV": 80}', 10.00)
ON CONFLICT (tenant_id, bo_type, field_name) DO NOTHING;

COMMIT;
