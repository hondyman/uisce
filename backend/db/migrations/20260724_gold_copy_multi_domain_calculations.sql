-- Gold Copy Catalog Registration Script
-- Tenant: 99e99e99-99e9-49e9-89e9-99e99e99e999 (Master Core Tenant)
-- Multi-Domain Financial Calculation Library (40+ Functions)

BEGIN;

-- 1. BASE INPUT SEMANTIC TERMS
INSERT INTO catalog_node (node_id, tenant_id, node_key, node_name, node_type, term_type, data_type, is_active)
VALUES
  ('30000000-0000-0000-0000-000000000001', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'cash_flow_amount', 'Cash Flow Amount', 'SEMANTIC_TERM', 'MEASURE', 'number', true),
  ('30000000-0000-0000-0000-000000000002', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'cash_flow_date', 'Cash Flow Date', 'SEMANTIC_TERM', 'DIMENSION', 'date', true),
  ('30000000-0000-0000-0000-000000000003', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'beginning_market_value', 'Beginning Market Value', 'SEMANTIC_TERM', 'MEASURE', 'number', true),
  ('30000000-0000-0000-0000-000000000004', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'ending_market_value', 'Ending Market Value', 'SEMANTIC_TERM', 'MEASURE', 'number', true),
  ('30000000-0000-0000-0000-000000000005', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'clean_price', 'Clean Price', 'SEMANTIC_TERM', 'MEASURE', 'number', true),
  ('30000000-0000-0000-0000-000000000006', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'coupon_rate', 'Coupon Rate', 'SEMANTIC_TERM', 'ATTRIBUTE', 'number', true),
  ('30000000-0000-0000-0000-000000000007', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'portfolio_return', 'Portfolio Return', 'SEMANTIC_TERM', 'MEASURE', 'number', true),
  ('30000000-0000-0000-0000-000000000008', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'risk_free_rate', 'Risk Free Rate', 'SEMANTIC_TERM', 'ATTRIBUTE', 'number', true),
  ('30000000-0000-0000-0000-000000000009', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'portfolio_stdev', 'Portfolio Standard Deviation', 'SEMANTIC_TERM', 'MEASURE', 'number', true),
  ('30000000-0000-0000-0000-000000000010', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'cumulative_distributions', 'Cumulative Distributions', 'SEMANTIC_TERM', 'MEASURE', 'number', true),
  ('30000000-0000-0000-0000-000000000011', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'nav_current', 'Current NAV', 'SEMANTIC_TERM', 'MEASURE', 'number', true),
  ('30000000-0000-0000-0000-000000000012', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'cumulative_paid_in', 'Cumulative Paid-In Capital', 'SEMANTIC_TERM', 'MEASURE', 'number', true)
ON CONFLICT (node_id) DO NOTHING;

-- 2. CALCULATED SEMANTIC TERMS
INSERT INTO catalog_node (node_id, tenant_id, node_key, node_name, node_type, term_type, data_type, calculation_type, function_name, formula_expression, is_active)
VALUES
  -- Private Markets
  ('40000000-0000-0000-0000-000000000001', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'xirr', 'Extended Internal Rate of Return', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'XIRR', NULL, true),
  ('40000000-0000-0000-0000-000000000002', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'tvpi', 'Total Value to Paid-In', 'SEMANTIC_TERM', 'MEASURE', 'number', 'EXPRESSION', NULL, '(cumulative_distributions + nav_current) / cumulative_paid_in', true),
  ('40000000-0000-0000-0000-000000000003', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'dpi', 'Distributed to Paid-In', 'SEMANTIC_TERM', 'MEASURE', 'number', 'EXPRESSION', NULL, 'cumulative_distributions / cumulative_paid_in', true),
  ('40000000-0000-0000-0000-000000000004', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'rvpi', 'Residual Value to Paid-In', 'SEMANTIC_TERM', 'MEASURE', 'number', 'EXPRESSION', NULL, 'nav_current / cumulative_paid_in', true),
  ('40000000-0000-0000-0000-000000000005', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'waterfall_carry', 'Waterfall Carried Interest', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'WATERFALL_CARRY', NULL, true),

  -- Performance
  ('40000000-0000-0000-0000-000000000006', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'twr', 'Time-Weighted Return', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'TWR', NULL, true),
  ('40000000-0000-0000-0000-000000000007', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'modified_dietz', 'Modified Dietz Return', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'MODIFIED_DIETZ', NULL, true),
  ('40000000-0000-0000-0000-000000000008', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'brinson_attribution', 'Brinson Performance Attribution', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'BRINSON_ATTRIBUTION', NULL, true),

  -- Fixed Income
  ('40000000-0000-0000-0000-000000000009', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'ytm', 'Yield to Maturity', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'YTM', NULL, true),
  ('40000000-0000-0000-0000-000000000010', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'modified_duration', 'Modified Duration', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'MODIFIED_DURATION', NULL, true),
  ('40000000-0000-0000-0000-000000000011', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'dv01', 'Dollar Duration DV01', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'DV01', NULL, true),
  ('40000000-0000-0000-0000-000000000012', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'convexity', 'Bond Convexity', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'CONVEXITY', NULL, true),

  -- Risk & Derivatives
  ('40000000-0000-0000-0000-000000000013', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'parametric_var', 'Parametric VaR', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'PARAMETRIC_VAR', NULL, true),
  ('40000000-0000-0000-0000-000000000014', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'cvar', 'Conditional VaR Expected Shortfall', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'CVAR', NULL, true),
  ('40000000-0000-0000-0000-000000000015', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'option_delta', 'Black-Scholes Delta', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'BLACK_SCHOLES_DELTA', NULL, true),
  ('40000000-0000-0000-0000-000000000016', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'sharpe_ratio', 'Sharpe Ratio', 'SEMANTIC_TERM', 'MEASURE', 'number', 'EXPRESSION', NULL, '(portfolio_return - risk_free_rate) / portfolio_stdev', true),
  ('40000000-0000-0000-0000-000000000017', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'sortino_ratio', 'Sortino Ratio', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'SORTINO_RATIO', NULL, true),
  ('40000000-0000-0000-0000-000000000018', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'treynor_ratio', 'Treynor Ratio', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'TREYNOR_RATIO', NULL, true),

  -- Accounting (IBOR/ABOR)
  ('40000000-0000-0000-0000-000000000019', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'nav', 'Net Asset Value', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'NAV', NULL, true),
  ('40000000-0000-0000-0000-000000000020', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'realized_pnl', 'Realized Gain Loss', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'REALIZED_PNL', NULL, true),
  ('40000000-0000-0000-0000-000000000021', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'unrealized_pnl', 'Unrealized Gain Loss', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'UNREALIZED_PNL', NULL, true),
  ('40000000-0000-0000-0000-000000000022', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'day_count_fraction', 'Day Count Fraction', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'DAY_COUNT_FRACTION', NULL, true),

  -- Wealth & Hedge Funds
  ('40000000-0000-0000-0000-000000000023', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'performance_fee', 'Performance Fee Accrual HWM', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'PERFORMANCE_FEE', NULL, true),
  ('40000000-0000-0000-0000-000000000024', '99e99e99-99e9-49e9-89e9-99e99e99e999', 'tax_loss_harvesting', 'Tax Loss Harvesting Gain Loss', 'SEMANTIC_TERM', 'MEASURE', 'number', 'FUNCTION', 'TAX_LOSS_HARVESTING', NULL, true)
ON CONFLICT (node_id) DO NOTHING;

-- 3. DEPENDENCY GRAPH EDGES (USES_INPUT)
INSERT INTO catalog_edge (edge_id, tenant_id, from_node_id, to_node_id, edge_type, is_active)
VALUES
  -- XIRR
  ('50000000-0000-0000-0000-000000000001', '99e99e99-99e9-49e9-89e9-99e99e99e999', '40000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000001', 'USES_INPUT', true),
  ('50000000-0000-0000-0000-000000000002', '99e99e99-99e9-49e9-89e9-99e99e99e999', '40000000-0000-0000-0000-000000000001', '30000000-0000-0000-0000-000000000002', 'USES_INPUT', true),

  -- TWR
  ('50000000-0000-0000-0000-000000000003', '99e99e99-99e9-49e9-89e9-99e99e99e999', '40000000-0000-0000-0000-000000000006', '30000000-0000-0000-0000-000000000003', 'USES_INPUT', true),
  ('50000000-0000-0000-0000-000000000004', '99e99e99-99e9-49e9-89e9-99e99e99e999', '40000000-0000-0000-0000-000000000006', '30000000-0000-0000-0000-000000000004', 'USES_INPUT', true),

  -- Sharpe Ratio
  ('50000000-0000-0000-0000-000000000005', '99e99e99-99e9-49e9-89e9-99e99e99e999', '40000000-0000-0000-0000-000000000016', '30000000-0000-0000-0000-000000000007', 'USES_INPUT', true),
  ('50000000-0000-0000-0000-000000000006', '99e99e99-99e9-49e9-89e9-99e99e99e999', '40000000-0000-0000-0000-000000000016', '30000000-0000-0000-0000-000000000008', 'USES_INPUT', true),
  ('50000000-0000-0000-0000-000000000007', '99e99e99-99e9-49e9-89e9-99e99e99e999', '40000000-0000-0000-0000-000000000016', '30000000-0000-0000-0000-000000000009', 'USES_INPUT', true)
ON CONFLICT (edge_id) DO NOTHING;

COMMIT;
