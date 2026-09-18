-- Seed standard reference instruments from DEFAULT_TICK_UNIVERSE into gold-copy tenant
-- Applied by migration runner or operator db seed scripts.
-- Idempotent: ON CONFLICT (tenant_id, identifier_type, identifier_value) DO NOTHING.

DO $$
DECLARE
    gold_copy_tenant_id UUID;
BEGIN
    SELECT id INTO gold_copy_tenant_id FROM public.tenants WHERE gold_copy = true LIMIT 1;
    IF gold_copy_tenant_id IS NULL THEN
        -- Fallback to default platform gold-copy tenant if tenants table record not yet marked
        gold_copy_tenant_id := '99e99e99-99e9-49e9-89e9-99e99e99e999';
    END IF;

    INSERT INTO oms.security (
        id, tenant_id, security_name, identifier_type, identifier_value,
        subtype_code, ticker, isin, created_at, updated_at, valid_from, valid_to
    )
    VALUES
      (gen_random_uuid(), gold_copy_tenant_id, 'Apple Inc.', 'TICKER', 'AAPL', 'equity', 'AAPL', 'US0378331005', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Microsoft Corporation', 'TICKER', 'MSFT', 'equity', 'MSFT', 'US5949181045', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Alphabet Inc. Class A', 'TICKER', 'GOOGL', 'equity', 'GOOGL', 'US02079K3059', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Amazon.com Inc.', 'TICKER', 'AMZN', 'equity', 'AMZN', 'US0231351067', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'NVIDIA Corporation', 'TICKER', 'NVDA', 'equity', 'NVDA', 'US67066G1040', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Meta Platforms Inc.', 'TICKER', 'META', 'equity', 'META', 'US30303M1027', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Tesla Inc.', 'TICKER', 'TSLA', 'equity', 'TSLA', 'US88160R1014', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Berkshire Hathaway Inc. Class B', 'TICKER', 'BRK.B', 'equity', 'BRK.B', 'US0846707026', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'UnitedHealth Group Inc.', 'TICKER', 'UNH', 'equity', 'UNH', 'US91324P1021', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Johnson & Johnson', 'TICKER', 'JNJ', 'equity', 'JNJ', 'US4781601046', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'JPMorgan Chase & Co.', 'TICKER', 'JPM', 'equity', 'JPM', 'US46625H1005', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Visa Inc. Class A', 'TICKER', 'V', 'equity', 'V', 'US92826C8394', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Procter & Gamble Co.', 'TICKER', 'PG', 'equity', 'PG', 'US7427181091', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Exxon Mobil Corp.', 'TICKER', 'XOM', 'equity', 'XOM', 'US30231G1022', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Home Depot Inc.', 'TICKER', 'HD', 'equity', 'HD', 'US4370761029', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Chevron Corporation', 'TICKER', 'CVX', 'equity', 'CVX', 'US1667641005', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Mastercard Inc. Class A', 'TICKER', 'MA', 'equity', 'MA', 'US57636Q1040', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Bank of America Corp.', 'TICKER', 'BAC', 'equity', 'BAC', 'US0605051046', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'AbbVie Inc.', 'TICKER', 'ABBV', 'equity', 'ABBV', 'US00287Y1091', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Pfizer Inc.', 'TICKER', 'PFE', 'equity', 'PFE', 'US7170811035', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Broadcom Inc.', 'TICKER', 'AVGO', 'equity', 'AVGO', 'US11135F1012', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Costco Wholesale Corp.', 'TICKER', 'COST', 'equity', 'COST', 'US22160K1051', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Walt Disney Co.', 'TICKER', 'DIS', 'equity', 'DIS', 'US2546871060', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Coca-Cola Company', 'TICKER', 'KO', 'equity', 'KO', 'US1912161007', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'PepsiCo Inc.', 'TICKER', 'PEP', 'equity', 'PEP', 'US7134481081', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'US Treasury 10-Year Benchmark', 'TICKER', 'US10Y', 'sovereign_debt', 'US10Y', 'US91282CHE52', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'US Treasury 2-Year Benchmark', 'TICKER', 'US02Y', 'sovereign_debt', 'US02Y', 'US91282CDJ69', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Vanguard Total Bond Market ETF', 'TICKER', 'BND', 'equity', 'BND', 'US9219378356', NOW(), NOW(), NOW(), NULL),
      (gen_random_uuid(), gold_copy_tenant_id, 'Intel Corporation', 'TICKER', 'INTC', 'equity', 'INTC', 'US4581401001', NOW(), NOW(), NOW(), NULL)
    ON CONFLICT (tenant_id, identifier_type, identifier_value) DO NOTHING;
END $$;
