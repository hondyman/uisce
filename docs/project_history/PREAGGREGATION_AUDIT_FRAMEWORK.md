# Semantic-Layer Preaggregation Audit Framework

## Overview

This document outlines the comprehensive preaggregation audit framework implemented for the private markets semantic layer. The framework analyzes Excel-based metrics across LP, GP, and FoF bundles to determine optimal preaggregation strategies.

## Audit Results Summary

### Bundle Analysis

| Bundle | Total Metrics | Preaggregated | On-Demand | Est. Storage | Compute Cost |
|--------|---------------|---------------|-----------|--------------|--------------|
| LP Private Markets | 7 | 2 | 5 | 100 MB | Medium |
| GP Private Markets | 5 | 3 | 2 | 150 MB | Low |
| FoF Private Markets | 5 | 1 | 4 | 50 MB | Medium |

### Preaggregation Recommendations

#### LP Bundle (Limited Partner Focus)
**Preaggregated Metrics:**
- **Net IRR** (`private_markets_net_irr`)
  - Grain: `["fund_id", "month"]`
  - Refresh: Daily
  - Reason: High-frequency IRR queries, complex XIRR calculation
- **XIRR** (`private_markets_xirr`)
  - Grain: `["fund_id", "month"]`
  - Refresh: Daily
  - Reason: High-frequency IRR queries, complex XIRR calculation

**On-Demand Metrics:**
- TVPI, RVPI, DPI, PME, J-Curve (complex calculations, lower query frequency)

#### GP Bundle (General Partner Focus)
**Preaggregated Metrics:**
- **Gross IRR** (`private_markets_gross_irr`)
  - Grain: `["fund_id", "month"]`
  - Refresh: Daily
  - Reason: High-frequency performance monitoring
- **Gross MOIC** (`private_markets_gross_moic`)
  - Grain: `["fund_id", "quarter"]`
  - Refresh: Weekly
  - Reason: Stable quarterly grain, multiple dashboard usage
- **Fee Ratio** (`private_markets_fee_ratio`)
  - Grain: `["fund_id", "month"]`
  - Refresh: Daily
  - Reason: High-frequency fee monitoring
- **Deployment Pace** (`private_markets_deployment_pace`)
  - Grain: `["fund_id", "month"]`
  - Refresh: Daily
  - Reason: Key operational metric

**On-Demand Metrics:**
- Carry Accrual (high volatility, real-time calculations)

#### FoF Bundle (Fund of Funds Focus)
**Preaggregated Metrics:**
- **Net IRR** (`private_markets_net_irr`)
  - Grain: `["portfolio_id", "month"]`
  - Refresh: Daily
  - Reason: High-frequency portfolio performance monitoring

**On-Demand Metrics:**
- TVPI, PME, Diversification Score, Risk-Adjusted Return (complex calculations, benchmark dependencies)

## Preaggregation Decision Framework

### Complexity Scoring
- **Low (1-2)**: Simple SUM, COUNT operations
- **Medium (3-4)**: Basic Excel functions (SUMPRODUCT, AVERAGE)
- **High (5+)**: Complex functions (XIRR, CORREL, STDEV.P)

### Query Frequency Classification
- **High**: Performance metrics, fees, operational KPIs
- **Medium**: Multiples, diversification scores
- **Low**: Correlation analysis, complex risk metrics

### Data Volatility Assessment
- **Low**: Historical allocations, stable reference data
- **Medium**: Performance metrics, monthly calculations
- **High**: Real-time fees, volatile market data

### Preaggregation Rules
1. **Preaggregate** if: High frequency + Low complexity + Low/medium volatility
2. **On-demand** if: High complexity OR high volatility OR low frequency
3. **Grain Selection**: Monthly for high-frequency, quarterly for stable metrics
4. **Refresh Schedule**: Daily for operational, weekly for strategic metrics

## Implementation Architecture

### Semantic Model Structure
```go
type PreaggregatedMetric struct {
    Value           float64    `json:"value"`
    Grain           []string   `json:"grain"`
    LastRefresh     time.Time  `json:"last_refresh"`
    RefreshSchedule string     `json:"refresh_schedule"`
    SourceFormula   string     `json:"source_formula"`
}
```

### Precomputation Functions
Each preaggregated metric gets a dedicated precomputation function:
```go
func PrecomputeNetIRR(ctx context.Context, grain []string) error {
    // Excel formula execution
    // Data aggregation
    // Storage in semantic layer
    return nil
}
```

### Storage Strategy
- **Materialized Views**: For high-frequency metrics
- **Incremental Updates**: For daily refreshes
- **Partitioning**: By grain dimensions (fund_id, portfolio_id, date)
- **Compression**: For historical data

### Governance Integration
- **Audit Trail**: Track precomputation runs and data quality
- **Data Quality Checks**: Validate against source calculations
- **Refresh Monitoring**: Alert on failed precomputations
- **Cost Tracking**: Monitor storage and compute usage

## Cost-Benefit Analysis

### Storage Costs
- **Low**: < 100 MB per bundle
- **Medium**: 100-500 MB per bundle
- **High**: > 500 MB per bundle

### Compute Costs
- **Low**: Simple aggregations, daily refreshes
- **Medium**: Complex calculations, weekly refreshes
- **High**: Real-time calculations, complex Excel functions

### Performance Benefits
- **Query Speed**: 10-100x faster for preaggregated metrics
- **Resource Efficiency**: Reduced compute for repeated queries
- **Scalability**: Better handling of concurrent users

## Migration Strategy

### Phase 1: Foundation (Week 1-2)
1. Implement precomputation infrastructure
2. Create semantic model structures
3. Set up refresh scheduling

### Phase 2: Core Metrics (Week 3-4)
1. Deploy high-priority preaggregated metrics
2. Update bundle configurations
3. Test query performance improvements

### Phase 3: Optimization (Week 5-6)
1. Monitor usage patterns
2. Adjust grains and refresh schedules
3. Implement cost optimization

### Phase 4: Governance (Week 7-8)
1. Deploy audit and monitoring
2. Set up alerting and reporting
3. Document operational procedures

## Monitoring and Maintenance

### Key Metrics to Track
- Query performance (P95 latency)
- Precomputation success rate
- Storage utilization
- Compute cost per bundle
- Data freshness (time since last refresh)

### Alerting Rules
- Precomputation failures
- Data freshness > 24 hours
- Query performance degradation
- Storage utilization > 80%

### Maintenance Tasks
- Weekly: Review preaggregation effectiveness
- Monthly: Optimize refresh schedules
- Quarterly: Audit metric usage and costs

## Conclusion

The preaggregation audit framework provides a data-driven approach to optimizing the semantic layer for private markets analytics. By selectively preaggregating high-value, high-frequency metrics while keeping complex calculations on-demand, we achieve optimal performance and cost efficiency.

The framework is designed to be:
- **Automated**: Rules-based decision making
- **Governed**: Audit trails and quality checks
- **Scalable**: Easy addition of new metrics
- **Maintainable**: Clear operational procedures

This approach ensures that Excel-powered metrics remain both powerful and performant within the semantic layer architecture.
