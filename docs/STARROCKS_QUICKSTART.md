# StarRocks Lakehouse Quick Start

## TL;DR

Replace ClickHouse with StarRocks OSS to query Iceberg directly. One engine, one source of truth.

```bash
# Start the StarRocks stack
docker compose -f docker-compose.starrocks.yml up -d

# Verify StarRocks is healthy
curl http://localhost:8030/api/health

# Connect via MySQL client
mysql -h 127.0.0.1 -P 9030 -u root

# Query Iceberg directly
SELECT * FROM iceberg_catalog.wealth.trades LIMIT 10;
```

## Architecture Decision

| Aspect | ClickHouse (Before) | StarRocks + Iceberg (After) |
|--------|---------------------|----------------------------|
| **Hot Data** | ClickHouse tables | StarRocks materialized views |
| **Cold Data** | Separate Iceberg | Same Iceberg tables |
| **Data Sync** | CDC pipeline needed | Not needed (single store) |
| **Multi-tenancy** | Row policies | Resource groups + partitions |
| **Query Protocol** | HTTP/Native | MySQL (simpler clients) |
| **Governance** | Two systems | One Iceberg catalog |

## Files Created

| File | Purpose |
|------|---------|
| `docker-compose.starrocks.yml` | StarRocks + Nessie stack |
| `backend/internal/analytics/starrocks_client.go` | Go client for StarRocks |
| `backend/internal/analytics/starrocks_init.sql` | Catalog + materialized views |
| `backend/internal/analytics/iceberg_lakehouse_schema.sql` | Iceberg table definitions |
| `docs/STARROCKS_LAKEHOUSE_MIGRATION.md` | Full migration guide |

## Quick Commands

```bash
# Start the stack
docker compose -f docker-compose.starrocks.yml up -d

# Stop
docker compose -f docker-compose.starrocks.yml down

# View StarRocks FE logs
docker logs starrocks-fe -f

# View BE logs
docker logs starrocks-be -f

# Connect to StarRocks
mysql -h 127.0.0.1 -P 9030 -u root

# Access MinIO console
open http://localhost:9003  # minioadmin/minioadmin

# Access Nessie UI (if available)
open http://localhost:19120
```

## Verify Setup

```sql
-- In StarRocks MySQL client:

-- Check catalogs
SHOW CATALOGS;

-- Check Iceberg catalog
USE iceberg_catalog;
SHOW DATABASES;

-- Check resource groups
SHOW RESOURCE GROUPS;

-- Check materialized views
SHOW MATERIALIZED VIEWS;

-- Run a test query
SELECT COUNT(*) FROM iceberg_catalog.wealth.trades 
WHERE tenant_id = 'tenant_001';
```

## Go Client Usage

```go
import "github.com/yourorg/semlayer/backend/internal/analytics"

// Create client
client, err := analytics.NewStarRocksClient(analytics.StarRocksConfig{
    Host:         "localhost",
    Port:         9030,
    User:         "root",
    Password:     "",
    CatalogName:  "iceberg_catalog",
    DatabaseName: "wealth",
})
if err != nil {
    log.Fatal(err)
}
defer client.Close()

// Query trades
trades, err := client.QueryTrades(ctx, "tenant_001", 60) // last 60 minutes

// Query daily P&L (uses materialized view)
pnl, err := client.QueryDailyPnL(ctx, "tenant_001", "portfolio_123", 30) // last 30 days
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `STARROCKS_DSN` | `root:@tcp(starrocks-fe:9030)/` | StarRocks connection string |
| `ICEBERG_CATALOG_URI` | `http://nessie:19120/api/v1` | Nessie catalog URI |
| `ICEBERG_WAREHOUSE` | `s3://lakehouse/warehouse` | S3 path for Iceberg data |
| `S3_ENDPOINT` | `http://minio:9000` | MinIO/S3 endpoint |
| `AWS_ACCESS_KEY_ID` | `minioadmin` | S3 access key |
| `AWS_SECRET_ACCESS_KEY` | `minioadmin` | S3 secret key |

## Handling 200M+ Trades/Day

### Partitioning Strategy

```sql
-- Iceberg table partitioned by day + tenant bucket
CREATE TABLE iceberg.wealth.trades (
    ...
)
WITH (
    partitioning = ARRAY[
        'day(event_time)',      -- Prune by date
        'bucket(tenant_id, 32)' -- Spread tenant data
    ],
    sorted_by = ARRAY['portfolio_id', 'event_time']
);
```

### Compaction (run daily via Spark/Trino)

```sql
CALL iceberg.system.rewrite_data_files(
    table => 'wealth.trades',
    strategy => 'sort',
    options => map('target-file-size-bytes', '268435456')
);
```

### Multi-Tenancy

```sql
-- Create resource groups for tenant isolation
CREATE RESOURCE GROUP tenant_premium WITH (
    cpu_weight = 100,
    mem_limit = '40%',
    concurrency_limit = 100
);

-- Assign users
SET PROPERTY FOR 'tenant_001' 'resource_group' = 'tenant_premium';
```

## Migration Checklist

- [ ] Deploy `docker-compose.starrocks.yml`
- [ ] Create Iceberg tables (via Trino/Spark)
- [ ] Backfill historical data from ClickHouse
- [ ] Update Go code to use `StarRocksClient`
- [ ] Update environment variables in deployments
- [ ] Remove ClickHouse from docker-compose.yml
- [ ] Update CI/CD workflows
- [ ] Run benchmark queries

## When to Keep ClickHouse

Consider keeping ClickHouse **only** if you need:

1. **< 10ms P99** on specific hot queries
2. **100k+ QPS** on simple aggregations  
3. **Real-time log analytics** separate from lakehouse

For most wealth management use cases, StarRocks + Iceberg provides sufficient performance with simpler operations.

## Support

- [StarRocks Docs](https://docs.starrocks.io/)
- [StarRocks Iceberg Catalog](https://docs.starrocks.io/docs/data_source/catalog/iceberg_catalog/)
- [Apache Iceberg](https://iceberg.apache.org/)
- [Nessie Catalog](https://projectnessie.org/)
