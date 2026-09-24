-- 20260824_010_seed_altinv_alternative_investment.up.sql
-- Seeds altinv.alternative_investment for gold-copy tenant

DO $$
DECLARE
    gold_copy_tenant_id UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
BEGIN

    INSERT INTO altinv.alternative_investment (id, tenant_id, subtype_code, investment_name, sponsor_name, asset_class, committed_capital, called_capital, unfunded_commitment, vintage_year, dpi, rvpi, status, created_at, updated_at, valid_from, valid_to)
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, 'private_equity', 'Apollo Fund VIII', 'Apollo Global', 'Private Equity', 100000000.00, 75000000.00, 25000000.00, 2018, 0.27, 0.85, 'ACTIVE', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'private_equity', 'KKR North America XI', 'KKR', 'Private Equity', 75000000.00, 60000000.00, 15000000.00, 2019, 0.25, 0.92, 'ACTIVE', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'private_equity', 'Carlyle Asia V', 'Carlyle', 'Private Equity', 50000000.00, 35000000.00, 15000000.00, 2020, 0.23, 0.78, 'ACTIVE', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'venture_capital', 'Sequoia Fund XII', 'Sequoia', 'Venture Capital', 25000000.00, 20000000.00, 5000000.00, 2021, 0.00, 1.15, 'ACTIVE', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'venture_capital', 'A16Z Bio Fund III', 'Andreessen Horowitz', 'Venture Capital', 30000000.00, 22000000.00, 8000000.00, 2022, 0.00, 0.95, 'ACTIVE', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'hedge_fund', 'Bridgewater All Weather', 'Bridgewater', 'Hedge Fund', 40000000.00, 40000000.00, 0.00, 2017, 0.12, 1.05, 'ACTIVE', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'hedge_fund', 'Two Sigma Absolute Return', 'Two Sigma', 'Hedge Fund', 30000000.00, 30000000.00, 0.00, 2018, 0.10, 1.02, 'ACTIVE', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'real_estate', 'Blackstone Real Estate IX', 'Blackstone', 'Real Estate', 60000000.00, 48000000.00, 12000000.00, 2019, 0.20, 1.10, 'ACTIVE', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'infrastructure', 'Macquarie Infrastructure Fund V', 'Macquarie', 'Infrastructure', 35000000.00, 28000000.00, 7000000.00, 2021, 0.08, 1.08, 'ACTIVE', NOW(), NOW(), NOW(), NULL);

END $$;
