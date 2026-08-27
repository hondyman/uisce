-- 20260824_007_seed_gold_copy_demo_data.up.sql
-- Seeds sample data into STI tables for the gold-copy tenant (99e99e99-99e9-49e9-89e9-99e99e99e999).
-- Focus: master.sales_ledger rows for rep-core-001 to render SUM/AVG aggregations.
-- Uses delete-then-insert pattern for idempotency.

DO $$
DECLARE
    gold_copy_tenant_id UUID := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    acc1_id UUID;
    acc2_id UUID;
    acc3_id UUID;
    acc4_id UUID;
    acc5_id UUID;
    acc6_id UUID;
    acc7_id UUID;
    acc8_id UUID;
BEGIN

-- ============================================================================
-- oms.account — minimal seed needed for FK references
-- ============================================================================
DELETE FROM oms.account WHERE tenant_id = gold_copy_tenant_id;

INSERT INTO oms.account (id, tenant_id, account_number, account_name, base_currency, status, subtype_code, fee_schedule_code, created_at, updated_at, valid_from, valid_to)
VALUES (gen_random_uuid(), gold_copy_tenant_id, 'ACC-001', 'BlackRock Institutional Master Trust', 'USD', 'ACTIVE', 'institutional', 'TIER_A_FLAT', NOW(), NOW(), NOW(), NULL)
RETURNING id INTO acc1_id;

INSERT INTO oms.account (id, tenant_id, account_number, account_name, base_currency, status, subtype_code, fee_schedule_code, created_at, updated_at, valid_from, valid_to)
VALUES (gen_random_uuid(), gold_copy_tenant_id, 'ACC-002', 'Vanguard Fiduciary Trust Co', 'USD', 'ACTIVE', 'institutional', 'TIER_B_TIERED', NOW(), NOW(), NOW(), NULL)
RETURNING id INTO acc2_id;

INSERT INTO oms.account (id, tenant_id, account_number, account_name, base_currency, status, subtype_code, fee_schedule_code, created_at, updated_at, valid_from, valid_to)
VALUES (gen_random_uuid(), gold_copy_tenant_id, 'ACC-003', 'State Street Qualified Plan', 'USD', 'ACTIVE', 'qualified_retirement', 'TIER_C_PERFORMANCE', NOW(), NOW(), NOW(), NULL)
RETURNING id INTO acc3_id;

INSERT INTO oms.account (id, tenant_id, account_number, account_name, base_currency, status, subtype_code, fee_schedule_code, created_at, updated_at, valid_from, valid_to)
VALUES (gen_random_uuid(), gold_copy_tenant_id, 'ACC-004', 'Goldman Sachs Family Office', 'USD', 'ACTIVE', 'trust_estate', 'TIER_B_TIERED', NOW(), NOW(), NOW(), NULL)
RETURNING id INTO acc4_id;

INSERT INTO oms.account (id, tenant_id, account_number, account_name, base_currency, status, subtype_code, fee_schedule_code, created_at, updated_at, valid_from, valid_to)
VALUES (gen_random_uuid(), gold_copy_tenant_id, 'ACC-005', 'MSDW Corporate Treasury', 'USD', 'ACTIVE', 'corporate_treasury', 'TIER_A_FLAT', NOW(), NOW(), NOW(), NULL)
RETURNING id INTO acc5_id;

INSERT INTO oms.account (id, tenant_id, account_number, account_name, base_currency, status, subtype_code, fee_schedule_code, created_at, updated_at, valid_from, valid_to)
VALUES (gen_random_uuid(), gold_copy_tenant_id, 'ACC-006', 'Fidelity Wealth Management', 'USD', 'ACTIVE', 'retail_wealth', 'TIER_B_TIERED', NOW(), NOW(), NOW(), NULL)
RETURNING id INTO acc6_id;

INSERT INTO oms.account (id, tenant_id, account_number, account_name, base_currency, status, subtype_code, fee_schedule_code, created_at, updated_at, valid_from, valid_to)
VALUES (gen_random_uuid(), gold_copy_tenant_id, 'ACC-007', 'Northern Trust SMA', 'USD', 'ACTIVE', 'sma', 'TIER_C_PERFORMANCE', NOW(), NOW(), NOW(), NULL)
RETURNING id INTO acc7_id;

INSERT INTO oms.account (id, tenant_id, account_number, account_name, base_currency, status, subtype_code, fee_schedule_code, created_at, updated_at, valid_from, valid_to)
VALUES (gen_random_uuid(), gold_copy_tenant_id, 'ACC-008', 'JPMorgan Institutional Trust', 'USD', 'ACTIVE', 'institutional', 'TIER_A_FLAT', NOW(), NOW(), NOW(), NULL)
RETURNING id INTO acc8_id;

-- ============================================================================
-- master.sales_ledger — THE critical table for rep-core-001
-- Contains aum_basis_amount (SUM), effective_fee_bps (AVG) for aggregation tests
-- invoice_status = fee_schedule_code grouping dimension
-- ============================================================================
DELETE FROM master.sales_ledger WHERE tenant_id = gold_copy_tenant_id;

-- aum_management_fee rows (rep-core-001 3rd binding)
INSERT INTO master.sales_ledger (id, tenant_id, invoice_number, client_id, subtype_code, billing_period_end, aum_basis_amount, effective_fee_bps, hwm_benchmark_nav, invoice_status, created_at, updated_at, valid_from, valid_to)
VALUES
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2024-Q1-001', acc1_id, 'aum_management_fee', '2024-03-31', 50000000.00, 85.5000, 52000000.00, 'TIER_A_FLAT', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2024-Q1-002', acc2_id, 'aum_management_fee', '2024-03-31', 25000000.00, 95.0000, 25500000.00, 'TIER_B_TIERED', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2024-Q1-003', acc3_id, 'aum_management_fee', '2024-03-31', 125000000.00, 75.0000, 128000000.00, 'TIER_A_FLAT', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2024-Q2-001', acc1_id, 'aum_management_fee', '2024-06-30', 52000000.00, 85.5000, 53000000.00, 'TIER_A_FLAT', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2024-Q2-002', acc4_id, 'aum_management_fee', '2024-06-30', 80000000.00, 90.0000, 81000000.00, 'TIER_C_PERFORMANCE', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2024-Q2-003', acc5_id, 'aum_management_fee', '2024-06-30', 35000000.00, 100.0000, 35500000.00, 'TIER_B_TIERED', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2024-Q3-001', acc2_id, 'aum_management_fee', '2024-09-30', 26000000.00, 95.0000, 26500000.00, 'TIER_B_TIERED', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2024-Q3-002', acc6_id, 'aum_management_fee', '2024-09-30', 200000000.00, 65.0000, 205000000.00, 'TIER_A_FLAT', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2024-Q3-003', acc3_id, 'aum_management_fee', '2024-09-30', 130000000.00, 75.0000, 132000000.00, 'TIER_A_FLAT', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2024-Q4-001', acc7_id, 'aum_management_fee', '2024-12-31', 45000000.00, 110.0000, 45500000.00, 'TIER_C_PERFORMANCE', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2024-Q4-002', acc4_id, 'aum_management_fee', '2024-12-31', 85000000.00, 90.0000, 86000000.00, 'TIER_B_TIERED', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2025-Q1-001', acc5_id, 'aum_management_fee', '2025-03-31', 38000000.00, 100.0000, 38500000.00, 'TIER_B_TIERED', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2025-Q1-002', acc8_id, 'aum_management_fee', '2025-03-31', 95000000.00, 80.0000, 96000000.00, 'TIER_A_FLAT', NOW(), NOW(), NOW(), NULL);

-- trading_commission rows
INSERT INTO master.sales_ledger (id, tenant_id, invoice_number, client_id, subtype_code, billing_period_end, aum_basis_amount, invoice_status, created_at, updated_at, valid_from, valid_to)
VALUES
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2025-Q1-003', acc1_id, 'trading_commission', '2025-03-31', 1200000.00, 'SETTLED', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2025-Q2-001', acc2_id, 'trading_commission', '2025-06-30', 980000.00, 'SETTLED', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2025-Q3-001', acc3_id, 'trading_commission', '2025-09-30', 2100000.00, 'SETTLED', NOW(), NOW(), NOW(), NULL);

-- performance_fee rows
INSERT INTO master.sales_ledger (id, tenant_id, invoice_number, client_id, subtype_code, billing_period_end, aum_basis_amount, invoice_status, created_at, updated_at, valid_from, valid_to)
VALUES
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2025-Q2-002', acc4_id, 'performance_fee', '2025-06-30', 5500000.00, 'PENDING', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2025-Q3-002', acc5_id, 'performance_fee', '2025-09-30', 3200000.00, 'APPROVED', NOW(), NOW(), NOW(), NULL),
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2025-Q4-001', acc6_id, 'performance_fee', '2025-12-31', 7800000.00, 'PENDING', NOW(), NOW(), NOW(), NULL);

-- platform_subscription rows
INSERT INTO master.sales_ledger (id, tenant_id, invoice_number, client_id, subtype_code, billing_period_end, aum_basis_amount, invoice_status, created_at, updated_at, valid_from, valid_to)
VALUES
  (gen_random_uuid(), gold_copy_tenant_id, 'INV-2025-Q3-003', acc7_id, 'platform_subscription', '2025-09-30', 180000.00, 'ACTIVE', NOW(), NOW(), NOW(), NULL);

END $$;
