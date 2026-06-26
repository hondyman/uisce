# Cube.js Multi-Tenant Architecture - Visual Reference

## System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        CONSUMPTION LAYER                             │
│                                                                       │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │   Web App    │  │  BI Tools    │  │  AI Agents   │              │
│  │  (React)     │  │ (Tableau,    │  │  (GraphQL)   │              │
│  │              │  │  Power BI,   │  │              │              │
│  │  /api/cube/* │  │  Excel)      │  │  REST API    │              │
│  └──────┬───────┘  └──────┬───────┘  └──────┬───────┘              │
│         │                 │                  │                       │
│         │   PostgreSQL    │    HTTP/GraphQL  │                       │
│         │   Wire Protocol │                  │                       │
└─────────┼─────────────────┼──────────────────┼───────────────────────┘
          │                 │                  │
          │         ┌───────┴──────────────────┘
          │         │
┌─────────▼─────────▼──────────────────────────────────────────────────┐
│                     SEMANTIC LAYER (Cube.js)                          │
│                                                                        │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  Security Context (Every Request)                              │  │
│  │  • X-Tenant-ID: 00000000-0000-0000-0000-000000000000          │  │
│  │  • X-Datasource-ID: 11111111-1111-1111-1111-111111111111      │  │
│  │  • X-User-ID: user@example.com                                │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                        │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐              │
│  │ queryRewrite │  │context_to_   │  │ repository   │              │
│  │              │  │  app_id      │  │  Factory     │              │
│  │ Injects      │  │              │  │              │              │
│  │ tenant_id    │  │ Creates      │  │ Loads tenant-│              │
│  │ filter       │  │ isolated     │  │ specific     │              │
│  │ (RLS)        │  │ queues (QoS) │  │ schemas      │              │
│  └──────────────┘  └──────────────┘  └──────────────┘              │
│                                                                        │
│  ┌────────────────────────────────────────────────────────────────┐  │
│  │  Cube Schema (YAML)                                            │  │
│  │  • Trades.yml          (data_source: starrocks)               │  │
│  │  • HistoricalTrades.yml (data_source: trino)                  │  │
│  │  • PortfolioHoldings.yml (data_source: starrocks)            │  │
│  └────────────────────────────────────────────────────────────────┘  │
│                                                                        │
└────────┬──────────────────────┬──────────────────────┬────────────────┘
         │                      │                      │
         │ Hot Queries          │ Cold Queries         │ Pre-agg Refresh
         │                      │                      │
┌────────▼────────┐   ┌─────────▼────────┐   ┌────────▼────────┐
│   StarRocks     │   │      Trino       │   │   StarRocks     │
│   (Hot Tier)    │   │   (Cold Tier)    │   │  (Pre-aggs)     │
│                 │   │                  │   │                 │
│ • Real-time ops │   │ • Historical     │   │ • Materialized  │
│ • Last 90 days  │   │   analysis       │   │   rollups       │
│ • <200ms p95    │   │ • Parquet/Iceberg│   │ • HA storage    │
│ • MySQL protocol│   │ • 2-10s queries  │   │ • OLAP optimized│
└─────────────────┘   └──────────────────┘   └─────────────────┘
         │                      │                      
         │                      │                      
┌────────▼──────────────────────▼──────────────────────────────┐
│              Data Lake (Iceberg on MinIO/S3)                  │
│                                                                │
│  • Parquet files partitioned by tenant_id, date             │
│  • Nessie catalog for versioning                             │
│  • Single source of truth                                    │
└───────────────────────────────────────────────────────────────┘
```

## Request Flow with Tenant Isolation

```
┌─────────────┐
│  Frontend   │
│  (React)    │
└──────┬──────┘
       │
       │ 1. Request with tenant headers
       │    X-Tenant-ID: tenant-a
       │    X-Datasource-ID: ds-1
       ▼
┌─────────────────────────────────────────────┐
│  Go Backend (/api/cube/query)               │
│                                             │
│  • Validates tenant exists in Postgres     │
│  • Checks user permissions                 │
│  • Forwards to Cube.js with headers        │
└──────────────────┬──────────────────────────┘
                   │
                   │ 2. Proxied request
                   ▼
┌─────────────────────────────────────────────┐
│  Cube.js Security Middleware                │
│                                             │
│  checkAuth() {                              │
│    tenant_id = req.headers['x-tenant-id']  │
│    if (!tenant_id) throw Error             │
│    return { tenant_id, datasource_id }     │
│  }                                          │
└──────────────────┬──────────────────────────┘
                   │
                   │ 3. Build security context
                   ▼
┌─────────────────────────────────────────────┐
│  Cube.js Query Rewrite                      │
│                                             │
│  queryRewrite(query, { securityContext }) { │
│    return {                                 │
│      ...query,                              │
│      filters: [                             │
│        ...query.filters,                    │
│        {                                    │
│          member: 'tenant_id',               │
│          operator: 'equals',                │
│          values: [securityContext.tenant_id]│
│        }                                    │
│      ]                                      │
│    }                                        │
│  }                                          │
└──────────────────┬──────────────────────────┘
                   │
                   │ 4. Filtered query
                   ▼
┌─────────────────────────────────────────────┐
│  Query Router                                │
│                                             │
│  if (cube.data_source === 'starrocks') {   │
│    → Route to StarRocks                     │
│  } else if (cube.data_source === 'trino') {│
│    → Route to Trino                         │
│  }                                          │
└──────────────────┬──────────────────────────┘
                   │
       ┌───────────┴────────────┐
       │                        │
       ▼                        ▼
┌──────────────┐        ┌──────────────┐
│  StarRocks   │        │    Trino     │
│              │        │              │
│ SELECT *     │        │ SELECT *     │
│ FROM trades  │        │ FROM trades  │
│ WHERE        │        │ WHERE        │
│  tenant_id=  │        │  tenant_id=  │
│  'tenant-a'  │        │  'tenant-a'  │
└──────────────┘        └──────────────┘
```

## Multi-Tenancy Layers

```
┌───────────────────────────────────────────────────────────┐
│  Layer 1: Network Isolation                               │
│  • Each tenant gets X-Tenant-ID header                   │
│  • Frontend ensures headers are always present           │
└───────────────────────────────────────────────────────────┘
                            ▼
┌───────────────────────────────────────────────────────────┐
│  Layer 2: Authentication & Authorization                  │
│  • Go backend validates tenant exists                    │
│  • Checks user has access to tenant                      │
│  • Future: JWT with tenant claims                        │
└───────────────────────────────────────────────────────────┘
                            ▼
┌───────────────────────────────────────────────────────────┐
│  Layer 3: Query Rewrite (RLS)                             │
│  • Cube.js injects tenant_id filter into EVERY query     │
│  • Impossible to query cross-tenant data                 │
│  • Enforced at semantic layer (defense in depth)         │
└───────────────────────────────────────────────────────────┘
                            ▼
┌───────────────────────────────────────────────────────────┐
│  Layer 4: QoS Isolation                                   │
│  • context_to_app_id creates per-tenant queues           │
│  • Prevents noisy neighbor problems                      │
│  • Fair resource allocation                              │
└───────────────────────────────────────────────────────────┘
                            ▼
┌───────────────────────────────────────────────────────────┐
│  Layer 5: Database Resource Groups                        │
│  • StarRocks resource groups per tenant tier             │
│  • CPU/memory/concurrency limits                         │
│  • Prevents single tenant monopolizing resources         │
└───────────────────────────────────────────────────────────┘
                            ▼
┌───────────────────────────────────────────────────────────┐
│  Layer 6: Data Partitioning (Physical)                   │
│  • Iceberg tables partitioned by tenant_id               │
│  • Efficient pruning at storage layer                    │
│  • Reduced I/O for tenant queries                        │
└───────────────────────────────────────────────────────────┘
```

## Hot/Cold Tiering Strategy

```
                    Query arrives at Cube.js
                              │
                              ▼
                    ┌─────────────────┐
                    │ Check data_source│
                    │ property in cube │
                    └────────┬─────────┘
                            │
                ┌───────────┴───────────┐
                │                       │
                ▼                       ▼
        data_source:              data_source:
         starrocks                   trino
                │                       │
                ▼                       ▼
    ┌────────────────────┐   ┌────────────────────┐
    │   Hot Tier         │   │   Cold Tier        │
    │   (StarRocks)      │   │   (Trino)          │
    │                    │   │                    │
    │ Use cases:         │   │ Use cases:         │
    │ • Real-time        │   │ • Historical       │
    │   dashboards       │   │   analysis         │
    │ • Operational      │   │ • Year-over-year   │
    │   queries          │   │   comparisons      │
    │ • Last 90 days     │   │ • Long-term        │
    │                    │   │   trends           │
    │ Performance:       │   │                    │
    │ • <200ms p95       │   │ Performance:       │
    │ • Low latency      │   │ • 2-10s queries    │
    │                    │   │ • High throughput  │
    │ Refresh:           │   │                    │
    │ • Every 5-15 min   │   │ Refresh:           │
    │                    │   │ • Daily/weekly     │
    └────────────────────┘   └────────────────────┘
```

## Pre-Aggregation Flow

```
┌─────────────────────────────────────────────────────────────┐
│  Cube.js Refresh Worker (Background Process)                │
│                                                              │
│  For each tenant in scheduledRefreshContexts():             │
│                                                              │
│  1. Fetch schema for tenant                                 │
│  2. Identify pre_aggregations that need refresh            │
│  3. Check last_refresh time                                 │
│  4. If stale:                                               │
│     a) Execute complex query against source (Trino)        │
│     b) Write results to StarRocks (cube_preaggs DB)        │
│     c) Update metadata table                                │
│                                                              │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  StarRocks: cube_preaggs Database                            │
│                                                              │
│  Tables:                                                     │
│  • preagg_metadata (tracking)                               │
│  • trades_by_day_20241101_20241130 (actual rollup)         │
│  • portfolio_summary_20241101_20241130                      │
│  • ...                                                       │
│                                                              │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       │ When user query arrives
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  Cube.js Query Planner                                       │
│                                                              │
│  Query: "Show daily trade volume for last 30 days"         │
│                                                              │
│  Checks:                                                     │
│  1. Is there a pre-aggregation matching this query?        │
│     → trades_by_day matches!                                │
│  2. Is it fresh enough?                                     │
│     → last_refresh: 10 minutes ago ✓                        │
│  3. Route to pre-aggregation table in StarRocks            │
│                                                              │
│  Result: <200ms response (vs 5-10s from raw Trino)         │
└─────────────────────────────────────────────────────────────┘
```

### Tenant Metadata Sync

`scripts/sync_cube_tenants.go` snapshots the authoritative tenant hierarchy (`tenant_product_datasource`, `alpha_datasource`, `tenant_instance`) into `cube/generated/tenant-scopes.json` and writes any `schema_overrides` JSON to `cube/schema/tenants/<tenant>/<datasource>/auto/*.yml`. At runtime `cube.js` uses that snapshot to:

- Build `scheduledRefreshContexts` without hitting Postgres.
- Stamp `contextToAppId`/`contextToOrchestratorId` with the right resource group so StarRocks workload management stays tenant-aware.
- Overlay tenant/datasource-specific cube files on top of the shared schema (via `repositoryFactory`).

Regenerate the snapshot with `make sync-cube-tenants` whenever tenants or datasource configs change.

## Governance: Rollup-Only Mode

```
┌─────────────────────────────────────────────────────────────┐
│  User submits query                                          │
│  "Show me detailed trades for last 2 years"                 │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  Cube.js Query Planner (CUBEJS_ROLLUP_ONLY=true)            │
│                                                              │
│  1. Check if pre-aggregation exists for this query          │
│  2. No matching pre-aggregation found                       │
│  3. Check if ROLLUP_ONLY is enabled                         │
│  4. ROLLUP_ONLY=true → REJECT QUERY                         │
│                                                              │
└──────────────────────┬──────────────────────────────────────┘
                       │
                       ▼
┌─────────────────────────────────────────────────────────────┐
│  Error Response                                              │
│                                                              │
│  {                                                           │
│    "error": "Pre-aggregation required",                     │
│    "message": "Query cannot be satisfied by existing        │
│                rollups. Contact admin to create one."       │
│  }                                                           │
│                                                              │
│  ✅ BENEFIT: Trino/Parquet lake protected from expensive   │
│              ad-hoc queries                                  │
└─────────────────────────────────────────────────────────────┘
```

## Comparison: Before vs After Cube.js

### Before (cube-gonja only)

```
Frontend → Go Backend → cube-gonja → StarRocks/Trino
                                      (manual SQL generation)
                                      (no pre-aggregations)
                                      (limited BI tool support)
```

### After (with Cube.js)

```
┌─────────────────────────────────────────────────────────┐
│  Frontend → Go Backend → Cube.js → StarRocks/Trino     │
│                                     (intelligent routing)│
│                                     (automatic rollups)  │
│                                                          │
│  BI Tools ────────────→ Cube.js → StarRocks/Trino      │
│  (via SQL API)                     (SQL generation)     │
│                                                          │
│  AI Agents ────────────→ Cube.js → StarRocks/Trino     │
│  (via GraphQL)                     (consistent metrics) │
└─────────────────────────────────────────────────────────┘
```

### Benefits

| Feature | Before | After | Improvement |
|---------|--------|-------|-------------|
| Query latency | 2-10s | <500ms | 4-20x faster |
| BI tool support | Limited | Universal | Any tool |
| Multi-tenancy | Manual | Automatic | Safer |
| Pre-aggregations | None | Automatic | Much faster |
| Governance | None | Rollup-only | Protected lake |

## Summary

This architecture provides:

✅ **Multi-tenant isolation** at 6 layers (network → physical storage)  
✅ **Hot/cold tiering** for optimal performance and cost  
✅ **Automatic pre-aggregations** for sub-second queries  
✅ **Universal API** (REST, GraphQL, SQL) for any consumer  
✅ **Governance controls** to protect data lake  
✅ **HA storage** via StarRocks (no Cube Store dependency)  

The system is production-ready and follows industry best practices.
