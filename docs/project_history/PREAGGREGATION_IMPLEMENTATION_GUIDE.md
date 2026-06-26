# Preaggregation Implementation Guide

This guide explains how to implement and use the semantic-layer preaggregation system for Excel-powered financial metrics.

## Overview

The preaggregation system provides:
- **Automated calculation** of complex Excel formulas
- **Fast query performance** through precomputed metrics
- **Data quality monitoring** and freshness tracking
- **Scheduled refresh** with cron-based automation
- **Audit trails** for governance and compliance

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Raw Data      │ -> │ Preaggregation   │ -> │ Semantic Layer  │
│   Sources       │    │ Service          │    │ (Precomputed)   │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────────┐
                       │   Scheduler      │
                       │   (Cron Jobs)    │
                       └──────────────────┘
```

## Quick Start

### 1. Database Setup

Run the schema creation script:

```bash
psql -d your_database -f backend/internal/models/preaggregation_schema.sql
```

### 2. Run Preaggregation Demo

```bash
cd backend/cmd/preaggregation
go run main.go
```

This will:
- Connect to your database
- Run all preaggregation jobs
- Display results summary
- Show scheduler status

## Preaggregated Metrics

### LP Bundle Metrics

| Metric | Node ID | Grain | Refresh | Purpose |
|--------|---------|-------|---------|---------|
| Net IRR | `private_markets_net_irr` | fund_id, month | Daily | Performance monitoring |
| XIRR | `private_markets_xirr` | fund_id, month | Daily | Extended IRR with irregular cash flows |

### GP Bundle Metrics

| Metric | Node ID | Grain | Refresh | Purpose |
|--------|---------|-------|---------|---------|
| Gross IRR | `private_markets_gross_irr` | fund_id, month | Daily | GP performance tracking |
| Gross MOIC | `private_markets_gross_moic` | fund_id, quarter | Weekly | Quarterly reporting |
| Fee Ratio | `private_markets_fee_ratio` | fund_id, month | Daily | Fee monitoring |
| Deployment Pace | `private_markets_deployment_pace` | fund_id, month | Daily | Operational KPIs |

### FoF Bundle Metrics

| Metric | Node ID | Grain | Refresh | Purpose |
|--------|---------|-------|---------|---------|
| Net IRR | `private_markets_net_irr` | portfolio_id, month | Daily | Portfolio performance |

## Usage Examples

### Query Preaggregated Metrics

```sql
-- Get latest Net IRR for a specific fund
SELECT * FROM semantic_layer.get_preaggregated_metric(
    'private_markets_net_irr',
    '{"fund_id": "FUND001", "month": "2024-01"}'::jsonb,
    24 -- max age in hours
);
```

### Check Data Quality

```sql
-- Get data quality summary for last 7 days
SELECT * FROM semantic_layer.get_data_quality_summary(7);
```

### Manual Preaggregation

```go
// Initialize service
preaggService := services.NewPreaggregationService(db)

// Precompute Net IRR
ctx := context.Background()
grain := []string{"fund_id", "month"}
err := preaggService.PrecomputeNetIRR(ctx, grain)
```

### Automated Scheduling

```go
// Initialize scheduler
scheduler := services.NewPreaggregationScheduler(preaggService)
scheduler.Start()

// Run specific job manually
err := scheduler.RunJobManually("net_irr_daily")
```

## Configuration

### Database Connection

Update the connection string in your demo:

```go
db, err := sql.Open("postgres", "postgres://user:password@localhost/semlayer?sslmode=disable")
```

### Refresh Schedules

Modify schedules in the database:

```sql
UPDATE semantic_layer.refresh_schedules
SET schedule_expression = '0 8 * * *'  -- 8 AM UTC
WHERE metric_node_id = 'private_markets_net_irr';
```

## Monitoring & Maintenance

### Key Metrics to Monitor

1. **Preaggregation Success Rate**
   ```sql
   SELECT job_name, status, COUNT(*)
   FROM semantic_layer.preaggregation_audit
   WHERE started_at >= NOW() - INTERVAL '7 days'
   GROUP BY job_name, status;
   ```

2. **Data Freshness**
   ```sql
   SELECT node_id, AVG(EXTRACT(EPOCH FROM (NOW() - last_refresh))/3600) as avg_hours_old
   FROM semantic_layer.preaggregated_metrics
   GROUP BY node_id;
   ```

3. **Query Performance**
   ```sql
   SELECT node_id, COUNT(*) as query_count
   FROM semantic_layer.preaggregated_metrics
   WHERE last_refresh >= NOW() - INTERVAL '1 hour'
   GROUP BY node_id;
   ```

### Maintenance Tasks

#### Daily
- Monitor preaggregation job success
- Check data freshness (< 24 hours for daily metrics)
- Validate data quality scores

#### Weekly
- Review preaggregation performance
- Optimize slow-running calculations
- Update refresh schedules if needed

#### Monthly
- Audit data quality trends
- Review storage utilization
- Plan for new metrics to preaggregate

## Troubleshooting

### Common Issues

1. **Preaggregation Job Failures**
   - Check database connectivity
   - Verify source data availability
   - Review error logs in audit table

2. **Stale Data**
   - Verify cron job scheduling
   - Check system time synchronization
   - Monitor job execution times

3. **Performance Issues**
   - Add database indexes
   - Consider data partitioning
   - Optimize query grains

### Debug Commands

```bash
# Check recent job executions
SELECT * FROM semantic_layer.preaggregation_audit
WHERE started_at >= NOW() - INTERVAL '1 day'
ORDER BY started_at DESC;

# Verify data freshness
SELECT node_id, last_refresh, NOW() - last_refresh as age
FROM semantic_layer.preaggregated_metrics
ORDER BY age DESC;

# Check data quality
SELECT metric_id, check_type, check_value, status
FROM semantic_layer.data_quality_monitoring
WHERE checked_at >= NOW() - INTERVAL '1 day';
```

## Performance Optimization

### Database Tuning

1. **Indexes**
   ```sql
   CREATE INDEX CONCURRENTLY idx_preagg_composite
   ON semantic_layer.preaggregated_metrics(node_id, last_refresh, grain_values);
   ```

2. **Partitioning** (for large datasets)
   ```sql
   CREATE TABLE semantic_layer.preaggregated_metrics_y2024 PARTITION OF semantic_layer.preaggregated_metrics
   FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');
   ```

3. **Query Optimization**
   - Use JSONB operators for grain filtering
   - Implement query result caching
   - Consider materialized views for complex aggregations

### Application Tuning

1. **Batch Processing**
   - Process metrics in batches
   - Implement parallel processing for independent calculations
   - Use connection pooling for database access

2. **Memory Management**
   - Stream large result sets
   - Implement garbage collection hints
   - Monitor memory usage in production

## Security Considerations

1. **Access Control**
   - Implement row-level security on preaggregated tables
   - Audit all data access
   - Encrypt sensitive financial data

2. **Data Validation**
   - Validate input data before preaggregation
   - Implement business rule checks
   - Monitor for data anomalies

## Future Enhancements

### Planned Features

1. **Real-time Preaggregation**
   - Trigger-based updates for critical metrics
   - Streaming data processing integration

2. **Machine Learning Integration**
   - Predictive preaggregation based on usage patterns
   - Anomaly detection for data quality

3. **Multi-cloud Deployment**
   - Cross-region data replication
   - Global load balancing for queries

### Extending the System

To add new preaggregated metrics:

1. **Define the metric** in your bundle JSON
2. **Implement the precomputation function** following the existing patterns
3. **Add database queries** for data extraction
4. **Configure refresh scheduling** in the database
5. **Update monitoring dashboards** for the new metric

## Support

For issues or questions:
1. Check the troubleshooting section above
2. Review audit logs in the database
3. Monitor system performance metrics
4. Contact the platform engineering team

## Appendix

### Excel Formula Reference

| Function | Purpose | Complexity |
|----------|---------|------------|
| XIRR | Internal Rate of Return | High |
| SUMPRODUCT | Vector multiplication | Medium |
| CORREL | Correlation coefficient | High |
| STDEV.P | Population standard deviation | Medium |
| SUM/COUNT | Basic aggregation | Low |

### Grain Reference

| Grain Level | Use Case | Refresh Frequency |
|-------------|----------|-------------------|
| fund_id, month | Operational monitoring | Daily |
| fund_id, quarter | Strategic reporting | Weekly |
| portfolio_id, month | Portfolio analysis | Daily |
| portfolio_id, quarter | Performance attribution | Weekly |
