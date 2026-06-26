# Wealth Management Analytics Guide

This guide covers the comprehensive wealth management metrics and calculations added to the semantic layer, focusing on risk-adjusted returns, benchmarking, and portfolio analytics.

## 🎯 Risk-Adjusted Return Metrics

### Sharpe Ratio
```yaml
measures:
  - name: sharpe_ratio
    sql: "{{ sharpe_ratio([0.08, 0.06, 0.09, -0.03, 0.07], 0.025) }}"
    type: number
```
**Formula**: (Portfolio Return - Risk-Free Rate) / Standard Deviation
**Purpose**: Measures excess return per unit of volatility

### Sortino Ratio
```yaml
measures:
  - name: sortino_ratio
    sql: "{{ sortino_ratio([0.08, 0.06, 0.09, -0.03, 0.07], 0.025, 0.05) }}"
    type: number
```
**Formula**: (Portfolio Return - Risk-Free Rate) / Downside Deviation
**Purpose**: Focuses only on downside volatility

### Information Ratio
```yaml
measures:
  - name: information_ratio
    sql: "{{ information_ratio([0.08, 0.06, 0.09, -0.03, 0.07], [0.07, 0.05, 0.08, -0.02, 0.06]) }}"
    type: number
```
**Formula**: (Portfolio Return - Benchmark Return) / Tracking Error
**Purpose**: Measures active return per unit of active risk

## 📊 Risk Metrics

### Value at Risk (VaR)
```yaml
measures:
  - name: var_95
    sql: "{{ value_at_risk([0.08, 0.06, 0.09, -0.03, 0.07], 0.95) }}"
    type: number
```
**Purpose**: Estimates maximum potential loss over a period
**Confidence**: 95% (default), 99% available

### Conditional VaR (CVaR)
```yaml
measures:
  - name: cvar_95
    sql: "{{ conditional_var([0.08, 0.06, 0.09, -0.03, 0.07], 0.95) }}"
    type: number
```
**Purpose**: Expected loss given that VaR threshold is breached

### Downside Deviation
```yaml
measures:
  - name: downside_deviation
    sql: "{{ downside_deviation([0.08, 0.06, 0.09, -0.03, 0.07], 0.05) }}"
    type: number
```
**Purpose**: Measures volatility of negative returns only

## 📈 Benchmarking Metrics

### Alpha (Jensen's Alpha)
```yaml
measures:
  - name: jensen_alpha
    sql: "{{ jensens_alpha([0.08, 0.06, 0.09, -0.03, 0.07], [0.07, 0.05, 0.08, -0.02, 0.06], 0.025) }}"
    type: number
```
**Purpose**: Measures excess return not explained by market risk

### Beta
```yaml
measures:
  - name: portfolio_beta
    sql: "{{ beta([0.08, 0.06, 0.09, -0.03, 0.07], [0.07, 0.05, 0.08, -0.02, 0.06]) }}"
    type: number
```
**Purpose**: Measures sensitivity to market movements

### R-Squared
```yaml
measures:
  - name: r_squared
    sql: "{{ r_squared([0.08, 0.06, 0.09, -0.03, 0.07], [0.07, 0.05, 0.08, -0.02, 0.06]) }}"
    type: number
```
**Purpose**: Percentage of portfolio variation explained by benchmark

## 📊 Capture Ratios

### Upside Capture Ratio
```yaml
measures:
  - name: upside_capture
    sql: "{{ upside_capture_ratio([0.08, 0.06, 0.09, -0.03, 0.07], [0.07, 0.05, 0.08, -0.02, 0.06]) }}"
    type: number
```
**Purpose**: How portfolio performs when benchmark is up
**Target**: > 100% (outperforms in good times)

### Downside Capture Ratio
```yaml
measures:
  - name: downside_capture
    sql: "{{ downside_capture_ratio([0.08, 0.06, 0.09, -0.03, 0.07], [0.07, 0.05, 0.08, -0.02, 0.06]) }}"
    type: number
```
**Purpose**: How portfolio performs when benchmark is down
**Target**: < 100% (loses less in bad times)

## 🏢 Private Equity Metrics

### MOIC (Multiple on Invested Capital)
```yaml
measures:
  - name: moic
    sql: "{{ moic([50, 30, 20], [100, 0, 0], 75) }}"
    type: number
```
**Formula**: (Distributions + Residual Value) / Total Invested Capital
**Purpose**: Total value generated per dollar invested

### TVPI (Total Value to Paid-In Capital)
```yaml
measures:
  - name: tvpi
    sql: "{{ tvpi([50, 30, 20], [100, 0, 0], 75) }}"
    type: number
```
**Formula**: (Distributions + Residual Value) / Paid-In Capital
**Purpose**: Total value relative to capital called

### DPI (Distributions to Paid-In Capital)
```yaml
measures:
  - name: dpi
    sql: "{{ dpi([50, 30, 20], [100, 0, 0]) }}"
    type: number
```
**Formula**: Cumulative Distributions / Paid-In Capital
**Purpose**: Realized returns (cash received)

### RVPI (Residual Value to Paid-In Capital)
```yaml
measures:
  - name: rvpi
    sql: "{{ rvpi(75, [100, 0, 0]) }}"
    type: number
```
**Formula**: Net Asset Value / Paid-In Capital
**Purpose**: Unrealized value (on-paper gains)

### PME (Public Market Equivalent)
```yaml
measures:
  - name: pme_vs_sp500
    sql: "{{ pme([50, 30, 20], [100, 0, 0], [0.08, 0.06, 0.09, -0.03, 0.07]) }}"
    type: number
```
**Purpose**: Compares PE performance to public market equivalent
**Interpretation**: > 1.0 = Outperformed public market

## 📊 Portfolio Analytics

### Portfolio Volatility
```yaml
measures:
  - name: portfolio_volatility
    sql: "{{ portfolio_volatility([0.4, 0.3, 0.2, 0.1], [0.15, 0.20, 0.25, 0.30], [[1.0, 0.5, 0.3, 0.2], [0.5, 1.0, 0.4, 0.3], [0.3, 0.4, 1.0, 0.5], [0.2, 0.3, 0.5, 1.0]]) }}"
    type: number
```
**Purpose**: Calculates portfolio volatility considering correlations

### Portfolio Sharpe Ratio
```yaml
measures:
  - name: portfolio_sharpe
    sql: "{{ portfolio_sharpe([0.4, 0.3, 0.2, 0.1], [0.08, 0.06, 0.09, 0.07], [0.15, 0.20, 0.25, 0.30], [[1.0, 0.5, 0.3, 0.2], [0.5, 1.0, 0.4, 0.3], [0.3, 0.4, 1.0, 0.5], [0.2, 0.3, 0.5, 1.0]], 0.025) }}"
    type: number
```
**Purpose**: Risk-adjusted return for multi-asset portfolio

## 🔧 Configuration Examples

### Tenant-Specific Wealth Management Setup
```json
{
  "tenant_params": {
    "wealth_tenant": {
      "default_risk_free_rate": 0.025,
      "default_benchmark": "S&P 500",
      "custom_metrics": {
        "risk_adjusted_alpha": {
          "formula": "alpha / (1 + beta)",
          "parameters": {}
        }
      }
    }
  }
}
```

### Data Quality Rules for Wealth Data
```json
{
  "data_quality_rules": {
    "portfolio_data": [
      {
        "name": "positive_aum",
        "type": "range",
        "severity": "error",
        "parameters": {
          "column": "aum",
          "min": 0
        }
      }
    ]
  }
}
```

## 📈 Performance Optimization

### Materialized Views for Wealth Analytics
```json
{
  "scaling_config": {
    "materialized_views": [
      {
        "name": "daily_risk_metrics",
        "refresh_type": "incremental",
        "refresh_schedule": "0 6 * * *"
      }
    ]
  }
}
```

### Partitioning Strategy
```json
{
  "partitioning": [
    {
      "table": "portfolio_returns",
      "column": "date",
      "type": "range",
      "granularity": "month"
    }
  ]
}
```

## 🎯 Use Cases

### 1. Portfolio Performance Attribution
- Track alpha generation by asset class
- Measure contribution to overall portfolio risk
- Compare performance vs. benchmarks

### 2. Risk Management Dashboard
- Real-time VaR and CVaR calculations
- Stress testing scenarios
- Risk factor attribution

### 3. Client Reporting
- Personalized performance reports
- Risk-adjusted return comparisons
- Benchmark-relative performance

### 4. Investment Strategy Evaluation
- Backtesting portfolio strategies
- Scenario analysis
- Optimization recommendations

## 🚀 Integration

All metrics are available in templates:
```yaml
# Access wealth management metrics
{% set wm_metrics = get_wealth_management_metrics("wealth_portfolio") %}
{% set risk_metrics = get_risk_metrics("wealth_portfolio") %}

# Use in calculations
measures:
  - name: composite_score
    sql: "{{ wm_metrics[0].sharpe_ratio }} * {{ risk_metrics[0].var_95 }}"
```

This comprehensive suite provides everything needed for sophisticated wealth management analytics, from basic performance measurement to advanced risk management and portfolio optimization.

## 🧮 Additional Wealth Management Metrics

### Portfolio Efficiency & Structure

#### Expense Ratio
```yaml
measures:
  - name: expense_ratio
    sql: "{{ expense_ratio(150000, 10000000) }}"
    type: number
    format: percent
```
**Formula**: Total Fund Expenses / Average Assets Under Management
**Purpose**: Measures cost efficiency of fund management

#### Turnover Ratio
```yaml
measures:
  - name: turnover_ratio
    sql: "{{ turnover_ratio(5000000, 4500000, 10000000) }}"
    type: number
    format: percent
```
**Formula**: Lesser of Purchases or Sales / Average Portfolio Value
**Purpose**: Assesses trading activity and potential tax implications

#### Liquidity Ratio
```yaml
measures:
  - name: liquidity_ratio
    sql: "{{ liquidity_ratio(2500000, 10000000) }}"
    type: number
    format: percent
```
**Formula**: Liquid Assets / Total Portfolio Value
**Purpose**: Measures portfolio's ability to meet short-term obligations

### Tax-Aware Performance Metrics

#### Tax Drag
```yaml
measures:
  - name: tax_drag
    sql: "{{ tax_drag(0.12, 0.09) }}"
    type: number
    format: percent
```
**Formula**: Pre-Tax Return - After-Tax Return
**Purpose**: Quantifies return lost to taxes

#### Effective Tax Rate on Gains
```yaml
measures:
  - name: effective_tax_rate
    sql: "{{ effective_tax_rate(18000, 200000) }}"
    type: number
    format: percent
```
**Formula**: Taxes Paid / Realized Gains
**Purpose**: Measures actual tax burden on investment gains

### Goal-Based Planning Metrics

#### Funding Ratio
```yaml
measures:
  - name: funding_ratio
    sql: "{{ funding_ratio(8500000, 8000000) }}"
    type: number
    format: percent
```
**Formula**: Present Value of Assets / Present Value of Liabilities
**Purpose**: Assesses whether goals/obligations are adequately funded

#### Probability of Success (Monte Carlo)
```yaml
measures:
  - name: probability_of_success
    sql: "{{ probability_of_success([50000, 55000, 60000, 65000, 70000], [0.02, 0.025, 0.03], [10000000, 10500000, 11000000, 11500000, 12000000], 10000) }}"
    type: number
    format: percent
```
**Purpose**: Estimates likelihood of meeting future goals through simulation
**Parameters**: Goal cash flows, inflation assumptions, asset projections, simulation count

### Behavioral & Strategic Metrics

#### Behavior Gap
```yaml
measures:
  - name: behavior_gap
    sql: "{{ behavior_gap(0.11, 0.08) }}"
    type: number
    format: percent
```
**Formula**: Portfolio Return - Investor Return
**Purpose**: Measures cost of poor timing decisions

#### Diversification Score
```yaml
measures:
  - name: diversification_score
    sql: "{{ diversification_score({'equity': 0.6, 'fixed_income': 0.3, 'alternatives': 0.1}, {'us': 0.7, 'international': 0.3}, {'technology': 0.25, 'healthcare': 0.2, 'financials': 0.15, 'consumer': 0.4}, {'growth': 0.5, 'value': 0.3, 'momentum': 0.2}) }}"
    type: number
```
**Purpose**: Quantifies portfolio diversification across asset classes, geographies, sectors, and factors
**Parameters**: Weight maps for different diversification dimensions
