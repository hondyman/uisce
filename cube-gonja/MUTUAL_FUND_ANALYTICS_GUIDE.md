# Mutual Fund and Advanced Financial Analytics Guide

This guide covers the advanced financial calculations, mutual fund metrics, Greeks calculations, and scaling features added to the semantic layer.

## Weighted Averages

Calculate weighted averages for portfolio returns, allocations, and other metrics.

### Template Functions

- `weighted_average(weights, values)`: Calculate weighted average from arrays
- `weighted_average_sql(weight_col, value_col, table)`: Generate SQL for weighted average calculation

### Example Usage

```yaml
measures:
  - name: weighted_portfolio_return
    sql: "{{ weighted_average_sql('allocation_percentage', 'fund_return', 'portfolio_holdings') }}"
    type: number
    format: percent
```

## Greeks Calculations

Calculate option Greeks for derivatives analysis.

### Available Functions

- `delta(asset_price, strike_price, time_to_expiry, volatility, risk_free_rate, dividend_yield)`
- `gamma(asset_price, strike_price, time_to_expiry, volatility, risk_free_rate, dividend_yield)`
- `theta(asset_price, strike_price, time_to_expiry, volatility, risk_free_rate, dividend_yield)`
- `vega(asset_price, strike_price, time_to_expiry, volatility, risk_free_rate, dividend_yield)`
- `rho(asset_price, strike_price, time_to_expiry, volatility, risk_free_rate, dividend_yield)`

### Example Usage

```yaml
measures:
  - name: option_delta
    sql: "{{ delta(100.0, 105.0, 0.25, 0.2, 0.03, 0.02) }}"
    type: number
```

## Mutual Fund Metrics

Common mutual fund and portfolio performance metrics.

### Available Functions

- `sharpe_ratio(returns, risk_free_rate)`: Risk-adjusted return measure
- `sortino_ratio(returns, risk_free_rate, target_return)`: Downside risk-adjusted return
- `alpha(portfolio_returns, benchmark_returns, risk_free_rate)`: Excess return over benchmark
- `beta(portfolio_returns, benchmark_returns)`: Market sensitivity
- `max_drawdown(values)`: Maximum peak-to-trough decline
- `volatility(returns, annualize)`: Standard deviation of returns
- `tracking_error(portfolio_returns, benchmark_returns)`: Deviation from benchmark

### Example Usage

```yaml
measures:
  - name: portfolio_sharpe
    sql: "{{ sharpe_ratio([0.05, 0.03, 0.08, -0.02, 0.06], 0.02) }}"
    type: number
  - name: portfolio_volatility
    sql: "{{ volatility([0.05, 0.03, 0.08, -0.02, 0.06], true) }}"
    type: number
    format: percent
```

## Tenant-Specific Configuration

Configure tenant-specific calculation parameters and rules.

### Structure

```json
{
  "tenant_params": {
    "tenant_123": {
      "tenant_id": "tenant_123",
      "default_risk_free_rate": 0.025,
      "default_benchmark": "S&P 500 Total Return",
      "custom_metrics": {
        "custom_calculation": {
          "formula": "custom_formula",
          "parameters": {}
        }
      },
      "data_quality_rules": [],
      "performance_hints": []
    }
  }
}
```

## Data Quality Rules for Financial Data

Enhanced data quality validation for financial calculations.

### Financial-Specific Rules

- **Range Validation**: Ensure values are within expected ranges (e.g., positive NAV)
- **Completeness**: Required fields for financial calculations
- **Consistency**: Cross-field validation (e.g., total = sum of parts)
- **Accuracy**: Statistical outlier detection

### Example

```json
{
  "data_quality_rules": {
    "mutual_fund_portfolio": [
      {
        "name": "nav_positive",
        "type": "range",
        "severity": "error",
        "threshold": 0,
        "parameters": {
          "column": "nav_per_share",
          "min": 0
        }
      }
    ]
  }
}
```

## Scaling and Performance Optimization

### Materialized Views

Pre-computed aggregations for performance.

```json
{
  "scaling_config": {
    "materialized_views": [
      {
        "name": "daily_portfolio_performance",
        "refresh_type": "incremental",
        "refresh_schedule": "0 6 * * *",
        "partition_by": "date",
        "cluster_by": "fund_id"
      }
    ]
  }
}
```

### Partitioning

Optimize large dataset queries.

```json
{
  "partitioning": [
    {
      "table": "fund_returns",
      "column": "date",
      "type": "range",
      "granularity": "month"
    }
  ]
}
```

### Caching

Cache frequently accessed calculations.

```json
{
  "caching": [
    {
      "name": "portfolio_metrics_cache",
      "table": "portfolio_metrics",
      "ttl": 3600,
      "refresh_type": "auto",
      "refresh_schedule": "*/15 * * * *"
    }
  ]
}
```

## Custom Financial Metrics

Add your own financial calculations.

### Template Context Access

```yaml
# Access tenant-specific parameters
{% set tenant_config = get_tenant_params("tenant_123") %}
{% set risk_free = tenant_config.default_risk_free_rate %}

# Use in calculations
measures:
  - name: custom_metric
    sql: "({{ ctx.portfolio_return }} - {{ risk_free }}) / {{ ctx.volatility }}"
    type: number
```

### API Integration

Use the `/update-context` endpoint to configure tenant-specific parameters:

```bash
curl -X POST http://localhost:3000/update-context \
  -H "Content-Type: application/json" \
  -d @mutual_fund_context.json
```

## Performance Considerations

1. **Materialized Views**: Use for complex aggregations over large datasets
2. **Partitioning**: Implement on time-series data for query optimization
3. **Caching**: Cache expensive calculations with appropriate TTL
4. **Indexing**: Use performance hints to suggest optimal indexes
5. **Pre-aggregations**: Define in Cube.js schema for common metrics

## Example Complete Template

See `templates/mutual_fund_analytics.yml.gonja` for a complete example including:
- Mutual fund portfolio cube
- Options portfolio with Greeks
- Weighted average calculations
- Performance metrics
- Pre-aggregations for optimization
