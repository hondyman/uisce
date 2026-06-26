# Insurance Calculation Library

This document provides a comprehensive overview of the insurance calculation templates available in the SemLayer semantic calculation library.

## 📊 Overview

The insurance calculation library covers four main categories:

- **Underwriting**: Core profitability and risk metrics
- **Reserving**: Loss reserve adequacy and development
- **Solvency**: Regulatory capital requirements
- **Profitability**: Overall financial performance

## 🧮 Calculation Categories

### Underwriting Metrics

#### 1. Loss Ratio
**Formula**: `SUM(claim_amount) / SUM(premium_amount)`
- **Purpose**: Measures underwriting profitability
- **Target**: < 100% indicates profitable underwriting
- **Use Case**: Monitor core insurance profitability

#### 2. Expense Ratio
**Formula**: `SUM(expenses) / SUM(premium_amount)`
- **Purpose**: Measures operating efficiency
- **Target**: Minimize while maintaining service quality
- **Use Case**: Cost management and efficiency analysis

#### 3. Combined Ratio
**Formula**: `(SUM(claim_amount) + SUM(expenses)) / SUM(premium_amount)`
- **Purpose**: Overall underwriting profitability
- **Target**: < 100% indicates profitable operations
- **Use Case**: Comprehensive profitability assessment

#### 4. Net Claims Ratio
**Formula**: `SUM(net_claims) / SUM(net_premiums)`
- **Purpose**: Claims net of reinsurance recoveries
- **Use Case**: True underwriting profitability after reinsurance

#### 5. Claim Frequency
**Formula**: `COUNT(claims) / COUNT(policies)`
- **Purpose**: Rate of claims per policy
- **Use Case**: Risk assessment and pricing

#### 6. Claim Severity
**Formula**: `SUM(claim_amount) / COUNT(claims)`
- **Purpose**: Average cost per claim
- **Use Case**: Loss cost analysis and reserving

#### 7. Retention Rate
**Formula**: `COUNT(renewed_policies) / COUNT(eligible_renewals)`
- **Purpose**: Customer loyalty metric
- **Use Case**: Customer satisfaction and growth planning

### Reserving Metrics

#### 8. Reserve Adequacy Ratio
**Formula**: `SUM(actual_claims_paid) / SUM(reserves_held)`
- **Purpose**: Sufficiency of loss reserves
- **Target**: ≈ 100% indicates adequate reserving
- **Use Case**: Reserve adequacy testing

#### 9. Reserve Development Ratio
**Formula**: `SUM(current_year_reserves) / SUM(prior_year_reserves)`
- **Purpose**: Reserve changes over time
- **Use Case**: Long-tail liability monitoring

#### 10. Loss Reserve Leverage
**Formula**: `SUM(loss_reserves) / SUM(policyholder_surplus)`
- **Purpose**: Exposure to reserving errors
- **Target**: Monitor leverage levels
- **Use Case**: Risk management and capital planning

### Solvency Metrics

#### 11. Solvency Margin Ratio
**Formula**: `SUM(available_solvency_margin) / SUM(required_solvency_margin)`
- **Purpose**: Regulatory capital adequacy
- **Target**: > 100% meets regulatory requirements
- **Use Case**: Regulatory compliance and capital management

### Profitability Metrics

#### 12. Operating Ratio
**Formula**: `(SUM(claim_amount) + SUM(expenses) - SUM(investment_income)) / SUM(premium_amount)`
- **Purpose**: Full operating performance
- **Target**: < 100% indicates profitable operations
- **Use Case**: Comprehensive profitability analysis

#### 13. Investment Yield
**Formula**: `SUM(investment_income) / SUM(invested_assets)`
- **Purpose**: Return on invested assets
- **Use Case**: Investment portfolio performance

#### 14. Premium Growth Rate
**Formula**: `(SUM(current_premium_amount) - SUM(prior_premium_amount)) / SUM(prior_premium_amount)`
- **Purpose**: Top-line growth momentum
- **Use Case**: Business growth monitoring

## 🔧 Technical Implementation

### Frontend Integration

The calculations are available in the `financialCalculations.ts` file:

```typescript
import { libraryOptions } from '../components/UnifiedSemanticBuilder/financialCalculations';

// Filter for insurance calculations
const insuranceCalcs = libraryOptions.filter(calc => calc.category === 'Insurance');
```

### Backend API Usage

All calculations use the standard calculation endpoint:

```json
{
  "node_id": "insurance_loss_ratio",
  "node_type": "calculation",
  "domain": "insurance",
  "category": "underwriting",
  "subcategory": "loss_ratio",
  "financial_calc": {
    "type": "ratio",
    "numerator": "SUM(claim_amount)",
    "denominator": "SUM(premium_amount)"
  }
}
```

### Database Schema

The calculations expect data in the following tables:
- `insurance_policies`: Policy information and premiums
- `insurance_claims`: Claims data and payments
- `insurance_expenses`: Operating expenses
- `insurance_investments`: Investment portfolio data
- `insurance_reserves`: Loss reserve information

## 📈 Key Performance Indicators

### Underwriting KPIs
- **Loss Ratio**: Core profitability metric
- **Combined Ratio**: Overall efficiency
- **Expense Ratio**: Cost management

### Reserving KPIs
- **Reserve Adequacy**: Reserve sufficiency
- **Reserve Development**: Reserve accuracy over time

### Solvency KPIs
- **Solvency Margin**: Regulatory compliance
- **Reserve Leverage**: Risk exposure

### Profitability KPIs
- **Operating Ratio**: Total performance
- **Investment Yield**: Asset performance
- **Premium Growth**: Business growth

## 🎯 Use Cases

### 1. Underwriting Performance
Monitor loss ratios, expense ratios, and combined ratios to assess underwriting profitability and efficiency.

### 2. Reserve Management
Track reserve adequacy and development to ensure sufficient funds for future claims payments.

### 3. Regulatory Compliance
Monitor solvency margins to ensure compliance with regulatory capital requirements.

### 4. Investment Performance
Track investment yields and their contribution to overall profitability.

### 5. Customer Analytics
Monitor retention rates and claim patterns for customer segmentation and pricing.

## 📊 Sample Queries

### Monthly Loss Ratio Trend
```sql
SELECT
    DATE_TRUNC('month', claim_date) as month,
    SUM(claim_amount) / SUM(premium_amount) as loss_ratio
FROM insurance_claims c
JOIN insurance_policies p ON c.policy_id = p.policy_id
GROUP BY month
ORDER BY month;
```

### Reserve Development Analysis
```sql
SELECT
    development_year,
    SUM(CASE WHEN reserve_date = '2024-01-01' THEN amount END) as initial_reserve,
    SUM(CASE WHEN reserve_date = '2024-06-30' THEN amount END) as current_reserve,
    SUM(CASE WHEN reserve_date = '2024-06-30' THEN amount END) /
    SUM(CASE WHEN reserve_date = '2024-01-01' THEN amount END) as development_ratio
FROM insurance_reserves
GROUP BY development_year;
```

## 🔄 Integration with Semantic Layer

These calculations integrate seamlessly with the SemLayer semantic calculation framework:

1. **Template Registration**: All calculations are pre-registered as templates
2. **Dynamic SQL Generation**: Calculations generate appropriate SQL based on context
3. **Caching**: Results are cached for performance
4. **Governance**: Full audit trail and version control

## 🚀 Getting Started

1. **Load Sample Data**: Use `schemas/insurance_sample_data.sql`
2. **Register Templates**: Templates are auto-registered on startup
3. **API Access**: Use `/api/calc/run` endpoint
4. **Frontend Integration**: Available in Calculations Library UI

This comprehensive insurance calculation library provides everything needed for full insurance analytics and reporting.
