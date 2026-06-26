# DAX-to-SQL Mapping for Power BI DirectQuery

## Overview

This directory contains SQL views generated from your unified financial services semantic layer's DAX formulas. These views enable Power BI DirectQuery to leverage DAX-like calculations without DirectQuery limitations.

## 🎯 Problem Solved

- **Before**: Power BI DirectQuery couldn't handle complex DAX functions
- **After**: Pre-computed SQL views provide the same calculations with full DirectQuery compatibility

## 📁 Generated Files

### Individual Metric Views
- `net_interest_margin.sql` - Banking NIM calculation
- `sharpe_ratio.sql` - Risk-adjusted return metric
- `value_at_risk.sql` - Market risk measurement
- ... and 77+ other financial metrics

### Master Script
- `create_all_views.sql` - Run all view creations in sequence

## 🚀 Implementation Steps

### 1. Database Setup
```sql
-- Create the views in your database
-- Run the master script:
psql -d your_database -f create_all_views.sql
-- or
sqlcmd -S your_server -d your_database -i create_all_views.sql
```

### 2. Power BI Configuration

#### Option A: Direct Table Import
1. Connect Power BI to your database using DirectQuery
2. Import the generated views as tables
3. Use simple measures like `SUM(net_interest_margin[value])`

#### Option B: Semantic Layer Integration
1. Add views to your existing semantic model
2. Create relationships with dimension tables
3. Build reports using the pre-computed metrics

### 3. Performance Optimization
```sql
-- Create indexes for better DirectQuery performance
CREATE INDEX idx_entity_date ON net_interest_margin(entity_id, as_of_date);
CREATE INDEX idx_portfolio_returns ON sharpe_ratio(entity_id, as_of_date);

-- For complex calculations, consider materialized views
CREATE MATERIALIZED VIEW mv_complex_metrics AS
SELECT * FROM beta_coefficient
UNION ALL
SELECT * FROM sortino_ratio;
```

## 📊 DirectQuery Compatibility Matrix

| Compatibility | Count | Examples | Notes |
|---------------|-------|----------|-------|
| High | 70+ | Ratios, simple aggregations | Full DirectQuery support |
| Medium | 5-7 | Statistical functions | May need optimization |
| Low | 1-2 | Complex correlations | Consider pre-computation |

## 🔧 Database-Specific Adaptations

### PostgreSQL
```sql
-- Use GREATEST instead of MAX for impairment
CREATE VIEW impairment_loss_incurred AS
SELECT GREATEST(0, ca.carrying_amount - ra.recoverable_amount) AS value
FROM carrying_amount ca
JOIN recoverable_amount ra ON ca.entity_id = ra.entity_id;
```

### SQL Server
```sql
-- Use schema-qualified names
CREATE VIEW [dbo].[net_interest_margin] AS
SELECT (SUM(ii.amount) - SUM(ie.amount)) / AVG(a.average_balance) AS value
FROM [dbo].[interest_income] ii
JOIN [dbo].[interest_expense] ie ON ii.entity_id = ie.entity_id;
```

### Oracle
```sql
-- Use Oracle-specific functions
CREATE VIEW net_interest_margin AS
SELECT (SUM(ii.amount) - SUM(ie.amount)) / AVG(a.average_balance) AS value
FROM interest_income ii, interest_expense ie, assets a
WHERE ii.entity_id = ie.entity_id
  AND ii.entity_id = a.entity_id;
```

## 🧪 Testing & Validation

### Query Folding Verification
```sql
-- Check if Power BI queries fold to single SQL statements
-- Look for "DirectQuery: Query folded to SQL" in Power BI performance analyzer
```

### Accuracy Testing
```sql
-- Compare DAX results with SQL view results
-- Example:
SELECT
    dax_result.net_interest_margin,
    sql_result.value,
    ABS(dax_result.net_interest_margin - sql_result.value) as difference
FROM dax_calculated_results dax_result
JOIN net_interest_margin sql_result ON dax_result.entity_id = sql_result.entity_id;
```

## 📈 Advanced Usage Patterns

### 1. Composite Models
- Use DirectQuery for large fact tables
- Import calculated views for complex metrics
- Combine both in Power BI for optimal performance

### 2. Incremental Refresh
```sql
-- Set up incremental refresh on views
-- Partition by date ranges for better performance
```

### 3. Calculation Groups
- Create calculation groups in Power BI
- Reference the SQL views as base measures
- Apply time intelligence and other transformations

## ⚡ Performance Best Practices

1. **Indexing Strategy**
   - Index on `entity_id`, `as_of_date`
   - Composite indexes for common filter combinations
   - Consider covering indexes for frequently queried columns

2. **Materialized Views**
   - Use for complex statistical calculations
   - Refresh on schedule or on-demand
   - Monitor refresh performance

3. **Query Optimization**
   - Avoid correlated subqueries
   - Use CTEs for complex logic
   - Test with realistic data volumes

## 🔍 Troubleshooting

### Common Issues

1. **Query Not Folding**
   - Check view complexity
   - Simplify JOIN conditions
   - Use database-specific optimizations

2. **Performance Issues**
   - Add missing indexes
   - Consider materialized views
   - Review query execution plans

3. **Schema Mismatches**
   - Verify table/column names
   - Update view definitions
   - Recreate views after schema changes

## 📚 Related Documentation

- [Power BI DirectQuery Best Practices](https://docs.microsoft.com/en-us/power-bi/connect-data/desktop-directquery-about)
- [SQL Server Performance Tuning](https://docs.microsoft.com/en-us/sql/relational-databases/performance/)
- [PostgreSQL Query Optimization](https://www.postgresql.org/docs/current/performance-tips.html)

## 🎯 Next Steps

1. **Deploy Views**: Run the master script in your database
2. **Test Connection**: Verify Power BI can connect via DirectQuery
3. **Build Reports**: Create reports using the new metrics
4. **Monitor Performance**: Use Power BI performance analyzer
5. **Optimize**: Add indexes and materialized views as needed

---

**Generated on**: September 13, 2025
**Source**: Unified Financial Services Super Bundle
**Metrics**: 82 DAX formulas converted to SQL views
**Compatibility**: 95%+ DirectQuery compatible
