-- ============================================================================
-- ADDEPAR-STYLE FINANCIAL BUSINESS OBJECTS
-- Core financial data model for wealth management platform
-- ============================================================================

-- ============================================================================
-- 1. PORTFOLIO BUSINESS OBJECT
-- ============================================================================

INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, created_at)
VALUES (
  gen_random_uuid(),
  (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1),
  'portfolio',
  'Portfolio',
  'Portfolio',
  'portfolio',
  'Investment portfolio grouping accounts and positions',
  'briefcase',
  true,
  now()
) ON CONFLICT DO NOTHING;

INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence, created_at, created_by, last_modified_at)
SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_objects WHERE key = 'portfolio'), key, label, label, key, type, true, required, '', seq, now(), NULL, now()
FROM (VALUES
  ('id', 'Portfolio ID', 'uuid', 1, true),
  ('name', 'Portfolio Name', 'string', 2, true),
  ('code', 'Portfolio Code', 'string', 3, true),
  ('type', 'Portfolio Type', 'picklist', 4, true),  -- Individual, Trust, Foundation, Family Office
  ('strategy', 'Investment Strategy', 'picklist', 5, false),  -- Growth, Income, Balanced, Conservative
  ('benchmark_id', 'Benchmark', 'reference', 6, false),
  ('inception_date', 'Inception Date', 'date', 7, true),
  ('currency', 'Base Currency', 'currency_code', 8, true),
  ('market_value', 'Market Value', 'currency', 9, false),
  ('cost_basis', 'Cost Basis', 'currency', 10, false),
  ('unrealized_gain_loss', 'Unrealized Gain/Loss', 'currency', 11, false),
  ('day_change', 'Day Change', 'currency', 12, false),
  ('day_change_percent', 'Day Change %', 'percentage', 13, false),
  ('ytd_return', 'YTD Return', 'percentage', 14, false),
  ('inception_return', 'Since Inception Return', 'percentage', 15, false),
  ('risk_profile', 'Risk Profile', 'picklist', 16, false),  -- Conservative, Moderate, Aggressive
  ('target_allocation', 'Target Allocation', 'json', 17, false),
  ('rebalance_frequency', 'Rebalance Frequency', 'picklist', 18, false),  -- Monthly, Quarterly, Annually
  ('last_rebalance_date', 'Last Rebalance Date', 'date', 19, false),
  ('advisor_id', 'Primary Advisor', 'reference', 20, false),
  ('custodian', 'Custodian', 'picklist', 21, false),
  ('status', 'Status', 'picklist', 22, true),  -- Active, Closed, Pending
  ('created_at', 'Created At', 'datetime', 23, false),
  ('updated_at', 'Updated At', 'datetime', 24, false)
) AS t(key, label, type, seq, required)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- 2. SECURITY BUSINESS OBJECT (Instruments)
-- ============================================================================

INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, created_at)
VALUES (
  gen_random_uuid(),
  (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1),
  'security',
  'Security',
  'Security',
  'security',
  'Financial instruments (stocks, bonds, funds, alternatives)',
  'trending-up',
  true,
  now()
) ON CONFLICT DO NOTHING;

INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence, created_at, created_by, last_modified_at)
SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_objects WHERE key = 'security'), key, label, label, key, type, true, required, '', seq, now(), NULL, now()
FROM (VALUES
  ('id', 'Security ID', 'uuid', 1, true),
  ('ticker', 'Ticker Symbol', 'string', 2, true),
  ('cusip', 'CUSIP', 'string', 3, false),
  ('isin', 'ISIN', 'string', 4, false),
  ('sedol', 'SEDOL', 'string', 5, false),
  ('name', 'Security Name', 'string', 6, true),
  ('type', 'Security Type', 'picklist', 7, true),  -- Equity, Fixed Income, Alternative, Cash, Real Estate, Private Equity
  ('subtype', 'Security Subtype', 'picklist', 8, false),  -- Common Stock, Preferred, Corporate Bond, Municipal, ETF, Mutual Fund
  ('asset_class', 'Asset Class', 'picklist', 9, true),
  ('sector', 'Sector', 'picklist', 10, false),
  ('industry', 'Industry', 'picklist', 11, false),
  ('exchange', 'Exchange', 'string', 12, false),
  ('currency', 'Trading Currency', 'currency_code', 13, true),
  ('country', 'Country', 'country_code', 14, false),
  ('price', 'Current Price', 'decimal', 15, false),
  ('price_date', 'Price Date', 'date', 16, false),
  ('previous_close', 'Previous Close', 'decimal', 17, false),
  ('day_change', 'Day Change', 'decimal', 18, false),
  ('day_change_percent', 'Day Change %', 'percentage', 19, false),
  ('fifty_two_week_high', '52 Week High', 'decimal', 20, false),
  ('fifty_two_week_low', '52 Week Low', 'decimal', 21, false),
  ('dividend_yield', 'Dividend Yield', 'percentage', 22, false),
  ('pe_ratio', 'P/E Ratio', 'decimal', 23, false),
  ('market_cap', 'Market Cap', 'currency', 24, false),
  ('esg_score', 'ESG Score', 'decimal', 25, false),
  ('maturity_date', 'Maturity Date', 'date', 26, false),  -- For bonds
  ('coupon_rate', 'Coupon Rate', 'percentage', 27, false),  -- For bonds
  ('credit_rating', 'Credit Rating', 'string', 28, false),  -- For bonds
  ('expense_ratio', 'Expense Ratio', 'percentage', 29, false),  -- For funds
  ('nav', 'NAV', 'decimal', 30, false),  -- For funds
  ('is_active', 'Is Active', 'boolean', 31, true),
  ('created_at', 'Created At', 'datetime', 32, false),
  ('updated_at', 'Updated At', 'datetime', 33, false)
) AS t(key, label, type, seq, required)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- 3. POSITION BUSINESS OBJECT
-- ============================================================================

INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, created_at)
VALUES (
  gen_random_uuid(),
  (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1),
  'position',
  'Position',
  'Position',
  'position',
  'Holdings of securities within accounts/portfolios',
  'layers',
  true,
  now()
) ON CONFLICT DO NOTHING;

INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence, created_at, created_by, last_modified_at)
SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_objects WHERE key = 'position'), key, label, label, key, type, true, required, '', seq, now(), NULL, now()
FROM (VALUES
  ('id', 'Position ID', 'uuid', 1, true),
  ('account_id', 'Account', 'reference', 2, true),
  ('security_id', 'Security', 'reference', 3, true),
  ('quantity', 'Quantity', 'decimal', 4, true),
  ('cost_basis', 'Cost Basis', 'currency', 5, false),
  ('average_cost', 'Average Cost', 'currency', 6, false),
  ('market_value', 'Market Value', 'currency', 7, false),
  ('unrealized_gain_loss', 'Unrealized Gain/Loss', 'currency', 8, false),
  ('unrealized_gain_loss_percent', 'Unrealized %', 'percentage', 9, false),
  ('day_change', 'Day Change', 'currency', 10, false),
  ('day_change_percent', 'Day Change %', 'percentage', 11, false),
  ('weight', 'Weight %', 'percentage', 12, false),
  ('acquired_date', 'Acquired Date', 'date', 13, false),
  ('lot_method', 'Lot Method', 'picklist', 14, false),  -- FIFO, LIFO, SpecID, HIFO
  ('currency', 'Currency', 'currency_code', 15, true),
  ('income_ytd', 'Income YTD', 'currency', 16, false),
  ('is_accrued', 'Accrued Position', 'boolean', 17, false),
  ('as_of_date', 'As Of Date', 'date', 18, true),
  ('created_at', 'Created At', 'datetime', 19, false),
  ('updated_at', 'Updated At', 'datetime', 20, false)
) AS t(key, label, type, seq, required)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- 4. TAX LOT BUSINESS OBJECT
-- ============================================================================

INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, created_at)
VALUES (
  gen_random_uuid(),
  (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1),
  'taxlot',
  'Tax Lot',
  'Tax Lot',
  'taxlot',
  'Individual tax lots for cost basis tracking',
  'receipt',
  true,
  now()
) ON CONFLICT DO NOTHING;

INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence, created_at, created_by, last_modified_at)
SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_objects WHERE key = 'taxlot'), key, label, label, key, type, true, required, '', seq, now(), NULL, now()
FROM (VALUES
  ('id', 'Lot ID', 'uuid', 1, true),
  ('position_id', 'Position', 'reference', 2, true),
  ('acquisition_date', 'Acquisition Date', 'date', 3, true),
  ('quantity', 'Quantity', 'decimal', 4, true),
  ('original_quantity', 'Original Quantity', 'decimal', 5, true),
  ('cost_per_share', 'Cost Per Share', 'currency', 6, true),
  ('total_cost', 'Total Cost', 'currency', 7, true),
  ('market_value', 'Market Value', 'currency', 8, false),
  ('unrealized_gain_loss', 'Unrealized Gain/Loss', 'currency', 9, false),
  ('holding_period', 'Holding Period', 'picklist', 10, false),  -- Short Term, Long Term
  ('wash_sale_disallowed', 'Wash Sale Disallowed', 'currency', 11, false),
  ('adjusted_cost', 'Adjusted Cost Basis', 'currency', 12, false),
  ('is_covered', 'Covered Security', 'boolean', 13, false),
  ('source_transaction_id', 'Source Transaction', 'reference', 14, false),
  ('created_at', 'Created At', 'datetime', 15, false)
) AS t(key, label, type, seq, required)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- 5. BENCHMARK BUSINESS OBJECT
-- ============================================================================

INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, created_at)
VALUES (
  gen_random_uuid(),
  (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1),
  'benchmark',
  'Benchmark',
  'Benchmark',
  'benchmark',
  'Performance benchmarks for comparison',
  'target',
  true,
  now()
) ON CONFLICT DO NOTHING;

INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence, created_at, created_by, last_modified_at)
SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_objects WHERE key = 'benchmark'), key, label, label, key, type, true, required, '', seq, now(), NULL, now()
FROM (VALUES
  ('id', 'Benchmark ID', 'uuid', 1, true),
  ('name', 'Benchmark Name', 'string', 2, true),
  ('code', 'Benchmark Code', 'string', 3, true),
  ('type', 'Benchmark Type', 'picklist', 4, true),  -- Index, Blended, Custom
  ('description', 'Description', 'text', 5, false),
  ('currency', 'Currency', 'currency_code', 6, true),
  ('components', 'Components', 'json', 7, false),  -- For blended benchmarks
  ('mtd_return', 'MTD Return', 'percentage', 8, false),
  ('qtd_return', 'QTD Return', 'percentage', 9, false),
  ('ytd_return', 'YTD Return', 'percentage', 10, false),
  ('one_year_return', '1 Year Return', 'percentage', 11, false),
  ('three_year_return', '3 Year Return', 'percentage', 12, false),
  ('five_year_return', '5 Year Return', 'percentage', 13, false),
  ('inception_return', 'Since Inception', 'percentage', 14, false),
  ('volatility', 'Volatility', 'percentage', 15, false),
  ('sharpe_ratio', 'Sharpe Ratio', 'decimal', 16, false),
  ('is_active', 'Is Active', 'boolean', 17, true),
  ('created_at', 'Created At', 'datetime', 18, false)
) AS t(key, label, type, seq, required)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- 6. PERFORMANCE BUSINESS OBJECT
-- ============================================================================

INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, created_at)
VALUES (
  gen_random_uuid(),
  (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1),
  'performance',
  'Performance',
  'Performance',
  'performance',
  'Portfolio performance calculations',
  'chart-line',
  true,
  now()
) ON CONFLICT DO NOTHING;

INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence, created_at, created_by, last_modified_at)
SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_objects WHERE key = 'performance'), key, label, label, key, type, true, required, '', seq, now(), NULL, now()
FROM (VALUES
  ('id', 'Performance ID', 'uuid', 1, true),
  ('portfolio_id', 'Portfolio', 'reference', 2, true),
  ('as_of_date', 'As Of Date', 'date', 3, true),
  ('period', 'Period', 'picklist', 4, true),  -- MTD, QTD, YTD, 1Y, 3Y, 5Y, 10Y, ITD
  ('return_twr', 'TWR Return', 'percentage', 5, false),
  ('return_mwr', 'MWR Return', 'percentage', 6, false),
  ('benchmark_return', 'Benchmark Return', 'percentage', 7, false),
  ('alpha', 'Alpha', 'percentage', 8, false),
  ('beta', 'Beta', 'decimal', 9, false),
  ('sharpe_ratio', 'Sharpe Ratio', 'decimal', 10, false),
  ('sortino_ratio', 'Sortino Ratio', 'decimal', 11, false),
  ('max_drawdown', 'Max Drawdown', 'percentage', 12, false),
  ('volatility', 'Volatility', 'percentage', 13, false),
  ('tracking_error', 'Tracking Error', 'percentage', 14, false),
  ('information_ratio', 'Information Ratio', 'decimal', 15, false),
  ('upside_capture', 'Upside Capture', 'percentage', 16, false),
  ('downside_capture', 'Downside Capture', 'percentage', 17, false),
  ('beginning_value', 'Beginning Value', 'currency', 18, false),
  ('ending_value', 'Ending Value', 'currency', 19, false),
  ('net_contributions', 'Net Contributions', 'currency', 20, false),
  ('income', 'Income', 'currency', 21, false),
  ('realized_gain_loss', 'Realized Gain/Loss', 'currency', 22, false),
  ('unrealized_gain_loss', 'Unrealized Gain/Loss', 'currency', 23, false),
  ('fees', 'Fees', 'currency', 24, false),
  ('created_at', 'Created At', 'datetime', 25, false)
) AS t(key, label, type, seq, required)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- 7. ALLOCATION BUSINESS OBJECT
-- ============================================================================

INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, created_at)
VALUES (
  gen_random_uuid(),
  (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1),
  'allocation',
  'Allocation',
  'Asset Allocation',
  'allocation',
  'Asset and sector allocation analysis',
  'pie-chart',
  true,
  now()
) ON CONFLICT DO NOTHING;

INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence, created_at, created_by, last_modified_at)
SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_objects WHERE key = 'allocation'), key, label, label, key, type, true, required, '', seq, now(), NULL, now()
FROM (VALUES
  ('id', 'Allocation ID', 'uuid', 1, true),
  ('portfolio_id', 'Portfolio', 'reference', 2, true),
  ('as_of_date', 'As Of Date', 'date', 3, true),
  ('dimension', 'Dimension', 'picklist', 4, true),  -- Asset Class, Sector, Geography, Security Type
  ('category', 'Category', 'string', 5, true),
  ('market_value', 'Market Value', 'currency', 6, true),
  ('weight', 'Weight %', 'percentage', 7, true),
  ('target_weight', 'Target Weight %', 'percentage', 8, false),
  ('drift', 'Drift', 'percentage', 9, false),
  ('contribution', 'Return Contribution', 'percentage', 10, false),
  ('created_at', 'Created At', 'datetime', 11, false)
) AS t(key, label, type, seq, required)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- 8. FEE BUSINESS OBJECT
-- ============================================================================

INSERT INTO business_objects (id, tenant_id, key, name, display_name, technical_name, description, icon, is_core, created_at)
VALUES (
  gen_random_uuid(),
  (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1),
  'fee',
  'Fee',
  'Fee',
  'fee',
  'Management and advisory fees',
  'dollar-sign',
  true,
  now()
) ON CONFLICT DO NOTHING;

INSERT INTO bo_fields (id, tenant_id, business_object_id, key, name, display_name, technical_name, type, is_core, is_required, description, sequence, created_at, created_by, last_modified_at)
SELECT gen_random_uuid(), (SELECT id FROM public.tenants WHERE code = 'default-tenant' LIMIT 1), (SELECT id FROM business_objects WHERE key = 'fee'), key, label, label, key, type, true, required, '', seq, now(), NULL, now()
FROM (VALUES
  ('id', 'Fee ID', 'uuid', 1, true),
  ('account_id', 'Account', 'reference', 2, true),
  ('fee_type', 'Fee Type', 'picklist', 3, true),  -- Management, Advisory, Performance, Custody, Trading
  ('calculation_method', 'Calculation Method', 'picklist', 4, true),  -- Flat, Tiered, Performance-Based
  ('fee_schedule', 'Fee Schedule', 'json', 5, false),  -- Rate tiers
  ('billing_frequency', 'Billing Frequency', 'picklist', 6, true),  -- Monthly, Quarterly, Annually
  ('billing_method', 'Billing Method', 'picklist', 7, true),  -- In Advance, In Arrears
  ('billable_amount', 'Billable Amount', 'currency', 8, false),
  ('fee_amount', 'Fee Amount', 'currency', 9, false),
  ('period_start', 'Period Start', 'date', 10, true),
  ('period_end', 'Period End', 'date', 11, true),
  ('due_date', 'Due Date', 'date', 12, false),
  ('status', 'Status', 'picklist', 13, true),  -- Pending, Billed, Paid, Waived
  ('created_at', 'Created At', 'datetime', 14, false)
) AS t(key, label, type, seq, required)
ON CONFLICT DO NOTHING;

-- ============================================================================
-- SUCCESS MESSAGE
-- ============================================================================

DO $$
BEGIN
    RAISE NOTICE '✓ Financial Business Objects created successfully!';
    RAISE NOTICE '✓ Objects: Portfolio, Security, Position, TaxLot, Benchmark, Performance, Allocation, Fee';
END $$;
