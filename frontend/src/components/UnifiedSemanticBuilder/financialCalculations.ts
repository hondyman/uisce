export interface CoreOption {
  name: string;
  title: string;
  type: string;
  sql: string;
  description?: string;
  sourceTable?: string;
  sourceColumn?: string;
  format?: string;
  aggregationType?: string;
  defaultValue?: string;
  category?: string;
  subcategory?: string;
  preAggregationTemplate?: any;
  backendEndpoint?: string;
  accessType?: 'read-only';
  financial_calc?: {
    type: string;
    formula?: string;
    arguments?: Record<string, string>;
  };
}

export const libraryOptions: CoreOption[] = [
    { 
      name: 'investment_xirr', 
      title: 'Investment XIRR', 
      type: 'measure', 
      sql: "{{ xirr(ARRAY_AGG(${pre_agg_name}.cash_flow), ARRAY_AGG(${pre_agg_name}.transaction_date)) }}", 
      description: 'Calculate the internal rate of return for a series of cash flows that is not necessarily periodic. Requires a pre-aggregation for performance.', 
      aggregationType: 'none', 
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'IRR',
      preAggregationTemplate: {
        name: 'xirr_inputs_rollup',
        type: 'rollup',
        measures: ['<replace_with_cashflow_measure>'],
        dimensions: ['<replace_with_grouping_dimension>'],
        timeDimension: '<replace_with_date_dimension>',
        granularity: 'day',
        is_template: true,
        description: 'Pre-aggregates cash flows and dates to accelerate XIRR calculations.'
      }
    },
    { name: 'net_present_value', title: 'Net Present Value', type: 'measure', sql: 'NPV(discount_rate, value1, value2, ...)', description: 'Calculate the net present value of an investment using a discount rate and a series of future payments', aggregationType: 'none', category: 'Performance', subcategory: 'Valuation', accessType: 'read-only' },
    { name: 'sharpe_ratio', title: 'Sharpe Ratio', type: 'measure', sql: '(AVG(returns) - risk_free_rate) / STDDEV(returns)', description: 'Measure risk-adjusted return by calculating excess return per unit of risk', aggregationType: 'none', category: 'Risk', subcategory: 'Volatility', accessType: 'read-only' },
    { name: 'value_at_risk', title: 'Value at Risk (VaR)', type: 'measure', sql: 'PERCENTILE(portfolio_returns, 0.05) * portfolio_value', description: 'Estimate the potential loss in value of a portfolio over a defined period for a given confidence interval', aggregationType: 'none', category: 'Risk', subcategory: 'Market Risk', accessType: 'read-only' },
    { name: 'total_return', title: 'Total Return', type: 'measure', sql: '((ending_value + dividends) / beginning_value) - 1', description: 'Calculate the actual rate of return including capital appreciation and income', aggregationType: 'none', category: 'Performance', subcategory: 'Returns', accessType: 'read-only' },
    { name: 'compound_annual_growth', title: 'CAGR', type: 'measure', sql: 'POWER(ending_value / beginning_value, 1.0 / years) - 1', description: 'Compound Annual Growth Rate - the mean annual growth rate over a specified time period', aggregationType: 'none', category: 'Performance', subcategory: 'Growth', accessType: 'read-only' },
    { name: 'beta_coefficient', title: 'Beta Coefficient', type: 'measure', sql: 'COVAR(stock_returns, market_returns) / VAR(market_returns)', description: 'Measure of systematic risk - how much a security moves relative to the market', aggregationType: 'none', category: 'Risk', subcategory: 'Correlation', accessType: 'read-only' },
    { name: 'portfolio_allocation', title: 'Portfolio Allocation %', type: 'measure', sql: '(position_value / total_portfolio_value) * 100', description: 'Calculate the percentage allocation of each position within a portfolio', aggregationType: 'none', category: 'Wealth', subcategory: 'Allocation', accessType: 'read-only' },
    { name: 'drawdown_max', title: 'Maximum Drawdown', type: 'measure', sql: 'MIN((current_value - peak_value) / peak_value)', description: 'Measure the largest peak-to-trough decline in portfolio value', aggregationType: 'none', category: 'Risk', subcategory: 'Drawdown', accessType: 'read-only' },
    { 
      name: 'irr_calculation', 
      title: 'Internal Rate of Return', 
      type: 'measure', 
      sql: "{{ irr(ARRAY_AGG(${pre_agg_name}.cash_flow)) }}", 
      description: 'Calculate the discount rate that makes NPV equal to zero for a series of cash flows. Uses backend calculation service for accuracy.', 
      aggregationType: 'none', 
      accessType: 'read-only',
      subcategory: 'IRR',
      category: 'Private Markets',
      preAggregationTemplate: {
        name: 'irr_inputs_rollup',
        type: 'rollup',
        measures: ['<replace_with_cashflow_measure>'],
        dimensions: ['<replace_with_grouping_dimension>'],
        timeDimension: '<replace_with_period_dimension>',
        granularity: 'month',
        is_template: true,
        description: 'Pre-aggregates periodic cash flows to accelerate IRR calculations.'
      },
      backendEndpoint: '/api/fabric/financial/irr'
    },
    { name: 'multiple_invested_capital', title: 'Multiple of Invested Capital', type: 'measure', sql: 'total_distributions / total_contributions', description: 'Private equity metric showing total value returned relative to capital invested', aggregationType: 'none', category: 'Private Markets', subcategory: 'Multiples', accessType: 'read-only' },
    { name: 'distributed_paid_in', title: 'DPI Ratio', type: 'measure', sql: 'cumulative_distributions / paid_in_capital', description: 'Distributed to Paid-In capital ratio for private equity investments', aggregationType: 'none', category: 'Private Markets', subcategory: 'Ratios', accessType: 'read-only' },
    {
      name: 'net_present_value_calc',
      title: 'Net Present Value (NPV)',
      type: 'measure',
      sql: "{{ npv(0.08, ARRAY_AGG(${pre_agg_name}.cash_flow)) }}",
      description: 'Calculates the present value of future cash flows minus the initial investment, using a backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Performance',
      subcategory: 'Valuation',
      backendEndpoint: '/api/fabric/financial/npv',
      preAggregationTemplate: {
        name: 'npv_inputs_rollup',
        type: 'rollup',
        measures: ['<replace_with_cashflow_measure>'],
        dimensions: ['<replace_with_grouping_dimension>'],
        timeDimension: '<replace_with_period_dimension>',
        granularity: 'month',
        is_template: true,
        description: 'Pre-aggregates periodic cash flows to accelerate NPV calculations.'
      }
    },
    {
      name: 'modified_irr_calc',
      title: 'Modified IRR (MIRR)',
      type: 'measure',
      sql: "{{ mirr(ARRAY_AGG(${pre_agg_name}.cash_flow), 0.07, 0.05) }}",
      description: 'Calculates MIRR, which accounts for cost of capital and reinvestment rate. Uses backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Performance',
      subcategory: 'IRR',
      backendEndpoint: '/api/fabric/financial/mirr',
      preAggregationTemplate: {
        name: 'mirr_inputs_rollup',
        type: 'rollup',
        measures: ['<replace_with_cashflow_measure>'],
        dimensions: ['<replace_with_grouping_dimension>'],
        timeDimension: '<replace_with_period_dimension>',
        granularity: 'month',
        is_template: true,
        description: 'Pre-aggregates periodic cash flows to accelerate MIRR calculations.'
      }
    },
    {
      name: 'payback_period_calc',
      title: 'Payback Period',
      type: 'measure',
      sql: "{{ payback_period(ARRAY_AGG(${pre_agg_name}.cash_flow)) }}",
      description: 'Calculates the time required to recover the initial investment. Uses backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Performance',
      subcategory: 'Valuation',
      backendEndpoint: '/api/fabric/financial/payback',
      preAggregationTemplate: {
        name: 'payback_inputs_rollup',
        type: 'rollup',
        measures: ['<replace_with_cashflow_measure>'],
        dimensions: ['<replace_with_grouping_dimension>'],
        timeDimension: '<replace_with_period_dimension>',
        granularity: 'month',
        is_template: true,
        description: 'Pre-aggregates periodic cash flows to accelerate Payback Period calculations.'
      }
    },
    {
      name: 'weighted_irr_calc',
      title: 'Weighted IRR (WIRR)',
      type: 'measure',
      sql: "{{ wirr(ARRAY_AGG(${pre_agg_name}.cash_flow), ARRAY_AGG(${pre_agg_name}.weight)) }}",
      description: 'Calculates the portfolio IRR weighted by investment size. Uses backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Performance',
      subcategory: 'IRR',
      backendEndpoint: '/api/fabric/financial/wirr',
      preAggregationTemplate: {
        name: 'wirr_inputs_rollup',
        type: 'rollup',
        measures: ['<replace_with_cashflow_measure>', '<replace_with_weight_measure>'],
        dimensions: ['<replace_with_grouping_dimension>'],
        timeDimension: '<replace_with_period_dimension>',
        granularity: 'month',
        is_template: true,
        description: 'Pre-aggregates periodic cash flows and weights to accelerate WIRR calculations.'
      }
    },
    {
      name: 'cash_on_cash_return',
      title: 'Cash-on-Cash Return',
      type: 'measure',
      sql: 'SUM(annual_cash_flow) / SUM(total_cash_invested)',
      description: 'Measures the annual pre-tax cash flow as a percentage of the total cash invested.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Performance',
      subcategory: 'Ratios'
    },
    {
      name: 'equity_multiple',
      title: 'Equity Multiple',
      type: 'measure',
      sql: 'SUM(total_distributions) / SUM(total_invested)',
      description: 'Measures the total cash returned to an investor relative to the total cash invested.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Multiples'
    },
    {
      name: 'annualized_return_cagr',
      title: 'Annualized Return (CAGR)',
      type: 'measure',
      sql: 'POWER(ending_value / beginning_value, 1.0 / years) - 1',
      description: 'Calculates the compound annual growth rate of an investment over a specified period.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Performance',
      subcategory: 'Growth'
    },
    {
      name: 'sharpe_ratio_calc',
      title: 'Sharpe Ratio',
      type: 'measure',
      sql: '({{ avg_return }} - {{ risk_free_rate }}) / {{ std_dev }}',
      description: 'Measures the risk-adjusted return of an investment portfolio. Uses backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk',
      subcategory: 'Volatility',
      backendEndpoint: '/api/fabric/financial/sharpe'
    },
    // --- Insurance Calculation Templates ---
    {
      name: 'loss_ratio',
      title: 'Loss Ratio',
      type: 'measure',
      sql: 'SUM(claim_amount) / SUM(premium_amount)',
      description: 'Measures the claims paid out by an insurer as a percentage of premiums earned.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Profitability'
    },
    {
      name: 'combined_ratio_calc',
      title: 'Combined Ratio',
      type: 'measure',
      sql: "{{ sum_of_ratios(SUM(claim_amount), SUM(premium_amount), SUM(expenses), SUM(premium_amount)) }}",
      description: 'Calculates the combined ratio (Loss Ratio + Expense Ratio) to measure underwriting profitability. Uses backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Profitability',
      backendEndpoint: '/api/fabric/financial/sum-of-ratios',
      preAggregationTemplate: {
        name: 'combined_ratio_inputs_rollup',
        type: 'rollup',
        measures: ['<replace_with_claim_measure>', '<replace_with_premium_measure>', '<replace_with_expense_measure>'],
        dimensions: ['<replace_with_grouping_dimension>'],
        timeDimension: '<replace_with_period_dimension>',
        granularity: 'month',
        is_template: true,
        description: 'Pre-aggregates claims, premiums, and expenses to accelerate Combined Ratio calculations.'
      }
    },
    {
      name: 'claims_reserve_adequacy',
      title: 'Claims Reserve Adequacy',
      type: 'measure',
      sql: 'SUM(reserve_amount) / SUM(outstanding_claims)',
      description: 'Measures the sufficiency of funds set aside to pay for future claims.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Risk'
    },
    // --- Banking & Lending Calculation Templates ---
    {
      name: 'loan_to_value_ratio',
      title: 'Loan-to-Value (LTV) Ratio',
      type: 'measure',
      sql: 'SUM(outstanding_balance) / SUM(appraised_value)',
      description: 'Assesses lending risk by comparing the loan amount to the market value of the asset.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Banking',
      subcategory: 'Risk'
    },
    {
      name: 'net_interest_margin',
      title: 'Net Interest Margin (NIM)',
      type: 'measure',
      sql: '(SUM(interest_income) - SUM(interest_expense)) / AVG(earning_assets)',
      description: 'Measures the difference between interest income and interest expense, as a percentage of earning assets.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Banking',
      subcategory: 'Profitability'
    },
    {
      name: 'capital_adequacy_ratio',
      title: 'Capital Adequacy Ratio (CAR)',
      type: 'measure',
      sql: '(SUM(tier1_capital) + SUM(tier2_capital)) / SUM(risk_weighted_assets)',
      description: 'Measures a bank\'s capital in relation to its risk-weighted assets to ensure it can absorb potential losses.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Banking',
      subcategory: 'Regulatory'
    },
    // --- Quant Finance & Advanced Risk Templates ---
    {
      name: 'portfolio_volatility_calc',
      title: 'Portfolio Volatility (Markowitz)',
      type: 'measure',
      sql: "{{ portfolio_volatility(ARRAY_AGG(weights), ARRAY_AGG(volatilities), CORR_MATRIX(asset_id)) }}",
      description: 'Calculates portfolio volatility using a covariance matrix, a key component for Markowitz optimization. Uses backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Portfolio Analytics',
      backendEndpoint: '/api/fabric/financial/portfolio-volatility'
    },
    {
      name: 'tracking_error_calc',
      title: 'Tracking Error',
      type: 'measure',
      sql: "{{ tracking_error(ARRAY_AGG(asset_return), ARRAY_AGG(benchmark_return)) }}",
      description: 'Measures the standard deviation of the difference between portfolio and benchmark returns (active risk). Uses backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Portfolio Analytics',
      backendEndpoint: '/api/fabric/financial/tracking-error'
    },
    {
      name: 'information_ratio_calc',
      title: 'Information Ratio',
      type: 'measure',
      sql: "{{ information_ratio(ARRAY_AGG(asset_return), ARRAY_AGG(benchmark_return)) }}",
      description: 'Measures risk-adjusted returns of a portfolio against a benchmark (excess return per unit of active risk). Uses backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Portfolio Analytics',
      backendEndpoint: '/api/fabric/financial/information-ratio'
    },
    {
      name: 'gbm_simulation_calc',
      title: 'Geometric Brownian Motion (GBM)',
      type: 'measure',
      sql: "{{ gbm_simulation(100.0, 0.05, 0.2, 1.0, 252) }}",
      description: 'Simulates an asset price path using Geometric Brownian Motion. Uses backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Stochastic Models',
      backendEndpoint: '/api/fabric/financial/gbm'
    },
    {
      name: 'monte_carlo_option_calc',
      title: 'Monte Carlo Option Pricing',
      type: 'measure',
      sql: "{{ monte_carlo_option('call', 100, 105, 0.5, 0.02, 0.25, 10000) }}",
      description: 'Prices a European option using Monte Carlo simulation with GBM. Uses backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Derivatives Pricing',
      backendEndpoint: '/api/fabric/financial/mc-option'
    },
    {
      name: 'var_parametric_calc',
      title: 'Value at Risk (Parametric)',
      type: 'measure',
      sql: "{{ var_parametric(0.99, 0.001, 0.02, 1) }}",
      description: 'Calculates Parametric Value at Risk (VaR) using the variance-covariance method. Uses backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk',
      subcategory: 'Market Risk',
      backendEndpoint: '/api/fabric/financial/var-parametric'
    },
    {
      name: 'black_scholes_calc',
      title: 'Black-Scholes Option Price',
      type: 'measure',
      sql: "{{ black_scholes('call', 100, 105, 0.5, 0.02, 0.25) }}",
      description: 'Calculates the price and Greeks of a European option using the Black-Scholes model. Uses backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Derivatives Pricing',
      backendEndpoint: '/api/fabric/financial/black-scholes'
    },
    {
      name: 'credit_var_calc',
      title: 'Credit Value at Risk (Credit VaR)',
      type: 'measure',
      sql: "{{ credit_var(0.99, ARRAY_AGG(exposure), ARRAY_AGG(pd), ARRAY_AGG(lgd)) }}",
      description: 'Estimates the potential for loss on a credit portfolio. Uses backend calculation service.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk',
      subcategory: 'Credit Risk',
      backendEndpoint: '/api/fabric/financial/credit-var'
    },
    // Insurance Calculations - Underwriting Category
    {
      name: 'insurance_loss_ratio',
      title: 'Loss Ratio',
      type: 'measure',
      sql: 'SUM(claim_amount) / SUM(premium_amount)',
      description: 'Measures underwriting profitability by comparing claims paid to premiums earned',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Underwriting',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'insurance_expense_ratio',
      title: 'Expense Ratio',
      type: 'measure',
      sql: 'SUM(expenses) / SUM(premium_amount)',
      description: 'Measures operating expenses as a percentage of premiums earned',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Underwriting',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'insurance_combined_ratio',
      title: 'Combined Ratio',
      type: 'measure',
      sql: '(SUM(claim_amount) + SUM(expenses)) / SUM(premium_amount)',
      description: 'Combined measure of underwriting profitability including both losses and expenses',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Underwriting',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'insurance_net_claims_ratio',
      title: 'Net Claims Ratio',
      type: 'measure',
      sql: 'SUM(net_claims) / SUM(net_premiums)',
      description: 'Focuses on claims net of reinsurance recoveries',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Underwriting',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'insurance_claim_frequency',
      title: 'Claim Frequency',
      type: 'measure',
      sql: 'COUNT(claims) / COUNT(policies)',
      description: 'Measures the rate at which claims occur per policy',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Underwriting',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'insurance_claim_severity',
      title: 'Claim Severity',
      type: 'measure',
      sql: 'SUM(claim_amount) / COUNT(claims)',
      description: 'Measures the average cost per claim',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Underwriting',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'insurance_retention_rate',
      title: 'Policyholder Retention Rate',
      type: 'measure',
      sql: 'COUNT(renewed_policies) / COUNT(eligible_renewals)',
      description: 'Customer loyalty and renewal strength metric',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Underwriting',
      backendEndpoint: '/api/calc/run'
    },
    // Insurance Calculations - Reserving Category
    {
      name: 'insurance_reserve_adequacy',
      title: 'Reserve Adequacy Ratio',
      type: 'measure',
      sql: 'SUM(actual_claims_paid) / SUM(reserves_held)',
      description: 'Measures whether reserves are sufficient to cover actual claims',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Reserving',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'insurance_reserve_development_ratio',
      title: 'Reserve Development Ratio',
      type: 'measure',
      sql: 'SUM(current_year_reserves) / SUM(prior_year_reserves)',
      description: 'Tracks how reserves evolve over time - key for long-tail lines',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Reserving',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'insurance_loss_reserve_leverage',
      title: 'Net Loss Reserve Leverage',
      type: 'measure',
      sql: 'SUM(loss_reserves) / SUM(policyholder_surplus)',
      description: 'Measures exposure to reserving errors',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Reserving',
      backendEndpoint: '/api/calc/run'
    },
    // Insurance Calculations - Solvency Category
    {
      name: 'insurance_solvency_margin_ratio',
      title: 'Solvency Margin Ratio',
      type: 'measure',
      sql: 'SUM(available_solvency_margin) / SUM(required_solvency_margin)',
      description: 'Regulatory capital adequacy metric',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Solvency',
      backendEndpoint: '/api/calc/run'
    },
    // Insurance Calculations - Profitability Category
    {
      name: 'insurance_operating_ratio',
      title: 'Operating Ratio',
      type: 'measure',
      sql: '(SUM(claim_amount) + SUM(expenses) - SUM(investment_income)) / SUM(premium_amount)',
      description: 'Adds investment income to combined ratio for full operating performance',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Profitability',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'insurance_investment_yield',
      title: 'Investment Yield',
      type: 'measure',
      sql: 'SUM(investment_income) / SUM(invested_assets)',
      description: 'Returns on invested assets - critical for profitability',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Profitability',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'insurance_premium_growth_rate',
      title: 'Premium Growth Rate',
      type: 'measure',
      sql: '(SUM(current_premium_amount) - SUM(prior_premium_amount)) / SUM(prior_premium_amount)',
      description: 'Top-line momentum across periods',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Insurance',
      subcategory: 'Profitability',
      backendEndpoint: '/api/calc/run'
    },

    // --- Private Markets Calculation Templates ---
    {
      name: 'private_markets_xirr',
      title: 'Private Markets XIRR',
      type: 'measure',
      sql: 'XIRR cash flows and dates',
      description: 'Internal rate of return for irregular cash flows.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Performance',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_irr',
      title: 'Private Markets IRR',
      type: 'measure',
      sql: 'IRR cash flows',
      description: 'Discount rate that makes NPV = 0 for a series of cash flows.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Performance',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_tvpi',
      title: 'Total Value to Paid-In (TVPI)',
      type: 'measure',
      sql: '(SUM(cumulative_distributions) + SUM(remaining_value)) / SUM(paid_in_capital)',
      description: 'Total value to paid-in capital ratio (distributions + residual value).',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Performance',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_rvpi',
      title: 'Residual Value to Paid-In (RVPI)',
      type: 'measure',
      sql: 'SUM(remaining_value) / SUM(paid_in_capital)',
      description: 'Residual value to paid-in capital ratio (unrealized NAV).',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Performance',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_pme',
      title: 'Public Market Equivalent (PME)',
      type: 'measure',
      sql: 'PME vs benchmark',
      description: 'Public Market Equivalent vs benchmark index.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Performance',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_pme_kaplan_schoar',
      title: 'PME (Kaplan-Schoar)',
      type: 'measure',
      sql: 'SUM(distributions_indexed) / SUM(contributions_indexed)',
      description: 'Kaplan–Schoar PME ratio (indexed distributions / indexed contributions).',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Performance',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_gross_irr',
      title: 'Gross IRR',
      type: 'measure',
      sql: 'Gross IRR cash flows',
      description: 'Gross internal rate of return before fees and carry.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Performance',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_net_irr',
      title: 'Net IRR',
      type: 'measure',
      sql: 'Net IRR cash flows',
      description: 'Net internal rate of return after fees and carry.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Performance',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_moic',
      title: 'Multiple of Invested Capital (MOIC)',
      type: 'measure',
      sql: 'SUM(total_distributions) / SUM(total_contributions)',
      description: 'Multiple of invested capital (total value returned / capital invested).',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Multiples',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_dpi',
      title: 'Distributed to Paid-In (DPI)',
      type: 'measure',
      sql: 'SUM(cumulative_distributions) / SUM(paid_in_capital)',
      description: 'Distributed to Paid-In capital ratio.',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Multiples',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_equity_multiple',
      title: 'Equity Multiple',
      type: 'measure',
      sql: 'SUM(total_value) / SUM(equity_invested)',
      description: 'Total value returned relative to equity invested.',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Multiples',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_gross_moic',
      title: 'Gross MOIC',
      type: 'measure',
      sql: '(SUM(gross_distributions) + SUM(remaining_value)) / SUM(gross_contributions)',
      description: 'Gross multiple of invested capital before fees.',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Multiples',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_net_moic',
      title: 'Net MOIC',
      type: 'measure',
      sql: '(SUM(net_distributions) + SUM(remaining_value)) / SUM(net_contributions)',
      description: 'Net multiple of invested capital after fees.',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Multiples',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_j_curve',
      title: 'J-Curve Analysis',
      type: 'measure',
      sql: 'Cumulative net cash flow ordered by date',
      description: 'Cumulative net cash flow over time to visualize the J-curve effect.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Cash Flow',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_capital_call_ratio',
      title: 'Capital Call Ratio',
      type: 'measure',
      sql: 'SUM(called_capital) / SUM(committed_capital)',
      description: 'Capital called as a percentage of committed capital.',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Liquidity',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_unfunded_commitment_ratio',
      title: 'Unfunded Commitment Ratio',
      type: 'measure',
      sql: 'SUM(unfunded_commitments) / SUM(committed_capital)',
      description: 'Unfunded commitments relative to total committed capital.',
      aggregationType: 'ratio',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Liquidity',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'private_markets_nav',
      title: 'Net Asset Value (NAV)',
      type: 'measure',
      sql: 'SUM(current_fair_value)',
      description: 'Net asset value of portfolio holdings.',
      aggregationType: 'sum',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'Valuation',
      backendEndpoint: '/api/calc/run'
    },
    // Quant Finance - Market Risk
    {
      name: 'quant_market_var_historical',
      title: 'VaR - Historical Simulation',
      type: 'measure',
      sql: 'PERCENTILE(returns, 1 - confidence_level) * -1',
      description: 'Value at Risk using historical simulation method at specified confidence level.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Market Risk',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'quant_market_var_parametric',
      title: 'VaR - Parametric',
      type: 'measure',
      sql: '(mean_return - (std_dev * NORM_S_INV(confidence_level))) * holding_period_days',
      description: 'Value at Risk using variance-covariance method at specified confidence level.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Market Risk',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'quant_market_var_montecarlo',
      title: 'VaR - Monte Carlo',
      type: 'measure',
      sql: 'PERCENTILE(simulated_returns, 1 - confidence_level) * -1',
      description: 'Value at Risk using Monte Carlo simulation at specified confidence level.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Market Risk',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'quant_market_cvar',
      title: 'Conditional VaR (CVaR)',
      type: 'measure',
      sql: 'AVG(returns[returns < PERCENTILE(returns, 1 - confidence_level)])',
      description: 'Expected shortfall beyond VaR at specified confidence level.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Market Risk',
      backendEndpoint: '/api/calc/run'
    },
    // Quant Finance - Derivatives Pricing
    {
      name: 'quant_derivatives_greeks',
      title: 'Option Greeks',
      type: 'measure',
      sql: 'BLACK_SCHOLES_GREEKS(spot_price, strike_price, time_to_maturity, risk_free_rate, volatility, dividend_yield, option_type)',
      description: 'Calculate delta, gamma, theta, vega, rho for European options.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Derivatives Pricing',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'quant_pricing_black_scholes',
      title: 'Black-Scholes Option Price',
      type: 'measure',
      sql: 'BLACK_SCHOLES(spot_price, strike_price, time_to_maturity, risk_free_rate, volatility, dividend_yield, option_type)',
      description: 'European option price using Black-Scholes model.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Derivatives Pricing',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'quant_pricing_binomial',
      title: 'Binomial Option Price',
      type: 'measure',
      sql: 'BINOMIAL_OPTION(spot_price, strike_price, time_to_maturity, risk_free_rate, volatility, steps, option_type)',
      description: 'European option price using binomial tree model.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Derivatives Pricing',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'quant_pricing_implied_vol',
      title: 'Implied Volatility',
      type: 'measure',
      sql: 'IMPLIED_VOLATILITY(market_price, spot_price, strike_price, time_to_maturity, risk_free_rate, dividend_yield, option_type)',
      description: 'Implied volatility from market option price using Black-Scholes model.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Derivatives Pricing',
      backendEndpoint: '/api/calc/run'
    },
    // Quant Finance - Fixed Income
    {
      name: 'quant_fixed_income_duration_convexity',
      title: 'Bond Duration & Convexity',
      type: 'measure',
      sql: 'BOND_DURATION_CONVEXITY(cash_flows, yield_to_maturity, frequency)',
      description: 'Calculate Macaulay duration, modified duration, and convexity for bonds.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Fixed Income',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'quant_fixed_income_yield_curve_bootstrap',
      title: 'Yield Curve Bootstrapping',
      type: 'measure',
      sql: 'YIELD_CURVE_BOOTSTRAP(instruments, compounding)',
      description: 'Bootstrap zero-coupon yield curve from par yields.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Quant Finance',
      subcategory: 'Fixed Income',
      backendEndpoint: '/api/calc/run'
    },
    // Risk Management - Market Risk
    {
      name: 'risk_market_stress_testing',
      title: 'Stress Testing - P&L Impact',
      type: 'measure',
      sql: 'STRESS_TEST_PNL_IMPACT(scenarios, portfolio_value)',
      description: 'Calculate P&L impact under various stress scenarios.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk Management',
      subcategory: 'Market Risk',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'risk_market_beta_coefficient',
      title: 'Beta Coefficient',
      type: 'measure',
      sql: 'BETA_COEFFICIENT(asset_returns, benchmark_returns)',
      description: 'Calculate beta coefficient measuring systematic risk.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk Management',
      subcategory: 'Market Risk',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'risk_market_correlation_matrix',
      title: 'Correlation Matrix',
      type: 'measure',
      sql: 'CORRELATION_MATRIX(asset_returns_matrix)',
      description: 'Calculate correlation matrix for portfolio diversification analysis.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk Management',
      subcategory: 'Market Risk',
      backendEndpoint: '/api/calc/run'
    },
    // Risk Management - Credit Risk
    {
      name: 'risk_credit_pd_calculation',
      title: 'Probability of Default (PD)',
      type: 'measure',
      sql: 'PROBABILITY_OF_DEFAULT(credit_score, pd_model, risk_factors)',
      description: 'Estimate probability of default using credit scoring models.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk Management',
      subcategory: 'Credit Risk',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'risk_credit_lgd_calculation',
      title: 'Loss Given Default (LGD)',
      type: 'measure',
      sql: 'LOSS_GIVEN_DEFAULT(collateral_value, exposure_amount, recovery_rate, collateral_type)',
      description: 'Calculate loss given default with collateral recovery.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk Management',
      subcategory: 'Credit Risk',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'risk_credit_ead_calculation',
      title: 'Exposure at Default (EAD)',
      type: 'measure',
      sql: 'EXPOSURE_AT_DEFAULT(current_exposure, credit_limit, drawn_amount, undrawn_commitment)',
      description: 'Calculate exposure at default for credit risk measurement.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk Management',
      subcategory: 'Credit Risk',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'risk_credit_ecl_calculation',
      title: 'Expected Credit Loss (ECL)',
      type: 'measure',
      sql: 'EXPECTED_CREDIT_LOSS(pd, lgd, ead, exposure_period)',
      description: 'Calculate expected credit loss under IFRS 9/CECL frameworks.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk Management',
      subcategory: 'Credit Risk',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'risk_credit_var_calculation',
      title: 'Credit Value at Risk',
      type: 'measure',
      sql: 'CREDIT_VAR(confidence_level, exposures, pds, lgds)',
      description: 'Calculate credit VaR for portfolio credit loss distribution.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk Management',
      subcategory: 'Credit Risk',
      backendEndpoint: '/api/calc/run'
    },
    // Risk Management - Operational Risk
    {
      name: 'risk_operational_rcsa_scoring',
      title: 'RCSA Risk Scoring',
      type: 'measure',
      sql: 'RCSA_SCORING(risk_factors, risk_category, business_unit)',
      description: 'Calculate Risk Control Self-Assessment scores.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk Management',
      subcategory: 'Operational Risk',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'risk_operational_fair_quantification',
      title: 'FAIR Cyber Risk Quantification',
      type: 'measure',
      sql: 'FAIR_QUANTIFICATION(primary_loss_factors, secondary_loss_factors)',
      description: 'Quantify cyber risk using Factor Analysis of Information Risk (FAIR).',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk Management',
      subcategory: 'Operational Risk',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'risk_operational_kri_monitoring',
      title: 'KRI Threshold Monitoring',
      type: 'measure',
      sql: 'KRI_MONITORING(kris)',
      description: 'Monitor Key Risk Indicators with threshold-based alerting.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Risk Management',
      subcategory: 'Operational Risk',
      backendEndpoint: '/api/calc/run'
    },
    // Compliance & Regulatory - Banking/Basel III
    {
      name: 'compliance_banking_lcr_calculation',
      title: 'Liquidity Coverage Ratio (LCR)',
      type: 'measure',
      sql: 'LIQUIDITY_COVERAGE_RATIO(high_quality_liquid_assets, net_cash_outflows)',
      description: 'Calculate LCR under Basel III liquidity framework.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Compliance & Regulatory',
      subcategory: 'Banking/Basel III',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'compliance_banking_nsfr_calculation',
      title: 'Net Stable Funding Ratio (NSFR)',
      type: 'measure',
      sql: 'NET_STABLE_FUNDING_RATIO(available_stable_funding, required_stable_funding)',
      description: 'Calculate NSFR under Basel III funding framework.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Compliance & Regulatory',
      subcategory: 'Banking/Basel III',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'compliance_banking_leverage_ratio',
      title: 'Leverage Ratio',
      type: 'measure',
      sql: 'LEVERAGE_RATIO(tier1_capital, total_exposures)',
      description: 'Calculate leverage ratio under Basel III supplementary framework.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Compliance & Regulatory',
      subcategory: 'Banking/Basel III',
      backendEndpoint: '/api/calc/run'
    },
    // Compliance & Regulatory - Insurance/Solvency II
    {
      name: 'compliance_insurance_scr_calculation',
      title: 'Solvency Capital Requirement (SCR)',
      type: 'measure',
      sql: 'SOLVENCY_CAPITAL_REQUIREMENT(market_risk_scr, credit_risk_scr, operational_risk_scr, insurance_risk_scr, correlation_matrix)',
      description: 'Calculate SCR under Solvency II capital framework.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Compliance & Regulatory',
      subcategory: 'Insurance/Solvency II',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'compliance_insurance_mcr_calculation',
      title: 'Minimum Capital Requirement (MCR)',
      type: 'measure',
      sql: 'MINIMUM_CAPITAL_REQUIREMENT(written_premiums, technical_provisions, mcr_floor, mcr_cap)',
      description: 'Calculate MCR under Solvency II minimum capital framework.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Compliance & Regulatory',
      subcategory: 'Insurance/Solvency II',
      backendEndpoint: '/api/calc/run'
    },
    // Compliance & Regulatory - AML/KYC
    {
      name: 'compliance_aml_transaction_monitoring',
      title: 'AML Transaction Risk Scoring',
      type: 'measure',
      sql: 'TRANSACTION_RISK_SCORING(transaction_amount, transaction_frequency, customer_risk_profile, geographic_risk, product_risk)',
      description: 'Calculate risk scores for AML transaction monitoring.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Compliance & Regulatory',
      subcategory: 'AML/KYC',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'compliance_aml_customer_risk_rating',
      title: 'Customer Risk Rating (KYC)',
      type: 'measure',
      sql: 'CUSTOMER_RISK_RATING(customer_profile, risk_weights)',
      description: 'Calculate customer risk ratings using weighted scoring methodology.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Compliance & Regulatory',
      subcategory: 'AML/KYC',
      backendEndpoint: '/api/calc/run'
    },
    // Compliance & Regulatory - Market Conduct
    {
      name: 'compliance_market_best_execution',
      title: 'Best Execution Slippage',
      type: 'measure',
      sql: 'BEST_EXECUTION_SLIPPAGE(execution_price, benchmark_price, order_size, market_volatility)',
      description: 'Analyze execution slippage for best execution compliance.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Compliance & Regulatory',
      subcategory: 'Market Conduct',
      backendEndpoint: '/api/calc/run'
    },
    {
      name: 'compliance_market_trade_surveillance',
      title: 'Trade Surveillance Alerts',
      type: 'measure',
      sql: 'TRADE_SURVEILLANCE_ALERTS(trade_volume, average_volume, price_movement, normal_price_std_dev, time_window)',
      description: 'Generate statistical alerts for unusual trading activity.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Compliance & Regulatory',
      subcategory: 'Market Conduct',
      backendEndpoint: '/api/calc/run'
    },
    // Excel Formula Metrics
    {
      name: 'excel_xirr',
      title: 'Excel XIRR (Internal Rate of Return)',
      type: 'measure',
      sql: "{{ excel_formula('=XIRR({cash_flows}, {dates})') }}",
      description: 'Calculate the internal rate of return for irregular cash flows using Excel XIRR function.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'IRR',
      financial_calc: {
        type: 'excel_formula',
        formula: '=XIRR({cash_flows}, {dates})',
        arguments: {
          cash_flows: 'ARRAY_AGG(cash_flow)',
          dates: 'ARRAY_AGG(transaction_date)'
        }
      },
      backendEndpoint: '/api/fabric/financial/excel'
    },
    {
      name: 'excel_npv',
      title: 'Excel NPV (Net Present Value)',
      type: 'measure',
      sql: "{{ excel_formula('=NPV({rate}, {cash_flows})') }}",
      description: 'Calculate the net present value using Excel NPV function.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Performance',
      subcategory: 'Valuation',
      financial_calc: {
        type: 'excel_formula',
        formula: '=NPV({rate}, {cash_flows})',
        arguments: {
          rate: 'discount_rate',
          cash_flows: 'ARRAY_AGG(cash_flow)'
        }
      },
      backendEndpoint: '/api/fabric/financial/excel'
    },
    {
      name: 'excel_pv',
      title: 'Excel PV (Present Value)',
      type: 'measure',
      sql: "{{ excel_formula('=PV({rate}, {nper}, {pmt}, {fv})') }}",
      description: 'Calculate the present value using Excel PV function.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Performance',
      subcategory: 'Valuation',
      financial_calc: {
        type: 'excel_formula',
        formula: '=PV({rate}, {nper}, {pmt}, {fv})',
        arguments: {
          rate: 'discount_rate',
          nper: 'periods',
          pmt: 'payment',
          fv: 'future_value'
        }
      },
      backendEndpoint: '/api/fabric/financial/excel'
    },
    {
      name: 'excel_fv',
      title: 'Excel FV (Future Value)',
      type: 'measure',
      sql: "{{ excel_formula('=FV({rate}, {nper}, {pmt}, {pv})') }}",
      description: 'Calculate the future value using Excel FV function.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Performance',
      subcategory: 'Valuation',
      financial_calc: {
        type: 'excel_formula',
        formula: '=FV({rate}, {nper}, {pmt}, {pv})',
        arguments: {
          rate: 'interest_rate',
          nper: 'periods',
          pmt: 'payment',
          pv: 'present_value'
        }
      },
      backendEndpoint: '/api/fabric/financial/excel'
    },
    {
      name: 'excel_pmt',
      title: 'Excel PMT (Payment)',
      type: 'measure',
      sql: "{{ excel_formula('=PMT({rate}, {nper}, {pv}, {fv})') }}",
      description: 'Calculate the payment for a loan using Excel PMT function.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Performance',
      subcategory: 'Valuation',
      financial_calc: {
        type: 'excel_formula',
        formula: '=PMT({rate}, {nper}, {pv}, {fv})',
        arguments: {
          rate: 'interest_rate',
          nper: 'periods',
          pv: 'present_value',
          fv: 'future_value'
        }
      },
      backendEndpoint: '/api/fabric/financial/excel'
    },
    {
      name: 'excel_irr',
      title: 'Excel IRR (Internal Rate of Return)',
      type: 'measure',
      sql: "{{ excel_formula('=IRR({cash_flows})') }}",
      description: 'Calculate the internal rate of return for periodic cash flows using Excel IRR function.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'IRR',
      financial_calc: {
        type: 'excel_formula',
        formula: '=IRR({cash_flows})',
        arguments: {
          cash_flows: 'ARRAY_AGG(cash_flow)'
        }
      },
      backendEndpoint: '/api/fabric/financial/excel'
    },
    {
      name: 'excel_mirr',
      title: 'Excel MIRR (Modified IRR)',
      type: 'measure',
      sql: "{{ excel_formula('=MIRR({cash_flows}, {finance_rate}, {reinvest_rate})') }}",
      description: 'Calculate the modified internal rate of return using Excel MIRR function.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Performance',
      subcategory: 'IRR',
      financial_calc: {
        type: 'excel_formula',
        formula: '=MIRR({cash_flows}, {finance_rate}, {reinvest_rate})',
        arguments: {
          cash_flows: 'ARRAY_AGG(cash_flow)',
          finance_rate: 'finance_rate',
          reinvest_rate: 'reinvest_rate'
        }
      },
      backendEndpoint: '/api/fabric/financial/excel'
    },
    {
      name: 'excel_xirr_vectorized',
      title: 'Excel XIRR Vectorized (Batch IRR Calculation)',
      type: 'measure',
      sql: "{{ excel_formula_vectorized('=XIRR({cash_flows}, {dates})') }}",
      description: 'Calculate internal rate of return for multiple portfolios/funds in one batch operation using Excel XIRR function.',
      aggregationType: 'none',
      accessType: 'read-only',
      category: 'Private Markets',
      subcategory: 'IRR',
      financial_calc: {
        type: 'excel_formula',
        formula: '=XIRR({cash_flows}, {dates})',
        arguments: {
          cash_flows: 'ARRAY_AGG(ARRAY_AGG(cash_flow) OVER (PARTITION BY portfolio_id))',
          dates: 'ARRAY_AGG(ARRAY_AGG(transaction_date) OVER (PARTITION BY portfolio_id))'
        }
      },
      backendEndpoint: '/api/fabric/financial/excel'
    }
];