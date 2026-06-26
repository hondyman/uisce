# Cube.js Multi-Tenant Semantic Layer Integration

## Overview

This integration implements a **production-ready, multi-tenant Cube.js semantic layer** following industry best practices for:

- **Multi-source data tiering**: StarRocks (hot/real-time) + Trino (cold/historical)
- **Mandatory tenant isolation**: Row-level security (RLS) with `queryRewrite`
- **Resource QoS**: Per-tenant query queues via `context_to_app_id`
- **HA pre-aggregations**: StarRocks replaces Cube Store for high availability
- **Governance**: `CUBEJS_ROLLUP_ONLY` protects the Trino/Parquet lake

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Cube.js (Port 4000)                     │
│  Universal Semantic Layer + Query Router                     │
│  - REST API, GraphQL, SQL (PostgreSQL wire protocol)        │
│  - Tenant-aware via headers: X-Tenant-ID, X-Datasource-ID  │
└─────────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
        ▼                   ▼                   ▼
┌──────────────┐    ┌──────────────┐   ┌──────────────┐
│  StarRocks   │    │    Trino     │   │  StarRocks   │
│  (Hot Tier)  │    │ (Cold Tier)  │   │ (Pre-aggs)   │
│              │    │              │   │              │
│ Real-time    │    │ Historical   │   │ Materialized │
│ Operational  │    │ Parquet/     │   │ Rollups      │
│ Data (90d)   │    │ Iceberg Lake │   │ (HA Cache)   │
└──────────────┘    └──────────────┘   └──────────────┘
```

## Quick Start

### 1. Start the Stack

```bash
# Start all services including Cube.js
docker compose up -d

# Wait for Cube.js to be healthy
docker compose ps cube

# Initialize StarRocks pre-aggregation database
docker exec -i starrocks-fe mysql -uroot < cube/init-starrocks-preaggs.sql
```

### 2. Verify Cube.js is Running

```bash
# Health check
curl http://localhost:4000/readyz

# Check meta endpoint (requires tenant headers)
curl -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
     -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
     -H "Authorization: dev-secret-change-in-production" \
     http://localhost:4000/cubejs-api/v1/meta
```

### 3. Example Query via REST API

```bash
curl -X POST http://localhost:4000/cubejs-api/v1/load \
  -H "Content-Type: application/json" \
  -H "Authorization: dev-secret-change-in-production" \
  -H "X-Tenant-ID: 00000000-0000-0000-0000-000000000000" \
  -H "X-Tenant-Datasource-ID: 11111111-1111-1111-1111-111111111111" \
  -d '{
    "query": {
      "measures": ["Trades.count", "Trades.total_notional"],
      "dimensions": ["Trades.symbol"],
      "timeDimensions": [{
        "dimension": "Trades.event_time",
        "granularity": "day",
        "dateRange": ["2024-01-01", "2024-12-31"]
      }],
      "order": {
        "Trades.total_notional": "desc"
      },
      "limit": 10
    }
  }'
```

### 4. Connect via SQL (PostgreSQL Protocol)

```bash
# Using psql
psql -h localhost -p 15432 -U cube

# Example query
SELECT 
  "Trades.symbol" as symbol,
  SUM("Trades.total_notional") as total_notional
FROM "Trades"
WHERE "Trades.event_time" >= '2024-01-01'
GROUP BY "Trades.symbol"
ORDER BY total_notional DESC
LIMIT 10;
```

## Tenant Scope Automation

Cube now reads tenant + datasource metadata from `cube/generated/tenant-scopes.json`, which is generated from the platform tables. Regenerate it whenever tenants, resource groups, or schema overrides change:

```bash
# Ensure DATABASE_URL / ALPHA_DB_URL points at the admin Postgres
make sync-cube-tenants
```

The sync script:

- Queries `tenant_product_datasource` for every active tenant scope.
- Writes `cube/generated/tenant-scopes.json` (used for `scheduledRefreshContexts`, QoS routing, and header validation).
- Materializes schema overrides from `schema_overrides` JSON into `cube/schema/tenants/<tenant>/<datasource>/auto/*.yml` so Cube’s `repositoryFactory` automatically overlays them on top of the base schema.

`cube.js` falls back to a safe default scope if the file is missing, but that mode disables tenant-aware scheduled refreshes and resource-group tagging, so always run the sync before deploying.

## Multi-Tenancy Architecture

### Tenant Isolation (RLS)

Every query automatically gets tenant filters injected via `queryRewrite`:

```javascript
// In cube.js
queryRewrite: (query, { securityContext }) => {
  return {
    ...query,
    filters: [
      ...query.filters,
      {
        member: 'tenant_id',
        operator: 'equals',
        values: [securityContext.tenant_id]
      }
    ]
  };
}
```

**Result**: All queries are automatically scoped to the tenant, preventing cross-tenant data leakage.

### QoS Isolation

Per-tenant query queues and caching via `context_to_app_id`:

```javascript
contextToAppId: ({ securityContext }) => {
  return `tenant_${securityContext.tenant_id}_ds_${securityContext.datasource_id}`;
}
```

**Result**: Heavy queries from Tenant A won't slow down Tenant B (noisy neighbor prevention).

### Tenant-Specific Schemas

The `repository_factory` allows per-tenant schema customization:

```
cube/schema/
├── Trades.yml                    # Base schema
├── HistoricalTrades.yml         # Base schema
└── tenants/
    ├── tenant-a/
    │   └── Trades.yml           # Custom overrides for Tenant A
    └── tenant-b/
        └── Trades.yml           # Custom overrides for Tenant B
```

## Observability & SLO Metrics

- Cube exposes a Prometheus endpoint at `http://localhost:4000/metrics` via the built-in middleware. The payload includes `cube_query_total`, `cube_query_duration_seconds`, `cube_preaggregation_usage_total`, and gauges for latency/rollup SLO targets plus refresh context inventory.
- Prometheus already scrapes the `cubejs` job (`cube:4000/metrics`) from `prometheus/prometheus.yml`. Restart the stack after enabling metrics so Prometheus discovers the target.
- Grafana dashboard `grafana/dashboards/slo-sli-dashboard.json` now contains dedicated Cube panels: query throughput, p95 latency overlayed with the `cube_latency_slo_threshold_seconds` gauge, rollup hit rate vs `cube_rollup_hit_slo_target_percent`, and refresh context counts for visibility into scheduled workers.
- Adjust SLOs by exporting `CUBE_SLO_LATENCY_THRESHOLD_SECONDS` or `CUBE_SLO_ROLLUP_TARGET_PERCENT` before `docker compose up`. The gauges and dashboard lines update automatically because the values are emitted directly to Prometheus.

## Data Source Routing

### Hot Tier (StarRocks)

For real-time, operational data:

```yaml
cubes:
  - name: Trades
    data_source: starrocks  # Route to StarRocks
    refresh_key:
      every: 5 minutes      # Frequent refresh
```

### Cold Tier (Trino)

For historical, analytical queries:

```yaml
cubes:
  - name: HistoricalTrades
    data_source: trino      # Route to Trino/Iceberg
    refresh_key:
      every: 1 day          # Infrequent refresh
```

## Pre-Aggregations Strategy

### StarRocks as HA Cache

All pre-aggregations are materialized in StarRocks (not Cube Store):

```yaml
pre_aggregations:
  - name: trades_by_day
    type: rollup
    measures: [count, total_notional]
    dimensions: [symbol, side]
    time_dimension: event_time
    granularity: day
    external: true          # Store in StarRocks
    refresh_key:
      every: 1 hour
```

**Benefits**:
- High availability (StarRocks BE cluster)
- Faster queries (MySQL protocol is faster than HTTP)
- No Cube Store OSS limitations

### Rollup-Only Mode (Governance)

```bash
CUBEJS_ROLLUP_ONLY=true
```

**Effect**: Queries that can't be satisfied by a pre-aggregation **fail** instead of hitting the raw Trino lake. This protects your Parquet backend from expensive ad-hoc queries.

## Go Backend Integration

### Using the Cube Client

```go
import (
    "github.com/hondyman/semlayer/backend/internal/cube"
    "github.com/google/uuid"
)

// Initialize client
cubeClient := cube.NewClient("http://localhost:4000", "dev-secret")

// Build tenant context
tenantCtx := cube.TenantContext{
    TenantID:     uuid.MustParse("00000000-0000-0000-0000-000000000000"),
    DatasourceID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
    UserID:       "user@example.com",
}

// Execute query
query := cube.BuildQuery(
    "Trades",
    []string{"Trades.count", "Trades.total_notional"},
    []string{"Trades.symbol"},
    nil,
)

result, err := cubeClient.ExecuteQuery(ctx, query, tenantCtx)
if err != nil {
    log.Fatalf("Query failed: %v", err)
}

// Process results
for _, row := range result.Data {
    symbol := row["Trades.symbol"]
    notional := row["Trades.total_notional"]
    fmt.Printf("Symbol: %s, Notional: %v\n", symbol, notional)
}
```

## BI Tool Connectivity

### Tableau / Power BI

1. Install PostgreSQL connector
2. Connection details:
   - Host: `localhost`
   - Port: `15432`
   - Database: `cube`
   - User: `cube`
   - Password: (leave blank)

3. Add custom SQL in connection settings to set tenant context:
   ```sql
   SET SESSION cube.tenant_id = 'your-tenant-id';
   ```

### Excel / ODBC

Use PostgreSQL ODBC driver with same connection details.

## Schema Definition Guide

### Cube YAML Structure

```yaml
cubes:
  - name: YourCube
    sql: SELECT * FROM your_table
    title: Display Name
    description: Description
    
    # Route to appropriate backend
    data_source: starrocks  # or 'trino'
    
    # Refresh strategy
    refresh_key:
      every: 10 minutes
    
    dimensions:
      - name: tenant_id
        sql: tenant_id
        type: string
        public: false      # Hide from end users
      
      - name: your_dimension
        sql: column_name
        type: string
        
    measures:
      - name: your_measure
        sql: column_name
        type: sum
        format: currency
    
    pre_aggregations:
      - name: rollup_name
        type: rollup
        measures: [your_measure]
        dimensions: [your_dimension]
        external: true    # Store in StarRocks
```

### Supported Data Types

- **Dimensions**: `string`, `number`, `time`, `boolean`, `geo`
- **Measures**: `count`, `sum`, `avg`, `min`, `max`, `count_distinct`
- **Time Granularities**: `second`, `minute`, `hour`, `day`, `week`, `month`, `quarter`, `year`

## Performance Tuning

### 1. Pre-Aggregation Strategy

- **Hot path**: Refresh every 5-15 minutes
- **Warm path**: Refresh every 1-4 hours
- **Cold path**: Refresh daily/weekly

### 2. Partition Pre-Aggregations

```yaml
pre_aggregations:
  - name: partitioned_rollup
    partition_granularity: month  # Partition by month
    time_dimension: event_time
    refresh_key:
      every: 1 hour
```

### 3. Monitor Pre-Aggregation Health

```sql
-- Query StarRocks monitoring view
USE cube_preaggs;
SELECT * FROM v_preagg_health WHERE tenant_id = 'your-tenant-id';
```

## Security Considerations

### 1. Production API Secret

```bash
# Generate strong secret
export CUBE_API_SECRET=$(openssl rand -hex 32)
```

### 2. JWT Authentication (Production)

Update `cube.js`:

```javascript
checkAuth: async (req, authorization) => {
  const jwt = require('jsonwebtoken');
  const token = authorization.replace('Bearer ', '');
  
  const decoded = jwt.verify(token, process.env.JWT_SECRET);
  
  return {
    tenant_id: decoded.tenant_id,
    datasource_id: decoded.datasource_id,
    user_id: decoded.sub
  };
}
```

### 3. Database Credentials

Store in environment variables or secrets manager, never in code.

## Monitoring & Observability

### Health Checks

```bash
# Cube.js health
curl http://localhost:4000/readyz

# StarRocks FE health
curl http://localhost:8030/api/health
```

### Query Performance

Access Cube.js Dev Tools:
```bash
# Only in dev mode
open http://localhost:4000
```

### Logs

```bash
# Cube.js logs
docker logs -f cube-semantic-layer

# StarRocks FE logs
docker logs -f starrocks-fe
```

## Troubleshooting

### Query Returns Empty

**Cause**: Missing tenant headers or incorrect tenant_id

**Fix**: Ensure `X-Tenant-ID` and `X-Tenant-Datasource-ID` headers are set.

### Pre-Aggregation Not Used

**Cause**: Query doesn't match any pre-aggregation definition

**Fix**: Use `/dry-run` endpoint to see which pre-aggregations match:

```bash
curl -X POST http://localhost:4000/cubejs-api/v1/dry-run \
  -H "Authorization: dev-secret" \
  -H "X-Tenant-ID: your-tenant-id" \
  -d '{"query": {...}}'
```

### Rollup-Only Error

**Cause**: No pre-aggregation exists for the query

**Solution**: Either create a pre-aggregation or disable `CUBEJS_ROLLUP_ONLY` for development.

## Cost Optimization

### 1. Right-size Pre-Aggregations

Only materialize frequently accessed aggregations:

```yaml
# Good: Saves 90% of query time
pre_aggregations:
  - name: daily_summary
    granularity: day

# Bad: Rarely accessed, wastes storage
pre_aggregations:
  - name: second_level_detail
    granularity: second
```

### 2. Use Incremental Refresh

For large datasets:

```yaml
pre_aggregations:
  - name: incremental_rollup
    type: rollup
    refresh_key:
      every: 1 hour
      incremental: true
      update_window: 7 days  # Only update last 7 days
```

## Comparison: Cube.js OSS vs. Cube Store vs. StarRocks

| Feature | Cube Store OSS | StarRocks (Our Setup) |
|---------|----------------|----------------------|
| High Availability | ❌ No | ✅ Yes (BE cluster) |
| Multi-tenancy | ⚠️ Limited | ✅ Native (resource groups) |
| Protocol | HTTP | MySQL (faster) |
| Cost | Free | Free (OSS) |
| Ops Complexity | Medium | Low (already deployed) |

## Next Steps

1. **Initialize tenant data**: Load sample data into StarRocks/Iceberg
2. **Create tenant-specific schemas**: Add customizations in `cube/schema/tenants/`
3. **Set up refresh schedule**: Configure `scheduledRefreshContexts` to fetch tenant list from Postgres
4. **Enable JWT auth**: Update `checkAuth` for production
5. **Monitor performance**: Set up dashboards for query latency and pre-aggregation health

## Support

- Cube.js Docs: https://cube.dev/docs
- StarRocks Docs: https://docs.starrocks.io
- Trino Docs: https://trino.io/docs/current/

## License

This integration follows your existing project license. Cube.js is Apache 2.0.
