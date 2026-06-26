# Financial Calculations in Semantic Layer

This document demonstrates advanced financial calculation capabilities including XIRR, IRR, NPV, and other investment analysis functions integrated into the semantic layer.

## 🎯 **Financial Functions Overview**

### **Core Financial Calculations**

| Function | Description | Parameters | Use Case |
|----------|-------------|------------|----------|
| **XIRR** | Extended Internal Rate of Return | `values[], dates[], guess` | Non-periodic cash flows |
| **IRR** | Internal Rate of Return | `values[], guess` | Periodic cash flows |
| **NPV** | Net Present Value | `rate, values[]` | Investment valuation |
| **FV** | Future Value | `rate, periods, payment, pv` | Investment projection |
| **PV** | Present Value | `rate, periods, payment, fv` | Asset valuation |
| **PMT** | Payment Amount | `rate, periods, pv, fv` | Loan payments |

### **Risk & Performance Metrics**

| Metric | Formula | Description |
|--------|---------|-------------|
| **Sharpe Ratio** | `(Avg Return - Risk-Free) / StdDev * √252` | Risk-adjusted returns |
| **Sortino Ratio** | `(Avg Return - Risk-Free) / Downside StdDev * √252` | Downside risk focus |
| **Maximum Drawdown** | `Min(Peak-to-Trough / Peak)` | Worst-case loss |

## 📊 **Usage Examples**

### **XIRR Calculation**
```yaml
measures:
  - name: portfolio_xirr
    sql: "{{ xirr([-10000, 2750, 4250, 3250, 2750], ['2008-01-01', '2008-03-01', '2008-10-30', '2009-02-15', '2009-04-01'], 0.1) }}"
    type: number
    format: percent
    financial_calc:
      type: "xirr"
      cash_flows:
        - amount: -10000
          date: "2008-01-01"
          category: "investment"
        - amount: 2750
          date: "2008-03-01"
          category: "return"
        - amount: 4250
          date: "2008-10-30"
          category: "return"
        - amount: 3250
          date: "2009-02-15"
          category: "return"
        - amount: 2750
          date: "2009-04-01"
          category: "return"
      guess: 0.1
```

### **IRR Calculation**
```yaml
measures:
  - name: quarterly_returns_irr
    sql: "{{ irr([-10000, 2750, 4250, 3250, 2750], 0.1) }}"
    type: number
    format: percent
    financial_calc:
      type: "irr"
      cash_flows:
        - amount: -10000
          period: 0
          category: "investment"
        - amount: 2750
          period: 1
          category: "return"
        - amount: 4250
          period: 2
          category: "return"
        - amount: 3250
          period: 3
          category: "return"
        - amount: 2750
          period: 4
          category: "return"
      guess: 0.1
```

### **NPV Calculation**
```yaml
measures:
  - name: project_npv
    sql: "{{ npv(0.1, [-100000, 30000, 40000, 50000, 60000]) }}"
    type: number
    format: currency
    financial_calc:
      type: "npv"
      rate: 0.1
      cash_flows:
        - amount: -100000
          period: 0
          category: "investment"
        - amount: 30000
          period: 1
          category: "return"
        - amount: 40000
          period: 2
          category: "return"
        - amount: 50000
          period: 3
          category: "return"
        - amount: 60000
          period: 4
          category: "return"
```

### **Loan Payment Calculation**
```yaml
measures:
  - name: monthly_loan_payment
    sql: "{{ pmt(0.05/12, 60, 100000, 0) }}"
    type: number
    format: currency
    financial_calc:
      type: "pmt"
      rate: 0.05
      periods: 60
      present_value: 100000
      future_value: 0
```

## 🔧 **Template Functions**

### **Available Financial Functions**
```gonja
{# XIRR - Extended Internal Rate of Return #}
{{ xirr([-10000, 2750, 4250, 3250, 2750], ['2008-01-01', '2008-03-01', '2008-10-30', '2009-02-15', '2009-04-01'], 0.1) }}

{# IRR - Internal Rate of Return #}
{{ irr([-10000, 2750, 4250, 3250, 2750], 0.1) }}

{# NPV - Net Present Value #}
{{ npv(0.1, [-100000, 30000, 40000, 50000, 60000]) }}

{# FV - Future Value #}
{{ fv(0.08, 10, 0, -50000) }}

{# PV - Present Value #}
{{ pv(0.06, 20, 0, 100000) }}

{# PMT - Payment Amount #}
{{ pmt(0.05/12, 60, 100000, 0) }}
```

## 📈 **Performance Optimization**

### **Database-Level Optimizations**
```yaml
performance_hints:
  - type: "index"
    table: "investments"
    columns: ["investment_date", "investor_type", "investment_id"]
    description: "Index for time-series financial queries"

  - type: "partition"
    table: "cash_flows"
    columns: ["cash_flow_date"]
    parameters:
      partition_type: "monthly"
    description: "Monthly partitioning for cash flow data"

  - type: "cache"
    table: "financial_metrics"
    parameters:
      ttl: "1h"
      size: "500MB"
    description: "Cache financial calculations for 1 hour"
```

### **Materialized Views for Complex Calculations**
```yaml
materialized_views:
  - name: "portfolio_performance_mv"
    refresh_type: "incremental"
    refresh_schedule: "daily"
    partition_by: "calculation_date"
    cluster_by: "portfolio_id"
```

## 🎯 **Advanced Financial Analytics**

### **Portfolio Performance**
```yaml
measures:
  - name: portfolio_xirr
    sql: "XIRR(ARRAY_AGG(CASE WHEN amount < 0 THEN amount END), ARRAY_AGG(investment_date))"
    type: number
    format: percent
    description: "Portfolio XIRR across all investments"

  - name: sharpe_ratio
    sql: "(AVG(daily_return) - 0.02/252) / STDDEV(daily_return) * SQRT(252)"
    type: number
    format: "decimal"
    description: "Risk-adjusted return measure"

  - name: maximum_drawdown
    sql: "MIN(running_total / GREATEST(running_max, 0.0001) - 1)"
    type: number
    format: percent
    description: "Maximum peak-to-trough decline"
```

### **Risk Analysis**
```yaml
measures:
  - name: value_at_risk
    sql: "PERCENTILE_CONT(0.05) WITHIN GROUP (ORDER BY daily_return)"
    type: number
    format: percent
    description: "5% Value at Risk"

  - name: expected_shortfall
    sql: "AVG(daily_return) FILTER (WHERE daily_return <= PERCENTILE_CONT(0.05) WITHIN GROUP (ORDER BY daily_return))"
    type: number
    format: percent
    description: "Expected shortfall beyond VaR"
```

## 🚀 **Integration Examples**

### **API Usage**
```bash
# Update context with financial calculations
curl -X POST http://localhost:3000/update-context \
  -H "Content-Type: application/json" \
  -d @financial_context.json

# Render financial analytics template
curl -X POST http://localhost:3000/render \
  -H "Content-Type: application/json" \
  -d '{"template_name": "financial_analytics"}'
```

### **Multi-Tenant Financial Analytics**
```yaml
cubes:
  - name: "{{ tenant_id }}_financial_portfolio"
    sql_table: "{{ CUBE }}_{{ COMPILE_CONTEXT.securityContext.tenant_id }}.portfolio_data"
    data_source: "{{ get_data_source(CUBE) }}"

    # Tenant-specific financial calculations
    measures:
      - name: tenant_portfolio_xirr
        sql: "{{ xirr(get_tenant_cash_flows(tenant_id), get_tenant_dates(tenant_id), 0.1) }}"
        type: number
        format: percent
```

## 📊 **Real-World Use Cases**

### **Investment Portfolio Tracking**
- Track IRR/XIRR for individual investments and portfolios
- Calculate Sharpe ratios and other risk metrics
- Monitor maximum drawdown and recovery times

### **Loan Portfolio Analysis**
- Calculate payment schedules and remaining balances
- Analyze loan performance and default risk
- Track NPV of loan portfolios

### **Financial Planning**
- Project future values of investments
- Calculate present values for asset valuations
- Analyze annuity and retirement planning scenarios

### **Risk Management**
- Value at Risk (VaR) calculations
- Expected shortfall analysis
- Stress testing with different scenarios

## 🔧 **Implementation Details**

### **XIRR Algorithm**
The XIRR function uses an iterative approach to solve:
```
Σ [Pi / (1 + r)^((di - d1)/365)] = 0
```

Where:
- `Pi` = cash flow amount at period i
- `di` = date of cash flow i
- `d1` = date of first cash flow
- `r` = internal rate of return (what we're solving for)

### **Performance Considerations**
- Complex financial calculations are cached where possible
- Materialized views pre-compute expensive operations
- Database indexes optimize date-based queries
- Partitioning improves query performance on large datasets

This implementation provides enterprise-grade financial calculation capabilities that rival dedicated financial analysis tools while maintaining the flexibility and performance of your semantic layer.</content>
<parameter name="filePath">/Users/eganpj/GitHub/semlayer/cube-gonja/FINANCIAL_CALCULATIONS_GUIDE.md
