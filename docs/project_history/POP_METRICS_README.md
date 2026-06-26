# Period-over-Period (PoP) Metrics System

## Overview

The PoP Metrics System provides comprehensive monitoring, anomaly detection, and governance for financial services metrics. Built for mutual fund companies and other financial institutions, it enables data stewards to monitor key performance indicators, detect anomalies, and maintain governance standards.

## Architecture

### Core Components

1. **PoP Handler** (`/backend/internal/handlers/pop_handler.go`)
   - RESTful API endpoints for metric operations
   - Direct database integration
   - Real-time anomaly detection

2. **Database Schema** (`/backend/migrations/000006_pop_analysis_schema.sql`)
   - Comprehensive metric definitions
   - Computation results storage
   - Anomaly detection and governance tracking

3. **Seed Data** (`/backend/migrations/000007_pop_seed_data.sql`)
   - Sample financial metrics for mutual fund company
   - Realistic computation results and anomalies

4. **Enhancement Functions** (`/backend/migrations/000008_pop_enhancement_functions.sql`)
   - Advanced anomaly detection algorithms
   - Data quality checks
   - Governance utilities

5. **Dashboard Configuration** (`/backend/migrations/000009_pop_cockpit_dashboard.sql`)
   - Complete cockpit setup
   - Alert configurations
   - User preferences

6. **Admin Scripts** (`/backend/migrations/000010_pop_admin_scripts.sql`)
   - System health checks
   - Bulk operations
   - Maintenance utilities

## API Endpoints

### Manifest and Discovery
```
GET /api/pop/manifest
```
Returns enriched PoP manifest with governance metadata, anomaly counts, and golden path status.

### Individual Metrics
```
GET /api/pop/metrics/:id
```
Retrieves full metric details including recent computations and anomaly status.

### Anomaly Analysis
```
POST /api/pop/metrics/:id/analyze
```
Runs anomaly detection on a specific metric. Supports multiple detection methods:
- `zscore`: Statistical Z-score based detection
- `iqr`: Interquartile range outlier detection
- `prophet`: Trend-based anomaly detection

### Governance Operations
```
POST /api/pop/metrics/:id/promote
```
Promotes a metric to golden path status with audit logging.

```
POST /api/pop/metrics/:id/flag
```
Flags an anomaly for steward review with severity classification.

```
POST /api/pop/metrics/:id/comment
```
Adds steward comments for audit trail and collaboration.

## Sample Metrics

### Financial Metrics
- **Total AUM**: Assets Under Management across all funds
- **NAV Growth Rate**: Net Asset Value monthly growth
- **Net Inflows/Outflows**: Capital flow monitoring
- **30-Day Volatility**: Risk measurement
- **Sharpe Ratio**: Risk-adjusted performance

### Operational Metrics
- **Transaction Volume**: Daily processing volume
- **Processing Time**: Average transaction processing time
- **Compliance Filing Status**: Regulatory compliance tracking

## Database Schema

### Core Tables

#### `pop_metrics`
Primary metric definitions with governance metadata:
```sql
- id: UUID (Primary Key)
- name: TEXT (Unique identifier)
- display_name: TEXT (Human-readable name)
- domain: TEXT (finance, operations, compliance, etc.)
- category: TEXT (assets, performance, risk, etc.)
- status: TEXT (draft, active, deprecated, golden)
- golden_path: BOOLEAN (Governance designation)
- SLA fields for data quality
- Audit fields (created_at, updated_at, etc.)
```

#### `pop_computations`
Period-over-period computation results:
```sql
- metric_id: UUID (Foreign Key)
- period_start/end: DATE
- current_value, previous_value: DECIMAL
- delta, percent_change: DECIMAL
- computation_status: TEXT
```

#### `pop_anomalies`
Anomaly detection results:
```sql
- metric_id: UUID (Foreign Key)
- anomaly_type: TEXT (z_score, iqr, trend_break)
- severity: TEXT (low, medium, high, critical)
- confidence: DECIMAL
- detection_method: TEXT
- status: TEXT (open, investigating, resolved)
```

#### `pop_steward_reviews`
Governance review sessions:
```sql
- metric_id: UUID (Foreign Key)
- reviewer_user_id: TEXT
- review_type: TEXT
- overall_rating: TEXT
- status: TEXT (in_progress, completed, overdue)
```

#### `pop_steward_comments`
Audit trail and collaboration:
```sql
- metric_id/review_id: UUID (Foreign Keys)
- commenter_user_id: TEXT
- comment_type: TEXT (general, anomaly_feedback, golden_path)
- comment_text: TEXT
```

## Dashboard Configuration

### Main Cockpit Dashboard
The system includes a comprehensive dashboard with:

1. **KPI Summary Cards**
   - Executive-level metrics overview
   - Trend indicators and targets

2. **Anomaly Heatmap**
   - Visual anomaly overview by domain
   - Severity-based color coding

3. **Trend Charts**
   - Historical performance visualization
   - Forecasting capabilities

4. **Risk Metrics Panel**
   - Real-time risk monitoring
   - Threshold-based alerting

5. **Operations Metrics**
   - Processing efficiency tracking
   - SLA compliance monitoring

6. **Governance Status**
   - Golden path compliance
   - Review status tracking

7. **Activity Feed**
   - Recent system activities
   - Audit trail summary

## Alert System

### Threshold-Based Alerts
- **High Volatility**: Triggers when 30-day volatility exceeds 20%
- **Negative Inflows**: Alerts on negative capital flows
- **SLA Breaches**: Processing time violations
- **Compliance Issues**: Filing deadline misses

### Notification Channels
- Dashboard notifications
- Email alerts
- Slack integration

## Installation and Setup

### 1. Database Setup
```bash
# Apply migrations in order
psql -d your_database < 000006_pop_analysis_schema.sql
psql -d your_database < 000007_pop_seed_data.sql
psql -d your_database < 000008_pop_enhancement_functions.sql
psql -d your_database < 000009_pop_cockpit_dashboard.sql
psql -d your_database < 000010_pop_admin_scripts.sql
```

### 2. Backend Integration
The PoP handler is already integrated into the main API router at `/backend/internal/api/api.go`. The system will automatically:
- Initialize the PoP handler with database connection
- Register all API endpoints
- Enable the cockpit dashboard

### 3. Frontend Integration
Connect your frontend application to the PoP API endpoints:

```javascript
// Get PoP manifest
const manifest = await fetch('/api/pop/manifest');

// Analyze metric for anomalies
const analysis = await fetch(`/api/pop/metrics/${metricId}/analyze`, {
  method: 'POST',
  body: JSON.stringify({ method: 'zscore' })
});

// Promote to golden path
await fetch(`/api/pop/metrics/${metricId}/promote`, {
  method: 'POST',
  body: JSON.stringify({
    user_id: 'steward@company.com',
    reason: 'Consistent performance'
  })
});
```

## System Health Checks

### Automated Monitoring
Use the built-in health check function:
```sql
SELECT * FROM pop_system_health_check();
```

This provides:
- Active metrics count
- Recent computation status
- Stale data detection
- Anomaly processing status
- Data quality metrics

### Maintenance Tasks
Regular maintenance functions:
```sql
-- Archive old anomalies
SELECT archive_old_anomalies(90);

-- Clean up orphaned records
SELECT * FROM cleanup_orphaned_records();

-- Refresh materialized views
SELECT refresh_pop_views();
```

## Security and Governance

### Access Control
- Role-based access to dashboards
- User-specific preferences
- Audit trail for all actions
- Compliance reporting

### Data Quality
- SLA monitoring for data freshness
- Completeness thresholds
- Automated quality checks
- Issue escalation workflows

## Performance Optimization

### Database Indexes
The schema includes optimized indexes for:
- Metric lookups by domain/status
- Computation queries by date ranges
- Anomaly filtering by severity
- Review status queries

### Query Optimization
- Efficient aggregation queries
- Minimal data retrieval
- Cached computations where appropriate

## Monitoring and Alerting

### System Metrics
Track system health with:
```sql
SELECT * FROM get_pop_system_metrics();
```

### Custom Alerts
Configure alerts for:
- Metric threshold violations
- Data quality issues
- SLA breaches
- Anomaly spikes

## Troubleshooting

### Common Issues

1. **Missing Data**
   - Check data source connections
   - Verify query permissions
   - Review SLA configurations

2. **High False Positives**
   - Adjust anomaly detection sensitivity
   - Review detection method parameters
   - Update baseline periods

3. **Performance Issues**
   - Check database indexes
   - Review query optimization
   - Monitor system resources

### Support
For issues or questions:
1. Check system health: `SELECT * FROM pop_system_health_check();`
2. Review recent logs in `pop_steward_comments`
3. Check anomaly patterns in `pop_anomalies`

## Future Enhancements

### Planned Features
- Machine learning-based anomaly detection
- Predictive analytics integration
- Advanced dashboard customization
- Mobile application support
- Real-time streaming analytics

### Integration Points
- External data source connectors
- Third-party analytics platforms
- Enterprise notification systems
- Compliance reporting tools

---

**Version**: 1.0.0
**Created**: September 10, 2025
**Last Updated**: September 10, 2025</content>
<parameter name="filePath">/Users/eganpj/GitHub/semlayer/POP_METRICS_README.md
