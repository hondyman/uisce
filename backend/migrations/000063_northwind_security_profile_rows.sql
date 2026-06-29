-- +goose Up
-- Migration: Seed Northwind Gold Copy Security Profiles
-- Created: 2026-06-29
--
-- Companion to 000062_abac_security_profiles.sql.
-- 000062 created the security.security_profiles table and seeded four
-- public.abac_policies (Sales Rep allow+deny, Inventory Manager allow+deny)
-- referencing role strings like "northwind_sales_rep" and
-- "northwind_inventory_manager".
--
-- This migration inserts the corresponding rows into security.security_profiles
-- so that:
--   * The Gold Copy profile catalogue is queryable (admin UI, enrichment pipeline).
--   * Tenant overrides can be created via parent_profile_id FK.
--   * Two additional Northwind profiles (Billing Specialist, Commerce Executive)
--     are added that 000062 did not yet cover.
--
-- All rows have tenant_id = NULL, marking them as System-Level Gold Copy Blueprints.
-- New tenants inherit these automatically; existing tenants are unaffected.

INSERT INTO security.security_profiles
    (profile_id, tenant_id, profile_key, profile_name)
VALUES
    -- Already referenced by 000062 seed policies:
    (gen_random_uuid(), NULL, 'northwind_sales_rep',
     'Gold Copy - Sales Representative'),
    (gen_random_uuid(), NULL, 'northwind_inventory_manager',
     'Gold Copy - Inventory Specialist'),

    -- New profiles (policies to be added in a later migration):
    (gen_random_uuid(), NULL, 'northwind_billing_specialist',
     'Gold Copy - Billing Specialist'),
    (gen_random_uuid(), NULL, 'northwind_executive',
     'Gold Copy - Commerce Executive')
ON CONFLICT (tenant_id, profile_key) DO NOTHING;

-- +goose Down
-- Removing Gold Copy profile rows. Tenant overrides (tenant_id IS NOT NULL)
-- are preserved so client customizations survive the rollback of the seed.
DELETE FROM security.security_profiles
 WHERE tenant_id IS NULL
   AND profile_key IN (
       'northwind_sales_rep',
       'northwind_inventory_manager',
       'northwind_billing_specialist',
       'northwind_executive'
   );